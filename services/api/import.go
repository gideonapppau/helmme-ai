// Bookmark import: Netscape HTML exports (Chrome/Safari/Firefox) -> Items.
// Folders become metadata, never homework. ADD_DATE becomes captured_at so
// a 2019 bookmark stays from 2019. Duplicates are counted honestly, and one
// bad entry never fails the run (§97).
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/net/html"
)

type bookmark struct {
	url     string
	title   string
	folders []string
	addedAt time.Time
	hasDate bool
}

// parseBookmarks walks a bookmark tree. Anchors inherit the folder path of
// the DL nesting that contains them.
func parseBookmarks(data []byte) []bookmark {
	doc, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return nil
	}
	var out []bookmark
	emit := func(a *html.Node, folders []string) {
		href := attrOf(a, "href")
		title := strings.TrimSpace(textOf(a))
		b := bookmark{url: href, title: title, folders: folders}
		if d := attrOf(a, "add_date"); d != "" {
			if unix, err := strconv.ParseInt(d, 10, 64); err == nil && unix > 0 {
				b.addedAt = time.Unix(unix, 0).UTC()
				b.hasDate = true
			}
		}
		out = append(out, b)
	}
	// walkDL handles both real-world shapes: the sub-DL as a sibling of
	// its H3's DT (Chrome/Firefox exports) and nested inside it, plus
	// parser-hoisted bare H3/A nodes.
	var walkDL func(dl *html.Node, folders []string)
	walkDL = func(dl *html.Node, folders []string) {
		var pending string
		for c := dl.FirstChild; c != nil; c = c.NextSibling {
			if c.Type != html.ElementNode {
				continue
			}
			withPending := func() []string {
				if pending == "" {
					return folders
				}
				return append(append([]string{}, folders...), pending)
			}
			switch c.Data {
			case "dt":
				// A DT can hold the folder name (H3) AND the folder body
				// (DL) at once, check both, not either.
				name := ""
				if h := findChild(c, "h3"); h != nil {
					name = strings.TrimSpace(textOf(h))
				}
				if sub := findChild(c, "dl"); sub != nil {
					if name != "" {
						walkDL(sub, append(append([]string{}, folders...), name))
					} else {
						walkDL(sub, withPending())
					}
					pending = ""
					continue
				}
				if name != "" {
					pending = name
					continue
				}
				if a := findChild(c, "a"); a != nil {
					emit(a, folders)
				}
			case "dl":
				walkDL(c, withPending())
				pending = ""
			case "h3":
				pending = strings.TrimSpace(textOf(c))
			case "a":
				emit(c, folders)
			}
		}
	}
	var findDL func(n *html.Node) *html.Node
	findDL = func(n *html.Node) *html.Node {
		if n.Type == html.ElementNode && n.Data == "dl" {
			return n
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if dl := findDL(c); dl != nil {
				return dl
			}
		}
		return nil
	}
	if dl := findDL(doc); dl != nil {
		walkDL(dl, nil)
	}
	return out
}

func findChild(n *html.Node, tag string) *html.Node {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == tag {
			return c
		}
	}
	return nil
}

func attrOf(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, key) {
			return strings.TrimSpace(a.Val)
		}
	}
	return ""
}

