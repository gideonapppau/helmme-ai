// Cleanup (§35, §37): duplicate suggestions, never actions.
// GET /v1/maintenance/duplicates -> groups with plain reasons.
// Exact hashes, same canonical links, and high text overlap. The user
// deletes; the system only points.
package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

type dupeItem struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Source     string `json:"source_type"`
	Domain     string `json:"domain"`
	CapturedAt string `json:"captured_at"`
}

type dupeGroup struct {
	Reason string     `json:"reason"`
	Items  []dupeItem `json:"items"`
}

func fetchDupeItems(ctx context.Context, pool *pgxpool.Pool, user, where, arg string) []dupeItem {
	rows, err := pool.Query(ctx, `
	  SELECT id, COALESCE(title,''), source_type, COALESCE(domain,''),
	         captured_at::text FROM items
	  WHERE user_id=$1 AND `+where+` ORDER BY captured_at`, user, arg)
	if err != nil {
		log.Printf("dupe fetch failed err=%T", err)
		return nil
	}
	defer rows.Close()
	var out []dupeItem
	for rows.Next() {
		var d dupeItem
		if err := rows.Scan(&d.ID, &d.Title, &d.Source, &d.Domain, &d.CapturedAt); err == nil {
			out = append(out, d)
		}
	}
	return out
}

func pairGroup(pool *pgxpool.Pool, ctx context.Context, user, x, y, reason string) dupeGroup {
	var out []dupeItem
	for _, id := range []string{x, y} {
		rows, err := pool.Query(ctx, `
		  SELECT id, COALESCE(title,''), source_type, COALESCE(domain,''),
		         captured_at::text FROM items WHERE id=$1 AND user_id=$2`, id, user)
		if err != nil {
			continue
		}
		for rows.Next() {
			var d dupeItem
			if err := rows.Scan(&d.ID, &d.Title, &d.Source, &d.Domain, &d.CapturedAt); err == nil {
				out = append(out, d)
			}
		}
		rows.Close()
	}
	return dupeGroup{Reason: reason, Items: out}
}

func registerMaintenanceRoutes(mux *http.ServeMux, pool *pgxpool.Pool) {
	mux.HandleFunc("POST /v1/maintenance/ignore", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		var in struct {
			AID string `json:"a_id"`
			BID string `json:"b_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, "bad json", 400)
			return
		}
		if in.AID == "" || in.AID == in.BID {
			http.Error(w, "bad pair", 400)
			return
		}
		user := tenantID(r)
		for _, id := range []string{in.AID, in.BID} {
			var owner string
			if err := pool.QueryRow(ctx, `SELECT user_id FROM items WHERE id=$1`, id).Scan(&owner); err != nil || owner != user {
				http.Error(w, "not found", 404)
				return
			}
		}
		a, b := in.AID, in.BID
		if a > b {
			a, b = b, a
		}
		if _, err := pool.Exec(ctx, `
		  INSERT INTO ignored_pairs (user_id, a_id, b_id) VALUES ($1,$2,$3) ON CONFLICT DO NOTHING`,
			user, a, b); err != nil {
			log.Printf("ignore failed err=%T", err)
			http.Error(w, "internal error", 500)
			return
		}
		w.WriteHeader(204)
	})

	mux.HandleFunc("GET /v1/maintenance/duplicates", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user := tenantID(r)
		var groups []dupeGroup

		// Same file bytes saved more than once.
		if rows, err := pool.Query(ctx, `
		  SELECT content_hash FROM items WHERE user_id=$1 AND status='active'
		  GROUP BY content_hash HAVING COUNT(*) > 1`, user); err == nil {
			for rows.Next() {
				var h string
				if err := rows.Scan(&h); err != nil {
					continue
				}
				items := fetchDupeItems(ctx, pool, user, "content_hash=$2", h)
				if len(items) > 1 {
					groups = append(groups, dupeGroup{Reason: "same-file", Items: items})
				}
			}
			rows.Close()
		} else {
			log.Printf("dupe hash query failed err=%T", err)
		}

		// Same canonical link from separate captures.
		if rows, err := pool.Query(ctx, `
		  SELECT canonical_url FROM items WHERE user_id=$1 AND status='active' AND canonical_url IS NOT NULL
		  GROUP BY canonical_url HAVING COUNT(*) > 1`, user); err == nil {
			for rows.Next() {
				var u string
				if err := rows.Scan(&u); err != nil {
					continue
				}
				items := fetchDupeItems(ctx, pool, user, "canonical_url=$2", u)
				if len(items) > 1 {
					groups = append(groups, dupeGroup{Reason: "same-link", Items: items})
				}
			}
			rows.Close()
		} else {
			log.Printf("dupe url query failed err=%T", err)
		}

		// Very similar text (conservative bar, small archives only).
		if rows, err := pool.Query(ctx, `
		  SELECT a.item_id, b.item_id FROM item_contents a
		  JOIN item_contents b ON b.item_id > a.item_id
		  JOIN items ia ON ia.id = a.item_id JOIN items ib ON ib.id = b.item_id
		  WHERE ia.user_id=$1 AND ib.user_id=$1 AND ia.status='active' AND ib.status='active'
		    AND LENGTH(a.extracted_text) > 100 AND LENGTH(b.extracted_text) > 100
		    AND similarity(a.extracted_text, b.extracted_text) > 0.6
		  LIMIT 20`, user); err == nil {
			seen := map[string]bool{}
			for rows.Next() {
				var x, y string
				if err := rows.Scan(&x, &y); err != nil {
					continue
				}
				key := x + "|" + y
				if seen[key] {
					continue
				}
				seen[key] = true
				if g := pairGroup(pool, ctx, user, x, y, "similar"); len(g.Items) == 2 {
					groups = append(groups, g)
				}
			}
			rows.Close()
		} else {
			log.Printf("dupe similarity query failed err=%T", err)
		}

		// Dismissed pairs never resurface. Within a group, the earlier
		// save stays and later dismissed ones fall away.
		ignored := map[string]bool{}
		if irows, err := pool.Query(ctx, `SELECT a_id::text, b_id::text FROM ignored_pairs WHERE user_id=$1`, user); err == nil {
			for irows.Next() {
				var a, b string
				if err := irows.Scan(&a, &b); err == nil {
					ignored[a+"|"+b] = true
				}
			}
			irows.Close()
		}
		pairIgnored := func(x, y string) bool {
			if x > y {
				x, y = y, x
			}
			return ignored[x+"|"+y]
		}
		var kept []dupeGroup
		for _, g := range groups {
			var members []dupeItem
			for _, it := range g.Items {
				drop := false
				for _, keptIt := range members {
					if pairIgnored(it.ID, keptIt.ID) {
						drop = true
						break
					}
				}
				if !drop {
					members = append(members, it)
				}
			}
			if len(members) > 1 {
				g.Items = members
				kept = append(kept, g)
			}
		}
		groups = kept
		if groups == nil {
			groups = []dupeGroup{}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"groups": groups})
	})
}
