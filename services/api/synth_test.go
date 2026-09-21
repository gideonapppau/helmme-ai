// Synthesis tests: citation gate, prompt assembly, LLM client paths.
// Pure except the client, which runs against a local stub server.
package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestExtractCitations(t *testing.T) {
	got := extractCitations("First [1][3], again [1], bad [0], far [12].")
	want := []int{1, 3, 12}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
	if out := extractCitations("no refs here"); len(out) != 0 {
		t.Errorf("false positives: %v", out)
	}
}

func TestValidateCitations(t *testing.T) {
	if ok, _ := validateCitations("claims [1][2]", 2); !ok {
		t.Error("valid citations rejected")
	}
	ok, bad := validateCitations("claims [1][9]", 2)
	if ok || len(bad) != 1 || bad[0] != 9 {
		t.Errorf("invented citation missed: ok=%v bad=%v", ok, bad)
	}
}

func TestBuildSynthesisPrompt(t *testing.T) {
	ev := []synthEvidence{
		{ID: "a", Title: "T", Source: "note", Excerpt: "<b>Hi</b> there"},
	}
	p := buildSynthesisPrompt("what?", ev)
	for _, want := range []string{"[1]", "what?", "Hi there", "Never invent"} {
		if !strings.Contains(p, want) {
			t.Errorf("prompt missing %q", want)
		}
	}
	if strings.Contains(p, "<b>") {
		t.Error("prompt leaks headline markup to the model")
	}
}

func TestCallChatCompletions(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer k" {
			t.Error("missing bearer key")
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["model"] != "m" {
			t.Errorf("model not passed: %v", body["model"])
		}
		w.Write([]byte(`{"choices":[{"message":{"content":"Summary [1]."}}]}`))
	}))
	defer srv.Close()
	got, err := callChatCompletions(srv.URL, "k", "m", "sys", "user")
	if err != nil || got != "Summary [1]." {
		t.Errorf("got %q err %v", got, err)
	}
}

func TestCallChatCompletionsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(429)
		w.Write([]byte(`{"error":{"message":"slow down"}}`))
	}))
	defer srv.Close()
	if _, err := callChatCompletions(srv.URL, "k", "m", "sys", "user"); err == nil {
		t.Error("upstream error swallowed")
	}
}

func TestNeedsMoreEvidence(t *testing.T) {
	// Plain words: zero is "nothing found" (handled elsewhere), one or two
	// is "not enough to conclude", three is a pile worth reasoning over.
	if needsMoreEvidence(0) {
		t.Error("zero evidence misread as thin")
	}
	for _, n := range []int{1, 2} {
		if !needsMoreEvidence(n) {
			t.Errorf("%d sources treated as enough", n)
		}
	}
	for _, n := range []int{3, 12} {
		if needsMoreEvidence(n) {
			t.Errorf("%d sources treated as thin", n)
		}
	}
}
