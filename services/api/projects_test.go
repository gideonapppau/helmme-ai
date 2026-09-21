// Project-name rules. Plain words: names are short, never blank.
// Pure checks only, no database needed.
package main

import (
	"strings"
	"testing"
)

func TestCleanProjectName(t *testing.T) {
	if got := cleanProjectName("  Corsa Cloud  "); got != "Corsa Cloud" {
		t.Errorf("not trimmed: %q", got)
	}
	if got := cleanProjectName(""); got != "" {
		t.Errorf("blank accepted: %q", got)
	}
	if got := cleanProjectName(strings.Repeat("a", 200)); len(got) != 120 {
		t.Errorf("not capped: len=%d", len(got))
	}
}
