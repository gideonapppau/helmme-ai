// Phase-0 slice: capture + FTS search. Constitution: docs/APP_CONSTITUTION.md.
// Endpoints: POST /v1/items, GET /v1/items/:id, POST /v1/search, GET /healthz,
// sources (X first-class §11-§14, §49), recent + archive views (§3, §19).
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ItemIn struct {
	SourceType string         `json:"source_type"`
	RawRef     string         `json:"raw_ref"`
	Original   map[string]any `json:"original"`
	Title      string         `json:"title"`
	// ClientID is the phone/app's own id for this save. Send it when the save
	// might be retried (offline queue). Same id twice = one save, never twins.
	ClientID string `json:"client_id"`
	// CapturedAt is when the save really happened, in RFC3339 form
	// ("2026-09-21T10:00:00Z"). Empty means "right now".
	CapturedAt string `json:"captured_at"`
}

const (
	maxBodyBytes   = 1 << 20 // 1MB (§26: size limits)
	maxTitleLen    = 500
	maxRawRefLen   = 8000
	maxClientIDLen = 120
)

var allowedSourceTypes = map[string]bool{
	"url": true, "text": true, "file": true, "pdf": true,
	"image": true, "bookmark": true, "note": true,
	"x": true, "share_sheet": true, // §11-§14 X first-class; §10 share sheet
}

// validateItem enforces §26: enum, type, size validation before any DB touch.
// Untrusted content is data, never instructions (§25/§27).
func validateItem(in *ItemIn) string {
	if !allowedSourceTypes[in.SourceType] {
		return "invalid source_type"
	}
	if len(in.RawRef) == 0 || len(in.RawRef) > maxRawRefLen {
		return "raw_ref length out of bounds"
	}
	if len(in.Title) > maxTitleLen {
		return "title too long"
	}
	if len(in.Original) > 100 {
		return "original too large"
	}
	if len(in.ClientID) > maxClientIDLen {
		return "client_id too long"
	}
	if in.CapturedAt != "" && !validCaptureTime(in.CapturedAt) {
		return "captured_at must be RFC3339 like 2026-09-21T10:00:00Z"
	}
	return ""
}

// validCaptureTime checks the "when did this really happen" stamp.
// Plain words: the date must be readable as a real point in time.
func validCaptureTime(s string) bool {
	_, err := time.Parse(time.RFC3339, s)
	return err == nil
}

// withQuotedWords files what the user highlighted or jotted alongside a
// save (extension selected_text / note_text). Plain words: your highlight
// is findable, not just stored. Capped so a pasted novel can't bloat the index.
func withQuotedWords(text string, original map[string]any) string {
	for _, key := range []string{"selected_text", "note_text"} {
		if s, _ := original[key].(string); strings.TrimSpace(s) != "" {
			text += "\n" + s
		}
	}
	if len(text) > 16000 {
		text = text[:16000]
	}
	return text
}

// searchNorm de-slugs text for the index: URL paths like /quick-save-test
// become findable words. The stored extracted_text is untouched, this twin
// feeds the tsvector only. Shared by all three index writes.
var nonWord = regexp.MustCompile(`[^a-zA-Z0-9]+`)

func searchNorm(s string) string {
	return nonWord.ReplaceAllString(s, " ")
}

// canonicalFields returns canonical URL + domain for URL-ish items,
// nil pair otherwise (stored as SQL NULL).
func canonicalFields(sourceType, rawRef string) (any, any) {
	if sourceType == "url" || sourceType == "bookmark" || sourceType == "x" {
		c := canonicalizeURL(rawRef)
		return c, extractDomain(c)
	}
	return nil, nil
}

// tenantID derives ownership server-side (§55). Phase-0: single local user.
// NEVER trust client-supplied user_id, no such field is read.
func tenantID(_ *http.Request) string { return "local" }

func secureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}

