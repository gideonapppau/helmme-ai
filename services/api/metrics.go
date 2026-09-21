// Usage metrics (§80). The north star is Useful Context Retrieved: finding
// and using, not collecting. Searches leave a small trace here; syntheses,
// opens, and captures already log themselves elsewhere. This endpoint reads
// the last 7 days back as one honest page. Internal eyes only.
// GET /v1/metrics
package main

import (
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

func registerMetricsRoutes(mux *http.ServeMux, pool *pgxpool.Pool) {
	mux.HandleFunc("GET /v1/metrics", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user := tenantID(r)
		count := func(table, extra string) int {
			var n int
			q := `SELECT COUNT(*) FROM ` + table + ` WHERE user_id=$1 AND created_at > now() - interval '7 days'` + extra
			_ = pool.QueryRow(ctx, q, user).Scan(&n)
			return n
		}
		searches := count("search_events", "")
		withHits := count("search_events", " AND hits > 0")
		out := map[string]any{
			"searches_7d":           searches,
			"searches_with_hits_7d": withHits,
			"syntheses_7d":          count("synthesis_runs", ""),
			"opens_7d":              count("item_events", " AND kind='opened'"),
			"captures_7d":           count("items", ""),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(out)
	})
}
