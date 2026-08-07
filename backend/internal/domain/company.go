// Package domain holds the entity types and validation rules shared by the
// store and http layers.
package domain

import (
	"strings"
	"time"
)

const maxCompanyNameLength = 200

// Company is the aggregate root — the owner of every Document, Discovery
// Run and Node in the system.
type Company struct {
	ID            string
	Name          string
	Slug          string
	DocumentCount int
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// RenderCompanyContext concatenates a Company's context entries into the
// single block of text that goes into every prompt (ADR-0001). Order
// follows entries, which the store keeps in first-write order — a
// re-summarise replaces an entry in place rather than moving it to the end.
func RenderCompanyContext(entries []CompanyContextEntry) string {
	texts := make([]string, len(entries))
	for i, e := range entries {
		texts[i] = e.Text
	}
	return strings.Join(texts, "\n\n---\n\n")
}

// ValidateCompanyName trims the name and checks it against the rules the
// frontend surfaces inline, next to the field.
func ValidateCompanyName(name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return &ValidationError{Fields: map[string]string{"name": "is required"}}
	}
	if len(trimmed) > maxCompanyNameLength {
		return &ValidationError{Fields: map[string]string{"name": "must be 200 characters or fewer"}}
	}
	return nil
}
