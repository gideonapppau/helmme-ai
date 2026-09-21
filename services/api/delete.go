// Delete one item and everything derived from it (§79, §164.16).
// DELETE /v1/items/:id -> 204. Unknown ids and other tenants' items
// both answer 404 with no reason why.
// File bytes are shared by content hash: the stored copy is removed only
// when no remaining item points at it.
package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/jackc/pgx/v5/pgxpool"
)

func registerDeleteRoutes(mux *http.ServeMux, pool *pgxpool.Pool, uploadDir string) {
	mux.HandleFunc("DELETE /v1/items/{id}", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user := tenantID(r)
		id := r.PathValue("id")
		if id == "" {
			http.Error(w, "not found", 404)
			return
		}

		var hash, storageRef string
		err := pool.QueryRow(ctx, `
		  SELECT i.content_hash, COALESCE(a.storage_ref,'')
		  FROM items i LEFT JOIN item_assets a ON a.item_id = i.id
		  WHERE i.id=$1 AND i.user_id=$2`, id, user).Scan(&hash, &storageRef)
		if err != nil {
			http.Error(w, "not found", 404)
			return
		}

		// Deleting a survivor restores its children first: merge hides,
		// it never destroys. Nothing is lost through merge, ever.
		_, _ = pool.Exec(ctx, `
		  UPDATE items SET status='active', merged_into=NULL
		  WHERE merged_into=$1 AND user_id=$2`, id, user)

		if _, err := pool.Exec(ctx, `DELETE FROM items WHERE id=$1`, id); err != nil {
			log.Printf("delete failed err=%T", err)
			http.Error(w, "internal error", 500)
			return
		}

		// Last copy wins the file: keep bytes while any item shares the hash.
		if storageRef != "" {
			var remaining int
			_ = pool.QueryRow(ctx, `SELECT COUNT(*) FROM items WHERE content_hash=$1`, hash).Scan(&remaining)
			if remaining == 0 {
				_ = os.Remove(filepath.Join(uploadDir, storageRef))
			}
		}
		w.WriteHeader(204)
	})
}
