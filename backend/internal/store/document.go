package store

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/discovery/backend/internal/domain"
	"github.com/jackc/pgx/v5"
)

const documentColumns = `id, company_id, s3_key, title, kind, raw_text, summary, status, error, checksum, created_at, updated_at`

func scanDocument(row pgx.Row) (domain.Document, error) {
	var d domain.Document
	var summaryJSON []byte
	if err := row.Scan(&d.ID, &d.CompanyID, &d.S3Key, &d.Title, &d.Kind, &d.RawText, &summaryJSON, &d.Status, &d.Error, &d.Checksum, &d.CreatedAt, &d.UpdatedAt); err != nil {
		return domain.Document{}, err
	}
	if len(summaryJSON) > 0 {
		var summary domain.DocumentSummary
		if err := json.Unmarshal(summaryJSON, &summary); err != nil {
			return domain.Document{}, fmt.Errorf("store: decode document summary: %w", err)
		}
		d.Summary = &summary
	}
	return d, nil
}

func (s *Store) FindDocumentByChecksum(ctx context.Context, companyID, checksum string) (domain.Document, error) {
	row := s.Pool.QueryRow(ctx, `
		SELECT `+documentColumns+`
		FROM documents
		WHERE company_id = $1 AND checksum = $2
	`, companyID, checksum)
	d, err := scanDocument(row)
	if isNotFoundError(err) {
		return domain.Document{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Document{}, fmt.Errorf("store: find document by checksum: %w", err)
	}
	return d, nil
}

// CreateDocumentIfNew inserts a new Document in status 'parsing' and
// enqueues its process_document River job in the same transaction —
// either both happen or neither (ticket 16) — unless a Document already
// exists for this (company_id, checksum), in which case that existing row
// is returned, created is false, and no job is enqueued (one already ran,
// or is running, for it). This is the mechanism that keeps the same file
// uploaded twice from being reprocessed.
func (s *Store) CreateDocumentIfNew(ctx context.Context, companyID, s3Key, title, kind, checksum string) (doc domain.Document, created bool, err error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return domain.Document{}, false, fmt.Errorf("store: create document: begin: %w", err)
	}
	defer tx.Rollback(ctx)

	row := tx.QueryRow(ctx, `
		INSERT INTO documents (company_id, s3_key, title, kind, checksum, status)
		VALUES ($1, $2, $3, $4, $5, 'parsing')
		ON CONFLICT (company_id, checksum) DO NOTHING
		RETURNING `+documentColumns, companyID, s3Key, title, kind, checksum)
	d, err := scanDocument(row)
	if isNotFoundError(err) {
		existing, findErr := s.FindDocumentByChecksum(ctx, companyID, checksum)
		if findErr != nil {
			return domain.Document{}, false, fmt.Errorf("store: create document: conflict but no existing row: %w", findErr)
		}
		return existing, false, nil
	}
	if err != nil {
		return domain.Document{}, false, fmt.Errorf("store: create document: %w", err)
	}

	if _, err := s.River.InsertTx(ctx, tx, domain.ProcessDocumentArgs{DocumentID: d.ID}, nil); err != nil {
		return domain.Document{}, false, fmt.Errorf("store: enqueue process document job: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Document{}, false, fmt.Errorf("store: create document: commit: %w", err)
	}
	return d, true, nil
}

// GetDocumentByID loads a Document by id alone, unscoped by company_id —
// unlike GetDocument, which the HTTP layer uses. A process_document job
// body only carries the Document's id (River job args stay minimal), so
// ProcessDocumentWorker.Work needs this to load the row it was told to
// process.
func (s *Store) GetDocumentByID(ctx context.Context, id string) (domain.Document, error) {
	row := s.Pool.QueryRow(ctx, `
		SELECT `+documentColumns+`
		FROM documents
		WHERE id = $1
	`, id)
	d, err := scanDocument(row)
	if isNotFoundError(err) {
		return domain.Document{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Document{}, fmt.Errorf("store: get document by id: %w", err)
	}
	return d, nil
}

// MarkDocumentReady marks doc ready with rawText and summary, and folds the
// summary into the owning Company's company_context in the same
// transaction — the append ADR-0003 calls correct-by-SQL: the Company row
// is locked with SELECT ... FOR UPDATE before the jsonb array is
// read-modified-written, so two Documents finishing at once serialise
// instead of one clobbering the other's entry. Called both for a
// Document's first summary and for every re-summarise after it — doc.ID
// already existing in the array means an in-place replace, not a
// duplicate (see upsertContextEntry).
func (s *Store) MarkDocumentReady(ctx context.Context, doc domain.Document, rawText string, summary domain.DocumentSummary) (domain.Document, error) {
	summaryJSON, err := json.Marshal(summary)
	if err != nil {
		return domain.Document{}, fmt.Errorf("store: encode document summary: %w", err)
	}
	entryText := domain.FormatCompanyContextEntry(doc.Title, summary)

	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return domain.Document{}, fmt.Errorf("store: mark document ready: begin: %w", err)
	}
	defer tx.Rollback(ctx)

	mutate := func(entries []domain.CompanyContextEntry) []domain.CompanyContextEntry {
		return upsertContextEntry(entries, doc.ID, entryText)
	}
	if err := lockAndMutateCompanyContext(ctx, tx, doc.CompanyID, mutate); err != nil {
		return domain.Document{}, err
	}

	row := tx.QueryRow(ctx, `
		UPDATE documents
		SET status = 'ready', raw_text = $2, summary = $3, error = NULL, updated_at = now()
		WHERE id = $1
		RETURNING `+documentColumns, doc.ID, rawText, summaryJSON)
	ready, err := scanDocument(row)
	if isNotFoundError(err) {
		return domain.Document{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Document{}, fmt.Errorf("store: mark document ready: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Document{}, fmt.Errorf("store: mark document ready: commit: %w", err)
	}
	return ready, nil
}

// DeleteDocument removes a Document and its Company Context entry in the
// same transaction, under the same row lock MarkDocumentReady appends
// under — so a delete can never race a concurrent ingestion into
// resurrecting the entry it just removed, or vice versa.
func (s *Store) DeleteDocument(ctx context.Context, companyID, id string) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("store: delete document: begin: %w", err)
	}
	defer tx.Rollback(ctx)

	mutate := func(entries []domain.CompanyContextEntry) []domain.CompanyContextEntry {
		return removeContextEntry(entries, id)
	}
	if err := lockAndMutateCompanyContext(ctx, tx, companyID, mutate); err != nil {
		return err
	}

	tag, err := tx.Exec(ctx, `DELETE FROM documents WHERE id = $1 AND company_id = $2`, id, companyID)
	if err != nil {
		return fmt.Errorf("store: delete document: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("store: delete document: commit: %w", err)
	}
	return nil
}

func (s *Store) MarkDocumentFailed(ctx context.Context, id, reason string) (domain.Document, error) {
	row := s.Pool.QueryRow(ctx, `
		UPDATE documents
		SET status = 'failed', error = $2, updated_at = now()
		WHERE id = $1
		RETURNING `+documentColumns, id, reason)
	d, err := scanDocument(row)
	if isNotFoundError(err) {
		return domain.Document{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Document{}, fmt.Errorf("store: mark document failed: %w", err)
	}
	return d, nil
}

// ListDocuments omits raw_text — list rows are polled while ingestion is in
// flight, and the text is reachable via GetDocument once a row is ready.
func (s *Store) ListDocuments(ctx context.Context, companyID string, page, perPage int) ([]domain.Document, int, error) {
	var total int
	if err := s.Pool.QueryRow(ctx, `SELECT count(*) FROM documents WHERE company_id = $1`, companyID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("store: count documents: %w", err)
	}

	rows, err := s.Pool.Query(ctx, `
		SELECT id, company_id, s3_key, title, kind, status, error, checksum, created_at, updated_at
		FROM documents
		WHERE company_id = $1
		ORDER BY created_at ASC
		LIMIT $2 OFFSET $3
	`, companyID, perPage, (page-1)*perPage)
	if err != nil {
		return nil, 0, fmt.Errorf("store: list documents: %w", err)
	}
	defer rows.Close()

	docs := []domain.Document{}
	for rows.Next() {
		var d domain.Document
		if err := rows.Scan(&d.ID, &d.CompanyID, &d.S3Key, &d.Title, &d.Kind, &d.Status, &d.Error, &d.Checksum, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("store: scan document: %w", err)
		}
		docs = append(docs, d)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("store: list documents: %w", err)
	}
	return docs, total, nil
}

func (s *Store) GetDocument(ctx context.Context, companyID, id string) (domain.Document, error) {
	row := s.Pool.QueryRow(ctx, `
		SELECT `+documentColumns+`
		FROM documents
		WHERE id = $1 AND company_id = $2
	`, id, companyID)
	d, err := scanDocument(row)
	if isNotFoundError(err) {
		return domain.Document{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Document{}, fmt.Errorf("store: get document: %w", err)
	}
	return d, nil
}
