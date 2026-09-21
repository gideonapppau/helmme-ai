// Related items (§83): Similar, Same site, Earlier, Later, Saved together.
// Phase-1 version uses existing data only, no entities graph yet. Every
// connection carries its reason, so the UI can answer "why this?" (§156).
package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Reasons are stable codes; the UI renders them in plain words.
const (
	reasonReferenced    = "referenced"
	reasonSameSite      = "same-site"
	reasonSimilarWords  = "similar-words"
	reasonSavedTogether = "saved-together"
	reasonSameTopic     = "same-topic"
	reasonEarlier       = "earlier"
	reasonLater         = "later"
)

type relatedHit struct {
	ID         string    `json:"id"`
	Title      string    `json:"title"`
	Source     string    `json:"source_type"`
	Domain     string    `json:"domain"`
	CapturedAt time.Time `json:"captured_at"`
	Reasons    []string  `json:"reasons"`
}

// mergeRelated dedupes by id, combines reasons, keeps signal priority
// order, and caps the list. Pure, unit tested.
func mergeRelated(groups [][]relatedHit, cap int) []relatedHit {
	seen := map[string]int{}
	var out []relatedHit
	for _, g := range groups {
		for _, h := range g {
			if i, ok := seen[h.ID]; ok {
				out[i].Reasons = append(out[i].Reasons, h.Reasons...)
				continue
			}
			if len(out) >= cap {
				continue
			}
			seen[h.ID] = len(out)
			out = append(out, h)
		}
	}
	return out
}

func registerRelatedRoutes(mux *http.ServeMux, pool *pgxpool.Pool) {
	mux.HandleFunc("GET /v1/items/{id}/related", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user := tenantID(r)
		id := r.PathValue("id")
		if id == "" {
			http.Error(w, "missing id", 400)
			return
		}

		var owner string
		if err := pool.QueryRow(ctx, `SELECT user_id FROM items WHERE id=$1`, id).Scan(&owner); err != nil || owner != user {
			http.Error(w, "not found", 404)
			return
		}

		var base struct {
			sourceType string
			rawRef     string
			domain     string
			text       string
			capturedAt time.Time
		}
		_ = pool.QueryRow(ctx, `
		  SELECT i.source_type, i.raw_ref, COALESCE(i.domain,''),
		         COALESCE(c.extracted_text,''), i.captured_at
		  FROM items i LEFT JOIN item_contents c ON c.item_id = i.id
		  WHERE i.id = $1`,
			id).Scan(&base.sourceType, &base.rawRef, &base.domain, &base.text, &base.capturedAt)

		scan := func(rows interface {
			Next() bool
			Scan(...interface{}) error
			Close()
		}, reason string) []relatedHit {
			var out []relatedHit
			defer rows.Close()
			for rows.Next() {
				var h relatedHit
				var title, domain *string
				if err := rows.Scan(&h.ID, &title, &h.Source, &domain, &h.CapturedAt); err != nil {
					continue
				}
				if h.ID == id {
					continue
				}
				if title != nil {
					h.Title = *title
				}
				if domain != nil {
					h.Domain = *domain
				}
				h.Reasons = []string{reason}
				out = append(out, h)
			}
			return out
		}

		// Referenced: saves that mention this item's address (§48).
		// Plain words: if your note pastes a link to an article you saved,
		// the two meet here. Only addressable items (links, posts) qualify.
		var referenced []relatedHit
		addrAny, _ := canonicalFields(base.sourceType, base.rawRef)
		if addr, ok := addrAny.(string); ok && addr != "" {
			refRows, err := pool.Query(ctx, `
			  SELECT i.id, i.title, i.source_type, i.domain, i.captured_at
			  FROM items i JOIN item_contents c ON c.item_id=i.id
			  WHERE i.user_id=$1 AND i.id<>$2 AND i.status='active' AND i.sync_state<>'deleted'
			    AND POSITION($3 IN COALESCE(c.extracted_text,'')) > 0 LIMIT 5`, user, id, addr)
			if err == nil {
				referenced = scan(refRows, reasonReferenced)
			}
		}

		// Same import run = saved together.
		togetherRows, err := pool.Query(ctx, `
		  SELECT i.id, i.title, i.source_type, i.domain, i.captured_at
		  FROM import_items m JOIN items i ON i.id = m.item_id
		  WHERE m.import_id IN (SELECT import_id FROM import_items WHERE item_id=$1)
		    AND i.id <> $1 AND i.user_id=$2 LIMIT 5`, id, user)
		var together []relatedHit
		if err == nil {
			together = scan(togetherRows, reasonSavedTogether)
		}

		// Same topic (visible ones only, hidden guesses stay quiet).
		var sameTopic []relatedHit
		topicRows, err := pool.Query(ctx, `
		  SELECT DISTINCT i.id, i.title, i.source_type, i.domain, i.captured_at
		  FROM item_topics mine
		  JOIN item_topics theirs ON theirs.topic_id = mine.topic_id
		  JOIN topics t ON t.id = theirs.topic_id AND NOT t.hidden
		  JOIN items i ON i.id = theirs.item_id
		  WHERE mine.item_id=$1 AND i.id<>$1 AND i.user_id=$2 AND i.status='active' LIMIT 5`, id, user)
		if err == nil {
			sameTopic = scan(topicRows, reasonSameTopic)
		}

		// Same site.
		var sameSite []relatedHit
		if base.domain != "" {
			siteRows, err := pool.Query(ctx, `
			  SELECT id, title, source_type, domain, captured_at FROM items
			  WHERE user_id=$1 AND status='active' AND domain=$2 AND id<>$3
			  ORDER BY captured_at DESC LIMIT 5`, user, base.domain, id)
			if err == nil {
				sameSite = scan(siteRows, reasonSameSite)
			}
		}

		// Similar words (trigram on extracted text).
		var similar []relatedHit
		if len(base.text) > 20 {
			simRows, err := pool.Query(ctx, `
			  SELECT i.id, i.title, i.source_type, i.domain, i.captured_at
			  FROM items i JOIN item_contents c ON c.item_id=i.id
			  WHERE i.user_id=$1 AND i.id<>$2 AND i.status='active'
			    AND similarity(c.extracted_text, $3) > 0.08
			  ORDER BY similarity(c.extracted_text, $3) DESC LIMIT 5`, user, id, base.text)
			if err == nil {
				similar = scan(simRows, reasonSimilarWords)
			}
		}

		// Time neighbors.
		var neighbors []relatedHit
		earlierRows, err := pool.Query(ctx, `
		  SELECT id, title, source_type, domain, captured_at FROM items
		  WHERE user_id=$1 AND status='active' AND captured_at < $2
		  ORDER BY captured_at DESC LIMIT 1`, user, base.capturedAt)
		if err == nil {
			neighbors = append(neighbors, scan(earlierRows, reasonEarlier)...)
		}
		laterRows, err := pool.Query(ctx, `
		  SELECT id, title, source_type, domain, captured_at FROM items
		  WHERE user_id=$1 AND status='active' AND captured_at > $2
		  ORDER BY captured_at ASC LIMIT 1`, user, base.capturedAt)
		if err == nil {
			neighbors = append(neighbors, scan(laterRows, reasonLater)...)
		}

		related := mergeRelated([][]relatedHit{referenced, together, sameTopic, sameSite, similar, neighbors}, 6)
		if related == nil {
			related = []relatedHit{}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"related": related})
	})
}
