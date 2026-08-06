# 15 — Concurrency invariants and end-to-end

**What to build:** the three operations correctness depends on stop being able to break silently, and one test walks the whole system from upload to final plan.

**Blocked by:** 11, 13

**Status:** ready-for-agent

These failures show up in no high-level test — they arrive in production as a duplicated Node or a Document missing from the Company Context. See ADR-0003.

- [ ] 20 goroutines racing for the same `pending` Node: exactly one wins, the rest get zero rows
- [ ] 10 concurrent ingestions into one Company: all 10 summaries are in the Company Context afterwards
- [ ] The same River job executed twice: children are not duplicated and the second execution reports success
- [ ] A Node abandoned past its lease returns to `pending` and completes
- [ ] The whole concurrency suite runs under the race detector and passes
- [ ] End-to-end test at Seam 1, with the fake provider: Company created, Documents uploaded, Run started with `opportunity_count` above 1, Gate answered, final plan present
- [ ] Verification that no application goroutine outlives the request or the job that spawned it
- [ ] `make test-race` runs all of this in one command
