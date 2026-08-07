package httpapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/discovery/backend/internal/config"
	"github.com/discovery/backend/internal/httpapi"
	"github.com/discovery/backend/internal/ingest"
	"github.com/discovery/backend/internal/llm"
	"github.com/discovery/backend/internal/storage"
	"github.com/discovery/backend/internal/store"
)

type companyContextDoc struct {
	Text string `json:"text"`
}

type companyContextEnvelopeDoc struct {
	Data companyContextDoc `json:"data"`
}

func getCompanyContext(t *testing.T, srv *httptest.Server, companyID string) companyContextDoc {
	t.Helper()
	resp, err := http.Get(srv.URL + "/companies/" + companyID + "/context")
	if err != nil {
		t.Fatalf("get company context: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get company context status = %d", resp.StatusCode)
	}
	var env companyContextEnvelopeDoc
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		t.Fatalf("decode company context response: %v", err)
	}
	return env.Data
}

func deleteDocument(t *testing.T, srv *httptest.Server, companyID, documentID string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodDelete, srv.URL+"/companies/"+companyID+"/documents/"+documentID, nil)
	if err != nil {
		t.Fatalf("build delete request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("delete document: %v", err)
	}
	return resp
}

func resummariseDocument(t *testing.T, srv *httptest.Server, companyID, documentID string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/companies/"+companyID+"/documents/"+documentID+"/resummarise", nil)
	if err != nil {
		t.Fatalf("build resummarise request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("resummarise document: %v", err)
	}
	return resp
}

func scriptedSummaryReply(theme string) llm.ToolCall {
	body, _ := json.Marshal(map[string]any{
		"theme":         theme,
		"context":       "context for " + theme,
		"topics":        []string{"topic"},
		"pains":         []string{"pain"},
		"examples":      []string{"example"},
		"opportunities": []string{"opportunity"},
	})
	return llm.ToolCall{Name: llm.SummariseToolName, Arguments: body}
}

// TestCompanyContext_ReadyDocumentAppendsSummaryToCompanyContext covers the
// "add" behaviour in ticket 06: once a Document reaches 'ready', its
// summary shows up in the Company Context, in the same text a prompt reads.
func TestCompanyContext_ReadyDocumentAppendsSummaryToCompanyContext(t *testing.T) {
	srv, _ := newTestServer(t)
	company := createCompany(t, srv, "Aurora Logistics")

	before := getCompanyContext(t, srv, company.ID)
	if before.Text != "" {
		t.Fatalf("expected an empty Company Context before any Document, got %q", before.Text)
	}

	result := uploadOne(t, srv, company.ID, "sales-process.md", []byte("# Sales process\n\nCold call, then demo."))
	if result.Document == nil {
		t.Fatal("expected a document in the result")
	}
	doc := waitForTerminalStatus(t, srv, company.ID, result.Document.ID)
	if doc.Status != "ready" {
		t.Fatalf("status = %q, want %q", doc.Status, "ready")
	}

	after := getCompanyContext(t, srv, company.ID)
	if !strings.Contains(after.Text, "sales-process.md") {
		t.Errorf("company context = %q, want it to name the document", after.Text)
	}
	if !strings.Contains(after.Text, "test fixture") {
		t.Errorf("company context = %q, want the scripted summary's theme", after.Text)
	}
}

// TestCompanyContext_ConcurrentIngestionsLoseNeither covers the row-lock
// requirement (ADR-0003): several Documents finishing at once — which is
// exactly what happens when the ingest Worker's errgroup processes a batch
// concurrently — must all end up in the Company Context, not just the last
// writer.
func TestCompanyContext_ConcurrentIngestionsLoseNeither(t *testing.T) {
	srv, _ := newTestServer(t)
	company := createCompany(t, srv, "Concurrent Uploads Inc")

	const n = 6
	ids := make([]string, n)
	for i := 0; i < n; i++ {
		filename := "doc" + strconv.Itoa(i) + ".md"
		result := uploadOne(t, srv, company.ID, filename, []byte("# Doc "+strconv.Itoa(i)+"\n\nBody."))
		if result.Document == nil {
			t.Fatalf("expected a document for %s", filename)
		}
		ids[i] = result.Document.ID
	}

	for _, id := range ids {
		doc := waitForTerminalStatus(t, srv, company.ID, id)
		if doc.Status != "ready" {
			t.Fatalf("document %s status = %q, want %q", id, doc.Status, "ready")
		}
	}

	after := getCompanyContext(t, srv, company.ID)
	entries := strings.Split(after.Text, "\n\n---\n\n")
	if len(entries) != n {
		t.Fatalf("company context has %d entries, want %d (lost an entry under concurrent append): %q", len(entries), n, after.Text)
	}
	for i := 0; i < n; i++ {
		if !strings.Contains(after.Text, "doc"+strconv.Itoa(i)+".md") {
			t.Errorf("company context missing entry for doc%d.md: %q", i, after.Text)
		}
	}
}

// TestDeleteDocument_RemovesCompanyContextEntry covers the "remove"
// behaviour in ticket 06: deleting a ready Document removes its Company
// Context entry in the same transaction, and the Document itself is gone.
func TestDeleteDocument_RemovesCompanyContextEntry(t *testing.T) {
	srv, _ := newTestServer(t)
	company := createCompany(t, srv, "Riverside Manufacturing")

	kept := uploadOne(t, srv, company.ID, "kept.md", []byte("# Kept\n\nStays."))
	removed := uploadOne(t, srv, company.ID, "removed.md", []byte("# Removed\n\nGoes."))
	waitForTerminalStatus(t, srv, company.ID, kept.Document.ID)
	waitForTerminalStatus(t, srv, company.ID, removed.Document.ID)

	before := getCompanyContext(t, srv, company.ID)
	if !strings.Contains(before.Text, "removed.md") {
		t.Fatalf("expected removed.md in company context before delete: %q", before.Text)
	}

	resp := deleteDocument(t, srv, company.ID, removed.Document.ID)
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete status = %d, want %d", resp.StatusCode, http.StatusNoContent)
	}

	after := getCompanyContext(t, srv, company.ID)
	if strings.Contains(after.Text, "removed.md") {
		t.Errorf("company context still contains removed.md: %q", after.Text)
	}
	if !strings.Contains(after.Text, "kept.md") {
		t.Errorf("company context lost kept.md's entry too: %q", after.Text)
	}

	getResp, err := http.Get(srv.URL + "/companies/" + company.ID + "/documents/" + removed.Document.ID)
	if err != nil {
		t.Fatalf("get deleted document: %v", err)
	}
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusNotFound {
		t.Errorf("get deleted document status = %d, want %d", getResp.StatusCode, http.StatusNotFound)
	}
}