func textOf(n *html.Node) string {
	var sb strings.Builder
	var walk func(*html.Node)
	walk = func(x *html.Node) {
		if x.Type == html.TextNode {
			sb.WriteString(x.Data)
		}
		for c := x.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return sb.String()
}

// cleanBookmarkURL keeps http(s) URLs only. Anything else is a skip, not an error.
func cleanBookmarkURL(raw string) (string, bool) {
	u := strings.TrimSpace(raw)
	if u == "" {
		return "", false
	}
	lower := strings.ToLower(u)
	if !strings.HasPrefix(lower, "http://") && !strings.HasPrefix(lower, "https://") {
		return "", false
	}
	return u, true
}

func registerImportRoutes(mux *http.ServeMux, pool *pgxpool.Pool) {
	mux.HandleFunc("POST /v1/imports", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes+(1<<20))
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			http.Error(w, "file too big or bad upload", 400)
			return
		}
		f, hdr, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "missing file field", 400)
			return
		}
		defer f.Close()
		data, err := io.ReadAll(f)
		if err != nil || len(data) == 0 || len(data) > maxUploadBytes {
			http.Error(w, "empty or oversize file", 400)
			return
		}

		marks := parseBookmarks(data)
		user := tenantID(r)
		filename := hdr.Filename
		if filename == "" {
			filename = "bookmarks.html"
		}

		var importID string
		if err := pool.QueryRow(ctx, `
		  INSERT INTO imports (user_id, source, filename) VALUES ($1,'browser-bookmarks',$2) RETURNING id`,
			user, filename).Scan(&importID); err != nil {
			log.Printf("import record failed err=%T", err)
			http.Error(w, "internal error", 500)
			return
		}

		total, imported, dups, skipped := 0, 0, 0, 0
		seen := map[string]bool{}
		for _, b := range marks {
			total++
			raw, ok := cleanBookmarkURL(b.url)
			if !ok {
				skipped++
				_, _ = pool.Exec(ctx,
					`INSERT INTO import_items (import_id, canonical_url, status, note) VALUES ($1,$2,'skipped','bad url')`,
					importID, b.url)
				continue
			}
			canonical := canonicalizeURL(raw)
			title := b.title
			if title == "" {
				title = extractDomain(canonical)
			}
			if seen[canonical] {
				dups++
				_, _ = pool.Exec(ctx,
					`INSERT INTO import_items (import_id, canonical_url, status, note) VALUES ($1,$2,'duplicate','same file')`,
					importID, canonical)
				continue
			}
			seen[canonical] = true

			var exists bool
			if err := pool.QueryRow(ctx,
				`SELECT EXISTS(SELECT 1 FROM items WHERE user_id=$1 AND canonical_url=$2)`,
				user, canonical).Scan(&exists); err != nil || exists {
				dups++
				_, _ = pool.Exec(ctx,
					`INSERT INTO import_items (import_id, canonical_url, status, note) VALUES ($1,$2,'duplicate','already saved')`,
					importID, canonical)
				continue
			}

			h := sha256.Sum256([]byte("bookmark|" + canonical))
			hash := hex.EncodeToString(h[:])
			orig := map[string]any{"folders": b.folders, "source": "bookmark-import"}
			origJSON, _ := json.Marshal(orig)
			captured := time.Now().UTC()
			if b.hasDate {
				captured = b.addedAt
			}
			var id string
			if err := pool.QueryRow(ctx, `
			  INSERT INTO items (user_id, source_type, raw_ref, content_hash, original, canonical_url, title, domain, captured_at)
			  VALUES ($1,'bookmark',$2,$3,$4,$5,$6,$7,$8) RETURNING id`,
				user, raw, hash, origJSON, canonical, title, extractDomain(canonical), captured).Scan(&id); err != nil {
				skipped++
				_, _ = pool.Exec(ctx,
					`INSERT INTO import_items (import_id, canonical_url, status, note) VALUES ($1,$2,'skipped','store failed')`,
					importID, canonical)
				continue
			}
			searchText := title + "\n" + canonical + "\n" + strings.Join(b.folders, " ")
			if _, err := pool.Exec(ctx, `
			  INSERT INTO item_contents (item_id, extracted_text, search_tsv)
			  VALUES ($1,$2,to_tsvector('english',$2 || ' ' || $3))`, id, searchText, searchNorm(searchText)); err != nil {
				skipped++
				continue
			}
			_, _ = pool.Exec(ctx, `INSERT INTO provenance_events (item_id, kind) VALUES ($1,'captured')`, id)
			_, _ = pool.Exec(ctx,
				`INSERT INTO import_items (import_id, item_id, canonical_url, status) VALUES ($1,$2,$3,'imported')`,
				importID, id, canonical)
			if err := enrichItem(ctx, pool, user, id); err != nil {
				log.Printf("import enrich failed err=%T", err)
			}
			imported++
		}

		_, _ = pool.Exec(ctx,
			`UPDATE imports SET total=$1, imported=$2, duplicates=$3, skipped=$4 WHERE id=$5`,
			total, imported, dups, skipped, importID)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id": importID, "total": total, "imported": imported,
			"duplicates": dups, "skipped": skipped,
		})
	})
}
