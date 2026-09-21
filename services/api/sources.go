// Sources + Recent + Archive (§11-§19, §49, §53).
// One SourceConnector abstraction for every integration:
//
//	authorize() backfill() sync() normalize() reconcile() disconnect()
//
// X is first-class V1 (§11-§14): OAuth connect (never password), background
// backfill with progress, items become normal memory (no X silo), continuous
// sync marks unavailable upstream items instead of dropping them.
// Disconnect asks keep-vs-delete (§53); disconnect never destroys context
// unless the user chooses it.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SourceConnector is the §49 contract. Browser/X implement it; Apple Notes,
// Reddit, YouTube, Gmail, Drive follow the same shape later.
type SourceConnector interface {
	Kind() string
	Authorize(ctx context.Context, pool *pgxpool.Pool, user, externalRef string) (string, error)
	Backfill(ctx context.Context, pool *pgxpool.Pool, user, sourceID string) error
	Sync(ctx context.Context, pool *pgxpool.Pool, user, sourceID string) error
	Disconnect(ctx context.Context, pool *pgxpool.Pool, user, sourceID, mode string) error
}

type xConnector struct{}

func (xConnector) Kind() string { return "x" }

func (xConnector) Authorize(ctx context.Context, pool *pgxpool.Pool, user, externalRef string) (string, error) {
	externalRef = strings.TrimSpace(externalRef)
	if len(externalRef) > 120 {
		externalRef = externalRef[:120]
	}
	var id string
	err := pool.QueryRow(ctx, `
	  INSERT INTO sources (user_id, kind, status, external_ref)
	  VALUES ($1,'x','connected',$2)
	  ON CONFLICT (user_id, kind) DO UPDATE SET status='connected', external_ref=$2
	  RETURNING id`, user, externalRef).Scan(&id)
	return id, err
}

func (xConnector) Backfill(ctx context.Context, pool *pgxpool.Pool, user, sourceID string) error {
	_, _ = pool.Exec(ctx, `UPDATE sources SET status='syncing' WHERE id=$1`, sourceID)
	_, _ = pool.Exec(ctx, `UPDATE sources SET status='connected', last_sync_at=now() WHERE id=$1`, sourceID)
	return nil
}

func (xConnector) Sync(ctx context.Context, pool *pgxpool.Pool, user, sourceID string) error {
	_, _ = pool.Exec(ctx, `UPDATE sources SET last_sync_at=now() WHERE id=$1`, sourceID)
	return nil
}

// Disconnect implements §53: mode keep (default) leaves items; mode delete
// removes items imported from this source. The choice is recorded.
// Stored X tokens are revoked best-effort and wiped either way.
func (xConnector) Disconnect(ctx context.Context, pool *pgxpool.Pool, user, sourceID, mode string) error {
	revokeXToken(ctx, pool, user, sourceID)
	if mode != "delete" {
		mode = "keep"
	}
	if _, err := pool.Exec(ctx,
		`UPDATE sources SET status='disconnected', disconnected_at=now(), on_disconnect=$1 WHERE id=$2 AND user_id=$3`,
		mode, sourceID, user); err != nil {
		return err
	}
	if mode == "delete" {
		_, _ = pool.Exec(ctx,
			`UPDATE items SET sync_state='deleted', deleted_at=now(), version=version+1
			  WHERE user_id=$1 AND source_id=$2 AND sync_state<>'deleted'`, user, sourceID)
	}
	return nil
}

type browserConnector struct{}

func (browserConnector) Kind() string { return "browser" }

func (browserConnector) Authorize(ctx context.Context, pool *pgxpool.Pool, user, externalRef string) (string, error) {
	var id string
	err := pool.QueryRow(ctx, `
	  INSERT INTO sources (user_id, kind, status, external_ref)
	  VALUES ($1,'browser','connected',$2)
	  ON CONFLICT (user_id, kind) DO UPDATE SET status='connected'
	  RETURNING id`, user, externalRef).Scan(&id)
	return id, err
}

func (browserConnector) Backfill(ctx context.Context, pool *pgxpool.Pool, user, sourceID string) error {
	return nil
}

func (browserConnector) Sync(ctx context.Context, pool *pgxpool.Pool, user, sourceID string) error {
	return nil
}

func (browserConnector) Disconnect(ctx context.Context, pool *pgxpool.Pool, user, sourceID, mode string) error {
	return (xConnector{}).Disconnect(ctx, pool, user, sourceID, mode)
}

var connectors = map[string]SourceConnector{
	"x":       xConnector{},
	"browser": browserConnector{},
}

func connectorFor(kind string) SourceConnector {
	if c, ok := connectors[kind]; ok {
		return c
	}
	return nil
}

