// OAuth helper tests. Pure functions only, no Docker, no keys needed.
package main

import (
	"os"
	"testing"
)

func TestPkceChallengeRFC7636(t *testing.T) {
	// RFC 7636 Appendix B test vector.
	if got := pkceChallenge("dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"); got != "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM" {
		t.Errorf("pkce challenge=%q, want RFC 7636 vector", got)
	}
}

func TestHandleOr(t *testing.T) {
	if got := handleOr("nasa", "i"); got != "nasa" {
		t.Errorf("handleOr kept %q", got)
	}
	if got := handleOr("", "i"); got != "i" {
		t.Errorf("handleOr fallback=%q", got)
	}
}

func TestXOAuthConfigMissing(t *testing.T) {
	os.Unsetenv("X_CLIENT_ID")
	os.Unsetenv("X_CLIENT_SECRET")
	os.Unsetenv("X_REDIRECT_URL")
	if _, _, _, ok := xOAuthConfig(); ok {
		t.Error("x configured with no keys, routes would attempt real calls")
	}
}

func TestRandB64URLLengths(t *testing.T) {
	a, err := randB64URL(24)
	if err != nil || a == "" {
		t.Fatalf("state generation failed: %v", err)
	}
	b, err := randB64URL(24)
	if err != nil || a == b {
		t.Error("oauth state not unique, CSRF protection broken")
	}
}
