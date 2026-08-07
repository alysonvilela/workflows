package ingest_test

import (
	"context"
	"testing"

	"github.com/discovery/backend/internal/domain"
	"github.com/discovery/backend/internal/ingest"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

// fakeDocumentStore is the minimal DocumentStore double these tests need:
// a single in-memory Document that GetDocumentByID reads and
// MarkDocumentReady/MarkDocumentFailed mutate, so a second Work() call
// observes whatever the first one left behind — same as a real duplicate
// River delivery reading the row back from Postgres.
type fakeDocumentStore struct {
	doc domain.Document
}

func (f *fakeDocumentStore) FindDocumentByChecksum(ctx context.Context, companyID, checksum string) (domain.Document, error) {
	panic("not used by this test")
}

func (f *fakeDocumentStore) CreateDocumentIfNew(ctx context.Context, companyID, s3Key, title, kind, checksum string) (domain.Document, bool, error) {
	panic("not used by this test")
}

func (f *fakeDocumentStore) GetDocumentByID(ctx context.Context, id string) (domain.Document, error) {
	return f.doc, nil
}

func (f *fakeDocumentStore) MarkDocumentReady(ctx context.Context, doc domain.Document, rawText string, summary domain.DocumentSummary) (domain.Document, error) {
	f.doc.Status = domain.DocumentStatusReady
	f.doc.RawText = &rawText
	f.doc.Summary = &summary
	return f.doc, nil
}

func (f *fakeDocumentStore) MarkDocumentFailed(ctx context.Context, id, reason string) (domain.Document, error) {
	f.doc.Status = domain.DocumentStatusFailed
	f.doc.Error = &reason
	return f.doc, nil
}

// countingObjectStore records how many times Get is called — a duplicate
// delivery over an already-terminal Document must never re-fetch.
type countingObjectStore struct {
	content  []byte
	getCalls int
}

func (o *countingObjectStore) Put(ctx context.Context, key string, body []byte, contentType string) error {
	panic("not used by this test")
}

func (o *countingObjectStore) Get(ctx context.Context, key string) ([]byte, error) {
	o.getCalls++
	return o.content, nil
}

func (o *countingObjectStore) Delete(ctx context.Context, key string) error { return nil }

// countingSummariser records how many times Summarise is called — a
// duplicate delivery must never re-run the LLM call.
type countingSummariser struct {
	calls int
}

func (s *countingSummariser) Summarise(ctx context.Context, rawText string) (domain.DocumentSummary, error) {
	s.calls++
	return domain.DocumentSummary{Theme: "t", Context: "c"}, nil
}

// unreachableObjectStore and unreachableSummariser panic if touched,
// proving Work() never reaches Process for an already-terminal Document.
type unreachableObjectStore struct{}

func (unreachableObjectStore) Put(ctx context.Context, key string, body []byte, contentType string) error {
	panic("not used by this test")
}

func (unreachableObjectStore) Get(ctx context.Context, key string) ([]byte, error) {
	panic("ingest: Work must not touch object storage for an already-terminal Document")
}

func (unreachableObjectStore) Delete(ctx context.Context, key string) error {
	panic("not used by this test")
}

type unreachableSummariser struct{}

func (unreachableSummariser) Summarise(ctx context.Context, rawText string) (domain.DocumentSummary, error) {
	panic("ingest: Work must not call the summariser for an already-terminal Document")
}

func newProcessDocumentJob(documentID string) *river.Job[domain.ProcessDocumentArgs] {
	return &river.Job[domain.ProcessDocumentArgs]{
		JobRow: &rivertype.JobRow{ID: 1, Kind: domain.ProcessDocumentKind},
		Args:   domain.ProcessDocumentArgs{DocumentID: documentID},
	}
}

// TestProcessDocumentWorker_DuplicateDeliveryOnTerminalDocumentIsANoOp is
// ticket 16's explicit test requirement: a duplicate delivery over a
// Document already ready/failed returns nil, not an error, and never
// re-runs Process.
func TestProcessDocumentWorker_DuplicateDeliveryOnTerminalDocumentIsANoOp(t *testing.T) {
	for _, status := range []string{domain.DocumentStatusReady, domain.DocumentStatusFailed} {
		t.Run(status, func(t *testing.T) {
			store := &fakeDocumentStore{doc: domain.Document{ID: "doc-1", Status: status}}
			worker := &ingest.ProcessDocumentWorker{
				Ingestor: &ingest.Ingestor{
					Documents:  store,
					Objects:    unreachableObjectStore{},
					Summariser: unreachableSummariser{},
				},
			}

			if err := worker.Work(context.Background(), newProcessDocumentJob("doc-1")); err != nil {
				t.Fatalf("Work() on a %s Document returned an error, want nil: %v", status, err)
			}
		})
	}
}

// TestProcessDocumentWorker_ExecutedTwiceProcessesOnlyOnceAndReportsSuccess
// is the fuller version of the same guarantee: the same job body run
// twice — a River retry, or a redelivery after a crash right after the
// first Work() committed — processes the Document exactly once and both
// calls report success.
func TestProcessDocumentWorker_ExecutedTwiceProcessesOnlyOnceAndReportsSuccess(t *testing.T) {
	objects := &countingObjectStore{content: []byte("# Title\n\nBody.")}
	summariser := &countingSummariser{}
	store := &fakeDocumentStore{doc: domain.Document{
		ID:     "doc-2",
		S3Key:  "key",
		Kind:   domain.DocumentKindMarkdown,
		Status: domain.DocumentStatusParsing,
	}}
	worker := &ingest.ProcessDocumentWorker{
		Ingestor: &ingest.Ingestor{
			Documents:  store,
			Objects:    objects,
			Summariser: summariser,
		},
	}
	job := newProcessDocumentJob("doc-2")

	if err := worker.Work(context.Background(), job); err != nil {
		t.Fatalf("first Work(): %v", err)
	}
	if store.doc.Status != domain.DocumentStatusReady {
		t.Fatalf("status after first Work() = %q, want %q", store.doc.Status, domain.DocumentStatusReady)
	}

	if err := worker.Work(context.Background(), job); err != nil {
		t.Fatalf("second Work() (duplicate delivery) returned an error, want nil: %v", err)
	}

	if objects.getCalls != 1 {
		t.Errorf("object storage Get called %d times, want 1", objects.getCalls)
	}
	if summariser.calls != 1 {
		t.Errorf("summariser called %d times, want 1", summariser.calls)
	}
}
