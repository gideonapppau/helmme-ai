// Merge (§35): duplicates fold into the oldest save without losing rows.
// POST /v1/maintenance/merge {ids} -> {kept, merged[]}
// POST /v1/maintenance/unmerge {ids} -> restored count.
// Merged items hide from search, related, and duplicate-finding, and come
// back exactly as they were. Deleting a survivor restores its children.
package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

func registerMergeRoutes(mux *http.ServeMux, pool *pgxpool.Pool) {
	mux.HandleFunc("POST /v1/maintenance/merge", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		var in struct {
			IDs []string `json:"ids"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil || len(in.IDs) < 2 {
			http.Error(w, "need at least two items", 400)
			return
		}
		if len(in.IDs) > 50 {
			http.Error(w, "too many at once", 400)
			return
		}
		user := tenantID(r)
		type row struct {
			id       string
			captured string
		}
		var rows []row
		for _, id := range in.IDs {
			var c string
			var status string
			err := pool.QueryRow(ctx, `
			  SELECT captured_at::text, status FROM items WHERE id=$1 AND user_id=$2`,
				id, user).Scan(&c, &status)
			if err != nil || status != "active" {
				http.Error(w, "not found", 404)
				return
			}
			rows = append(rows, row{id, c})
		}
		// Oldest save survives. Ties keep first listed.
		kept := rows[0].id
		oldest := rows[0].captured
		for _, rw := range rows[1:] {
			if rw.captured < oldest {
				kept, oldest = rw.id, rw.captured
			}
		}
		var merged []string
		for _, rw := range rows {
			if rw.id == kept {
				continue
			}
			if _, err := pool.Exec(ctx, `
			  UPDATE items SET status='merged', merged_into=$1 WHERE id=$2`,
				kept, rw.id); err != nil {
				log.Printf("merge failed err=%T", err)
				http.Error(w, "internal error", 500)
				return
			}
			merged = append(merged, rw.id)
		}
		if merged == nil {
			merged = []string{}
		}
		payload, _ := json.Marshal(map[string]any{"merged": merged})
		_, _ = pool.Exec(ctx, `INSERT INTO provenance_events (item_id, kind, payload) VALUES ($1,'merged',$2)`,
			kept, payload)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"kept": kept, "merged": merged})
	})

	mux.HandleFunc("POST /v1/maintenance/unmerge", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		var in struct {
			IDs []string `json:"ids"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil || len(in.IDs) == 0 {
			http.Error(w, "bad request", 400)
			return
		}
		if len(in.IDs) > 50 {
			http.Error(w, "too many at once", 400)
			return
		}
		user := tenantID(r)
		restored := 0
		for _, id := range in.IDs {
			res, err := pool.Exec(ctx, `
			  UPDATE items SET status='active', merged_into=NULL
			  WHERE id=$1 AND user_id=$2 AND status='merged'`, id, user)
			if err != nil {
				log.Printf("unmerge failed err=%T", err)
				http.Error(w, "internal error", 500)
				return
			}
			restored += int(res.RowsAffected())
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"restored": restored})
	})
}
