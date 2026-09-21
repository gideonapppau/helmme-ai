// Embedding tests: client shape, vector literal, rank fusion. No key,
// no network, the stub server stands in for the provider.
package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEmbedTexts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Model string   `json:"model"`
			Input []string `json:"input"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body.Model != "m" || len(body.Input) != 2 {
			t.Errorf("bad request: %+v", body)
		}
		if r.Header.Get("Authorization") != "Bearer k" {
			t.Error("missing bearer key")
		}
		w.Write([]byte(`{"data":[{"embedding":[0.1,0.2],"index":0},{"embedding":[0.3,0.4],"index":1}]}`))
	}))
	defer srv.Close()
	ec := embedClient{baseURL: srv.URL, apiKey: "k", model: "m"}
	vecs, err := ec.embedTexts([]string{"a", "b"})
	if err != nil || len(vecs) != 2 || vecs[1][1] != 0.4 {
		t.Errorf("got %v err %v", vecs, err)
	}
}

func TestEmbedTextsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		w.Write([]byte(`{"error":{"message":"bad key"}}`))
	}))
	defer srv.Close()
	ec := embedClient{baseURL: srv.URL, apiKey: "k", model: "m"}
	if _, err := ec.embedTexts([]string{"a"}); err == nil {
		t.Error("bad key swallowed")
	}
}

func TestEmbedQueryNoKey(t *testing.T) {
	t.Setenv("EMBED_KEY", "")
	if v, err := embedQuery("hello"); v != nil || err != nil {
		t.Errorf("no key must skip: %v %v", v, err)
	}
}

func TestVectorLiteral(t *testing.T) {
	if got := vectorLiteral([]float32{0.5, -1}); got != "[0.5,-1]" {
		t.Errorf("got %q", got)
	}
}

func TestFuseRRF(t *testing.T) {
	a := []Hit{{ID: "1"}, {ID: "2"}}
	b := []Hit{{ID: "2"}, {ID: "3"}}
	out := fuseRRF(10, a, b)
	if len(out) != 3 {
		t.Fatalf("len=%d", len(out))
	}
	// "2" appears in both lists: first overall.
	if out[0].ID != "2" || out[1].ID != "1" || out[2].ID != "3" {
		t.Errorf("order wrong: %+v", out)
	}
	// First list lends its row (excerpt, rank source).
	if len(fuseRRF(10, nil, b)) != 2 {
		t.Error("nil list should be skipped")
	}
	if out := fuseRRF(2, a, b); len(out) != 2 {
		t.Errorf("cap ignored: %d", len(out))
	}
}
