// Related-merge tests: dedupe, reason combining, priority, cap. Pure.
package main

import (
	"testing"
)

func mkRel(id string, reasons ...string) relatedHit {
	return relatedHit{ID: id, Title: "t-" + id, Reasons: reasons}
}

func TestMergeRelated(t *testing.T) {
	a := []relatedHit{mkRel("1", reasonSavedTogether), mkRel("2", reasonSameSite)}
	b := []relatedHit{mkRel("2", reasonSimilarWords), mkRel("3", reasonEarlier)}
	out := mergeRelated([][]relatedHit{a, b}, 6)
	if len(out) != 3 {
		t.Fatalf("len=%d want 3", len(out))
	}
	if out[0].ID != "1" || out[1].ID != "2" || out[2].ID != "3" {
		t.Errorf("order lost: %+v", out)
	}
	if len(out[1].Reasons) != 2 {
		t.Errorf("reasons not combined: %+v", out[1].Reasons)
	}
}

func TestMergeRelatedCap(t *testing.T) {
	var g []relatedHit
	for _, id := range []string{"1", "2", "3", "4"} {
		g = append(g, mkRel(id, reasonLater))
	}
	if out := mergeRelated([][]relatedHit{g}, 2); len(out) != 2 {
		t.Errorf("cap ignored, len=%d", len(out))
	}
	if out := mergeRelated(nil, 6); out != nil {
		t.Errorf("nil groups should stay nil, got %+v", out)
	}
}
