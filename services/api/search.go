// Search understanding (§20-23): filler stripping + relaxed fallback.
// "anything about marketmate" must find MarketMate. Strict AND first
// (precise), OR-of-prefixes fallback second (never empty when something
// vaguely matches). Shared by /v1/search and /v1/synthesize.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Hit is one search result. Excerpt carries <b> match markers.
type Hit struct {
	ID         string    `json:"id"`
	Title      string    `json:"title"`
	Source     string    `json:"source_type"`
	Domain     string    `json:"domain"`
	CapturedAt time.Time `json:"captured_at"`
	RawRef     string    `json:"raw_ref"`
	Excerpt    string    `json:"excerpt"`
	Topics     []string  `json:"topics"`
	Rank       float64   `json:"rank"`
}

// searchScope narrows a search. Parsed from the query itself:
// `in:notes`, `topic:Dev`, `view:"My view"`. Unknown tokens are left
// alone, a typo never errors.
type searchScope struct {
	Types []string `json:"types"`
	Topic string   `json:"topic"`
	View  string   `json:"view"`
}

// scopeTypes maps UI type names to source_type values.
var scopeTypes = map[string][]string{
	"notes":    {"text", "note"},
	"articles": {"url", "pdf"},
	"code":     {"file"},
	"media":    {"image"},
	"saved":    {"bookmark"},
	"x":        {"x"},
}

// sourceScopeTypes maps a `source:` word to item kinds. Plain words: typing
// "source:x ..." shows only X posts, the same as tapping the X chip.
var sourceScopeTypes = map[string][]string{
	"x": {"x"}, "twitter": {"x"},
	"notes": {"text", "note"}, "note": {"text", "note"},
	"articles": {"url", "pdf"}, "article": {"url", "pdf"},
	"code": {"file"}, "media": {"image"},
	"saved": {"bookmark"}, "bookmarks": {"bookmark"},
	// Plain type names work too: "source:url", "source:pdf", ...
	"url": {"url"}, "pdf": {"pdf"}, "file": {"file"},
	"image": {"image"}, "bookmark": {"bookmark"}, "text": {"text"},
}

// parseScope pulls scope tokens out, returning the clean text first.
func parseScope(raw string) (string, searchScope) {
	var sc searchScope
	var words []string
	toks := strings.Fields(raw)
	for i := 0; i < len(toks); i++ {
		t := toks[i]
		lower := strings.ToLower(t)
		if rest, ok := strings.CutPrefix(lower, "in:"); ok {
			if types, ok := scopeTypes[rest]; ok {
				sc.Types = append(sc.Types, types...)
				continue
			}
		}
		if rest, ok := strings.CutPrefix(lower, "source:"); ok {
			if types, ok := sourceScopeTypes[rest]; ok {
				sc.Types = append(sc.Types, types...)
				continue
			}
		}
		if rest, ok := strings.CutPrefix(lower, "topic:"); ok && rest != "" {
			sc.Topic = toks[i][len("topic:"):]
			continue
		}
		if rest, ok := strings.CutPrefix(lower, "view:"); ok {
			name := toks[i][len("view:"):]
			// view:"multi word name" collects quoted words.
			if strings.HasPrefix(name, "\"") {
				name = name[1:]
				for i+1 < len(toks) && !strings.HasSuffix(toks[i], "\"") {
					i++
					name += " " + toks[i]
				}
				name = strings.TrimSuffix(name, "\"")
			}
			if rest != "" || name != "" {
				sc.View = name
				continue
			}
		}
		words = append(words, t)
	}
	return strings.Join(words, " "), sc
}

// orQuery builds the fallback: exact match for short words, prefix for
// long ones. Short prefixes ("la:*") match everything starting with two
// letters, which is noise, not recall. Exact short words ("go") still
// work. Postgres drops its own stopwords; single characters are skipped.
func orQuery(normed string) string {
	var parts []string
	for _, w := range strings.Fields(normed) {
		if len(w) < 2 {
			continue
		}
		if len(w) < 4 {
			parts = append(parts, w)
		} else {
			parts = append(parts, w+":*")
		}
	}
	return strings.Join(parts, " | ")
}

// topNames takes the first N topic names from the aggregated JSON.
func topNames(raw string, n int) []string {
	var all []string
	if err := json.Unmarshal([]byte(raw), &all); err != nil {
		return []string{}
	}
	if len(all) > n {
		all = all[:n]
	}
	if all == nil {
		return []string{}
	}
	return all
}

// scopeActive reports whether any filter was parsed.
func (s searchScope) active() bool {
	return len(s.Types) > 0 || s.Topic != "" || s.View != ""
}

