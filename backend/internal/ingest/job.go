package ingest

import (
	"context"
	"fmt"

	"github.com/discovery/backend/internal/domain"
	"github.com/riverqueue/river"
)

// ProcessDocumentWorker runs Ingestor.Process for one Document per River
// job. River's own delivery is the claim (ADR-0003) — there is no SQL
// claim step here, unlike the poll loop this replaces.
type ProcessDocumentWorker struct {
	river.WorkerDefaults[domain.ProcessDocumentArgs]
	Ingestor *Ingestor
}

// Work loads the Document fresh and runs it through Process, unless it
// has already reached a terminal status. That check is what makes a
// duplicate delivery — a River retry, or a redelivery after a crash
// mid-Process — a no-op instead of a re-run: it returns nil rather than
// re-processing or erroring, so River records the job as succeeded.
func (w *ProcessDocumentWorker) Work(ctx context.Context, job *river.Job[domain.ProcessDocumentArgs]) error {
	doc, err := w.Ingestor.Documents.GetDocumentByID(ctx, job.Args.DocumentID)
	if err != nil {
		return fmt.Errorf("ingest: load document %s: %w", job.Args.DocumentID, err)
	}
	if doc.Status != domain.DocumentStatusParsing {
		return nil
	}
	_, err = w.Ingestor.Process(ctx, doc)
	return err
}
