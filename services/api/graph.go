// Context graph v1 (§§62-64): deterministic extraction only.
// Folders become topics, domains become site entities, capitalized title
// words become low-confidence guesses. An LLM extractor plugs in beside
// this one when a key exists — same tables, same API.
package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"unicode"

	"github.com/jackc/pgx/v5/pgxpool"
)

type topicCand struct {
	name       string
	confidence float64
}

type entityCand struct {
	typ        string
	name       string
	confidence float64
}

// topicsFromFolders treats every folder level as a topic. A folder that
// holds saves is a real grouping with real confidence behind it.
func topicsFromFolders(folders []string) []topicCand {
	var out []topicCand
	for _, f := range folders {
		if name := strings.TrimSpace(f); name != "" {
			out = append(out, topicCand{name: name, confidence: 0.6})
		}
	}
	return out
}

// entitiesFromDomain records where things came from as a site entity.
// Type "site" extends the §63 list for provenance-grade facts.
func entitiesFromDomain(domain string) []entityCand {
	if strings.TrimSpace(domain) == "" {
		return nil
	}
	return []entityCand{{typ: "site", name: domain, confidence: 0.9}}
}

// titleCandidates picks capitalized words as guesses. Filename artifacts
// (underscores, extensions) are skipped. Weak signal, low confidence,
// always marked deterministic — precision comes with the LLM.
func titleCandidates(title string) []entityCand {
	seen := map[string]bool{}
	var out []entityCand
	for _, w := range strings.Fields(title) {
		w = strings.Trim(w, "\"'()[],.:;!?")
		if strings.ContainsAny(w, "_/.") {
			continue
		}
		if len([]rune(w)) < 2 || seen[strings.ToLower(w)] {
			continue
		}
		if r := []rune(w)[0]; !unicode.IsUpper(r) {
			continue
		}
		seen[strings.ToLower(w)] = true
		out = append(out, entityCand{typ: "concept", name: w, confidence: 0.3})
	}
	return out
}

// contentEntities mines frequent capitalized tokens from extracted text.
// A name repeated across a document is evidence of a real entity.
// Mined concepts cap at 0.5 by design: text mining suggests, it never
// asserts. Higher confidence must come from structure (folders, domains),
// an LLM pass, or the user. Only folder topics drive relations.
// Tokens that only ever open sentences are usually verbs ("Built",
// "Designed"), not entities, and are skipped. Deterministic, no lists.
func contentEntities(text string) []entityCand {
	counts := map[string]int{}
	initial := map[string]int{}
	casing := map[string]string{}
	words := strings.Fields(text)
	for i, w := range words {
		w = strings.Trim(w, "\"'()[],.:;!?")
		if len([]rune(w)) < 2 || strings.ContainsAny(w, "_/") {
			continue
		}
		if r := []rune(w)[0]; !unicode.IsUpper(r) {
			continue
		}
		key := strings.ToLower(w)
		counts[key]++
		if _, ok := casing[key]; !ok {
			casing[key] = w
		}
		if i == 0 || endsSentence(words[i-1]) {
			initial[key]++
		}
	}
	var out []entityCand
	for key, n := range counts {
		if n < 3 {
			continue
		}
		// Only-ever-initial means verb-like ("Built this. Built that.").
		// Anything also appearing mid-sentence stays.
		if float64(initial[key])/float64(n) >= 1.0 {
			continue
		}
		conf := 0.4 + 0.05*float64(n-3)
		if conf > 0.5 {
			conf = 0.5
		}
		out = append(out, entityCand{typ: "concept", name: casing[key], confidence: conf})
	}
	return out
}

// endsSentence reports whether a raw token ends a sentence or bullet.
func endsSentence(prev string) bool {
	for _, mark := range []string{".", "!", "?", "•", "-"} {
		if strings.HasSuffix(prev, mark) {
			return true
		}
	}
	return false
}

