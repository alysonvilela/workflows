// Package parser holds the Parse Strategies that turn a Document's raw
// bytes into raw_text. One file per format; Parse is the registry.
package parser

import (
	"fmt"
	"unicode/utf8"

	"github.com/discovery/backend/internal/domain"
)

// Parse runs the Parse Strategy registered for kind against data. Only
// Markdown is registered today; csv arrives in a later ticket.
func Parse(kind string, data []byte) (string, error) {
	switch kind {
	case domain.DocumentKindMarkdown:
		return parseMarkdown(data)
	default:
		return "", fmt.Errorf("parser: no Parse Strategy registered for kind %q", kind)
	}
}

// parseMarkdown decodes data as UTF-8 text. Markdown source is already the
// text the summariser will read later — there is no syntax to strip.
func parseMarkdown(data []byte) (string, error) {
	if !utf8.Valid(data) {
		return "", fmt.Errorf("file is not valid UTF-8 text")
	}
	return string(data), nil
}
