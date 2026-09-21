// Synthesis v1 (§26-29, §73-75): retrieval + prose + citation gate + run log.
// POST /v1/synthesize {query} -> {id, summary|null, evidence, citations_ok, note}
//
// The LLM writes from numbered evidence only. A code gate checks every
// [n] citation points at real evidence. Without a key nothing fails:
// evidence is returned with a plain note instead of prose.
package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const synthPromptVersion = "synth-v1"
const synthTopK = 12

type synthEvidence struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Source     string `json:"source_type"`
	Domain     string `json:"domain"`
	Excerpt    string `json:"excerpt"`
	CapturedAt string `json:"captured_at"`
}

var citeRe = regexp.MustCompile(`\[(\d+)\]`)
var tagRe = regexp.MustCompile(`<[^>]+>`)

// extractCitations returns the distinct [n] references in prose, in order.
func extractCitations(prose string) []int {
	seen := map[int]bool{}
	var out []int
	for _, m := range citeRe.FindAllStringSubmatch(prose, -1) {
		var n int
		for _, ch := range m[1] {
			n = n*10 + int(ch-'0')
		}
		if n < 1 {
			continue
		}
		if !seen[n] {
			seen[n] = true
			out = append(out, n)
		}
	}
	return out
}

// validateCitations reports whether every [n] points at real evidence.
func validateCitations(prose string, evidenceCount int) (bool, []int) {
	var bad []int
	for _, n := range extractCitations(prose) {
		if n > evidenceCount {
			bad = append(bad, n)
		}
	}
	return len(bad) == 0, bad
}

func stripTags(s string) string {
	return tagRe.ReplaceAllString(s, "")
}

func buildSynthesisPrompt(query string, ev []synthEvidence) string {
	var sb strings.Builder
	sb.WriteString("You summarize a personal archive. Below are numbered sources from the user's own saved material.\n")
	sb.WriteString("Write a concise summary answering: \"" + query + "\"\n")
	sb.WriteString("Rules:\n")
	sb.WriteString("- Every substantive claim must cite its sources like [1][3].\n")
	sb.WriteString("- Only cite the numbered sources below. Never invent sources.\n")
	sb.WriteString("- If the sources do not support a conclusion, say what is missing instead of guessing.\n")
	sb.WriteString("- Mark guesses explicitly with the word \"Inference:\".\n")
	sb.WriteString("SOURCES:\n")
	for i, e := range ev {
		sb.WriteString("\n[")
		sb.WriteString(itoa(i + 1))
		sb.WriteString("] " + e.Title + " (" + e.Source)
		if e.Domain != "" {
			sb.WriteString(", " + e.Domain)
		}
		sb.WriteString(", " + e.CapturedAt + ")\n" + stripTags(e.Excerpt) + "\n")
	}
	return sb.String()
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [8]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

// callChatCompletions posts to an OpenAI-compatible endpoint (Groq default).
// baseURL is a parameter so tests can point it at a local stub.
func callChatCompletions(baseURL, apiKey, model, system, user string) (string, error) {
	body, _ := json.Marshal(map[string]any{
		"model":       model,
		"temperature": 0.2,
		"max_tokens":  800,
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": user},
		},
	})
	req, err := http.NewRequest("POST", strings.TrimSuffix(baseURL, "/")+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if resp.StatusCode/100 != 2 {
		msg := resp.Status
		if out.Error != nil && out.Error.Message != "" {
			msg = out.Error.Message
		}
		return "", errBadUpstream(msg)
	}
	if len(out.Choices) == 0 {
		return "", errBadUpstream("empty completion")
	}
	return out.Choices[0].Message.Content, nil
}

type upstreamError struct{ msg string }

func (e *upstreamError) Error() string { return "upstream: " + e.msg }

func errBadUpstream(msg string) error { return &upstreamError{msg} }

func registerSynthRoutes(mux *http.ServeMux, pool *pgxpool.Pool) {
	mux.HandleFunc("POST /v1/synthesize", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		var in struct {
			Query string `json:"query"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, "bad json", 400)
			return
		}
		trimmed := strings.TrimSpace(in.Query)
		if trimmed == "" || len(trimmed) > 500 {
			http.Error(w, "invalid query", 400)
			return
		}
		user := tenantID(r)

		hits, _, _, err := runSearch(ctx, pool, user, trimmed, synthTopK,
			"MaxWords=40, MinWords=15, ShortWord=3")
		if err != nil {
			log.Printf("synth retrieval failed err=%T", err)
			http.Error(w, "internal error", 500)
			return
		}
		var ev []synthEvidence
		var ids []string
		for _, h := range hits {
			ev = append(ev, synthEvidence{
				ID: h.ID, Title: h.Title, Source: h.Source,
				Domain: h.Domain, Excerpt: h.Excerpt,
				CapturedAt: h.CapturedAt.UTC().Format("2006-01-02"),
			})
			ids = append(ids, h.ID)
		}
		if ev == nil {
			ev = []synthEvidence{}
		}

		apiKey := os.Getenv("GROQ_API_KEY")
		model := os.Getenv("SYNTH_MODEL")
		if model == "" {
			model = "openai/gpt-oss-20b"
		}

		var summary *string
		citationsOK := true
		note := ""
		if len(ev) == 0 {
			note = "Nothing in your archive matches that yet."
		} else if apiKey == "" {
			note = "Summaries need a GROQ_API_KEY on the server. Evidence below."
		} else {
			prose, err := callChatCompletions(
				"https://api.groq.com/openai/v1", apiKey, model,
				"You summarize personal archives. Cite every claim. Never invent sources.",
				buildSynthesisPrompt(in.Query, ev),
			)
			if err != nil {
				log.Printf("synth llm failed err=%T", err)
				note = "The summary failed. Evidence below still works."
			} else {
				ok, bad := validateCitations(prose, len(ev))
				citationsOK = ok
				if !ok {
					note = "Some citations did not check out. Read the evidence directly."
					_ = bad
				} else if len(extractCitations(prose)) == 0 {
					// A summary with no citations proves nothing, even if
					// every marker it does have is valid.
					citationsOK = false
					note = "No sources cited. Read the evidence directly."
				}
				summary = &prose
			}
		}

		idsJSON, _ := json.Marshal(ids)
		result := ""
		if summary != nil {
			result = *summary
		}
		claims := []claimVerdict{}
		if summary != nil {
			if jev, ok := jevFromEnv(); ok {
				claims = mapClaimSources(jev.check(splitClaims(*summary), evidencePack(ev)), ids)
			}
		}
		claimsJSON, _ := json.Marshal(claims)
		var runID string
		_ = pool.QueryRow(ctx, `
		  INSERT INTO synthesis_runs (user_id, query, source_ids, model, prompt_version, result, citations_ok, claims)
		  VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id`,
			user, in.Query, idsJSON, model, synthPromptVersion, result, citationsOK, claimsJSON).Scan(&runID)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id": runID, "summary": summary, "evidence": ev,
			"citations_ok": citationsOK, "note": note, "claims": claims,
		})
	})
}
