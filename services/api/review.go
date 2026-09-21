// Weekly review (§37): one honest page about the week.
// Saved count, top topics, resurfaced items, cleanup needs, unopened
// bookmarks. Every number is live; empty states say "nothing yet".
package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

func registerReviewRoutes(mux *http.ServeMux, pool *pgxpool.Pool) {
	mux.HandleFunc("GET /v1/review", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user := tenantID(r)

		var saved int
		_ = pool.QueryRow(ctx, `
		  SELECT COUNT(*) FROM items
		  WHERE user_id=$1 AND status='active' AND captured_at > now() - interval '7 days'`,
			user).Scan(&saved)

		type topicCount struct {
			Name  string `json:"name"`
			Count int    `json:"count"`
		}
		var topics []topicCount
		if rows, err := pool.Query(ctx, `
		  SELECT t.name, COUNT(*) FROM item_topics it
		  JOIN topics t ON t.id = it.topic_id
		  JOIN items i ON i.id = it.item_id
		  WHERE i.user_id=$1 AND i.status='active' AND i.captured_at > now() - interval '7 days'
		  GROUP BY t.name ORDER BY COUNT(*) DESC LIMIT 5`, user); err == nil {
			defer rows.Close()
			for rows.Next() {
				var t topicCount
				if err := rows.Scan(&t.Name, &t.Count); err == nil {
					topics = append(topics, t)
				}
			}
		} else {
			log.Printf("review topics failed err=%T", err)
		}
		if topics == nil {
			topics = []topicCount{}
		}
		var entities []topicCount
		if rows, err := pool.Query(ctx, `
		  SELECT e.canonical_name, COUNT(*) FROM item_entities ie
		  JOIN entities e ON e.id = ie.entity_id
		  JOIN items i ON i.id = ie.item_id
		  WHERE i.user_id=$1 AND i.status='active' AND i.captured_at > now() - interval '7 days'
		  GROUP BY e.canonical_name ORDER BY COUNT(*) DESC LIMIT 8`, user); err == nil {
			defer rows.Close()
			for rows.Next() {
				var t topicCount
				if err := rows.Scan(&t.Name, &t.Count); err == nil {
					entities = append(entities, t)
				}
			}
		} else {
			log.Printf("review entities failed err=%T", err)
		}
		if entities == nil {
			entities = []topicCount{}
		}

		surf, err := fetchResurfaced(ctx, pool, user, false)
		if err != nil {
			log.Printf("review resurface failed err=%T", err)
			surf = []resurfaced{}
		}

		var dupeGroups int
		_ = pool.QueryRow(ctx, `
		  SELECT COUNT(*) FROM (
		    SELECT content_hash FROM items WHERE user_id=$1 AND status='active'
		    GROUP BY content_hash HAVING COUNT(*) > 1) s`, user).Scan(&dupeGroups)

		var unopened int
		_ = pool.QueryRow(ctx, `
		  SELECT COUNT(*) FROM items i
		  WHERE i.user_id=$1 AND i.status='active' AND i.source_type='bookmark'
		    AND NOT EXISTS (SELECT 1 FROM item_events e
		      WHERE e.item_id=i.id AND e.kind='opened')`, user).Scan(&unopened)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"saved_this_week":    saved,
			"top_topics":         topics,
			"top_entities":       entities,
			"resurfaced":         surf,
			"duplicate_groups":   dupeGroups,
			"unopened_bookmarks": unopened,
		})
	})
}
