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

// fillTitleIfMissing fetches a page title for link items that arrived
// without one, then re-indexes so the words become searchable. Silent on
// any failure: the domain fallback stays.
func fillTitleIfMissing(ctx context.Context, pool *pgxpool.Pool, id string) {
	var sourceType, rawRef, title string
	if err := pool.QueryRow(ctx, `SELECT source_type, raw_ref, COALESCE(title,'') FROM items WHERE id=$1`,
		id).Scan(&sourceType, &rawRef, &title); err != nil {
		return
	}
	if sourceType != "url" && sourceType != "bookmark" {
		return
	}
	if title != "" && !strings.HasPrefix(strings.ToLower(title), "http") {
		return
	}
	t, err := fetchTitle(rawRef)
	if err != nil {
		return
	}
	if _, err := pool.Exec(ctx, `UPDATE items SET title=$1 WHERE id=$2`, t, id); err != nil {
		return
	}
	text := t
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
