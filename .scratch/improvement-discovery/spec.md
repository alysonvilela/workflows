# Improvement Discovery — MVP

Status: ready-for-agent

## Problem Statement

A consultant is hired to find where a company's processes could be improved with software. Today that means reading whatever the client sends — process docs, exported conversations, operational spreadsheets — holding all of it in their head, and writing a proposal by hand. It is slow, it does not scale past one client at a time, and the reasoning behind "these are the five things worth building" lives nowhere except the final document.

Two things make it worse. First, the client's material is almost never complete: halfway through analysing a pain point the consultant realises they need the cost of the current process, or the volume per month, or the actual spreadsheet the sales team uses — and the work stalls on an email. Second, the analysis is not reviewable. Nobody can see which ideas were considered and discarded, or on what basis.

## Solution

A system that holds a Company's documents as a readable Company Context, then runs a Discovery Run over it: an LLM scans the whole context for Opportunities, ranks them by impact against effort, and the consultant chooses how many of the top ones to investigate. Each Opportunity becomes a Node that reasons about it independently, and each Node either produces a concrete implementation spec, discards the Opportunity with a reason, or opens a Gate — a stop that asks a human for exactly the missing piece, either as a dynamic form or as a request for another Document.

A Gate blocks only its own branch; every sibling keeps working. A blocked Discovery Run costs one database row and zero running processes for as long as it waits — days, if needed. Answering the Gate resumes that branch exactly where it stopped, with the answer folded into the reasoning. When no branch has work left, the Run synthesises the surviving specs into one final plan.

The whole tree is visible as a graph, so the reasoning is the artifact, not just the conclusion.

## User Stories

### Company and Documents

1. As a consultant, I want to create a Company, so that everything I learn about one client accumulates in one place across months.
2. As a consultant, I want to upload Markdown files to a Company, so that process documentation the client wrote becomes analysable.
3. As a consultant, I want to upload CSV files to a Company, so that exported conversations and operational spreadsheets become analysable alongside the prose.
4. As a consultant, I want to upload several files at once, so that onboarding a client is one action rather than ten.
5. As a consultant, I want to see each Document's ingestion state, so that I know whether it is still parsing, ready, or failed.
6. As a consultant, I want a failed Document to tell me why it failed, so that I can fix the file rather than guess.
7. As a consultant, I want a Document that exceeds the size limit to fail with a clear message, so that I find out immediately instead of through a provider error.
8. As a consultant, I want to remove a Document, so that superseded or wrong material stops influencing analysis.
9. As a consultant, I want removing a Document to remove its contribution to the Company Context, so that the analysis never reasons from something I deleted.
10. As a consultant, I want to read the Company Context the system assembled, so that I can tell whether the summariser understood the material before I spend a Run on it.
11. As a consultant, I want the original uploaded file kept, so that improving the summariser prompt never means asking the client to send everything again.

### Starting a Discovery Run

12. As a consultant, I want to start a Discovery Run on a Company, so that the analysis begins from everything currently known about them.
13. As a consultant, I want to give the Run an optional theme, so that I can focus the analysis on one area — sales, onboarding, billing — instead of the whole business.
14. As a consultant, I want to choose how many Opportunities to explore, so that I control how wide and how expensive the Run is.
15. As a consultant, I want choosing one Opportunity to give me a Single Workflow, so that I can go deep on a pain point I already know matters.
16. As a consultant, I want choosing more than one to give me a Discovery Workflow, so that I can compare several improvement paths side by side.
17. As a consultant, I want to run a new Discovery Run over the same Company without re-uploading anything, so that a second opinion or a different theme costs nothing but LLM time.
18. As a consultant, I want to see previous Runs for a Company, so that I can revisit what was concluded before.

### The Discovery Run

19. As a consultant, I want the system to scan the whole Company Context for Opportunities, so that findings come from the client's own material and not from a generic checklist.
20. As a consultant, I want each Opportunity scored for impact and effort, so that the ranking reflects quick wins rather than arbitrary order.
21. As a consultant, I want only the top-ranked Opportunities investigated, so that budget goes to the ones worth investigating.
22. As a consultant, I want each Opportunity investigated in its own branch, so that one weak idea does not contaminate the reasoning about a strong one.
23. As a consultant, I want branches to run in parallel, so that a five-Opportunity Run does not take five times as long as a one-Opportunity Run.
24. As a consultant, I want an Opportunity the system judges unsolvable to be discarded with a written reason, so that I can defend "we looked at that and here is why not".
25. As a consultant, I want a viable Opportunity to produce a concrete implementation spec, so that the output is something a developer can act on.
26. As a consultant, I want the Run to stop growing at a fixed depth and node count, so that a bad prompt cannot turn into an open-ended bill.
27. As a consultant, I want a Node that fails to record its error without killing the whole Run, so that one bad branch does not waste the others.
28. As a consultant, I want the Run to produce one consolidated final plan when every branch has finished, so that I have a single document to bring to the client.
29. As a consultant, I want the final plan to explain why these specs and not others, so that the narrative survives outside my head.

