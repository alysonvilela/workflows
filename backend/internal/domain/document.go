package domain

import (
	"fmt"
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
	Summary   *DocumentSummary
	Status    string
	Error     *string
	Checksum  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// DocumentSummary is the summariser's structured reading of a Document's
// raw_text — the only form of the Document the LLM sees again after
// ingestion, since it is what gets appended to companies.company_context,
// never raw_text itself.
type DocumentSummary struct {
	Theme         string   `json:"theme"`
	Context       string   `json:"context"`
	Topics        []string `json:"topics"`
	Pains         []string `json:"pains"`
	Examples      []string `json:"examples"`
	Opportunities []string `json:"opportunities"`
}

// CompanyContextEntry is one Document's contribution to a Company's
// company_context — the jsonb array store.MarkDocumentReady rewrites under
// a row lock on the owning Company (ADR-0003). DocumentID is what lets a
// re-summarise replace an entry in place instead of duplicating it, and
// lets a Document removal find the right entry to drop.
type CompanyContextEntry struct {
	DocumentID string `json:"document_id"`
	Text       string `json:"text"`
}

// FormatCompanyContextEntry renders a Document's title and Summary as the
// plain text a consultant — and, per ADR-0001, every prompt — reads. It is
// the only place a Summary turns back into text after the summariser
// produced it.
func FormatCompanyContextEntry(title string, s DocumentSummary) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## %s\n\n%s\n\n%s\n", title, s.Theme, s.Context)
	writeBulletList(&b, "Topics", s.Topics)
	writeBulletList(&b, "Pains", s.Pains)
	writeBulletList(&b, "Examples", s.Examples)
	writeBulletList(&b, "Opportunities", s.Opportunities)
	return strings.TrimRight(b.String(), "\n")
}

func writeBulletList(b *strings.Builder, heading string, items []string) {
	if len(items) == 0 {
		return
	}
	fmt.Fprintf(b, "\n%s:\n", heading)
	for _, item := range items {
		fmt.Fprintf(b, "- %s\n", item)
	}
}

// DetectDocumentKind maps a filename's extension to a supported Document
// Kind. An unrecognised extension returns ok=false so the caller can fail
// the Document with a clear message.
func DetectDocumentKind(filename string) (kind string, ok bool) {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".md", ".markdown":
		return DocumentKindMarkdown, true
	case ".csv":
		return DocumentKindCSV, true
	default:
		return "", false
	}
}
