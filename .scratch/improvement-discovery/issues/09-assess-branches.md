# 09 — Parallel `assess_opportunity` branches

**What to build:** each Opportunity is investigated in its own branch, at the same time as the others, and ends either in a concrete implementation spec or in a discard with a written reason.

**Blocked by:** 08

**Status:** ready-for-agent

- [ ] `assess_opportunity` runs with the Company Context plus what its parent produced, and answers with one terminal tool call
- [ ] `specify_solution` creates a `spec_solution` Node; `discard` closes the branch as a leaf with the reason recorded
- [ ] `spec_solution` produces the concrete spec and is a leaf by contract
- [ ] Sibling branches genuinely process in parallel, bounded by `RIVER_MAX_WORKERS`
- [ ] A failing Node records its error and ends only its own branch; siblings carry on
- [ ] A Node can never spawn a child of a type its strategy does not declare
- [ ] Behavioural test with `opportunity_count = 5` and several workers: the final tree and Node count match, with no duplication
