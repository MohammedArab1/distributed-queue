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

## Step 1.3 — Config, logging, and a server that connects to Postgres (2026-10-14)
**Built:** `internal/config` (env helpers, `Server` struct, `LoadServer` with dev defaults) with table-driven `t.Setenv` tests; `internal/logging.New(level)` returning a slog text logger; `cmd/server` rewritten with the `run()` pattern — `signal.NotifyContext`, `pgxpool` + 5s ping, clean shutdown on Ctrl-C; `make run-server` target.
**Key concepts:**
- `context.Context` is a stop signal + deadline carried *down* a call tree; `ctx.Done()` stays silent until someone cancels it or the deadline passes. Cancellation flows parent → child only, never up.
- `signal.NotifyContext` turns Ctrl-C/SIGTERM into ordinary context cancellation, so every blocking call stops through one mechanism.
- The `run()` pattern: `main` stays tiny and is the only place that calls `os.Exit`; cleanup lives in `run()`'s defers, which run LIFO (`cancel` → `pool.Close` → `stop`). `os.Exit` skips defers — proven with live demos in `/tmp/opencode/ctxdemo`.
- `%w` builds an error chain: `ping postgres: …: dial tcp …: connection refused` was three links; `errors.Is` can walk it. `%v` would keep the text but lose the chain.
- `pgxpool.New` does not connect; `Ping` opens the first connection. A 5s **child** deadline makes startup fail fast while still inheriting Ctrl-C.
- Separation of concerns: only the composition root reads env vars; libraries receive values (a pool, not a URL), so tests can point them at different databases.
**Gotchas we hit:** `make up` failed with "The command 'docker' could not be found in this WSL 2 distro" — Docker Desktop was not running and the `docker` on PATH was its Windows stub. Starting Docker Desktop restored WSL integration (`/usr/bin/docker`) and all verifications passed.
**Learner questions:**
- "Explain `context`/`NotifyContext` and the `run()` pattern in simpler terms" → re-taught with four tiny runnable demos (timeout, SIGINT, run-pattern, os.Exit-skips-defers) instead of prose; first explanation went too fast.
- "Does `pingCtx` end the program?" → it cancels only the Ping call; the early error return from `run` ends the program, not parent cancellation.
- "Why shouldn't `store` read env vars?" → separation of concerns: config is read once at the composition root; libraries receive their dependencies.
**Interview soundbite:** "Startup is a `run() error` called by a tiny `main`: `signal.NotifyContext` cancels everything on Ctrl-C, the Postgres ping gets its own 5-second child deadline so startup fails fast, and defers guarantee the pool closes before `os.Exit` is ever reached."

## Step 2.1 — Design the job lifecycle (state machine) (2026-10-22)
**Built:** `docs/ARCHITECTURE.md` "Job lifecycle" section: a mermaid state diagram with six states, a 12-row transition table (each move tagged with the step that builds it), and five invariants with how each is enforced. No code, by design: this step is the contract for later phases.
**Key concepts:**
- A state says *where the job is*, not what just happened. `fail-retry` and `replay` are triggers, not states.
- Waiting comes in two kinds: time-based (`pending` + `run_at`) and event-based (`blocked`). The first is a column, the second is a state, because no timestamp predicts another job succeeding.
- Terminal = no automatic exit. `succeeded` has no exit at all; `dead` exits only on operator replay; `cancelled` only when the upstream is replayed.
- Store the source, derive the projection: `claimable = state='pending' AND run_at <= now()` is never stored, because time passes with no write to correct a stale boolean.
- `attempts` counts deliveries, incremented atomically with the claim. That bounds crash loops: a job that kills its worker still reaches `dead`.
- An invariant is a query that must return zero rows. `CHECK` sees only the new row, so monotonic counters and claimability live in the write path.
**Design discussion:** Asked how to store the lifecycle: one table per state vs one `jobs` table with a `state` column. Learner chose one table, plus a `states` lookup table with an FK. The lookup table is defensible but adds a join and lets a state exist that no Go code handles; a `CHECK` is enough for a fixed six-value list. Learner first listed `fail-retry` and `replay` as states; corrected — a retried job waits in the same state as a new job, distinguished only by `run_at`. For a dependent whose upstream died, learner tried "stays blocked", then "reuse dead", then reached `cancelled`: a job that never ran needs a different resurrection path (replay the upstream, not the job). Learner then found its exit goes back to `blocked`.
**Gotchas we hit:** The learner proposed a stored `claimable` column. The decisive argument: nothing writes to the row when `run_at` passes, so the flag is stale by construction. Also: a `CHECK` cannot compare against the old row value, so "`attempts` never goes down" cannot be a constraint — the guard goes in the `WHERE` clause (compare-and-set).
**Learner questions:**
- "Are we competing with Kafka/RabbitMQ, or depending on them?" → we build the queue; Postgres is the engine. The comparison table is interview vocabulary and design justification, not an integration list.
- "Why is a delay needed at all, and who sets `run_at`?" → no delay by default (`now()`); producers can ask for one, and the server computes backoff. Without it, a failing job retries in a hot loop.
- "If a worker crashes, should we redo the job? How can we know?" → we cannot know whether the side effect happened, so we redeliver (at-least-once) and require idempotent handlers.
**Interview soundbite:** "A queue is a state machine in one table: six states, and every transition is a compare-and-set `UPDATE` on the expected current state. Claimability is derived from state plus `run_at`, never stored, and attempts count deliveries — so a job that crashes its worker still runs out of budget."
