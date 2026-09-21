// Import tests: folder nesting, tracking-param dupes, bad urls, dates.
// Parser is pure — no DB needed.
package main

import (
	"strings"
	"testing"
)

const bookmarkFixture = `<!DOCTYPE NETSCAPE-Bookmark-file-1>
<META HTTP-EQUIV="Content-Type" CONTENT="text/html; charset=UTF-8">
<TITLE>Bookmarks</TITLE>
<H1>Bookmarks</H1>
<DL><p>
<DT><H3 ADD_DATE="1600000000">Research</H3>
<DL><p>
<DT><A HREF="https://example.com/pg?utm_source=x" ADD_DATE="1577836800">Postgres notes</A>
<DT><A HREF="https://example.com/pg?utm_campaign=y">Postgres notes again</A>
<DT><H3>Nested</H3>
<DL><p>
<DT><A HREF="https://nested.example/deep">Deep link</A>
</DL><p>
</DL><p>
<DT><H3>Empty</H3>
<DL><p>
</DL><p>
<DT><A HREF="javascript:void(0)">Bad link</A>
<DT><A HREF="https://notitle.example/x">   </A>
<DT><A>No href at all</A>
</DL><p>`

func TestParseBookmarks(t *testing.T) {
	marks := parseBookmarks([]byte(bookmarkFixture))
	if len(marks) != 6 {
		t.Fatalf("parsed=%d want 6 (3 good + 1 nested + 2 bad-but-listed)", len(marks))
	}
	if len(marks[0].folders) != 1 || marks[0].folders[0] != "Research" {
		t.Errorf("folders=%v want [Research]", marks[0].folders)
	}
	if !marks[0].hasDate || marks[0].addedAt.Unix() != 1577836800 {
		t.Errorf("add_date lost: %+v", marks[0])
	}
	deep := marks[2]
	if len(deep.folders) != 2 || deep.folders[1] != "Nested" {
		t.Errorf("nested folders=%v want [Research Nested]", deep.folders)
	}
	// Tracking-param variants must canonicalize identically (dupe signal).
	if canonicalizeURL(marks[0].url) != canonicalizeURL(marks[1].url) {
		t.Errorf("tracking variants differ: %q vs %q", marks[0].url, marks[1].url)
	}
}

func TestCleanBookmarkURL(t *testing.T) {
	if _, ok := cleanBookmarkURL("javascript:void(0)"); ok {
		t.Error("javascript url accepted")
	}
	if _, ok := cleanBookmarkURL(""); ok {
		t.Error("empty url accepted")
	}
	if _, ok := cleanBookmarkURL("ftp://example.com/x"); ok {
		t.Error("non-http url accepted")
	}
	u, ok := cleanBookmarkURL("  https://example.com/a  ")
	if !ok || !strings.HasPrefix(u, "https://") {
		t.Errorf("valid url rejected: %q", u)
	}
}
