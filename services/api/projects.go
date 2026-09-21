// Projects (§33): living views, not folders.
// A project is a saved question ("everything about Corsa Cloud") that stays
// fresh as new saves arrive. Confirming, renaming, or removing a project
// never moves or deletes a single item, views only.
// Suggesting is plain counting (topics with 3+ saves); clever clustering
// arrives later. Every suggestion carries its reason.
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// cleanProjectName trims a project name to something displayable.
// Plain words: names are short, never blank.
func cleanProjectName(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 120 {
		s = s[:120]
	}
	return s
}

func registerProjectRoutes(mux *http.ServeMux, pool *pgxpool.Pool) {
	// GET /v1/projects, every view with a live count.
	mux.HandleFunc("GET /v1/projects", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user := tenantID(r)
		rows, err := pool.Query(ctx,
			`SELECT id, name, status, reason FROM projects WHERE user_id=$1 ORDER BY status, created_at DESC`, user)
		if err != nil {
			http.Error(w, "internal error", 500)
			return
		}
		defer rows.Close()
		out := []map[string]any{}
		for rows.Next() {
			var id, name, status, reason string
			if err := rows.Scan(&id, &name, &status, &reason); err != nil {
				continue
			}
			out = append(out, map[string]any{
				"id": id, "name": name, "status": status, "reason": reason,
				"count": matchCount(ctx, pool, user, name),
			})
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"projects": out})
	})

	// POST /v1/projects/suggest, look for clusters (topics with 3+ saves)
	// and propose them. Proposals never made twice; confirming is separate.
	mux.HandleFunc("POST /v1/projects/suggest", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user := tenantID(r)
		rows, err := pool.Query(ctx, `
		  SELECT t.name, COUNT(*) FROM item_topics it
		  JOIN topics t ON t.id = it.topic_id
		  JOIN items i ON i.id = it.item_id
		  WHERE t.user_id=$1 AND NOT t.hidden AND i.status='active' AND i.sync_state<>'deleted'
		  GROUP BY t.name HAVING COUNT(*) >= 3
		  ORDER BY COUNT(*) DESC LIMIT 10`, user)
		if err != nil {
			http.Error(w, "internal error", 500)
			return
		}
		defer rows.Close()
		made := []map[string]any{}
		for rows.Next() {
			var name string
			var n int
			if err := rows.Scan(&name, &n); err != nil {
				continue
			}
			name = cleanProjectName(name)
			if name == "" {
				continue
			}
			reason := "You have " + strconv.Itoa(n) + " saves about " + name + ". Confirm to keep this view."
			var id string
			err := pool.QueryRow(ctx, `
			  INSERT INTO projects (user_id, name, status, reason)
			  VALUES ($1,$2,'suggested',$3)
			  ON CONFLICT (user_id, name) DO NOTHING RETURNING id`,
				user, name, reason).Scan(&id)
			if err != nil {
				continue // already suggested, proposals never repeat
			}
			made = append(made, map[string]any{"id": id, "name": name, "reason": reason, "count": n})
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"suggested": made})
	})

	// POST /v1/projects {name}, declare a project yourself. Confirmed at birth.
	mux.HandleFunc("POST /v1/projects", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		var in struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, "bad json", 400)
			return
		}
		name := cleanProjectName(in.Name)
		if name == "" {
			http.Error(w, "name is required", 400)
			return
		}
		user := tenantID(r)
		var id string
		if err := pool.QueryRow(ctx, `
		  INSERT INTO projects (user_id, name, status, reason)
		  VALUES ($1,$2,'confirmed','Declared by you.')
		  ON CONFLICT (user_id, name) DO UPDATE SET status='confirmed', updated_at=now()
		  RETURNING id`, user, name).Scan(&id); err != nil {
			log.Printf("project create failed err=%T", err)
			http.Error(w, "internal error", 500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"id": id, "name": name, "status": "confirmed"})
	})

	// POST /v1/projects/{id}/confirm, a suggestion becomes a living view.
	mux.HandleFunc("POST /v1/projects/{id}/confirm", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user := tenantID(r)
		id := r.PathValue("id")
		res, err := pool.Exec(ctx,
			`UPDATE projects SET status='confirmed', updated_at=now() WHERE id=$1 AND user_id=$2`,
			id, user)
		if err != nil || res.RowsAffected() == 0 {
			http.Error(w, "not found", 404)
			return
		}
		w.WriteHeader(204)
	})

	// PATCH /v1/projects/{id} {name}, rename the view, not the items.
	mux.HandleFunc("PATCH /v1/projects/{id}", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		var in struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, "bad json", 400)
			return
		}
		name := cleanProjectName(in.Name)
		if name == "" {
			http.Error(w, "name is required", 400)
			return
		}
		user := tenantID(r)
		id := r.PathValue("id")
		if _, err := pool.Exec(ctx,
			`UPDATE projects SET name=$1, updated_at=now() WHERE id=$2 AND user_id=$3`,
			name, id, user); err != nil {
			http.Error(w, "name taken or not found", 409)
			return
		}
		w.WriteHeader(204)
	})

	// DELETE /v1/projects/{id}, remove the view. Items stay exactly where they are.
	mux.HandleFunc("DELETE /v1/projects/{id}", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user := tenantID(r)
		id := r.PathValue("id")
		res, err := pool.Exec(ctx, `DELETE FROM projects WHERE id=$1 AND user_id=$2`, id, user)
		if err != nil || res.RowsAffected() == 0 {
			http.Error(w, "not found", 404)
			return
		}
		w.WriteHeader(204)
	})
}
