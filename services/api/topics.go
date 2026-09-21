// Topic controls (§34). Topics are guesses, so the user overrules them:
// rename a bad name, hide a noisy one, merge twins. Hiding never deletes;
// merged-away names are gone but every item keeps its links.
// GET /v1/topics (?show_hidden=1) · PATCH /v1/topics/{id} {name} ·
// POST /v1/topics/{id}/hide · POST /v1/topics/{id}/unhide ·
// POST /v1/topics/merge {keep_id, drop_id}
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// cleanTopicName trims a topic name to something displayable.
// Plain words: names are short, never blank.
func cleanTopicName(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 120 {
		s = s[:120]
	}
	return s
}

func registerTopicRoutes(mux *http.ServeMux, pool *pgxpool.Pool) {
	// GET /v1/topics — visible topics with live counts. Hidden ones stay
	// out unless asked for, so quiet means quiet.
	mux.HandleFunc("GET /v1/topics", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user := tenantID(r)
		showHidden := r.URL.Query().Get("show_hidden") == "1"
		filter := "AND NOT t.hidden"
		if showHidden {
			filter = ""
		}
		rows, err := pool.Query(ctx, `
		  SELECT t.id, t.name, t.confidence, t.hidden, COUNT(it.item_id)
		  FROM topics t LEFT JOIN item_topics it ON it.topic_id = t.id
		  LEFT JOIN items i ON i.id = it.item_id AND i.status='active' AND i.sync_state<>'deleted'
		  WHERE t.user_id=$1 `+filter+`
		  GROUP BY t.id ORDER BY COUNT(it.item_id) DESC, t.name`, user)
		if err != nil {
			http.Error(w, "internal error", 500)
			return
		}
		defer rows.Close()
		out := []map[string]any{}
		for rows.Next() {
			var id, name string
			var conf float64
			var hidden bool
			var n int
			if err := rows.Scan(&id, &name, &conf, &hidden, &n); err == nil {
				out = append(out, map[string]any{
					"id": id, "name": name, "confidence": conf, "hidden": hidden, "count": n,
				})
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"topics": out})
	})

	// PATCH /v1/topics/{id} {name} — fix a bad guess. Taken names refuse.
	mux.HandleFunc("PATCH /v1/topics/{id}", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		var in struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, "bad json", 400)
			return
		}
		name := cleanTopicName(in.Name)
		if name == "" {
			http.Error(w, "name is required", 400)
			return
		}
		user := tenantID(r)
		id := r.PathValue("id")
		res, err := pool.Exec(ctx,
			`UPDATE topics SET name=$1 WHERE id=$2 AND user_id=$3`, name, id, user)
		if err != nil {
			http.Error(w, "name taken", 409)
			return
		}
		if res.RowsAffected() == 0 {
			http.Error(w, "not found", 404)
			return
		}
		w.WriteHeader(204)
	})

	// POST /v1/topics/{id}/hide|unhide — quiet a noisy guess, reversibly.
	for _, route := range []struct {
		path string
		hide bool
	}{
		{"POST /v1/topics/{id}/hide", true},
		{"POST /v1/topics/{id}/unhide", false},
	} {
		mux.HandleFunc(route.path, func(w http.ResponseWriter, r *http.Request) {
			hide := strings.HasSuffix(r.URL.Path, "/hide")
			ctx := r.Context()
			user := tenantID(r)
			id := r.PathValue("id")
			res, err := pool.Exec(ctx,
				`UPDATE topics SET hidden=$1 WHERE id=$2 AND user_id=$3`, hide, id, user)
			if err != nil || res.RowsAffected() == 0 {
				http.Error(w, "not found", 404)
				return
			}
			w.WriteHeader(204)
		})
	}

	// POST /v1/topics/merge {keep_id, drop_id} — twins become one.
	// Links move, the dropped name is deleted, items never move.
	mux.HandleFunc("POST /v1/topics/merge", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		var in struct {
			KeepID string `json:"keep_id"`
			DropID string `json:"drop_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, "bad json", 400)
			return
		}
		if in.KeepID == "" || in.DropID == "" || in.KeepID == in.DropID {
			http.Error(w, "keep_id and drop_id must differ", 400)
			return
		}
		user := tenantID(r)
		var keepOwner, dropOwner string
		if err := pool.QueryRow(ctx, `SELECT user_id FROM topics WHERE id=$1`, in.KeepID).Scan(&keepOwner); err != nil || keepOwner != user {
			http.Error(w, "not found", 404)
			return
		}
		if err := pool.QueryRow(ctx, `SELECT user_id FROM topics WHERE id=$1`, in.DropID).Scan(&dropOwner); err != nil || dropOwner != user {
			http.Error(w, "not found", 404)
			return
		}
		if _, err := pool.Exec(ctx, `
		  INSERT INTO item_topics (item_id, topic_id, confidence)
		  SELECT item_id, $1, confidence FROM item_topics WHERE topic_id=$2
		  ON CONFLICT DO NOTHING`, in.KeepID, in.DropID); err != nil {
			log.Printf("topic merge move failed err=%T", err)
			http.Error(w, "internal error", 500)
			return
		}
		if _, err := pool.Exec(ctx, `DELETE FROM topics WHERE id=$1`, in.DropID); err != nil {
			log.Printf("topic merge drop failed err=%T", err)
			http.Error(w, "internal error", 500)
			return
		}
		w.WriteHeader(204)
	})
}
