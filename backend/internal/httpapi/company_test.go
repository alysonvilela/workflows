package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/discovery/backend/internal/config"
	"github.com/discovery/backend/internal/httpapi"
	"github.com/discovery/backend/internal/ingest"
	"github.com/discovery/backend/internal/llm"
	"github.com/discovery/backend/internal/storage"
	"github.com/discovery/backend/internal/store"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
)

// newTestServer boots the router against real Postgres and real rustfs
// (Seam 1): faking either would mean testing the fakes instead of the
// system. Document processing runs for real, in the background, against a
// ScriptedProvider default reply — see newTestServerWithProvider for tests
// that need to control the Provider directly (e.g. to assert that upload
// does not wait on it).
func newTestServer(t *testing.T) (*httptest.Server, *store.Store) {
	t.Helper()
	return newTestServerWithProvider(t, defaultScriptedProvider())
}

// defaultScriptedProvider is the summariser reply every test gets unless it
// asks for something else — most tests only care that ingestion reaches
// 'ready', not what the model "said".
func defaultScriptedProvider() *llm.ScriptedProvider {
	provider := llm.NewScriptedProvider()
	provider.SetDefault(llm.SummariseToolName, llm.ToolCall{
		Name: llm.SummariseToolName,
		Arguments: json.RawMessage(`{
			"theme": "test fixture",
			"context": "context assembled by the ScriptedProvider test default",
			"topics": ["topic"],
			"pains": ["pain"],
			"examples": ["example"],
			"opportunities": ["opportunity"]
		}`),
	})
	return provider
}

// newTestServerWithProvider is newTestServer with the LLM Provider
// supplied by the caller, for tests that need to script or delay the
// summariser's reply directly (see document_test.go's slow-provider test).
func newTestServerWithProvider(t *testing.T, provider llm.Provider) (*httptest.Server, *store.Store) {
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
	if err := db.MigrateRiver(ctx); err != nil {
		t.Fatalf("migrate river schema: %v", err)
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
		Summariser:        &llm.Summariser{Provider: provider, Model: "test-model"},
		MaxUploadBytes:    cfg.MaxUploadBytes,
		DeleteAfterIngest: cfg.StorageDeleteAfterIngest,
	}

	startRiverClient(t, ctx, db, ingestor)

	srv := httptest.NewServer(httpapi.NewRouter(db, objectStore, ingestor, "*"))
	t.Cleanup(srv.Close)
	return srv, db
}

// startRiverClient builds and starts the River client every ingestion test
// server needs, registers it on db.River so CreateDocumentIfNew can
// enqueue jobs, and stops it on cleanup — shared by newTestServerWithProvider
// and company_context_test.go's ingestOneWithDeleteAfterIngest, which
// otherwise duplicate this bootstrap identically.
func startRiverClient(t *testing.T, ctx context.Context, db *store.Store, ingestor *ingest.Ingestor) {
	t.Helper()

	workers := river.NewWorkers()
	river.AddWorker(workers, &ingest.ProcessDocumentWorker{Ingestor: ingestor})
	riverClient, err := river.NewClient(riverpgxv5.New(db.Pool), &river.Config{
		Queues: map[string]river.QueueConfig{
			// A tight FetchPollInterval keeps tests fast even if this
			// client's LISTEN/NOTIFY fetch is delayed under load —
			// production leaves this at River's default.
			river.QueueDefault: {MaxWorkers: 8, FetchPollInterval: 20 * time.Millisecond, FetchCooldown: 5 * time.Millisecond},
		},
		Workers: workers,
	})
	if err != nil {
		t.Fatalf("river client: %v", err)
	}
	db.River = riverClient
	if err := riverClient.Start(ctx); err != nil {
		t.Fatalf("river client start: %v", err)
	}
	// Registered after db's own Cleanup (in the caller), so cleanups run in
	// the right order (LIFO): stop the River client and let its in-flight
	// job finish before the pool it depends on is closed underneath it.
	t.Cleanup(func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := riverClient.Stop(stopCtx); err != nil {
			t.Errorf("river client stop: %v", err)
		}
	})
}

type companyDoc struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Slug          string `json:"slug"`
	DocumentCount int    `json:"document_count"`
}

type companyEnvelope struct {
	Data companyDoc `json:"data"`
}

type listEnvelope struct {
	Data []companyDoc `json:"data"`
	Meta struct {
		Pagination struct {
			Page       int `json:"page"`
			PerPage    int `json:"per_page"`
			Total      int `json:"total"`
			TotalPages int `json:"total_pages"`
		} `json:"pagination"`
	} `json:"meta"`
}

type errorEnvelope struct {
	Error struct {
		Message string            `json:"message"`
		Fields  map[string]string `json:"fields"`
	} `json:"error"`
}

func createCompany(t *testing.T, srv *httptest.Server, name string) companyDoc {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"name": name})
	resp, err := http.Post(srv.URL+"/companies", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("create company: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create company %q: status %d", name, resp.StatusCode)
	}
	var env companyEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	return env.Data
}

