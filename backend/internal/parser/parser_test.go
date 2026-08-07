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

func TestParse_UnregisteredKind(t *testing.T) {
	_, err := parser.Parse(domain.DocumentKindCSV, []byte("a,b\n1,2"))
	if err == nil || !strings.Contains(err.Error(), "csv") {
		t.Fatalf("expected an error naming the unregistered kind, got %v", err)
	}
}
