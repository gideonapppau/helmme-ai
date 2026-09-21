// Search-understanding tests: filler stripping + relaxed OR fallback.
// Pure functions, the "anything about marketmate" regression lives here.
package main

import (
	"testing"
)

func TestOrQuery(t *testing.T) {
	if got := orQuery("marketmate"); got != "marketmate:*" {
		t.Errorf("single word: %q", got)
	}
	// Every word rides along, even filler, Postgres drops stopwords,
	// the ranker buries the rest. No lists.
	if got := orQuery("anything about marketmate"); got != "anything:* | about:* | marketmate:*" {
		t.Errorf("words dropped: %q", got)
	}
	if got := orQuery("a i"); got != "" {
		t.Errorf("single chars should yield nothing: %q", got)
	}
	if got := orQuery("postgres redis"); got != "postgres:* | redis:*" {
		t.Errorf("multi word: %q", got)
	}
	// Short words match exactly: "la" must not prefix-match "layers",
	// while "go" still finds Go.
	if got := orQuery("la liga"); got != "la | liga:*" {
		t.Errorf("short word handling: %q", got)
	}
	if got := orQuery("go"); got != "go" {
		t.Errorf("short exact word: %q", got)
	}
	if got := orQuery(""); got != "" {
		t.Errorf("empty should yield nothing: %q", got)
	}
}

func TestParseScope(t *testing.T) {
	text, sc := parseScope("postgres in:notes")
	if text != "postgres" || len(sc.Types) != 2 {
		t.Errorf("type scope: %q %+v", text, sc)
	}
	text, sc = parseScope("rentals topic:Dev")
	if text != "rentals" || sc.Topic != "Dev" {
		t.Errorf("topic scope: %q %+v", text, sc)
	}
	text, sc = parseScope(`ideas view:"YC research" extra`)
	if text != "ideas extra" || sc.View != "YC research" {
		t.Errorf("quoted view scope: %q %+v", text, sc)
	}
	text, sc = parseScope("plain query")
	if text != "plain query" || sc.active() {
		t.Errorf("plain query touched: %q %+v", text, sc)
	}
	text, sc = parseScope("weird in:nowhere")
	if text != "weird in:nowhere" || sc.active() {
		t.Errorf("unknown scope should stay text: %q %+v", text, sc)
	}
}
