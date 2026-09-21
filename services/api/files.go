// File serving: open what you saved (§60 raw storage, §164.11).
// GET /v1/items/:id/file -> bytes with the stored content type.
// Ownership first; paths never come from the URL; sandbox headers so a
// saved file can never run code in the browser.
package main

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// safeJoin keeps resolved paths inside dir. Rejects escapes, absolute
// paths, and separators, storage refs are content hashes, never names.
func safeJoin(dir, ref string) (string, bool) {
	if ref == "" || ref == "." || ref != filepath.Base(ref) {
		return "", false
	}
	if strings.Contains(ref, "\\") {
		return "", false
	}
	joined := filepath.Join(dir, ref)
	absDir, err1 := filepath.Abs(dir)
	absJoined, err2 := filepath.Abs(joined)
	if err1 != nil || err2 != nil {
		return "", false
	}
	if absJoined != absDir && !strings.HasPrefix(absJoined, absDir+string(os.PathSeparator)) {
		return "", false
	}
	return joined, true
}

func registerFileRoutes(mux *http.ServeMux, pool *pgxpool.Pool, uploadDir string) {
	mux.HandleFunc("GET /v1/items/{id}/file", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user := tenantID(r)
		id := r.PathValue("id")
		if id == "" {
			http.Error(w, "not found", 404)
			return
		}
		var storageRef, mime string
		err := pool.QueryRow(ctx, `
		  SELECT a.storage_ref, a.mime FROM item_assets a
		  JOIN items i ON i.id = a.item_id
		  WHERE a.item_id=$1 AND i.user_id=$2`, id, user).Scan(&storageRef, &mime)
		if err != nil {
			// Text and generic files carry no asset row (their words are the asset).
			// Their bytes still live under raw_ref ("upload/<hash>.<ext>").
			// Plain words: same locked door, second key.
			var sourceType, rawRef, origMime string
			if err2 := pool.QueryRow(ctx, `
			  SELECT source_type, raw_ref, COALESCE(original->>'mime','')
			  FROM items WHERE id=$1 AND user_id=$2`,
				id, user).Scan(&sourceType, &rawRef, &origMime); err2 != nil {
				http.Error(w, "not found", 404)
				return
			}
			if (sourceType != "text" && sourceType != "file") || !strings.HasPrefix(rawRef, "upload/") {
				http.Error(w, "not found", 404)
				return
			}
			storageRef = strings.TrimPrefix(rawRef, "upload/")
			mime = origMime
			if mime == "" {
				mime = "text/plain; charset=utf-8"
			}
		}
		path, ok := safeJoin(uploadDir, storageRef)
		if !ok {
			http.Error(w, "not found", 404)
			return
		}
		data, err := os.ReadFile(path)
		if err != nil {
			http.Error(w, "not found", 404)
			return
		}
		w.Header().Set("Content-Type", mime)
		w.Header().Set("Content-Security-Policy", "sandbox")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Write(data)
	})
}
