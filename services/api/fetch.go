// Page titles for pasted links (§114 ingestion).
// Fetches AFTER capture answers, never during it. SSRF-guarded:
// http(s) only, no private targets, short timeout, small body, title
// parsed as text. Anything suspicious is skipped, never retried blindly.
package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/net/html"
)

const (
	fetchTimeout = 5 * time.Second
	maxFetchBody = 1 << 20 // 1MB
	maxRedirect  = 2
	maxBodyText  = 20000 // past this, a page is quoted not stored
)

// testAllowPrivate opens the guard for httptest servers (always loopback).
// Never set outside tests.
var testAllowPrivate = false

// publicTarget reports whether a URL is safe to fetch. Literal private,
// loopback, link-local (covers cloud metadata), and multicast IPs are
// refused, as are internal-looking hostnames. DNS rebinding is a known
// remaining limit, documented not ignored.
func publicTarget(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	host := u.Hostname()
	if host == "" {
		return false
	}
	lower := strings.ToLower(host)
	for _, blocked := range []string{"localhost", ".local", ".internal", ".lan", ".home", ".corp"} {
		if lower == strings.TrimPrefix(blocked, ".") || strings.HasSuffix(lower, blocked) {
			return false
		}
	}
	if ip := net.ParseIP(host); ip != nil && !testAllowPrivate {
		if ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() ||
			ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
			return false
		}
	}
	return true
}

func fetchTitle(raw string) (string, error) {
	if !publicTarget(raw) {
		return "", fmt.Errorf("refused target")
	}
	client := &http.Client{
		Timeout: fetchTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirect {
				return fmt.Errorf("too many redirects")
			}
			if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
				return fmt.Errorf("bad redirect scheme")
			}
			return nil
		},
	}
	req, err := http.NewRequest("GET", raw, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "helmme/0.1 (+capture)")
	resp, err := client.Do(req.WithContext(context.Background()))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return "", fmt.Errorf("status %d", resp.StatusCode)
	}
	doc, err := html.Parse(io.LimitReader(resp.Body, maxFetchBody))
	if err != nil {
		return "", err
	}
	title := findTitle(doc)
	title = strings.Join(strings.Fields(title), " ")
	if title == "" {
		return "", fmt.Errorf("no title")
	}
	if len(title) > 500 {
		title = title[:500]
	}
	return title, nil
}

func findTitle(n *html.Node) string {
	if n.Type == html.ElementNode && n.Data == "title" {
		return textOf(n)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if t := findTitle(c); t != "" {
			return t
		}
	}
	return ""
}

// fetchBody reads a saved article's real words (not just its title) so the
// page becomes findable by what it actually says. Plain words: we open the
// page after saving, copy the readable text, and file it with the save.
// Anything that is not a normal web page (a PDF, a download) is skipped.
func fetchBody(raw string) (title, body, author, pubdate string, err error) {
	if !publicTarget(raw) {
		return "", "", "", "", fmt.Errorf("refused target")
	}
	client := &http.Client{
		Timeout: fetchTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirect {
				return fmt.Errorf("too many redirects")
			}
			if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
				return fmt.Errorf("bad redirect scheme")
			}
			return nil
		},
	}
	req, err := http.NewRequest("GET", raw, nil)
	if err != nil {
		return "", "", "", "", err
	}
	req.Header.Set("User-Agent", "helmme/0.1 (+capture)")
	resp, err := client.Do(req.WithContext(context.Background()))
	if err != nil {
		return "", "", "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return "", "", "", "", fmt.Errorf("status %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "" &&
		!strings.Contains(strings.ToLower(ct), "html") && !strings.Contains(strings.ToLower(ct), "text") {
		return "", "", "", "", fmt.Errorf("not a readable page")
	}
	doc, err := html.Parse(io.LimitReader(resp.Body, maxFetchBody))
	if err != nil {
		return "", "", "", "", err
	}
	title = strings.Join(strings.Fields(findTitle(doc)), " ")
	body = readableText(doc)
	author = metaContent(doc, "author")
	pubdate = metaContent(doc, "article:published_time")
	if pubdate == "" {
		pubdate = metaContent(doc, "date")
	}
	return title, body, author, pubdate, nil
}

// skipTags holds page furniture: code and buttons are never article words.
var skipTags = map[string]bool{
	"script": true, "style": true, "noscript": true, "nav": true,
	"header": true, "footer": true, "form": true, "button": true,
	"select": true, "input": true, "aside": true,
}

// textTags holds the tags we actually read aloud.
var textTags = map[string]bool{
	"p": true, "h1": true, "h2": true, "h3": true, "h4": true,
	"li": true, "blockquote": true,
}