### Gates — the human in the loop

30. As a consultant, I want a Node to stop and ask me when the Company Context does not contain something it needs, so that the reasoning never quietly invents a fact.
31. As a consultant, I want the question rendered as a form built from the schema the Node produced, so that I answer exactly what is missing without free-typing into a chat.
32. As a consultant, I want a Node to be able to ask for a whole Document instead of a field, so that "I need their cost spreadsheet" is expressible.
33. As a consultant, I want the Document I upload in answer to a Gate to join the Company Context, so that every later Node benefits from it too.
34. As a consultant, I want answering a Gate to resume only that branch, so that the rest of the Run was never waiting on me.
35. As a consultant, I want sibling branches to keep running while a Gate is open, so that my response time is not the Run's bottleneck.
36. As a consultant, I want to dismiss a Gate I cannot answer, so that missing information does not strand a Run forever.
37. As a consultant, I want a dismissed Gate to make the Node proceed on a stated conservative assumption, so that the output stays honest about what it did not know.
38. As a consultant, I want my answer validated against the Node's schema on the server, so that a malformed answer is rejected at the door instead of surfacing as strange reasoning three Nodes later.
39. As a consultant, I want a double-submitted answer to take effect once, so that an impatient click does not duplicate a branch.
40. As a consultant, I want one list of every open Gate across all Runs, so that answering questions is a work queue rather than a hunt.
41. As a consultant, I want to see how long a Gate has been open, so that I can tell which Runs are stalled on me.

### Seeing the reasoning

42. As a consultant, I want the Run drawn as a graph of Nodes, so that I can see the shape of the reasoning rather than a log.
43. As a consultant, I want a Node's status visible on the graph, so that I can tell at a glance what is running, blocked, done, or failed.
44. As a consultant, I want blocked Nodes visually distinct, so that I can find what needs me without reading every node.
45. As a consultant, I want discarded branches dimmed rather than hidden, so that rejected reasoning stays auditable.
46. As a consultant, I want to click a Node and read its input and output, so that I can debug a bad conclusion back to the prompt that caused it.
47. As a consultant, I want the graph to update while the Run progresses, so that I can watch a Run rather than refresh it.

### Operating it

48. As a developer, I want a single `.env` for the whole monorepo, so that there is one place to configure the system.
49. As a developer, I want to point the system at any OpenAI-compatible provider from that `.env`, so that I can switch between a hosted model and a local one without touching code.
50. As a developer, I want `make dev` to bring up infrastructure, the API with hot reload, and the frontend, so that starting work is one command.
51. As a developer, I want the Go API to rebuild on save, so that the feedback loop is seconds.
52. As a developer, I want in-flight Nodes to finish on shutdown rather than being killed, so that a hot reload does not corrupt a Run.
53. As a developer, I want a Node interrupted by a crash or deploy to become available again, so that no Run is permanently stuck in `running`.
54. As a developer, I want one compose file for local dependencies and one for the full deployment, so that dev and Dokploy do not fight over the same file.

## Implementation Decisions

### Shape

Go monorepo, flat: `backend/` and `frontend/` at the root, `pnpm-workspace.yaml` at the root, one `.env` at the root read by compose, by the Go API, and — for `VITE_*` only — by the frontend. Backend follows Standard Go Project Layout: `cmd/api` plus `internal/{domain,strategy,parser,llm,store,storage,http}` and `migrations/`.

**One binary.** The HTTP server and the River worker pool run in the same process, started from `cmd/api`. `WORKER_ENABLED` can turn the worker half off so the process becomes HTTP-only, which is the seam for splitting deployment later. Two binaries today buys nothing and costs two Dockerfiles.

**Postgres via pgx, migrations via goose, no ORM and no sqlc.** Four tables do not justify a codegen step. Adopting `squirrel` as a query builder over the hand-written SQL is a follow-up; goose stays either way.

