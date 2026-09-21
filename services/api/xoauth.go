// X OAuth 2.0 (PKCE) + bookmark backfill scaffold (§11-§14, §49, §66).
//
// Shape of the real flow; honest until keys land:
//
//	GET  /v1/sources/x/oauth/start    → authorize URL (or 503 x_not_configured)
//	GET  /v1/sources/x/oauth/callback → code→token exchange, stored server-side
//	POST /v1/sources/x/backfill       → async paged bookmark import w/ progress
//
// Security (§66): imported content is data; tokens never appear in logs or
// responses (opaque `%T` logging only); state + PKCE-S256 on every round-trip;
// redirect URI pinned to X_REDIRECT_URL. Without X_CLIENT_ID/_SECRET/_REDIRECT_URL
// every route refuses with 503 instead of faking success.
package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	xAuthorizeURL = "https://x.com/i/oauth2/authorize"
	xTokenURL     = "https://api.x.com/2/oauth2/token"
	xRevokeURL    = "https://api.x.com/2/oauth2/revoke"
	xAPIScope     = "tweet.read users.read bookmark.read offline.access"
)

// xOAuthConfig reads credentials from env only, never from the client.
func xOAuthConfig() (id, secret, redirect string, ok bool) {
	id = strings.TrimSpace(os.Getenv("X_CLIENT_ID"))
	secret = strings.TrimSpace(os.Getenv("X_CLIENT_SECRET"))
	redirect = strings.TrimSpace(os.Getenv("X_REDIRECT_URL"))
	return id, secret, redirect, id != "" && secret != "" && redirect != ""
}

