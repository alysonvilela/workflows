package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/discovery/backend/internal/config"
	"github.com/discovery/backend/internal/llm"
)

type documentDoc struct {
	ID        string  `json:"id"`
	CompanyID string  `json:"company_id"`
	Title     string  `json:"title"`
	Kind      string  `json:"kind"`
	Status    string  `json:"status"`
	Error     *string `json:"error"`
	Checksum  string  `json:"checksum"`
	RawText   *string `json:"raw_text"`
}

type uploadResultDoc struct {
	Filename string       `json:"filename"`
	Document *documentDoc `json:"document"`
	Error    *string      `json:"error"`
}

type uploadEnvelope struct {
	Data []uploadResultDoc `json:"data"`
}

type documentEnvelopeDoc struct {
	Data documentDoc `json:"data"`
}

type listDocumentsEnvelope struct {
	Data []documentDoc `json:"data"`
	Meta struct {
		Pagination struct {
			Total int `json:"total"`
		} `json:"pagination"`
	} `json:"meta"`
}

// uploadFiles posts a multipart/form-data request with one "files" part per
// entry in files (filename -> content).
func uploadFiles(t *testing.T, srv *httptest.Server, companyID string, files map[string][]byte) *http.Response {
	t.Helper()

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	for filename, content := range files {
		part, err := w.CreateFormFile("files", filename)
		if err != nil {
			t.Fatalf("create form file %s: %v", filename, err)
		}
		if _, err := part.Write(content); err != nil {
			t.Fatalf("write content for %s: %v", filename, err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, srv.URL+"/companies/"+companyID+"/documents", &body)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	return resp
}

func uploadOne(t *testing.T, srv *httptest.Server, companyID, filename string, content []byte) uploadResultDoc {
	t.Helper()
	resp := uploadFiles(t, srv, companyID, map[string][]byte{filename: content})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("upload %q: status %d", filename, resp.StatusCode)
	}
	var env uploadEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		t.Fatalf("decode upload response: %v", err)
	}
	if len(env.Data) != 1 {
		t.Fatalf("expected 1 result, got %d", len(env.Data))
	}
	return env.Data[0]
}

func getDocument(t *testing.T, srv *httptest.Server, companyID, documentID string) documentDoc {
	t.Helper()
	resp, err := http.Get(srv.URL + "/companies/" + companyID + "/documents/" + documentID)
	if err != nil {
		t.Fatalf("get document: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get document status = %d", resp.StatusCode)
	}
	var env documentEnvelopeDoc
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		t.Fatalf("decode get response: %v", err)
	}
	return env.Data
}

