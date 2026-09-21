// Single item view (§82): original content plus everything derived,
// with derived material flagged as such by the UI. Ownership enforced.
package main

import (
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

type assetInfo struct {
	Kind      string `json:"kind"`
	Mime      string `json:"mime"`
	ByteSize  int64  `json:"byte_size"`
	Width     *int   `json:"width"`
	Height    *int   `json:"height"`
	PageCount *int   `json:"page_count"`
}

func registerItemRoutes(mux *http.ServeMux, pool *pgxpool.Pool) {
	mux.HandleFunc("GET /v1/items/{id}", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user := tenantID(r)
		id := r.PathValue("id")
		if id == "" {
			http.Error(w, "not found", 404)
			return
		}

		var (
			sourceType, rawRef, title, domain, capturedAt, origRaw, status, mergedInto, content string
			truncated                                                                           bool
		)
		err := pool.QueryRow(ctx, `
		  SELECT source_type, raw_ref, COALESCE(title,''), COALESCE(domain,''),
		         captured_at::text, COALESCE(original::text,'{}'),
		         status, COALESCE(merged_into::text,''),
		         COALESCE(LEFT(c.extracted_text, 100000),''),
		         COALESCE(LENGTH(c.extracted_text),0) > 100000
		  FROM items i LEFT JOIN item_contents c ON c.item_id = i.id
		  WHERE i.id=$1 AND i.user_id=$2`,
			id, user).Scan(&sourceType, &rawRef, &title, &domain, &capturedAt, &origRaw, &status, &mergedInto, &content, &truncated)
		if err != nil {
			http.Error(w, "not found", 404)
			return
		}
		var original any
		_ = json.Unmarshal([]byte(origRaw), &original)

		type named struct {
			ID         string  `json:"id"`
			Name       string  `json:"name"`
			Confidence float64 `json:"confidence"`
		}
		var topics []named
		trows, err := pool.Query(ctx, `
		  SELECT t.id, t.name, it.confidence FROM item_topics it
		  JOIN topics t ON t.id = it.topic_id
		  WHERE it.item_id=$1 ORDER BY it.confidence DESC`, id)
		if err == nil {
			defer trows.Close()
			for trows.Next() {
				var n named
				if err := trows.Scan(&n.ID, &n.Name, &n.Confidence); err == nil {
					topics = append(topics, n)
				}
			}
		}
		type ent struct {
			ID         string  `json:"id"`
			Type       string  `json:"type"`
			Name       string  `json:"name"`
			Confidence float64 `json:"confidence"`
		}
		var entities []ent
		erows, err := pool.Query(ctx, `
		  SELECT e.id, e.type, e.canonical_name, ie.confidence FROM item_entities ie
		  JOIN entities e ON e.id = ie.entity_id
		  WHERE ie.item_id=$1 ORDER BY ie.confidence DESC`, id)
		if err == nil {
			defer erows.Close()
			for erows.Next() {
				var e ent
				if err := erows.Scan(&e.ID, &e.Type, &e.Name, &e.Confidence); err == nil {
					entities = append(entities, e)
				}
			}
		}
		var asset *assetInfo
		var kind, mime string
		var size int64
		var width, height, pages *int
		if err := pool.QueryRow(ctx, `
		  SELECT kind, mime, byte_size, width, height, page_count
		  FROM item_assets WHERE item_id=$1`, id).Scan(
			&kind, &mime, &size, &width, &height, &pages); err == nil {
			asset = &assetInfo{kind, mime, size, width, height, pages}
		}
		// No asset row is normal (notes, links) — not worth a log line.

		if topics == nil {
			topics = []named{}
		}
		if entities == nil {
			entities = []ent{}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id": id, "source_type": sourceType, "raw_ref": rawRef,
			"title": title, "domain": domain, "captured_at": capturedAt,
			"original": original, "topics": topics, "entities": entities,
			"asset": asset, "status": status, "merged_into": mergedInto,
			"content": content, "truncated": truncated,
		})
	})
}