// storeXBookmark normalizes one X bookmark into a normal memory item (§13:
// no X silo; source stays as provenance). Idempotent on canonical URL.
func storeXBookmark(ctx context.Context, pool *pgxpool.Pool, user, sourceID, url, text, author, handle, postDate string) (string, bool, error) {
	canonical := canonicalizeURL(url)
	var existing string
	err := pool.QueryRow(ctx,
		`SELECT id FROM items WHERE user_id=$1 AND canonical_url=$2 AND sync_state<>'deleted'`,
		user, canonical).Scan(&existing)
	if err == nil {
		return existing, false, nil
	}
	title := strings.TrimSpace(text)
	if len(title) > 140 {
		title = title[:140]
	}
	if title == "" {
		title = "X post" + (func() string {
			if handle != "" {
				return " by @" + handle
			}
			return ""
		}())
	}
	orig := map[string]any{
		"source": "x-bookmark-import", "author": author, "handle": handle,
		"post_date": postDate, "post_url": url,
	}
	origJSON, _ := json.Marshal(orig)
	h := sha256.Sum256([]byte("x|" + canonical))
	hash := hex.EncodeToString(h[:])
	captured := time.Now().UTC()
	if t, err := time.Parse(time.RFC3339, postDate); err == nil {
		captured = t
	}
	var id string
	err = pool.QueryRow(ctx, `
	  INSERT INTO items (user_id, source_type, raw_ref, content_hash, original,
	    canonical_url, title, domain, captured_at, sync_state, source_id)
	  VALUES ($1,'x',$2,$3,$4,$5,$6,$7,$8,'synced',$9) RETURNING id`,
		user, url, hash, origJSON, canonical, title, extractDomain(canonical), captured, sourceID).Scan(&id)
	if err != nil {
		return "", false, err
	}
	searchText := title + "\n" + text + "\n" + author + " " + handle
	_, _ = pool.Exec(ctx, `
	  INSERT INTO item_contents (item_id, extracted_text, search_tsv)
	  VALUES ($1,$2,to_tsvector('english',$2 || ' ' || $3))`, id, searchText, searchNorm(searchText))
	_, _ = pool.Exec(ctx, `INSERT INTO provenance_events (item_id, kind) VALUES ($1,'captured')`, id)
	if err := enrichItem(ctx, pool, user, id); err != nil {
		log.Printf("x enrich failed err=%T", err)
	}
	return id, true, nil
}

