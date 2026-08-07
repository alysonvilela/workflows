package store

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/discovery/backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// maxSlugAttempts bounds the retry loop that appends -2, -3, ... to a slug
// already taken by another Company. 25 collisions on the same name is not a
// case worth handling gracefully — it is a caller doing something odd.
const maxSlugAttempts = 25

func (s *Store) CreateCompany(ctx context.Context, name string) (domain.Company, error) {
	if err := domain.ValidateCompanyName(name); err != nil {
		return domain.Company{}, err
	}
	trimmed := strings.TrimSpace(name)
	base := domain.Slugify(trimmed)

	company, err := withUniqueSlug(base, func(slug string) (domain.Company, error) {
		var c domain.Company
		err := s.Pool.QueryRow(ctx, `
			INSERT INTO companies (name, slug)
			VALUES ($1, $2)
			RETURNING id, name, slug, created_at, updated_at
		`, trimmed, slug).Scan(&c.ID, &c.Name, &c.Slug, &c.CreatedAt, &c.UpdatedAt)
		return c, err
	})
	if err != nil {
		return domain.Company{}, fmt.Errorf("store: create company: %w", err)
	}
	return company, nil
}

func (s *Store) ListCompanies(ctx context.Context, page, perPage int) ([]domain.Company, int, error) {
	var total int
	if err := s.Pool.QueryRow(ctx, `SELECT count(*) FROM companies`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("store: count companies: %w", err)
	}

	rows, err := s.Pool.Query(ctx, `
		SELECT c.id, c.name, c.slug, c.created_at, c.updated_at, count(d.id)
		FROM companies c
		LEFT JOIN documents d ON d.company_id = c.id
		GROUP BY c.id
		ORDER BY c.name ASC
		LIMIT $1 OFFSET $2
	`, perPage, (page-1)*perPage)
	if err != nil {
		return nil, 0, fmt.Errorf("store: list companies: %w", err)
	}
	defer rows.Close()

	companies := []domain.Company{}
	for rows.Next() {
		var c domain.Company
		if err := rows.Scan(&c.ID, &c.Name, &c.Slug, &c.CreatedAt, &c.UpdatedAt, &c.DocumentCount); err != nil {
			return nil, 0, fmt.Errorf("store: scan company: %w", err)
		}
		companies = append(companies, c)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("store: list companies: %w", err)
	}
	return companies, total, nil
}

func (s *Store) GetCompany(ctx context.Context, id string) (domain.Company, error) {
	var c domain.Company
	err := s.Pool.QueryRow(ctx, `
		SELECT c.id, c.name, c.slug, c.created_at, c.updated_at, count(d.id)
		FROM companies c
		LEFT JOIN documents d ON d.company_id = c.id
		WHERE c.id = $1
		GROUP BY c.id
	`, id).Scan(&c.ID, &c.Name, &c.Slug, &c.CreatedAt, &c.UpdatedAt, &c.DocumentCount)
	if isNotFoundError(err) {
		return domain.Company{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Company{}, fmt.Errorf("store: get company: %w", err)
	}
	return c, nil
}

func (s *Store) RenameCompany(ctx context.Context, id, name string) (domain.Company, error) {
	if err := domain.ValidateCompanyName(name); err != nil {
		return domain.Company{}, err
	}
	trimmed := strings.TrimSpace(name)
	base := domain.Slugify(trimmed)

	company, err := withUniqueSlug(base, func(slug string) (domain.Company, error) {
		var c domain.Company
		err := s.Pool.QueryRow(ctx, `
			UPDATE companies
			SET name = $2, slug = $3, updated_at = now()
			WHERE id = $1
			RETURNING id, name, slug, created_at, updated_at
		`, id, trimmed, slug).Scan(&c.ID, &c.Name, &c.Slug, &c.CreatedAt, &c.UpdatedAt)
		return c, err
	})
	if isNotFoundError(err) {
		return domain.Company{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Company{}, fmt.Errorf("store: rename company: %w", err)
	}

	count, err := s.documentCount(ctx, id)
	if err != nil {
		return domain.Company{}, err
	}
	company.DocumentCount = count
	return company, nil
}

// GetCompanyContext reads a Company's company_context and renders it as the
// text a consultant — and every prompt — reads (ADR-0001). No row lock: a
// read racing an in-flight append just sees the state before or after it,
// which is fine for a read that isn't itself the write.
func (s *Store) GetCompanyContext(ctx context.Context, id string) (string, error) {
	var raw []byte
	err := s.Pool.QueryRow(ctx, `SELECT company_context FROM companies WHERE id = $1`, id).Scan(&raw)
	if isNotFoundError(err) {
		return "", domain.ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("store: get company context: %w", err)
	}
	entries, err := decodeContextEntries(raw)
	if err != nil {
		return "", err
	}
	return domain.RenderCompanyContext(entries), nil
}

func (s *Store) DeleteCompany(ctx context.Context, id string) error {
	tag, err := s.Pool.Exec(ctx, `DELETE FROM companies WHERE id = $1`, id)
	if isNotFoundError(err) {
		return domain.ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("store: delete company: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *Store) documentCount(ctx context.Context, companyID string) (int, error) {
	var n int
	if err := s.Pool.QueryRow(ctx, `SELECT count(*) FROM documents WHERE company_id = $1`, companyID).Scan(&n); err != nil {
		return 0, fmt.Errorf("store: count documents: %w", err)
	}
	return n, nil
}

// withUniqueSlug retries run with candidateSlug(base, attempt) each time it
// fails on a slug uniqueness collision, up to maxSlugAttempts. Any other
// error — including a not-found from an UPDATE — is returned unwrapped, so
// the caller can still tell it apart from a real collision.
func withUniqueSlug(base string, run func(slug string) (domain.Company, error)) (domain.Company, error) {
	for attempt := 0; attempt < maxSlugAttempts; attempt++ {
		c, err := run(candidateSlug(base, attempt))
		if err == nil {
			return c, nil
		}
		if isUniqueSlugViolation(err) {
			continue
		}
		return domain.Company{}, err
	}
	return domain.Company{}, fmt.Errorf("could not derive a unique slug for %q after %d attempts", base, maxSlugAttempts)
}

func candidateSlug(base string, attempt int) string {
	if attempt == 0 {
		return base
	}
	return base + "-" + strconv.Itoa(attempt+1)
}

func isUniqueSlugViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// isNotFoundError covers both "no row matched" and a malformed uuid in the
// query parameter — from the caller's perspective both mean the id does not
// resolve to a Company.
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return true
	}
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "22P02"
}
