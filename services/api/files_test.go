// File-serving tests: path safety first, since paths protect bytes.
package main

import (
	"testing"
)

func TestSafeJoin(t *testing.T) {
	ok, good := safeJoin("/data/uploads", "abc123.pdf")
	if !good || ok == "" {
		t.Errorf("plain ref rejected: %q %v", ok, good)
	}
	for _, evil := range []string{
		"", "../secret", "a/../../x", "/abs/path", "..\\win",
		"sub/dir.pdf", ".",
	} {
		if _, good := safeJoin("/data/uploads", evil); good {
			t.Errorf("escape accepted: %q", evil)
		}
	}
}