// cors allows only our own web origins (§55: restrictive allowlist, never *).
// Configurable via WEB_ORIGINS (comma-separated) since Next auto-bumps
// ports when one is taken (:3001 -> :3002).
func cors(next http.Handler) http.Handler {
	allowed := map[string]bool{}
	for _, o := range strings.Split(os.Getenv("WEB_ORIGINS"), ",") {
		if o = strings.TrimSpace(o); o != "" {
			allowed[o] = true
		}
	}
	if len(allowed) == 0 {
		for _, o := range []string{
			"http://localhost:3000", "http://127.0.0.1:3000",
			"http://localhost:3001", "http://127.0.0.1:3001",
			"http://localhost:3002", "http://127.0.0.1:3002",
		} {
			allowed[o] = true
		}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if allowed[r.Header.Get("Origin")] {
			w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		}
		if r.Method == "OPTIONS" {
			w.WriteHeader(204)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	ctx := context.Background()
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://helmme:helmme@localhost:5432/helmme?sslmode=disable"
	}
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("ok"))
	})

	uploadDir := os.Getenv("UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "./data/uploads"
	}
	registerUploadRoutes(mux, pool, uploadDir)
	registerImportRoutes(mux, pool)
	registerRelatedRoutes(mux, pool)
	registerSynthRoutes(mux, pool)
	registerDeleteRoutes(mux, pool, uploadDir)
	registerGraphRoutes(mux, pool)
	registerFileRoutes(mux, pool, uploadDir)
	registerItemRoutes(mux, pool)
	registerCorrectionRoutes(mux, pool)
	registerMaintenanceRoutes(mux, pool)
	registerQueryRoutes(mux, pool)
	registerMergeRoutes(mux, pool)
	registerResurfaceRoutes(mux, pool)
	registerReviewRoutes(mux, pool)
	registerSourceRoutes(mux, pool)
	registerXOAuthRoutes(mux, pool)
	registerDecisionRoutes(mux, pool)
	registerProjectRoutes(mux, pool)
	registerSettingsRoutes(mux, pool)
	registerExportRoutes(mux, pool)
	registerTopicRoutes(mux, pool)
	registerMetricsRoutes(mux, pool)
	registerEdgeRoutes(mux, pool)

	// POST /v1/items, Tier-0 capture + Tier-1 deterministic (canonicalize, hash, FTS)
	mux.HandleFunc("POST /v1/items", func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		var in ItemIn
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, "bad json", 400)
			return
		}
		if msg := validateItem(&in); msg != "" {
			http.Error(w, msg, 400)
			return
		}
		if in.Original == nil {
			in.Original = map[string]any{"raw_ref": in.RawRef}
		}
		// Retry-safe: a phone that saved offline and retries later carries the
		// same client_id. If we already stored it, hand back the first save
		// instead of making a twin. Plain words: same ticket, same seat.
		if in.ClientID != "" {
			var dupe string
			if err := pool.QueryRow(ctx,
				`SELECT id FROM items WHERE user_id=$1 AND client_id=$2`,
				tenantID(r), in.ClientID).Scan(&dupe); err == nil {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]any{"id": dupe, "duplicate": true})
				return
			}
		}
		h := sha256.Sum256([]byte(in.SourceType + "|" + in.RawRef))
		hash := hex.EncodeToString(h[:])
		// Canonical identity only means something for URLs. Notes and
		// other non-URL captures keep NULL so junk never lands in domain.
		canonical, domain := canonicalFields(in.SourceType, in.RawRef)
		origJSON, _ := json.Marshal(in.Original)

		var id string
		// captured_at keeps the real moment ("this morning on the train"), not
		// the moment the phone finally found signal. Empty means right now.
		capturedAt := "now()"
		capturedArg := []any{}
		if in.CapturedAt != "" {
			capturedAt = "$10"
			capturedArg = append(capturedArg, in.CapturedAt)
		}
		args := append([]any{
			tenantID(r), in.SourceType, in.RawRef, hash, origJSON, canonical, in.Title, domain, in.ClientID,
		}, capturedArg...)
		err := pool.QueryRow(ctx, `
		  INSERT INTO items (user_id, source_type, raw_ref, content_hash, original, canonical_url, title, domain, client_id, captured_at)
		  VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,`+capturedAt+`) RETURNING id`,
			args...).Scan(&id)
		if err != nil {
			log.Printf("capture failed item=opaque err=%T", err) // §64: opaque ids, never raw content
			http.Error(w, "internal error", 500)
			return
		}
		text := in.Title + "\n" + in.RawRef
		// Index the canonical URL, not the raw one: tracking soup must
		// never reach excerpts. The full address stays in raw_ref.
		if c, ok := canonical.(string); ok && c != "" {
			text = in.Title + "\n" + c
		}
		text = withQuotedWords(text, in.Original)
		_, err = pool.Exec(ctx, `
		  INSERT INTO item_contents (item_id, extracted_text, search_tsv)
		  VALUES ($1,$2,to_tsvector('english',$2 || ' ' || $3))`, id, text, searchNorm(text))
		if err != nil {
			log.Printf("enrich failed err=%T", err)
			http.Error(w, "internal error", 500)
			return
		}
		_, _ = pool.Exec(ctx, `INSERT INTO provenance_events (item_id, kind) VALUES ($1,'captured')`, id)
		if err := enrichItem(ctx, pool, tenantID(r), id); err != nil {
			log.Printf("enrich failed err=%T", err)
		}
		// Titles fill in after the answer: the save never waits for a fetch.
		go fillTitleIfMissing(context.Background(), pool, id)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"id": id})
	})

	// POST /v1/search, FTS first (§89, §119.9). Embeddings layer added later.
	mux.HandleFunc("POST /v1/search", func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		var q struct {
			Query string `json:"query"`
			Limit int    `json:"limit"`
		}
		if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
			http.Error(w, "bad json", 400)
			return
		}
		q.Query = strings.TrimSpace(q.Query)
		if q.Query == "" || len(q.Query) > 500 {
			http.Error(w, "invalid query", 400)
			return
		}
		if q.Limit <= 0 || q.Limit > 50 {
			q.Limit = 20
		}
		hits, relaxed, scope, err := runSearch(ctx, pool, tenantID(r), q.Query, q.Limit,
			"MaxWords=24, MinWords=12, ShortWord=3")
		if err != nil {
			log.Printf("search failed err=%T", err)
			http.Error(w, "internal error", 500)
			return
		}
		// The trace never breaks the answer: if logging fails, search still stands.
		_, _ = pool.Exec(ctx, `
		  INSERT INTO search_events (user_id, query, hits, relaxed)
		  VALUES ($1,$2,$3,$4)`, tenantID(r), q.Query, len(hits), relaxed)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"hits": hits, "relaxed": relaxed, "scope": scope})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Println("api on :" + port)
	log.Fatal(http.ListenAndServe(":"+port, cors(secureHeaders(mux))))
}

