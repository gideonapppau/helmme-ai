// Export tests. Plain words: the takeout must read well and leak nothing.
// Pure checks only, no database needed.
package main

import (
	"strings"
	"testing"
)

func TestExportMarkdown(t *testing.T) {
	items := []exportItem{
		{ID: "1", Type: "text", Title: "My note", Content: "real words", URL: "", CapturedAt: "2026-09-21", Topics: []string{"PostgreSQL"}},
		{ID: "2", Type: "url", Title: "Article", Content: "body", URL: "https://example.com/a", CapturedAt: "2026-09-20"},
	}
	out := exportMarkdown(items, []string{"Corsa Cloud"}, []string{"Launch"})
	for _, want := range []string{"# Memory export", "## My note", "real words", "Topics: PostgreSQL", "https://example.com/a", "- Corsa Cloud", "- Launch"} {
		if !strings.Contains(out, want) {
			t.Errorf("takeout missing %q", want)
		}
	}
	for _, leak := range []string{"access_token", "refresh_token", "password", "secret"} {
		if strings.Contains(strings.ToLower(out), leak) {
			t.Errorf("takeout leaks %q", leak)
		}
	}
}

func TestExportMarkdownEmpty(t *testing.T) {
	out := exportMarkdown(nil, nil, nil)
	if !strings.Contains(out, "# Memory export") {
		t.Errorf("empty takeout broken: %q", out)
	}
}