func TestCreateCompany(t *testing.T) {
	srv, _ := newTestServer(t)

	company := createCompany(t, srv, "Aurora Logistics")

	if company.Name != "Aurora Logistics" {
		t.Errorf("name = %q, want %q", company.Name, "Aurora Logistics")
	}
	if company.Slug != "aurora-logistics" {
		t.Errorf("slug = %q, want %q", company.Slug, "aurora-logistics")
	}
	if company.DocumentCount != 0 {
		t.Errorf("document_count = %d, want 0", company.DocumentCount)
	}
	if company.ID == "" {
		t.Error("expected a generated id")
	}
}

func TestCreateCompany_BlankNameRejectedWithInlineFieldError(t *testing.T) {
	srv, _ := newTestServer(t)

	body, _ := json.Marshal(map[string]string{"name": "   "})
	resp, err := http.Post(srv.URL+"/companies", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusUnprocessableEntity)
	}
	var env errorEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if env.Error.Fields["name"] == "" {
		t.Errorf("expected an inline error on the name field, got %+v", env.Error.Fields)
	}
}

func TestCreateCompany_DuplicateNameGetsASuffixedSlug(t *testing.T) {
	srv, _ := newTestServer(t)

	first := createCompany(t, srv, "Vera Accounting")
	second := createCompany(t, srv, "Vera Accounting")

	if first.Slug != "vera-accounting" {
		t.Errorf("first slug = %q, want %q", first.Slug, "vera-accounting")
	}
	if second.Slug != "vera-accounting-2" {
		t.Errorf("second slug = %q, want %q", second.Slug, "vera-accounting-2")
	}
}

func TestListCompanies(t *testing.T) {
	srv, _ := newTestServer(t)

	createCompany(t, srv, "Bem-Estar Clinic")
	createCompany(t, srv, "Aurora Logistics")

	resp, err := http.Get(srv.URL + "/companies")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var env listEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(env.Data) != 2 {
		t.Fatalf("got %d companies, want 2", len(env.Data))
	}
	if env.Data[0].Name != "Aurora Logistics" || env.Data[1].Name != "Bem-Estar Clinic" {
		t.Errorf("expected alphabetical order, got %q then %q", env.Data[0].Name, env.Data[1].Name)
	}
	if env.Meta.Pagination.Total != 2 {
		t.Errorf("total = %d, want 2", env.Meta.Pagination.Total)
	}
	if env.Meta.Pagination.Page != 1 || env.Meta.Pagination.TotalPages != 1 {
		t.Errorf("pagination = %+v, want page 1 of 1", env.Meta.Pagination)
	}
}

func TestRenameCompany(t *testing.T) {
	srv, _ := newTestServer(t)
	company := createCompany(t, srv, "Old Name")

	body, _ := json.Marshal(map[string]string{"name": "New Name"})
	req, _ := http.NewRequest(http.MethodPatch, srv.URL+"/companies/"+company.ID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("rename: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var env companyEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if env.Data.Name != "New Name" || env.Data.Slug != "new-name" {
		t.Errorf("got name=%q slug=%q, want name=%q slug=%q", env.Data.Name, env.Data.Slug, "New Name", "new-name")
	}
}

func TestRenameCompany_NotFound(t *testing.T) {
	srv, _ := newTestServer(t)

	body, _ := json.Marshal(map[string]string{"name": "New Name"})
	req, _ := http.NewRequest(http.MethodPatch, srv.URL+"/companies/00000000-0000-0000-0000-000000000000", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("rename: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestDeleteCompany(t *testing.T) {
	srv, _ := newTestServer(t)
	company := createCompany(t, srv, "To Delete")

	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/companies/"+company.ID, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusNoContent)
	}

	getResp, err := http.Get(srv.URL + "/companies/" + company.ID)
	if err != nil {
		t.Fatalf("get after delete: %v", err)
	}
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusNotFound {
		t.Fatalf("status after delete = %d, want %d", getResp.StatusCode, http.StatusNotFound)
	}
}

func TestDeleteCompany_CascadesToDocuments(t *testing.T) {
	srv, db := newTestServer(t)
	ctx := context.Background()

	company := createCompany(t, srv, "Has Documents")
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO documents (company_id, s3_key, title, kind, checksum)
		VALUES ($1, 'key', 'doc.md', 'markdown', 'checksum')
	`, company.ID)
	if err != nil {
		t.Fatalf("insert document: %v", err)
	}

	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/companies/"+company.ID, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusNoContent)
	}

	var count int
	if err := db.Pool.QueryRow(ctx, "SELECT count(*) FROM documents WHERE company_id = $1", company.ID).Scan(&count); err != nil {
		t.Fatalf("count documents: %v", err)
	}
	if count != 0 {
		t.Errorf("documents still present after company delete: %d", count)
	}
}
