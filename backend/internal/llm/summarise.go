package llm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/discovery/backend/internal/domain"
)

// SummariseToolName is the single tool the summariser offers, and the
// model is forced to call — see ADR-0002.
const SummariseToolName = "record_summary"

var summariseTool = Tool{
	Name:        SummariseToolName,
	Description: "Record a structured summary of the supplied document text.",
	Parameters: json.RawMessage(`{
		"type": "object",
		"properties": {
			"theme": {"type": "string", "description": "One line naming what this document is about."},
			"context": {"type": "string", "description": "A short paragraph of background a reader would need."},
			"topics": {"type": "array", "items": {"type": "string"}, "description": "Subjects the document covers."},
			"pains": {"type": "array", "items": {"type": "string"}, "description": "Problems or friction described or implied."},
			"examples": {"type": "array", "items": {"type": "string"}, "description": "Concrete examples, numbers, or cases mentioned."},
			"opportunities": {"type": "array", "items": {"type": "string"}, "description": "Places software could plausibly help this process."}
		},
		"required": ["theme", "context", "topics", "pains", "examples", "opportunities"]
	}`),
}

// Summariser turns a Document's raw_text into the structured shape that
// later gets appended to Company Context (ADR-0001) — after this call, the
// LLM never reads raw_text directly again.
type Summariser struct {
	Provider Provider
	Model    string
}

// Summarise runs the one LLM call that produces a Document's summary. The
// returned error is the Provider's error, close to verbatim — Ingest
// records it on the Document rather than swallowing it.
func (s *Summariser) Summarise(ctx context.Context, rawText string) (domain.DocumentSummary, error) {
	call, err := s.Provider.Complete(ctx, s.Model, summarisePrompt(rawText), []Tool{summariseTool})
	if err != nil {
		return domain.DocumentSummary{}, fmt.Errorf("llm: summarise: %w", err)
	}
	if call.Name != SummariseToolName {
		return domain.DocumentSummary{}, fmt.Errorf("llm: summarise: model called %q, expected %q", call.Name, SummariseToolName)
	}

	var summary domain.DocumentSummary
	if err := json.Unmarshal(call.Arguments, &summary); err != nil {
		return domain.DocumentSummary{}, fmt.Errorf("llm: summarise: decode tool arguments: %w", err)
	}
	return summary, nil
}

func summarisePrompt(rawText string) string {
	return "Summarise the following document for a consultant looking for software-improvement " +
		"opportunities in the company that wrote it. Call " + SummariseToolName + " with your answer.\n\n---\n\n" + rawText
}
