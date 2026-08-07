package store

import (
	"context"
	"fmt"

	"github.com/discovery/backend/internal/domain"
	"github.com/jackc/pgx/v5"
)

const documentColumns = `id, company_id, s3_key, title, kind, raw_text, status, error, checksum, created_at, updated_at`

func scanDocument(row pgx.Row) (domain.Document, error) {
	var d domain.Document
	err := row.Scan(&d.ID, &d.CompanyID, &d.S3Key, &d.Title, &d.Kind, &d.RawText, &d.Status, &d.Error, &d.Checksum, &d.CreatedAt, &d.UpdatedAt)
	return d, err
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

// CreateDocumentIfNew inserts a new Document in status 'parsing', unless one
// already exists for this (company_id, checksum) — in which case that
// existing row is returned and created is false. This is the mechanism that
// keeps the same file uploaded twice from being reprocessed.
func (s *Store) CreateDocumentIfNew(ctx context.Context, companyID, s3Key, title, kind, checksum string) (doc domain.Document, created bool, err error) {
	row := s.Pool.QueryRow(ctx, `
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
	return d, true, nil
}

func (s *Store) MarkDocumentReady(ctx context.Context, id, rawText string) (domain.Document, error) {
	row := s.Pool.QueryRow(ctx, `
		UPDATE documents
		SET status = 'ready', raw_text = $2, error = NULL, updated_at = now()
		WHERE id = $1
		RETURNING `+documentColumns, id, rawText)
	d, err := scanDocument(row)
	if isNotFoundError(err) {
		return domain.Document{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Document{}, fmt.Errorf("store: mark document ready: %w", err)
	}
	return d, nil
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
