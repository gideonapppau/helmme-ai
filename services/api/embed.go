// Embeddings (§72): meaning-vectors for semantic retrieval.
// OpenAI-compatible API (base URL + key + model from env). Missing key
// means vectors are skipped, never an error — keyword search stands alone.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

const defaultEmbedModel = "text-embedding-3-small"
const defaultEmbedBase = "https://api.openai.com/v1"
const maxEmbedChars = 6000

type embedClient struct {
	baseURL string
	apiKey  string
	model   string
}

func embedFromEnv() (embedClient, bool) {
	key := os.Getenv("EMBED_KEY")
	if key == "" {
		return embedClient{}, false
	}
	base := os.Getenv("EMBED_BASE_URL")
	if base == "" {
		base = defaultEmbedBase
	}
	model := os.Getenv("EMBED_MODEL")
	if model == "" {
		model = defaultEmbedModel
	}
	return embedClient{baseURL: base, apiKey: key, model: model}, true
}

// embedTexts calls the API once for a batch. Returns one vector per input,
// in order. Any failure is an error — the caller decides it is non-fatal.
func (e embedClient) embedTexts(inputs []string) ([][]float32, error) {
	body, _ := json.Marshal(map[string]any{"model": e.model, "input": inputs})
	req, err := http.NewRequest("POST", strings.TrimSuffix(e.baseURL, "/")+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.apiKey)
	resp, err := (&http.Client{Timeout: 60 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
			Index     int       `json:"index"`
		} `json:"data"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if resp.StatusCode/100 != 2 {
		msg := resp.Status
		if out.Error != nil && out.Error.Message != "" {
			msg = out.Error.Message
		}
		return nil, fmt.Errorf("embeddings: %s", msg)
	}
	vecs := make([][]float32, len(inputs))
	for _, d := range out.Data {
		if d.Index >= 0 && d.Index < len(vecs) {
			vecs[d.Index] = d.Embedding
		}
	}
	for i, v := range vecs {
		if v == nil {
			return nil, fmt.Errorf("embeddings: missing vector %d", i)
		}
	}
	return vecs, nil
}

// queryCache avoids re-embedding identical searches. Small, in-memory,
// prototype-grade — a real cache lands with Redis-backed query work.
var queryCache sync.Map // string -> []float32

func embedQuery(text string) ([]float32, error) {
	ec, ok := embedFromEnv()
	if !ok {
		return nil, nil
	}
	if v, hit := queryCache.Load(text); hit {
		return v.([]float32), nil
	}
	vecs, err := ec.embedTexts([]string{text})
	if err != nil {
		log.Printf("embed query failed err=%T", err)
		return nil, err
	}
	if queryCacheLen() < 200 {
		queryCache.Store(text, vecs[0])
	}
	return vecs[0], nil
}

func queryCacheLen() int {
	n := 0
	queryCache.Range(func(_, _ any) bool { n++; return n < 201 })
	return n
}

// vectorLiteral formats for a ::vector cast. No new dependency needed.
func vectorLiteral(vec []float32) string {
	var sb strings.Builder
	sb.WriteByte('[')
	for i, f := range vec {
		if i > 0 {
			sb.WriteByte(',')
		}
		fmt.Fprintf(&sb, "%g", f)
	}
	sb.WriteByte(']')
	return sb.String()
}
