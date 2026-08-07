package llm_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/discovery/backend/internal/llm"
)

func TestSummariser_Summarise_ReturnsStructuredSummary(t *testing.T) {
	provider := llm.NewScriptedProvider()
	provider.Enqueue(llm.SummariseToolName, llm.ToolCall{
		Name: llm.SummariseToolName,
		Arguments: json.RawMessage(`{
			"theme": "Sales process",
			"context": "A B2B company describing its cold-call-to-close funnel.",
			"topics": ["cold calling", "demos"],
			"pains": ["no CRM, leads tracked in a spreadsheet"],
			"examples": ["50 cold calls a week"],
			"opportunities": ["automate lead capture"]
		}`),
	})
	s := &llm.Summariser{Provider: provider, Model: "gpt-4.1-mini"}

	summary, err := s.Summarise(context.Background(), "# Sales process\n\nCold call, then demo.")
	if err != nil {
		t.Fatalf("Summarise: %v", err)
	}
	if summary.Theme != "Sales process" {
		t.Errorf("Theme = %q, want %q", summary.Theme, "Sales process")
	}
	if len(summary.Topics) != 2 || summary.Topics[0] != "cold calling" {
		t.Errorf("Topics = %v, want [cold calling demos]", summary.Topics)
	}
	if len(summary.Pains) != 1 {
		t.Errorf("Pains = %v, want 1 entry", summary.Pains)
	}
	if len(summary.Opportunities) != 1 {
		t.Errorf("Opportunities = %v, want 1 entry", summary.Opportunities)
	}
}

func TestSummariser_Summarise_ProviderErrorPropagates(t *testing.T) {
	provider := llm.NewScriptedProvider() // nothing queued or defaulted
	s := &llm.Summariser{Provider: provider, Model: "gpt-4.1-mini"}

	_, err := s.Summarise(context.Background(), "some text")
	if err == nil {
		t.Fatal("expected the provider's error to propagate")
	}
}

func TestSummariser_Summarise_WrongToolNameIsError(t *testing.T) {
	provider := llm.NewScriptedProvider()
	provider.Enqueue(llm.SummariseToolName, llm.ToolCall{Name: "not_record_summary", Arguments: json.RawMessage(`{}`)})
	s := &llm.Summariser{Provider: provider, Model: "gpt-4.1-mini"}

	_, err := s.Summarise(context.Background(), "some text")
	if err == nil || !strings.Contains(err.Error(), "not_record_summary") {
		t.Fatalf("expected an error naming the unexpected tool, got %v", err)
	}
}

func TestSummariser_Summarise_InvalidArgumentsJSONIsError(t *testing.T) {
	provider := llm.NewScriptedProvider()
	provider.Enqueue(llm.SummariseToolName, llm.ToolCall{Name: llm.SummariseToolName, Arguments: json.RawMessage(`not json`)})
	s := &llm.Summariser{Provider: provider, Model: "gpt-4.1-mini"}

	_, err := s.Summarise(context.Background(), "some text")
	if err == nil {
		t.Fatal("expected a decode error for malformed tool arguments")
	}
}
