// Rediscovery (§38-39): old saves, relevant again, with reasons.
// GET /v1/resurface -> up to 3 items older than 90 days that share
// topics or entities with the last 30 days of captures. No reason,
// no surface. POST /v1/events logs opens for the future filter.
package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type resurfaced struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Source string `json:"source_type"`
	Domain string `json:"domain"`
	Reason string `json:"reason"`
	Seen   bool   `json:"seen"`
}

func registerResurfaceRoutes(mux *http.ServeMux, pool *pgxpool.Pool) {
	mux.HandleFunc("POST /v1/events", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		var in struct {
			ItemID string `json:"item_id"`
			Kind   string `json:"kind"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, "bad json", 400)
			return
		}
		if in.Kind != "opened" {
			http.Error(w, "unknown kind", 400)
			return
		}
		user := tenantID(r)
		var owner string
		if err := pool.QueryRow(ctx, `SELECT user_id FROM items WHERE id=$1`, in.ItemID).Scan(&owner); err != nil || owner != user {
			http.Error(w, "not found", 404)
			return
		}
		_, _ = pool.Exec(ctx, `INSERT INTO item_events (user_id, item_id, kind) VALUES ($1,$2,'opened')`,
			user, in.ItemID)
		w.WriteHeader(204)
	})

	mux.HandleFunc("GET /v1/resurface", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user := tenantID(r)
		// Seen things stay buried by default; ?show_seen=1 digs them up.
		hideSeen := r.URL.Query().Get("show_seen") != "1"
		out, err := fetchResurfaced(ctx, pool, user, hideSeen)
		if err != nil {
			log.Printf("resurface failed err=%T", err)
			http.Error(w, "internal error", 500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"items": out})
	})
}

// fetchResurfaced is shared with the weekly review: same candidates,
// same reasons, same honesty. The user's mood (Quiet/Balanced/Proactive)
// decides how many come back: servers enforce it, clients only display it.
func fetchResurfaced(ctx context.Context, pool *pgxpool.Pool, user string, hideSeen bool) ([]resurfaced, error) {
	seenFilter := ""
	if hideSeen {
		seenFilter = `
	  AND NOT EXISTS (SELECT 1 FROM item_events e
	    WHERE e.item_id = items.id AND e.kind = 'opened')`
	}
	// Recent context: topics and entities from the last 30 days.
	rows, err := pool.Query(ctx, `
	  WITH recent AS (
	    SELECT id FROM items
	    WHERE user_id=$1 AND status='active' AND captured_at > now() - interval '30 days'
	  ), old AS (
	    SELECT id FROM items
	    WHERE user_id=$1 AND status='active' AND captured_at < now() - interval '90 days'`+seenFilter+`
	  )
	  SELECT o.id, COALESCE(i.title,''), i.source_type, COALESCE(i.domain,''),
	         COALESCE((
	           SELECT t.name FROM item_topics it
	           JOIN topics t ON t.id = it.topic_id
	           WHERE it.item_id = o.id AND NOT t.hidden AND t.id IN (
	             SELECT topic_id FROM item_topics WHERE item_id IN (SELECT id FROM recent))
	           ORDER BY it.confidence DESC LIMIT 1), ''),
	         COALESCE((
	           SELECT e.canonical_name FROM item_entities ie
	           JOIN entities e ON e.id = ie.entity_id
	           WHERE ie.item_id = o.id AND e.id IN (
	             SELECT entity_id FROM item_entities WHERE item_id IN (SELECT id FROM recent))
	           ORDER BY ie.confidence DESC LIMIT 1), ''),
	         (SELECT COUNT(*) FROM item_topics it
	          WHERE it.item_id = o.id AND it.topic_id IN (
	            SELECT topic_id FROM item_topics WHERE item_id IN (SELECT id FROM recent)))
	         + (SELECT COUNT(*) FROM item_entities ie
	          WHERE ie.item_id = o.id AND ie.entity_id IN (
	            SELECT entity_id FROM item_entities WHERE item_id IN (SELECT id FROM recent))),
	         EXISTS (SELECT 1 FROM item_events e
	           WHERE e.item_id = o.id AND e.kind = 'opened'
	             AND e.created_at > now() - interval '30 days')
	  FROM old o JOIN items i ON i.id = o.id
	  ORDER BY 8 ASC, 7 DESC, i.captured_at DESC LIMIT 5`, user)
	if err != nil {
		log.Printf("resurface failed err=%T", err)
		return nil, err
	}
	defer rows.Close()
	out := []resurfaced{}
	mode := "quiet"
	_ = pool.QueryRow(ctx,
		`SELECT resurface_mode FROM user_settings WHERE user_id=$1`, user).Scan(&mode)
	limit := resurfaceLimit(mode)
	for rows.Next() {
		var s resurfaced
		var topic, entity string
		var score int
		if err := rows.Scan(&s.ID, &s.Title, &s.Source, &s.Domain, &topic, &entity, &score, &s.Seen); err != nil {
			continue
		}
		if score == 0 {
			continue
		}
		name := topic
		if name == "" {
			name = entity
		}
		if strings.Contains(name, ".") {
			s.Reason = "More from " + name + ", also in recent saves."
		} else {
			s.Reason = "Connected to recent saves about " + name + "."
		}
		out = append(out, s)
		if len(out) >= limit {
			break
		}
	}
	return out, rows.Err()
}