func randB64URL(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func pkceChallenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

// backfillProgress records pollable progress on the sources row (§12).
func backfillProgress(ctx context.Context, pool *pgxpool.Pool, sourceID, status, note string) {
	_, _ = pool.Exec(ctx,
		`UPDATE sources SET backfill_status=$1, backfill_note=$2 WHERE id=$3`, status, note, sourceID)
}

// revokeXToken best-effort revokes the stored access token, then wipes all
// token material. Called from xConnector.Disconnect. Never logs secrets.
func revokeXToken(ctx context.Context, pool *pgxpool.Pool, user, sourceID string) {
	var access, refresh, id, secret string
	if err := pool.QueryRow(ctx,
		`SELECT access_token, refresh_token FROM sources WHERE id=$1 AND user_id=$2`,
		sourceID, user).Scan(&access, &refresh); err != nil || access == "" {
		return
	}
	id, secret, _, ok := xOAuthConfig()
	if ok {
		form := url.Values{"token": {access}, "token_type_hint": {"access_token"}}
		req, err := http.NewRequestWithContext(ctx, "POST", xRevokeURL, strings.NewReader(form.Encode()))
		if err == nil {
			req.SetBasicAuth(url.QueryEscape(id), url.QueryEscape(secret))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			client := &http.Client{Timeout: 10 * time.Second}
			if resp, err := client.Do(req); err == nil {
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			}
		}
	}
	_, _ = pool.Exec(ctx,
		`UPDATE sources SET access_token='', refresh_token='', token_expires_at=NULL,
		  token_scope='', oauth_state='', code_verifier='' WHERE id=$1`, sourceID)
	_ = refresh
}

func registerXOAuthRoutes(mux *http.ServeMux, pool *pgxpool.Pool) {
	// GET /v1/sources/x/oauth/start, begin PKCE flow, return the X authorize URL.
	mux.HandleFunc("GET /v1/sources/x/oauth/start", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		id, _, redirect, ok := xOAuthConfig()
		if !ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(503)
			json.NewEncoder(w).Encode(map[string]any{
				"error": "x_not_configured",
				"hint":  "Set X_CLIENT_ID, X_CLIENT_SECRET and X_REDIRECT_URL (docs/env.md).",
			})
			return
		}
		state, err := randB64URL(24)
		if err != nil {
			http.Error(w, "internal error", 500)
			return
		}
		verifier, err := randB64URL(48)
		if err != nil {
			http.Error(w, "internal error", 500)
			return
		}
		user := tenantID(r)
		var sourceID string
		if err := pool.QueryRow(ctx, `
		  INSERT INTO sources (user_id, kind, status, oauth_state, code_verifier)
		  VALUES ($1,'x','connecting',$2,$3)
		  ON CONFLICT (user_id, kind) DO UPDATE
		    SET status='connecting', oauth_state=$2, code_verifier=$3
		  RETURNING id`, user, state, verifier).Scan(&sourceID); err != nil {
			log.Printf("x oauth start failed err=%T", err)
			http.Error(w, "internal error", 500)
			return
		}
		q := url.Values{
			"response_type":         {"code"},
			"client_id":             {id},
			"redirect_uri":          {redirect},
			"scope":                 {xAPIScope},
			"state":                 {state},
			"code_challenge":        {pkceChallenge(verifier)},
			"code_challenge_method": {"S256"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"source_id":     sourceID,
			"authorize_url": xAuthorizeURL + "?" + q.Encode(),
		})
	})

	// GET /v1/sources/x/oauth/callback?code=…&state=…, validate state, exchange
	// code for tokens, store server-side. Tokens never leave the server.
	mux.HandleFunc("GET /v1/sources/x/oauth/callback", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		id, secret, redirect, ok := xOAuthConfig()
		if !ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(503)
			json.NewEncoder(w).Encode(map[string]any{"error": "x_not_configured"})
			return
		}
		code := r.URL.Query().Get("code")
		state := r.URL.Query().Get("state")
		if code == "" || state == "" {
			http.Error(w, "missing code or state", 400)
			return
		}
		user := tenantID(r)
		var sourceID, verifier string
		if err := pool.QueryRow(ctx,
			`SELECT id, code_verifier FROM sources
			  WHERE user_id=$1 AND kind='x' AND oauth_state=$2 AND status='connecting'`,
			user, state).Scan(&sourceID, &verifier); err != nil {
			http.Error(w, "unknown or expired oauth state", 400)
			return
		}
		form := url.Values{
			"grant_type":    {"authorization_code"},
			"code":          {code},
			"redirect_uri":  {redirect},
			"code_verifier": {verifier},
		}
		req, err := http.NewRequestWithContext(ctx, "POST", xTokenURL, strings.NewReader(form.Encode()))
		if err != nil {
			http.Error(w, "internal error", 500)
			return
		}
		req.SetBasicAuth(url.QueryEscape(id), url.QueryEscape(secret))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		client := &http.Client{Timeout: 15 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("x token exchange failed err=%T", err)
			http.Error(w, "token exchange failed", 502)
			return
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		if resp.StatusCode != 200 {
			log.Printf("x token exchange status=%d", resp.StatusCode)
			http.Error(w, "token exchange rejected", 502)
			return
		}
		var tok struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			ExpiresIn    int    `json:"expires_in"`
			Scope        string `json:"scope"`
		}
		if err := json.Unmarshal(body, &tok); err != nil || tok.AccessToken == "" {
			http.Error(w, "bad token response", 502)
			return
		}
		expires := time.Now().UTC().Add(time.Duration(tok.ExpiresIn) * time.Second)
		if _, err := pool.Exec(ctx, `
		  UPDATE sources SET access_token=$1, refresh_token=$2, token_expires_at=$3,
		    token_scope=$4, oauth_state='', code_verifier='', status='connected',
		    external_ref=COALESCE(NULLIF(external_ref,''),'x-oauth'), last_sync_at=now()
		  WHERE id=$5`, tok.AccessToken, tok.RefreshToken, expires, tok.Scope, sourceID); err != nil {
			log.Printf("x token store failed err=%T", err)
			http.Error(w, "internal error", 500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"source_id": sourceID, "status": "connected"})
	})

	// POST /v1/sources/x/backfill, async paged import with pollable progress
	// (GET /v1/sources shows total/imported/backfill). Refuses honestly at 503
	// until keys are configured; never fabricates items.
	mux.HandleFunc("POST /v1/sources/x/backfill", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if _, _, _, ok := xOAuthConfig(); !ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(503)
			json.NewEncoder(w).Encode(map[string]any{
				"error": "x_not_configured",
				"hint":  "Set X_CLIENT_ID, X_CLIENT_SECRET and X_REDIRECT_URL (docs/env.md).",
			})
			return
		}
		user := tenantID(r)
		var sourceID, access string
		if err := pool.QueryRow(ctx,
			`SELECT id, access_token FROM sources
			  WHERE user_id=$1 AND kind='x' AND status='connected'`, user).Scan(&sourceID, &access); err != nil || access == "" {
			http.Error(w, "x not connected, run oauth first", 409)
			return
		}
		var cur string
		_ = pool.QueryRow(ctx, `SELECT backfill_status FROM sources WHERE id=$1`, sourceID).Scan(&cur)
		if cur == "running" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{"source_id": sourceID, "backfill": "running"})
			return
		}
		go runXBackfill(context.Background(), pool, user, sourceID, access)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"source_id": sourceID, "backfill": "running"})
	})
}

