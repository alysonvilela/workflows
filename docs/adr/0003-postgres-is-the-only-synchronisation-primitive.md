# Postgres is the only synchronisation primitive

A Discovery Run explores several branches at once, so the system is genuinely concurrent — but it holds no shared state in memory and contains no mutexes, channels or atomics of its own. Every synchronisation point is a row lock or a conditional write in Postgres: a Node is claimed with `UPDATE nodes SET status='running' WHERE id=$1 AND status='pending'`, the Company Context is appended under `SELECT ... FOR UPDATE`, and duplicate children are prevented by a unique index on `(parent_id, kind, ord)`. The worker pool itself belongs to River; we only supply the job body.

The alternative — an in-process orchestrator holding the run in memory with a goroutine pool — is faster to write and loses every run in flight on deploy, restart or crash. Because the database is the only place state lives, a worker can die mid-Node and the Node simply returns to `pending`, and a Run that is waiting on a human costs exactly one row and zero processes for as long as it waits.

The consequence to remember: correctness here is written in SQL, not in Go. A change that moves coordination into application memory — caching the node tree, batching writes, holding a run object across jobs — breaks the property silently, and no Go-level test will catch it.