func TestDeleteDocument_NotFound(t *testing.T) {
	srv, _ := newTestServer(t)
	company := createCompany(t, srv, "Vera Accounting")

	resp := deleteDocument(t, srv, company.ID, "00000000-0000-0000-0000-000000000000")
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

// TestResummariseDocument_ReplacesEntryWithoutReupload covers the
// "re-summarise" behaviour in ticket 06: reprocessing the same raw_text
// with a different scripted reply replaces the existing Company Context
// entry in place, with no new upload involved.
func TestResummariseDocument_ReplacesEntryWithoutReupload(t *testing.T) {
	provider := llm.NewScriptedProvider()
	provider.SetDefault(llm.SummariseToolName, scriptedSummaryReply("first pass"))
	srv, db := newTestServerWithProvider(t, provider)
	company := createCompany(t, srv, "Second Chance Retail")

	result := uploadOne(t, srv, company.ID, "onboarding.md", []byte("# Onboarding\n\nStep one, step two."))
	doc := waitForTerminalStatus(t, srv, company.ID, result.Document.ID)
	if doc.Status != "ready" {
		t.Fatalf("status = %q, want %q", doc.Status, "ready")
	}

	before := getCompanyContext(t, srv, company.ID)
	if !strings.Contains(before.Text, "first pass") {
		t.Fatalf("expected the first summary's theme in company context: %q", before.Text)
	}

	provider.Enqueue(llm.SummariseToolName, scriptedSummaryReply("second pass"))
	resp := resummariseDocument(t, srv, company.ID, result.Document.ID)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("resummarise status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	var env documentEnvelopeDoc
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		t.Fatalf("decode resummarise response: %v", err)
	}
	if env.Data.Status != "ready" {
		t.Errorf("status after resummarise = %q, want %q", env.Data.Status, "ready")
	}

	after := getCompanyContext(t, srv, company.ID)
	if strings.Contains(after.Text, "first pass") {
		t.Errorf("company context still has the stale entry: %q", after.Text)
	}
	if !strings.Contains(after.Text, "second pass") {
		t.Errorf("company context missing the re-summarised entry: %q", after.Text)
	}
	entries := strings.Split(after.Text, "\n\n---\n\n")
	if len(entries) != 1 {
		t.Errorf("expected the entry to be replaced in place, not duplicated: %q", after.Text)
	}

	// No re-upload: exactly one Document row for this Company throughout.
	var count int
	if err := db.Pool.QueryRow(context.Background(), "SELECT count(*) FROM documents WHERE company_id = $1", company.ID).Scan(&count); err != nil {
		t.Fatalf("count documents: %v", err)
	}
	if count != 1 {
		t.Errorf("document count = %d, want 1 (resummarise must not create a new Document)", count)
	}
}

func TestResummariseDocument_NotYetParsedIsRejected(t *testing.T) {
	slow := &slowProvider{inner: defaultScriptedProvider(), delay: 300 * time.Millisecond}
	srv, _ := newTestServerWithProvider(t, slow)
	company := createCompany(t, srv, "Torque Industrial")

	result := uploadOne(t, srv, company.ID, "process.md", []byte("# Process\n\nDetails."))
	if result.Document.Status != "parsing" {
		t.Fatalf("expected the document to still be parsing right after upload, got %q", result.Document.Status)
	}

	resp := resummariseDocument(t, srv, company.ID, result.Document.ID)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusUnprocessableEntity)
	}
}

