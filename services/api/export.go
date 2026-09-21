// Export (§51): the user can always leave with everything.
// Plain words: your archive is yours. One download carries your saves,
// your views, your projects, never passwords, tokens, or keys.
// GET /v1/export?format=json|markdown&source=text (source optional)
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// exportItem is one save in the takeout. Secrets are never fields here:
// tokens live in `sources`, which this endpoint never reads.
type exportItem struct {
	ID         string   `json:"id"`
	Type       string   `json:"type"`
	Title      string   `json:"title"`
	Content    string   `json:"content"`
	URL        string   `json:"url"`
	CapturedAt string   `json:"captured_at"`
	Topics     []string `json:"topics"`
}

// exportMarkdown renders the takeout as one readable document.
func exportMarkdown(items []exportItem, queries []string, projects []string) string {
	var sb strings.Builder
	sb.WriteString("# Memory export\n\n")
	sb.WriteString("Your saves, newest first. Raw words first, machine guesses never.\n")
	for _, it := range items {
		sb.WriteString("\n## " + it.Title + "\n\n")
		sb.WriteString("*" + it.Type + " · " + it.CapturedAt + "*\n\n")
		if it.URL != "" {
			sb.WriteString(it.URL + "\n\n")
		}
		if strings.TrimSpace(it.Content) != "" {
			sb.WriteString(it.Content + "\n\n")
		}
		if len(it.Topics) > 0 {
			sb.WriteString("Topics: " + strings.Join(it.Topics, ", ") + "\n\n")
		}
	}
	if len(queries) > 0 {
		sb.WriteString("\n# Saved searches\n\n")
		for _, q := range queries {
			sb.WriteString("- " + q + "\n")
		}
	}
	if len(projects) > 0 {
		sb.WriteString("\n# Projects\n\n")
		for _, p := range projects {
			sb.WriteString("- " + p + "\n")
		}
	}
	return sb.String()
}

func registerExportRoutes(mux *http.ServeMux, pool *pgxpool.Pool) {
	mux.HandleFunc("GET /v1/export", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user := tenantID(r)
		format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
		if format == "" {
			format = "json"
		}
		if format != "json" && format != "markdown" {
			http.Error(w, "format must be json or markdown", 400)
			return
		}
		source := strings.TrimSpace(r.URL.Query().Get("source"))
		if source != "" && !allowedSourceTypes[source] {
			http.Error(w, "unknown source", 400)
			return
		}

		rows, err := pool.Query(ctx, `
		  SELECT i.id, i.source_type, COALESCE(i.title,''), COALESCE(c.extracted_text,''),
		         CASE WHEN i.source_type IN ('url','bookmark','x')
		              THEN COALESCE(i.canonical_url, i.raw_ref, '')
		              ELSE '' END, i.captured_at::text,
		         COALESCE((SELECT json_agg(t.name ORDER BY it.confidence DESC)
		           FROM item_topics it JOIN topics t ON t.id = it.topic_id
		           WHERE it.item_id = i.id), '[]')
		  FROM items i LEFT JOIN item_contents c ON c.item_id = i.id
		  WHERE i.user_id=$1 AND i.status='active' AND i.sync_state<>'deleted'
		    AND ($2='' OR i.source_type=$2)
		  ORDER BY i.captured_at DESC LIMIT 5000`, user, source)
		if err != nil {
			log.Printf("export failed err=%T", err)
			http.Error(w, "internal error", 500)
			return
		}
		defer rows.Close()
		items := []exportItem{}
		for rows.Next() {
			var it exportItem
			var topicsRaw string
			if err := rows.Scan(&it.ID, &it.Type, &it.Title, &it.Content, &it.URL, &it.CapturedAt, &topicsRaw); err != nil {
				continue
			}
			it.Topics = topNames(topicsRaw, 10)
			if it.Topics == nil {
				it.Topics = []string{}
			}
			items = append(items, it)
		}
		var queries []string
		qrows, err := pool.Query(ctx, `SELECT name FROM queries WHERE user_id=$1 ORDER BY created_at DESC`, user)
		if err == nil {
			defer qrows.Close()
			for qrows.Next() {
				var n string
				if err := qrows.Scan(&n); err == nil {
					queries = append(queries, n)
				}
			}
		}
		var projects []string
		prows, err := pool.Query(ctx, `SELECT name FROM projects WHERE user_id=$1 AND status='confirmed' ORDER BY created_at DESC`, user)
		if err == nil {
			defer prows.Close()
			for prows.Next() {
				var n string
				if err := prows.Scan(&n); err == nil {
					projects = append(projects, n)
				}
			}
		}

		if format == "markdown" {
			w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
			w.Header().Set("Content-Disposition", `attachment; filename="memory-export.md"`)
			w.Write([]byte(exportMarkdown(items, queries, projects)))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"items": items, "saved_searches": queries, "projects": projects,
		})
	})
}
