package llm_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/discovery/backend/internal/llm"
)

func TestScriptedProvider_QueuedRepliesConsumedInOrderBeforeDefault(t *testing.T) {
	p := llm.NewScriptedProvider()
	tool := llm.Tool{Name: "record_summary"}

	p.Enqueue("record_summary", llm.ToolCall{Name: "record_summary", Arguments: json.RawMessage(`{"n":1}`)})
	p.Enqueue("record_summary", llm.ToolCall{Name: "record_summary", Arguments: json.RawMessage(`{"n":2}`)})
	p.SetDefault("record_summary", llm.ToolCall{Name: "record_summary", Arguments: json.RawMessage(`{"n":"default"}`)})

	for _, want := range []string{`{"n":1}`, `{"n":2}`, `{"n":"default"}`, `{"n":"default"}`} {
		call, err := p.Complete(context.Background(), "any-model", "any-prompt", []llm.Tool{tool})
		if err != nil {
			t.Fatalf("Complete: %v", err)
		}
		if string(call.Arguments) != want {
			t.Errorf("Arguments = %s, want %s", call.Arguments, want)
		}
	}
}

func TestScriptedProvider_NoReplyIsError(t *testing.T) {
	p := llm.NewScriptedProvider()
	_, err := p.Complete(context.Background(), "any-model", "any-prompt", []llm.Tool{{Name: "unscripted"}})
	if err == nil {
		t.Fatal("expected an error when no reply is queued or defaulted")
	}
}
