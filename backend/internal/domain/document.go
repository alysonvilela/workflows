package domain

import (
	"path/filepath"
	"strings"
	"time"
)

const (
	DocumentKindMarkdown = "markdown"
	DocumentKindCSV      = "csv"

	DocumentStatusParsing = "parsing"
	DocumentStatusReady   = "ready"
	DocumentStatusFailed  = "failed"
)

// Document is a file a Company supplied, on its way to becoming an entry in
// the Company Context.
type Document struct {
	ID        string
	CompanyID string
	S3Key     string
	Title     string
	Kind      string
	RawText   *string
	Status    string
	Error     *string
	Checksum  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// DetectDocumentKind maps a filename's extension to a supported Document
// Kind. Only Markdown has a Parse Strategy today; csv arrives in a later
// ticket, so its extension is deliberately not recognised yet.
func DetectDocumentKind(filename string) (kind string, ok bool) {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".md", ".markdown":
		return DocumentKindMarkdown, true
	default:
		return "", false
	}
}
