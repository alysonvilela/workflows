package store

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/discovery/backend/internal/domain"
	"github.com/jackc/pgx/v5"
)

func decodeContextEntries(raw []byte) ([]domain.CompanyContextEntry, error) {
	entries := []domain.CompanyContextEntry{}
	if len(raw) == 0 {
		return entries, nil
	}
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil, fmt.Errorf("store: decode company context: %w", err)
	}
	return entries, nil
}

func encodeContextEntries(entries []domain.CompanyContextEntry) ([]byte, error) {
	raw, err := json.Marshal(entries)
	if err != nil {
		return nil, fmt.Errorf("store: encode company context: %w", err)
	}
	return raw, nil
}

// upsertContextEntry replaces the entry for documentID in place if one
// exists, or appends a new one otherwise — the single code path that
// serves both a Document's first summary and every re-summarise after it.
func upsertContextEntry(entries []domain.CompanyContextEntry, documentID, text string) []domain.CompanyContextEntry {
	for i, e := range entries {
		if e.DocumentID == documentID {
			entries[i].Text = text
			return entries
		}
	}
	return append(entries, domain.CompanyContextEntry{DocumentID: documentID, Text: text})
}

// removeContextEntry drops the entry for documentID, if any.
func removeContextEntry(entries []domain.CompanyContextEntry, documentID string) []domain.CompanyContextEntry {
	out := entries[:0]
	for _, e := range entries {
		if e.DocumentID != documentID {
			out = append(out, e)
		}
	}
	return out
}

// lockAndMutateCompanyContext locks the Company row with SELECT ... FOR
// UPDATE, decodes company_context, applies mutate, and writes the result
// back — the row-lock discipline ADR-0003 calls correct-by-SQL, shared by
// every caller that appends to or removes from a Company's context as part
// of a larger transaction (MarkDocumentReady, DeleteDocument).
func lockAndMutateCompanyContext(ctx context.Context, tx pgx.Tx, companyID string, mutate func([]domain.CompanyContextEntry) []domain.CompanyContextEntry) error {
	var contextRaw []byte
	err := tx.QueryRow(ctx, `SELECT company_context FROM companies WHERE id = $1 FOR UPDATE`, companyID).Scan(&contextRaw)
	if isNotFoundError(err) {
		return domain.ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("store: lock company context: %w", err)
	}

	entries, err := decodeContextEntries(contextRaw)
	if err != nil {
		return err
	}
	newContextRaw, err := encodeContextEntries(mutate(entries))
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE companies SET company_context = $2, updated_at = now() WHERE id = $1`, companyID, newContextRaw); err != nil {
		return fmt.Errorf("store: update company context: %w", err)
	}
	return nil
}
