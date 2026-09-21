// Edge-helper tests. Plain words: links file once, addresses are found.
// Pure checks only, no database needed.
package main

import (
	"testing"
)

func TestOrderPair(t *testing.T) {
	a, b := orderPair("zzz", "aaa")
	if a != "aaa" || b != "zzz" {
		t.Errorf("not ordered: %q %q", a, b)
	}
	a, b = orderPair("aaa", "zzz")
	if a != "aaa" || b != "zzz" {
		t.Errorf("swapped ordered pair: %q %q", a, b)
	}
}

func TestExtractURLs(t *testing.T) {
	got := extractURLs("see https://example.com/a and https://example.com/a plus http://x.test/b.")
	if len(got) != 2 || got[0] != "https://example.com/a" || got[1] != "http://x.test/b" {
		t.Errorf("bad addresses: %q", got)
	}
	if len(extractURLs("no links here")) != 0 {
		t.Error("phantom links")
	}
}
