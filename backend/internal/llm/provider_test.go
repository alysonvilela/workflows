package llm_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/discovery/backend/internal/llm"
)

func newTestTool() llm.Tool {
	return llm.Tool{
		Name:        "record_summary",
		Description: "Record a summary.",
		Parameters:  json.RawMessage(`{"type":"object","properties":{"theme":{"type":"string"}}}`),
	}
}

func TestOpenAIProvider_Complete_ReturnsToolCall(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("Authorization header = %q, want %q", got, "Bearer test-key")
		}

		var body struct {
			Model      string `json:"model"`
			ToolChoice string `json:"tool_choice"`
			Tools      []struct {
				Function struct {
					Name string `json:"name"`
				} `json:"function"`
			} `json:"tools"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if body.Model != "gpt-4.1-mini" {
			t.Errorf("model = %q, want %q", body.Model, "gpt-4.1-mini")
		}
		if body.ToolChoice != "required" {
			t.Errorf("tool_choice = %q, want %q", body.ToolChoice, "required")
		}
		if len(body.Tools) != 1 || body.Tools[0].Function.Name != "record_summary" {
			t.Fatalf("tools = %+v, want one record_summary tool", body.Tools)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"choices": [{
				"message": {
					"tool_calls": [{
						"function": {"name": "record_summary", "arguments": "{\"theme\":\"ok\"}"}
					}]
				}
			}]
		}`))
	}))
	defer srv.Close()

	p := llm.NewOpenAIProvider(llm.Config{
		BaseURL:       srv.URL,
		APIKey:        "test-key",
		MaxConcurrent: 4,
		Timeout:       5 * time.Second,
	})

	call, err := p.Complete(context.Background(), "gpt-4.1-mini", "summarise this", []llm.Tool{newTestTool()})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if call.Name != "record_summary" {
		t.Errorf("Name = %q, want %q", call.Name, "record_summary")
	}
	if string(call.Arguments) != `{"theme":"ok"}` {
		t.Errorf("Arguments = %s, want %s", call.Arguments, `{"theme":"ok"}`)
	}
}

func TestOpenAIProvider_Complete_NoToolCallInResponseIsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices": [{"message": {}}]}`))
	}))
	defer srv.Close()

	p := llm.NewOpenAIProvider(llm.Config{BaseURL: srv.URL, MaxConcurrent: 4, Timeout: 5 * time.Second})
	_, err := p.Complete(context.Background(), "gpt-4.1-mini", "prompt", []llm.Tool{newTestTool()})
	if err == nil {
		t.Fatal("expected an error when the model returns no tool call")
	}
}

func TestOpenAIProvider_Complete_NonOKStatusIsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"message":"rate limited"}}`))
	}))
	defer srv.Close()

	p := llm.NewOpenAIProvider(llm.Config{BaseURL: srv.URL, MaxConcurrent: 4, Timeout: 5 * time.Second})
	_, err := p.Complete(context.Background(), "gpt-4.1-mini", "prompt", []llm.Tool{newTestTool()})
	if err == nil {
		t.Fatal("expected an error for a non-200 response")
	}
}

func TestOpenAIProvider_Complete_TimeoutIsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"tool_calls":[{"function":{"name":"record_summary","arguments":"{}"}}]}}]}`))
	}))
	defer srv.Close()

	p := llm.NewOpenAIProvider(llm.Config{BaseURL: srv.URL, MaxConcurrent: 4, Timeout: 20 * time.Millisecond})
	_, err := p.Complete(context.Background(), "gpt-4.1-mini", "prompt", []llm.Tool{newTestTool()})
	if err == nil {
		t.Fatal("expected the per-call timeout to produce an error")
	}
}

func TestOpenAIProvider_Complete_ConcurrencyIsCapped(t *testing.T) {
	var inFlight, maxInFlight int64

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt64(&inFlight, 1)
		defer atomic.AddInt64(&inFlight, -1)
		for {
			max := atomic.LoadInt64(&maxInFlight)
			if n <= max || atomic.CompareAndSwapInt64(&maxInFlight, max, n) {
				break
			}
		}
		time.Sleep(50 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"tool_calls":[{"function":{"name":"record_summary","arguments":"{}"}}]}}]}`))
	}))
	defer srv.Close()

	p := llm.NewOpenAIProvider(llm.Config{BaseURL: srv.URL, MaxConcurrent: 1, Timeout: 5 * time.Second})

	done := make(chan error, 3)
	for i := 0; i < 3; i++ {
		go func() {
			_, err := p.Complete(context.Background(), "gpt-4.1-mini", "prompt", []llm.Tool{newTestTool()})
			done <- err
		}()
	}
	for i := 0; i < 3; i++ {
		if err := <-done; err != nil {
			t.Fatalf("Complete: %v", err)
		}
	}

	if got := atomic.LoadInt64(&maxInFlight); got != 1 {
		t.Errorf("max concurrent requests observed = %d, want 1 (LLM_MAX_CONCURRENT=1)", got)
	}
}
