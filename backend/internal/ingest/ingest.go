// Package ingest is the seam described in the spec: upload → object storage
// → Parse Strategy → raw_text. It is reached from an HTTP upload today and,
// from a later ticket, from answering an ask_document Gate — both call
// Ingest with the same bytes-in, Document-out shape.
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
	MarkDocumentReady(ctx context.Context, id, rawText string) (domain.Document, error)
	MarkDocumentFailed(ctx context.Context, id, reason string) (domain.Document, error)
}

// ObjectStore is the object storage seam Ingestor depends on.
type ObjectStore interface {
	Put(ctx context.Context, key string, body []byte, contentType string) error
	Delete(ctx context.Context, key string) error
}

type Ingestor struct {
	Documents         DocumentStore
	Objects           ObjectStore
	MaxUploadBytes    int64
	DeleteAfterIngest bool
}

// Ingest runs one file through the pipeline: validate kind and size,
// dedupe by checksum, store the original, run its Parse Strategy, and
// record the result on the Document. A returned error means the file was
// rejected outright (unsupported type, too large) — the caller reports it
// per-file rather than failing the whole request. A parsing failure, by
// contrast, is not an error return: it is a Document with status 'failed'
// and its Error field set.
func (in *Ingestor) Ingest(ctx context.Context, companyID, filename string, data []byte) (domain.Document, error) {
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

	doc, created, err := in.Documents.CreateDocumentIfNew(ctx, companyID, s3Key, filename, kind, checksum)
	if err != nil {
		return domain.Document{}, fmt.Errorf("ingest: create document row for %s: %w", filename, err)
	}
	if !created {
		// Lost the race to a concurrent upload of the same file — the object
		// just written is a harmless duplicate of the one that will be kept.
		return doc, nil
	}

	rawText, parseErr := parser.Parse(kind, data)
	if parseErr != nil {
		failed, err := in.Documents.MarkDocumentFailed(ctx, doc.ID, parseErr.Error())
		if err != nil {
			return domain.Document{}, fmt.Errorf("ingest: mark %s failed: %w", filename, err)
		}
		return failed, nil
	}

	ready, err := in.Documents.MarkDocumentReady(ctx, doc.ID, rawText)
	if err != nil {
		return domain.Document{}, fmt.Errorf("ingest: mark %s ready: %w", filename, err)
	}

	if in.DeleteAfterIngest {
		if err := in.Objects.Delete(ctx, s3Key); err != nil {
			slog.Warn("ingest: could not delete object after ingest", "key", s3Key, "error", err)
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
