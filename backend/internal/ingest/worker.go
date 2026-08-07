package ingest

import (
	"context"
	"log/slog"
	"time"

	"golang.org/x/sync/errgroup"
)

// Worker repeatedly claims staged Documents and runs them through Process,
// so an HTTP upload never blocks on a Parse Strategy or an LLM call —
// Stage returns as soon as the bytes are stored, and the frontend watches
// the parsing -> ready/failed transition by polling.
type Worker struct {
	Ingestor     *Ingestor
	PollInterval time.Duration
	BatchSize    int
}

// Run polls until ctx is cancelled. The claim query (SKIP LOCKED, see
// store.ClaimPendingDocuments) is the correctness guard, not anything
// held here — nothing about which Document belongs to which tick is kept
// in memory across a poll (ADR-0003).
//
// A batch already claimed runs to completion with its own background
// context, not ctx — so a shutdown signal stops new batches from being
// claimed without cutting off an LLM call already in flight. Run itself
// only returns once the in-flight batch's goroutines have all finished,
// so a caller that waits for Run to return gets a proper drain.
func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.PollInterval)
	defer ticker.Stop()

	for {
		w.tick(context.Background())
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (w *Worker) tick(ctx context.Context) {
	docs, err := w.Ingestor.Documents.ClaimPendingDocuments(ctx, w.BatchSize)
	if err != nil {
		slog.Error("ingest worker: claim pending documents", "error", err)
		return
	}
	if len(docs) == 0 {
		return
	}

	var g errgroup.Group
	for _, doc := range docs {
		g.Go(func() error {
			if _, err := w.Ingestor.Process(ctx, doc); err != nil {
				slog.Error("ingest worker: process document", "document_id", doc.ID, "title", doc.Title, "error", err)
			}
			return nil
		})
	}
	_ = g.Wait()
}
