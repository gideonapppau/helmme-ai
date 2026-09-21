// Claim checks (§28): every prose claim gets a Jev verdict in ONE call.
// Labels: backed | guess | contradicted | unknown. Thresholds match the
// context-engine policy (0.8 / 0.5 / contradicted 0.6). Skipped silently
// when there is no key, no prose, or nothing to check.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
	"unicode"
)

const (
	maxClaims     = 12
	maxClaimLen   = 400
	maxEvidenceCh = 8000
	jevModel      = "jev-latest"
	jevEndpoint   = "https://api.typesafe.ai/v1/systemone"
)

type claimVerdict struct {
	Text         string   `json:"text"`
	Status       string   `json:"status"`
	Supported    float64  `json:"supported"`
	Contradicted float64  `json:"contradicted"`
	Sources      []string `json:"sources"`
}

// mapClaimSources resolves each claim's [n] markers to evidence ids.
// Out-of-range markers are dropped; marker-less claims keep an empty
// list. The author cited them, Jev judged them; both stay visible.
func mapClaimSources(claims []claimVerdict, ids []string) []claimVerdict {
	for i := range claims {
		srcs := []string{}
		for _, n := range extractCitations(claims[i].Text) {
			if n >= 1 && n <= len(ids) {
				srcs = append(srcs, ids[n-1])
			}
		}
		claims[i].Sources = srcs
	}
	return claims
}

// splitClaims cuts prose into checkable sentences. Headings, bare
// citations ("[1][2]"), and fragments are dropped, they carry no claim.
func splitClaims(prose string) []string {
	var out []string
	for _, line := range strings.Split(prose, "\n") {
		start := 0
		for i, r := range line {
			if r == '.' || r == '!' || r == '?' {
				out = append(out, line[start:i+1])
				start = i + 1
			}
		}
		if tail := strings.TrimSpace(line[start:]); tail != "" {
			out = append(out, tail)
		}
	}
	var kept []string
	for _, c := range out {
		c = strings.TrimSpace(c)
		letters := 0
		for _, r := range c {
			if unicode.IsLetter(r) {
				letters++
			}
		}
		// Short but real ("Go is fast.") stays; bare refs ("[1][2]") go.
		if letters < 8 {
			continue
		}
		if len(c) > maxClaimLen {
			c = c[:maxClaimLen]
		}
		kept = append(kept, c)
		if len(kept) >= maxClaims {
			break
		}
	}
	return kept
}

// verdictFor turns odds into a label. Same bars as the TS policy.
// "disputed" means real support and real pushback coexist, surfacing
// the conflict beats picking a side.
func verdictFor(supported, contradicted float64) string {
	if contradicted > 0.6 {
		return "contradicted"
	}
	if supported >= 0.5 && contradicted > 0.3 {
		return "disputed"
	}
	if supported >= 0.8 {
		return "backed"
	}
	if supported >= 0.5 {
		return "guess"
	}
	return "unknown"
}

type jevClient struct {
	baseURL string
	apiKey  string
	model   string
}

func jevFromEnv() (jevClient, bool) {
	key := os.Getenv("TYPESAFE_AI_API_KEY")
	if key == "" {
		key = os.Getenv("TYPESAFE_API_KEY")
	}
	if key == "" {
		return jevClient{}, false
	}
	return jevClient{baseURL: jevEndpoint, apiKey: key, model: jevModel}, true
}

// check routes one boolean question per claim through Jev in a single call:
// is this claim supported by the evidence? contradicted by it?
func (j jevClient) check(claims []string, evidence string) []claimVerdict {
	out := make([]claimVerdict, len(claims))
	questions := map[string]any{}
	for i, c := range claims {
		questions[fmt.Sprintf("supported_%d", i)] = map[string]any{
			"type":         "noul",
			"instructions": "Is this claim supported by the evidence? Claim: " + c,
		}
		questions[fmt.Sprintf("contradicted_%d", i)] = map[string]any{
			"type":         "noul",
			"instructions": "Is this claim contradicted by the evidence? Claim: " + c,
		}
	}
	body, _ := json.Marshal(map[string]any{
		"model":     j.model,
		"state":     "Evidence:\n" + evidence + "\n\nEvaluate each claim against the evidence above.",
		"questions": questions,
	})
	req, err := http.NewRequest("POST", j.baseURL, bytes.NewReader(body))
	if err != nil {
		return verdictsUnknown(claims)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+j.apiKey)
	resp, err := (&http.Client{Timeout: 60 * time.Second}).Do(req)
	if err != nil {
		return verdictsUnknown(claims)
	}
	defer resp.Body.Close()
	var decoded struct {
		Answers map[string]struct {
			Noul float64 `json:"noul"`
		} `json:"answers"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil || resp.StatusCode/100 != 2 {
		return verdictsUnknown(claims)
	}
	for i, c := range claims {
		s := decoded.Answers[fmt.Sprintf("supported_%d", i)].Noul
		k := decoded.Answers[fmt.Sprintf("contradicted_%d", i)].Noul
		out[i] = claimVerdict{Text: c, Status: verdictFor(s, k), Supported: s, Contradicted: k}
	}
	return out
}

func verdictsUnknown(claims []string) []claimVerdict {
	out := make([]claimVerdict, len(claims))
	for i, c := range claims {
		out[i] = claimVerdict{Text: c, Status: "unknown"}
	}
	return out
}

// evidencePack joins excerpts up to a char budget for the Jev state.
func evidencePack(ev []synthEvidence) string {
	var sb strings.Builder
	for i, e := range ev {
		part := "[" + itoa(i+1) + "] " + stripTags(e.Excerpt) + "\n"
		if sb.Len()+len(part) > maxEvidenceCh {
			break
		}
		sb.WriteString(part)
	}
	return sb.String()
}
