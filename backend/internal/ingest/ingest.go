// Package ingest is the seam described in the spec: upload → object storage
// → Parse Strategy → summariser → raw_text and summary on the Document.
//
// Stage and Process are deliberately split. Stage is what an HTTP upload
// calls: validate, dedupe, store the bytes, create the Document row —
// fast, no LLM call on the request path. Process is what Worker calls in
// the background: Parse Strategy, then the summariser, then ready/failed.
// A later ticket's ask_document Gate answers through Stage the same way
// an upload does.
package ingest

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"

	"github.com/discovery/backend/internal/domain"
	"github.com/discovery/backend/internal/parser"
)

// DocumentStore is the persistence seam Ingestor depends on.
type DocumentStore interface {
	FindDocumentByChecksum(ctx context.Context, companyID, checksum string) (domain.Document, error)
	CreateDocumentIfNew(ctx context.Context, companyID, s3Key, title, kind, checksum string) (doc domain.Document, created bool, err error)
	ClaimPendingDocuments(ctx context.Context, limit int) ([]domain.Document, error)
	MarkDocumentReady(ctx context.Context, id, rawText string, summary domain.DocumentSummary) (domain.Document, error)
	MarkDocumentFailed(ctx context.Context, id, reason string) (domain.Document, error)
}

// ObjectStore is the object storage seam Ingestor depends on.
type ObjectStore interface {
	Put(ctx context.Context, key string, body []byte, contentType string) error
	Get(ctx context.Context, key string) ([]byte, error)
	Delete(ctx context.Context, key string) error
}

// Summariser is the LLM seam Ingestor depends on — Provider.Complete one
// level up (see internal/llm), already narrowed to raw_text-in,
// summary-out.
type Summariser interface {
	Summarise(ctx context.Context, rawText string) (domain.DocumentSummary, error)
}

type Ingestor struct {
	Documents         DocumentStore
	Objects           ObjectStore
	Summariser        Summariser
	MaxUploadBytes    int64
	DeleteAfterIngest bool
}

// Stage validates kind and size, dedupes by checksum, stores the original
// in object storage, and creates the Document row in status 'parsing'. It
// does not parse or summarise — that is Process's job, run later by
// Worker — so this returns as soon as the bytes are safely stored,
// regardless of how long the LLM call behind the summariser would take.
// A returned error means the file was rejected outright (unsupported
// type, too large) — the caller reports it per-file rather than failing
// the whole request.
func (in *Ingestor) Stage(ctx context.Context, companyID, filename string, data []byte) (domain.Document, error) {
	kind, ok := domain.DetectDocumentKind(filename)
	if !ok {
		return domain.Document{}, fmt.Errorf("%q is not a supported file type — only .md, .markdown and .csv are accepted", filename)
	}
	if int64(len(data)) > in.MaxUploadBytes {
		return domain.Document{}, fmt.Errorf("%q exceeds the %d byte upload limit", filename, in.MaxUploadBytes)
	}

	sum := sha256.Sum256(data)
	checksum := hex.EncodeToString(sum[:])

	if existing, err := in.Documents.FindDocumentByChecksum(ctx, companyID, checksum); err == nil {
		return existing, nil
	} else if !errors.Is(err, domain.ErrNotFound) {
		return domain.Document{}, fmt.Errorf("ingest: lookup checksum for %s: %w", filename, err)
	}

	s3Key := fmt.Sprintf("companies/%s/documents/%s/%s", companyID, checksum, filename)
	if err := in.Objects.Put(ctx, s3Key, data, contentType(kind)); err != nil {
		return domain.Document{}, fmt.Errorf("ingest: store %s: %w", filename, err)
	}

	doc, _, err := in.Documents.CreateDocumentIfNew(ctx, companyID, s3Key, filename, kind, checksum)
	if err != nil {
		return domain.Document{}, fmt.Errorf("ingest: create document row for %s: %w", filename, err)
	}
	// created=false means a concurrent upload of the same file already won
	// the race — the object just written is a harmless duplicate of the one
	// that will be kept, and doc is that existing row, staged or further.
	return doc, nil
}

// Process runs the rest of the pipeline for one already-staged Document:
// fetch its bytes back from object storage, run its Parse Strategy, run
// the summariser, and record the result. Called by Worker once it has
// claimed doc — never by the HTTP path directly. A Parse or summariser
// failure is not a Go error return: it is a Document left in status
// 'failed' with its Error field set, same as Stage's synchronous
// rejections are for the caller to report per-file.
func (in *Ingestor) Process(ctx context.Context, doc domain.Document) (domain.Document, error) {
	data, err := in.Objects.Get(ctx, doc.S3Key)
	if err != nil {
		return domain.Document{}, fmt.Errorf("ingest: fetch staged %s: %w", doc.Title, err)
	}

	rawText, parseErr := parser.Parse(doc.Kind, data)
	if parseErr != nil {
		failed, err := in.Documents.MarkDocumentFailed(ctx, doc.ID, parseErr.Error())
		if err != nil {
			return domain.Document{}, fmt.Errorf("ingest: mark %s failed: %w", doc.Title, err)
		}
		return failed, nil
	}

	summary, sumErr := in.Summariser.Summarise(ctx, rawText)
	if sumErr != nil {
		failed, err := in.Documents.MarkDocumentFailed(ctx, doc.ID, sumErr.Error())
		if err != nil {
			return domain.Document{}, fmt.Errorf("ingest: mark %s failed: %w", doc.Title, err)
		}
		return failed, nil
	}

	ready, err := in.Documents.MarkDocumentReady(ctx, doc.ID, rawText, summary)
	if err != nil {
		return domain.Document{}, fmt.Errorf("ingest: mark %s ready: %w", doc.Title, err)
	}

	if in.DeleteAfterIngest {
		if err := in.Objects.Delete(ctx, doc.S3Key); err != nil {
			slog.Warn("ingest: could not delete object after ingest", "key", doc.S3Key, "error", err)
		}
	}

	return ready, nil
}

func contentType(kind string) string {
	switch kind {
	case domain.DocumentKindMarkdown:
		return "text/markdown; charset=utf-8"
	case domain.DocumentKindCSV:
		return "text/csv; charset=utf-8"
	default:
		return "application/octet-stream"
	}
}
