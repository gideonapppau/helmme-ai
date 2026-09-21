// Behavior settings (§38). Quiet / Balanced / Proactive are volume knobs
// for resurfacing, not personalities. They live on the server so the phone,
// the desktop, and the web all behave the same way.
// GET /v1/settings -> {resurface_mode}
// POST /v1/settings {resurface_mode} -> saved
package main

import (
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

// resurfaceModes names the only moods allowed. Anything else is refused.
var resurfaceModes = map[string]bool{
	"quiet": true, "balanced": true, "proactive": true,
}

// resurfaceLimit turns a mood into a number. Plain words: quiet means
// "at most one tap on the shoulder", proactive means "up to five".
func resurfaceLimit(mode string) int {
	switch mode {
	case "proactive":
		return 5
	case "balanced":
		return 3
	default:
		return 1
	}
}

func registerSettingsRoutes(mux *http.ServeMux, pool *pgxpool.Pool) {
	mux.HandleFunc("GET /v1/settings", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user := tenantID(r)
		mode := "quiet" // no row yet means Quiet, the default
		enrich := true  // and enrichment on
		_ = pool.QueryRow(ctx,
			`SELECT resurface_mode, ai_enrichment FROM user_settings WHERE user_id=$1`, user).Scan(&mode, &enrich)
		if !resurfaceModes[mode] {
			mode = "quiet"
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"resurface_mode": mode, "ai_enrichment": enrich})
	})

	mux.HandleFunc("POST /v1/settings", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		var in struct {
			ResurfaceMode *string `json:"resurface_mode"`
			AIEnrichment  *bool   `json:"ai_enrichment"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, "bad json", 400)
			return
		}
		// Pointers tell "not sent" apart from "sent false": each knob
		// changes only when named.
		if in.ResurfaceMode != nil && !resurfaceModes[*in.ResurfaceMode] {
			http.Error(w, "unknown resurface_mode", 400)
			return
		}
		user := tenantID(r)
		if in.ResurfaceMode != nil {
			_, _ = pool.Exec(ctx, `
			  INSERT INTO user_settings (user_id, resurface_mode, updated_at)
			  VALUES ($1,$2,now())
			  ON CONFLICT (user_id) DO UPDATE SET resurface_mode=$2, updated_at=now()`,
				user, *in.ResurfaceMode)
		}
		if in.AIEnrichment != nil {
			_, _ = pool.Exec(ctx, `
			  INSERT INTO user_settings (user_id, ai_enrichment, updated_at)
			  VALUES ($1,$2,now())
			  ON CONFLICT (user_id) DO UPDATE SET ai_enrichment=$2, updated_at=now()`,
				user, *in.AIEnrichment)
		}
		mode := "quiet"
		enrich := true
		_ = pool.QueryRow(ctx,
			`SELECT resurface_mode, ai_enrichment FROM user_settings WHERE user_id=$1`, user).Scan(&mode, &enrich)
		if !resurfaceModes[mode] {
			mode = "quiet"
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"resurface_mode": mode, "ai_enrichment": enrich})
	})
}
