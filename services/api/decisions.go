// Decision log (§75, App. J). Every machine judgment is written down with
// what went in and what came out, so any answer can be audited later.
// Plain words: the judges show their work.
// POST /v1/decision-events {component, model, input, result}
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// decisionComponents names the only judges allowed to log here.
// Anything else is rejected, the log stays clean and countable.
var decisionComponents = map[string]bool{
	"query-router": true, "memory-state": true, "source-classifier": true,
	"relationship": true, "claim-verifier": true,
}

func registerDecisionRoutes(mux *http.ServeMux, pool *pgxpool.Pool) {
	mux.HandleFunc("POST /v1/decision-events", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		var in struct {
			Component string         `json:"component"`
			Model     string         `json:"model"`
			Input     map[string]any `json:"input"`
			Result    map[string]any `json:"result"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, "bad json", 400)
			return
		}
		if !decisionComponents[in.Component] {
			http.Error(w, "unknown component", 400)
			return
		}
		in.Model = strings.TrimSpace(in.Model)
		if in.Model == "" || len(in.Model) > 120 {
			http.Error(w, "invalid model", 400)
			return
		}
		if in.Input == nil || in.Result == nil {
			http.Error(w, "input and result are required", 400)
			return
		}
		inJSON, _ := json.Marshal(in.Input)
		resJSON, _ := json.Marshal(in.Result)
		if len(inJSON) > 20000 || len(resJSON) > 20000 {
			http.Error(w, "input or result too large", 400)
			return
		}
		var id string
		if err := pool.QueryRow(ctx, `
		  INSERT INTO decision_events (component, model, input, result)
		  VALUES ($1,$2,$3,$4) RETURNING id`,
			in.Component, in.Model, inJSON, resJSON).Scan(&id); err != nil {
			log.Printf("decision log failed err=%T", err)
			http.Error(w, "internal error", 500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"id": id})
	})
}