**Frontend API client is hand-written**, roughly fifty lines. No OpenAPI or Orval on day one.

### Data model — four tables

`companies` — the aggregate root. Carries `company_context jsonb`, an array of `[filename, summary]` entries, one per ready Document. It is what gets ingested into every prompt.

`documents` — belongs to a Company. Carries `s3_key`, `title`, `kind` (`markdown` | `csv`), `raw_text` (the extracted text, never sent to the LLM), `summary jsonb` (the summariser's output, which is what lands in `company_context`), `status`, `error`, `checksum`.

`runs` — a Discovery Run. Carries `company_id`, `theme` (nullable), `opportunity_count`, `max_depth`, `max_nodes`, `status` (`running` | `awaiting_input` | `synthesizing` | `done` | `failed`), `result jsonb`, timestamps.

`nodes` — one unit of reasoning. Carries `run_id`, `parent_id` (null at the root), `ord`, `kind`, `status` (`pending` | `running` | `blocked` | `done` | `failed`), `depth`, `input jsonb`, `output jsonb`, `error`, `claimed_at`, timestamps. Unique index on `(parent_id, kind, ord)`.

River owns its own tables through `rivermigrate`; they are not part of these four.

Three tables were deliberately not created, and each has a query that replaces it:

- **No `chunks`.** There are no embeddings (ADR-0001).
- **No `human_requests`.** A Gate is `nodes.kind IN ('ask_human','ask_document')` with the form schema in `input` and the answer in `output`. The cross-Run inbox is `WHERE kind IN (...) AND status='blocked'`.
- **No `opportunities`.** Delivered Opportunities are `WHERE kind='spec_solution' AND status='done'`. Promoting this to a table is the right move once specs accumulate across Runs and the spec format stabilises — it is a follow-up, not a day-one need.

### Ingestion

Upload writes the file to object storage first, then parses. A **Parse Strategy** per format (`markdown`, `csv`) turns bytes into `raw_text`; the CSV one streams rather than loading the file. `MAX_UPLOAD_BYTES` is checked before parsing so an oversized file fails as a Document error, not as a provider error.

`raw_text` then goes to the **summariser**, a single LLM call producing `{theme, context, topics, pains, examples, opportunities}`. That summary — not `raw_text` — is appended to `companies.company_context`.

The append is serialised by `SELECT ... FROM companies WHERE id = $1 FOR UPDATE` inside the ingestion transaction: a second concurrent ingestion blocks until the first commits, then reads the new value and appends to it. This is the lost-update guard and it is the entire mechanism — no application-level locking.

`STORAGE_DELETE_AFTER_INGEST` removes the original object once the Document is ready. It is `true` in dev to keep the volume small and must be `false` in production, where the original is the only way to re-parse. It is an environment flag, not commented-out code, so the production path is the one that gets exercised.

Ingestion is reachable from inside a Run, not only before one — answering an `ask_document` Gate runs the same path.

### Orchestration

One River job per Node, inserted with `river.InsertTx` in the same transaction that inserts the Node. Either the Node exists and is enqueued, or neither happened.

A **Node Strategy** is one LLM call. The Node's prompt carries the entire Company Context, and the model must answer with exactly one terminal tool call; the orchestrator translates that call into child Nodes. There is no read-tool loop and no per-Node message history (ADR-0002). The tool arguments *are* the structured output — schema validation happens on the tool call, and there is no separate verdict enum.

| kind | tools exposed | orchestrator creates |
|---|---|---|
| `scan_opportunities` | `propose_opportunities(items[])` | N × `assess_opportunity`, cut to `opportunity_count` |
| `assess_opportunity` | `request_information(title, json_schema)` | `ask_human` child |
| | `request_document(title, what_for)` | `ask_document` child |
| | `specify_solution(rationale)` | `spec_solution` child |
| | `discard(reason)` | nothing — leaf |
| `ask_human` | none — no LLM call | unblocks on answer |
| `ask_document` | none — no LLM call | unblocks on upload |
| `spec_solution` | `deliver_spec(...)` | nothing — leaf |
| `synthesize` | `deliver_plan(...)` | ends the Run |

`propose_opportunities` items carry description, rationale, `impact` (1–5) and `effort` (1–5), already ordered by the model. Ranking lives in the prompt, not in Go — the backend validates the schema and cuts the list at `opportunity_count`. Changing how Opportunities are prioritised is a prompt change, not an orchestration change.

The JSON Schema a Node may emit for a Gate is restricted to a flat object whose properties are `string`, `number`, `boolean` or `enum` — no nesting, no `oneOf`, no `$ref`. A schema outside the subset is a retry against the model, not an error shown to the consultant.

**Termination.** No Node is marked final. A Run has work left while any Node is `pending`, `running` or `blocked`. When that count reaches zero, the Run transitions to `synthesizing` and the orchestrator — not a strategy — creates the single `synthesize` Node. The status transition is what prevents the classic loop where `synthesize` completes, the "any work left?" check goes false, and a second `synthesize` is created.

**Brakes.** `max_depth` (3) and `max_nodes` (40) are bug brakes, not product settings, and are invisible in the UI. Tripping one is an anomaly worth logging loudly, not swallowing. `opportunity_count` (1–6, from the frontend) is the only breadth control the consultant sees — `1` yields a Single Workflow, more yields a Discovery Workflow.

### Gates

`POST /runs/{runId}/nodes/{nodeId}/answer` validates the body against the Node's stored schema server-side, then in one short transaction claims the Gate (`UPDATE ... WHERE status='blocked'` — zero rows means already answered, and the request returns quietly), writes the answer to `output`, sets the Node back to `pending`, and enqueues its job. The LLM call happens in the worker afterwards, never inside the transaction.

`POST .../dismiss` does the same with a null answer; the Node's prompt is written to proceed on the most conservative assumption and record that assumption in its output, so the resulting spec states what it did not know.

`ask_document` is answered by an upload, which runs the ingestion path and then unblocks the Node the same way.

### Concurrency

Goroutines and channels are used where they are local: `errgroup` for a multi-file upload, a streaming pipeline for large CSVs, `semaphore.Weighted` in the LLM Provider sized from `LLM_MAX_CONCURRENT`, and the two long-lived goroutines in `main` (the River client and the HTTP server). River owns the Node worker pool, sized by `RIVER_MAX_WORKERS` — that is the parallelism of the branch fan-out.

What is forbidden is coordinating Run or Node state in memory (ADR-0003): no channel hands a Node from a handler to a worker, no struct caches the node tree between jobs, no goroutine holds a Run. The test is lifetime — **a goroutine that outlives the HTTP request or the River job that spawned it is a bug.**

Three writes carry the correctness, all in SQL:

```sql
-- claim a Node; zero rows means another worker already has it
UPDATE nodes SET status='running', claimed_at=now()
WHERE id=$1 AND status='pending' RETURNING *;

-- claim a Gate; zero rows means it was already answered
UPDATE nodes SET status='pending', output=$2
WHERE id=$1 AND status='blocked' RETURNING *;

-- serialise Company Context appends
SELECT company_context FROM companies WHERE id=$1 FOR UPDATE;
```

A worker whose claim returns zero rows must return `nil` from the job, not an error — River retries errors, and a duplicate-delivery retry is a success, not a failure. A lease sweeper returns Nodes stuck in `running` with `claimed_at` older than five minutes to `pending`, which is what makes a crashed or redeployed worker recoverable. Shutdown drains: `signal.NotifyContext` → `riverClient.Stop(ctx)` → `srv.Shutdown(ctx)`, and `air` is configured with a kill delay so hot reload does not sever a Node mid-call.

### Frontend

Vite+ (`vp`), React Flow for the graph, `@coss` components. Dynamic Gate forms use RJSF with a custom theme built via `withTheme` — roughly seven overrides (`FieldTemplate`, `ObjectFieldTemplate`, `BaseInputTemplate`, `SelectWidget`, `CheckboxWidget`, `ErrorListTemplate`, submit button) mapped onto the existing `field`, `fieldset`, `form`, `select` and `checkbox` components. The restricted schema subset is what keeps that override list short. Live updates are by polling; the frontend is untested on day one.

## Testing Decisions

A good test here drives the system the way a consultant does and asserts what a consultant would see — Documents ready, a tree of Nodes with given kinds and statuses, a Gate open, a final plan present. It never reaches into a use case, asserts on a prompt string, or checks that a particular function was called. Prompts and Node internals will change constantly; the shape of the resulting tree is the contract.

**Seam 1 — the HTTP API, against real Postgres and real rustfs.** Where almost every test lives. Tests boot the process against a clean schema and drive it over HTTP: create a Company, upload Documents, start a Discovery Run, answer a Gate, read the plan. Behavioural tests and the end-to-end test share this seam. Postgres and object storage are real because faking them means testing the fakes.

**Seam 2 — the LLM Provider interface.** The only fake in the backend. `Provider.Complete(ctx, prompt, tools) → ToolCall` is backed in tests by a scripted provider that returns a queued sequence of tool calls keyed by Node kind — `scan_opportunities` returns four Opportunities, the second `assess_opportunity` returns `request_information`, and so on. This makes the whole tree deterministic and the end-to-end suite fast and free.

**Seam 3 — the concurrency invariants, at the repository.** Deliberately low, because these three failures are silent: nothing at the HTTP level notices them, and they surface in production as duplicated Nodes or a Document missing from the Company Context. Each is driven by N goroutines hitting real Postgres and run under `-race`:

- 20 goroutines claiming the same `pending` Node — exactly one wins, nineteen get zero rows.
- 10 concurrent ingestions into one Company — all ten summaries are present in `company_context` afterwards.
- the same River job body executed twice — children are not duplicated, and the second execution returns `nil`.
- a Node abandoned in `running` past its lease — the sweeper returns it to `pending` and it completes.

Concurrency is also asserted at Seam 1, without dropping a level: a Discovery Run with `opportunity_count=5` and `RIVER_MAX_WORKERS=8` runs five branches simultaneously, and the test asserts the final tree and Node count. An orchestration race shows up there.

**Unit tests** cover the pure pieces: the Gate schema validator and its rejection of out-of-subset schemas, the CSV and Markdown Parse Strategies, and the tool-call-to-children translation. These are the `-short` suite and need no infrastructure.

There is no prior art in this repository — it is greenfield, so these tests set the pattern. Tests are co-located with the code they cover.

## Out of Scope

- **Vector retrieval.** No embeddings, no chunk table, no similarity search. The Company Context is concatenated text (ADR-0001). This is the planned next phase, and `BuildCompanyContext` is the seam it lands behind.
- **Agentic Nodes.** No read tools, no message history, no `max_steps`. Nodes are one LLM call (ADR-0002). Revisit together with vector retrieval, when the corpus stops fitting in a prompt — it is considered a good design, just not one this MVP needs.
- **Authentication and multi-user.** No login, no `access_token`, no public Gate links. Gates are answered by the consulting team, from the app.
- **An `opportunities` table.** Delivered specs stay derivable from Nodes until the spec format stabilises.
- **CSV sampling and map-reduce summarisation.** No column profiling, no chunked summarise-then-reduce. Oversized files fail with a clear error instead.
- **`squirrel` as query builder**, OpenAPI/Orval client generation, and a message broker (NATS/Redis) — River on Postgres is the queue.
- **Frontend tests**, node version history, spec editing, diffing Runs, and any automation on top of stale Gates. Gate age is displayed, and nothing acts on it.

## Further Notes

Delivery order, each band demonstrable on its own:

1. **Skeleton** — compose (Postgres + rustfs), migrations for the four tables, chi with health, Makefile, `.env`. Demo: `make dev` brings everything up.
2. **Ingestion** — upload → object storage → Parse Strategy → summariser → Company Context. Demo: three files in, assembled context out.
3. **Orchestration** — River, Node claiming, `scan_opportunities`, `assess_opportunity`, `discard`. No Gates yet. Demo: a Run that finds and ranks Opportunities.
4. **Gates** — `ask_human` with the themed RJSF form, then `ask_document`. Demo: a Run blocks, gets answered, resumes.
5. **Closing** — `spec_solution`, `synthesize`, final plan on screen.
6. **Graph** — React Flow tree with the Node panel. Last deliberately: it is the most visible layer and the least necessary to prove the thesis.

Tests are a separate ticket rather than a band, covering all three seams.

The three failure modes most likely to appear first, all guarded above and all worth checking early: an unrestricted JSON Schema from the model breaking the form renderer; a duplicate `synthesize` Node from checking "any work left?" without the Run status transition; and an unbounded fan-out turning into an open-ended provider bill.

Relevant decisions: ADR-0001 (Company Context instead of vector RAG), ADR-0002 (Nodes are one-shot, not agent loops), ADR-0003 (Postgres coordinates Discovery Runs; goroutines stay request-scoped). Vocabulary is defined in `CONTEXT.md` and used throughout this spec.
