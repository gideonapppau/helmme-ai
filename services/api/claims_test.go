// Claim-check tests: splitter, verdict bars, Jev client shape.
package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSplitClaims(t *testing.T) {
	got := splitClaims("Postgres is fast [1]. Inference: you like it.\n\nIs that all clear now?")
	if len(got) != 3 {
		t.Fatalf("got %v", got)
	}
	if got := splitClaims("[1][2]\n\n..."); len(got) != 0 {
		t.Errorf("fragments kept: %v", got)
	}
	many := ""
	for i := 0; i < 30; i++ {
		many += "This is a reasonably long claim about things. "
	}
	if got := splitClaims(many); len(got) != maxClaims {
		t.Errorf("cap ignored: %d", len(got))
	}
}

func TestVerdictFor(t *testing.T) {
	cases := map[[2]float64]string{
		{0.9, 0.1}:  "backed",
		{0.6, 0.05}: "guess",
		{0.9, 0.7}:  "contradicted",
		{0.2, 0.1}:  "unknown",
		{0.5, 0.0}:  "guess",
		{0.8, 0.0}:  "backed",
		{0.7, 0.4}:  "disputed",
		{0.6, 0.35}: "disputed",
		{0.9, 0.4}:  "disputed",
		{0.9, 0.3}:  "backed",
		{0.4, 0.5}:  "unknown",
	}
	for in, want := range cases {
		if got := verdictFor(in[0], in[1]); got != want {
			t.Errorf("verdict(%v)=%q want %q", in, got, want)
		}
	}
}

func TestJevCheck(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Model     string         `json:"model"`
			Questions map[string]any `json:"questions"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body.Model != "jev-latest" {
			t.Errorf("model=%q", body.Model)
		}
		if len(body.Questions) != 4 {
			t.Errorf("want 2 questions per claim, got %d", len(body.Questions))
		}
		w.Write([]byte(`{"model":"jev-1.13.0","answers":{
		  "supported_0":{"type":"noul","noul":0.9},"contradicted_0":{"type":"noul","noul":0.05},
		  "supported_1":{"type":"noul","noul":0.3},"contradicted_1":{"type":"noul","noul":0.1}}}`))
	}))
	defer srv.Close()
	j := jevClient{baseURL: srv.URL, apiKey: "k", model: "jev-latest"}
	got := j.check([]string{"Postgres is fast.", "Maybe Redis too."}, "[1] speed")
	if len(got) != 2 || got[0].Status != "backed" || got[1].Status != "unknown" {
		t.Errorf("verdicts wrong: %+v", got)
	}
}

func TestJevCheckDown(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()
	j := jevClient{baseURL: srv.URL, apiKey: "k", model: "jev-latest"}
	got := j.check([]string{"A solid claim here."}, "ev")
	if len(got) != 1 || got[0].Status != "unknown" {
		t.Errorf("outage must degrade to unknown: %+v", got)
	}
}

func TestMapClaimSources(t *testing.T) {
	ids := []string{"a", "b", "c"}
	in := []claimVerdict{
		{Text: "First thing [1][3].", Status: "backed"},
		{Text: "Wild guess here.", Status: "guess"},
		{Text: "Bad ref [9].", Status: "unknown"},
	}
	got := mapClaimSources(in, ids)
	if len(got[0].Sources) != 2 || got[0].Sources[0] != "a" || got[0].Sources[1] != "c" {
		t.Errorf("refs unresolved: %+v", got[0])
	}
	if len(got[1].Sources) != 0 {
		t.Errorf("marker-less should be empty: %+v", got[1])
	}
	if len(got[2].Sources) != 0 {
		t.Errorf("out-of-range should drop: %+v", got[2])
	}
}
