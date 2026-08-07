// Package llm is the backend's one mocked boundary (see ScriptedProvider):
// an OpenAI-compatible Provider that answers only through tool calls, per
// ADR-0002. The summariser is its first caller; Node Strategies are next.
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/sync/semaphore"
)

// Tool is one function the model may call, in the shape an
// OpenAI-compatible /chat/completions endpoint expects: Parameters is a
// JSON Schema object.
type Tool struct {
	Name        string
	Description string
	Parameters  json.RawMessage
}

// ToolCall is the model's answer: which Tool it chose, and the arguments
// it filled in — still raw JSON, since only the caller knows the shape
// its own Tool's Parameters describe.
type ToolCall struct {
	Name      string
	Arguments json.RawMessage
}

// Provider is every LLM caller's only dependency. A prompt and a set of
// Tools go in; the model must answer with exactly one tool call — there is
// no read-tool loop (ADR-0002), so Complete is one round trip.
type Provider interface {
	Complete(ctx context.Context, model, prompt string, tools []Tool) (ToolCall, error)
}

// Config configures OpenAIProvider entirely from the LLM_* environment
// values — see config.Config.
type Config struct {
	BaseURL       string
	APIKey        string
	MaxConcurrent int
	Timeout       time.Duration
}

// OpenAIProvider talks to any OpenAI-compatible /chat/completions endpoint
// (OpenAI, OpenRouter, Groq, Together, vLLM, Ollama, LM Studio, ...).
// Concurrency and per-call timeout are enforced here, once, so every
// caller gets them for free.
type OpenAIProvider struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	sem        *semaphore.Weighted
	timeout    time.Duration
}

func NewOpenAIProvider(cfg Config) *OpenAIProvider {
	return &OpenAIProvider{
		baseURL:    strings.TrimRight(cfg.BaseURL, "/"),
		apiKey:     cfg.APIKey,
		httpClient: &http.Client{},
		sem:        semaphore.NewWeighted(int64(cfg.MaxConcurrent)),
		timeout:    cfg.Timeout,
	}
}

type chatRequest struct {
	Model      string        `json:"model"`
	Messages   []chatMessage `json:"messages"`
	Tools      []chatTool    `json:"tools"`
	ToolChoice string        `json:"tool_choice"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatTool struct {
	Type     string       `json:"type"`
	Function chatFunction `json:"function"`
}

type chatFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			ToolCalls []struct {
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"message"`
	} `json:"choices"`
}

// Complete acquires a concurrency slot (bounded by LLM_MAX_CONCURRENT),
// applies the per-call timeout (LLM_TIMEOUT_SECONDS), and forces the model
// to answer with one of tools — tool_choice is "required", never "auto",
// because the orchestrator has nothing to do with free text.
func (p *OpenAIProvider) Complete(ctx context.Context, model, prompt string, tools []Tool) (ToolCall, error) {
	if err := p.sem.Acquire(ctx, 1); err != nil {
		return ToolCall{}, fmt.Errorf("llm: acquire concurrency slot: %w", err)
	}
	defer p.sem.Release(1)

	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	reqBody, err := json.Marshal(chatRequest{
		Model:      model,
		Messages:   []chatMessage{{Role: "user", Content: prompt}},
		Tools:      toChatTools(tools),
		ToolChoice: "required",
	})
	if err != nil {
		return ToolCall{}, fmt.Errorf("llm: encode request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(reqBody))
	if err != nil {
		return ToolCall{}, fmt.Errorf("llm: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return ToolCall{}, fmt.Errorf("llm: call timed out after %s", p.timeout)
		}
		return ToolCall{}, fmt.Errorf("llm: request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ToolCall{}, fmt.Errorf("llm: read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return ToolCall{}, fmt.Errorf("llm: provider returned status %d: %s", resp.StatusCode, string(body))
	}

	var parsed chatResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return ToolCall{}, fmt.Errorf("llm: decode response: %w", err)
	}
	if len(parsed.Choices) == 0 || len(parsed.Choices[0].Message.ToolCalls) == 0 {
		return ToolCall{}, fmt.Errorf("llm: model did not return a tool call")
	}

	call := parsed.Choices[0].Message.ToolCalls[0].Function
	return ToolCall{Name: call.Name, Arguments: json.RawMessage(call.Arguments)}, nil
}

func toChatTools(tools []Tool) []chatTool {
	out := make([]chatTool, len(tools))
	for i, t := range tools {
		out[i] = chatTool{
			Type: "function",
			Function: chatFunction{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.Parameters,
			},
		}
	}
	return out
}
