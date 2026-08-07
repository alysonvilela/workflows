# 16 — River-backed Document ingestion

**What to build:** Document processing (Parse Strategy → summariser → ready/failed) moves off the ad-hoc in-process poller onto River, so there is one queue every background job — including Nodes in ticket 08 — runs on. Not hypothetical: summarising averages ~10s per call and this is a service on an existing platform, not a toy — a hand-rolled claim loop isn't worth maintaining once River is coming in anyway.

**Blocked by:** 05

**Status:** ready-for-agent

Pulls forward the "River integrated into the same binary" work ticket 08 was going to do first. Landing it here means ticket 08 only adds the Node job kind on top of an already-running queue instead of standing one up itself. Keep this ticket to exactly that swap — change nothing about what ingestion does, only what runs it.

- [ ] River integrated into the same binary via `rivermigrate`, with `RIVER_MAX_WORKERS` from `.env`; `WORKER_ENABLED` still turns the worker half off
- [ ] One River job per Document, inserted with `InsertTx` in the same transaction that creates the Document row (Stage) — either both happen or neither
- [ ] The job body claims nothing itself — River's own delivery is the claim; a duplicate delivery over a Document already `ready`/`failed` returns `nil`, not an error
- [ ] `documents.claimed_at` and the `SELECT ... FOR UPDATE SKIP LOCKED` poll loop are removed — River replaces both, no two queuing mechanisms left side by side
- [ ] `LLM_MAX_CONCURRENT`'s semaphore in the Provider stays the hard cap on outbound LLM calls; `RIVER_MAX_WORKERS` only bounds how many jobs may be dequeued at once and can sit well above it
- [ ] Graceful shutdown: `riverClient.Stop(ctx)` drains an in-flight summarise call before the process exits; air's `kill_delay` doesn't sever it mid-call
- [ ] Tests: the Seam 1 regression that upload responds before the job runs still holds on the River path; a job body executed twice against an already-terminal Document does not re-process and reports success, checked under `-race`
