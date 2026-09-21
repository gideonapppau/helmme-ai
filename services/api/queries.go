// Saved questions (§31): collections are queries, not containers.
// POST /v1/queries {name, query_text} — stores the question.
// GET /v1/queries — lists with fresh counts and new-since numbers.
// POST /v1/queries/:id/open — stamps the view after looking.
// DELETE /v1/queries/:id — removes it.
package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// matchCount answers "how many items match this question right now".
// Strict matching only: the badge counts on certainty, never guesses.
func matchCount(ctx context.Context, pool *pgxpool.Pool, user, rawQuery string) int {
	query := searchNorm(strings.TrimSpace(rawQuery))
	if query == "" {
		return 0
	}
	var n int
	if err := pool.QueryRow(ctx, `
	  SELECT COUNT(*) FROM items i JOIN item_contents c ON c.item_id=i.id
	  WHERE i.user_id=$2 AND i.status='active' AND c.search_tsv @@ plainto_tsquery('english',$1)`,
		query, user).Scan(&n); err != nil {
		log.Printf("match count failed err=%T", err)
		return 0
	}
	return n
}

func registerQueryRoutes(mux *http.ServeMux, pool *pgxpool.Pool) {
	mux.HandleFunc("POST /v1/queries", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		var in struct {
			Name  string `json:"name"`
			Query string `json:"query_text"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, "bad json", 400)
			return
		}
		name := strings.TrimSpace(in.Name)
		query := strings.TrimSpace(in.Query)
		if query == "" || len(query) > 500 {
			http.Error(w, "invalid query", 400)
			return
		}
		if name == "" {
			name = query
		}
		if len(name) > 120 {
			name = name[:120]
		}
		user := tenantID(r)
		var id string
		if err := pool.QueryRow(ctx, `
		  INSERT INTO queries (user_id, name, query_text) VALUES ($1,$2,$3) RETURNING id`,
			user, name, query).Scan(&id); err != nil {
			log.Printf("query save failed err=%T", err)
			http.Error(w, "internal error", 500)
			return
		}
		n := matchCount(ctx, pool, user, query)
		_, _ = pool.Exec(ctx, `UPDATE queries SET last_count=$1, last_run_at=now() WHERE id=$2`, n, id)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"id": id})
	})

	mux.HandleFunc("GET /v1/queries", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user := tenantID(r)
		rows, err := pool.Query(ctx, `
		  SELECT id, name, query_text, last_count FROM queries
		  WHERE user_id=$1 ORDER BY created_at DESC`, user)
		if err != nil {
			http.Error(w, "internal error", 500)
			return
		}
		defer rows.Close()
		type view struct {
			ID    string `json:"id"`
			Name  string `json:"name"`
			Query string `json:"query_text"`
			Count int    `json:"count"`
			New   int    `json:"new"`
		}
		var out []view
		for rows.Next() {
			var v view
			var last int
			var qtext string
			if err := rows.Scan(&v.ID, &v.Name, &qtext, &last); err != nil {
				continue
			}
			v.Query = qtext
			v.Count = matchCount(ctx, pool, user, qtext)
			if v.Count > last {
				v.New = v.Count - last
			}
			out = append(out, v)
		}
		if out == nil {
			out = []view{}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"views": out})
	})

	mux.HandleFunc("POST /v1/queries/{id}/open", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user := tenantID(r)
		id := r.PathValue("id")
		var qtext string
		var owner string
		if err := pool.QueryRow(ctx, `SELECT query_text, user_id FROM queries WHERE id=$1`,
			id).Scan(&qtext, &owner); err != nil || owner != user {
			http.Error(w, "not found", 404)
			return
		}
		n := matchCount(ctx, pool, user, qtext)
		_, _ = pool.Exec(ctx, `UPDATE queries SET last_count=$1, last_run_at=now() WHERE id=$2`, n, id)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"count": n})
	})

	mux.HandleFunc("DELETE /v1/queries/{id}", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user := tenantID(r)
		id := r.PathValue("id")
		res, err := pool.Exec(ctx, `DELETE FROM queries WHERE id=$1 AND user_id=$2`, id, user)
		if err != nil || res.RowsAffected() == 0 {
			http.Error(w, "not found", 404)
			return
		}
		w.WriteHeader(204)
	})
}
