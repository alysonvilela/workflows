# 10 — Worker recovery

**What to build:** no Discovery Run stays stuck forever because a worker died. A deploy, a crash or a hot reload in the middle of a Node returns the work to the queue instead of losing it.

**Blocked by:** 08

**Status:** ready-for-agent

Gates nothing, but the later it lands the more development time is spent with Runs stranded in `running`.

- [ ] A Node in `running` whose `claimed_at` is older than the lease returns to `pending` and is re-enqueued
- [ ] The sweeper runs periodically and records each recovery as an observable anomaly, never silently
- [ ] `riverClient.Stop` drains in-flight Nodes before the process exits; air's `kill_delay` does not sever an LLM call mid-flight
- [ ] A River retry over an already-completed Node neither duplicates children nor rewrites `output`
- [ ] Test: a Node abandoned in `running` is recovered and completes
- [ ] Test: the same job executed twice does not duplicate children, and the second execution reports success
