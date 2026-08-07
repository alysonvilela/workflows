// Package parser holds the Parse Strategies that turn a Document's raw
// bytes into raw_text. One file per format; Parse is the registry.
package parser

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"unicode/utf8"

	"github.com/discovery/backend/internal/domain"
)

// Parse runs the Parse Strategy registered for kind against data.
func Parse(kind string, data []byte) (string, error) {
	switch kind {
	case domain.DocumentKindMarkdown:
		return parseMarkdown(data)
	case domain.DocumentKindCSV:
		return parseCSV(data)
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

// parseCSV validates data as CSV one record at a time — via csv.Reader.Read
// in a loop rather than ReadAll, so a malformed file is caught without ever
// materialising the whole table in memory — then, like parseMarkdown,
// returns the source unchanged: CSV text is already the text the summariser
// will read later, so there is nothing to transform. The shape is never
// assumed: no fixed header, no fixed column count, since the column count
// check is disabled by FieldsPerRecord = -1.
func parseCSV(data []byte) (string, error) {
	if !utf8.Valid(data) {
		return "", fmt.Errorf("file is not valid UTF-8 text")
	}

	reader := csv.NewReader(bytes.NewReader(data))
	reader.FieldsPerRecord = -1
	reader.ReuseRecord = true

	for {
		_, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", fmt.Errorf("malformed CSV: %w", err)
		}
	}

	return string(data), nil
}