// TestIngest_DeletesObjectFromStorageWhenEnabled and
// TestIngest_KeepsObjectInStorageWhenDisabled cover the
// STORAGE_DELETE_AFTER_INGEST checkbox directly, rather than relying on
// whatever the root .env happens to be set to — the disabled path is the
// one the rest of this file's tests (and document_test.go's) implicitly
// depend on, since re-summarise never needs the object back either way.
func TestIngest_DeletesObjectFromStorageWhenEnabled(t *testing.T) {
	objectStore, s3Key := ingestOneWithDeleteAfterIngest(t, true)

	if _, err := objectStore.Get(context.Background(), s3Key); err == nil {
		t.Error("expected the object to be deleted from storage once STORAGE_DELETE_AFTER_INGEST is enabled")
	}
}

func TestIngest_KeepsObjectInStorageWhenDisabled(t *testing.T) {
	objectStore, s3Key := ingestOneWithDeleteAfterIngest(t, false)

	data, err := objectStore.Get(context.Background(), s3Key)
	if err != nil {
		t.Fatalf("expected the object to still be in storage with STORAGE_DELETE_AFTER_INGEST disabled: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected the retrieved object to have content")
	}
}

// ingestOneWithDeleteAfterIngest boots a server with DeleteAfterIngest
// forced to deleteAfter (independent of whatever the root .env happens to
// have), uploads one Document, waits for it to become ready, and returns
// the object store plus the S3 key to assert against directly — the
// document handler response never carries s3_key, so the key comes
// straight from the row.
func ingestOneWithDeleteAfterIngest(t *testing.T, deleteAfter bool) (*storage.Storage, string) {
	t.Helper()
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	db, err := store.New(ctx, cfg.DatabaseURL)
	if err != nil {
		t.Fatalf("connect store: %v", err)
	}
	t.Cleanup(db.Close)
	if err := db.Migrate(cfg.DatabaseURL); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := db.Pool.Exec(ctx, "TRUNCATE companies CASCADE"); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	objectStore, err := storage.New(ctx, storage.Config{
		Endpoint:     cfg.S3Endpoint,
		Region:       cfg.S3Region,
		Bucket:       cfg.S3Bucket,
		AccessKey:    cfg.S3AccessKey,
		SecretKey:    cfg.S3SecretKey,
		UsePathStyle: cfg.S3UsePathStyle,
	})
	if err != nil {
		t.Fatalf("connect object store: %v", err)
	}
	if err := objectStore.EnsureBucket(ctx); err != nil {
		t.Fatalf("ensure bucket: %v", err)
	}

	ingestor := &ingest.Ingestor{
		Documents:         db,
		Objects:           objectStore,
		Summariser:        &llm.Summariser{Provider: defaultScriptedProvider(), Model: "test-model"},
		MaxUploadBytes:    cfg.MaxUploadBytes,
		DeleteAfterIngest: deleteAfter,
	}

	worker := &ingest.Worker{Ingestor: ingestor, PollInterval: 20 * time.Millisecond, BatchSize: 8}
	workerCtx, cancelWorker := context.WithCancel(ctx)
	workerDone := make(chan struct{})
	go func() {
		defer close(workerDone)
		worker.Run(workerCtx)
	}()
	t.Cleanup(func() {
		cancelWorker()
		<-workerDone
	})

	srv := httptest.NewServer(httpapi.NewRouter(db, objectStore, ingestor, "*"))
	t.Cleanup(srv.Close)

	company := createCompany(t, srv, "Storage Flag Co")
	result := uploadOne(t, srv, company.ID, "notes.md", []byte("# Notes\n\nSome content."))
	doc := waitForTerminalStatus(t, srv, company.ID, result.Document.ID)
	if doc.Status != "ready" {
		t.Fatalf("status = %q, want %q", doc.Status, "ready")
	}

	var s3Key string
	if err := db.Pool.QueryRow(ctx, "SELECT s3_key FROM documents WHERE id = $1", doc.ID).Scan(&s3Key); err != nil {
		t.Fatalf("read s3_key: %v", err)
	}

	return objectStore, s3Key
}
