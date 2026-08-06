# 05 — LLM Provider and summariser

**What to build:** the text extracted from a Document becomes a structured summary produced by the LLM. This is where the Provider is born — the backend's only mocked boundary — along with the fake that makes everything downstream testable.

**Blocked by:** 04

**Status:** ready-for-agent

- [ ] The Provider talks to any OpenAI-compatible API, configured entirely from `.env` (`LLM_BASE_URL`, `LLM_API_KEY`, `LLM_MODEL`)
- [ ] The Provider supports tool calling — the whole orchestration depends on it later
- [ ] Concurrent calls capped by `LLM_MAX_CONCURRENT`, using a semaphore inside the Provider
- [ ] Per-call timeout from `LLM_TIMEOUT_SECONDS`, with the error recorded on the Document rather than swallowed
- [ ] The summariser produces `{theme, context, topics, pains, examples, opportunities}` from `raw_text` and writes it to `documents.summary`
- [ ] `LLM_SUMMARY_MODEL` allows a cheaper model for summarising
- [ ] Scripted fake provider returning a sequence of tool calls per call type — the basis of every test that follows
- [ ] Unit tests for the Provider and the summariser against the fake, with no network
