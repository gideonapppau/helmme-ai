// Corrections (§31): the user can remove system guesses, and the
// removal sticks. Enrichment checks memory before linking, so rejected
// pairs never come back. Claim dismissals ride on synthesis runs.
package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

// rejectedSet returns "kind:id" pairs the user has rejected for an item.
func rejectedSet(rows interface {
	Next() bool
	Scan(...interface{}) error
	Close()
}) map[string]bool {
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var kind, target string
		if err := rows.Scan(&kind, &target); err != nil {
			continue
		}
		out[kind+":"+target] = true
	}
	return out
}

func registerCorrectionRoutes(mux *http.ServeMux, pool *pgxpool.Pool) {
	// POST /v1/corrections {item_id, kind: topic|entity, target_id}
	mux.HandleFunc("POST /v1/corrections", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		var in struct {
			ItemID   string `json:"item_id"`
			Kind     string `json:"kind"`
			TargetID string `json:"target_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, "bad json", 400)
			return
		}
		if in.Kind != "topic" && in.Kind != "entity" {
			http.Error(w, "unknown kind", 400)
			return
		}
		user := tenantID(r)
		var owner string
		if err := pool.QueryRow(ctx, `SELECT user_id FROM items WHERE id=$1`, in.ItemID).Scan(&owner); err != nil || owner != user {
			http.Error(w, "not found", 404)
			return
		}
		table := "item_topics"
		key := "topic_id"
		if in.Kind == "entity" {
			table = "item_entities"
			key = "entity_id"
		}
		if _, err := pool.Exec(ctx, `DELETE FROM `+table+` WHERE item_id=$1 AND `+key+`=$2`, in.ItemID, in.TargetID); err != nil {
			log.Printf("correction delete failed err=%T", err)
			http.Error(w, "internal error", 500)
			return
		}
		_, _ = pool.Exec(ctx, `
		  INSERT INTO user_corrections (user_id, item_id, kind, target_id)
		  VALUES ($1,$2,$3,$4) ON CONFLICT DO NOTHING`,
			user, in.ItemID, in.Kind, in.TargetID)
		w.WriteHeader(204)
	})

	// PATCH /v1/synthesis/:id/claims {index, rejected}
	mux.HandleFunc("PATCH /v1/synthesis/{id}/claims", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user := tenantID(r)
		id := r.PathValue("id")
		var in struct {
			Index    int  `json:"index"`
			Rejected bool `json:"rejected"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, "bad json", 400)
			return
		}
		var owner, claimsRaw string
		if err := pool.QueryRow(ctx, `SELECT user_id, claims::text FROM synthesis_runs WHERE id=$1`,
			id).Scan(&owner, &claimsRaw); err != nil || owner != user {
			http.Error(w, "not found", 404)
			return
		}
		var claims []map[string]any
		if err := json.Unmarshal([]byte(claimsRaw), &claims); err != nil {
			http.Error(w, "internal error", 500)
			return
		}
		if in.Index < 0 || in.Index >= len(claims) {
			http.Error(w, "unknown claim", 400)
			return
		}
		claims[in.Index]["rejected"] = in.Rejected
		updated, _ := json.Marshal(claims)
		if _, err := pool.Exec(ctx, `UPDATE synthesis_runs SET claims=$1 WHERE id=$2`, updated, id); err != nil {
			http.Error(w, "internal error", 500)
			return
		}
		w.WriteHeader(204)
	})
}
