# Postgres coordinates Discovery Runs; goroutines stay request-scoped

Goroutines and channels are used freely for local parallelism — fanning out a multi-file upload with `errgroup`, streaming a large CSV through a parse pipeline, rate-limiting the LLM Provider with a semaphore, and the two long-lived ones in `main` (the River client and the HTTP server). River owns the worker pool that processes Nodes. None of this is discouraged.

What is forbidden is coordinating **Discovery Run or Node state in memory**. No goroutine holds a run, no channel hands a Node from a handler to a worker, no shared struct caches the node tree between jobs. Which worker owns a Node, whether a Run is finished, and whether a Gate has been answered are answered only by Postgres: the Node is claimed with `UPDATE nodes SET status='running' WHERE id=$1 AND status='pending'`, the Company Context is appended under `SELECT ... FOR UPDATE`, and duplicate children are prevented by a unique index on `(parent_id, kind, ord)`.

The test is lifetime: **a goroutine that outlives the HTTP request or the River job that spawned it is a bug.** The reason is durability, not purity — an in-process orchestrator holding runs in memory is faster to write and loses every run in flight on deploy, restart or crash. Because the run lives only in the database, a worker can die mid-Node and the Node simply returns to `pending`, and a Run waiting on a human costs one row and zero processes for as long as it waits.

The consequence to remember: this correctness is written in SQL, not in Go. A change that moves run coordination into application memory — caching the node tree, batching status writes, holding a run object across jobs — breaks it silently, and no Go-level test of a single process will catch it.
