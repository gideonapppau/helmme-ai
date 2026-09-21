// Graph extraction tests: folders to topics, domains to entities,
// title guesses. Pure, idempotency is verified live on the backfill.
package main

import (
	"testing"
)

func TestTopicsFromFolders(t *testing.T) {
	got := topicsFromFolders([]string{"Research", " Nested ", ""})
	if len(got) != 2 || got[0].name != "Research" || got[1].name != "Nested" {
		t.Errorf("got %+v", got)
	}
	if len(topicsFromFolders(nil)) != 0 {
		t.Error("nil folders should yield nothing")
	}
}

func TestEntitiesFromDomain(t *testing.T) {
	got := entitiesFromDomain("example.com")
	if len(got) != 1 || got[0].typ != "site" || got[0].confidence < 0.8 {
		t.Errorf("got %+v", got)
	}
	if len(entitiesFromDomain("  ")) != 0 {
		t.Error("blank domain should yield nothing")
	}
}

func TestTitleCandidates(t *testing.T) {
	got := titleCandidates("Postgres pooling notes for Clerk")
	names := map[string]bool{}
	for _, e := range got {
		names[e.name] = true
		if e.confidence >= 0.5 {
			t.Errorf("guess overconfident: %+v", e)
		}
	}
	if !names["Postgres"] || !names["Clerk"] {
		t.Errorf("missed capitals: %+v", got)
	}
	if names["for"] || names["notes"] {
		t.Errorf("lowercase leaked in: %+v", got)
	}
	dup := titleCandidates("Go Go Go")
	if len(dup) != 1 {
		t.Errorf("dupes not collapsed: %+v", dup)
	}
	if got := titleCandidates("Gideon_Appau_CV.pdf"); len(got) != 0 {
		t.Errorf("filename junk kept: %+v", got)
	}
}

func TestContentEntities(t *testing.T) {
	text := "PostgreSQL is great. I chose PostgreSQL for Clerk. " +
		"PostgreSQL handles it. Flutterwave once. a B c D e"
	got := contentEntities(text)
	names := map[string]float64{}
	for _, e := range got {
		names[e.name] = e.confidence
	}
	if _, ok := names["PostgreSQL"]; !ok {
		t.Errorf("frequent name missed: %+v", got)
	}
	if _, ok := names["Flutterwave"]; ok {
		t.Errorf("loner kept: %+v", got)
	}
	if c, ok := names["PostgreSQL"]; !ok || c < 0.4 {
		t.Errorf("frequency not rewarded: %+v", got)
	}
}

func TestContentEntitiesSkipsSentenceVerbs(t *testing.T) {
	text := "Built this. Built that. Built more. " +
		"Chose PostgreSQL here and PostgreSQL there and PostgreSQL again."
	got := contentEntities(text)
	names := map[string]bool{}
	for _, e := range got {
		names[e.name] = true
	}
	if names["Built"] {
		t.Errorf("sentence verb kept: %+v", got)
	}
	if !names["PostgreSQL"] {
		t.Errorf("mid-sentence name dropped: %+v", got)
	}
}

func TestTopNames(t *testing.T) {
	got := topNames(`["a","b","c"]`, 2)
	if len(got) != 2 || got[0] != "a" {
		t.Errorf("got %v", got)
	}
	if got := topNames(`not json`, 2); len(got) != 0 {
		t.Errorf("bad json should yield empty: %v", got)
	}
}