// scopeIDs resolves scope filters to item ids in one query. Empty scope
// means everything (nil, no filter). Dynamic args stay positional and
// parameterized, values never touch SQL text.
func scopeIDs(ctx context.Context, pool *pgxpool.Pool, user string, scope searchScope, viewText string) []string {
	if !scope.active() {
		return nil
	}
	conds := []string{"user_id=$1"}
	args := []any{user}
	if len(scope.Types) > 0 {
		args = append(args, scope.Types)
		conds = append(conds, fmt.Sprintf("source_type = ANY($%d)", len(args)))
	}
	if scope.Topic != "" {
		args = append(args, scope.Topic)
		conds = append(conds, fmt.Sprintf(`EXISTS (SELECT 1 FROM item_topics it
		  JOIN topics t ON t.id = it.topic_id
		  WHERE it.item_id=items.id AND NOT t.hidden AND LOWER(t.name)=LOWER($%d))`, len(args)))
	}
	if viewText != "" {
		args = append(args, searchNorm(viewText))
		conds = append(conds, fmt.Sprintf(`EXISTS (SELECT 1 FROM item_contents c2
		  WHERE c2.item_id=items.id AND c2.search_tsv @@ plainto_tsquery('english',$%d))`, len(args)))
	}
	rows, err := pool.Query(ctx, `SELECT id FROM items WHERE `+strings.Join(conds, " AND "), args...)
	if err != nil {
		log.Printf("scope resolve failed err=%T", err)
		return []string{}
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	return ids
}

func queryFTS(ctx context.Context, pool *pgxpool.Pool, user, qtext string, limit int, headlineOpts, qexpr string, ids []string) ([]Hit, error) {
	rows, err := pool.Query(ctx, `
	  SELECT i.id, COALESCE(i.title,''), i.source_type, COALESCE(i.domain,''), i.captured_at,
	         i.raw_ref,
	         ts_headline('english', COALESCE(c.extracted_text,''), `+qexpr+`,
	           '`+headlineOpts+`') AS headline,
	         COALESCE((SELECT json_agg(t.name ORDER BY it.confidence DESC)
	           FROM item_topics it JOIN topics t ON t.id = it.topic_id
	           WHERE it.item_id = i.id AND NOT t.hidden), '[]') AS topics,
	         ts_rank(c.search_tsv, `+qexpr+`) AS rank
	  FROM items i JOIN item_contents c ON c.item_id=i.id
	  WHERE i.user_id=$3 AND i.status='active' AND ($4::uuid[] IS NULL OR i.id = ANY($4::uuid[])) AND c.search_tsv @@ `+qexpr+`
	  ORDER BY rank DESC LIMIT $2`, qtext, limit, user, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanHits(rows)
}

// scanHits reads a result shaped like queryFTS/vectorSearch output.
func scanHits(rows interface {
	Next() bool
	Scan(...interface{}) error
	Close()
}) ([]Hit, error) {
	defer rows.Close()
	var out []Hit
	for rows.Next() {
		var h Hit
		var topicsRaw string
		if err := rows.Scan(&h.ID, &h.Title, &h.Source, &h.Domain, &h.CapturedAt, &h.RawRef, &h.Excerpt, &topicsRaw, &h.Rank); err != nil {
			continue
		}
		if tops := topNames(topicsRaw, 2); tops != nil {
			h.Topics = tops
		} else {
			h.Topics = []string{}
		}
		out = append(out, h)
	}
	if out == nil {
		out = []Hit{}
	}
	return out, nil
}

// vectorSearch finds by meaning, not words. Empty when no key exists.
func vectorSearch(ctx context.Context, pool *pgxpool.Pool, user string, vec []float32, limit int, headlineOpts, hlQuery string, ids []string) ([]Hit, error) {
	rows, err := pool.Query(ctx, `
	  SELECT i.id, COALESCE(i.title,''), i.source_type, COALESCE(i.domain,''), i.captured_at,
	         i.raw_ref,
	         ts_headline('english', COALESCE(c.extracted_text,''), plainto_tsquery('english',$2),
	           '`+headlineOpts+`') AS headline,
	         COALESCE((SELECT json_agg(t.name ORDER BY it.confidence DESC)
	           FROM item_topics it JOIN topics t ON t.id = it.topic_id
	           WHERE it.item_id = i.id AND NOT t.hidden), '[]') AS topics,
	         1 - (c.embedding <=> $1::vector) AS rank
	  FROM items i JOIN item_contents c ON c.item_id=i.id
	  WHERE i.user_id=$3 AND i.status='active' AND ($5::uuid[] IS NULL OR i.id = ANY($5::uuid[])) AND c.embedding IS NOT NULL
	  ORDER BY c.embedding <=> $1::vector LIMIT $4`,
		vectorLiteral(vec), hlQuery, user, limit, ids)
	if err != nil {
		return nil, err
	}
	return scanHits(rows)
}

// fuseRRF merges ranked lists by reciprocal rank. First list wins ties
// and lends its excerpt: word matches outrank meaning matches on ties.
func fuseRRF(limit int, lists ...[]Hit) []Hit {
	const k = 60.0
	scores := map[string]float64{}
	byID := map[string]Hit{}
	for _, list := range lists {
		for rank, h := range list {
			scores[h.ID] += 1.0 / (k + float64(rank+1))
			if _, ok := byID[h.ID]; !ok {
				byID[h.ID] = h
			}
		}
	}
	ids := make([]string, 0, len(scores))
	for id := range scores {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(a, b int) bool { return scores[ids[a]] > scores[ids[b]] })
	out := make([]Hit, 0, len(ids))
	for _, id := range ids {
		h := byID[id]
		h.Rank = scores[id]
		out = append(out, h)
		if len(out) >= limit {
			break
		}
	}
	if out == nil {
		out = []Hit{}
	}
	return out
}

// nudgeOwnThinking lifts the user's own words when the question is about
// their thinking ("what did I decide", "my notes on"). Plain words: when you
// ask what YOU thought, your notes answer first, not strangers' articles.
func nudgeOwnThinking(rawQuery string, hits []Hit) []Hit {
	if !ownThinkingQuery(rawQuery) || len(hits) == 0 {
		return hits
	}
	for i := range hits {
		if hits[i].Source == "text" || hits[i].Source == "note" {
			hits[i].Rank *= 1.5
		}
	}
	sort.Slice(hits, func(a, b int) bool { return hits[a].Rank > hits[b].Rank })
	return hits
}

// ownThinkingQuery spots questions about the user's own mind.
func ownThinkingQuery(q string) bool {
	lower := strings.ToLower(q)
	for _, w := range []string{
		"i decid", "i learn", "i thought", "i think", "i wrote", "i said",
		"my notes", "my thinking", "did i ", "have i ", "what do i ",
	} {
		if strings.Contains(lower, w) {
			return true
		}
	}
	return false
}

// nudgeRecentContext lifts saves from places you've been reading lately.
// Plain words: if you've spent the month on one site, its older saves rise
// a little. A small lift (1.2x) for recency of interest, never quality.
func nudgeRecentContext(recentDomains map[string]bool, hits []Hit) []Hit {
	if len(recentDomains) == 0 || len(hits) == 0 {
		return hits
	}
	for i := range hits {
		if hits[i].Domain != "" && recentDomains[hits[i].Domain] {
			hits[i].Rank *= 1.2
		}
	}
	sort.Slice(hits, func(a, b int) bool { return hits[a].Rank > hits[b].Rank })
	return hits
}

// recentDomains lists where the last 30 days of saves came from.
func recentDomains(ctx context.Context, pool *pgxpool.Pool, user string) map[string]bool {
	out := map[string]bool{}
	rows, err := pool.Query(ctx, `
	  SELECT DISTINCT domain FROM items
	  WHERE user_id=$1 AND status='active' AND sync_state<>'deleted'
	    AND captured_at > now() - interval '30 days' AND domain <> ''`, user)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var d string
		if err := rows.Scan(&d); err == nil && d != "" {
			out[d] = true
		}
	}
	return out
}

// runSearch blends word matches with meaning matches, then falls back
// to the relaxed OR query. Returns hits, approximate flag, and the scope
// that was applied (for chips). Meaning search is skipped silently
// without a key: keywords stand alone.
func runSearch(ctx context.Context, pool *pgxpool.Pool, user, rawQuery string, limit int, headlineOpts string) ([]Hit, bool, searchScope, error) {
	cleanText, scope := parseScope(rawQuery)
	query := searchNorm(strings.TrimSpace(cleanText))
	viewText := ""
	if scope.View != "" {
		_ = pool.QueryRow(ctx, `SELECT query_text FROM queries
		  WHERE user_id=$1 AND LOWER(name)=LOWER($2)
		  ORDER BY created_at DESC LIMIT 1`, user, scope.View).Scan(&viewText)
	}
	ids := scopeIDs(ctx, pool, user, scope, viewText)
	strict := "plainto_tsquery('english',$1)"
	hits, err := queryFTS(ctx, pool, user, query, limit, headlineOpts, strict, ids)
	if err != nil {
		return nil, false, scope, err
	}
	if vec, verr := embedQuery(query); verr == nil && vec != nil {
		if vhits, verr := vectorSearch(ctx, pool, user, vec, limit, headlineOpts, query, ids); verr == nil {
			fused := fuseRRF(limit, hits, vhits)
			if len(fused) > 0 {
				merged := nudgeOwnThinking(cleanText, fused)
				return nudgeRecentContext(recentDomains(ctx, pool, user), merged), len(hits) == 0, scope, nil
			}
		} else {
			log.Printf("vector search failed err=%T", verr)
		}
	}
	if len(hits) > 0 {
		merged := nudgeOwnThinking(cleanText, hits)
		return nudgeRecentContext(recentDomains(ctx, pool, user), merged), false, scope, nil
	}
	if ors := orQuery(query); ors != "" {
		// All-stopword queries error here; that means "nothing to match",
		// not a server failure. The fallback never 500s.
		if fb, ferr := queryFTS(ctx, pool, user, ors, limit, headlineOpts, "to_tsquery('english',$1)", ids); ferr == nil {
			return fb, true, scope, nil
		}
	}
	return []Hit{}, false, scope, nil
}
