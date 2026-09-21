// Quoted-words tests. Plain words: highlights and jottings join the index.
// Pure checks only, no database needed.
package main

import (
	"strings"
	"testing"
)

func TestWithQuotedWords(t *testing.T) {
	got := withQuotedWords("Title\nhttps://example.com", map[string]any{
		"selected_text": "the highlighted sentence",
		"note_text":     "my jot",
	})
	for _, want := range []string{"Title", "the highlighted sentence", "my jot"} {
		if !strings.Contains(got, want) {
			t.Errorf("index text missing %q: %q", want, got)
		}
	}
	plain := withQuotedWords("Title\nhttps://example.com", map[string]any{})
	if plain != "Title\nhttps://example.com" {
		t.Errorf("plain save altered: %q", plain)
	}
	big := withQuotedWords("t", map[string]any{"note_text": strings.Repeat("a", 20000)})
	if len(big) > 16000 {
		t.Errorf("index text uncapped: %d", len(big))
	}
}
