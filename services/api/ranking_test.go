// Ranking tests. Plain words: your own notes must win when the question
// is about your thinking; plain searches must stay untouched.
// Pure checks only, no database needed.
package main

import (
	"testing"
)

func TestOwnThinkingQuery(t *testing.T) {
	mine := []string{
		"What did I decide about Redis?",
		"what have I learned about postgres",
		"my notes on pricing",
		"postgres", // control: filled below
	}
	for _, q := range mine[:3] {
		if !ownThinkingQuery(q) {
			t.Errorf("missed own-thinking query: %q", q)
		}
	}
	if ownThinkingQuery(mine[3]) {
		t.Errorf("plain search flagged as own-thinking: %q", mine[3])
	}
}

func TestNudgeOwnThinking(t *testing.T) {
	hits := []Hit{
		{ID: "article", Source: "url", Rank: 1.0},
		{ID: "mine", Source: "text", Rank: 0.7},
	}
	out := nudgeOwnThinking("what did I decide about caching?", hits)
	if out[0].ID != "mine" {
		t.Errorf("own note did not rise first: %+v", out)
	}
	plain := []Hit{
		{ID: "article", Source: "url", Rank: 1.0},
		{ID: "mine", Source: "text", Rank: 0.7},
	}
	out = nudgeOwnThinking("caching", plain)
	if out[0].ID != "article" {
		t.Errorf("plain search order disturbed: %+v", out)
	}
}

func TestSourceScope(t *testing.T) {
	// Plain words: "source:x ..." keeps only X posts, and the word itself
	// never reaches the search.
	clean, sc := parseScope("source:x postgres")
	if clean != "postgres" {
		t.Errorf("scope word leaked into query: %q", clean)
	}
	found := false
	for _, ty := range sc.Types {
		if ty == "x" {
			found = true
		}
	}
	if !found {
		t.Errorf("x type missing from scope: %+v", sc)
	}
	_, sc2 := parseScope("source:url helmme")
	if len(sc2.Types) != 1 || sc2.Types[0] != "url" {
		t.Errorf("literal type name not scoped: %+v", sc2)
	}
}

func TestNudgeRecentContext(t *testing.T) {
	// Plain words: a familiar place rises a little; strangers never fall
	// for any other reason.
	hits := []Hit{
		{ID: "far", Source: "url", Domain: "elsewhere.test", Rank: 1.0},
		{ID: "near", Source: "url", Domain: "home.test", Rank: 0.9},
	}
	out := nudgeRecentContext(map[string]bool{"home.test": true}, hits)
	if out[0].ID != "near" {
		t.Errorf("familiar place did not rise: %+v", out)
	}
	same := []Hit{{ID: "a", Source: "url", Domain: "x.test", Rank: 1.0}}
	if got := nudgeRecentContext(map[string]bool{}, same); got[0].Rank != 1.0 {
		t.Error("empty recent map changed ranks")
	}
}
