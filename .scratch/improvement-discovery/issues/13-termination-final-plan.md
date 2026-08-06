# 13 — Termination and final plan

**What to build:** when no branch has work left, the Discovery Run closes itself and produces a consolidated plan — the narrative of why these specs and not others. This is the deliverable that goes to the client.

**Blocked by:** 09

**Status:** ready-for-agent

- [ ] No Node is marked final; the Run ends when no Node is `pending`, `running` or `blocked`
- [ ] On ending, the Run transitions to `synthesizing` and the orchestrator — not a strategy — creates the single `synthesize` Node
- [ ] The Run status transition is what prevents a second `synthesize`; the work-remaining check alone is not enough
- [ ] `synthesize` consolidates the surviving specs, writes `runs.result`, and the Run becomes `done`
- [ ] The final plan is readable in the interface, with the delivered specs and the justified discards
- [ ] A Run where everything was discarded still produces a plan, explaining why nothing was proposed
- [ ] Behavioural test: a complete Run produces exactly one `synthesize` and one result
- [ ] Dedicated regression test for the duplicate `synthesize`
