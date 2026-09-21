// File capture: PDFs + images + text files + Word docs (§9, §114, §116 MVP content).
// Multipart upload -> sniffed validation -> content-addressed storage ->
// deterministic extraction (PDF text, image dimensions) -> Item + asset.
// Bytes never enter the database; storage_ref can move to S3 later.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ledongthuc/pdf"
	"github.com/nguyenthenguyen/docx"
)

const (
	maxUploadBytes  = 25 << 20 // 25MB (§26: size limits)
	maxPDFPages     = 50
	maxExtractChars = 200_000
)

// Extension allowlist. Bytes are sniffed too, extension alone proves nothing.
var uploadKinds = map[string]string{
	".pdf": "pdf",
	".png": "image", ".jpg": "image", ".jpeg": "image",
	".gif": "image", ".webp": "image",
	".md": "text", ".markdown": "text", ".txt": "text",
	".docx": "docx",
}

// classifyUpload maps filename + sniffed bytes to (kind, mime).
// Rejects mismatches (e.g. .exe renamed to .pdf) and unknown types.
func classifyUpload(filename string, head []byte) (kind, mime string, errMsg string) {
	ext := strings.ToLower(filepath.Ext(filename))
	kind, ok := uploadKinds[ext]
	if !ok {
		return "", "", "unsupported file type"
	}
	mime = http.DetectContentType(head)
	switch kind {
	case "pdf":
		if mime != "application/pdf" {
			return "", "", "file is not a PDF"
		}
	case "image":
		want := map[string]string{
			".png": "image/png", ".jpg": "image/jpeg", ".jpeg": "image/jpeg",
			".gif": "image/gif", ".webp": "image/webp",
		}[ext]
		if mime != want {
			return "", "", "image content does not match its extension"
		}
	case "text":
		// Plain words: a .txt that is secretly a program is refused.
		if !strings.HasPrefix(mime, "text/plain") {
			return "", "", "text file is not plain text"
		}
	case "docx":
		// Plain words: a Word file is a zip in disguise. Anything else
		// wearing .docx is refused.
		if mime != "application/zip" {
			return "", "", "file is not a Word document"
		}
	}
	return kind, mime, ""
}

// extractPDFText returns plain text (capped) and page count.
//
// Many exporters (Word, Google Docs) position every glyph separately, which
// makes row-grouped extraction come out as "H e l l o". Stream order keeps
// words intact, so it is preferred; row grouping is the fallback.
// Scanned-image PDFs yield no text, recorded, not an error (OCR is §97,
// explicitly later; enrichment stays at the deterministic tier).
func extractPDFText(data []byte) (string, int, string) {
	r, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", 0, "unreadable PDF"
	}
	pages := r.NumPage()
	var sb strings.Builder
	n := pages
	if n > maxPDFPages {
		n = maxPDFPages
	}
	for i := 1; i <= n; i++ {
		p := r.Page(i)
		if p.V.IsNull() {
			continue
		}
		page := streamText(p)
		if strings.TrimSpace(page) == "" {
			page = rowText(p)
		}
		sb.WriteString(page)
		sb.WriteByte('\n')
		if sb.Len() >= maxExtractChars {
			break
		}
	}
	out := sb.String()
	if len(out) > maxExtractChars {
		out = out[:maxExtractChars]
	}
	// Drop control artifacts (ligature bytes, stray escapes) that pollute
	// excerpts. Keep printable text plus line breaks.
	out = cleanExtracted(out)
	return out, pages, ""
}

// cleanExtracted strips non-printable runes while keeping line structure.
func cleanExtracted(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' {
			return r
		}
		if unicode.IsPrint(r) {
			return r
		}
		return -1
	}, s)
}