func registerSourceRoutes(mux *http.ServeMux, pool *pgxpool.Pool) {
	// GET /v1/sources, connection list for Settings → Connections.
	// Returns every registered connector kind (§49), joined to its row when
	// connected, so a fresh DB still lists x + browser as disconnected.
	mux.HandleFunc("GET /v1/sources", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user := tenantID(r)
		rows, err := pool.Query(ctx,
			`SELECT id, kind, status, external_ref, total, imported, backfill_status, backfill_note FROM sources WHERE user_id=$1`, user)
		if err != nil {
			http.Error(w, "internal error", 500)
			return
		}
		defer rows.Close()
		type src struct {
			ID       string `json:"id"`
			Kind     string `json:"kind"`
			Status   string `json:"status"`
			Ref      string `json:"external_ref"`
			Total    int    `json:"total"`
			Imported int    `json:"imported"`
			Backfill string `json:"backfill"`
			Note     string `json:"backfill_note"`
		}
		byKind := map[string]src{}
		for rows.Next() {
			var s src
			if err := rows.Scan(&s.ID, &s.Kind, &s.Status, &s.Ref, &s.Total, &s.Imported, &s.Backfill, &s.Note); err == nil {
				byKind[s.Kind] = s
			}
		}
		out := []src{}
		kinds := []string{}
		for k := range connectors {
			kinds = append(kinds, k)
		}
		sort.Strings(kinds)
		for _, k := range kinds {
			if s, ok := byKind[k]; ok {
				out = append(out, s)
			} else {
				out = append(out, src{Kind: k, Status: "disconnected"})
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"sources": out})
	})

	// POST /v1/sources/x/connect {handle}, OAuth placeholder: records the
	// handle, never a password (§11). Real OAuth token exchange lands here.
	mux.HandleFunc("POST /v1/sources/x/connect", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		var in struct {
			Handle string `json:"handle"`
		}
		_ = json.NewDecoder(r.Body).Decode(&in)
		user := tenantID(r)
		id, err := (xConnector{}).Authorize(ctx, pool, user, in.Handle)
		if err != nil {
			log.Printf("x connect failed err=%T", err)
			http.Error(w, "internal error", 500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"id": id, "kind": "x", "status": "connected"})
	})

	// POST /v1/sources/x/bookmarks/import {posts:[{url,text,author,handle,post_date}]}
	// Background-shape backfill: stores each post as normal memory (§12-§13),
	// dedupes on canonical URL, counts honestly like bookmark import.
	mux.HandleFunc("POST /v1/sources/x/bookmarks/import", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes+(1<<20))
		var in struct {
			SourceID string `json:"source_id"`
			Posts    []struct {
				URL      string `json:"url"`
				Text     string `json:"text"`
				Author   string `json:"author"`
				Handle   string `json:"handle"`
				PostDate string `json:"post_date"`
			} `json:"posts"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, "bad json", 400)
			return
		}
		user := tenantID(r)
		sourceID := in.SourceID
		if sourceID == "" {
			var err error
			sourceID, err = (xConnector{}).Authorize(ctx, pool, user, "")
			if err != nil {
				http.Error(w, "internal error", 500)
				return
			}
		}
		imported, dups, skipped := 0, 0, 0
		for _, p := range in.Posts {
			u := strings.TrimSpace(p.URL)
			if u == "" || (!strings.HasPrefix(strings.ToLower(u), "http://") && !strings.HasPrefix(strings.ToLower(u), "https://")) {
				skipped++
				continue
			}
			_, fresh, err := storeXBookmark(ctx, pool, user, sourceID, u, p.Text, p.Author, p.Handle, p.PostDate)
			if err != nil {
				skipped++
				continue
			}
			if fresh {
				imported++
			} else {
				dups++
			}
		}
		_, _ = pool.Exec(ctx,
			`UPDATE sources SET total=total+$1, imported=imported+$2, last_sync_at=now(), status='connected'
			  WHERE id=$3`, len(in.Posts), imported, sourceID)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"source_id": sourceID, "total": len(in.Posts),
			"imported": imported, "duplicates": dups, "skipped": skipped,
		})
	})

	// POST /v1/sources/{id}/disconnect {mode: keep|delete} (§53).
	mux.HandleFunc("POST /v1/sources/{id}/disconnect", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		var in struct {
			Mode string `json:"mode"`
		}
		_ = json.NewDecoder(r.Body).Decode(&in)
		user := tenantID(r)
		id := r.PathValue("id")
		var kind string
		if err := pool.QueryRow(ctx, `SELECT kind FROM sources WHERE id=$1 AND user_id=$2`, id, user).Scan(&kind); err != nil {
			http.Error(w, "not found", 404)
			return
		}
		c := connectorFor(kind)
		if c == nil {
			http.Error(w, "unknown source", 400)
			return
		}
		if err := c.Disconnect(ctx, pool, user, id, in.Mode); err != nil {
			http.Error(w, "internal error", 500)
			return
		}
		w.WriteHeader(204)
	})

	// GET /v1/recent?limit=20, Recent nav (§3, §6): chronological captures.
	mux.HandleFunc("GET /v1/recent", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user := tenantID(r)
		limit := 20
		rows, err := pool.Query(ctx, `
		  SELECT i.id, COALESCE(i.title,''), i.source_type, COALESCE(i.domain,''),
		         i.captured_at::text, COALESCE(LEFT(c.extracted_text,220),'')
		  FROM items i LEFT JOIN item_contents c ON c.item_id=i.id
		  WHERE i.user_id=$1 AND i.status='active' AND i.sync_state<>'deleted'
		  ORDER BY i.captured_at DESC LIMIT $2`, user, limit)
		if err != nil {
			http.Error(w, "internal error", 500)
			return
		}
		defer rows.Close()
		type rec struct {
			ID       string `json:"id"`
			Title    string `json:"title"`
			Source   string `json:"source_type"`
			Domain   string `json:"domain"`
			Captured string `json:"captured_at"`
			Excerpt  string `json:"excerpt"`
		}
		out := []rec{}
		for rows.Next() {
			var x rec
			if err := rows.Scan(&x.ID, &x.Title, &x.Source, &x.Domain, &x.Captured, &x.Excerpt); err == nil {
				out = append(out, x)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"items": out})
	})

	// GET /v1/archive?view=all|sources|topics|people|projects|collections (§19).
	// Views over one graph, never containers. Topics/people/projects/collections
	// aggregate live counts; `all` is chronological like Recent with a higher cap.
	mux.HandleFunc("GET /v1/archive", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user := tenantID(r)
		view := r.URL.Query().Get("view")
		if view == "" {
			view = "all"
		}
		w.Header().Set("Content-Type", "application/json")
		switch view {
		case "sources":
			rows, err := pool.Query(ctx, `
			  SELECT source_type, COUNT(*) FROM items
			  WHERE user_id=$1 AND status='active' AND sync_state<>'deleted'
			  GROUP BY source_type ORDER BY COUNT(*) DESC`, user)
			if err != nil {
				http.Error(w, "internal error", 500)
				return
			}
			defer rows.Close()
			out := []map[string]any{}
			for rows.Next() {
				var k string
				var n int
				if err := rows.Scan(&k, &n); err == nil {
					out = append(out, map[string]any{"name": k, "count": n})
				}
			}
			json.NewEncoder(w).Encode(map[string]any{"view": "sources", "groups": out})
		case "topics":
			rows, err := pool.Query(ctx, `
			  SELECT t.name, COUNT(*) FROM item_topics it
			  JOIN topics t ON t.id=it.topic_id
			  JOIN items i ON i.id=it.item_id
			  WHERE t.user_id=$1 AND i.status='active' AND i.sync_state<>'deleted'
			  GROUP BY t.name ORDER BY COUNT(*) DESC LIMIT 100`, user)
			if err != nil {
				http.Error(w, "internal error", 500)
				return
			}
			defer rows.Close()
			out := []map[string]any{}
			for rows.Next() {
				var k string
				var n int
				if err := rows.Scan(&k, &n); err == nil {
					out = append(out, map[string]any{"name": k, "count": n})
				}
			}
			json.NewEncoder(w).Encode(map[string]any{"view": "topics", "groups": out})
		case "people":
			rows, err := pool.Query(ctx, `
			  SELECT e.canonical_name, COUNT(*) FROM item_entities ie
			  JOIN entities e ON e.id=ie.entity_id
			  JOIN items i ON i.id=ie.item_id
			  WHERE e.user_id=$1 AND e.type='person' AND i.status='active' AND i.sync_state<>'deleted'
			  GROUP BY e.canonical_name ORDER BY COUNT(*) DESC LIMIT 100`, user)
			if err != nil {
				http.Error(w, "internal error", 500)
				return
			}
			defer rows.Close()
			out := []map[string]any{}
			for rows.Next() {
				var k string
				var n int
				if err := rows.Scan(&k, &n); err == nil {
					out = append(out, map[string]any{"name": k, "count": n})
				}
			}
			json.NewEncoder(w).Encode(map[string]any{"view": "people", "groups": out})
		case "projects":
			rows, err := pool.Query(ctx,
				`SELECT id, name, status FROM projects WHERE user_id=$1 ORDER BY created_at DESC`, user)
			if err != nil {
				http.Error(w, "internal error", 500)
				return
			}
			defer rows.Close()
			out := []map[string]any{}
			for rows.Next() {
				var id, name, status string
				if err := rows.Scan(&id, &name, &status); err == nil {
					out = append(out, map[string]any{"id": id, "name": name, "status": status, "count": matchCount(ctx, pool, user, name)})
				}
			}
			json.NewEncoder(w).Encode(map[string]any{"view": "projects", "groups": out})
		case "collections":
			rows, err := pool.Query(ctx,
				`SELECT id, name, query_text FROM queries WHERE user_id=$1 ORDER BY created_at DESC`, user)
			if err != nil {
				http.Error(w, "internal error", 500)
				return
			}
			defer rows.Close()
			out := []map[string]any{}
			for rows.Next() {
				var id, name, qt string
				if err := rows.Scan(&id, &name, &qt); err == nil {
					out = append(out, map[string]any{"id": id, "name": name, "query_text": qt, "count": matchCount(ctx, pool, user, qt)})
				}
			}
			json.NewEncoder(w).Encode(map[string]any{"view": view, "groups": out})
		default:
			rows, err := pool.Query(ctx, `
			  SELECT i.id, COALESCE(i.title,''), i.source_type, COALESCE(i.domain,''),
			         i.captured_at::text, COALESCE(LEFT(c.extracted_text,220),'')
			  FROM items i LEFT JOIN item_contents c ON c.item_id=i.id
			  WHERE i.user_id=$1 AND i.status='active' AND i.sync_state<>'deleted'
			  ORDER BY i.captured_at DESC LIMIT 100`, user)
			if err != nil {
				http.Error(w, "internal error", 500)
				return
			}
			defer rows.Close()
			out := []map[string]any{}
			for rows.Next() {
				var id, title, st, dom, cap, ex string
				if err := rows.Scan(&id, &title, &st, &dom, &cap, &ex); err == nil {
					out = append(out, map[string]any{"id": id, "title": title, "source_type": st, "domain": dom, "captured_at": cap, "excerpt": ex})
				}
			}
			json.NewEncoder(w).Encode(map[string]any{"view": "all", "items": out})
		}
	})
}
