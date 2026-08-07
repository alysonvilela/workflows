package parser_test

import (
	"strings"
	"testing"

	"github.com/discovery/backend/internal/domain"
	"github.com/discovery/backend/internal/parser"
)

func TestParse_Markdown(t *testing.T) {
	rawText, err := parser.Parse(domain.DocumentKindMarkdown, []byte("# Sales process\n\nSteps..."))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if rawText != "# Sales process\n\nSteps..." {
		t.Errorf("rawText = %q, want file content unchanged", rawText)
	}
}

func TestParse_Markdown_InvalidUTF8(t *testing.T) {
	_, err := parser.Parse(domain.DocumentKindMarkdown, []byte{0xff, 0xfe, 0x00})
	if err == nil {
		t.Fatal("expected an error for invalid UTF-8 input")
	}
}

func TestParse_CSV(t *testing.T) {
	rawText, err := parser.Parse(domain.DocumentKindCSV, []byte("name,volume\nCold call,120"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if rawText != "name,volume\nCold call,120" {
		t.Errorf("rawText = %q, want file content unchanged", rawText)
	}
}

func TestParse_CSV_ArbitraryColumnCount(t *testing.T) {
	// No fixed header, no fixed column count — rows may carry a different
	// number of fields than the ones around them.
	rawText, err := parser.Parse(domain.DocumentKindCSV, []byte("a,b,c\n1,2\n3,4,5,6"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if rawText != "a,b,c\n1,2\n3,4,5,6" {
		t.Errorf("rawText = %q, want every row preserved regardless of column count", rawText)
	}
}

func TestParse_CSV_Empty(t *testing.T) {
	rawText, err := parser.Parse(domain.DocumentKindCSV, []byte(""))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if rawText != "" {
		t.Errorf("rawText = %q, want empty output for an empty file", rawText)
	}
}

func TestParse_CSV_Malformed(t *testing.T) {
	_, err := parser.Parse(domain.DocumentKindCSV, []byte(`a,"unterminated quote`+"\n"))
	if err == nil {
		t.Fatal("expected an error for malformed CSV")
	}
}

func TestParse_CSV_InvalidUTF8(t *testing.T) {
	_, err := parser.Parse(domain.DocumentKindCSV, []byte{0xff, 0xfe, 0x00})
	if err == nil {
		t.Fatal("expected an error for invalid UTF-8 input")
	}
}

func TestParse_UnregisteredKind(t *testing.T) {
	_, err := parser.Parse("pdf", []byte("%PDF-1.4"))
	if err == nil || !strings.Contains(err.Error(), "pdf") {
		t.Fatalf("expected an error naming the unregistered kind, got %v", err)
	}
}
