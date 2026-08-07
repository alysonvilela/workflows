package llm

import (
	"context"
	"fmt"
	"sync"
)

// ScriptedProvider is the fake every test in this repository uses in place
// of a real network call — the basis of every test that follows, per the
// spec's Seam 2. Replies are queued per tool name; SetDefault covers the
// common case of a test that only cares that the call succeeded, not what
// the model "said".
type ScriptedProvider struct {
	mu       sync.Mutex
	queued   map[string][]ToolCall
	fallback map[string]ToolCall
}

func NewScriptedProvider() *ScriptedProvider {
	return &ScriptedProvider{
		queued:   make(map[string][]ToolCall),
		fallback: make(map[string]ToolCall),
	}
}

// Enqueue appends a one-time reply, returned the next time Complete is
// offered a set of tools that includes toolName. Queued replies are
// consumed before any default.
func (p *ScriptedProvider) Enqueue(toolName string, call ToolCall) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.queued[toolName] = append(p.queued[toolName], call)
}

// SetDefault sets the reply Complete falls back to for toolName once its
// queue is empty.
func (p *ScriptedProvider) SetDefault(toolName string, call ToolCall) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.fallback[toolName] = call
}

func (p *ScriptedProvider) Complete(_ context.Context, _, _ string, tools []Tool) (ToolCall, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, tool := range tools {
		if q := p.queued[tool.Name]; len(q) > 0 {
			call := q[0]
			p.queued[tool.Name] = q[1:]
			return call, nil
		}
	}
	for _, tool := range tools {
		if call, ok := p.fallback[tool.Name]; ok {
			return call, nil
		}
	}

	names := make([]string, len(tools))
	for i, t := range tools {
		names[i] = t.Name
	}
	return ToolCall{}, fmt.Errorf("llm: scripted provider has no reply queued or defaulted for %v", names)
}