// waitForTerminalStatus polls a Document until Process — running in the
// background worker started by the test server — has taken it out of
// 'parsing'. Upload itself (Stage) never waits on this; it is what a
// consultant's own polling would observe, matching the frontend.
func waitForTerminalStatus(t *testing.T, srv *httptest.Server, companyID, documentID string) documentDoc {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		doc := getDocument(t, srv, companyID, documentID)
		if doc.Status != "parsing" {
			return doc
		}
		if time.Now().After(deadline) {
			t.Fatalf("document %s still %q after 5s", documentID, doc.Status)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestUploadDocument_MarkdownBecomesReadyWithRawText(t *testing.T) {
	srv, _ := newTestServer(t)
	company := createCompany(t, srv, "Aurora Logistics")

	result := uploadOne(t, srv, company.ID, "sales-process.md", []byte("# Sales process\n\nCold call, then demo."))

	if result.Error != nil {
		t.Fatalf("unexpected per-file error: %s", *result.Error)
	}
	if result.Document == nil {
		t.Fatal("expected a document in the result")
	}

	doc := waitForTerminalStatus(t, srv, company.ID, result.Document.ID)
	if doc.Status != "ready" {
		t.Errorf("status = %q, want %q", doc.Status, "ready")
	}
	if doc.Kind != "markdown" {
		t.Errorf("kind = %q, want %q", doc.Kind, "markdown")
	}
	if doc.Error != nil {
		t.Errorf("expected no error on a ready document, got %q", *doc.Error)
	}
	if doc.RawText == nil || *doc.RawText != "# Sales process\n\nCold call, then demo." {
		t.Errorf("raw_text = %v, want the uploaded content", doc.RawText)
	}
}

// TestUploadDocument_RespondsBeforeProcessingCompletes is the async
// contract itself: upload must not block on the summariser's LLM call.
func TestUploadDocument_RespondsBeforeProcessingCompletes(t *testing.T) {
	slow := &slowProvider{inner: defaultScriptedProvider(), delay: 300 * time.Millisecond}
	srv, _ := newTestServerWithProvider(t, slow)
	company := createCompany(t, srv, "Torque Industrial")

	start := time.Now()
	result := uploadOne(t, srv, company.ID, "process.md", []byte("# Process\n\nDetails."))
	elapsed := time.Since(start)

	if elapsed >= slow.delay {
		t.Errorf("upload took %s, want well under the %s summariser delay — Stage must not wait on Process", elapsed, slow.delay)
	}
	if result.Document == nil {
		t.Fatal("expected a document in the result")
	}
	if result.Document.Status != "parsing" {
		t.Errorf("status right after upload = %q, want %q (still unprocessed)", result.Document.Status, "parsing")
	}

	doc := waitForTerminalStatus(t, srv, company.ID, result.Document.ID)
	if doc.Status != "ready" {
		t.Errorf("status after the background worker runs = %q, want %q", doc.Status, "ready")
	}
}

// slowProvider adds an artificial delay before delegating — the fixture
// TestUploadDocument_RespondsBeforeProcessingCompletes uses to prove
// upload does not wait on the Provider call.
type slowProvider struct {
	inner llm.Provider
	delay time.Duration
}

func (p *slowProvider) Complete(ctx context.Context, model, prompt string, tools []llm.Tool) (llm.ToolCall, error) {
	select {
	case <-time.After(p.delay):
	case <-ctx.Done():
		return llm.ToolCall{}, ctx.Err()
	}
	return p.inner.Complete(ctx, model, prompt, tools)
}

func TestUploadDocument_CSVBecomesReadyWithRawText(t *testing.T) {
	srv, _ := newTestServer(t)
	company := createCompany(t, srv, "Riverside Manufacturing")

	result := uploadOne(t, srv, company.ID, "tickets.csv", []byte("subject,minutes\nInvoice mismatch,45"))

	if result.Error != nil {
		t.Fatalf("unexpected per-file error: %s", *result.Error)
	}
	if result.Document == nil {
		t.Fatal("expected a document in the result")
	}

	doc := waitForTerminalStatus(t, srv, company.ID, result.Document.ID)
	if doc.Status != "ready" {
		t.Errorf("status = %q, want %q", doc.Status, "ready")
	}
	if doc.Kind != "csv" {
		t.Errorf("kind = %q, want %q", doc.Kind, "csv")
	}
	if doc.Error != nil {
		t.Errorf("expected no error on a ready document, got %q", *doc.Error)
	}
	if doc.RawText == nil || *doc.RawText != "subject,minutes\nInvoice mismatch,45" {
		t.Errorf("raw_text = %v, want the uploaded content unchanged", doc.RawText)
	}
}

func TestUploadDocument_MalformedCSVRecordsFailureOnTheDocument(t *testing.T) {
	srv, _ := newTestServer(t)
	company := createCompany(t, srv, "Bem-Estar Clinic")

	result := uploadOne(t, srv, company.ID, "broken.csv", []byte(`a,"unterminated`+"\n"))

	if result.Error != nil {
		t.Fatalf("a parsing failure must not fail the request, got top-level error %q", *result.Error)
	}
	if result.Document == nil {
		t.Fatal("expected a document even though parsing failed")
	}

	doc := waitForTerminalStatus(t, srv, company.ID, result.Document.ID)
	if doc.Status != "failed" {
		t.Errorf("status = %q, want %q", doc.Status, "failed")
	}
	if doc.Error == nil || *doc.Error == "" {
		t.Error("expected a readable reason recorded on the document")
	}
}

func TestUploadDocument_OversizedFileFailsBeforeAnyProcessing(t *testing.T) {
	srv, db := newTestServer(t)
	company := createCompany(t, srv, "Vera Accounting")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	oversized := bytes.Repeat([]byte("a"), int(cfg.MaxUploadBytes)+1)

	result := uploadOne(t, srv, company.ID, "huge.md", oversized)

	if result.Document != nil {
		t.Fatalf("expected no document for an oversized file, got %+v", result.Document)
	}
	if result.Error == nil || *result.Error == "" {
		t.Fatal("expected a readable error message for an oversized file")
	}

	var count int
	if err := db.Pool.QueryRow(context.Background(), "SELECT count(*) FROM documents WHERE company_id = $1", company.ID).Scan(&count); err != nil {
		t.Fatalf("count documents: %v", err)
	}
	if count != 0 {
		t.Errorf("expected no Document row for an oversized file, found %d", count)
	}
}

func TestUploadDocument_UnreadableFileRecordsFailureOnTheDocument(t *testing.T) {
	srv, _ := newTestServer(t)
	company := createCompany(t, srv, "Bem-Estar Clinic")

	invalidUTF8 := []byte{0xff, 0xfe, 0x00, 0x11}
	result := uploadOne(t, srv, company.ID, "broken.md", invalidUTF8)

	if result.Error != nil {
		t.Fatalf("a parsing failure must not fail the request, got top-level error %q", *result.Error)
	}
	if result.Document == nil {
		t.Fatal("expected a document even though parsing failed")
	}

	doc := waitForTerminalStatus(t, srv, company.ID, result.Document.ID)
	if doc.Status != "failed" {
		t.Errorf("status = %q, want %q", doc.Status, "failed")
	}
	if doc.Error == nil || *doc.Error == "" {
		t.Error("expected a readable reason recorded on the document")
	}
}

func TestUploadDocument_SameChecksumTwiceIsNotReprocessed(t *testing.T) {
	srv, db := newTestServer(t)
	company := createCompany(t, srv, "Second Chance Retail")

	content := []byte("# Onboarding\n\nStep one, step two.")
	first := uploadOne(t, srv, company.ID, "onboarding.md", content)
	second := uploadOne(t, srv, company.ID, "onboarding.md", content)

	if first.Document == nil || second.Document == nil {
		t.Fatalf("expected both uploads to resolve to a document: first=%+v second=%+v", first, second)
	}
	if first.Document.ID != second.Document.ID {
		t.Errorf("expected the same document id on re-upload, got %q then %q", first.Document.ID, second.Document.ID)
	}

	var count int
	if err := db.Pool.QueryRow(context.Background(), "SELECT count(*) FROM documents WHERE company_id = $1", company.ID).Scan(&count); err != nil {
		t.Fatalf("count documents: %v", err)
	}
	if count != 1 {
		t.Errorf("expected exactly 1 document row after uploading the same file twice, got %d", count)
	}
}

func TestUploadDocument_UnsupportedFileTypeRejectedPerFile(t *testing.T) {
	srv, _ := newTestServer(t)
	company := createCompany(t, srv, "Nimbus Freight")

	result := uploadOne(t, srv, company.ID, "notes.txt", []byte("plain text"))

	if result.Document != nil {
		t.Fatalf("expected no document for an unsupported file type, got %+v", result.Document)
	}
	if result.Error == nil || *result.Error == "" {
		t.Fatal("expected a readable error for an unsupported file type")
	}
}

func TestUploadDocument_CompanyNotFound(t *testing.T) {
	srv, _ := newTestServer(t)

	resp := uploadFiles(t, srv, "00000000-0000-0000-0000-000000000000", map[string][]byte{"a.md": []byte("hi")})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestListDocuments(t *testing.T) {
	srv, _ := newTestServer(t)
	company := createCompany(t, srv, "Multi Doc Co")

	uploadOne(t, srv, company.ID, "one.md", []byte("one"))
	uploadOne(t, srv, company.ID, "two.md", []byte("two"))

	resp, err := http.Get(srv.URL + "/companies/" + company.ID + "/documents")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var env listDocumentsEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(env.Data) != 2 {
		t.Fatalf("got %d documents, want 2", len(env.Data))
	}
	if env.Meta.Pagination.Total != 2 {
		t.Errorf("total = %d, want 2", env.Meta.Pagination.Total)
	}
	for _, d := range env.Data {
		if d.RawText != nil {
			t.Errorf("list response should omit raw_text, got %q", *d.RawText)
		}
	}
}