func canonicalizeURL(u string) string {
	u = strings.TrimSpace(u)
	lower := strings.ToLower(u)
	// Search pages carry identity in params (?q=...). Strip only
	// tracking params there; strip everything elsewhere.
	if strings.Contains(lower, "/search?") || strings.Contains(lower, "/search/?") {
		return stripTracking(lower)
	}
	if i := strings.Index(u, "?"); i >= 0 {
		// strip tracking params (§8 canonical identity, minimal)
		u = u[:i]
	}
	return strings.TrimSuffix(strings.ToLower(u), "/")
}

// stripTracking removes marketing params, keeping the rest.
func stripTracking(lower string) string {
	parts := strings.SplitN(lower, "?", 2)
	if len(parts) < 2 {
		return strings.TrimSuffix(lower, "/")
	}
	kept := []string{}
	for _, kv := range strings.Split(parts[1], "&") {
		k := kv
		if i := strings.Index(kv, "="); i >= 0 {
			k = kv[:i]
		}
		if strings.HasPrefix(k, "utm_") || k == "fbclid" || k == "gclid" || k == "mc_cid" {
			continue
		}
		kept = append(kept, kv)
	}
	out := parts[0]
	if len(kept) > 0 {
		out += "?" + strings.Join(kept, "&")
	}
	return strings.TrimSuffix(out, "/")
}

func extractDomain(u string) string {
	u = strings.TrimPrefix(strings.TrimPrefix(u, "https://"), "http://")
	if i := strings.Index(u, "/"); i >= 0 {
		u = u[:i]
	}
	return u
}
