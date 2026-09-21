// Settings tests. Plain words: quiet means one, balanced three, proactive
// five, anything else is refused. Pure checks only, no database needed.
package main

import (
	"testing"
)

func TestResurfaceLimit(t *testing.T) {
	if got := resurfaceLimit("quiet"); got != 1 {
		t.Errorf("quiet=%d", got)
	}
	if got := resurfaceLimit("balanced"); got != 3 {
		t.Errorf("balanced=%d", got)
	}
	if got := resurfaceLimit("proactive"); got != 5 {
		t.Errorf("proactive=%d", got)
	}
	if got := resurfaceLimit("loud"); got != 1 {
		t.Errorf("unknown mood=%d, want quiet default 1", got)
	}
}

func TestResurfaceModes(t *testing.T) {
	for _, m := range []string{"quiet", "balanced", "proactive"} {
		if !resurfaceModes[m] {
			t.Errorf("known mood refused: %q", m)
		}
	}
	if resurfaceModes["party"] {
		t.Error("unknown mood allowed")
	}
}
