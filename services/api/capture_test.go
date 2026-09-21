// Offline-capture rules. Plain words: a save may carry the phone's own id
// (so retries never make twins) and the real moment it happened.
// Pure checks only, no database needed.
package main

import (
	"strings"
	"testing"
)

func TestClientIDTooLong(t *testing.T) {
	in := &ItemIn{SourceType: "text", RawRef: "hello", ClientID: strings.Repeat("a", 121)}
	if msg := validateItem(in); msg == "" {
		t.Error("oversize client_id accepted")
	}
}

func TestClientIDOk(t *testing.T) {
	in := &ItemIn{SourceType: "text", RawRef: "hello", ClientID: "phone-123"}
	if msg := validateItem(in); msg != "" {
		t.Errorf("good client_id rejected: %s", msg)
	}
}

func TestCapturedAtRules(t *testing.T) {
	good := &ItemIn{SourceType: "text", RawRef: "hello", CapturedAt: "2026-09-21T10:00:00Z"}
	if msg := validateItem(good); msg != "" {
		t.Errorf("good captured_at rejected: %s", msg)
	}
	bad := &ItemIn{SourceType: "text", RawRef: "hello", CapturedAt: "yesterday morning"}
	if msg := validateItem(bad); msg == "" {
		t.Error("unreadable captured_at accepted")
	}
	empty := &ItemIn{SourceType: "text", RawRef: "hello"}
	if msg := validateItem(empty); msg != "" {
		t.Errorf("missing captured_at (means right now) rejected: %s", msg)
	}
}
