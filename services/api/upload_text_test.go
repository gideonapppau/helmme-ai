// Text-file upload rules. Plain words: notes and markdown ride along with
// PDFs and images; programs wearing a .txt mask are refused.
// Pure checks only, no database needed.
package main

import (
	"archive/zip"
	"bytes"
	"testing"
)

func TestClassifyTextUpload(t *testing.T) {
	kind, mime, msg := classifyUpload("notes.md", []byte("# hello\nworld\n"))
	if msg != "" || kind != "text" {
		t.Errorf("markdown refused: kind=%q mime=%q msg=%q", kind, mime, msg)
	}
	kind, _, msg = classifyUpload("list.txt", []byte("eggs\nmilk\n"))
	if msg != "" || kind != "text" {
		t.Errorf("txt refused: kind=%q msg=%q", kind, msg)
	}
	if _, _, msg := classifyUpload("run.txt", []byte{0x7f, 'E', 'L', 'F', 0x01}); msg == "" {
		t.Error("binary masked as .txt accepted")
	}
	if _, _, msg := classifyUpload("app.exe", []byte("MZ")); msg == "" {
		t.Error("exe accepted")
	}
}

// minDocx builds the smallest Word file that still counts: one sentence
// plus the empty links file the reader insists on.
func minDocx(t *testing.T, sentence string) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, err := w.Create("word/document.xml")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.Write([]byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">` +
		`<w:body><w:p><w:r><w:t>` + sentence + `</w:t></w:r></w:p></w:body></w:document>`))
	r, err := w.Create("word/_rels/document.xml.rels")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = r.Write([]byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"></Relationships>`))
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestClassifyDocxUpload(t *testing.T) {
	data := minDocx(t, "Hello docx loop")
	kind, _, msg := classifyUpload("notes.docx", data)
	if msg != "" || kind != "docx" {
		t.Errorf("docx refused: kind=%q msg=%q", kind, msg)
	}
	if _, _, msg := classifyUpload("notes.docx", []byte("not a zip")); msg == "" {
		t.Error("fake docx accepted")
	}
}

func TestExtractDOCXText(t *testing.T) {
	text, msg := extractDOCXText(minDocx(t, "Hello docx loop"))
	if msg != "" {
		t.Fatalf("docx unreadable: %s", msg)
	}
	if !bytes.Contains([]byte(text), []byte("Hello docx loop")) {
		t.Errorf("sentence lost: %q", text)
	}
	if _, msg := extractDOCXText([]byte("garbage")); msg == "" {
		t.Error("garbage docx accepted")
	}
}