// streamText concatenates glyph runs in content-stream order, which keeps
// words whole. A space is inserted only where the horizontal gap between
// runs shows the words were separate, never between touching glyphs.
// Whitespace is normalized for search, not for display.
func streamText(p pdf.Page) string {
	var sb strings.Builder
	var prevEnd, prevY, prevSize float64
	var hasPrev bool
	for _, t := range p.Content().Text {
		if hasPrev {
			sameLine := prevSize <= 0 || (t.Y-prevY < prevSize*0.5 && prevY-t.Y < prevSize*0.5)
			threshold := prevSize * 0.25
			if threshold < 0.5 {
				threshold = 0.5
			}
			if !sameLine || t.X-prevEnd > threshold {
				sb.WriteByte(' ')
			}
		}
		sb.WriteString(t.S)
		prevEnd, prevY, prevSize = t.X+t.W, t.Y, t.FontSize
		hasPrev = true
	}
	return strings.Join(strings.Fields(sb.String()), " ")
}

// rowText groups glyphs by visual row. Used only when the stream is empty.
func rowText(p pdf.Page) string {
	var sb strings.Builder
	t, err := p.GetTextByRow()
	if err != nil {
		return ""
	}
	for _, row := range t {
		for _, w := range row.Content {
			sb.WriteString(w.S)
			sb.WriteByte(' ')
		}
	}
	return strings.Join(strings.Fields(sb.String()), " ")
}

// extractDOCXText reads a Word file's words. Plain words: .docx is a zip
// of XML pages; we unzip, keep only the text runs (<w:t>…</w:t>), and drop
// the markup. Password-locked or broken files are refused, not guessed at.
func extractDOCXText(data []byte) (string, string) {
	r, err := docx.ReadDocxFromMemory(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", "unreadable Word file"
	}
	defer r.Close()
	var parts []string
	for _, m := range docxTextRe.FindAllStringSubmatch(r.Editable().GetContent(), -1) {
		if t := strings.TrimSpace(m[1]); t != "" {
			parts = append(parts, t)
		}
	}
	out := cleanExtracted(strings.Join(parts, " "))
	if len(out) > maxExtractChars {
		out = out[:maxExtractChars]
	}
	if strings.TrimSpace(out) == "" {
		return "", "Word file has no readable text"
	}
	return out, ""
}

// docxTextRe finds Word text runs. Inside <w:t> lives only text by spec,
// so a small matcher beats a full XML parser here.
var docxTextRe = regexp.MustCompile(`<w:t[^>]*>([^<]*)</w:t>`)

// imageDims reads dimensions without decoding pixels. Formats the stdlib
// cannot parse return ok=false, stored with null dims, still searchable.
func imageDims(data []byte) (w, h int, ok bool) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return 0, 0, false
	}
	return cfg.Width, cfg.Height, true
}

