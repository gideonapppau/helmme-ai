// Stored edges (§32, §72). The related endpoint computes links live; this
// file writes the deterministic kinds down with kind, confidence, evidence,
// and stamps, so every connection is a row you can point at, not a guess.
// Kinds: same-site (undirected), references (directed, quoter to quoted),
// same-topic (undirected), saved-together (undirected).
// GET /v1/items/{id}/edges reads them back.
package main

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// urlFinder spots addresses quoted inside saved text.
var urlFinder = regexp.MustCompile(`https?://[^\s"'<>\]\)]+`)

// extractURLs lists the addresses inside text, trimmed and capped.
// Plain words: find every link pasted in the words, at most twenty.
func extractURLs(text string) []string {
	seen := map[string]bool{}
	var out []string
	for _, raw := range urlFinder.FindAllString(text, -1) {
		u := strings.TrimRight(raw, ".,;:!?")
		if u == "" || seen[u] {
			continue
		}
		seen[u] = true
		out = append(out, u)
		if len(out) >= 20 {
			break
		}
	}
	return out
}

// orderPair lines two ids up the same way every time, so an undirected
// link files once no matter which end found it first.
func orderPair(a, b string) (string, string) {
	if a < b {
		return a, b
	}
	return b, a
}

// storeEdge files one link, or refreshes its stamp when already known.
// Evidence names the item that proved it.
func storeEdge(ctx context.Context, pool *pgxpool.Pool, source, target, kind string, confidence float64, evidence []string) {
	if source == "" || target == "" || source == target {
		return
	}
	ev, _ := json.Marshal(evidence)
	_, _ = pool.Exec(ctx, `
	  INSERT INTO relationships (source_item_id, target_item_id, type, confidence, evidence)
	  VALUES ($1,$2,$3,$4,$5)
	  ON CONFLICT (source_item_id, target_item_id, type)
	  DO UPDATE SET verified_at=now(), confidence=GREATEST(relationships.confidence,$4)`,
		source, target, kind, confidence, ev)
}

// storeEdges files what this item proves about its neighbors. Runs at the
// end of enrichment, so it never fires when the user switched guesses off.
func storeEdges(ctx context.Context, pool *pgxpool.Pool, user, itemID, domain, content string, topicIDs []string) {
	// Same place: other live saves from this address.
	if domain != "" {
		rows, err := pool.Query(ctx, `
		  SELECT id FROM items
		  WHERE user_id=$1 AND status='active' AND sync_state<>'deleted'
		    AND domain=$2 AND id<>$3 LIMIT 20`, user, domain, itemID)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var peer string
				if err := rows.Scan(&peer); err == nil {
					a, b := orderPair(itemID, peer)
					storeEdge(ctx, pool, a, b, "same-site", 0.5, []string{itemID})
				}
			}
		}
	}
	// Quoted addresses: this item points at that one.
	for _, raw := range extractURLs(content) {
		canon, _ := canonicalFields("url", raw)
		cs, ok := canon.(string)
		if !ok || cs == "" {
			continue
		}
		var target string
		if err := pool.QueryRow(ctx, `
		  SELECT id FROM items
		  WHERE user_id=$1 AND status='active' AND sync_state<>'deleted'
		    AND canonical_url=$2 AND id<>$3 LIMIT 1`, user, cs, itemID).Scan(&target); err != nil {
			continue
		}
		storeEdge(ctx, pool, itemID, target, "references", 0.8, []string{itemID})
	}
	// Same idea: other saves carrying these topics.
	for _, tid := range topicIDs {
		rows, err := pool.Query(ctx, `
		  SELECT item_id FROM item_topics
		  WHERE topic_id=$1 AND item_id<>$2 LIMIT 20`, tid, itemID)
		if err != nil {
			continue
		}
		func() {
			defer rows.Close()
			for rows.Next() {
				var peer string
				if err := rows.Scan(&peer); err == nil {
					a, b := orderPair(itemID, peer)
					storeEdge(ctx, pool, a, b, "same-topic", 0.6, []string{itemID})
				}
			}
		}()
	}
}

func registerEdgeRoutes(mux *http.ServeMux, pool *pgxpool.Pool) {
	// GET /v1/items/{id}/edges — every stored link touching this item,
	// each with kind, confidence, evidence, and stamps.
	mux.HandleFunc("GET /v1/items/{id}/edges", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user := tenantID(r)
		id := r.PathValue("id")
		var owner string
		if err := pool.QueryRow(ctx, `SELECT user_id FROM items WHERE id=$1`, id).Scan(&owner); err != nil || owner != user {
			http.Error(w, "not found", 404)
			return
		}
		rows, err := pool.Query(ctx, `
		  SELECT source_item_id, target_item_id, type, confidence,
		         COALESCE(evidence::text,'[]'), model_version,
		         created_at::text, verified_at::text
		  FROM relationships
		  WHERE source_item_id=$1 OR target_item_id=$1
		  ORDER BY confidence DESC LIMIT 50`, id)
		if err != nil {
			http.Error(w, "internal error", 500)
			return
		}
		defer rows.Close()
		out := []map[string]any{}
		for rows.Next() {
			var src, tgt, typ string
			var conf float64
			var evRaw, model, created, verified string
			if err := rows.Scan(&src, &tgt, &typ, &conf, &evRaw, &model, &created, &verified); err != nil {
				continue
			}
			var ev any
			_ = json.Unmarshal([]byte(evRaw), &ev)
			role := "outgoing"
			other := tgt
			if src != id {
				role = "incoming"
				other = src
			}
			out = append(out, map[string]any{
				"other_id": other, "role": role, "type": typ, "confidence": conf,
				"evidence": ev, "model": model, "created_at": created, "verified_at": verified,
			})
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"edges": out})
	})
}
