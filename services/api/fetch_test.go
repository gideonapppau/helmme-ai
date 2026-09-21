// Fetcher tests: SSRF refusals, title shapes, hostile servers.
// The guard is the product here — every blocked shape is asserted.
package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestPublicTarget(t *testing.T) {
	good := []string{
		"https://example.com/a",
		"http://example.com:8080/x",
	}
	for _, u := range good {
		if !publicTarget(u) {
			t.Errorf("refused public url: %q", u)
		}
	}
	bad := map[string]string{
		"http://127.0.0.1/x":       "loopback",
		"http://10.0.0.5/x":        "private 10",
		"http://192.168.1.1/x":     "private 192",
		"http://172.16.0.9/x":      "private 172",
		"http://169.254.169.254/x": "cloud metadata",
		"http://[::1]/x":           "ipv6 loopback",
		"http://localhost:8081/x":  "localhost name",
		"http://printer.local/x":   "mdns name",
		"http://db.internal/x":     "internal name",
		"ftp://example.com/x":      "scheme",
		"file:///etc/passwd":       "file scheme",
		"":                         "empty",
		"http://":                  "no host",
		"http://0.0.0.0/x":         "unspecified",
		"http://224.0.0.1/x":       "multicast",
	}
	for u, why := range bad {
		if publicTarget(u) {
			t.Errorf("allowed %s: %q", why, u)
		}
	}
}

func TestFetchTitle(t *testing.T) {
	testAllowPrivate = true
	defer func() { testAllowPrivate = false }()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok":
			w.Write([]byte("<html><head><title>  Real   Title </title></head><body>x</body></html>"))
		case "/notitle":
			w.Write([]byte("<html><body>no title here</body></html>"))
		case "/redir":
			http.Redirect(w, r, "/ok", http.StatusFound)
		case "/redir-loop":
			http.Redirect(w, r, "/redir-loop", http.StatusFound)
		case "/big":
			w.Write([]byte("<html><head><title>T</title></head><body>" + strings.Repeat("x", 2<<20) + "</body></html>"))
		case "/slow":
			time.Sleep(10 * time.Second)
		}
	}))
	defer srv.Close()

	if got, err := fetchTitle(srv.URL + "/ok"); err != nil || got != "Real Title" {
		t.Errorf("got %q err %v", got, err)
	}
	if _, err := fetchTitle(srv.URL + "/notitle"); err == nil {
		t.Error("missing title accepted")
	}
	if got, err := fetchTitle(srv.URL + "/redir"); err != nil || got != "Real Title" {
		t.Errorf("single redirect failed: %q %v", got, err)
	}
	if _, err := fetchTitle(srv.URL + "/redir-loop"); err == nil {
		t.Error("redirect loop accepted")
	}
	if _, err := fetchTitle(srv.URL + "/slow"); err == nil {
		t.Error("slow server accepted past timeout")
	}
}
