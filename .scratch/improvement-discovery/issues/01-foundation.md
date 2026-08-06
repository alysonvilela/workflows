# 01 — Foundation: infrastructure and Go API

**What to build:** `make dev` brings up everything the project needs and the Go API answers. A developer clones the repo, copies `.env.example`, runs one command, and has Postgres, rustfs and a hot-reloading API running.

**Blocked by:** None — can start immediately.

**Status:** ready-for-agent

- [ ] `docker-compose.yaml` brings up Postgres and rustfs with a local volume, on the ports from `.env`
- [ ] `docker-compose.dokploy.yaml` covers the same services plus the API and the frontend
- [ ] Go module initialised with chi; `GET /health` reports database and object storage status
- [ ] Migrations via goose, embedded and applied at boot; the four tables (`companies`, `documents`, `runs`, `nodes`) exist with the columns and the `(parent_id, kind, ord)` unique index from the spec
- [ ] Configuration read from the single root `.env`, failing at boot with a clear message when a required variable is missing
- [ ] `make dev`, `make deps`, `make migrate` and `make tools` work; `make dev-api` reloads on save
- [ ] Graceful shutdown: `SIGINT`/`SIGTERM` stop accepting requests and drain before exiting
- [ ] `frontend/` is brought under version control (currently untracked) while `.env` stays out