// enrichItem links one item to its topics and entities. Idempotent:
// re-running never duplicates links.
func enrichItem(ctx context.Context, pool *pgxpool.Pool, user, itemID string) error {
	var sourceType, title, domain, origRaw, content string
	err := pool.QueryRow(ctx, `
	  SELECT i.source_type, COALESCE(i.title,''), COALESCE(i.domain,''),
	         COALESCE(i.original::text,'{}'), COALESCE(c.extracted_text,'')
	  FROM items i LEFT JOIN item_contents c ON c.item_id = i.id
	  WHERE i.id=$1 AND i.user_id=$2`, itemID, user).Scan(&sourceType, &title, &domain, &origRaw, &content)
	if err != nil {
		return err
	}
	var orig map[string]any
	_ = json.Unmarshal([]byte(origRaw), &orig)
	var folders []string
	if fl, ok := orig["folders"].([]any); ok {
		for _, f := range fl {
			if s, ok := f.(string); ok {
				folders = append(folders, s)
			}
		}
	}

	// Stable identities: names are the identity, so topic/entity rows
	// survive re-runs and corrections keep pointing at the same rows.
	// Only links that no longer apply are dropped; orphans are pruned.
	rejected := map[string]bool{}
	if rejRows, err := pool.Query(ctx, `SELECT kind, target_id::text FROM user_corrections WHERE item_id=$1`, itemID); err == nil {
		rejected = rejectedSet(rejRows)
	}

	keepTopics := []string{}
	for _, t := range topicsFromFolders(folders) {
		var tid string
		err := pool.QueryRow(ctx, `
		  INSERT INTO topics (user_id, name, confidence, source)
		  VALUES ($1,$2,$3,'deterministic')
		  ON CONFLICT (user_id, name) DO UPDATE SET confidence = GREATEST(topics.confidence, $3)
		  RETURNING id`, user, t.name, t.confidence).Scan(&tid)
		if err != nil {
			continue
		}
		if rejected["topic:"+tid] {
			continue
		}
		keepTopics = append(keepTopics, tid)
		_, _ = pool.Exec(ctx, `
		  INSERT INTO item_topics (item_id, topic_id, confidence) VALUES ($1,$2,$3)
		  ON CONFLICT DO NOTHING`, itemID, tid, t.confidence)
	}
	_, _ = pool.Exec(ctx, `DELETE FROM item_topics WHERE item_id=$1 AND NOT (topic_id = ANY($2::uuid[]))`, itemID, keepTopics)
	ents := append(entitiesFromDomain(domain), titleCandidates(title)...)
	ents = append(ents, contentEntities(content)...)
	keepEntities := []string{}
	for _, e := range ents {
		var eid string
		err := pool.QueryRow(ctx, `
		  INSERT INTO entities (user_id, type, canonical_name, confidence, source)
		  VALUES ($1,$2,$3,$4,'deterministic')
		  ON CONFLICT (user_id, type, canonical_name) DO UPDATE SET confidence = GREATEST(entities.confidence, $4)
		  RETURNING id`, user, e.typ, e.name, e.confidence).Scan(&eid)
		if err != nil {
			continue
		}
		if rejected["entity:"+eid] {
			continue
		}
		keepEntities = append(keepEntities, eid)
		_, _ = pool.Exec(ctx, `
		  INSERT INTO item_entities (item_id, entity_id, confidence) VALUES ($1,$2,$3)
		  ON CONFLICT DO NOTHING`, itemID, eid, e.confidence)
	}
	_, _ = pool.Exec(ctx, `DELETE FROM item_entities WHERE item_id=$1 AND NOT (entity_id = ANY($2::uuid[]))`, itemID, keepEntities)
	// Prune orphans, but never rows the user has ruled on: a rejected
	// topic stays dead even with no links, so corrections keep working.
	_, _ = pool.Exec(ctx, `DELETE FROM topics WHERE source='deterministic'
	  AND id NOT IN (SELECT topic_id FROM item_topics)
	  AND id NOT IN (SELECT target_id FROM user_corrections WHERE kind='topic')`)
	_, _ = pool.Exec(ctx, `DELETE FROM entities WHERE source='deterministic'
	  AND id NOT IN (SELECT entity_id FROM item_entities)
	  AND id NOT IN (SELECT target_id FROM user_corrections WHERE kind='entity')`)
	embedItem(ctx, pool, title, content, itemID)
	return nil
}

// embedItem stores the meaning-vector for one item. Missing key or
// missing column (pre-vector database) skips quietly: keywords stand
// alone, and the backfill picks it up once the column exists.
func embedItem(ctx context.Context, pool *pgxpool.Pool, title, content, itemID string) {
	ec, ok := embedFromEnv()
	if !ok {
		return
	}
	input := title + "\n" + content
	if r := []rune(input); len(r) > maxEmbedChars {
		input = string(r[:maxEmbedChars])
	}
	vecs, err := ec.embedTexts([]string{input})
	if err != nil {
		log.Printf("embed item failed err=%T", err)
		return
	}
	if _, err := pool.Exec(ctx, `UPDATE item_contents SET embedding=$1 WHERE item_id=$2`,
		vectorLiteral(vecs[0]), itemID); err != nil {
		log.Printf("embed store failed err=%T", err)
	}
}

func registerGraphRoutes(mux *http.ServeMux, pool *pgxpool.Pool) {
	// POST /v1/enrich/backfill — one-shot pass over everything saved.
	mux.HandleFunc("POST /v1/enrich/backfill", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user := tenantID(r)
		rows, err := pool.Query(ctx, `SELECT id FROM items WHERE user_id=$1`, user)
		if err != nil {
			http.Error(w, "internal error", 500)
			return
		}
		defer rows.Close()
		done, failed := 0, 0
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				continue
			}
			if err := enrichItem(ctx, pool, user, id); err != nil {
				log.Printf("backfill item failed err=%T", err)
				failed++
				continue
			}
			fillTitleIfMissing(ctx, pool, id)
			done++
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"items": done, "failed": failed})
	})
}
