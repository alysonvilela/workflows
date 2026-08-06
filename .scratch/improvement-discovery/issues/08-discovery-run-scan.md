# 08 — Discovery Run and `scan_opportunities`

**What to build:** the consultant starts a Discovery Run over a Company, choosing an optional theme and how many Opportunities to explore. The root Node reads the entire Company Context, finds and ranks Opportunities, and the first generation of Nodes is born from the best ones.

**Blocked by:** 06

**Status:** ready-for-agent

- [ ] River integrated into the same binary, with its own migrations and `RIVER_MAX_WORKERS` from `.env`
- [ ] One job per Node, inserted with `InsertTx` in the same transaction that inserts the Node — either both happen or neither
- [ ] Atomic Node claim at the start of the worker; zero rows returns `nil`, never an error
- [ ] `scan_opportunities` receives the entire Company Context in its prompt and answers with exactly one tool call
- [ ] Each Opportunity carries a description, a rationale, `impact` (1–5) and `effort` (1–5), already ordered by the model
- [ ] The backend validates the schema and truncates the list to `opportunity_count` — ranking lives in the prompt, not in Go
- [ ] `opportunity_count = 1` yields a Single Workflow; more yields a Discovery Workflow
- [ ] `MAX_DEPTH` and `MAX_NODES_PER_RUN` enforced, and tripping either is logged as an anomaly rather than silenced
- [ ] The consultant starts a Run from the interface and sees the Nodes created
- [ ] Behavioural tests at Seam 1: a Run with the fake provider produces the expected tree