// runXBackfill pages the X bookmarks endpoint, storing each post as normal
// memory via storeXBookmark (§13: no X silo). Progress lands on the sources
// row after every page; any failure records error + note, never partial silence.
func runXBackfill(ctx context.Context, pool *pgxpool.Pool, user, sourceID, access string) {
	backfillProgress(ctx, pool, sourceID, "running", "starting")
	client := &http.Client{Timeout: 20 * time.Second}
	get := func(u string) (int, []byte) {
		req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
		if err != nil {
			return 0, nil
		}
		req.Header.Set("Authorization", "Bearer "+access)
		resp, err := client.Do(req)
		if err != nil {
			return 0, nil
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		return resp.StatusCode, body
	}
	// Resolve our user id first (bookmarks live under /users/:id/bookmarks).
	code, body := get("https://api.x.com/2/users/me")
	if code != 200 {
		backfillProgress(ctx, pool, sourceID, "error", fmt.Sprintf("users/me answered %d", code))
		return
	}
	var me struct {
		Data struct {
			ID       string `json:"id"`
			Username string `json:"username"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &me); err != nil || me.Data.ID == "" {
		backfillProgress(ctx, pool, sourceID, "error", "could not resolve x user")
		return
	}
	total, imported := 0, 0
	pagination := ""
	for pages := 0; pages < 40; pages++ {
		u := fmt.Sprintf(
			"https://api.x.com/2/users/%s/bookmarks?max_results=100&expansions=author_id&tweet.fields=created_at,entities&user.fields=username,name%s",
			url.PathEscape(me.Data.ID), pagination)
		code, body := get(u)
		if code != 200 {
			backfillProgress(ctx, pool, sourceID, "error", fmt.Sprintf("bookmarks answered %d after %d imported", code, imported))
			return
		}
		var page struct {
			Data []struct {
				ID        string `json:"id"`
				Text      string `json:"text"`
				CreatedAt string `json:"created_at"`
				AuthorID  string `json:"author_id"`
				Entities  struct {
					URLs []struct {
						ExpandedURL string `json:"expanded_url"`
					} `json:"urls"`
				} `json:"entities"`
			} `json:"data"`
			Includes struct {
				Users []struct {
					ID       string `json:"id"`
					Name     string `json:"name"`
					Username string `json:"username"`
				} `json:"users"`
			} `json:"includes"`
			Meta struct {
				NextToken string `json:"next_token"`
			} `json:"meta"`
		}
		if err := json.Unmarshal(body, &page); err != nil {
			backfillProgress(ctx, pool, sourceID, "error", "unreadable bookmarks page")
			return
		}
		authors := map[string]struct{ name, handle string }{}
		for _, a := range page.Includes.Users {
			authors[a.ID] = struct{ name, handle string }{a.Name, a.Username}
		}
		for _, t := range page.Data {
			total++
			a := authors[t.AuthorID]
			postURL := fmt.Sprintf("https://x.com/%s/status/%s",
				handleOr(a.handle, "i"), t.ID)
			text := t.Text
			for _, e := range t.Entities.URLs {
				if e.ExpandedURL != "" {
					text += "\n" + e.ExpandedURL
				}
			}
			if _, fresh, err := storeXBookmark(ctx, pool, user, sourceID, postURL, text, a.name, a.handle, t.CreatedAt); err == nil && fresh {
				imported++
			}
		}
		_, _ = pool.Exec(ctx,
			`UPDATE sources SET total=total+$1, imported=imported+$2, last_sync_at=now() WHERE id=$3`,
			len(page.Data), imported, sourceID)
		backfillProgress(ctx, pool, sourceID, "running", fmt.Sprintf("%d imported so far", imported))
		if page.Meta.NextToken == "" {
			break
		}
		pagination = "&pagination_token=" + url.QueryEscape(page.Meta.NextToken)
	}
	_, _ = pool.Exec(ctx, `UPDATE sources SET status='connected', last_sync_at=now() WHERE id=$1`, sourceID)
	backfillProgress(ctx, pool, sourceID, "done", fmt.Sprintf("%d imported of %d seen", imported, total))
}

// handleOr keeps post URLs valid when the author expansion is missing.
func handleOr(handle, fallback string) string {
	if handle != "" {
		return handle
	}
	return fallback
}
