# 06 — Company Context: assembly, removal and re-summarising

**What to build:** the consultant reads exactly what the LLM will read. Documents add to the Company Context, removing a Document removes its contribution, and re-summarising a Document refreshes the context without another upload.

**Blocked by:** 05

**Status:** ready-for-agent

This is the piece the spec calls correct-by-SQL rather than correct-by-Go — the append is serialised with `SELECT ... FOR UPDATE` on the Company row. See ADR-0003.

- [ ] A ready Document's summary is appended to `companies.company_context`
- [ ] Two simultaneous ingestions into the same Company lose neither — the append happens under a row lock
- [ ] Company Context readable in the interface, presented as the text that goes into the prompt
- [ ] Removing a Document removes its Company Context entry in the same transaction
- [ ] Re-summarising a Document reprocesses the stored `raw_text`, with no re-upload, and replaces the matching entry
- [ ] `STORAGE_DELETE_AFTER_INGEST` deletes the original from object storage when enabled, and the disabled path is the one under test
- [ ] Behavioural tests at Seam 1 covering add, remove and re-summarise
