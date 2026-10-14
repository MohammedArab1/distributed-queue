# Learning Log

## Step 1.1 — Tooling check and repository skeleton (2026-10-13)
**Built:** A git repo on `main` with a Go module (`github.com/MohammedArab1/jobqueue`), a `cmd/server` entry point that prints a startup line, a `.gitignore`, a `Makefile` (`build`/`test`/`lint`), and empty `docs/LEARNING_LOG.md` + `docs/ARCHITECTURE.md`.
**Key concepts:**
- A Go module path doubles as the import path, so the `host/user/repo` shape gives global uniqueness with no registry.
- `go.sum` is a committed record of the exact bytes each dependency was built against; the public checksum DB covers the *first* fetch, `go.sum` covers *later* changes.
- `internal/` is a compiler-enforced boundary: only code whose import path shares the module prefix may import it.
- Exported (`Enqueue`) vs unexported (`enqueue`) is Go's whole visibility system.
- `go build -o bin/ ./cmd/...` — the `...` wildcard builds every `main` package under `cmd/` into a matching binary.
- Makefile recipes must be indented with tabs; non-file targets need `.PHONY`.
**Gotchas we hit:** Git defaulted the initial branch to `master`; renamed to `main` to match the plan's conventions (the Phase 5 `buf breaking` target references `main`). Also: `go build ./...` with a single `main` package wrote a `server` executable to the repo root, and `git add -A` committed it. Fixed by untracking it and adding `/server` to `.gitignore` (recorded in PLAN.md Appendix D).
**Learner questions:**
- Why are module paths GitHub-shaped? → The path *is* the fetch location; no central registry, DNS gives uniqueness, and a real path means imports resolve if/when the repo is published.
- "I compare the downloaded checksum to go.sum" → right idea, but go.sum is *recorded automatically* at first fetch (then verified), not hand-computed and trusted.
**Interview soundbite:** "Go modules use a single path for both naming and location — `github.com/<user>/<repo>` is globally unique without a registry, and `go.sum` + the checksum database make builds reproducible and tamper-evident."

## Step 1.2 — Local infrastructure with Docker Compose (2026-10-13)
**Built:** A `docker-compose.yml` bringing up Postgres 17 (port 5433, `pg_stat_statements` preloaded, `max_connections=200`) and Redis 7 (port 6380), each with a healthcheck; `.env.example` (committed) + `.env` (ignored); Makefile targets `up`/`down`/`reset-db`/`psql`/`redis-cli` with `-include .env` + `export` at the top.
**Key concepts:**
- "Container started" ≠ "ready for connections"; healthchecks + `docker compose up --wait` close that gap.
- Named volumes survive `docker compose down`; only `down -v` destroys them.
- Named volumes are Docker-managed storage; *bind mounts* use a host path you choose. Databases prefer named volumes (permissions + portability).
- `5433:5432` avoids clashing with a local Postgres (there's a Windows Postgres 16 on this machine).
- `.env` (local, git-ignored) vs `.env.example` (committed, documents the variables).
- `-include .env` + `export` injects `.env` values into every recipe, but they *override* shell exports — so per-run overrides go after the target (`make run-server JQ_LEASE_DURATION=10s`).
- `docker compose exec <service> <cmd>` runs a command inside the *running* container (vs `run`, which starts a new one).
**Gotchas we hit:** Docker Desktop's engine was wedged — the client answered but the daemon didn't (`docker version` hung on its `Server:` section). Fixed by fully quitting and restarting Docker Desktop.
**Learner questions:**
- "Isn't a volume just a folder on my OS mapped into the container?" → that's a bind mount; a named volume is Docker-managed (you don't pick the host path).
- `docker compose exec postgres psql -U jobqueue jobqueue` breakdown → exec into the running `postgres` service, run its `psql` client as user `jobqueue` against database `jobqueue`.
**Interview soundbite:** "I run Postgres and Redis through Docker Compose with healthchecks and `up --wait`, so the dev database is reproducible — container startup and database readiness are two different things."

