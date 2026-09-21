// Upload-slice tests: classification, PDF text, image dims, hostile names.
// Pure functions only � no DB, no network.
package main

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"os"
	"strings"
	"testing"
)

func TestClassifyUpload(t *testing.T) {
	pdfHead := []byte("%PDF-1.4 fake")
	pngHead := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0, 0, 0, 0}
	exeHead := []byte("MZ" + strings.Repeat("\x00", 510))

	if k, m, e := classifyUpload("notes.pdf", pdfHead); e != "" || k != "pdf" || m != "application/pdf" {
		t.Errorf("pdf rejected: %q %q %q", k, m, e)
	}
	if k, _, e := classifyUpload("shot.PNG", pngHead); e != "" || k != "image" {
		t.Errorf("png rejected: %q %q", k, e)
	}
	// Renamed executable must not pass as PDF.
	if _, _, e := classifyUpload("evil.pdf", exeHead); e == "" {
		t.Error("exe masquerading as pdf accepted")
	}
	// PNG bytes with .pdf name must not pass.
	if _, _, e := classifyUpload("evil.pdf", pngHead); e == "" {
		t.Error("png masquerading as pdf accepted")
	}
	if _, _, e := classifyUpload("run.exe", exeHead); e == "" {
		t.Error("exe accepted")
	}
	if _, _, e := classifyUpload("noext", pdfHead); e == "" {
		t.Error("extensionless file accepted")
	}
}

func TestTraversalNeutralized(t *testing.T) {
	// filepath.Base is applied in the handler; the classifier must not
	// be fooled by directory components into a different type.
	if _, _, e := classifyUpload("../../etc/passwd.pdf", []byte("%PDF-1.4 x")); e != "" {
		t.Errorf("basename pdf rejected: %q", e)
	}
}

// minimalPDF builds a valid 1-page PDF in-memory so extraction is tested
// against real bytes, not mocks.
func minimalPDF(text string) []byte {
	esc := strings.ReplaceAll(text, "(", "\\(")
	stream := fmt.Sprintf("BT /F1 12 Tf 10 10 Td (%s) Tj ET", esc)
	objs := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 200 200] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(stream), stream),
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
	}
	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n")
	offsets := []int{}
	for i, o := range objs {
		offsets = append(offsets, buf.Len())
		fmt.Fprintf(&buf, "%d 0 obj\n%s\nendobj\n", i+1, o)
	}
	xref := buf.Len()
	fmt.Fprintf(&buf, "xref\n0 %d\n0000000000 65535 f \n", len(objs)+1)
	for _, off := range offsets {
		fmt.Fprintf(&buf, "%010d 00000 n \n", off)
	}
	fmt.Fprintf(&buf, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF", len(objs)+1, xref)
	return buf.Bytes()
}

func TestExtractPDFText(t *testing.T) {
	data := minimalPDF("Hello helmme Postgres")
	text, pages, emsg := extractPDFText(data)
	if emsg != "" {
		t.Fatalf("extraction failed: %q", emsg)
	}
	if pages != 1 {
		t.Errorf("pages=%d want 1", pages)
	}
	if !strings.Contains(text, "Hello helmme Postgres") {
		t.Errorf("text missing, got %q", text)
	}
	if _, _, emsg := extractPDFText([]byte("not a pdf at all")); emsg == "" {
		t.Error("garbage accepted as PDF")
	}
}

func TestExtractPDFTextPerCharPositioning(t *testing.T) {
	// Word-style exports position every glyph separately. Extraction must
	// still read "Hello", not "H e l l o".
	var runs strings.Builder
	for _, r := range "Hello Postgres" {
		ch := string(r)
		if ch == "(" || ch == ")" {
			ch = "\\" + ch
		}
		fmt.Fprintf(&runs, "(%s) Tj ", ch)
	}
	data := minimalPDFStream(runs.String())
	text, _, emsg := extractPDFText(data)
	if emsg != "" {
		t.Fatalf("extraction failed: %q", emsg)
	}
	joined := strings.Join(strings.Fields(text), " ")
	if !strings.Contains(joined, "Hello Postgres") {
		t.Errorf("words broken apart, got %q", text)
	}
}

func TestExtractPDFTextWordGaps(t *testing.T) {
	// Runs placed far apart must stay separate words; touching runs must not
	// gain spaces. Catches "app/MarketMate"-style merges.
	stream := "BT /F1 12 Tf 10 0 0 10 10 10 Tm (Hello) Tj 10 0 0 10 100 10 Tm (World) Tj ET"
	text, _, emsg := extractPDFText(minimalPDFStream(stream))
	if emsg != "" {
		t.Fatalf("extraction failed: %q", emsg)
	}
	joined := strings.Join(strings.Fields(text), " ")
	if !strings.Contains(joined, "Hello World") {
		t.Errorf("separate words merged or split, got %q", text)
	}
	if strings.Contains(joined, "HelloWorld") {
		t.Errorf("missing space between words, got %q", text)
	}
}

// minimalPDFStream is minimalPDF with a caller-supplied content stream.
func minimalPDFStream(stream string) []byte {
	objs := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 200 200] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>",
		fmt.Sprintf("<< /Length %d >>\nstream\nBT /F1 12 Tf 10 10 Td %sET\nendstream", len(stream)+30, stream),
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
	}
	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n")
	offsets := []int{}
	for i, o := range objs {
		offsets = append(offsets, buf.Len())
		fmt.Fprintf(&buf, "%d 0 obj\n%s\nendobj\n", i+1, o)
	}
	xref := buf.Len()
	fmt.Fprintf(&buf, "xref\n0 %d\n0000000000 65535 f \n", len(objs)+1)
	for _, off := range offsets {
		fmt.Fprintf(&buf, "%010d 00000 n \n", off)
	}
	fmt.Fprintf(&buf, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF", len(objs)+1, xref)
	return buf.Bytes()
}

func TestDumpMinimalPDF(t *testing.T) {
	// Fixture generator for live upload checks, not an assertion test.
	if out := os.Getenv("HELMME_DUMP_PDF"); out != "" {
		if err := os.WriteFile(out, minimalPDF("Postgres pooling helmme"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestImageDims(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 3, 2))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	w, h, ok := imageDims(buf.Bytes())
	if !ok || w != 3 || h != 2 {
		t.Errorf("dims=%dx%d ok=%v want 3x2 true", w, h, ok)
	}
	if _, _, ok := imageDims([]byte("nope")); ok {
		t.Error("garbage reported dimensions")
	}
}

func TestCleanExtracted(t *testing.T) {
	in := "Flutterwave\x00 and Stripe\x01\nNew line\tkept"
	got := cleanExtracted(in)
	if strings.ContainsAny(got, "\x00\x01") {
		t.Errorf("control chars survived: %q", got)
	}
	if !strings.Contains(got, "Flutterwave and Stripe\nNew line\tkept") {
		t.Errorf("printable text altered: %q", got)
	}
}
