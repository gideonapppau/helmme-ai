// Topic-control rules. Plain words: names are short, never blank.
// Pure checks only, no database needed.
package main

import (
	"strings"
	"testing"
)

func TestCleanTopicName(t *testing.T) {
	if got := cleanTopicName("  Postgres  "); got != "Postgres" {
		t.Errorf("not trimmed: %q", got)
	}
	if got := cleanTopicName(""); got != "" {
		t.Errorf("blank accepted: %q", got)
	}
	if got := cleanTopicName(strings.Repeat("a", 200)); len(got) != 120 {
		t.Errorf("not capped: len=%d", len(got))
	}
}