func registerUploadRoutes(mux *http.ServeMux, pool *pgxpool.Pool, uploadDir string) {
	mux.HandleFunc("POST /v1/items/upload", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes+(1<<20))
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			http.Error(w, "file too big or bad upload", 400)
			return
		}
		f, hdr, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "missing file field", 400)
			return
		}
		defer f.Close()
		data, err := io.ReadAll(f)
		if err != nil || len(data) == 0 || len(data) > maxUploadBytes {
			http.Error(w, "empty or oversize file", 400)
			return
		}

		name := filepath.Base(hdr.Filename)
		if name == "" || name == "." {
			name = "upload"
		}
		if len(name) > 200 {
			name = name[len(name)-200:]
		}
		kind, mime, msg := classifyUpload(name, data)
		if msg != "" {
			http.Error(w, msg, 400)
			return
		}
		sourceType := kind // "pdf" | "image" | "text", all in allowedSourceTypes
		if kind == "docx" {
			sourceType = "file" // generic file shelf; words still indexed below
		}

		sum := sha256.Sum256(data)
		hash := hex.EncodeToString(sum[:])
		ext := strings.ToLower(filepath.Ext(name))
		stored := hash + ext
		if err := os.MkdirAll(uploadDir, 0o755); err != nil {
			log.Printf("upload dir failed")
			http.Error(w, "internal error", 500)
			return
		}
		// Content-addressed: identical bytes share one stored copy (§8).
		// Each upload still records its own Item row (an occurrence).
		if _, err := os.Stat(filepath.Join(uploadDir, stored)); os.IsNotExist(err) {
			if err := os.WriteFile(filepath.Join(uploadDir, stored), data, 0o644); err != nil {
				log.Printf("upload store failed")
				http.Error(w, "internal error", 500)
				return
			}
		}

		var text string
		orig := map[string]any{"filename": name, "mime": mime, "byte_size": len(data)}
		var width, height, pages *int
		if kind == "text" {
			// Plain words: notes and markdown arrive readable already, 			// no extraction needed, just a wash and a cap.
			text = cleanExtracted(string(data))
			if len(text) > maxExtractChars {
				text = text[:maxExtractChars]
			}
			if strings.TrimSpace(text) == "" {
				http.Error(w, "empty text file", 400)
				return
			}
		} else if kind == "docx" {
			t, emsg := extractDOCXText(data)
			if emsg != "" {
				http.Error(w, emsg, 400)
				return
			}
			text = t
			orig["has_text"] = strings.TrimSpace(t) != ""
		} else if kind == "pdf" {
			t, p, emsg := extractPDFText(data)
			if emsg != "" {
				http.Error(w, emsg, 400)
				return
			}
			text = t
			pages = &p
			orig["page_count"] = p
			orig["has_text"] = strings.TrimSpace(t) != ""
		} else {
			if w, h, ok := imageDims(data); ok {
				width, height = &w, &h
				orig["width"], orig["height"] = w, h
			}
		}

		in := ItemIn{SourceType: sourceType, RawRef: "upload/" + stored, Original: orig, Title: name}
		if vmsg := validateItem(&in); vmsg != "" {
			http.Error(w, vmsg, 400)
			return
		}
		origJSON, _ := json.Marshal(orig)

		var id string
		err = pool.QueryRow(ctx, `
		  INSERT INTO items (user_id, source_type, raw_ref, content_hash, original, title)
		  VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`,
			tenantID(r), sourceType, in.RawRef, hash, origJSON, name).Scan(&id)
		if err != nil {
			log.Printf("upload capture failed err=%T", err)
			http.Error(w, "internal error", 500)
			return
		}
		searchText := name + "\n" + text
		if _, err := pool.Exec(ctx, `
		  INSERT INTO item_contents (item_id, extracted_text, search_tsv)
		  VALUES ($1,$2,to_tsvector('english',$2 || ' ' || $3))`, id, searchText, searchNorm(searchText)); err != nil {
			log.Printf("upload index failed err=%T", err)
			http.Error(w, "internal error", 500)
			return
		}
		// The asset table only tracks PDFs and images (sizes, pages, dims).
		// Text files need no such row, their words are the whole asset.
		if kind == "pdf" || kind == "image" {
			if _, err := pool.Exec(ctx, `
			  INSERT INTO item_assets (item_id, kind, storage_ref, mime, byte_size, width, height, page_count)
			  VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
				id, kind, stored, mime, len(data), nullInt(width), nullInt(height), nullInt(pages)); err != nil {
				log.Printf("upload asset failed err=%T", err)
				http.Error(w, "internal error", 500)
				return
			}
		}
		if _, err := pool.Exec(ctx, `INSERT INTO provenance_events (item_id, kind) VALUES ($1,'captured')`, id); err != nil {
			log.Printf("upload provenance failed err=%T", err)
		}
		if err := enrichItem(ctx, pool, tenantID(r), id); err != nil {
			log.Printf("upload enrich failed err=%T", err)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id": id, "content_hash": hash, "kind": kind,
			"byte_size": len(data), "page_count": pages,
			"width": width, "height": height,
		})
	})
}

func nullInt(p *int) interface{} {
	if p == nil {
		return nil
	}
	return *p
}