// readableText walks the page and keeps only human sentences.
// It prefers the <article> or <main> part when the page names one,
// because sidebars and menus are noise, not memory.
func readableText(doc *html.Node) string {
	root := findTag(doc, "article")
	if root == nil {
		root = findTag(doc, "main")
	}
	if root == nil {
		root = doc
	}
	var parts []string
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			if skipTags[n.Data] {
				return // whole branch is furniture, skip it
			}
			if textTags[n.Data] {
				t := strings.Join(strings.Fields(textOf(n)), " ")
				if len(t) >= 20 { // crumbs ("click here") are not sentences
					parts = append(parts, t)
				}
				return // children already counted inside this sentence
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(root)
	out := strings.Join(parts, "\n")
	if len(out) > maxBodyText {
		out = out[:maxBodyText]
	}
	return out
}

// findTag returns the first element with this name, or nil.
func findTag(n *html.Node, tag string) *html.Node {
	if n.Type == html.ElementNode && n.Data == tag {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := findTag(c, tag); found != nil {
			return found
		}
	}
	return nil
}

// metaContent reads a page's own labels, like author or publish date.
// It checks <meta name="...">, <meta property="...">, and <time datetime>.
func metaContent(doc *html.Node, key string) string {
	var out string
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if out != "" {
			return
		}
		if n.Type == html.ElementNode && n.Data == "meta" {
			var name, prop, content string
			for _, a := range n.Attr {
				switch strings.ToLower(a.Key) {
				case "name":
					name = strings.ToLower(a.Val)
				case "property":
					prop = strings.ToLower(a.Val)
				case "content":
					content = strings.TrimSpace(a.Val)
				}
			}
			if (name == key || prop == key) && content != "" {
				out = content
				return
			}
		}
		if n.Type == html.ElementNode && n.Data == "time" && key == "article:published_time" {
			for _, a := range n.Attr {
				if strings.ToLower(a.Key) == "datetime" && strings.TrimSpace(a.Val) != "" {
					out = strings.TrimSpace(a.Val)
					return
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return out
}

// fillTitleIfMissing finishes a link save after the answer went out:
// the page title, then the page's real words, so the save becomes findable
// by what it says. Silent on any failure: the link itself always stays.
func fillTitleIfMissing(ctx context.Context, pool *pgxpool.Pool, id string) {
	var sourceType, rawRef, title, extracted, origRaw string
	if err := pool.QueryRow(ctx, `
	  SELECT source_type, raw_ref, COALESCE(title,''), COALESCE(c.extracted_text,''),
	         COALESCE(i.original::text,'{}')
	  FROM items i LEFT JOIN item_contents c ON c.item_id = i.id
	  WHERE i.id=$1`,
		id).Scan(&sourceType, &rawRef, &title, &extracted, &origRaw); err != nil {
		return
	}
	if sourceType != "url" && sourceType != "bookmark" && sourceType != "x" {
		return
	}
	// Step 1: title, if the save arrived without a real one.
	if title == "" || strings.HasPrefix(strings.ToLower(title), "http") {
		if t, err := fetchTitle(rawRef); err == nil && t != "" {
			title = t
			_, _ = pool.Exec(ctx, `UPDATE items SET title=$1 WHERE id=$2`, t, id)
		}
	}
	// Step 2: body words, if we only filed the title and address so far.
	// Plain words: a save that only says "example.com/prices" is not findable
	// by "predictable pricing", the article's own sentences fix that.
	if len(extracted) < len(title)+len(rawRef)+100 {
		t, body, author, pubdate, err := fetchBody(rawRef)
		if err != nil || body == "" {
			// Title-only index still stands below; nothing lost.
		} else {
			if t != "" && (title == "" || strings.HasPrefix(strings.ToLower(title), "http")) {
				title = t
				_, _ = pool.Exec(ctx, `UPDATE items SET title=$1 WHERE id=$2`, t, id)
			}
			// Guesses go beside the capture, never inside it (§42): the
			// user's original bytes stay byte-identical in `original`,
			// while author/date land in item_metadata with their source.
			if author != "" {
				_, _ = pool.Exec(ctx, `
				  INSERT INTO item_metadata (item_id, key, value, source, confidence)
				  VALUES ($1,'fetched_author',$2,'fetch',0.7)
				  ON CONFLICT (item_id, key) DO UPDATE
				    SET value=$2, updated_at=now()`, id, author)
			}
			if pubdate != "" {
				_, _ = pool.Exec(ctx, `
				  INSERT INTO item_metadata (item_id, key, value, source, confidence)
				  VALUES ($1,'fetched_published',$2,'fetch',0.7)
				  ON CONFLICT (item_id, key) DO UPDATE
				    SET value=$2, updated_at=now()`, id, pubdate)
			}
			text := title + "\n" + rawRef + "\n" + body
			_, _ = pool.Exec(ctx, `
			  UPDATE item_contents SET extracted_text=$1,
			    search_tsv=to_tsvector('english',$1 || ' ' || $2)
			  WHERE item_id=$3`, text, searchNorm(text), id)
			return
		}
	}
	// Fallback index (title + address) for pages that refused the full read.
	text := title
	if c, _ := canonicalFields(sourceType, rawRef); c != nil {
		if cs, ok := c.(string); ok {
			text += "\n" + cs
		}
	}
	_, _ = pool.Exec(ctx, `
	  UPDATE item_contents SET extracted_text=$1,
	    search_tsv=to_tsvector('english',$1 || ' ' || $2)
	  WHERE item_id=$3`, text, searchNorm(text), id)
}
