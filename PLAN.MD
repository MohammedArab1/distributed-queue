# Build a Distributed Job Queue in Go: A Guided Learning Plan

> Created using Anthropic Claude. Keep this note on internal versions until a human has reviewed and verified the content. Library APIs and versions change; when in doubt, check the official docs linked in each phase.

---

## How to use this document

This file is two things at once:

1. **A build plan.** An AI coding assistant follows it step by step to build a production-style distributed job queue in Go.
2. **A curriculum.** For every step it says what to teach, which problem the step solves, how to solve it, and how to walk the learner through the finished code.

Put this file at the root of the repository as `PLAN.md`, then point your AI assistant at it at the start of every session (see [0.4 Starting or resuming a session](#04-starting-or-resuming-a-session)).

### Learner profile (the AI must calibrate to this)

- **Go:** has written Go before but is rusty. Refresh syntax and idioms briefly the first time they come up. Don't assume they remember `context`, interfaces, error wrapping, embedding, or generics.
- **PostgreSQL and Docker:** comfortable. Skip basic SQL and basic `docker compose`. **Do** explain Postgres *concurrency* features (row locks, `SKIP LOCKED`, isolation levels, MVCC, advisory locks), because they're new in this context.
- **Concurrent programming:** new. This is the biggest learning goal. Explain goroutines, channels, `select`, mutexes, `sync.WaitGroup`, `context` cancellation, data races, deadlocks and goroutine leaks slowly and concretely, with runnable examples.
- **Scope:** core features only. No Kubernetes, no transactional outbox, no full chaos-engineering suite. A simple crash-recovery test is included because it proves the core at-least-once guarantee.

### Roadmap at a glance

| Phase | What gets built | The big idea you learn |
|---|---|---|
| 1 | Repo, Docker Compose, config, Go refresher | Go project structure, `context`, error wrapping |
| 2 | Job state machine + first migration | Design states before code; let the DB enforce invariants |
| 3 | Postgres storage layer | `FOR UPDATE SKIP LOCKED`, atomic claims, fencing |
| 4 | Idempotent enqueue | Idempotency keys, unique indexes, `ON CONFLICT` |
| 5 | gRPC API + CLI | Protobuf contracts, status codes, layering |
| 6 | Concurrency primer (runnable lessons) | Goroutines, channels, `select`, races, semaphores |
| 7 | Worker pool | Backpressure, bounded concurrency, panic recovery |
| 8 | Leases, heartbeats, reaper | Failure detection, at-least-once delivery, zombies |
| 9 | Graceful shutdown | Signals, draining, two-context pattern |
| 10 | Retries with backoff + jitter | Thundering herds, testable randomness |
| 11 | Dead-letter queue + replay | Poison messages, error classification |
| 12 | Priorities + delayed jobs + query-plan lab | Index design, `EXPLAIN ANALYZE`, clock skew |
| 13 | Cron jobs + leader election | Advisory locks, "never trust leader election alone" |
| 14 | DAG workflows | Topological sort, lost updates, recursive CTEs |
| 15 | Prometheus metrics + Grafana | Counters/gauges/histograms, cardinality, PromQL |
| 16 | Benchmarks + profiling | Little's Law, pprof, finding bottlenecks |
| 17 | Redis fast-scheduling layer | Dual-write problem, derived indexes, reconciliation |
| 18 | Crash test, docs, resume, interview prep | Verifying guarantees; telling the story |

---

## 0. Instructions for the AI assistant (read every session)

### 0.1 Ground rules

1. **One step at a time.** Work only on the current step from the Progress Tracker (0.5). Never start the next step until the learner says so.
2. **Follow the plan literally.** Use the exact file paths, package names, table and column names, RPC names, metric names and env vars defined here. Section 3 is the master list. Later steps depend on these names being exact.
3. **Don't add features, libraries or abstractions the step doesn't ask for.** If something in the plan looks wrong or missing, stop, explain the problem, propose a fix, and wait for approval. If approved, edit `PLAN.md` and record it in [Appendix D: Plan changes](#appendix-d-plan-changes).
4. **Never leave the build broken.** At the end of every step, `go build ./...`, `make lint` and `go test -race ./...` must pass (once tests exist). (`make lint` runs `go vet` on everything except `examples/`, which holds deliberately buggy teaching code from Phase 6.) If a test is flaky, fix the cause; don't skip the test.
5. **Small, readable code.** Prefer plain, explicit code over clever code. Aim for functions under ~50 lines. Comments explain *why*, not *what*. Every exported identifier gets a doc comment.
6. **Errors:** always handle them. Wrap with context: `fmt.Errorf("claim jobs: %w", err)`. Never `panic` in library code. Never discard an error with `_` without a comment saying why.
7. **Contexts:** every function that does I/O or can block takes `ctx context.Context` as its first parameter.
8. **Goroutines:** every goroutine must have (a) an owner, (b) a way to be told to stop (usually a context), and (c) a way for the owner to wait for it to finish (`sync.WaitGroup`, `errgroup`, or a `done` channel). In the walkthrough, say out loud who owns each goroutine and how it stops.
9. **SQL:** write SQL by hand (no ORM, no sqlc). Always use `$1`-style parameters, never string-concatenated values. Every state transition is one statement or one transaction, guarded by the expected current state.
10. **Secrets:** this project only uses local-only dev passwords from `.env.example` (`jobqueue` / `jobqueue`). Never put real credentials, tokens or keys in the repo.
11. **Failures are teaching moments.** When something fails (compile error, test failure, Docker issue), show the error, explain in plain words what it means, explain the fix, then fix it.
12. **Don't paste whole files into chat.** Write them to disk. In chat, show only the 5–30 line snippets you're explaining.
13. **Check before you guess.** If you're unsure an API exists or how it behaves, check with `go doc <pkg>.<Symbol>`, the library source, or official docs before using it. Say when you're unsure.
14. **Lost?** Re-read this file's Progress Tracker, `docs/LEARNING_LOG.md`, and `git log --oneline -20`.

### 0.2 The teaching protocol (follow it for EVERY step)

Every step in this plan has these sections: **Problem**, **Teach**, **Design decisions**, **Build**, **Verify**, **Walkthrough focus**, **Check your understanding**, **Done when**. Turn them into this conversation flow:

| Stage | What the AI does | Wait for the learner? |
|---|---|---|
| **1. FRAME** | In 3–6 sentences: what we're building and the **Problem** it solves. If the step says "Demo the problem", run it so the learner *sees* the problem. | No |
| **2. TEACH** | Teach every item in **Teach**. For each one: plain definition → analogy or tiny example → why it matters *here*. Then explain the **Design decisions**: the options, what this plan picks, and why. End with 1–2 **Predict** questions (e.g. "What do you think happens if two workers run this query at the same instant?"). | **Yes.** Ask: "Any questions before I build this?" |
| **3. BUILD** | Implement the **Build** list in order. Narrate briefly as you go ("Now the claim query, which is the heart of the queue"). | No |
| **4. VERIFY** | Run every command in **Verify**. Show the relevant output and explain what it proves. | No |
| **5. WALKTHROUGH** | Walk through what you built following the data flow ("a job enters here → goes here → ends here"). Cover each **Walkthrough focus** item. For key snippets: what it does, why it's written this way, what breaks if you remove it. Answer the Predict questions. | No |
| **6. CHECK** | Ask the **Check your understanding** questions one or two at a time. Confirm what's right, gently correct what's wrong, and point to the line of code that proves it. The learner may say "skip". | **Yes** |
| **7. WRAP UP** | Append to `docs/LEARNING_LOG.md` (template 0.3), tick the step in the Progress Tracker, commit: `git commit -m "Step X.Y: <title>"`. Then say: "Step X.Y done. Next: X.Z — <title>. Say 'next' when ready." | **Yes** |

**Teaching style**

- Assume intelligence, not knowledge. Define each term the first time you use it.
- Whenever possible, use simpler language before introducing jargon. "Simpler language" here refers to ASD-STE100 Simplified Technical English. The user learns better with this style of communication.
- Be concrete: real numbers, real SQL rows, real log lines.
- Use ASCII timelines for concurrency, e.g.:
  ```
  time →   t1                 t2                    t3
  Worker A SELECT id=7 ───────────────────────────── UPDATE id=7 → running
  Worker B          SELECT id=7 ── UPDATE id=7 → running
  Result: both workers run job 7 (double processing!)
  ```
- Where relevant, connect each concept to a real system in one sentence (Sidekiq, Celery, AWS SQS, RabbitMQ, Kafka, Temporal). This builds interview vocabulary.
- Keep each TEACH stage to roughly 300–700 words. Split big topics across turns.
- The learner can switch modes any time:
  - **"fast mode"**: one-paragraph FRAME/TEACH, skip CHECK, short WALKTHROUGH.
  - **"deep mode"**: more depth, extra examples, alternative implementations.
  - **"quiz me"**: harder, interview-style questions.

### 0.3 Learning log template

After every step, append this to `docs/LEARNING_LOG.md`:

```markdown
## Step X.Y — <title> (<YYYY-MM-DD>)
**Built:** 1–3 bullets of what exists now that didn't before.
**Key concepts:** 3–6 one-line bullets, in the learner's words where possible.
**Gotchas we hit:** errors or surprises and how we fixed them.
**Learner questions:** questions asked + short answers.
**Interview soundbite:** 1–2 sentences the learner could say in an interview about this step.
```

### 0.4 Starting or resuming a session

The learner pastes this at the start of each session:

```text
Read PLAN.md (sections 0–3 in full, then the current step), docs/LEARNING_LOG.md,
and run `git log --oneline -15`. Tell me which step we're on and summarize in
3 bullets what we've built so far. Then start the current step using the
teaching protocol in section 0.2.
```

### 0.5 Progress tracker

The AI ticks these off (`[x]`) as steps are completed.

**Phase 1 — Project setup and Go refresher**
- [x] 1.1 Tooling check and repository skeleton
- [ ] 1.2 Local infrastructure with Docker Compose
- [ ] 1.3 Config, logging, and a server that connects to Postgres

**Phase 2 — Data model**
- [ ] 2.1 Design the job lifecycle (state machine)
- [ ] 2.2 First migration: the `jobs` table

**Phase 3 — Storage layer**
- [ ] 3.1 Store package, Enqueue/GetJob, test database helper
- [ ] 3.2 Claiming jobs safely with `FOR UPDATE SKIP LOCKED`
- [ ] 3.3 Completing and failing jobs, with fencing

**Phase 4 — Idempotency**
- [ ] 4.1 Idempotent enqueue

**Phase 5 — gRPC API**
- [ ] 5.1 Protobuf contract and code generation
- [ ] 5.2 Service layer and validation
- [ ] 5.3 gRPC server and error mapping
- [ ] 5.4 `jqctl` command-line tool

**Phase 6 — Concurrency primer**
- [ ] 6.1 Goroutines, WaitGroups, channels
- [ ] 6.2 `select`, timeouts, and `context`
- [ ] 6.3 Shared memory: races, mutexes, atomics
- [ ] 6.4 Patterns and pitfalls: semaphores, errgroup, leaks, deadlocks

**Phase 7 — Worker**
- [ ] 7.1 Handler registry and demo handlers
- [ ] 7.2 Worker pool: fetch loop and semaphore
- [ ] 7.3 Worker integration tests and running it for real

**Phase 8 — Leases and at-least-once delivery**
- [ ] 8.1 Stuck jobs and the lease reaper
- [ ] 8.2 Heartbeats and lost-lease cancellation
- [ ] 8.3 Idempotent handlers and the crash demo

**Phase 9 — Graceful shutdown**
- [ ] 9.1 Worker graceful shutdown and job release
- [ ] 9.2 Server graceful shutdown and health checks

**Phase 10 — Retries**
- [ ] 10.1 Backoff package and simulation
- [ ] 10.2 Wire backoff into FailJob

**Phase 11 — Dead-letter queue**
- [ ] 11.1 Max attempts, non-retryable errors, the `dead` state
- [ ] 11.2 Admin API and CLI: list and replay

**Phase 12 — Priority and delayed jobs**
- [ ] 12.1 Priority
- [ ] 12.2 Delayed jobs
- [ ] 12.3 Query-plan lab with `EXPLAIN ANALYZE`

**Phase 13 — Cron jobs**
- [ ] 13.1 Leader election with advisory locks
- [ ] 13.2 Cron schedules table and scheduler loop
- [ ] 13.3 Cron admin API and CLI

**Phase 14 — DAG workflows**
- [ ] 14.1 DAG validation (pure Go)
- [ ] 14.2 Schema and workflow submission
- [ ] 14.3 Unblocking children when a job succeeds
- [ ] 14.4 Failure propagation and workflow replay

**Phase 15 — Observability**
- [ ] 15.1 Metrics in the server and worker
- [ ] 15.2 Prometheus, Grafana and the dashboard

**Phase 16 — Benchmarking**
- [ ] 16.1 Bench tool
- [ ] 16.2 Baseline runs, profiling and tuning

**Phase 17 — Redis fast scheduling**
- [ ] 17.1 Design: Redis as a derived index, and the dual-write problem
- [ ] 17.2 Implement the Redis index: notify, promote, fetch
- [ ] 17.3 Reconciler and failure drills
- [ ] 17.4 Re-benchmark and compare

**Phase 18 — Prove it and tell the story**
- [ ] 18.1 Crash-recovery test
- [ ] 18.2 README, architecture doc, resume bullet, interview prep

---

## 1. The big picture

### 1.1 What we're building

A **durable job queue**. Producers submit *jobs* (a type like `send_email` plus a JSON payload). The queue stores them durably in PostgreSQL. Worker processes ask for jobs, run them, and report success or failure. The system retries failures with backoff, quarantines jobs that keep failing (dead-letter queue), supports priorities, delays, cron schedules and multi-step workflows (DAGs), survives worker crashes without losing jobs, and exposes metrics.

### 1.2 Architecture

```mermaid
flowchart LR
  subgraph clients["Clients"]
    P["Producers: jqctl, your apps"]
    A["Operators: jqctl dlq / cron"]
  end
  subgraph server["jobqueue server (1..N replicas)"]
    API["gRPC API: Producer, Worker, Admin services"]
    SVC["queue.Service: business rules"]
    BG["Background loops: reaper, stats; leader-only: cron, promoter, reconciler"]
  end
  subgraph workers["worker processes (1..N)"]
    W["worker pool: fetch, run handler, heartbeat, ack"]
  end
  PG[("PostgreSQL: source of truth")]
  R[("Redis: fast ready/delayed index (Phase 17)")]
  PROM["Prometheus"] --> GRAF["Grafana"]
  P -->|Enqueue| API
  A -->|Replay, Cron| API
  W -->|Fetch, Complete, Fail, Heartbeat| API
  API --> SVC --> PG
  SVC -.-> R
  BG --> PG
  BG -.-> R
  PROM -.->|scrape /metrics| server
  PROM -.->|scrape /metrics| workers
```

### 1.3 Life of a job (the happy path and the crash path)

1. A producer calls `Enqueue`. The server inserts a row with `state = 'pending'`.
2. A worker calls `FetchJobs`. The server atomically *claims* rows: `pending → running`, `attempts + 1`, sets a **lease** (`lease_owner`, `lease_expires_at`), and returns them.
3. The worker runs the handler. While it runs, the worker sends **heartbeats** that extend the lease.
4. On success, `CompleteJob` → `succeeded`. On failure, `FailJob` → back to `pending` with a backoff delay, or to `dead` if out of attempts.
5. **Crash path:** if the worker dies, heartbeats stop, the lease expires, and the server's **reaper** puts the job back to `pending`. Another worker runs it. That's **at-least-once delivery**.

### 1.4 The laws of this codebase (design principles)

The AI should cite these by number when explaining decisions.

1. **Postgres is the source of truth.** Redis (Phase 17) is a derived index that can be rebuilt from Postgres at any time.
2. **One clock.** Every lease and schedule comparison uses Postgres `now()`, never a worker's or server's local clock.
3. **Compare-and-set transitions.** Every state change is a single SQL statement (or one transaction) whose `WHERE` clause checks the expected current state. Check rows-affected.
4. **Workers never touch the queue database.** They only talk gRPC to the server.
5. **Delivery is at-least-once. Handlers must be idempotent.**
6. **Fence every worker write** with `(job_id, worker_id, attempt)`.
7. **Every goroutine has an owner and a stop signal.** Every blocking call takes a `context.Context`.
8. **Let the database enforce invariants** with `CHECK`, `UNIQUE` and foreign-key constraints, so bugs fail loudly.

---

## 2. Tech stack

| Concern | Choice | Notes |
|---|---|---|
| Language | Go, latest stable (must be ≥ 1.24) | Check with `go version`. |
| Postgres driver | `github.com/jackc/pgx/v5` (+ `pgxpool`, `pgx/v5/stdlib` for goose) | The standard high-performance driver. |
| Migrations | `github.com/pressly/goose/v3`, used as a library; SQL files embedded with `//go:embed` | Plain SQL files. Use goose's `Provider` API (`goose.NewProvider`). Verify the API with `go doc github.com/pressly/goose/v3.NewProvider`. |
| RPC | `google.golang.org/grpc`, `google.golang.org/protobuf` | Use `grpc.NewClient` (not the deprecated `grpc.Dial`). |
| Code generation | `buf` CLI + `protoc-gen-go` + `protoc-gen-go-grpc` (local plugins) | No Buf account needed. |
| Redis client | `github.com/redis/go-redis/v9` | Phase 17. |
| Metrics | `github.com/prometheus/client_golang` | Phase 15. |
| Cron parsing | `github.com/robfig/cron/v3` (parser only) | Widely used but unmaintained since 2020; we only call `ParseStandard` and `Schedule.Next`, wrapped in `recover`. `github.com/netresearch/go-cron` is an API-compatible maintained fork if you prefer. |
| Concurrency helpers | `golang.org/x/sync/errgroup` | |
| Logging | `log/slog` (standard library) | Text handler in dev. |
| CLI parsing | standard library `flag` | No cobra. |
| Tests | standard `testing` + `github.com/stretchr/testify/require`; **real Postgres/Redis** from Docker Compose; `google.golang.org/grpc/test/bufconn` for in-memory gRPC | No database mocks: queue behaviour *is* database behaviour. |
| Infra | Docker Compose: `postgres:17`, `redis:7`, `prom/prometheus`, `grafana/grafana` | `postgres:17` is pinned because the `postgres:18` image changed its data-directory layout. |
| Manual tools | `psql` (via `docker compose exec`), `redis-cli`, `grpcurl`, `jq` | |

---

## 3. Canonical names (the AI MUST use these exactly)

### 3.1 Repository layout (final state; created gradually)

```text
jobqueue/
├── PLAN.md                         # this file
├── README.md                       # Phase 18
├── Makefile
├── docker-compose.yml
├── .env.example                    # committed; .env is git-ignored
├── .gitignore
├── buf.yaml
├── buf.gen.yaml
├── go.mod / go.sum
├── proto/jobqueue/v1/jobqueue.proto
├── gen/jobqueue/v1/                # generated by `make proto`; committed
├── cmd/
│   ├── server/main.go              # gRPC server + background loops
│   ├── worker/main.go              # worker process
│   ├── migrate/main.go             # runs DB migrations (up/down/status)
│   ├── jqctl/main.go               # CLI for producers and operators
│   └── bench/main.go               # load generator (Phase 16)
├── internal/
│   ├── config/                     # env-var config loading
│   ├── logging/                    # slog setup
│   ├── store/                      # ALL SQL lives here
│   │   └── migrations/             # 0001_*.sql ... (embedded)
│   ├── queue/                      # Service: business rules, validation, background loops
│   ├── api/                        # gRPC handlers: proto <-> domain, error mapping, interceptors
│   ├── worker/                     # worker pool, heartbeats, shutdown
│   ├── handlers/                   # demo job handlers
│   ├── backoff/                    # retry delay policy
│   ├── leader/                     # advisory-lock leader election
│   ├── cronsched/                  # cron parsing + scheduler loop
│   ├── dag/                        # workflow graph validation (pure Go)
│   ├── metrics/                    # Prometheus metric definitions + /metrics server
│   ├── redisq/                     # Redis ready/delayed index (Phase 17)
│   └── testutil/                   # test helpers (test DB, test server, test Redis)
├── examples/
│   ├── concurrency/NN_name/main.go # Phase 6 runnable lessons
│   ├── backoff_sim/main.go         # Phase 10
│   └── workflows/diamond.json      # Phase 14
├── deploy/
│   ├── prometheus/prometheus.yml
│   ├── prometheus/alerts.yml
│   └── grafana/
│       ├── provisioning/datasources/prometheus.yml
│       ├── provisioning/dashboards/dashboards.yml
│       └── dashboards/jobqueue.json
├── scripts/
│   ├── sql/explain_lab.sql         # Phase 12
│   ├── bench_matrix.sh             # Phase 16
│   └── crash_test.sh               # Phase 18
└── docs/
    ├── LEARNING_LOG.md
    ├── ARCHITECTURE.md
    ├── BENCHMARKS.md
    └── INTERVIEW_PREP.md
```

**Dependency direction (no cycles):** `cmd/*` → `api` → `queue` → `store`. `worker` depends only on the generated gRPC client, never on `store` or `queue`. `testutil` may import anything, so tests that use it must live in **external test packages** (`package store_test`, not `package store`) to avoid import cycles.

### 3.2 Environment variables

| Variable | Default | Used by | Introduced |
|---|---|---|---|
| `JQ_DATABASE_URL` | `postgres://jobqueue:jobqueue@localhost:5433/jobqueue?sslmode=disable` | server, migrate, worker (demo handler only), bench | 1.3 |
| `JQ_TEST_DATABASE_URL` | `postgres://jobqueue:jobqueue@localhost:5433/postgres?sslmode=disable` | tests (admin connection used to create per-test DBs) | 3.1 |
| `JQ_LOG_LEVEL` | `info` | all | 1.3 |
| `JQ_GRPC_LISTEN_ADDR` | `:50051` | server | 5.3 |
| `JQ_SERVER_ADDR` | `localhost:50051` | worker, jqctl, bench | 5.4 |
| `JQ_LEASE_DURATION` | `30s` | server | 5.2 |
| `JQ_REAPER_INTERVAL` | `5s` | server | 8.1 |
| `JQ_WORKER_ID` | `<hostname>-<pid>` | worker | 7.2 |
| `JQ_WORKER_QUEUES` | `default` (comma-separated, in priority order) | worker | 7.2 |
| `JQ_WORKER_CONCURRENCY` | `10` | worker | 7.2 |
| `JQ_WORKER_SHUTDOWN_TIMEOUT` | `25s` | worker | 9.1 |
| `JQ_SERVER_SHUTDOWN_TIMEOUT` | `10s` | server | 9.2 |
| `JQ_BACKOFF_BASE` | `1s` | server | 10.2 |
| `JQ_BACKOFF_MAX` | `10m` | server | 10.2 |
| `JQ_METRICS_ADDR` | server `:2112`, worker `:2113` | server, worker | 15.1 |
| `JQ_REDIS_ADDR` | `localhost:6380` | server | 17.2 |
| `JQ_SCHEDULER_MODE` | `postgres` (`postgres` or `redis`) | server | 17.2 |

### 3.3 Ports

| Port | What |
|---|---|
| 5433 → 5432 | Postgres (host 5433 avoids clashing with a locally installed Postgres) |
| 6380 → 6379 | Redis |
| 50051 | gRPC server (second replica for experiments: 50052) |
| 2112 | server `/metrics` and `/debug/pprof` (second replica: 2122) |
| 2113–2116 | worker `/metrics` (one port per local worker process) |
| 9090 | Prometheus UI |
| 3000 | Grafana UI |

### 3.4 Database

**Migrations (never edit a migration after it has been applied; add a new one):**

| File | Step | Purpose |
|---|---|---|
| `0001_create_jobs.sql` | 2.2 | `jobs` table, ready index, invariant checks |
| `0002_add_idempotency_key.sql` | 4.1 | `idempotency_key` + partial unique index |
| `0003_add_running_lease_index.sql` | 8.1 | index for the reaper |
| `0004_create_demo_effects.sql` | 8.3 | table simulating an external system (idempotency demo) |
| `0005_add_state_queue_index.sql` | 11.2 | index for listing/replaying by state |
| `0006_add_priority.sql` | 12.1 | `priority` column, rebuilt ready index |
| `0007_create_cron_schedules.sql` | 13.2 | cron schedules |
| `0008_create_workflows.sql` | 14.2 | workflows, dependency edges, counters |

**Final `jobs` columns:** `id`, `queue`, `type`, `payload`, `state`, `priority`, `run_at`, `attempts`, `max_attempts`, `last_error`, `lease_owner`, `lease_expires_at`, `idempotency_key`, `workflow_id`, `workflow_node_key`, `pending_dependencies`, `created_at`, `updated_at`, `completed_at`.

**Other tables:** `demo_effects`, `cron_schedules`, `workflows`, `job_dependencies`.

**Job states** (Go constants in `store`: `StateBlocked`, `StatePending`, `StateRunning`, `StateSucceeded`, `StateDead`, `StateCancelled`): `blocked`, `pending`, `running`, `succeeded`, `dead`, `cancelled`.

### 3.5 gRPC surface (package `jobqueue.v1`)

| Service | RPC | Step |
|---|---|---|
| `ProducerService` | `Enqueue`, `GetJob` | 5.1 |
| `ProducerService` | `SubmitWorkflow`, `GetWorkflow` | 14.2 |
| `WorkerService` | `FetchJobs`, `CompleteJob`, `FailJob` | 5.1 |
| `WorkerService` | `Heartbeat` | 8.2 |
| `WorkerService` | `ReleaseJob` | 9.1 |
| `AdminService` | `ListJobs`, `ReplayJob`, `ReplayDeadJobs` | 11.2 |
| `AdminService` | `UpsertCronSchedule`, `ListCronSchedules`, `DeleteCronSchedule` | 13.3 |
| `grpc.health.v1.Health` | standard health service | 9.2 |

**Field-number plan** (never reuse or renumber):

| Message | Fields |
|---|---|
| `Job` | 1–12 (5.1); `priority = 13` (12.1); `workflow_id = 14`, `workflow_node_key = 15` (14.2) |
| `EnqueueRequest` | 1–5 (5.1); `priority = 6` (12.1); `oneof schedule { run_at = 7; delay = 8; }` (12.2) |
| `FetchJobsResponse` | `jobs = 1` (5.1); `lease_duration = 2` (8.2) |
| `FailJobRequest` | 1–4 (5.1); `non_retryable = 5` (11.1) |
| `FailJobResponse` | `new_state = 1` (5.1); `next_run_at = 2` (10.2) |

### 3.6 Errors

- `store.ErrNotFound`: row doesn't exist → gRPC `NotFound`.
- `store.ErrLeaseLost`: fenced update matched no row because the caller no longer owns the job → gRPC `FailedPrecondition`.
- `store.ErrInvalidState`: operation not allowed in the job's current state (e.g. replaying a job that isn't dead) → gRPC `FailedPrecondition`.
- `queue.ValidationError{Field, Reason}`: bad input → gRPC `InvalidArgument`.
- Anything else → log it in full, return gRPC `Internal` with a generic message.

### 3.7 Makefile targets (final)

`up`, `down`, `reset-db`, `psql`, `redis-cli`, `migrate-up`, `migrate-down`, `migrate-status`, `proto`, `proto-breaking`, `build`, `test`, `lint`, `run-server`, `run-worker`.

---
## Phase 1 — Project setup and Go refresher

### Step 1.1 — Tooling check and repository skeleton

**Problem.** Before writing any queue logic we need a reproducible workspace: a Go module, a git repo, and the tools to build, test and (later) generate code. A messy start causes confusing errors later.

**Teach.**
- Go modules: `go.mod` (module path, Go version, dependencies) and `go.sum` (checksums of every dependency, which protect against tampered downloads). Why `go.sum` is committed.
- Packages: one directory = one package. A package named `main` with `func main()` builds into an executable. The `cmd/<name>/` convention lets one repo hold several binaries.
- `internal/`: the compiler forbids other modules from importing anything under `internal/`. It's a real boundary, not just a convention.
- Exported vs unexported names (capitalized `Enqueue` vs lowercase `enqueue`).
- The everyday commands: `go build`, `go run`, `go test`, `go vet`, `go mod tidy`.
- Makefiles in one minute: targets, recipes, `.PHONY`, and the classic trap that recipes must be indented with a **tab**.

**Design decisions.**
- One repo with several binaries in `cmd/` (vs. one repo per service): shared code, atomic changes, simpler for one person.
- Module path `github.com/<github-username>/jobqueue`. **Ask the learner for their GitHub username.** If they don't want to use one, use `example.com/jobqueue`.

**Build.**
1. Check tools: `go version` (≥ 1.24; if older, ask the learner to run `brew upgrade go` or install from go.dev), `docker --version`, `docker compose version`, `git --version`, `make --version`. Install the extras on macOS: `brew install bufbuild/buf/buf grpcurl jq`. Confirm `$(go env GOPATH)/bin` is on `PATH` (needed in Phase 5); if not, tell the learner which line to add to `~/.zshrc`.
2. `git init`; create `.gitignore` with: `bin/`, `.env`, `*.out`, `*.prof`, `*.test`, `coverage.*`, `.DS_Store`, `tmp/`.
3. `go mod init github.com/<username>/jobqueue`.
4. Create `cmd/server/main.go` that just prints `jobqueue server starting` (replaced in 1.3).
5. Create `docs/LEARNING_LOG.md` (title only) and `docs/ARCHITECTURE.md` (title + "Filled in as we go").
6. Create a `Makefile` with:
   ```make
   .PHONY: build test lint
   build:
   	go build -o bin/ ./cmd/...
   test:
   	go test -race ./...
   lint:
   	go vet $$(go list ./... | grep -v /examples/)
   ```
   (`examples/` is excluded from vet because Phase 6 puts deliberately buggy demo code there; `$$` escapes `$` inside a Makefile.)
7. First commit: `git add -A && git commit -m "Step 1.1: repository skeleton"`.

**Verify.** `go build ./...`; `go run ./cmd/server`; `make build && ls bin/`; `git log --oneline`.

**Walkthrough focus.** What each line of `go.mod` means. Why `internal/` matters for a project like this. What `go build -o bin/ ./cmd/...` does (the `...` wildcard).

**Check your understanding.**
1. What's the difference between `go run` and `go build`?
2. If another repo tries to import `github.com/<you>/jobqueue/internal/store`, what happens and why?
3. What does `go.sum` protect you from?

**Done when.** The repo builds and has its first commit.

---

### Step 1.2 — Local infrastructure with Docker Compose

**Problem.** The queue needs Postgres (and later Redis) running locally, identically every time, with one command.

**Teach** (brief; the learner knows Docker):
- Healthchecks: "container started" ≠ "database ready to accept connections". `docker compose up --wait` blocks until healthchecks pass.
- Named volumes: data survives `docker compose down`; `down -v` wipes it.
- Host port mapping `5433:5432` avoids clashing with a Postgres already installed on the laptop.
- `.env` (git-ignored, your local values) vs `.env.example` (committed, documents the variables).
- `pg_stat_statements` is preloaded now because Phase 16 uses it to find slow queries.

**Build.**
1. `docker-compose.yml`:
   ```yaml
   services:
     postgres:
       image: postgres:17
       environment:
         POSTGRES_USER: jobqueue
         POSTGRES_PASSWORD: jobqueue   # local development only
         POSTGRES_DB: jobqueue
       ports:
         - "5433:5432"
       volumes:
         - pgdata:/var/lib/postgresql/data
       command: ["postgres", "-c", "max_connections=200", "-c", "shared_preload_libraries=pg_stat_statements"]
       healthcheck:
         test: ["CMD-SHELL", "pg_isready -h 127.0.0.1 -U jobqueue -d jobqueue"]  # -h: during first-time init Postgres listens only on a socket
         interval: 2s
         timeout: 3s
         retries: 30

     redis:
       image: redis:7
       ports:
         - "6380:6379"
       healthcheck:
         test: ["CMD", "redis-cli", "ping"]
         interval: 2s
         timeout: 3s
         retries: 30

   volumes:
     pgdata:
   ```
2. `.env.example` with every variable from Section 3.2 that exists so far (`JQ_DATABASE_URL`, `JQ_LOG_LEVEL`) and a comment that values are local-only. Copy it: `cp .env.example .env`. Add new variables to `.env.example` whenever a later step introduces them.
3. Extend the `Makefile` (put the first two lines at the very top):
   ```make
   -include .env
   export

   .PHONY: up down reset-db psql redis-cli
   up:
   	docker compose up -d --wait
   down:
   	docker compose down
   reset-db:
   	docker compose down -v && docker compose up -d --wait
   psql:
   	docker compose exec postgres psql -U jobqueue jobqueue
   redis-cli:
   	docker compose exec redis redis-cli
   ```
   (`reset-db` will also run migrations once they exist in 2.2.)

   **Gotcha to teach now:** with `-include .env` + `export`, values in `.env` **override** variables set in your shell, so `JQ_LEASE_DURATION=10s make run-server` is silently ignored. Put overrides **after** the target instead: `make run-server JQ_LEASE_DURATION=10s` (command-line variables beat the Makefile and are still exported). Every demo in this plan uses that form.

**Verify.** `make up`; `docker compose ps` shows both services `healthy`; `make psql` then `SELECT version();`; `make redis-cli` then `PING` → `PONG`.

**Walkthrough focus.** Each block of the compose file. What `-include .env` + `export` does in the Makefile (every recipe sees those env vars).

**Check your understanding.**
1. Why map host port 5433 instead of 5432?
2. What happens to your data after `docker compose down`? After `docker compose down -v`?

**Done when.** `make up` brings up healthy Postgres and Redis.

---

### Step 1.3 — Config, logging, and a server that connects to Postgres

**Problem.** Every binary needs settings (database URL, ports, timeouts) and readable logs. Hard-coding settings breaks the moment you run in a different environment. This step is also the Go refresher.

**Teach.**
- **Go refresher, using this step's code:** structs and methods; pointer vs value receivers; multiple return values `(T, error)`; `if err != nil` flow; wrapping errors with `fmt.Errorf("...: %w", err)` and inspecting them with `errors.Is` / `errors.As`; `defer` (runs when the function returns, LIFO order).
- **`context.Context`** basics: it carries cancellation and deadlines *down* a call tree. Cancelling a parent cancels all children. Functions that block should accept one and stop when `ctx.Done()` closes.
- **`signal.NotifyContext`**: turns Ctrl-C / SIGTERM into a cancelled context.
- **12-factor config:** settings come from environment variables; the code has sensible defaults.
- **Structured logging with `log/slog`:** key/value attributes (`logger.Info("claimed", "job_id", 7)`) instead of formatted strings, so logs can be searched and parsed.
- **Connection pools:** opening a Postgres connection is expensive (TCP, auth, and a new backend process on the server), so we keep a pool and reuse connections. `pgxpool` basics; `MaxConns`.
- **"main is the composition root":** `main` reads config and wires dependencies together. Library packages never read env vars themselves; they receive values.
- **The `run()` pattern:** `main` calls `run()` and exits non-zero on error. This keeps `defer`s working (`os.Exit` skips deferred calls) and makes startup testable.

**Build.**
1. `internal/config/config.go`:
   - Helpers: `getString(key, def string) string`, `getInt(key string, def int) (int, error)`, `getDuration(key string, def time.Duration) (time.Duration, error)`, `getLogLevel(key string, def slog.Level) (slog.Level, error)`.
   - `type Server struct { DatabaseURL string; LogLevel slog.Level }` and `func LoadServer() (Server, error)`. Later steps add fields to this struct (and add `Worker`, `Client` structs) as variables are introduced in Section 3.2.
   - Return clear errors: `fmt.Errorf("JQ_LEASE_DURATION: %w", err)`.
2. `internal/config/config_test.go`: table-driven tests using `t.Setenv` (defaults used when unset; valid override; invalid duration → error).
3. `internal/logging/logging.go`: `func New(level slog.Level) *slog.Logger` using `slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})`.
4. `cmd/server/main.go`:
   ```go
   func main() {
       if err := run(); err != nil {
           fmt.Fprintln(os.Stderr, "error:", err)
           os.Exit(1)
       }
   }

   func run() error {
       ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
       defer stop()

       cfg, err := config.LoadServer()
       if err != nil {
           return fmt.Errorf("load config: %w", err)
       }
       logger := logging.New(cfg.LogLevel)

       pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
       if err != nil {
           return fmt.Errorf("create pg pool: %w", err)
       }
       defer pool.Close()

       pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
       defer cancel()
       if err := pool.Ping(pingCtx); err != nil {
           return fmt.Errorf("ping postgres: %w", err)
       }
       logger.Info("connected to postgres")

       <-ctx.Done() // wait for Ctrl-C
       logger.Info("shutting down")
       return nil
   }
   ```
5. `go get github.com/jackc/pgx/v5 && go mod tidy`.
6. Add `run-server` to the Makefile: `go run ./cmd/server`.

**Verify.**
- `make up && make run-server` → logs "connected to postgres"; Ctrl-C → "shutting down".
- `docker compose stop postgres && make run-server` → a clear wrapped error like `error: ping postgres: ...`. Restart with `make up`.
- `make test`.

**Walkthrough focus.** The `run()` pattern and why `defer pool.Close()` would be skipped by `os.Exit`. How `ctx` flows from `NotifyContext` into `Ping`. How `%w` builds the error chain you saw in the output.

**Check your understanding.**
1. Why does `run()` return an error instead of calling `log.Fatal` itself?
2. What does passing `pingCtx` to `pool.Ping` buy us?
3. Why shouldn't `internal/store` call `os.Getenv` directly?

**Done when.** The server connects to Postgres, shuts down cleanly on Ctrl-C, and config tests pass.

---

## Phase 2 — Data model

### Step 2.1 — Design the job lifecycle (state machine)

**Problem.** A queue is a state machine. Most queue bugs are illegal or racy state transitions. We design every state and transition on paper before writing SQL. No code in this step.

**Teach.**
- What a job queue is for: decoupling (the web request returns fast while the work runs later), smoothing load spikes, retries, scheduling.
- **Why Postgres as a queue:** ACID durability, *transactional enqueue* (insert the job in the same transaction as your business data, so you never get "order saved but email job lost"), one less system to run. Limits: a throughput ceiling, and table bloat from constant updates (MVCC; covered in 2.2).
- Where other systems sit: Redis-based queues (Sidekiq), brokers (RabbitMQ, SQS), logs (Kafka), workflow engines (Temporal). One sentence each.
- State machines: states, transitions, terminal states, and **invariants** (facts that must always be true).

**Design decisions.** We keep every state in one `jobs` table with a `state` column (vs. a table per state). Moving between states is then a single atomic `UPDATE`.

**Build.** Add a "Job lifecycle" section to `docs/ARCHITECTURE.md` containing this diagram, the transition table, and the invariants.

```mermaid
stateDiagram-v2
  [*] --> pending: Enqueue
  [*] --> blocked: SubmitWorkflow (has dependencies)
  blocked --> pending: last dependency succeeded
  blocked --> cancelled: a dependency died
  pending --> running: claimed by a worker (lease set)
  running --> succeeded: CompleteJob
  running --> pending: FailJob with retries left / lease expired / ReleaseJob
  running --> dead: FailJob with no retries left or non-retryable / lease expired on last attempt
  dead --> pending: ReplayJob
  cancelled --> blocked: upstream job replayed
  succeeded --> [*]
```

| From | To | Trigger | Built in |
|---|---|---|---|
| (new) | `pending` | `Enqueue` | 3.1 |
| (new) | `blocked` | `SubmitWorkflow`, node has dependencies | 14.2 |
| `blocked` | `pending` | last dependency succeeded | 14.3 |
| `blocked` | `cancelled` | an upstream job became `dead` | 14.4 |
| `pending` | `running` | `ClaimJobs` (worker fetch) | 3.2 |
| `running` | `succeeded` | `CompleteJob` | 3.3 |
| `running` | `pending` | `FailJob` with retries left | 3.3, 10.2 |
| `running` | `pending` | lease expired (reaper) | 8.1 |
| `running` | `pending` | `ReleaseJob` (graceful shutdown) | 9.1 |
| `running` | `dead` | `FailJob` out of attempts or non-retryable; reaper on last attempt | 11.1 |
| `dead` | `pending` | `ReplayJob` | 11.2 |
| `cancelled` | `blocked` | upstream job replayed | 14.4 |

**Invariants** (most are enforced by `CHECK` constraints):
1. `state = 'running'` ⇔ `lease_owner` and `lease_expires_at` are both non-NULL.
2. `completed_at` is non-NULL ⇔ `state` is terminal (`succeeded`, `dead`, `cancelled`).
3. `attempts` only goes up, except `ReleaseJob` (−1, because a shutdown isn't the job's fault) and `ReplayJob` (reset to 0).
4. A job is claimable ⇔ `state = 'pending' AND run_at <= now()`.
5. `state = 'blocked'` ⇔ `pending_dependencies > 0` (Phase 14).

**Verify.** The AI asks the learner to trace three scenarios through the diagram, naming each transition: (a) a job fails twice, then succeeds; (b) a worker crashes mid-job; (c) a job with `max_attempts = 1` fails.

**Walkthrough focus.** Why `attempts` counts *deliveries* (it goes up when a job is claimed, not when it fails). This is explained fully in 3.2; preview it here.

**Check your understanding.**
1. Which states are terminal?
2. Why can't a job go straight from `pending` to `succeeded`?
3. Which invariant would catch a bug where `CompleteJob` forgets to clear `lease_owner`?

**Done when.** `docs/ARCHITECTURE.md` contains the diagram, table and invariants, and the learner can trace the scenarios.

---

### Step 2.2 — First migration: the `jobs` table

**Problem.** The schema will change many times over this project. We need versioned, repeatable migrations, and a table designed for a hot, constantly updated workload.

**Teach.**
- Migrations: numbered files with an *up* and a *down*; a version table (`goose_db_version`) records what has run. Never edit an applied migration; write a new one.
- Column choices:
  - `BIGINT GENERATED ALWAYS AS IDENTITY` vs UUID: identity IDs are small, ordered and readable in logs; UUIDs suit IDs generated by many machines. One Postgres generates our IDs, so identity is fine.
  - `TEXT` + `CHECK` instead of a Postgres `ENUM`: adding values to an enum is awkward inside migrations; a `CHECK` is easy to change.
  - `JSONB` for payloads; `TIMESTAMPTZ` always (never `TIMESTAMP` without time zone).
- **Partial indexes:** `... WHERE state = 'pending'` indexes only claimable rows. The index stays small and hot even with millions of finished jobs. Column order `(queue, run_at, id)` matches how we filter and sort when claiming (3.2).
- **`CHECK` constraints as invariant guards** (Law 8): if any later code forgets to clear the lease, Postgres rejects the write and the bug surfaces immediately.
- **MVCC in one paragraph:** every `UPDATE` writes a new row version and leaves a dead one behind; autovacuum cleans up. Queue tables update constantly, so long-running transactions (which block cleanup) are especially harmful. Reading: Brandur's "Postgres Job Queues & Failure By MVCC" (see Appendix C).

**Design decisions.** Goose as a *library* with migrations embedded in the binary (`//go:embed`), run via `cmd/migrate`. Tests reuse the same function, so tests always run against the real schema.

**Build.**
1. `internal/store/migrations/0001_create_jobs.sql`:
   ```sql
   -- +goose Up
   CREATE TABLE jobs (
       id               BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
       queue            TEXT        NOT NULL,
       type             TEXT        NOT NULL,
       payload          JSONB       NOT NULL DEFAULT '{}'::jsonb,
       state            TEXT        NOT NULL DEFAULT 'pending',
       run_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
       attempts         INT         NOT NULL DEFAULT 0,
       max_attempts     INT         NOT NULL DEFAULT 5,
       last_error       TEXT,
       lease_owner      TEXT,
       lease_expires_at TIMESTAMPTZ,
       created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
       updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
       completed_at     TIMESTAMPTZ,

       CONSTRAINT jobs_state_check
           CHECK (state IN ('blocked', 'pending', 'running', 'succeeded', 'dead', 'cancelled')),
       CONSTRAINT jobs_attempts_check     CHECK (attempts >= 0),
       CONSTRAINT jobs_max_attempts_check CHECK (max_attempts >= 1),
       -- Invariant 1: a job has a lease if and only if it is running.
       CONSTRAINT jobs_lease_check
           CHECK ((state = 'running') = (lease_owner IS NOT NULL AND lease_expires_at IS NOT NULL)),
       -- Invariant 2: completed_at is set if and only if the job is in a terminal state.
       CONSTRAINT jobs_completed_check
           CHECK ((state IN ('succeeded', 'dead', 'cancelled')) = (completed_at IS NOT NULL))
   );

   -- The "ready" index: only pending rows, in the order workers should take them.
   CREATE INDEX jobs_ready_idx ON jobs (queue, run_at, id) WHERE state = 'pending';

   -- +goose Down
   DROP TABLE jobs;
   ```
2. `internal/store/migrate.go`:
   - `//go:embed migrations/*.sql` into `var migrationsFS embed.FS`.
   - `func Migrate(ctx context.Context, databaseURL, command string) error` where `command` is `up`, `down` or `status`. Open a `database/sql` DB with the pgx stdlib driver (`import _ "github.com/jackc/pgx/v5/stdlib"`, driver name `"pgx"`), take `fs.Sub(migrationsFS, "migrations")`, create a goose `Provider` for Postgres, and call `Up`, `Down` or `Status`. For `status`, print each migration's version, name and state. Close the DB when done.
   - Before writing it, run `go doc github.com/pressly/goose/v3.NewProvider` and `go doc github.com/pressly/goose/v3.Provider` to confirm the exact method signatures.
3. `cmd/migrate/main.go`: `migrate up|down|status`, reads `JQ_DATABASE_URL`, uses the `run()` pattern.
4. Makefile: `migrate-up`, `migrate-down`, `migrate-status` (`go run ./cmd/migrate <cmd>`); change `reset-db` to also run `migrate-up`.

**Verify.**
- `make migrate-up`, then `make psql` → `\d jobs` shows columns, constraints and the partial index.
- `make migrate-down && make migrate-up` (the round trip works).
- `make migrate-status`.
- In psql, try breaking an invariant: `INSERT INTO jobs (queue, type, state) VALUES ('q', 't', 'running');` → fails on `jobs_lease_check`. Explain the error.

**Walkthrough focus.** Every column and why it exists. The two invariant `CHECK`s. Why the index is partial and why its column order matters.

**Check your understanding.**
1. Why a partial index rather than an index on all rows?
2. Why `TEXT` + `CHECK` instead of `ENUM`?
3. What goes wrong if you edit `0001` after it has already run on someone's database?

**Done when.** Migrations run up and down cleanly and the invariant checks reject bad rows.

---

## Phase 3 — Storage layer: the queue in Postgres

### Step 3.1 — Store package, Enqueue/GetJob, test database helper

**Problem.** We need one place where all SQL lives, a typed Go API over it, and tests that run against a real Postgres. Queue correctness depends on database locking behaviour that mocks can't reproduce.

**Teach.**
- The store (repository) pattern: all SQL in one package, so the rest of the code calls `store.Enqueue(...)` and never builds SQL.
- pgx basics: `QueryRow(...).Scan(...)`, `Query` + `rows.Next()`, `pgx.ErrNoRows`.
- Nullable columns map to pointers (`*string`, `*time.Time`); `nil` means SQL `NULL`.
- One shared column list plus one `scanJob` function, so adding a column later is a two-line change.
- **Go interfaces are implicit.** `DBTX` is any type with `Exec`, `Query` and `QueryRow`. Both `*pgxpool.Pool` and `pgx.Tx` satisfy it, so the same store code runs inside or outside a transaction.
- Integration tests vs mocks; test isolation with a fresh database per test; `t.Helper()`, `t.Cleanup()`.
- **External test packages** (`package store_test`): needed because `testutil` imports `store`. A test in `package store` that imports `testutil` would create an import cycle.

**Design decisions.** One freshly created and migrated database per test: simple, fully isolated, and fast enough (~100 ms). Optimization to mention but not build: template databases.

**Build.**
1. `internal/store/store.go`:
   ```go
   // DBTX is satisfied by *pgxpool.Pool and pgx.Tx.
   type DBTX interface {
       Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
       Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
       QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
   }

   type Store struct {
       db   DBTX
       pool *pgxpool.Pool
       inTx bool
   }

   func New(pool *pgxpool.Pool) *Store { return &Store{db: pool, pool: pool} }

   // InTx runs fn inside a transaction. If s is already in a transaction, fn joins it.
   func (s *Store) InTx(ctx context.Context, fn func(tx *Store) error) error {
       if s.inTx {
           return fn(s)
       }
       return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
           return fn(&Store{db: tx, pool: s.pool, inTx: true})
       })
   }

   var (
       ErrNotFound     = errors.New("not found")
       ErrLeaseLost    = errors.New("lease lost")
       ErrInvalidState = errors.New("invalid state for this operation")
   )
   ```
2. `internal/store/jobs.go`:
   - `type State string` with constants `StateBlocked`, `StatePending`, `StateRunning`, `StateSucceeded`, `StateDead`, `StateCancelled`.
   - `type Job struct { ID int64; Queue, Type string; Payload []byte; State State; RunAt time.Time; Attempts, MaxAttempts int; LastError, LeaseOwner *string; LeaseExpiresAt *time.Time; CreatedAt, UpdatedAt time.Time; CompletedAt *time.Time }`. Later steps add fields.
   - `var jobColumnNames = []string{"id", "queue", "type", "payload", "state", "run_at", "attempts", "max_attempts", "last_error", "lease_owner", "lease_expires_at", "created_at", "updated_at", "completed_at"}` and `func jobColumns(prefix string) string` (returns `"j.id, j.queue, ..."` when prefix is `"j."`).
   - `func scanJob(row pgx.Row) (Job, error)`, used for both `QueryRow` results and each row from `Query`. Add `func scanJobs(rows pgx.Rows) ([]Job, error)`.
   - `type EnqueueParams struct { Queue, Type string; Payload []byte; MaxAttempts int }`.
   - `func (s *Store) Enqueue(ctx context.Context, p EnqueueParams) (Job, error)`. The store applies safe defaults itself, so zero values never violate a constraint:
     ```sql
     INSERT INTO jobs (queue, type, payload, max_attempts)
     VALUES ($1, $2, COALESCE($3::jsonb, '{}'::jsonb), COALESCE(NULLIF($4::int, 0), 5))
     RETURNING <jobColumns("")>
     ```
     (A nil `Payload` becomes `{}`; `MaxAttempts: 0` becomes 5. Keep this `COALESCE` pattern whenever later steps extend this INSERT.)
   - `func (s *Store) GetJob(ctx context.Context, id int64) (Job, error)`: returns `ErrNotFound` on `pgx.ErrNoRows` (use `errors.Is`).
3. `internal/testutil/postgres.go`: `func NewTestDB(t *testing.T) *pgxpool.Pool`:
   - Connect to `JQ_TEST_DATABASE_URL` (default in Section 3.2). On failure, `t.Fatalf` with a hint: "is Postgres running? try `make up`".
   - Create a database named `jq_test_<16 random hex chars>`.
   - Build that database's URL by swapping the path with `net/url`, run `store.Migrate(ctx, url, "up")`, open a `pgxpool` to it.
   - `t.Cleanup`: close the pool, then `DROP DATABASE <name> WITH (FORCE)`. If env `JQ_KEEP_TEST_DB=1`, skip the drop and log the name (handy for debugging).
4. `internal/store/jobs_test.go` (`package store_test`): Enqueue→GetJob round trip; defaults (`pending`, `attempts = 0`, `max_attempts = 5`); payload JSON round trip; `GetJob(999999)` → `ErrNotFound` via `require.ErrorIs`.
5. `go get github.com/stretchr/testify && go mod tidy`.

**Verify.** `make test`. Then run one test with `JQ_KEEP_TEST_DB=1 go test ./internal/store -run TestEnqueue -v`, list databases in psql with `\l`, and inspect the kept DB. Drop it afterwards.

**Walkthrough focus.** `DBTX` and how `InTx` hands a transaction-backed `Store` to the callback. The `jobColumns`/`scanJob` pairing. Why `package store_test`.

**Check your understanding.**
1. Why use pointers for nullable columns?
2. What does the `DBTX` interface let us do that a concrete `*pgxpool.Pool` field wouldn't?
3. Why do the tests use a real database instead of a mock?

**Done when.** Enqueue/GetJob work and are tested against a real, isolated database.

---

### Step 3.2 — Claiming jobs safely with `FOR UPDATE SKIP LOCKED`

**Problem.** Many workers ask for work at the same moment. Two of them must never get the same job, and they shouldn't wait on each other either.

**Demo the problem** (as part of BUILD, item 3): a naive claim that double-claims jobs under concurrency.

**Teach.**
- The race: draw the ASCII timeline of a naive "SELECT the oldest pending job, then UPDATE it to running". Each statement is atomic; the *pair* isn't.
- **Row locks:** `SELECT ... FOR UPDATE` locks the selected rows until the transaction ends. A second transaction trying to lock the same row waits.
- **`SKIP LOCKED`:** instead of waiting, skip rows someone else has locked. That's ideal for queues: every worker grabs *different* rows without blocking. (It's wrong for most other workloads, because it deliberately gives an inconsistent view.)
- Doing claim-and-mark in **one statement**: a CTE selects and locks, the `UPDATE ... FROM` marks the rows, `RETURNING` hands them back. One round trip, no window for races.
- Why a CTE rather than `WHERE id IN (SELECT ... LIMIT n FOR UPDATE SKIP LOCKED)`: a CTE with `FOR UPDATE` is evaluated exactly once. A sub-select can, in some plans, be re-evaluated and return more rows than the `LIMIT`.
- **`attempts` goes up at claim time**, so it counts *deliveries*. If a job crashes the worker every time (a "poison pill"), it never reaches `FailJob`, but its attempts still climb and we can stop it (Phase 11).
- Leases are set now (explained fully in Phase 8). Times come from Postgres `now()` (Law 2). Durations are passed as milliseconds: `$4::bigint * interval '1 millisecond'`.
- **One queue per claim query:** the index `(queue, run_at, id)` can return rows already in order for a single queue. With several queues, Postgres would have to sort. The service layer (5.2) loops over queues in priority order.

**Build.**
1. In `internal/store/jobs.go`:
   ```go
   type ClaimParams struct {
       Queue         string
       Limit         int
       WorkerID      string
       LeaseDuration time.Duration
   }
   func (s *Store) ClaimJobs(ctx context.Context, p ClaimParams) ([]Job, error)
   ```
   SQL:
   ```sql
   WITH next AS (
       SELECT id
       FROM jobs
       WHERE state = 'pending'
         AND queue = $1
         AND run_at <= now()
       ORDER BY run_at, id
       LIMIT $2
       FOR UPDATE SKIP LOCKED
   )
   UPDATE jobs AS j
   SET state            = 'running',
       attempts         = j.attempts + 1,
       lease_owner      = $3,
       lease_expires_at = now() + ($4::bigint * interval '1 millisecond'),
       updated_at       = now()
   FROM next
   WHERE j.id = next.id
   RETURNING <jobColumns("j.")>
   ```
   Pass `p.LeaseDuration.Milliseconds()` as `$4`. Return an empty, non-nil slice when nothing is claimable.
2. Tests (`internal/store/claim_test.go`):
   - Claims the oldest first (use `Limit: 1` per call when asserting order: `UPDATE ... RETURNING` does **not** guarantee row order); sets `running`, `attempts = 1`, `lease_owner`, and `lease_expires_at` > now.
   - Ignores future jobs (in the test, move `run_at` an hour ahead with direct SQL).
   - Empty queue → empty slice, no error. Respects `Limit`. Ignores other queues.
   - **`TestClaimJobs_NoDoubleClaim`:** enqueue 1,000 jobs; start 20 goroutines that each loop `ClaimJobs(limit 5)` until they get nothing; record every claimed ID in a `map[int64]int` guarded by a `sync.Mutex`. Assert 1,000 distinct IDs, each claimed exactly once. (Preview of Phase 6: explain the goroutines, the mutex and the `WaitGroup` briefly.)
3. **Break-it demo** (`internal/store/naive_claim_demo_test.go`): `TestDemo_NaiveClaimDoubleClaims` is skipped unless `JQ_DEMO=1`. It implements a naive claim (plain `SELECT id ... LIMIT 1`, then `UPDATE jobs SET state = 'running', attempts = attempts + 1, lease_owner = $2, lease_expires_at = now() + interval '30 seconds' WHERE id = $1`, with no state check) and runs the same 20-goroutine test, logging how many jobs were claimed more than once. If you see zero duplicates, add `time.Sleep(time.Millisecond)` between the two statements and explain why widening the race window makes the bug appear. A second demo, `TestDemo_ForUpdateWithoutSkipLocked`, times `FOR UPDATE` without `SKIP LOCKED`: correct, but workers queue up behind each other.

**Verify.**
- `make test` (with `-race`).
- `JQ_DEMO=1 go test ./internal/store -run Demo -v` and read the output together.
- **Hands-on in two psql sessions** (`make psql` twice). Enqueue a few rows first.
  Session A: `BEGIN; SELECT id FROM jobs WHERE state='pending' ORDER BY id LIMIT 1 FOR UPDATE;`
  Session B: the same query → it **hangs**. Ctrl-C it, then run it with `SKIP LOCKED` → you get the *next* row immediately. `ROLLBACK` both.

**Walkthrough focus.** Read the claim SQL line by line. What the lock protects and when it's released (at the end of the statement's implicit transaction). Why `attempts` is incremented here.

**Check your understanding.**
1. Each statement in the naive version is atomic. Why does it still double-claim?
2. What happens with `FOR UPDATE` but without `SKIP LOCKED`?
3. Why increment `attempts` at claim time instead of when the job fails?
4. Why use Postgres `now()` instead of Go's `time.Now()`?

**Done when.** The no-double-claim test passes under `-race`, and the learner has seen the naive version fail.

---

### Step 3.3 — Completing and failing jobs, with fencing

**Problem.** When a worker reports a result, we must be sure it still owns the job. Otherwise a slow "zombie" worker (one we gave up on) could overwrite the result of the worker that legitimately took over.

**Teach.**
- **Compare-and-set (CAS) updates:** `UPDATE ... WHERE id = $1 AND state = 'running' AND lease_owner = $2 AND attempts = $3`, then check rows-affected. Zero rows means "you're not the owner any more".
- **Fencing tokens** (Martin Kleppmann): every new owner gets a strictly larger token, and writes carrying an older token are rejected. Our token is `attempts`, which goes up on every claim. Why `worker_id` alone isn't enough: the *same* worker could be handed the same job again after its lease expired, and the old run must still be rejected.
- Sentinel errors (`ErrLeaseLost`, `ErrNotFound`) and checking them with `errors.Is`.
- Good error messages: when zero rows match, tell "job doesn't exist" apart from "you lost the lease".
- **Bound untrusted input:** truncate stored error messages (to 4 KB) so a handler that returns a 50 MB error can't bloat the table.

**Build.**
1. `type CompleteParams struct { ID int64; WorkerID string; Attempt int }` and `func (s *Store) CompleteJob(ctx context.Context, p CompleteParams) (Job, error)`:
   ```sql
   UPDATE jobs
   SET state = 'succeeded', completed_at = now(), updated_at = now(),
       lease_owner = NULL, lease_expires_at = NULL
   WHERE id = $1 AND state = 'running' AND lease_owner = $2 AND attempts = $3
   RETURNING <jobColumns("")>
   ```
2. `type FailParams struct { ID int64; WorkerID string; Attempt int; ErrorMessage string }` and `func (s *Store) FailJob(ctx context.Context, p FailParams) (Job, error)`:
   ```sql
   UPDATE jobs
   SET state = 'pending', run_at = now(), last_error = $4, updated_at = now(),
       lease_owner = NULL, lease_expires_at = NULL
   WHERE id = $1 AND state = 'running' AND lease_owner = $2 AND attempts = $3
   RETURNING <jobColumns("")>
   ```
   Add the comment: `// NOTE: retries immediately and forever for now. Step 10.2 adds backoff; step 11.1 adds the dead-letter state.`
3. `func (s *Store) leaseLostOrNotFound(ctx context.Context, id int64) error`: on `pgx.ErrNoRows`, check whether the job exists and return `ErrNotFound` or `ErrLeaseLost`.
4. `func truncate(s string, max int) string` (UTF-8 safe: don't cut in the middle of a rune).
5. Tests: complete → `succeeded`, `completed_at` set, lease cleared; wrong attempt → `ErrLeaseLost`; wrong worker → `ErrLeaseLost`; completing twice → second call gets `ErrLeaseLost`; unknown ID → `ErrNotFound`; fail → `pending`, `last_error` set, claimable again with `attempts = 2`; a 10 KB error message is stored truncated. Then **prove the DB guards us**: a direct `UPDATE jobs SET state = 'running' WHERE id = $1` (no lease) fails. Check it with `errors.As(err, &pgErr)` and `pgErr.Code == "23514"` (check_violation).

**Verify.** `make test`.

**Walkthrough focus.** Draw the zombie timeline: Worker A claims (attempt 1) → A freezes → lease expires (Phase 8) → Worker B claims (attempt 2) → A wakes and calls Complete with attempt 1 → rejected. Point at the exact `WHERE` clause that rejects it.

**Check your understanding.**
1. Why fence with both `worker_id` *and* `attempt`?
2. What does the `jobs_lease_check` constraint protect us from?
3. Why does `FailJob` clear the lease?

**Done when.** Complete and Fail are fenced, tested, and the constraint test passes.

---

## Phase 4 — Idempotency keys

### Step 4.1 — Idempotent enqueue

**Problem.** A producer calls `Enqueue` and the network times out. Did the job get created? The producer can't tell. If it retries, it might create a duplicate (two "charge card" jobs). We need enqueue to be safe to retry.

**Teach.**
- **Idempotency:** doing an operation twice has the same effect as doing it once.
- The core uncertainty of networks: was the *request* lost, or only the *response*? The client can't distinguish them, so its only safe move is to retry, which is only safe if the operation is idempotent.
- **Idempotency keys** (Stripe's pattern): the client generates a unique key per *logical* operation and reuses it on every retry. The server stores the key and returns the original result for repeats.
- **Two sides of idempotency:** producer-side de-duplication (this step) and consumer-side idempotent handlers (Phase 8). You need both for "effectively once".
- Implementation: a **partial unique index** plus `INSERT ... ON CONFLICT DO NOTHING`. Why "SELECT to check, then INSERT" is the same race as Phase 3.
- `ON CONFLICT` with a partial unique index must repeat the index's `WHERE` predicate, or Postgres reports "there is no unique or exclusion constraint matching the ON CONFLICT specification".
- What Postgres does when two transactions insert the same key at once: the second waits for the first to commit, then hits the conflict.
- Key scope (per queue) and lifetime (as long as the row exists; retention/cleanup is future work).
- Same key, different payload: options are "error" (Stripe) or "return the existing job". We return the existing job and document that.

**Build.**
1. `internal/store/migrations/0002_add_idempotency_key.sql`:
   ```sql
   -- +goose Up
   ALTER TABLE jobs ADD COLUMN idempotency_key TEXT;
   CREATE UNIQUE INDEX jobs_idempotency_key_uq
       ON jobs (queue, idempotency_key) WHERE idempotency_key IS NOT NULL;

   -- +goose Down
   DROP INDEX jobs_idempotency_key_uq;
   ALTER TABLE jobs DROP COLUMN idempotency_key;
   ```
2. Add `IdempotencyKey *string` to `Job`, and to `jobColumnNames`. Add `IdempotencyKey string` to `EnqueueParams` (empty means none).
3. Change the signature to `func (s *Store) Enqueue(ctx context.Context, p EnqueueParams) (job Job, created bool, err error)`:
   ```sql
   INSERT INTO jobs (queue, type, payload, max_attempts, idempotency_key)
   VALUES ($1, $2, COALESCE($3::jsonb, '{}'::jsonb), COALESCE(NULLIF($4::int, 0), 5), NULLIF($5, ''))
   ON CONFLICT (queue, idempotency_key) WHERE idempotency_key IS NOT NULL DO NOTHING
   RETURNING <jobColumns("")>
   ```
   If this returns `pgx.ErrNoRows`, there was a conflict: `SELECT <jobColumns("")> FROM jobs WHERE queue = $1 AND idempotency_key = $2` and return `created = false`.
4. Update every caller and test for the new signature.
5. Tests: same key twice → same ID, `created` true then false; same key in two queues → two jobs; no key → always a new job; **concurrent:** 50 goroutines enqueue with the same key → exactly one `created == true`, and all 50 get the same ID.

**Verify.** `make test`; `make migrate-up`; in psql, `\d jobs` shows the new index.

**Walkthrough focus.** Step through the concurrent test's timeline: who inserts, who waits, who hits the conflict, and why the follow-up `SELECT` sees the committed row (under READ COMMITTED each statement gets a fresh snapshot).

**Check your understanding.**
1. Why does "SELECT then INSERT" fail under concurrency?
2. A client retries after 10 seconds but generates a *new* key. What happens?
3. Why isn't producer-side idempotency enough for exactly-once processing?

**Done when.** Enqueue is retry-safe and the concurrency test passes under `-race`.

---
## Phase 5 — gRPC API

### Step 5.1 — Protobuf contract and code generation

**Problem.** Producers and workers are separate processes, possibly written in other languages. They need a typed, versioned contract and an efficient transport.

**Teach.**
- RPC vs REST. gRPC = Protocol Buffers (a compact binary schema format) over HTTP/2, with generated client and server code.
- The four RPC shapes (unary, server-streaming, client-streaming, bidirectional). We use unary.
- Protobuf basics: messages; **field numbers are the wire identity** (names aren't sent); scalar types; `repeated`; enums whose zero value is `*_UNSPECIFIED`; `oneof`; well-known types (`Timestamp`, `Duration`).
- **proto3 default values:** an unset `int32` reads as `0` and an unset `bool` as `false`. You can't tell "unset" from "zero", so design fields so the zero value is safe (example: `max_attempts = 0` means "use the server default").
- Schema evolution rules: never change or reuse a field number; adding fields is safe; use `reserved` for removed ones. `buf breaking` checks this for us.
- `buf`: lints style, generates code, detects breaking changes.
- **Why a pull-based `FetchJobs`** instead of the server pushing jobs over a stream: the worker asks only when it has free capacity (natural **backpressure**), and failure handling is simpler.
- Three services split by audience (Producer / Worker / Admin) make it easy to lock down admin operations later.

**Design decisions.** Generated code goes in `gen/` and **is committed** (anyone can build without installing `buf`; the trade-off is noisy diffs). `bytes payload` carries JSON (simple and language-neutral; the server validates it).

**Build.**
1. Install plugins: `go install google.golang.org/protobuf/cmd/protoc-gen-go@latest` and `go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest`. Confirm `which protoc-gen-go` works.
2. `buf.yaml`:
   ```yaml
   version: v2
   modules:
     - path: proto
   lint:
     use:
       - STANDARD
   breaking:
     use:
       - FILE
   ```
3. `buf.gen.yaml`:
   ```yaml
   version: v2
   plugins:
     - local: protoc-gen-go
       out: gen
       opt: paths=source_relative
     - local: protoc-gen-go-grpc
       out: gen
       opt: paths=source_relative
   ```
4. `proto/jobqueue/v1/jobqueue.proto` (replace `USERNAME`):
   ```proto
   syntax = "proto3";

   package jobqueue.v1;

   import "google/protobuf/timestamp.proto";

   option go_package = "github.com/USERNAME/jobqueue/gen/jobqueue/v1;jobqueuev1";

   enum JobState {
     JOB_STATE_UNSPECIFIED = 0;
     JOB_STATE_BLOCKED = 1;
     JOB_STATE_PENDING = 2;
     JOB_STATE_RUNNING = 3;
     JOB_STATE_SUCCEEDED = 4;
     JOB_STATE_DEAD = 5;
     JOB_STATE_CANCELLED = 6;
   }

   message Job {
     int64 id = 1;
     string queue = 2;
     string type = 3;
     bytes payload = 4; // JSON document
     JobState state = 5;
     int32 attempts = 6;
     int32 max_attempts = 7;
     google.protobuf.Timestamp run_at = 8;
     google.protobuf.Timestamp created_at = 9;
     google.protobuf.Timestamp completed_at = 10; // unset unless terminal
     string last_error = 11;
     google.protobuf.Timestamp lease_expires_at = 12; // set only while running
     // Fields 13+ are added in later phases (see PLAN.md 3.5). Never reuse a number.
   }

   // ---------- Producer API ----------

   service ProducerService {
     rpc Enqueue(EnqueueRequest) returns (EnqueueResponse);
     rpc GetJob(GetJobRequest) returns (GetJobResponse);
   }

   message EnqueueRequest {
     string queue = 1;           // "" means "default"
     string type = 2;            // handler name, e.g. "send_email"
     bytes payload = 3;          // JSON; empty means {}
     string idempotency_key = 4; // optional
     int32 max_attempts = 5;     // 0 means the server default (5)
   }

   message EnqueueResponse {
     int64 job_id = 1;
     bool created = 2; // false if an existing job with this idempotency key was returned
   }

   message GetJobRequest {
     int64 job_id = 1;
   }

   message GetJobResponse {
     Job job = 1;
   }

   // ---------- Worker API ----------

   service WorkerService {
     rpc FetchJobs(FetchJobsRequest) returns (FetchJobsResponse);
     rpc CompleteJob(CompleteJobRequest) returns (CompleteJobResponse);
     rpc FailJob(FailJobRequest) returns (FailJobResponse);
   }

   message FetchJobsRequest {
     string worker_id = 1;
     repeated string queues = 2; // checked in order: earlier queues have priority
     int32 max_jobs = 3;         // 1..100
   }

   message FetchJobsResponse {
     repeated Job jobs = 1;
   }

   message CompleteJobRequest {
     int64 job_id = 1;
     string worker_id = 2;
     int32 attempt = 3; // fencing token: must equal Job.attempts from FetchJobs
   }

   message CompleteJobResponse {}

   message FailJobRequest {
     int64 job_id = 1;
     string worker_id = 2;
     int32 attempt = 3;
     string error_message = 4;
   }

   message FailJobResponse {
     JobState new_state = 1;
   }
   ```
5. Makefile:
   ```make
   proto:
   	buf lint && buf generate
   proto-breaking:
   	buf breaking --against '.git#branch=main'
   ```
   (Use your default branch name if it isn't `main`.)
6. `make proto && go mod tidy`.

**Verify.** `make proto` passes lint and writes `gen/jobqueue/v1/jobqueue.pb.go` and `jobqueue_grpc.pb.go`; `go build ./...`. Open the generated `_grpc.pb.go` and find `ProducerServiceServer` and `UnimplementedProducerServiceServer`.

**Walkthrough focus.** Tour the generated code: message structs and getters, the server interface, the client constructor, and why servers embed `Unimplemented...Server` (adding an RPC later won't break compilation). From now on, every proto change must pass `make proto-breaking` before committing.

**Check your understanding.**
1. Why do field numbers matter more than field names?
2. Why is `payload` `bytes` rather than `string`?
3. An old client doesn't know about `max_attempts` and never sets it. What does the server receive, and why is that safe?

**Done when.** The proto passes lint and the generated code compiles.

---

### Step 5.2 — Service layer and validation

**Problem.** Transport code (gRPC) shouldn't contain business rules, and bad input should be rejected before it reaches the database.

**Teach.**
- Layering: `api` (transport: proto ↔ domain types) → `queue` (rules: validation, defaults, multi-queue fetch) → `store` (SQL). Each layer can be tested on its own, and you could add an HTTP API later without touching the rules.
- Validate at the boundary. **Limits protect the system:** payload size, string lengths, batch sizes. Large blobs belong in object storage, with a reference in the payload.
- Typed errors: a `ValidationError` struct, detected with `errors.As`.
- Dependency injection through constructors (`queue.New(store, opts, logger)`), so tests can pass different options.

**Design decisions.** `queue` defines its own input structs and **does not import the generated proto package**. Keeping transport out of the core keeps it portable.

**Build.**
1. `internal/queue/errors.go`: `type ValidationError struct { Field, Reason string }` with a **pointer receiver** `func (e *ValidationError) Error() string` → `"invalid <field>: <reason>"`. Always return it as a pointer: `return &ValidationError{Field: "queue", Reason: "..."}`. Explain why value vs pointer matters for `errors.As`.
2. `internal/queue/validate.go`, with these rules:
   - queue: `""` → `"default"`; otherwise must match `^[a-z0-9][a-z0-9_-]{0,63}$`.
   - type: 1–128 chars, `^[A-Za-z0-9_.:-]+$`.
   - payload: empty → `{}`; must be valid JSON (`json.Valid`) and ≤ 1 MiB.
   - idempotency key: ≤ 255 chars.
   - max_attempts: 0 → default (5); otherwise 1–100.
   - worker_id: 1–128 chars. queues: 1–10 entries, each a valid queue name. max_jobs: 1–100.
3. `internal/queue/service.go`:
   ```go
   type Options struct {
       LeaseDuration      time.Duration // from JQ_LEASE_DURATION
       DefaultMaxAttempts int           // 5
   }
   type Service struct {
       store  *store.Store
       opts   Options
       logger *slog.Logger
   }
   func New(st *store.Store, opts Options, logger *slog.Logger) *Service
   ```
   Input types: `EnqueueInput`, `FetchInput{WorkerID string; Queues []string; MaxJobs int}`, `CompleteInput`, `FailInput`.
   Methods: `Enqueue(ctx, EnqueueInput) (store.Job, bool, error)`, `GetJob(ctx, id)`, `FetchJobs(ctx, FetchInput) ([]store.Job, error)` (loop over `Queues` in order, calling `store.ClaimJobs` with the *remaining* limit until it's filled or the queues run out), `CompleteJob(ctx, CompleteInput) error`, `FailJob(ctx, FailInput) (store.Job, error)`.
4. Add `LeaseDuration` to `config.Server` (`JQ_LEASE_DURATION`, default 30s).
5. Tests: table-driven validation tests (no DB), plus service tests with `testutil.NewTestDB`: multi-queue fetch takes from the first queue before the second; the payload default is `{}`.

**Verify.** `make test`.

**Walkthrough focus.** The `FetchJobs` loop over queues. Why validation lives in `queue`, not `api` or `store`.

**Check your understanding.**
1. Why doesn't `queue` import the generated proto package?
2. Why cap payloads at 1 MiB?
3. With `queues = ["critical", "default"]` and `max_jobs = 10`, and 3 jobs waiting in `critical`, what does `FetchJobs` return?

**Done when.** The service validates input, applies defaults, fetches across queues in order, and is tested.

---

### Step 5.3 — gRPC server and error mapping

**Problem.** Expose the service over the network, translate errors into proper gRPC status codes, and keep internal details private.

**Teach.**
- Server lifecycle: `net.Listen` → `grpc.NewServer` → register services → `Serve` (blocks). Server reflection lets tools like `grpcurl` discover the API.
- **Status codes:** `InvalidArgument`, `NotFound`, `FailedPrecondition`, `Unavailable`, `DeadlineExceeded`, `Internal`. Map them in one place. **Never leak internal errors** (SQL text, hostnames) to clients: log them in full and return a generic `Internal`.
- **Deadlines:** clients should always set one. The server's `ctx` carries it, and passing that `ctx` into pgx means a cancelled RPC also cancels its SQL query.
- Converting `time.Time` ↔ `timestamppb.Timestamp`.
- Testing with **`bufconn`**: an in-memory listener that runs the real gRPC stack without real ports.

**Build.**
1. `internal/api/convert.go`: `jobToProto(store.Job) *jobqueuev1.Job`, `stateToProto(store.State) jobqueuev1.JobState`, and a helper for nullable times.
2. `internal/api/errors.go`: `func toStatus(logger *slog.Logger, method string, err error) error`:
   - `var ve *queue.ValidationError; errors.As(err, &ve)` → `InvalidArgument` with the validation message. (This only matches if the service always returns `&ValidationError{...}`; see 5.2.)
   - `store.ErrNotFound` → `NotFound`; `store.ErrLeaseLost` → `FailedPrecondition("lease lost")`; `store.ErrInvalidState` → `FailedPrecondition`.
   - `context.Canceled` → `Canceled`; `context.DeadlineExceeded` → `DeadlineExceeded`.
   - Anything else → log at error level with `method` and the full error, then return `Internal("internal error")`.
3. `internal/api/producer.go` (`type ProducerServer struct { jobqueuev1.UnimplementedProducerServiceServer; svc *queue.Service; logger *slog.Logger }`) and `internal/api/worker.go` (same pattern for `WorkerServiceServer`). Handlers convert proto → input, call the service, convert back, and map errors.
4. `internal/api/server.go`: `func NewGRPCServer(svc *queue.Service, logger *slog.Logger, opts ...grpc.ServerOption) *grpc.Server` registers both services and reflection (`google.golang.org/grpc/reflection`).
5. `cmd/server/main.go`: add `GRPCListenAddr` to config; wire `store.New(pool)` → `queue.New(...)` → `api.NewGRPCServer(...)`; `net.Listen`; run `Serve` in a goroutine; on `ctx.Done()` call `GracefulStop()` (hardened in 9.2).
6. `internal/testutil/grpc.go`:
   ```go
   type ServerOptions struct {
       LeaseDuration time.Duration // default 30s
       // Later steps add fields: ReaperInterval (8.1), Backoff (10.2), SchedulerMode/Redis (17.2).
   }
   type TestServer struct {
       Conn  *grpc.ClientConn
       Pool  *pgxpool.Pool
       Store *store.Store
       Svc   *queue.Service
   }
   func NewTestServer(t *testing.T, opts ServerOptions) *TestServer
   ```
   Use `bufconn.Listen(1 << 20)`, serve in a goroutine, and dial with `grpc.NewClient("passthrough:///bufnet", grpc.WithContextDialer(...), grpc.WithTransportCredentials(insecure.NewCredentials()))`. **Pitfall:** `grpc.NewClient` uses the `dns` resolver by default, so the `passthrough:///` prefix is required for bufconn. Clean up with `t.Cleanup` (close conn, stop server).
7. Tests (`internal/api/api_test.go`, `package api_test`): Enqueue → GetJob; invalid queue → `codes.InvalidArgument`; unknown job → `codes.NotFound`; Fetch → Complete; Complete with the wrong attempt → `codes.FailedPrecondition`. Use `status.Code(err)`.

**Verify.**
- `make test`.
- `make run-server`, then in another terminal:
  - `grpcurl -plaintext localhost:50051 list`
  - `grpcurl -plaintext -d '{"queue":"default","type":"noop","payload":"e30="}' localhost:50051 jobqueue.v1.ProducerService/Enqueue`
  - `grpcurl -plaintext -d '{"worker_id":"me","queues":["default"],"max_jobs":1}' localhost:50051 jobqueue.v1.WorkerService/FetchJobs`
- Explain the quirk: in JSON, protobuf `bytes` fields are base64, and `e30=` is base64 for `{}`.

**Walkthrough focus.** Follow one Enqueue request: grpcurl → HTTP/2 → generated code → `ProducerServer.Enqueue` → `queue.Service.Enqueue` → `store.Enqueue` → SQL → back out. Then show the error path for a bad queue name.

**Check your understanding.**
1. Why return a generic `Internal` instead of the real error text?
2. If a client cancels its RPC, what happens to the SQL query running for it?
3. What does `bufconn` let us test that calling the service directly wouldn't?

**Done when.** The server runs, `grpcurl` works, and the API tests pass.

---

### Step 5.4 — `jqctl` command-line tool

**Problem.** Humans need an easy way to enqueue and inspect jobs without hand-writing base64 JSON for `grpcurl`.

**Teach.** CLI subcommands with the standard library (`flag.NewFlagSet` per subcommand). A per-call client deadline (`context.WithTimeout`). Printing protobuf messages as JSON with `protojson`.

**Build.**
1. Add `config.Client { ServerAddr string }` (`JQ_SERVER_ADDR`).
2. `cmd/jqctl/main.go`: a `map[string]func(ctx context.Context, conn *grpc.ClientConn, args []string) error` of subcommands, so later phases can slot new ones in. Every **RPC** gets its own 5-second timeout (create the `context.WithTimeout` per call, inside any loop, never one for the whole command).
   - `jqctl enqueue --type sleep --payload '{"ms":500}' [--queue default] [--key abc] [--max-attempts 5]` → prints `job_id` and `created`.
   - `jqctl get <id>` → prints the job as indented JSON (`protojson.MarshalOptions{Multiline: true}`).
   - `jqctl help` → lists subcommands.
3. `make build` now also produces `bin/jqctl`.

**Verify.** With the server running: `./bin/jqctl enqueue --type noop --key k1` twice (same ID, `created=false` the second time); `./bin/jqctl get <id>`; stop the server and run `./bin/jqctl get 1` to see an `Unavailable` error. Explain it.

**Walkthrough focus.** The subcommand map pattern; where the deadline is set and what the user sees when it fires.

**Check your understanding.**
1. What does the 5-second deadline protect the user from?
2. What would you add so a script could retry enqueues safely?

**Done when.** `jqctl` can enqueue and get jobs.

---

## Phase 6 — Concurrency primer

> These four steps are lessons, not features. Each example is a tiny runnable program in `examples/concurrency/NN_name/main.go`. For **every** example the AI: (1) explains the concept, (2) asks the learner to **predict** the output, (3) runs it, (4) explains the actual output, (5) where relevant, runs it again with `go run -race`. Keep each program under ~60 lines with comments. These examples are the reference the learner will come back to, so make them clear. Some examples are *meant* to be buggy (races, leaks, deadlocks, misplaced `wg.Add`); `go vet` may flag them, which is why `make lint` excludes `examples/`. Point that out when it happens: vet catching the bug is part of the lesson.

### Step 6.1 — Goroutines, WaitGroups, channels

**Problem.** The worker (Phase 7) runs many jobs at once. The learner needs a solid mental model of goroutines and channels first.

**Teach.**
- A goroutine is a function running concurrently, very cheap (KB-sized stacks), scheduled by the Go runtime onto OS threads (M:N scheduling).
- When `main` returns, the program exits and **every goroutine is killed**.
- `sync.WaitGroup`: `Add` *before* starting the goroutine, `defer Done()` inside it, `Wait()` to block until all are done. If the Go version has it, mention `wg.Go(func(){...})` (added in Go 1.25), which does Add/Done for you.
- Channels: typed pipes. **Unbuffered** = a hand-off (the sender blocks until a receiver takes the value). **Buffered** = a queue with capacity (the sender blocks only when it's full).
- `close(ch)` signals "no more values". `for v := range ch` stops at close. Receiving from a closed channel returns the zero value with `ok == false`. **Sending on a closed channel panics.** Rule of thumb: only the sender closes.

**Build** (one folder per program):
1. `01_goroutines`: start 5 goroutines that print; `main` returns immediately (predict: little or no output). Then "fix" it with `time.Sleep` (and explain why that's wrong).
2. `02_waitgroup`: the correct fix with `WaitGroup`. Show the bug of calling `wg.Add(1)` *inside* the goroutine (a race with `Wait`).
3. `03_channels_unbuffered`: ping-pong between two goroutines; print timestamps to show the blocking hand-off.
4. `04_channels_buffered`: a producer fills a buffer of 3 with a slow consumer; logs show when the producer blocks.
5. `05_pipeline`: generator → squarer → printer, connected by channels, with proper `close` so every stage stops.

**Verify.** `go run ./examples/concurrency/01_goroutines` (and so on), predicting before each run.

**Walkthrough focus.** For each program, which goroutine blocks where, and why.

**Check your understanding.**
1. Unbuffered vs buffered channel: when does the sender block in each?
2. Who should close a channel, and why?
3. Why is `time.Sleep` not a real fix for `01`?

**Done when.** All five programs run and the learner predicted most outputs.

---

### Step 6.2 — `select`, timeouts, and `context`

**Teach.**
- `select` waits on several channel operations and runs whichever is ready first; `default` makes it non-blocking.
- Timeouts with `time.After`; periodic work with `time.NewTicker` (and why you must `Stop()` it).
- **`context`:** `WithCancel`, `WithTimeout`, `WithDeadline`; the cancellation tree (cancel a parent and every child is cancelled); `ctx.Done()` and `ctx.Err()`.
- `context.WithCancelCause` / `context.Cause` (record *why* something was cancelled; used in 8.2 and 9.1).
- `context.WithoutCancel` (a child that keeps the parent's values but **not** its cancellation; used for acks during shutdown in Phase 9).
- The canonical background loop:
  ```go
  ticker := time.NewTicker(interval)
  defer ticker.Stop()
  for {
      select {
      case <-ctx.Done():
          return
      case <-ticker.C:
          doWork(ctx)
      }
  }
  ```
- `time.Sleep` ignores cancellation. Use a `select` on `ctx.Done()` and a timer instead (write a `sleepCtx(ctx, d) bool` helper).

**Build.**
1. `06_select_timeout`: two channels with different delays plus a timeout case.
2. `07_context_tree`: a parent context with three child goroutines running ticker loops; cancel the parent after 1s and watch all three stop; print `ctx.Err()`. Add a `WithTimeout` child that stops on its own.
3. `08_context_cause_and_without_cancel`: show `Cause` returning a custom error; show that a `WithoutCancel` child stays alive after its parent is cancelled.

**Verify.** Run each, predicting first.

**Check your understanding.**
1. How do you stop a goroutine that's running a ticker loop?
2. What's the difference between `ctx.Err()` and `context.Cause(ctx)`?
3. When would you want `WithoutCancel`?

**Done when.** The learner can write a cancellable background loop from memory (ask them to describe it).

---

### Step 6.3 — Shared memory: races, mutexes, atomics

**Teach.**
- A **data race**: two goroutines access the same memory at the same time and at least one writes. `counter++` is really read → add → write, and those steps interleave.
- The race detector (`-race`) finds races that *actually happen during that run*. It can't prove there are none.
- `sync.Mutex` (and `RWMutex`): protect a critical section; always `defer mu.Unlock()`.
- `sync/atomic` types (`atomic.Int64`, `atomic.Bool`): for single counters and flags.
- Go maps aren't safe for concurrent writes; the runtime may crash with `fatal error: concurrent map writes`.
- The proverb "Don't communicate by sharing memory; share memory by communicating", and when a mutex is simply the clearer choice.

**Build.**
1. `09_race_counter`: 1,000 goroutines do `counter++`. Predict the total, run it, then run with `-race` and read the report together. Fix with a mutex, then with `atomic.Int64`.
2. `10_map_race`: concurrent map writes crash; fix with a mutex-guarded struct.

**Verify.** `go run ./examples/concurrency/09_race_counter`, `go run -race ./examples/concurrency/09_race_counter`.

**Check your understanding.**
1. Why is `counter++` not safe?
2. Mutex or atomic: which would you use for an in-flight job counter, and which for a map of job IDs?
3. If `-race` reports nothing, is your code race-free?

**Done when.** The learner can read a race report and point at both conflicting lines.

---

### Step 6.4 — Patterns and pitfalls: semaphores, errgroup, leaks, deadlocks

**Teach.**
- **Semaphore** with a buffered channel: `sem := make(chan struct{}, N)`; send to acquire, receive to release. It limits concurrency to N. This is exactly how the worker will bound in-flight jobs.
- **`errgroup.WithContext`**: run a group of goroutines; the first error cancels the shared context; `Wait` returns that first error. `SetLimit(n)` bounds concurrency. Used in `cmd/server` to run the gRPC server and background loops together.
- **Goroutine leaks:** a goroutine blocked forever (for example, sending to a channel nobody reads) is never collected. Fix with buffering or a `ctx.Done()` case.
- **Deadlocks:** every goroutine waiting on another; Go detects the total case (`all goroutines are asleep - deadlock!`) but not partial ones.
- Two worker-pool shapes: a fixed pool of N goroutines reading a jobs channel, vs one goroutine per job bounded by a semaphore. We use the second in Phase 7 (preview why).

**Build.**
1. `11_semaphore`: 10 tasks, at most 3 at a time; print in-flight count (atomic) on start and finish.
2. `12_errgroup`: three tasks, one fails after 200ms; the others notice `ctx.Done()` and stop; print the returned error.
3. `13_leak`: a function that leaks a goroutine on each call; print `runtime.NumGoroutine()` growing; fix it.
4. `14_deadlock`: a minimal deadlock; read the crash message.
5. `15_worker_pool_shapes`: both pool shapes side by side processing 20 fake jobs.

**Verify.** Run each; `go get golang.org/x/sync/errgroup` for 12.

**Check your understanding.**
1. How does a buffered channel act as a semaphore?
2. In an errgroup, what happens to the other goroutines when one returns an error?
3. Name two ways a goroutine can leak.

**Done when.** All examples run, and the learner can explain the semaphore pattern in their own words. That pattern is next.

---

## Phase 7 — Worker

### Step 7.1 — Handler registry and demo handlers

**Problem.** A worker must map a job's `type` to the code that runs it, and we need test handlers that behave like real ones: fast, slow, flaky, broken, or panicking.

**Teach.** Function types (`type Handler func(ctx context.Context, job Job) error`). A map from string to handler. The **handler contract**: respect `ctx` cancellation, return an error on failure, be idempotent (explained in Phase 8), and never assume you'll run only once.

**Build.**
1. `internal/worker/handler.go`:
   ```go
   // Job is what a handler receives.
   type Job struct {
       ID          int64
       Queue       string
       Type        string
       Payload     []byte
       Attempt     int // 1 on first delivery
       MaxAttempts int
   }
   type Handler func(ctx context.Context, job Job) error

   type Registry struct{ handlers map[string]Handler }
   func NewRegistry() *Registry
   // Register panics on duplicate types: that's a programming error caught at startup.
   func (r *Registry) Register(jobType string, h Handler)
   func (r *Registry) Lookup(jobType string) (Handler, bool)
   ```
2. `internal/handlers/handlers.go`: `func RegisterDemo(r *worker.Registry)` registering:
   - `noop`: returns nil.
   - `sleep`: payload `{"ms": 500}`; waits using `select` on `ctx.Done()` and `time.After`. Returns `ctx.Err()` if cancelled.
   - `flaky`: payload `{"fail_rate": 0.3}`; fails randomly with `"flaky failure"`.
   - `always_fail`: returns `errors.New("always fails")`.
   - `panic`: `panic("boom")`.
   (Later: `record_effect` in 8.3, `permanent_fail` in 11.1.)
3. Tests: `sleep` returns promptly when its context is cancelled; a bad payload returns an error.

**Verify.** `make test`.

**Check your understanding.**
1. Why must `sleep` use `select` instead of `time.Sleep`?
2. Why is panicking on duplicate registration acceptable here?

**Done when.** The registry and demo handlers exist and are tested.

---

### Step 7.2 — Worker pool: fetch loop and semaphore

**Problem.** Run many jobs concurrently in one process, never more than N at once; don't hammer the server when there's no work; survive handler panics.

**Teach.**
- **Pull + backpressure:** fetch only as many jobs as there are free slots, so a worker never holds jobs it can't start (their leases would tick away unused).
- The semaphore from 6.4, applied: a token = one running job.
- **Idle backoff polling:** when a fetch returns nothing, wait 100ms, then 200ms, 400ms… up to 2s; reset on success. Trade-off: latency vs load on the server. Long-polling (Phase 17) improves on this.
- **Panic recovery:** `recover()` only works in a deferred function **in the same goroutine** that panicked. So it must sit in the goroutine that runs the handler.
- **Acks use their own context:** `context.WithTimeout(context.WithoutCancel(ctx), ackTimeout)`. Results must still be reported while the worker is shutting down (Phase 9 relies on this).
- **Ownership:** `Run` owns the fetch loop; the fetch loop starts one goroutine per job; a `WaitGroup` tracks them; `Run` never returns while job goroutines are still running.

**Design decisions.** Goroutine-per-job bounded by a semaphore (vs a fixed pool of N goroutines reading a channel). It makes "how many can I fetch?" trivial to compute, and goroutines are cheap.

**Build.**
1. `internal/worker/worker.go`:
   ```go
   type Config struct {
       WorkerID    string
       Queues      []string
       Concurrency int           // max jobs in flight
       MaxBatch    int           // max jobs per FetchJobs call; default = Concurrency, capped at 100
       PollMin     time.Duration // 100ms
       PollMax     time.Duration // 2s
       AckTimeout  time.Duration // 5s
   }
   type Worker struct {
       cfg      Config
       client   jobqueuev1.WorkerServiceClient
       registry *Registry
       logger   *slog.Logger
       sem      chan struct{}
       wg       sync.WaitGroup
       inflight atomic.Int64
   }
   func New(cfg Config, client jobqueuev1.WorkerServiceClient, registry *Registry, logger *slog.Logger) (*Worker, error)
   func (w *Worker) Run(ctx context.Context) error
   ```
   Reference shape for `Run` (the AI should implement and explain it; it gets restructured in 9.1):
   ```go
   func (w *Worker) Run(ctx context.Context) error {
       defer w.wg.Wait() // never return while jobs are still running
       backoff := w.cfg.PollMin
       for {
           // 1. Block until at least one slot is free, or we're told to stop.
           select {
           case w.sem <- struct{}{}:
           case <-ctx.Done():
               return nil
           }
           slots := 1
           // 2. Grab any other free slots without waiting.
       grab:
           for slots < w.cfg.MaxBatch {
               select {
               case w.sem <- struct{}{}:
                   slots++
               default:
                   break grab
               }
           }
           // 3. Ask for at most `slots` jobs.
           jobs, err := w.fetch(ctx, slots)
           // 4. Return the slots we didn't use.
           for i := len(jobs); i < slots; i++ {
               <-w.sem
           }
           if err != nil || len(jobs) == 0 {
               if err != nil && ctx.Err() == nil {
                   w.logger.Warn("fetch failed", "err", err)
               }
               if !sleepCtx(ctx, backoff) {
                   return nil
               }
               backoff = min(backoff*2, w.cfg.PollMax)
               continue
           }
           backoff = w.cfg.PollMin
           // 5. One goroutine per job; each releases its slot when finished.
           for _, j := range jobs {
               w.wg.Add(1)
               go func(j Job) {
                   defer w.wg.Done()
                   defer func() { <-w.sem }()
                   w.execute(ctx, j)
               }(j)
           }
       }
   }
   ```
   - `fetch` calls `FetchJobs` with a 10s timeout and converts proto jobs to `worker.Job`.
   - `execute(ctx, j)`: look up the handler (missing → fail with `no handler registered for type "<t>"`); call `runHandler`; on error call `fail`, otherwise `complete`. Increment/decrement `inflight`. Log start and finish with `job_id`, `type`, `attempt`, `duration`.
   - `runHandler` uses a named return value plus `defer func() { if r := recover(); r != nil { err = fmt.Errorf("handler panicked: %v\n%s", r, debug.Stack()) } }()`.
   - `complete` / `fail` use `context.WithTimeout(context.WithoutCancel(ctx), w.cfg.AckTimeout)`. If an ack fails, log it. The job will be retried after its lease expires (at-least-once), and this comment must say so.
2. Add `config.Worker` (`JQ_SERVER_ADDR`, `JQ_WORKER_ID` default `<hostname>-<pid>`, `JQ_WORKER_QUEUES` comma-split, `JQ_WORKER_CONCURRENCY`, `JQ_LOG_LEVEL`).
3. `cmd/worker/main.go`: load config; `grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))`; registry + `handlers.RegisterDemo`; `signal.NotifyContext`; `w.Run(ctx)`; close the connection. Add a comment that plaintext is for local dev only and production would use TLS/mTLS.
4. Makefile: `run-worker: go run ./cmd/worker`.

**Verify.** `go build ./...`, `make lint`. (Tests come in 7.3.)

**Walkthrough focus.** Trace the semaphore through one cycle with N = 3: tokens taken, the batch fetched, unused tokens returned, tokens released as jobs finish. Name each goroutine's owner and how it stops. Explain why `recover` sits where it does.

**Check your understanding.**
1. Why fetch only `slots` jobs instead of a fixed 100?
2. What would happen if `recover()` were in `Run` instead of in the job goroutine?
3. What happens to a job if the `CompleteJob` call fails because of a network blip?

**Done when.** The worker compiles and the learner can explain the loop.

---

### Step 7.3 — Worker integration tests and running it for real

**Problem.** Prove the worker processes everything, respects its concurrency limit, and survives panics. Then watch it run against the real server.

**Teach.**
- **Never sleep a fixed time in tests;** poll for a condition with a deadline. Write `testutil.WaitFor(t, timeout, func() bool)`.
- Measuring max concurrency with an atomic counter plus a compare-and-swap "max" loop.

**Build.**
1. `internal/testutil/wait.go`: `WaitFor(t *testing.T, timeout time.Duration, cond func() bool)`, polling every 20ms and failing with a clear message on timeout.
2. `internal/worker/worker_test.go` (`package worker_test`, using `testutil.NewTestServer`):
   - `TestWorker_ProcessesAllJobs`: enqueue 200 `noop` jobs; run a worker (concurrency 8) in a goroutine with a cancellable ctx; `WaitFor` all `succeeded` (count via SQL); cancel; `Run` returns.
   - `TestWorker_RespectsConcurrencyLimit`: a custom handler increments an atomic in-flight counter, records the max, sleeps 50ms, decrements. 50 jobs at concurrency 5 → the max is exactly 5.
   - `TestWorker_PanicDoesNotKillWorker`: 1 `panic` job + 5 `noop` jobs → the 5 succeed; the panic job has `attempts >= 1` and `last_error` containing `panicked`. **Don't assert its state**: it's retried immediately, so it may be `pending` or `running` at any instant.
   - `TestWorker_UnknownType`: the job has `attempts >= 1` and `last_error` containing `no handler` (again, don't assert the state).
3. Manual run (three terminals): `make run-server`; `make run-worker`; then
   `for i in $(seq 20); do ./bin/jqctl enqueue --type sleep --payload '{"ms":500}'; done`.
   Watch the logs; start a second worker and see jobs split between them; in psql, `SELECT state, count(*) FROM jobs GROUP BY state;`.

**Verify.** `make test`; the manual run above.

**Walkthrough focus.** How the tests prove each property; what the two-worker run shows about `SKIP LOCKED` in practice.

**Check your understanding.**
1. Why is `WaitFor` better than `time.Sleep(2 * time.Second)` in tests?
2. In the two-worker run, what stops both workers taking the same job?

**Done when.** All worker tests pass under `-race`, and the learner has watched two workers share a queue.

---
## Phase 8 — Leases, heartbeats, and at-least-once delivery

### Step 8.1 — Stuck jobs and the lease reaper

**Problem.** If a worker dies mid-job, that job stays `running` forever. Nobody will ever run it. This is the central failure of distributed systems: a process died while holding work.

**Demo the problem** (FRAME stage): `make build`; run the server, and run the worker as the **compiled binary** `./bin/worker` (not `make run-worker`: `go run` starts a child process, and killing the parent doesn't kill the child); `./bin/jqctl enqueue --type sleep --payload '{"ms":60000}'`; once the worker logs that it started, `kill -9 $(pgrep -f bin/worker)`. Then in psql: the job is `running` with a `lease_owner` that no longer exists. Restart the worker: it never picks the job up.

**Teach.**
- **You can't tell "slow" from "dead"** over a network. There's no perfect failure detector, so systems use **timeouts**.
- A **lease** is time-limited ownership: "you own this job until `lease_expires_at`". Same idea as SQS's visibility timeout, Kafka's session timeout, and Kubernetes Lease objects.
- The **reaper:** a periodic sweep that returns jobs with expired leases to `pending`.
- **One clock** (Law 2): leases are set *and* checked with Postgres `now()`, so clock differences between machines don't matter.
- The reaper is safe to run on every server replica: it uses the same CTE + `FOR UPDATE SKIP LOCKED` + `LIMIT` pattern, so replicas split the work instead of fighting over it.
- Choosing a lease duration: shorter = faster recovery, but more false positives on slow jobs (fixed by heartbeats in 8.2).
- The background-loop pattern from 6.2, now owned by the server's `errgroup`.

**Build.**
1. `internal/store/migrations/0003_add_running_lease_index.sql`:
   ```sql
   -- +goose Up
   CREATE INDEX jobs_running_lease_idx ON jobs (lease_expires_at) WHERE state = 'running';
   -- +goose Down
   DROP INDEX jobs_running_lease_idx;
   ```
2. `func (s *Store) ReapExpiredLeases(ctx context.Context, limit int) ([]Job, error)`:
   ```sql
   WITH expired AS (
       SELECT id
       FROM jobs
       WHERE state = 'running' AND lease_expires_at < now()
       ORDER BY lease_expires_at
       LIMIT $1
       FOR UPDATE SKIP LOCKED
   )
   UPDATE jobs AS j
   SET state = 'pending',
       run_at = now(),
       lease_owner = NULL,
       lease_expires_at = NULL,
       last_error = 'lease expired (worker crashed or stopped heartbeating)',
       updated_at = now()
   FROM expired
   WHERE j.id = expired.id
   RETURNING <jobColumns("j.")>
   ```
   (`attempts` is **not** changed: the claim already counted this delivery.)
3. `internal/queue/reaper.go`:
   - `func (s *Service) ReapOnce(ctx context.Context) (int, error)`: call `ReapExpiredLeases(ctx, 500)` repeatedly while it returns a full batch; log the total if > 0.
   - `func (s *Service) RunReaper(ctx context.Context, interval time.Duration) error`: ticker loop; log errors but keep going; return `nil` when `ctx` is cancelled. (Convention for all background loops: return `nil` on cancellation; return an error only for something fatal.)
4. `cmd/server/main.go`: switch to `errgroup.WithContext`, with three goroutines: `Serve`, `RunReaper`, and one that waits for `gctx.Done()` and calls `GracefulStop()`. Add `ReaperInterval` to config (`JQ_REAPER_INTERVAL`, default 5s).
5. Add `ReaperInterval time.Duration` to `testutil.ServerOptions` (0 = don't run a reaper) and start `RunReaper` in the test server when it's set. Run **all** test-server background loops (this one, and later stats/promoter/reconciler) in an errgroup under a cancellable context, and in the test server's `t.Cleanup` **cancel and wait for them first**. (`t.Cleanup` runs in reverse order, so this runs before `NewTestDB`'s cleanup drops the database; otherwise a loop would query a database that's being dropped.)
6. Tests: claim a job, force expiry with SQL (`UPDATE jobs SET lease_expires_at = now() - interval '1 second' WHERE id = $1`), `ReapOnce` → `pending`, lease cleared, `attempts` still 1, `last_error` set; re-claim → `attempts = 2`; the *old* owner's `CompleteJob` with attempt 1 → `ErrLeaseLost` (fencing proven end to end). A job with an unexpired lease is not touched.

**Verify.** `make test`. Then repeat the demo with `make run-server JQ_LEASE_DURATION=10s` (worker as `./bin/worker`): kill the worker mid-job, start a new worker, and watch the job get reaped within ~10–15s and complete with `attempts = 2`.

**Walkthrough focus.** The reaper SQL. The server's errgroup: who owns which goroutine and how each one stops. The worst-case "stuck" time (lease duration + reaper interval).

**Check your understanding.**
1. Why not let the worker's clock decide when a lease has expired?
2. What's the longest a crashed job can sit before another worker can take it?
3. What if a job legitimately takes longer than the lease? (Answer: it gets reaped while still running, which motivates 8.2.)

**Done when.** Crashed workers' jobs are recovered automatically and the tests pass.

---

### Step 8.2 — Heartbeats and lost-lease cancellation

**Problem.** A 5-minute job with a 30-second lease would be reaped while still running, so two workers would run it at once. We can't just set huge leases either, because crash recovery would get slow.

**Teach.**
- **Heartbeats:** while a job runs, the worker renews its lease every `lease / 3` (so two missed beats are tolerated).
- Renewal is a fenced CAS (Law 6). If it fails, someone else may now own the job, so **stop working**: cancel the handler's context.
- `context.WithCancelCause` lets us record *why* we cancelled (`errLeaseLost` vs a normal finish).
- **Zombies:** a handler that ignores `ctx` keeps running after we cancel it. Its result will be rejected by fencing, but any side effects it already caused (an email, a charge) may happen twice. That's why handlers must be idempotent (8.3).
- Heartbeat errors other than "lease lost" (for example, server briefly unavailable) aren't fatal: keep trying; the lease is still valid until it expires.

**Design decisions.** The server tells workers the lease duration in `FetchJobsResponse`, so lease policy lives in one place.

**Build.**
1. `func (s *Store) ExtendLease(ctx context.Context, p ExtendLeaseParams) (time.Time, error)` where `ExtendLeaseParams{ID int64; WorkerID string; Attempt int; LeaseDuration time.Duration}`:
   ```sql
   UPDATE jobs
   SET lease_expires_at = now() + ($4::bigint * interval '1 millisecond'), updated_at = now()
   WHERE id = $1 AND state = 'running' AND lease_owner = $2 AND attempts = $3
   RETURNING lease_expires_at
   ```
   No rows → `leaseLostOrNotFound`.
2. Proto (run `make proto-breaking` before committing): add `import "google/protobuf/duration.proto";` and
   ```proto
   // in WorkerService
   rpc Heartbeat(HeartbeatRequest) returns (HeartbeatResponse);

   message HeartbeatRequest {
     int64 job_id = 1;
     string worker_id = 2;
     int32 attempt = 3;
   }
   message HeartbeatResponse {
     google.protobuf.Timestamp lease_expires_at = 1;
   }
   ```
   and in `FetchJobsResponse`: `google.protobuf.Duration lease_duration = 2;`.
3. Service `Heartbeat`; API handler; `FetchJobs` handler sets `lease_duration` from options.
4. Worker: add `LeaseDuration` to the internal per-job struct (not to the handler-facing `Job`, if you can keep it separate cleanly; otherwise add an unexported field). In `execute`:
   ```go
   jobCtx, cancel := context.WithCancelCause(ctx)
   defer cancel(nil)
   hbDone := make(chan struct{})
   go func() {
       defer close(hbDone)
       w.heartbeatLoop(jobCtx, j, cancel)
   }()
   err := w.runHandler(jobCtx, h, j)
   cancel(nil) // stop heartbeats
   <-hbDone    // wait for the heartbeat goroutine to exit: no leaks
   if errors.Is(context.Cause(jobCtx), errLeaseLost) {
       w.logger.Warn("lease lost; dropping result", "job_id", j.ID)
       return // don't ack: someone else owns the job now
   }
   // ...then complete or fail as before
   ```
   `heartbeatLoop`: ticker at `LeaseDuration / 3` (if `LeaseDuration <= 0`, e.g. an older server that doesn't send it, use 30s: `time.NewTicker` panics on a zero interval); each tick calls `Heartbeat` with a timeout of `LeaseDuration / 3`; on `codes.FailedPrecondition` or `codes.NotFound` → `cancel(errLeaseLost)` and return; other errors → log and continue; return when `jobCtx` is done.
   Note the subtlety: `context.Cause` returns the *first* cause, so a later `cancel(nil)` doesn't overwrite `errLeaseLost`.
5. Tests:
   - `TestHeartbeat_KeepsLongJobAlive`: test server with lease 1s and reaper interval 200ms; a 3-second `sleep` job → `succeeded` with `attempts = 1`.
   - `TestHeartbeat_LostLeaseCancelsHandler`: a handler that blocks until `ctx.Done()` and records `context.Cause(ctx)`; once it's running, steal the lease with SQL (`UPDATE jobs SET lease_owner = 'thief' WHERE id = $1`); assert the handler was cancelled, the worker didn't ack, and the job is still owned by `thief`.

**Verify.** `make test`; `make proto-breaking`.

**Walkthrough focus.** The per-job goroutine pair (handler + heartbeat): who owns the heartbeat goroutine, how it stops, and why we wait on `hbDone`.

**Check your understanding.**
1. Why heartbeat every `lease / 3` and not every `lease`?
2. What should a well-behaved handler do when its `ctx` is cancelled?
3. Why must the worker *not* ack after losing the lease?

**Done when.** Long jobs survive via heartbeats, and lost leases cancel handlers.

---

### Step 8.3 — Idempotent handlers and the crash demo

**Problem.** At-least-once means duplicates *will* happen, for example when a worker crashes after doing the work but before acking. Handlers need effects that are safe to repeat.

**Teach.**
- **Delivery semantics:**
  - *At-most-once:* ack before working. A crash loses the job.
  - *At-least-once:* ack after working. A crash repeats the job.
  - *Exactly-once:* impossible in general across independent systems. What's achievable is **effectively-once** = at-least-once delivery + idempotent effects.
  Draw both crash windows on a timeline.
- Idempotent handler patterns:
  1. Naturally idempotent operations (`SET x = 5`, not `x = x + 1`).
  2. A **dedupe table** keyed by job ID, written *in the same transaction as the effect*, in the target system.
  3. Passing the job ID as the idempotency key to an external API (e.g. a payment provider).
- Why the dedupe insert and the effect must be in the same transaction.

**Build.**
1. `internal/store/migrations/0004_create_demo_effects.sql`:
   ```sql
   -- +goose Up
   -- Simulates an external system our jobs write to.
   -- In real life this would be another service's database, not the queue's.
   CREATE TABLE demo_effects (
       job_id       BIGINT PRIMARY KEY,
       worker_id    TEXT        NOT NULL,
       processed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
       deliveries   INT         NOT NULL DEFAULT 1
   );
   -- +goose Down
   DROP TABLE demo_effects;
   ```
   The *effect* is the row existing exactly once. `deliveries` only records how many times the handler ran.
2. `handlers.RegisterRecordEffect(r *worker.Registry, pool *pgxpool.Pool, workerID string)` registers `record_effect`. Payload: `{"sleep_ms": 200, "crash_after_effect": false}`. It sleeps (ctx-aware), then:
   ```sql
   INSERT INTO demo_effects (job_id, worker_id) VALUES ($1, $2)
   ON CONFLICT (job_id) DO UPDATE SET deliveries = demo_effects.deliveries + 1
   ```
   If `crash_after_effect` is true **and** `job.Attempt == 1`, call `os.Exit(1)` right after the insert. This deterministically simulates a crash in the worst window: work done, ack not sent.
3. Add `DatabaseURL string` (`JQ_DATABASE_URL`, default empty) to `config.Worker`. `cmd/worker/main.go`: if it's set, open a small pool (`MaxConns = 4`) and register `record_effect`. Comment that this is demo-only: real workers talk to *their own* systems, not the queue database (Law 4).
4. Test: call the handler twice directly for the same job → one row, `deliveries = 2`.
5. Add a "Delivery guarantees" section to `docs/ARCHITECTURE.md` (the AI drafts it with the learner).

**Verify.**
- `make test`.
- Crash demo: `make run-server JQ_LEASE_DURATION=10s`; worker running as `./bin/worker`; `./bin/jqctl enqueue --type record_effect --payload '{"sleep_ms":100,"crash_after_effect":true}'`. The worker exits. Restart it. After the lease expires, the job completes on attempt 2. In psql: `SELECT * FROM demo_effects;` → **one row, `deliveries = 2`**. The job ran twice, the effect happened once.

**Walkthrough focus.** Map the crash demo onto the delivery-semantics timeline. Point to the exact line where a non-idempotent handler would have caused a duplicate.

**Check your understanding.**
1. In which exact window does a crash cause a duplicate delivery?
2. Could we get exactly-once by acking *before* running the handler? What would we lose?
3. Why must the dedupe write and the effect be in one transaction?

**Done when.** The crash demo shows two deliveries and one effect.

---

## Phase 9 — Graceful shutdown

### Step 9.1 — Worker graceful shutdown and job release

**Problem.** Deploys and restarts send SIGTERM. Right now, Ctrl-C cancels running handlers, and they're recorded as *failures*: attempts are burned, `last_error` says "context canceled", and if the process is killed instead, jobs sit until their leases expire.

**Demo the problem:** enqueue 5 `sleep` jobs of 10s, start a worker, press Ctrl-C after 2s, and look at `attempts` and `last_error` in psql.

**Teach.**
- Signals: SIGINT (Ctrl-C), SIGTERM (`docker stop`, Kubernetes, systemd), SIGKILL (can't be caught). The Kubernetes flow: SIGTERM → grace period (default 30s) → SIGKILL.
- **The drain pattern:** (1) stop taking new work, (2) let in-flight work finish within a deadline, (3) hand back what didn't finish, (4) close connections, (5) exit.
- **The two-context pattern:** `ctx` (cancelled on signal: stops *fetching*) and `jobsCtx` (cancelled only when the shutdown deadline passes: stops *jobs*). Draw the timeline.
- **Release vs Fail:** release means "I'm stopping, not the job's fault". It doesn't count as an attempt.
- **You can't kill a goroutine in Go.** If a handler ignores cancellation, all we can do is stop waiting for it. Its lease will expire and the reaper will recover the job.
- A second Ctrl-C forces an immediate exit.
- Shutdown timeout must be shorter than the orchestrator's grace period (25s < 30s).

**Build.**
1. `func (s *Store) ReleaseJob(ctx context.Context, p ReleaseParams) (Job, error)` with `ReleaseParams{ID int64; WorkerID string; Attempt int}`:
   ```sql
   UPDATE jobs
   SET state = 'pending', run_at = now(), attempts = GREATEST(attempts - 1, 0),
       lease_owner = NULL, lease_expires_at = NULL,
       last_error = 'released during worker shutdown', updated_at = now()
   WHERE id = $1 AND state = 'running' AND lease_owner = $2 AND attempts = $3
   RETURNING <jobColumns("")>
   ```
2. Proto: `rpc ReleaseJob(ReleaseJobRequest) returns (ReleaseJobResponse);` in `WorkerService`, with `ReleaseJobRequest { int64 job_id = 1; string worker_id = 2; int32 attempt = 3; }` and an empty `ReleaseJobResponse {}`. Service + API handler. `make proto-breaking`.
3. Worker config: `ShutdownTimeout` (`JQ_WORKER_SHUTDOWN_TIMEOUT`, 25s) and `HardStopTimeout` (5s, not configurable).
4. Restructure the worker:
   ```go
   var errShutdown = errors.New("worker shutting down")

   // Run blocks until ctx is cancelled, then drains: stop fetching, wait up to
   // ShutdownTimeout for in-flight jobs, then cancel them and release them.
   func (w *Worker) Run(ctx context.Context) error {
       jobsCtx, cancelJobs := context.WithCancelCause(context.WithoutCancel(ctx))
       defer cancelJobs(nil)

       w.fetchLoop(ctx, jobsCtx) // returns once ctx is cancelled; jobs run under jobsCtx

       w.logger.Info("shutdown: stopped fetching", "inflight", w.inflight.Load())
       done := make(chan struct{})
       go func() { w.wg.Wait(); close(done) }()

       select {
       case <-done:
           return nil
       case <-time.After(w.cfg.ShutdownTimeout):
       }
       w.logger.Warn("shutdown: timeout reached; cancelling in-flight jobs", "inflight", w.inflight.Load())
       cancelJobs(errShutdown)

       select {
       case <-done:
       case <-time.After(w.cfg.HardStopTimeout):
           w.logger.Error("shutdown: abandoning jobs that ignore cancellation; their leases will expire",
               "inflight", w.inflight.Load())
       }
       return nil
   }
   ```
   - The old body of `Run` becomes `fetchLoop(ctx, jobsCtx)`, with **no** `defer w.wg.Wait()` (Run now does the waiting). Job goroutines call `w.execute(jobsCtx, j)`.
   - In `execute`, decide the outcome in **exactly** this order: cause is `errLeaseLost` → drop (even if the handler returned `nil`: someone else owns the job); handler returned `nil` → complete; cause is `errShutdown` → **release**; otherwise → fail.
5. `cmd/worker/main.go`, second-signal force exit:
   ```go
   ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
   defer stop()
   go func() {
       <-ctx.Done()
       stop() // restore default signal handling: a second Ctrl-C kills the process immediately
   }()
   ```
   Close the gRPC connection only *after* `Run` returns, so acks and releases can still be sent.
6. Tests:
   - `TestShutdown_DrainsInFlight`: concurrency 4, enqueue 8 `sleep` jobs of 300ms; wait until 4 are `running`; cancel; `Run` returns in < 1s; exactly 4 `succeeded`, 4 still `pending` with `attempts = 0` (nothing new was fetched after cancel).
   - `TestShutdown_ReleasesOnTimeout`: ShutdownTimeout 200ms, ctx-respecting 10s jobs → after cancel they end `pending` with `attempts = 0` and `last_error = 'released during worker shutdown'`.
   - `TestShutdown_AbandonsStubbornHandlers`: a handler that ignores ctx (`time.Sleep(10 * time.Second)`), ShutdownTimeout 100ms, HardStopTimeout 300ms (make it overridable in tests) → `Run` returns in < 1s; the job is still `running`.

**Verify.** `make test`; repeat the demo: Ctrl-C during 3s jobs → they finish; Ctrl-C during 60s jobs with `JQ_WORKER_SHUTDOWN_TIMEOUT=3s` → they're released; a quick second Ctrl-C exits immediately.

**Walkthrough focus.** Draw the shutdown timeline with both contexts. Why `jobsCtx` is built from `WithoutCancel(ctx)`. The outcome decision order in `execute`.

**Check your understanding.**
1. Why derive `jobsCtx` from `context.WithoutCancel(ctx)` instead of from `ctx`?
2. Why can't Go forcibly stop a goroutine, and what do we do instead?
3. Release subtracts one from `attempts`, but the reaper doesn't. Why the difference?
4. What happens in Kubernetes if `ShutdownTimeout` is longer than the pod's grace period?

**Done when.** Ctrl-C drains cleanly, timeouts release jobs, and the tests pass.

---

### Step 9.2 — Server graceful shutdown and health checks

**Problem.** The server must stop accepting RPCs, finish the ones in flight (like a `CompleteJob`), stop background loops, and close the database pool, in that order. Load balancers also need to know when to stop sending traffic.

**Teach.**
- `GracefulStop` (refuse new RPCs, wait for in-flight ones) vs `Stop` (immediate). `GracefulStop` can hang on long calls, so bound it with a timeout and then fall back to `Stop`.
- Shutdown order is the reverse of startup; `defer` runs LIFO.
- `errgroup` in `main`: the first component to fail cancels the others (for example, the port is already in use → everything stops cleanly).
- The standard gRPC **health service** (`grpc.health.v1.Health`): readiness for load balancers and Kubernetes probes. Mark NOT_SERVING at the start of shutdown.
- gRPC clients reconnect automatically: workers ride through a server restart, logging `Unavailable` with backoff meanwhile.

**Build.**
1. `cmd/server/main.go`: config `ShutdownTimeout` (`JQ_SERVER_SHUTDOWN_TIMEOUT`, 10s). In `run`:
   ```go
   hs := health.NewServer()
   healthpb.RegisterHealthServer(grpcServer, hs)

   g, gctx := errgroup.WithContext(ctx)
   g.Go(func() error { return grpcServer.Serve(lis) }) // returns nil after Stop/GracefulStop
   g.Go(func() error { return svc.RunReaper(gctx, cfg.ReaperInterval) })
   g.Go(func() error {
       <-gctx.Done()
       logger.Info("shutdown: draining gRPC")
       hs.Shutdown() // report NOT_SERVING to health checks
       stopped := make(chan struct{})
       go func() { grpcServer.GracefulStop(); close(stopped) }()
       select {
       case <-stopped:
       case <-time.After(cfg.ShutdownTimeout):
           logger.Warn("shutdown: graceful stop timed out; forcing")
           grpcServer.Stop()
       }
       return nil
   })
   err := g.Wait()
   // deferred pool.Close() runs after this
   ```
   Every later phase adds its background loops to this errgroup.
2. Log each shutdown stage, so the order is visible.

**Verify.**
- `grpcurl -plaintext localhost:50051 grpc.health.v1.Health/Check` → `SERVING`.
- With server and worker processing `sleep` jobs: Ctrl-C the server → watch the shutdown stages, and the worker logging `Unavailable` and retrying; restart the server → the worker carries on by itself.
- Start a second server on the same port → it fails fast with a clear "address already in use" error and exits cleanly.

**Walkthrough focus.** The errgroup and each goroutine's stop condition. The order: health → RPCs → background loops → DB pool.

**Check your understanding.**
1. Why close the DB pool last?
2. What does a client see while the server is in `GracefulStop`?
3. Why put a timeout on `GracefulStop`?

**Done when.** The server shuts down in order, health checks work, and workers survive a server restart.

---

## Phase 10 — Retries with exponential backoff and jitter

### Step 10.1 — Backoff package and simulation

**Problem.** A failing job retries instantly, forever. If the failure is a downstream outage, we're hammering that service while it's down.

**Demo the problem:** enqueue one `always_fail` job, run a worker for 3 seconds, stop it, and check `attempts` in psql (hundreds or thousands).

**Teach.**
- Transient failures (timeouts, 503s, deadlocks) vs permanent ones (bad input). Only transient ones deserve retries (permanent errors are handled in 11.1).
- **Exponential backoff:** `delay = base × 2^(attempt−1)`, capped at `max`.
- **Thundering herd:** 1,000 jobs fail at the same moment during an outage → without randomness they all retry at the *same* moments → synchronized spikes. **Jitter** spreads them out. Full jitter `random(0, ceiling)`, equal jitter, decorrelated jitter (AWS Architecture Blog, Appendix C). We use **full jitter**.
- Integer overflow: `base << 70` overflows. Stop doubling once you reach the cap.
- **Testable randomness:** inject the random source (`func() float64`) so tests are deterministic.
- Policy lives on the **server**: one consistent policy that workers can't get wrong.

**Build.**
1. `internal/backoff/backoff.go`:
   ```go
   // Policy computes retry delays using exponential backoff with full jitter.
   type Policy struct {
       Base time.Duration  // ceiling for the first retry, e.g. 1s
       Max  time.Duration  // cap, e.g. 10m (must be <= 24h)
       Rand func() float64 // returns a value in [0, 1); nil means math/rand/v2.Float64
   }

   // Delay returns how long to wait before retrying after the given attempt (1-based).
   func (p Policy) Delay(attempt int) time.Duration {
       if attempt < 1 {
           attempt = 1
       }
       ceiling := p.Base
       for i := 1; i < attempt && ceiling < p.Max; i++ {
           ceiling *= 2 // stop doubling at the cap, which also prevents overflow
       }
       if ceiling > p.Max {
           ceiling = p.Max
       }
       r := p.Rand
       if r == nil {
           r = rand.Float64
       }
       return time.Duration(r() * float64(ceiling))
   }
   ```
   Add `func (p Policy) Validate() error` (Base > 0, Max ≥ Base, Max ≤ 24h).
2. Tests (table-driven): with `Rand` returning `0.999999`, attempts 1..5 give just under 1s, 2s, 4s, 8s, 16s; a large attempt gives just under `Max`; attempt 10,000 doesn't overflow; `Rand` returning 0 gives 0; over 10,000 real random samples, the mean is within 5% of `ceiling / 2`.
3. `examples/backoff_sim/main.go`: simulate 1,000 clients that all fail at t = 0 and retry up to 5 times under three strategies: fixed 1s, exponential without jitter, exponential with full jitter. Print an ASCII histogram of retries per second for the first 60 seconds of each.

**Verify.** `make test`; `go run ./examples/backoff_sim` and read the three histograms together.

**Walkthrough focus.** The doubling loop and why it can't overflow. What the simulation shows about synchronized retries.

**Check your understanding.**
1. Full jitter can return 0 (an immediate retry). Why is that acceptable?
2. Why compute the delay on the server rather than the worker?
3. With base 1s and max 10m, at which attempt does the ceiling first hit the cap?

**Done when.** The backoff tests pass and the learner has seen the herd effect.

---

### Step 10.2 — Wire backoff into FailJob

**Build.**
1. Add `RetryDelay time.Duration` to `store.FailParams`. In the `FailJob` SQL, replace `run_at = now()` with `run_at = now() + ($5::bigint * interval '1 millisecond')` and pass `RetryDelay.Milliseconds()`. Remove the "retries immediately" part of the NOTE comment (keep the dead-letter part).
2. Add `Backoff backoff.Policy` to `queue.Options`. `Service.FailJob` computes `delay := s.opts.Backoff.Delay(in.Attempt)`. Config: `JQ_BACKOFF_BASE` (1s), `JQ_BACKOFF_MAX` (10m); validate at startup.
3. Proto: `FailJobResponse` gets `google.protobuf.Timestamp next_run_at = 2;`. `make proto-breaking`.
4. Worker logs `"job failed; retry scheduled"` with `next_run_at`.
5. Add `Backoff backoff.Policy` to `testutil.ServerOptions`, defaulting to `Base: 1ms, Max: 10ms` so tests stay fast.
6. Tests: a service test with `Rand` fixed at `0.5` and `Base = 2s` → after the first failure, `run_at ≈ now + 1s` (±200ms tolerance), and an immediate claim returns nothing.

**Verify.** `make test`. Manual: an `always_fail` job now retries with growing gaps. Read the log timestamps together. It still retries forever, which motivates Phase 11.

**Check your understanding.**
1. After this change, what stops a failing job from being claimed immediately?
2. Why use a millisecond integer for the delay instead of passing a Go `time.Time`?

**Done when.** Failures are retried with jittered exponential delays.

---

## Phase 11 — Dead-letter queue and manual replay

### Step 11.1 — Max attempts, non-retryable errors, the `dead` state

**Problem.** `always_fail` retries forever (eventually every 10 minutes, forever). That's a **poison message**. And some errors can never succeed on retry (invalid payload, unknown job type), so retrying them is pointless.

**Teach.**
- Poison messages; `max_attempts`; the dead-letter queue (DLQ) as **quarantine, not trash**. Compare SQS redrive policies and RabbitMQ dead-letter exchanges.
- **Error classification:** retryable (timeouts, 5xx) vs non-retryable (validation, 4xx, unknown type). Default to *retryable* when unsure, since it's the safer mistake.
- A `Permanent(err)` wrapper detected with `errors.As`.
- **proto3 bool pitfall:** a field named `retryable` defaults to `false`, so an old client would accidentally mark *every* failure as permanent. Name it `non_retryable` so the zero value is the safe one.
- **The reaper must respect `max_attempts` too.** A job that crashes the worker every time never reaches `FailJob`. Without this check it would crash workers forever. This is the payoff for counting attempts at claim time (3.2).
- Terminal states set `completed_at` (invariant 2).

**Build.**
1. Add `NonRetryable bool` to `store.FailParams`. New `FailJob` SQL:
   ```sql
   UPDATE jobs
   SET state        = CASE WHEN $6::boolean OR attempts >= max_attempts THEN 'dead' ELSE 'pending' END,
       run_at       = CASE WHEN $6::boolean OR attempts >= max_attempts THEN run_at
                           ELSE now() + ($5::bigint * interval '1 millisecond') END,
       completed_at = CASE WHEN $6::boolean OR attempts >= max_attempts THEN now() ELSE NULL END,
       last_error   = $4,
       lease_owner = NULL, lease_expires_at = NULL, updated_at = now()
   WHERE id = $1 AND state = 'running' AND lease_owner = $2 AND attempts = $3
   RETURNING <jobColumns("")>
   ```
   Remove the NOTE comment.
2. `ReapExpiredLeases`: `state = CASE WHEN j.attempts >= j.max_attempts THEN 'dead' ELSE 'pending' END`, `completed_at = CASE WHEN j.attempts >= j.max_attempts THEN now() ELSE NULL END`, and `last_error` set to `'lease expired on final attempt'` or the existing message accordingly.
3. Proto: `FailJobRequest` gets `bool non_retryable = 5;` with a comment explaining the default-safe naming. `make proto-breaking`.
4. `internal/worker/errors.go`: `func Permanent(err error) error`, `func IsPermanent(err error) bool`. In `execute`, an unknown type or `IsPermanent(err)` → `non_retryable = true`.
5. Add a `permanent_fail` demo handler returning `worker.Permanent(errors.New("invalid payload (demo)"))`.
6. Tests: `max_attempts = 3` with `always_fail` → `dead` after exactly 3 attempts, `completed_at` set; `permanent_fail` → `dead` after 1; reaper on an expired lease with `attempts = max_attempts` → `dead`; unknown type → `dead` after 1.
7. **Update the 7.3 worker tests:** `TestWorker_UnknownType` now expects `dead` after exactly 1 attempt; `TestWorker_PanicDoesNotKillWorker` should `WaitFor` the panic job to reach `dead` (it exhausts `max_attempts`; the test server's 1ms backoff keeps this fast).

**Verify.** `make test`. Manual: `./bin/jqctl enqueue --type always_fail --max-attempts 3` → watch three attempts with growing gaps, then the job lands in `dead`.

**Check your understanding.**
1. Why default to retryable?
2. Describe the poison pill that would loop forever without the reaper's `max_attempts` check.
3. Why `non_retryable` and not `retryable`?

**Done when.** Jobs stop after `max_attempts` or a permanent error and land in `dead`.

---

### Step 11.2 — Admin API and CLI: list and replay

**Problem.** Operators need to see what's in the DLQ, fix the root cause, and replay jobs, one at a time or in bulk.

**Teach.**
- Operating a DLQ: alert on growth, inspect, fix the root cause, *then* replay. Never auto-replay blindly.
- Replay is only safe because handlers are idempotent.
- **Keyset pagination** (`WHERE id > $after ORDER BY id LIMIT n`) vs `OFFSET` (scans and throws away rows; pages shift when rows are inserted).
- Bound bulk operations (batches with `LIMIT`) to avoid giant transactions and long-held locks.
- Admin APIs need authentication and authorization in real systems (mTLS, tokens, RBAC). That's out of scope here; note it in the README's limitations.

**Build.**
1. `internal/store/migrations/0005_add_state_queue_index.sql`:
   ```sql
   -- +goose Up
   CREATE INDEX jobs_state_queue_id_idx ON jobs (state, queue, id);
   -- +goose Down
   DROP INDEX jobs_state_queue_id_idx;
   ```
2. Store:
   - `ListJobs(ctx, ListJobsParams{State State; Queue string; AfterID int64; Limit int}) ([]Job, error)`. Use two SQL strings, one with `AND queue = $2` and one without, rather than `($2 = '' OR queue = $2)`, which stops Postgres from using the index well.
   - `ReplayJob(ctx, id) (Job, error)`:
     ```sql
     UPDATE jobs
     SET state = 'pending', attempts = 0, run_at = now(), completed_at = NULL, updated_at = now()
     WHERE id = $1 AND state = 'dead'
     RETURNING <jobColumns("")>
     ```
     No row → `ErrNotFound` if the job doesn't exist, else `ErrInvalidState`. (`last_error` is kept for history.)
   - `ReplayDeadJobs(ctx, queue, jobType string, limit int) ([]Job, error)`: the CTE pattern (`SELECT id ... WHERE state = 'dead' AND queue = $1 [AND type = $2] ORDER BY id LIMIT $n FOR UPDATE SKIP LOCKED`, then `UPDATE ... FROM` as above).
   - The service loops `ReplayDeadJobs` in batches of 1,000 until a short batch, capped at 100,000 total, and returns the count.
3. Proto:
   ```proto
   service AdminService {
     rpc ListJobs(ListJobsRequest) returns (ListJobsResponse);
     rpc ReplayJob(ReplayJobRequest) returns (ReplayJobResponse);
     rpc ReplayDeadJobs(ReplayDeadJobsRequest) returns (ReplayDeadJobsResponse);
   }
   message ListJobsRequest {
     JobState state = 1;
     string queue = 2;    // optional
     int64 after_id = 3;  // keyset cursor; 0 = from the start
     int32 limit = 4;     // 1..500, default 50
   }
   message ListJobsResponse {
     repeated Job jobs = 1;
     int64 next_after_id = 2; // 0 when there are no more pages
   }
   message ReplayJobRequest { int64 job_id = 1; }
   message ReplayJobResponse { Job job = 1; }
   message ReplayDeadJobsRequest {
     string queue = 1; // required
     string type = 2;  // optional filter
   }
   message ReplayDeadJobsResponse { int64 replayed = 1; }
   ```
   `make proto-breaking`.
4. `internal/api/admin.go` (`AdminServer`); register it in `NewGRPCServer`.
5. `jqctl`:
   - `jobs list --state dead [--queue q] [--limit 20] [--after N]`: a table with `id`, `queue`, `type`, `attempts`, `last_error` (truncated to 60 chars), `completed_at`, then a "next page: --after N" hint.
   - `dlq list` (shortcut for `jobs list --state dead`), `dlq replay <id>`, `dlq replay-all --queue q [--type t] --yes` (refuse to run without `--yes`).
6. Tests: 25 dead jobs paginated at 10 → three pages, no duplicates or gaps; replay → `pending`, `attempts = 0`; replaying a non-dead job → `FailedPrecondition`; `replay-all --type X` only touches type X.

**Verify.** `make test`; manually fill the DLQ (`always_fail --max-attempts 1` a few times), list it, replay one, replay all.

**Check your understanding.**
1. Why keyset pagination instead of `OFFSET`?
2. What must be true before you replay a batch of dead jobs?
3. Why does `replay-all` require `--yes`?

**Done when.** Operators can list and replay dead jobs from the CLI.

---
## Phase 12 — Priority and delayed jobs

### Step 12.1 — Priority

**Problem.** A password-reset email shouldn't wait behind 50,000 newsletter jobs in the same queue.

**Teach.**
- Two levels of priority in this system: **queue order** per worker (`JQ_WORKER_QUEUES=critical,default` checks `critical` first) and a **per-job priority** 0–9 within a queue (higher runs first, default 0).
- **Strict priority and starvation:** if high-priority work never stops arriving, low-priority work never runs. Mitigations: *aging* (priority grows with wait time), *weighted* queues (Sidekiq-style), separate worker pools. We document starvation rather than solve it.
- **The index must match the `ORDER BY` exactly**, including `DESC`, or Postgres has to sort every pending row before taking the first few.
- Changing indexes on a live production table: `CREATE INDEX CONCURRENTLY` (can't run inside a transaction; goose supports `-- +goose NO TRANSACTION`). We use a plain `CREATE INDEX` in dev and explain the production approach.

**Build.**
1. `internal/store/migrations/0006_add_priority.sql`:
   ```sql
   -- +goose Up
   ALTER TABLE jobs ADD COLUMN priority SMALLINT NOT NULL DEFAULT 0
       CONSTRAINT jobs_priority_check CHECK (priority BETWEEN 0 AND 9);
   DROP INDEX jobs_ready_idx;
   CREATE INDEX jobs_ready_idx ON jobs (queue, priority DESC, run_at, id) WHERE state = 'pending';

   -- +goose Down
   DROP INDEX jobs_ready_idx;
   CREATE INDEX jobs_ready_idx ON jobs (queue, run_at, id) WHERE state = 'pending';
   ALTER TABLE jobs DROP COLUMN priority;
   ```
2. Add `Priority int` to `Job` and `jobColumnNames`; add `Priority int` to `EnqueueParams` and the INSERT. Change the claim's `ORDER BY` to `priority DESC, run_at, id`.
3. Proto: `EnqueueRequest` gets `int32 priority = 6; // 0..9, higher runs first`; `Job` gets `int32 priority = 13;`. Validate 0–9. `make proto-breaking`.
4. `jqctl enqueue --priority N`; show priority in `jobs list`.
5. Tests (claim with `Limit: 1` per call, because `RETURNING` order isn't guaranteed): enqueue priorities 0, 9, 5 → claimed in order 9, 5, 0; equal priority → FIFO by `run_at`, then `id`.
6. Optional starvation demo: one worker with concurrency 1, one priority-0 job, then a loop enqueuing priority-9 jobs → the priority-0 job never runs until the loop stops.

**Verify.** `make test`; `make migrate-up`; `\d jobs`.

**Check your understanding.**
1. What happens to query speed if the index is `(queue, priority, run_at, id)` (ascending) but we `ORDER BY priority DESC`?
2. How would you prevent starvation if it became a real problem?

**Done when.** Higher-priority jobs are claimed first.

---

### Step 12.2 — Delayed jobs

**Problem.** "Send a reminder in 24 hours." "Retry this webhook at 9am."

**Teach.**
- Delayed jobs are just `run_at` in the future; the claim query already filters `run_at <= now()`.
- **Clock skew:** if a client sends an absolute `run_at` from a laptop whose clock is 5 minutes fast, the job runs 5 minutes early. A *relative* `delay` is converted with the database clock (Law 2) and is immune. Offer both through a protobuf `oneof`.
- Validation: a `run_at` in the past just means "now"; cap delays at 365 days.

**Build.**
1. `EnqueueParams` gets `RunAt *time.Time` and `Delay time.Duration`. In the INSERT: `run_at = COALESCE($n::timestamptz, now() + ($m::bigint * interval '1 millisecond'))`.
2. Proto `EnqueueRequest`:
   ```proto
   oneof schedule {
     google.protobuf.Timestamp run_at = 7;
     google.protobuf.Duration delay = 8;
   }
   ```
   `make proto-breaking`.
3. `jqctl enqueue --delay 30s` or `--at 2026-10-10T09:00:00Z` (mutually exclusive).
4. Tests: `delay = 1h` → not claimable; move `run_at` into the past with SQL → claimable; a past `run_at` is accepted and claimable at once; delay > 365 days → `InvalidArgument`.

**Verify.** `make test`. Manual: `./bin/jqctl enqueue --type noop --delay 10s` → the worker picks it up after ~10s, plus up to 2s of idle polling. Explain where that extra latency comes from.

**Check your understanding.**
1. Why is `delay` safer than `run_at` for most clients?
2. What's the worst-case gap between a job's `run_at` and its execution right now, and why?

**Done when.** Jobs can be delayed by duration or timestamp.

---

### Step 12.3 — Query-plan lab with `EXPLAIN ANALYZE`

**Problem.** We've *assumed* our indexes work. Let's prove it, and find where they don't.

**Teach.**
- `EXPLAIN` (the plan) vs `EXPLAIN ANALYZE` (runs the query and reports real timings) vs `BUFFERS` (pages read). Reading a plan: Index Scan vs Seq Scan vs Bitmap Heap Scan, Sort nodes, `LockRows`, "Rows Removed by Filter".
- **Gotcha:** `EXPLAIN ANALYZE` on an `UPDATE` *actually performs the update*. Wrap it in `BEGIN; ... ROLLBACK;`.
- Planner statistics and `ANALYZE`.
- **A hidden cost:** future-dated jobs sit inside the `pending` partial index. With priority first in the index, the claim may have to step over thousands of not-yet-due rows ("Rows Removed by Filter"). Fixes: a separate `scheduled` state with a promoter loop, Redis delayed sets (Phase 17), or a different index order (which loses priority ordering).
- Dead tuples and vacuum: `pg_stat_user_tables.n_dead_tup`, why queue tables churn, and why long transactions hurt.

**Build.** `scripts/sql/explain_lab.sql`, a commented script the AI runs section by section in psql while explaining each result:
1. Seed 200,000 due jobs in queue `lab`:
   ```sql
   INSERT INTO jobs (queue, type, payload, priority, run_at)
   SELECT 'lab', 'noop', '{}', (random() * 9)::int, now() - random() * interval '1 hour'
   FROM generate_series(1, 200000);
   ANALYZE jobs;
   ```
2. `BEGIN; EXPLAIN (ANALYZE, BUFFERS) <the claim CTE for queue 'lab', limit 10>; ROLLBACK;` → expect an Index Scan on `jobs_ready_idx`, no Sort, and a `LockRows` node.
3. Seed 200,000 *future* priority-9 jobs (`run_at = now() + interval '1 day'`), `ANALYZE`, re-run → note "Rows Removed by Filter" and the slower time.
4. Drop `jobs_ready_idx` inside a transaction, re-run, see Seq Scan + Sort, then `ROLLBACK` (the index comes back).
5. Simulate churn (this respects the `CHECK` constraints): `UPDATE jobs SET state = 'succeeded', completed_at = now(), updated_at = now() WHERE id IN (SELECT id FROM jobs WHERE queue = 'lab' AND state = 'pending' LIMIT 5000);` Then `SELECT n_live_tup, n_dead_tup FROM pg_stat_user_tables WHERE relname = 'jobs';`; run `VACUUM jobs;` and check again.
6. Clean up: `DELETE FROM jobs WHERE queue = 'lab'; VACUUM ANALYZE jobs;`.
7. Write the findings into a "Performance notes" section of `docs/ARCHITECTURE.md`.

**Verify.** Each section's output, explained.

**Check your understanding.**
1. Why wrap `EXPLAIN ANALYZE` of an `UPDATE` in a rolled-back transaction?
2. What does "Rows Removed by Filter: 200000" tell you?
3. Which fix for future-dated jobs will Phase 17 implement?

**Done when.** The learner can read the claim query's plan and explain the future-jobs problem.

---

## Phase 13 — Cron jobs

### Step 13.1 — Leader election with advisory locks

**Problem.** Some work must be done by **exactly one** server replica at a time. If two replicas both fire a cron schedule, the job runs twice.

**Teach.**
- Leader election and where it usually comes from: etcd, ZooKeeper, Consul, Kubernetes Leases, or database locks.
- **Postgres advisory locks:** application-defined locks on a number. Session-level (held until unlocked or the connection closes) vs transaction-level. `pg_try_advisory_lock(key)` returns immediately with true or false.
- **Pitfall with pools:** a session-level lock belongs to *one connection*. With a pool, the next query may run on a different connection. So the leader must `pool.Acquire()` a **dedicated connection** and hold it for as long as it leads.
- Failure cases: the leader crashes → its connection closes → Postgres releases the lock → another replica takes over. A **network partition** may leave the old leader *believing* it still leads for a while → two leaders briefly. **Never rely on leader election alone for correctness.** The protected work must also be idempotent (cron uses idempotency keys in 13.2). Same theme as fencing in 3.3.
- Check the leader's connection regularly and step down if it fails.

**Build.**
1. `internal/leader/leader.go`:
   ```go
   // SchedulerLockKey is an arbitrary application-chosen advisory lock key.
   const SchedulerLockKey int64 = 0x6A6F6271 // "jobq"

   type Elector struct {
       pool          *pgxpool.Pool
       lockKey       int64
       retryInterval time.Duration // 5s (tests: 100ms)
       checkInterval time.Duration // 5s (tests: 100ms)
       logger        *slog.Logger
       isLeader      atomic.Bool
   }
   type Option func(*Elector)
   func WithRetryInterval(d time.Duration) Option // tests use 100ms
   func WithCheckInterval(d time.Duration) Option // tests use 100ms
   func New(pool *pgxpool.Pool, lockKey int64, logger *slog.Logger, opts ...Option) *Elector
   func (e *Elector) IsLeader() bool
   // Run blocks until ctx is done. While this process holds the lock it calls lead(leaderCtx);
   // leaderCtx is cancelled as soon as leadership is lost.
   func (e *Elector) Run(ctx context.Context, lead func(ctx context.Context)) error
   ```
   Algorithm for `Run` (loop until `ctx` is done):
   1. `conn, err := e.pool.Acquire(ctx)`; on error, wait `retryInterval` and retry.
   2. `SELECT pg_try_advisory_lock($1)`. If false: `conn.Release()`, wait `retryInterval`, continue.
   3. Became leader: `isLeader.Store(true)`, log it, create `leaderCtx, cancel := context.WithCancel(ctx)`, run `lead(leaderCtx)` in a goroutine that closes a `done` channel when it returns.
   4. Watch loop: on `ctx.Done()` → stop; on `done` → stop; every `checkInterval`, `conn.Ping(ctx)` with a short timeout; on failure log "lost leader connection" and stop.
   5. Step down: `cancel()`, wait for `done`, `isLeader.Store(false)`. Unlock with `SELECT pg_advisory_unlock($1)` using `context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)` and ignore its error. If the ping failed, close the underlying connection (`conn.Conn().Close(...)`) before `conn.Release()` so the pool discards it.
   6. Note: `isLeader` is set to false **before** unlocking, so two electors in the same process can never both report leadership.
2. Tests (`internal/leader/leader_test.go`):
   - Two electors on the same key, sampling `IsLeader()` every 10ms for 1s: never both true, and exactly one true after startup.
   - Cancel the leader's ctx → the other becomes leader within ~200ms.
   - Kill the leader's connection from the database side: find its PID with `SELECT pid FROM pg_locks WHERE locktype = 'advisory' AND database = (SELECT oid FROM pg_database WHERE datname = current_database())`, then `SELECT pg_terminate_backend(pid)`. The old leader steps down within `checkInterval`, and the other takes over.

**Verify.** `make test` (run the leader tests with `-count=5` to shake out timing flakiness).

**Walkthrough focus.** The goroutines involved (lead function, watch loop) and how each stops. Why the dedicated connection.

**Check your understanding.**
1. Why can't the elector just run `pg_try_advisory_lock` through the pool like any other query?
2. What happens when the leader's database connection dies?
3. Why is leader election alone not enough to guarantee a cron fires once?

**Done when.** Exactly one elector leads at a time and failover works.

---

### Step 13.2 — Cron schedules table and scheduler loop

**Problem.** "Run the cleanup job every night at 2am." Schedules must survive restarts, fire once per tick even with several server replicas, and behave sensibly after downtime.

**Teach.**
- Cron expressions (5 fields: minute, hour, day-of-month, month, day-of-week) and computing `Next(t)`.
- **Persist `next_run_at`** in the database, so a restart neither loses nor double-fires a tick.
- **Atomic fire:** enqueue the job *and* advance `next_run_at` in the same transaction.
- **Idempotency key per tick:** `cron:<name>:<unix seconds of the tick>`, a safety net in case two leaders ever overlap.
- **Misfire policies** after downtime: fire every missed tick, fire once, or skip. We **fire once, then jump to the next future tick** ("coalesce"). Why: a 3-hour outage of an every-minute schedule shouldn't unleash 180 jobs.
- One clock: read `now()` from the database inside the transaction and compute `Next` from that value.
- Time zones and DST: default to UTC; `CRON_TZ=America/New_York 0 9 * * *` is supported by the parser. Mention that 2:30am doesn't exist on spring-forward days.
- The library: `robfig/cron/v3` (parser only). It's unmaintained, so wrap parsing in `recover` defensively. `netresearch/go-cron` is an API-compatible alternative.

**Build.**
1. `go get github.com/robfig/cron/v3`.
2. `internal/store/migrations/0007_create_cron_schedules.sql`:
   ```sql
   -- +goose Up
   CREATE TABLE cron_schedules (
       id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
       name         TEXT        NOT NULL UNIQUE,
       cron_expr    TEXT        NOT NULL,
       queue        TEXT        NOT NULL,
       type         TEXT        NOT NULL,
       payload      JSONB       NOT NULL DEFAULT '{}'::jsonb,
       priority     SMALLINT    NOT NULL DEFAULT 0 CHECK (priority BETWEEN 0 AND 9),
       max_attempts INT         NOT NULL DEFAULT 5 CHECK (max_attempts >= 1),
       enabled      BOOLEAN     NOT NULL DEFAULT true,
       next_run_at  TIMESTAMPTZ NOT NULL,
       last_run_at  TIMESTAMPTZ,
       created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
       updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
   );
   CREATE INDEX cron_schedules_due_idx ON cron_schedules (next_run_at) WHERE enabled;

   -- +goose Down
   DROP TABLE cron_schedules;
   ```
3. `internal/cronsched/parse.go`: `func Parse(expr string) (s cron.Schedule, err error)` using `cron.ParseStandard`, with a deferred `recover` that converts a panic into an error.
4. `internal/store/cron.go`:
   - `type CronSchedule struct { ID int64; Name, CronExpr, Queue, Type string; Payload []byte; Priority, MaxAttempts int; Enabled bool; NextRunAt time.Time; LastRunAt *time.Time; CreatedAt, UpdatedAt time.Time }`.
   - `func (s *Store) Now(ctx context.Context) (time.Time, error)` → `SELECT now()`.
   - `UpsertCronSchedule` (`INSERT ... ON CONFLICT (name) DO UPDATE SET ... RETURNING ...`), `ListCronSchedules`, `DeleteCronSchedule(name)` (`ErrNotFound` if missing).
   - `LockDueCronSchedules(ctx, limit int) ([]CronSchedule, time.Time, error)` (**must be called inside `InTx`**): first `SELECT now()`, then `SELECT ... FROM cron_schedules WHERE enabled AND next_run_at <= now() ORDER BY next_run_at LIMIT $1 FOR UPDATE SKIP LOCKED`.
   - `AdvanceCronSchedule(ctx, id int64, lastRunAt, nextRunAt time.Time) error`; `DisableCronSchedule(ctx, id int64) error`.
   - Extend `EnqueueParams` usage so cron can pass `Priority` and `IdempotencyKey`.
5. `internal/cronsched/scheduler.go`:
   ```go
   type Scheduler struct {
       store    *store.Store
       interval time.Duration // 1s
       logger   *slog.Logger
   }
   func New(st *store.Store, interval time.Duration, logger *slog.Logger) *Scheduler
   func (s *Scheduler) Run(ctx context.Context)                       // ticker loop calling TickOnce; returns when ctx is done
   func (s *Scheduler) TickOnce(ctx context.Context) (fired int, err error)
   ```
   `TickOnce`: inside `store.InTx`: `due, now := LockDueCronSchedules(100)`; for each schedule: parse its expression (if it fails, log and `DisableCronSchedule`: fail safe, don't crash the loop); `key := fmt.Sprintf("cron:%s:%d", c.Name, c.NextRunAt.Unix())`; `tx.Enqueue(...)` with that key (if `created == false`, log "duplicate fire suppressed"); `next := sched.Next(now.UTC())` (the first tick strictly after *now*, which coalesces missed ticks; `.UTC()` matters because pgx returns `timestamptz` in Go's local time zone and the parser evaluates schedules without `CRON_TZ` in the location of the time you pass); `tx.AdvanceCronSchedule(c.ID, c.NextRunAt, next)`.
   Leave a comment: "Phase 17 adds an after-commit callback here to notify the Redis index."
6. `cmd/server/main.go`: create `leader.New(pool, leader.SchedulerLockKey, logger)` and add to the errgroup: `elector.Run(gctx, func(lctx context.Context) { scheduler.Run(lctx) })`.
7. Tests:
   - `Parse("not a cron")` → error.
   - A `* * * * *` schedule whose `next_run_at` is set 5 minutes in the past (via SQL) → `TickOnce` fires **exactly 1** job with key `cron:<name>:<ts>`, and the new `next_run_at` is in the future and ≤ now + 1 minute.
   - `TickOnce` again immediately → 0 fired.
   - Two `TickOnce` calls in parallel goroutines on a freshly due schedule → 1 job in total.
   - Safety net: reset `next_run_at` to the *same* old value via SQL and tick again → no new job (the idempotency key dedupes it).

**Verify.** `make test`.

**Walkthrough focus.** The `TickOnce` transaction: what's locked, what's written, and what happens if the process crashes halfway (the whole transaction rolls back, so nothing is half-done).

**Check your understanding.**
1. The server is down for 3 hours with an every-minute schedule. How many jobs fire when it comes back, and why?
2. Which two mechanisms stop a double fire, and which failure does each handle?
3. Why compute `Next` from the database's `now()`?

**Done when.** Schedules fire once per tick, survive restarts, and coalesce missed ticks.

---

### Step 13.3 — Cron admin API and CLI

**Build.**
1. Proto, added to `AdminService`:
   ```proto
   rpc UpsertCronSchedule(UpsertCronScheduleRequest) returns (UpsertCronScheduleResponse);
   rpc ListCronSchedules(ListCronSchedulesRequest) returns (ListCronSchedulesResponse);
   rpc DeleteCronSchedule(DeleteCronScheduleRequest) returns (DeleteCronScheduleResponse);

   message CronSchedule {
     string name = 1;
     string cron_expr = 2;
     string queue = 3;
     string type = 4;
     bytes payload = 5;
     int32 priority = 6;
     int32 max_attempts = 7;
     bool disabled = 8; // named so the zero value (false) means "enabled"
     google.protobuf.Timestamp next_run_at = 9;  // output only
     google.protobuf.Timestamp last_run_at = 10; // output only
   }
   message UpsertCronScheduleRequest { CronSchedule schedule = 1; }
   message UpsertCronScheduleResponse { CronSchedule schedule = 1; }
   message ListCronSchedulesRequest {}
   message ListCronSchedulesResponse { repeated CronSchedule schedules = 1; }
   message DeleteCronScheduleRequest { string name = 1; }
   message DeleteCronScheduleResponse {}
   ```
   Point out that this is the same default-safe naming lesson as `non_retryable`. `make proto-breaking`.
2. Service: validate (name matches `^[a-z0-9][a-z0-9_-]{0,63}$`, the expression parses, and the queue/type/payload/priority rules from 5.2), compute `next_run_at = sched.Next(now.UTC())` where `now` comes from `store.Now(ctx)`, upsert.
3. `jqctl cron add --name nightly-noop --expr "*/1 * * * *" --type noop [--queue] [--payload] [--priority] [--disabled]`, `jqctl cron list` (table: name, expr, enabled, next_run_at, last_run_at), `jqctl cron rm <name>`.
4. Tests for the API: invalid expression → `InvalidArgument`; upsert twice → updated, not duplicated; delete a missing schedule → `NotFound`.

**Verify.**
- `make test`.
- Manual: add an every-minute `noop` schedule and watch a job appear each minute.
- Run a second replica: `make run-server JQ_GRPC_LISTEN_ADDR=:50052` (after Phase 15, also `JQ_METRICS_ADDR=:2122`). Only one logs "became leader". Kill the leader (Ctrl-C) → the other takes over within ~5s. Still exactly one job per minute (check `idempotency_key` values in psql).

**Check your understanding.**
1. Why is the field called `disabled` and not `enabled`?
2. During failover, what would happen if a tick were due at that exact moment?

**Done when.** Schedules can be managed from the CLI, and failover between two replicas works.

---

## Phase 14 — DAG workflows (job dependencies)

### Step 14.1 — DAG validation (pure Go)

**Problem.** Workflows like `extract → (transform_a, transform_b) → load` need jobs that wait for other jobs. A cycle (A waits for B, B waits for A) would deadlock forever, so we must reject invalid graphs up front.

**Teach.**
- A workflow is a **DAG** (directed acyclic graph): nodes are jobs, edges are "depends on".
- **Topological order:** an ordering where every node comes after its dependencies.
- **Kahn's algorithm,** step by step on the diamond example: compute in-degrees, start with zero-in-degree nodes, remove them and decrement their children. If some nodes are never processed, there's a cycle.
- Pure functions (no I/O) are the easiest code to test; use table-driven tests.

**Build.**
1. `internal/dag/dag.go`:
   ```go
   type Node struct {
       Key       string
       DependsOn []string
   }

   var (
       ErrEmpty               = errors.New("workflow has no nodes")
       ErrTooLarge            = errors.New("workflow has more than 1000 nodes")
       ErrDuplicateKey        = errors.New("duplicate node key")
       ErrUnknownDependency   = errors.New("unknown dependency")
       ErrSelfDependency      = errors.New("node depends on itself")
       ErrDuplicateDependency = errors.New("duplicate dependency")
       ErrCycle               = errors.New("workflow contains a cycle")
   )

   // TopoSort returns node keys ordered so every node comes after its dependencies.
   // Ties keep input order, so results are deterministic.
   func TopoSort(nodes []Node) ([]string, error)
   ```
   Wrap sentinels with details: `fmt.Errorf("%w: %q depends on %q", ErrUnknownDependency, n.Key, dep)`. For cycles, list the keys that couldn't be ordered.
2. Table-driven tests: single node; chain; diamond; two independent nodes; 2-cycle; 3-cycle; self-dependency; unknown dependency; duplicate key; duplicate dependency; empty; 1,001 nodes; a 1,000-node chain (finishes quickly).

**Verify.** `go test ./internal/dag -v`.

**Check your understanding.**
1. Walk Kahn's algorithm through the diamond by hand.
2. How does Kahn's algorithm detect a cycle?

**Done when.** Validation rejects every bad graph in the tests and orders good ones deterministically.

---

### Step 14.2 — Schema and workflow submission

**Problem.** Store a whole workflow atomically: either every job and edge is saved, or none.

**Teach.**
- Graphs in relational databases: an **edge table** (`job_dependencies`).
- **Counter-based readiness:** each job stores `pending_dependencies`. Checking readiness is O(1), but the counter must be updated atomically (14.3). The alternative, counting unfinished parents on every check, is simpler but slower.
- One transaction for the whole submission.
- Foreign keys and `ON DELETE CASCADE`. **Postgres doesn't index foreign-key columns automatically**, so we add the indexes ourselves.
- Invariant 5 enforced by a `CHECK`: `blocked` ⇔ `pending_dependencies > 0`.

**Build.**
1. `internal/store/migrations/0008_create_workflows.sql`:
   ```sql
   -- +goose Up
   CREATE TABLE workflows (
       id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
       name       TEXT        NOT NULL,
       created_at TIMESTAMPTZ NOT NULL DEFAULT now()
   );

   ALTER TABLE jobs
       ADD COLUMN workflow_id BIGINT REFERENCES workflows(id) ON DELETE CASCADE,
       ADD COLUMN workflow_node_key TEXT,
       ADD COLUMN pending_dependencies INT NOT NULL DEFAULT 0;

   ALTER TABLE jobs
       ADD CONSTRAINT jobs_pending_dependencies_check CHECK (pending_dependencies >= 0),
       -- Invariant 5: blocked if and only if waiting on dependencies.
       ADD CONSTRAINT jobs_blocked_deps_check CHECK ((state = 'blocked') = (pending_dependencies > 0));

   CREATE INDEX jobs_workflow_id_idx ON jobs (workflow_id) WHERE workflow_id IS NOT NULL;
   CREATE UNIQUE INDEX jobs_workflow_node_key_uq ON jobs (workflow_id, workflow_node_key)
       WHERE workflow_id IS NOT NULL;

   CREATE TABLE job_dependencies (
       job_id            BIGINT NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
       depends_on_job_id BIGINT NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
       PRIMARY KEY (job_id, depends_on_job_id),
       CONSTRAINT job_dependencies_no_self CHECK (job_id <> depends_on_job_id)
   );
   CREATE INDEX job_dependencies_depends_on_idx ON job_dependencies (depends_on_job_id);

   -- +goose Down
   DROP TABLE job_dependencies;
   ALTER TABLE jobs DROP CONSTRAINT jobs_blocked_deps_check;
   ALTER TABLE jobs DROP CONSTRAINT jobs_pending_dependencies_check;
   ALTER TABLE jobs DROP COLUMN pending_dependencies, DROP COLUMN workflow_node_key, DROP COLUMN workflow_id;
   DROP TABLE workflows;
   ```
2. Add `WorkflowID *int64`, `WorkflowNodeKey *string`, `PendingDependencies int` to `Job` and `jobColumnNames`.
3. `internal/store/workflows.go`:
   - `type WorkflowNodeParams struct { Key, Queue, Type string; Payload []byte; Priority, MaxAttempts int; DependsOn []string }`.
   - `CreateWorkflow(ctx, name string, nodes []WorkflowNodeParams, order []string) (workflowID int64, jobIDs map[string]int64, err error)` (**call inside `InTx`**): insert the workflow; insert each job in topological `order` with `state = 'blocked'` and `pending_dependencies = len(DependsOn)` if it has dependencies, else `state = 'pending'` and 0; then insert all edges. (A simple loop of INSERTs is fine for ≤ 1,000 nodes; mention `pgx.Batch` as an optimization.)
   - `GetWorkflow(ctx, id) (name string, jobs []Job, err error)`.
4. `queue.Service.SubmitWorkflow(ctx, name, nodes)`: validate each node with the 5.2 rules; `dag.TopoSort` (map its errors to `ValidationError`); `InTx` → `CreateWorkflow`. `GetWorkflow` derives the overall state: any `dead` or `cancelled` → `FAILED`; all `succeeded` → `SUCCEEDED`; otherwise `RUNNING`.
5. Proto, added to `ProducerService`:
   ```proto
   rpc SubmitWorkflow(SubmitWorkflowRequest) returns (SubmitWorkflowResponse);
   rpc GetWorkflow(GetWorkflowRequest) returns (GetWorkflowResponse);

   message WorkflowNode {
     string key = 1;                 // unique within the workflow, e.g. "extract"
     string queue = 2;
     string type = 3;
     bytes payload = 4;
     int32 priority = 5;
     int32 max_attempts = 6;
     repeated string depends_on = 7; // keys of other nodes
   }
   message SubmitWorkflowRequest {
     string name = 1;
     repeated WorkflowNode nodes = 2;
   }
   message SubmitWorkflowResponse {
     int64 workflow_id = 1;
     map<string, int64> job_ids = 2; // node key -> job id
   }
   message GetWorkflowRequest { int64 workflow_id = 1; }
   enum WorkflowState {
     WORKFLOW_STATE_UNSPECIFIED = 0;
     WORKFLOW_STATE_RUNNING = 1;
     WORKFLOW_STATE_SUCCEEDED = 2;
     WORKFLOW_STATE_FAILED = 3;
   }
   message GetWorkflowResponse {
     int64 workflow_id = 1;
     string name = 2;
     WorkflowState state = 3;
     repeated Job jobs = 4;
   }
   ```
   `Job` gets `int64 workflow_id = 14;` and `string workflow_node_key = 15;`. `make proto-breaking`.
6. `examples/workflows/diamond.json`:
   ```json
   {
     "name": "diamond-demo",
     "nodes": [
       {"key": "extract",     "type": "sleep", "payload": {"ms": 500}},
       {"key": "transform_a", "type": "sleep", "payload": {"ms": 800}, "depends_on": ["extract"]},
       {"key": "transform_b", "type": "sleep", "payload": {"ms": 300}, "depends_on": ["extract"]},
       {"key": "load",        "type": "noop",  "depends_on": ["transform_a", "transform_b"]}
     ]
   }
   ```
   `jqctl workflow submit -f <file>` (convert each `payload` object to JSON bytes; default queue `default`) and `jqctl workflow get <id>` (a table of key, job id, state, attempts, plus the overall state).
7. Tests: submit the diamond → `extract` is `pending`, the others `blocked` with `pending_dependencies` 1, 1, 2; a cyclic workflow → `InvalidArgument` **and zero rows inserted** (proof that the transaction rolled back).

**Verify.** `make test`; submit the diamond with `jqctl` and `workflow get` it. (Nothing past `extract` will run yet; that's 14.3.)

**Check your understanding.**
1. Why insert jobs in topological order?
2. What does the `jobs_blocked_deps_check` constraint protect against?
3. Why add an index on `depends_on_job_id` ourselves?

**Done when.** Workflows are stored atomically and invalid ones are rejected without side effects.

---

### Step 14.3 — Unblocking children when a job succeeds

**Problem.** When `extract` succeeds, both transforms must become `pending`. `load` must become `pending` only after **both** transforms succeed, even if they finish at the same instant on two different workers.

**Teach.**
- **The lost-update anomaly:** two transactions both read `pending_dependencies = 2`, both write 1, and `load` never unblocks. Draw the timeline.
- **The fix:** an in-place atomic update, `SET pending_dependencies = pending_dependencies - 1`. The row lock serializes the two updates, and under READ COMMITTED the second one re-reads the latest row version. Contrast with read-modify-write in Go.
- **In an `UPDATE`, every expression on the right of `SET` sees the OLD row.** So "this was the last dependency" is written `CASE WHEN pending_dependencies = 1 ...` (old value 1 → new value 0).
- Do it **in the same transaction** as marking the parent `succeeded`, so you can never get "parent succeeded but children still blocked".
- Idempotency: completing the same job twice can't double-decrement, because the fenced `CompleteJob` only succeeds once.
- **Deadlocks:** if two transactions lock the same child rows in different orders, Postgres detects it and aborts one (`40P01`). Locking children in `ORDER BY id` gives a consistent order.

**Build.**
1. Store: `func (s *Store) UnblockChildren(ctx context.Context, parentID int64) ([]Job, error)` (**call inside `InTx`**):
   ```sql
   WITH children AS (
       SELECT j.id
       FROM job_dependencies d
       JOIN jobs j ON j.id = d.job_id
       WHERE d.depends_on_job_id = $1 AND j.state = 'blocked'
       ORDER BY j.id
       FOR UPDATE OF j
   )
   UPDATE jobs AS j
   SET pending_dependencies = j.pending_dependencies - 1,
       state  = CASE WHEN j.pending_dependencies = 1 THEN 'pending' ELSE 'blocked' END,
       run_at = CASE WHEN j.pending_dependencies = 1 THEN now() ELSE j.run_at END,
       updated_at = now()
   FROM children
   WHERE j.id = children.id
   RETURNING <jobColumns("j.")>
   ```
   Return all updated rows; the caller filters for `state == pending` to get newly unblocked jobs.
2. Service: `CompleteJob` becomes `InTx`: `tx.CompleteJob(...)` (fenced) → if `job.WorkflowID != nil`, `tx.UnblockChildren(job.ID)`. Return a result containing the completed job and the newly unblocked jobs (Phase 15 counts them; Phase 17 notifies Redis about them).
3. Tests:
   - Diamond, step by step: complete `extract` → both transforms `pending`, `load` still `blocked` with 2; complete `transform_a` → `load` `blocked` with 1; complete `transform_b` → `load` `pending` with 0.
   - **Concurrency:** a fan-in workflow of 50 parents → 1 child. Claim all 50, then complete all 50 concurrently from 50 goroutines. The child ends `pending` with `pending_dependencies = 0` and no errors. Repeat the scenario 20 times in a loop.
   - End to end with a real worker: the diamond completes, and `load.completed_at` is after both transforms' `completed_at`.

**Verify.** `make test` (run the concurrency test with `-count=3`). Manual: submit the diamond with a worker running and watch it complete in order.

**Walkthrough focus.** The unblock SQL line by line, especially the OLD-value semantics of `CASE`. Why it's in the same transaction as `CompleteJob`.

**Check your understanding.**
1. Draw the lost-update timeline if we read the counter in Go and wrote it back.
2. Why `CASE WHEN pending_dependencies = 1` and not `= 0`?
3. Why lock children `ORDER BY id`?

**Done when.** Children unblock exactly once, even under concurrent completion.

---

### Step 14.4 — Failure propagation and workflow replay

**Problem.** If `transform_a` dies, `load` can never run. It would sit `blocked` forever. And if an operator fixes the bug and replays `transform_a`, its downstream jobs should come back to life.

**Teach.**
- **Recursive CTEs** (`WITH RECURSIVE`) to walk all descendants in a graph. `UNION` (not `UNION ALL`) avoids visiting a node twice in diamond shapes.
- Cascading cancellation semantics: *descendants* are cancelled, but **sibling branches keep running** (`transform_b` still completes). The alternative is failing the whole workflow fast; discuss the trade-off.
- **Recompute from the source of truth:** on replay, rebuild `pending_dependencies` by counting parents that haven't succeeded, rather than trying to re-increment counters.

**Build.**
1. Store `CancelDescendants(ctx, jobID int64) ([]Job, error)` (**inside `InTx`**):
   ```sql
   WITH RECURSIVE descendants AS (
       SELECT d.job_id FROM job_dependencies d WHERE d.depends_on_job_id = $1
       UNION
       SELECT d.job_id FROM job_dependencies d JOIN descendants x ON d.depends_on_job_id = x.job_id
   )
   UPDATE jobs
   SET state = 'cancelled',
       pending_dependencies = 0,
       completed_at = now(),
       updated_at = now(),
       last_error = 'cancelled: upstream job ' || $1::text || ' failed'
   WHERE id IN (SELECT job_id FROM descendants) AND state = 'blocked'
   RETURNING <jobColumns("")>
   ```
2. Hook it in **everywhere a workflow job can become `dead`**, inside the same transaction:
   - `Service.FailJob`: `InTx` → `tx.FailJob` → if the new state is `dead` and `WorkflowID != nil` → `tx.CancelDescendants`.
   - `Service.ReapOnce`: run each batch inside `InTx`; for each returned job that is `dead` with a workflow → `tx.CancelDescendants`.
3. Store `RestoreDescendants(ctx, jobID int64) ([]Job, error)` (**inside `InTx`**, called after the job itself has been replayed to `pending`):
   ```sql
   WITH RECURSIVE descendants AS (
       SELECT d.job_id FROM job_dependencies d WHERE d.depends_on_job_id = $1
       UNION
       SELECT d.job_id FROM job_dependencies d JOIN descendants x ON d.depends_on_job_id = x.job_id
   )
   UPDATE jobs AS j
   SET state = 'blocked',
       completed_at = NULL,
       last_error = NULL,
       attempts = 0,
       pending_dependencies = (
           SELECT count(*)
           FROM job_dependencies d
           JOIN jobs p ON p.id = d.depends_on_job_id
           WHERE d.job_id = j.id AND p.state <> 'succeeded'
       ),
       updated_at = now()
   WHERE j.id IN (SELECT job_id FROM descendants) AND j.state = 'cancelled'
   RETURNING <jobColumns("j.")>
   ```
   Explain why every restored job ends with `pending_dependencies ≥ 1`, so the `CHECK` is satisfied: each has at least one non-succeeded ancestor on the path (the replayed job itself, or another restored job).

   **Race to prevent (teach it):** a sibling parent (e.g. a still-running `transform_b`) might complete *during* the replay. Its `UnblockChildren` skips `load` (still `cancelled` in its snapshot), while the restore counts `transform_b` as unfinished, so `load` would wait forever on a dependency that already succeeded. Fix: in the same transaction, **before** the `UPDATE`, lock every parent of every descendant with a separate statement:
   ```sql
   WITH RECURSIVE descendants AS ( ...same CTE as above... )
   SELECT p.id
   FROM job_dependencies d
   JOIN jobs p ON p.id = d.depends_on_job_id
   WHERE d.job_id IN (SELECT job_id FROM descendants)
   ORDER BY p.id
   FOR SHARE OF p
   ```
   A parent that's mid-completion holds a row lock, so this waits for it to commit; the following `UPDATE` is a new statement (new snapshot under READ COMMITTED) and counts it correctly. A parent that tries to complete *after* this point blocks on the share lock until the replay commits, then sees `load` as `blocked` and decrements it normally.
4. `Service.ReplayJob` and `Service.ReplayDeadJobs`: run in `InTx`; for every replayed job with a workflow → `tx.RestoreDescendants`.
5. Tests:
   - Diamond where `transform_a` uses a test handler that fails permanently: `transform_a` → `dead`, `load` → `cancelled` (`pending_dependencies = 0`, `completed_at` set), `transform_b` still `succeeded`, workflow state `FAILED`.
   - Replay `transform_a` after "fixing" it (flip a flag the test handler reads): `load` returns to `blocked` with 1 → `transform_a` succeeds → `load` runs → workflow `SUCCEEDED`.
   - Chain `a → b → c → d`: `a` dies → `b`, `c`, `d` all `cancelled`; replay `a` → `b` `blocked` (1), `c` `blocked` (1), `d` `blocked` (1); everything then completes in order.

**Verify.** `make test`. Manual: a diamond where one transform is `always_fail --max-attempts 1`; watch `workflow get` show the cancellation; then replay.

**Check your understanding.**
1. Why `UNION` and not `UNION ALL` in the recursive CTE?
2. Why recompute `pending_dependencies` on replay instead of incrementing?
3. When `transform_a` dies, what happens to `transform_b`, and why might you choose differently?

**Done when.** Failures cancel downstream jobs, and replays restore them correctly.

---
## Phase 15 — Observability: Prometheus metrics and Grafana

### Step 15.1 — Metrics in the server and worker

**Problem.** We can't operate (or benchmark) what we can't see. How many jobs per second? How long do jobs wait? Is the DLQ growing? Are workers saturated?

**Teach.**
- Metrics vs logs vs traces: what each is for.
- The Prometheus **pull** model: each process serves `/metrics`, and Prometheus scrapes it.
- **Metric types:** Counter (only goes up; query with `rate()`), Gauge (goes up and down), Histogram (bucketed observations; quantiles computed in PromQL, and **aggregatable across instances**, unlike Summaries).
- **Labels and cardinality:** every unique label combination is a separate time series. **Never** label by job ID, error message, or anything unbounded. `queue` and `type` are acceptable because they're validated and few (but note that `type` comes from users, so a real system would cap it).
- Naming conventions: a `jobqueue_` prefix, `_total` for counters, base units (`_seconds`).
- **RED** (Rate, Errors, Duration) for request-driven services, **USE** (Utilization, Saturation, Errors) for resources, and queue-specific signals: depth, oldest-job age, wait time, throughput, failure rate, DLQ size.
- Measuring with one clock: **wait time** = `updated_at − run_at` at claim (both set by Postgres; the claim sets `updated_at = now()`). **End-to-end time** = `completed_at − run_at`.
- **gRPC interceptors** are middleware: one function wraps every RPC, which is perfect for RPC metrics.
- Expensive gauges (queue depth needs `count(*)`) are refreshed by a periodic loop, not on every scrape.
- Use a dedicated `prometheus.Registry` instead of the global one, so tests can create isolated metrics.
- Exposing `/debug/pprof` on the metrics port for Phase 16, and **never on a public port**.

**Build.**
1. `go get github.com/prometheus/client_golang`.
2. `internal/metrics/server.go`: `type ServerMetrics struct {...}` and `func NewServerMetrics(reg prometheus.Registerer) *ServerMetrics` with exactly these metrics:

   | Name | Type | Labels | Meaning |
   |---|---|---|---|
   | `jobqueue_jobs_enqueued_total` | counter | `queue`, `type` | jobs created (not idempotent repeats) |
   | `jobqueue_jobs_claimed_total` | counter | `queue` | jobs handed to workers |
   | `jobqueue_jobs_finished_total` | counter | `queue`, `type`, `outcome` | outcome: `succeeded`, `retried`, `dead`, `released` |
   | `jobqueue_job_wait_seconds` | histogram | `queue` | `run_at` → claimed |
   | `jobqueue_job_e2e_seconds` | histogram | `queue` | `run_at` → succeeded |
   | `jobqueue_leases_reaped_total` | counter | `outcome` | `pending` or `dead` |
   | `jobqueue_queue_jobs` | gauge | `queue`, `state` | current row counts |
   | `jobqueue_oldest_pending_job_age_seconds` | gauge | `queue` | age of the oldest *due* pending job |
   | `jobqueue_cron_fired_total` | counter | `schedule` | cron fires |
   | `jobqueue_is_leader` | gauge | — | 1 while this replica leads |
   | `jobqueue_workflow_jobs_unblocked_total` | counter | — | |
   | `jobqueue_workflow_jobs_cancelled_total` | counter | — | |
   | `jobqueue_grpc_requests_total` | counter | `method`, `code` | |
   | `jobqueue_grpc_request_duration_seconds` | histogram | `method` | |

   Buckets: `prometheus.ExponentialBuckets(0.001, 2, 20)` for wait/e2e (1ms … ~9 min), `prometheus.ExponentialBuckets(0.0005, 2, 16)` for RPC durations.
3. `internal/metrics/worker.go`: `NewWorkerMetrics(reg)` with:

   | Name | Type | Labels |
   |---|---|---|
   | `jobqueue_worker_jobs_inflight` | gauge | — |
   | `jobqueue_worker_concurrency` | gauge | — (set once to the configured capacity) |
   | `jobqueue_worker_job_duration_seconds` | histogram | `type`, `outcome` (`succeeded`, `failed`, `released`, `lease_lost`) |
   | `jobqueue_worker_fetches_total` | counter | `result` (`jobs`, `empty`, `error`) |
   | `jobqueue_worker_heartbeats_total` | counter | `result` (`ok`, `lost`, `error`) |

4. `internal/metrics/http.go`: `func Serve(ctx context.Context, addr string, reg *prometheus.Registry, logger *slog.Logger) error`, which runs an `http.Server` with `/metrics` (`promhttp.HandlerFor(reg, promhttp.HandlerOpts{})`) and the pprof handlers registered explicitly (`/debug/pprof/`, `/debug/pprof/cmdline`, `/debug/pprof/profile`, `/debug/pprof/symbol`, `/debug/pprof/trace`). Shut it down gracefully when `ctx` is done. Register `collectors.NewGoCollector()` and `collectors.NewProcessCollector(collectors.ProcessCollectorOpts{})`.
5. Instrument: the service (enqueue, fetch/claim, complete, fail, release, reap, workflow unblock/cancel), the cron scheduler, the leader elector (pass in a callback or the gauge), and the worker. Pass metrics in through constructors (`queue.Options.Metrics`, `worker.Config` or a constructor argument). Allow `nil` metrics in tests that don't care: wrap the calls in small helper methods that no-op on `nil`.
6. `internal/api/interceptor.go`: `func MetricsInterceptor(m *metrics.ServerMetrics) grpc.UnaryServerInterceptor` recording `info.FullMethod`, `status.Code(err).String()` and the duration. Install it with `grpc.ChainUnaryInterceptor` in `NewGRPCServer`, whose signature becomes `NewGRPCServer(svc *queue.Service, m *metrics.ServerMetrics, logger *slog.Logger, opts ...grpc.ServerOption)` (`m` may be nil in tests). Update `cmd/server` and `testutil.NewTestServer`.
7. Stats loop: `store.QueueStats(ctx)` running `SELECT queue, state, count(*) FROM jobs GROUP BY queue, state` and `SELECT queue, extract(epoch FROM now() - min(run_at)) FROM jobs WHERE state = 'pending' AND run_at <= now() GROUP BY queue`. `Service.RunStatsLoop(ctx, 15*time.Second)` calls `GaugeVec.Reset()` before setting values, so vanished combinations disappear. Every replica runs it; dashboards use `max by (...)`.
8. Wire `metrics.Serve` into both binaries' errgroups / run functions with `JQ_METRICS_ADDR` (server `:2112`, worker `:2113`).
9. Tests using `github.com/prometheus/client_golang/prometheus/testutil`: `testutil.ToFloat64` on counters after an enqueue, complete, and fail.

**Verify.** `make test`; with the server and a worker running: `curl -s localhost:2112/metrics | grep ^jobqueue_` and `curl -s localhost:2113/metrics | grep ^jobqueue_worker`.

**Walkthrough focus.** Pick three metrics and trace where each is incremented. Why there's no `job_id` label. How the interceptor wraps every RPC.

**Check your understanding.**
1. Why a histogram rather than a summary for wait time?
2. What would happen to Prometheus if we labelled by `job_id`?
3. Which single metric would you look at first to know whether the queue is keeping up?

**Done when.** Both binaries expose the metrics above and the tests pass.

---

### Step 15.2 — Prometheus, Grafana and the dashboard

**Problem.** Raw `/metrics` text isn't a dashboard. We want graphs, quantiles, and alert rules, all versioned in git.

**Teach.**
- Prometheus config: `scrape_configs`, `static_configs`, `scrape_interval`. Containers reach processes on the host through `host.docker.internal` (on Linux this needs `extra_hosts: host-gateway`).
- PromQL essentials: `rate(x_total[1m])`, `sum by (label) (...)`, `histogram_quantile(0.95, sum by (le) (rate(x_bucket[1m])))`, `increase(...)`.
- **Dashboards as code:** Grafana provisioning loads data sources and dashboards from files at startup.
- Alerting concepts and three useful starter alerts (rules file only; Alertmanager is out of scope).

**Build.**
1. `deploy/prometheus/prometheus.yml`:
   ```yaml
   global:
     scrape_interval: 5s
   rule_files:
     - /etc/prometheus/alerts.yml
   scrape_configs:
     - job_name: jobqueue-server
       static_configs:
         - targets: ["host.docker.internal:2112", "host.docker.internal:2122"]
     - job_name: jobqueue-worker
       static_configs:
         - targets:
             - "host.docker.internal:2113"
             - "host.docker.internal:2114"
             - "host.docker.internal:2115"
             - "host.docker.internal:2116"
   ```
   (Targets that aren't running just show as DOWN; that's fine.)
2. `deploy/prometheus/alerts.yml` with three rules: **DLQ growing** (`increase(jobqueue_jobs_finished_total{outcome="dead"}[10m]) > 0`), **queue falling behind** (`max by (queue) (jobqueue_oldest_pending_job_age_seconds) > 300` for 5m), **no workers fetching** (`sum(rate(jobqueue_worker_fetches_total[5m])) == 0` for 5m).
3. Add to `docker-compose.yml`:
   ```yaml
     prometheus:
       image: prom/prometheus
       ports:
         - "9090:9090"
       volumes:
         - ./deploy/prometheus:/etc/prometheus:ro
       extra_hosts:
         - "host.docker.internal:host-gateway"

     grafana:
       image: grafana/grafana
       ports:
         - "3000:3000"
       environment:
         GF_AUTH_ANONYMOUS_ENABLED: "true"     # local development only
         GF_AUTH_ANONYMOUS_ORG_ROLE: "Admin"   # local development only
       volumes:
         - ./deploy/grafana/provisioning:/etc/grafana/provisioning:ro
         - ./deploy/grafana/dashboards:/var/lib/grafana/dashboards:ro
       depends_on:
         - prometheus
   ```
4. `deploy/grafana/provisioning/datasources/prometheus.yml`:
   ```yaml
   apiVersion: 1
   datasources:
     - name: Prometheus
       uid: prometheus
       type: prometheus
       access: proxy
       url: http://prometheus:9090
       isDefault: true
   ```
   `deploy/grafana/provisioning/dashboards/dashboards.yml`:
   ```yaml
   apiVersion: 1
   providers:
     - name: jobqueue
       folder: jobqueue
       type: file
       options:
         path: /var/lib/grafana/dashboards
   ```
5. `deploy/grafana/dashboards/jobqueue.json`: a dashboard with `"uid": "jobqueue"`, `"title": "Job Queue"`, and `timeseries` panels, each with `"datasource": {"type": "prometheus", "uid": "prometheus"}` and a `gridPos`. Keep the JSON minimal and validate it with `jq . deploy/grafana/dashboards/jobqueue.json`. Panels (title → query):
   - **Enqueued/s by queue** → `sum by (queue) (rate(jobqueue_jobs_enqueued_total[1m]))`
   - **Finished/s by outcome** → `sum by (outcome) (rate(jobqueue_jobs_finished_total[1m]))`
   - **Wait time p50/p95/p99** → `histogram_quantile(0.95, sum by (le) (rate(jobqueue_job_wait_seconds_bucket[1m])))` (one target per quantile)
   - **End-to-end p95** → the same with `jobqueue_job_e2e_seconds_bucket`
   - **Handler duration p95 by type** → `histogram_quantile(0.95, sum by (le, type) (rate(jobqueue_worker_job_duration_seconds_bucket[1m])))`
   - **Queue depth by state** → `max by (queue, state) (jobqueue_queue_jobs)`
   - **Oldest pending job age** → `max by (queue) (jobqueue_oldest_pending_job_age_seconds)`
   - **Dead jobs** → `max by (queue) (jobqueue_queue_jobs{state="dead"})`
   - **Leases reaped/s** → `sum by (outcome) (rate(jobqueue_leases_reaped_total[5m]))`
   - **Worker utilization** → `sum(jobqueue_worker_jobs_inflight) / sum(jobqueue_worker_concurrency)`
   - **Fetch results/s** → `sum by (result) (rate(jobqueue_worker_fetches_total[1m]))`
   - **Heartbeats lost/s** → `sum(rate(jobqueue_worker_heartbeats_total{result="lost"}[5m]))`
   - **RPC rate and errors** → `sum by (method, code) (rate(jobqueue_grpc_requests_total[1m]))`
   - **RPC p95 by method** → `histogram_quantile(0.95, sum by (le, method) (rate(jobqueue_grpc_request_duration_seconds_bucket[1m])))`
   - **Leader** → `max by (instance) (jobqueue_is_leader)`
6. `jqctl enqueue --count N` (enqueue N copies quickly, reusing one connection) for generating demo load.

**Verify.**
- `make up`; open `http://localhost:9090/targets` (server and worker UP); open `http://localhost:3000` → the "Job Queue" dashboard.
- Generate load: run a worker, then in parallel `./bin/jqctl enqueue --count 2000 --type sleep --payload '{"ms":50}'`, `--count 200 --type flaky --payload '{"fail_rate":0.5}'`, `--count 20 --type always_fail --max-attempts 2`. Watch throughput, wait times, retries, and dead jobs appear.
- Kill a worker with `kill -9` during load → watch "Leases reaped/s" spike.
- If Grafana shows nothing: `docker compose logs grafana` and check the provisioning paths.

**Check your understanding.**
1. Write the PromQL for p99 wait time per queue.
2. Which alert would have caught the `always_fail` jobs?
3. Why does the depth panel use `max by` instead of `sum by`?

**Done when.** The dashboard shows live data during a load run.

---

## Phase 16 — Benchmarking and profiling

### Step 16.1 — Bench tool

**Problem.** The resume bullet needs honest numbers: jobs/sec across N workers, plus latency. We need a repeatable way to produce them.

**Teach.**
- What to measure: **throughput** (completed jobs/sec) and **latency distribution** (p50/p95/p99 wait and end-to-end). Averages hide the tail.
- **Saturation** runs (enqueue as fast as possible: finds max throughput) vs **fixed-rate** runs (enqueue at R/sec: shows latency under realistic load). Coordinated omission in one paragraph: a load generator that waits for slow responses under-reports latency.
- **Little's Law:** `L = λ × W` (items in the system = throughput × time in the system). Use it to sanity-check results and size concurrency.
- Benchmark hygiene: warm up, run several times and take the median, change one variable at a time, and record the environment (CPU, RAM, Docker resources, versions).
- Measure latency **from the database** (`completed_at − created_at`): one clock and exact.

**Build.**
1. `cmd/bench/main.go` with two modes. Both use queue `bench` and delete old bench rows first (`DELETE FROM jobs WHERE queue = 'bench'`) through a direct DB connection (`JQ_DATABASE_URL`).
   - `bench enqueue --jobs 100000 --producers 16`: measures enqueue throughput and client-side RPC latency p50/p95/p99 (collect durations per producer, merge, sort, pick percentiles).
   - `bench e2e --jobs 50000 --producers 16 --type noop|sleep --sleep-ms 10 [--rate 0] [--timeout 10m]`: workers must already be running on queue `bench`. Enqueue (as fast as possible if `--rate 0`, otherwise paced with a `time.Ticker` shared by producers), then poll until every bench job is `succeeded` (or time out), then compute from the database:
     ```sql
     SELECT count(*) AS jobs,
            extract(epoch FROM max(completed_at) - min(created_at)) AS wall_seconds,
            percentile_cont(ARRAY[0.5, 0.95, 0.99])
                WITHIN GROUP (ORDER BY extract(epoch FROM completed_at - created_at)) AS e2e
     FROM jobs
     WHERE queue = 'bench' AND state = 'succeeded';
     ```
     Throughput = jobs / wall_seconds. Print a human summary **and** a Markdown table row for `docs/BENCHMARKS.md`.
2. `scripts/bench_matrix.sh`: for each combination of W ∈ {1, 2, 4} worker processes, C ∈ {8, 32} concurrency, and job type ∈ {noop, sleep 10ms}: start W `bin/worker` processes (`JQ_WORKER_QUEUES=bench`, `JQ_WORKER_CONCURRENCY=C`, metrics ports 2113+i), run `bin/bench e2e` 3 times, keep the median, then kill the workers. Use `set -euo pipefail` and a `trap` that kills background processes on exit.
3. `docs/BENCHMARKS.md` with an "Environment" section (fill in CPU from `sysctl -n machdep.cpu.brand_string`, core count, RAM, Docker's CPU/memory allocation, Go and Postgres versions) and a results table.

**Verify.** A small smoke run: `bin/bench e2e --jobs 2000 --type noop` with one worker.

**Check your understanding.**
1. Using Little's Law: 4 workers × 32 concurrency running 10ms jobs. What's the theoretical maximum throughput? (≈ 12,800 jobs/s.) What would stop us from reaching it?
2. Why is p99 more useful than the average?

**Done when.** The bench tool produces reproducible numbers and a table row.

---

### Step 16.2 — Baseline runs, profiling and tuning

**Problem.** Get real baseline numbers, find the bottleneck, and try *one* improvement at a time.

**Teach.**
- **Finding the bottleneck:** is it database CPU, connection-pool waits, lock contention, network round trips, or idle polling?
- **pprof:** `go tool pprof -http=: 'http://localhost:2112/debug/pprof/profile?seconds=20'` (CPU flame graph); heap profiles; goroutine dumps (`/debug/pprof/goroutine?debug=2`) for spotting leaks or stuck goroutines.
- **`pg_stat_statements`:** `CREATE EXTENSION IF NOT EXISTS pg_stat_statements;` then the top queries by `total_exec_time`.
- Tuning knobs: server pool `MaxConns`, `MaxBatch` per fetch, worker concurrency, poll intervals, heartbeat frequency. Explain `synchronous_commit = off` (faster commits, but the last few transactions can be lost on a crash) and **don't** enable it by default.
- Diminishing returns and Amdahl's law: Postgres is the shared resource every worker contends for.

**Build / do.**
1. Run `scripts/bench_matrix.sh` in Postgres mode and record the results in `docs/BENCHMARKS.md`.
2. Profile one saturated run: a server CPU profile + the `pg_stat_statements` top 5 + `docker stats` during the run. The AI explains what dominates.
3. Try **one** change at a time and record before/after, e.g. `MaxBatch` 1 vs 10 vs 50, or server pool `MaxConns` 10 vs 50. Discuss the batch-size trade-off (throughput vs fairness between workers).
4. Optional: export `pgxpool` stats as gauges (`jobqueue_db_pool_acquired_conns`, `jobqueue_db_pool_empty_acquire_total`).
5. Draft the resume numbers ("X jobs/sec across Y workers on <hardware>"). Keep them honest and mention the hardware.

**Verify.** The `docs/BENCHMARKS.md` tables are filled in, and the profile findings are written up.

**Check your understanding.**
1. What did the profile say the bottleneck was, and what's the evidence?
2. Why change only one variable per run?

**Done when.** Baseline numbers and a bottleneck analysis are documented.

---

## Phase 17 — Redis fast-scheduling layer

### Step 17.1 — Design: Redis as a derived index, and the dual-write problem

**Problem.** Every fetch scans a Postgres index, idle workers poll, and future-dated jobs slow down claims (12.3). Redis can answer "what's next?" from memory. But now two systems hold job information, and they can disagree.

This step is **design and teaching only**. The AI writes the design into `docs/ARCHITECTURE.md` ("Redis scheduling layer").

**Teach.**
- Why Redis: in-memory sorted sets with O(log n) operations, atomic pops, and Lua scripts for atomic multi-step operations. Postgres keeps durability and every state transition (Law 1).
- **Sorted sets:** `ZADD`, `ZPOPMIN`, `ZRANGEBYSCORE`, `ZREM`, `ZSCORE`.
- **Key layout:**
  - `jq:ready:<queue>`: ZSET of due jobs. Member `"<priority>:<id>"` (e.g. `"9:12345"`), score `(9 − priority) × 10^13 + run_at_ms`. `ZPOPMIN` gives the highest priority first, then the earliest `run_at`. (Scores stay below 2^53, so doubles represent them exactly.)
  - `jq:delayed:<queue>`: ZSET of future jobs. Same member format; score `run_at_ms`.
  - `jq:queues`: SET of queue names seen.
- **The dual-write problem:** we write Postgres, then Redis. A crash or error in between leaves them inconsistent. Walk through all four cases and why each is safe:
  1. Postgres committed, Redis write failed → job missing from Redis → the **reconciler** re-adds it (delay, not loss).
  2. Redis holds an ID whose Postgres row isn't claimable any more → the fenced Postgres claim matches nothing → **re-check the row and drop or re-add it**.
  3. `ZPOPMIN` succeeded, then the server crashed before the Postgres claim → job pending in Postgres but gone from Redis → the reconciler re-adds it.
  4. Redis wiped completely → the reconciler rebuilds everything.
  **Rule: Redis can only cause *delay*, never *loss* or *double processing*, because Postgres's compare-and-set is the gate.** This is exactly the gap the transactional outbox pattern (a stretch goal) closes more elegantly.
- **Every path that makes a job `pending` must notify the index:** Enqueue (`pending`, not `blocked`), workflow submission roots, FailJob (retry), the reaper, ReleaseJob, ReplayJob/ReplayDeadJobs, unblocked workflow children, and cron fires. **Notify after the transaction commits**, never inside it.
- **Promotion:** a leader-only loop moves due members from `delayed` to `ready` with a Lua script (atomic).
- **Reconciliation (anti-entropy):** a leader-only loop pages through pending jobs in Postgres and re-adds any that Redis doesn't know about.
- **Graceful degradation:** if Redis errors during a fetch, fall back to the Postgres `SKIP LOCKED` claim for that call. Both paths are correct at the same time (Law 1).
- Optional long-polling (`BZPOPMIN` with a timeout) removes idle polling; blocking commands hold a Redis connection each, so pool sizing matters.

**Verify.** The learner explains the four failure cases back to the AI.

**Check your understanding.**
1. Why can Redis never cause a job to run twice in this design?
2. Why notify Redis only *after* the Postgres transaction commits?
3. What's the worst-case extra delay if a Redis write fails?

**Done when.** The design is written down and the learner can defend it.

---

### Step 17.2 — Implement the Redis index: notify, promote, fetch

**Build.**
1. `go get github.com/redis/go-redis/v9`. Config: `JQ_REDIS_ADDR` (`localhost:6380`), `JQ_SCHEDULER_MODE` (`postgres` | `redis`).
2. `internal/redisq/index.go`:
   ```go
   type Index struct {
       rdb     *redis.Client
       prefix  string        // "jq" in production; unique per test
       promote *redis.Script
       logger  *slog.Logger
   }
   func New(rdb *redis.Client, prefix string, logger *slog.Logger) *Index
   // Member zero-pads the id so that, when scores tie, Redis's lexicographic member order equals id order (matching Postgres).
   func Member(priority int, id int64) string              // fmt.Sprintf("%d:%019d", priority, id) → "9:0000000000000012345"
   func ParseMember(m string) (priority int, id int64, err error)
   func ReadyScore(priority int, runAt time.Time) float64  // float64(9-priority)*1e13 + float64(runAt.UnixMilli())
   // Add decides ready vs delayed with the job's own DB timestamps, never a Go clock (Law 2):
   // ready if !j.RunAt.After(j.UpdatedAt) (every pending transition sets updated_at = now() in Postgres), else delayed.
   func (ix *Index) Add(ctx context.Context, jobs []store.Job) error // one pipeline: SADD queues; ZADD ready or delayed
   func (ix *Index) Pop(ctx context.Context, queue string, n int) ([]int64, error)          // ZPOPMIN ready n
   func (ix *Index) Promote(ctx context.Context, queue string, now time.Time, limit int) (int, error)
   func (ix *Index) Queues(ctx context.Context) ([]string, error)                           // SMEMBERS
   func (ix *Index) Missing(ctx context.Context, jobs []store.Job) ([]store.Job, error)     // pipelined ZSCORE on both sets
   ```
   The promote script (`KEYS[1]` = delayed key, `KEYS[2]` = ready key, `ARGV[1]` = now in ms, `ARGV[2]` = limit):
   ```lua
   local items = redis.call('ZRANGEBYSCORE', KEYS[1], '-inf', ARGV[1], 'WITHSCORES', 'LIMIT', 0, tonumber(ARGV[2]))
   local moved = 0
   for i = 1, #items, 2 do
     local member = items[i]
     local runAtMs = tonumber(items[i + 1])
     local sep = string.find(member, ':', 1, true)
     local priority = tonumber(string.sub(member, 1, sep - 1))
     redis.call('ZADD', KEYS[2], (9 - priority) * 10000000000000 + runAtMs, member)
     redis.call('ZREM', KEYS[1], member)
     moved = moved + 1
   end
   return moved
   ```
3. Store: `ClaimJobsByID(ctx, ids []int64, workerID string, lease time.Duration) ([]Job, error)`:
   ```sql
   UPDATE jobs
   SET state = 'running', attempts = attempts + 1, lease_owner = $2,
       lease_expires_at = now() + ($3::bigint * interval '1 millisecond'), updated_at = now()
   WHERE id = ANY($1::bigint[]) AND state = 'pending' AND run_at <= now()
   RETURNING <jobColumns("")>
   ```
   and `GetJobsByID(ctx, ids []int64) ([]Job, error)`.
4. Service changes:
   - `Options.Index *redisq.Index` (nil in Postgres mode).
   - Add three metrics to `ServerMetrics`: `jobqueue_redis_errors_total{op}` (counter; `op` ∈ `add`, `pop`, `promote`, `reconcile`), `jobqueue_redis_stale_total` (counter), `jobqueue_redis_reconciled_total` (counter, used in 17.3). Add a "Redis" row to the Grafana dashboard with their rates.
   - `func (s *Service) notify(ctx context.Context, jobs ...store.Job)`: no-op if `Index` is nil; keep only `state == pending`; `Add`; on error, log and increment `jobqueue_redis_errors_total{op="add"}` (the reconciler will repair it).
   - **Call `notify` after commit on every pending transition listed in 17.1.** Change `ReleaseJob`/`ReplayDeadJobs` call sites to keep the returned jobs. Add an `OnEnqueued func([]store.Job)` callback to the cron `Scheduler`, called after its transaction commits, and wire it to `svc.notify`. **The AI must grep for every place a job becomes `pending` and list them in the walkthrough.**
   - `FetchJobs` in Redis mode: for each queue in order, `ids := Index.Pop(queue, remaining)` → `ClaimJobsByID`. For IDs that weren't claimed: `GetJobsByID` → `notify` those still `pending` (e.g. popped slightly early), drop the rest (stale) and count `jobqueue_redis_stale_total`. On any Redis error: log, count `jobqueue_redis_errors_total{op="pop"}`, and **fall back to the Postgres claim** for that queue.
   - `RunPromoter(ctx, 200*time.Millisecond)`: once per tick get `now` from `store.Now(ctx)` (the database clock, Law 2), then for each queue in `Index.Queues`, `Promote(now, 1000)` until fewer than 1,000 move.
5. `cmd/server`: in Redis mode, create the client, `Ping` at startup (fail fast with a clear error), build the index, and run the promoter **under the leader** alongside cron (inside `lead`, run both with a small errgroup).
6. `internal/testutil/redis.go`: `NewTestRedis(t) (*redis.Client, string)` connects to `localhost:6380` and returns a unique prefix `jqtest:<random>`; cleanup deletes `<prefix>:*` keys with `SCAN` + `DEL`. Add `SchedulerMode` and the Redis client to `testutil.ServerOptions`.
7. Tests:
   - `Pop` order matches the Postgres claim order (priority, then `run_at`).
   - A delayed job stays in `delayed`; after `Promote` with a later `now`, it's in `ready`.
   - End to end in Redis mode: 200 jobs processed by a worker.
   - Stale entry: `ZADD` a member for a `succeeded` job → `FetchJobs` drops it with no error.
   - Fallback: a service whose Redis client points to a closed port still fetches via Postgres.

**Verify.** `make test`; `make run-server JQ_SCHEDULER_MODE=redis` plus a worker; enqueue jobs and inspect Redis with `make redis-cli` → `KEYS jq:*`, `ZRANGE jq:ready:default 0 -1 WITHSCORES`.

**Walkthrough focus.** The complete list of notify call sites. The fetch path including the stale and fallback branches. Why the promoter is leader-only.

**Check your understanding.**
1. What happens to a job popped from Redis whose Postgres row is already `succeeded`?
2. Why is `(9 − priority) × 10^13 + run_at_ms` a valid single score?
3. Why does the fallback path stay correct even though Redis still holds entries for jobs claimed through Postgres?

**Done when.** Redis mode passes all tests, and falling back to Postgres works.

---

### Step 17.3 — Reconciler and failure drills

**Build.**
1. Store: `PendingQueues(ctx) ([]string, error)` (`SELECT DISTINCT queue FROM jobs WHERE state = 'pending'`) and `ListPendingPage(ctx, queue string, afterID int64, limit int) ([]Job, error)` (`WHERE state = 'pending' AND queue = $1 AND id > $2 ORDER BY id LIMIT $3`).
2. `Service.ReconcileOnce(ctx) (repaired int, err error)`: for each pending queue, page through in batches of 1,000; `Index.Missing(batch)` → `Index.Add(missing)`; count `jobqueue_redis_reconciled_total`.
3. `Service.RunReconciler(ctx, 30*time.Second)`, leader-only, next to the promoter.
4. Tests: enqueue 100 jobs in Redis mode, delete all `<prefix>:*` keys, `ReconcileOnce` → 100 repaired, then a worker processes all of them. Also: `Pop` 5 IDs directly (simulating a crash between pop and claim) → `ReconcileOnce` restores exactly those 5.

**Verify (failure drills; record the observations in the learning log).**
- During a steady load run, `make redis-cli` → `FLUSHALL`. Watch fetches fall back or slow briefly, the reconciler repair the index, and every job complete.
- `docker compose stop redis` for 20 seconds during load, then `docker compose start redis`. Watch the error metrics, the fallback to Postgres, and the recovery.

**Check your understanding.**
1. What's the longest a job can be "invisible" to Redis in the worst case?
2. Why page by `id` instead of `OFFSET`?

**Done when.** The system survives Redis being wiped or down, with no lost jobs.

---

### Step 17.4 — Re-benchmark and compare

**Do.**
1. Run the same matrix as 16.2 in Redis mode.
2. Run the "many future jobs" scenario from 12.3 in both modes: seed 200,000 delayed jobs, then run `bench e2e`.
3. Compare in `docs/BENCHMARKS.md`: throughput, p95 wait, database CPU (`docker stats`), and claim query time (`pg_stat_statements`).
4. **Interpret honestly.** Redis mode may *not* win at small scale: there's an extra network hop, and Postgres still does the compare-and-set `UPDATE`. Expected wins: delayed-heavy workloads, less index scanning, lower database CPU per fetch. "Measure, don't assume" makes a strong interview story whichever way the numbers go.
5. Record a decision ("ADR: default scheduler mode") in `docs/ARCHITECTURE.md` with the evidence.

**Check your understanding.**
1. Which workload benefited most from Redis, and why?
2. If Redis mode were slower overall, would you still keep it? Under what conditions?

**Done when.** Both modes are benchmarked, and the decision is documented with evidence.

---

## Phase 18 — Prove it and tell the story

### Step 18.1 — Crash-recovery test

**Problem.** Prove the core guarantee end to end with real processes: when workers are killed with SIGKILL mid-job, no job is lost, no effect is duplicated, and recovery time is bounded.

**Teach.**
- Verification mindset: state the invariants, then try hard to break them.
- What we expect: every job delivered **≥ 1** time; every effect applied **exactly once** (thanks to the idempotent handler); recovery time ≈ lease duration + reaper interval.
- How this differs from the stretch-goal chaos suite: this is one scripted scenario, not a framework.
- Bash essentials used here: `set -euo pipefail`, background processes, `trap` for cleanup.

**Build.** `scripts/crash_test.sh` (parameters via env vars with defaults: `JOBS=5000`, `WORKERS=4`, `KILL_EVERY=3`, `LEASE=5s`):
1. Load env vars at the top of the script (`set -a; source .env; set +a`), since it isn't run through `make`. Build binaries (`make build`); clean up: `DELETE FROM jobs WHERE queue = 'crash'; DELETE FROM demo_effects;` (through `docker compose exec -T postgres psql -U jobqueue jobqueue -c "..."`).
2. Start the server in the background with `JQ_LEASE_DURATION=$LEASE JQ_REAPER_INTERVAL=1s`.
3. Start `WORKERS` restart loops, each `while true; do JQ_WORKER_QUEUES=crash JQ_WORKER_CONCURRENCY=16 JQ_METRICS_ADDR=:$((2113+i)) ./bin/worker; sleep 0.5; done &`.
4. Enqueue: `./bin/jqctl enqueue --count $JOBS --queue crash --type record_effect --payload '{"sleep_ms":200}'`.
5. A killer loop: every `KILL_EVERY` seconds, `kill -9` one random PID from `pgrep -f 'bin/worker'`, until no `crash` jobs remain unfinished.
6. Wait (timeout 10 minutes) until `SELECT count(*) FROM jobs WHERE queue = 'crash' AND state <> 'succeeded'` returns 0.
7. Assertions (print a summary table; exit non-zero if any fails):
   - **Lost jobs:** crash-queue jobs not `succeeded` = 0.
   - **Effects:** rows in `demo_effects` joined to crash jobs = `JOBS` (exactly one per job).
   - **Duplicate deliveries absorbed:** count of `demo_effects` rows with `deliveries > 1`. **Report it; don't fail on it**: it's usually > 0 with kills (that's the point) and 0 without.
   - **Re-deliveries:** count of crash jobs with `attempts > 1`; max `attempts`.
   - **Latency:** p50/p99 of `completed_at - created_at`, compared with a no-kill run (`KILL_EVERY=0` skips the killer).
8. `trap` on EXIT: kill the background loops, `pkill -f bin/worker`, kill the server.

**Verify.** Run it twice (with kills and without). Record the results in `docs/BENCHMARKS.md` under "Crash recovery".

**Check your understanding.**
1. Why is `deliveries > 1` expected, and why is it fine?
2. What bounds the extra latency for jobs whose worker was killed?
3. What result would you see if the handler weren't idempotent?

**Done when.** The script passes with workers being killed, and the results are documented.

---

### Step 18.2 — README, architecture doc, resume bullet, interview prep

**Problem.** A project only helps your career if you can explain it clearly and back it up with evidence.

**Build.**
1. `README.md`: a one-paragraph pitch; feature list; architecture diagram (reuse 1.2); quickstart (`make up`, `make migrate-up`, run the server and a worker, `jqctl` examples, open Grafana); design highlights (`SKIP LOCKED` claims, leases + fencing, backoff with full jitter, DLQ + replay, cron with advisory-lock leader election + idempotency keys, counter-based DAG unblocking, Redis as a derived index + reconciler); benchmark and crash-test results **with hardware details**; limitations and future work (exactly-once via transactional outbox, Kubernetes + HPA, a chaos suite, authentication for the admin API, multi-tenant fairness, retention and cleanup of finished jobs, partitioning the `jobs` table); links to `docs/`.
2. Finish `docs/ARCHITECTURE.md`: lifecycle, guarantees, components, data model, performance notes, and a list of decisions (ADRs): Postgres as source of truth; pull-based fetch; `attempts` as the fencing token; full jitter; DLQ as a state; advisory-lock leader + idempotency keys; counter-based DAG; Redis as a derived index.
3. The resume bullet with **real** numbers from `docs/BENCHMARKS.md`, in a short and a long version. Template:
   > Built a distributed job queue in Go with PostgreSQL-backed durability, gRPC workers, retries with exponential backoff and jitter, dead-letter queues, cron scheduling with leader election, DAG workflows, and Prometheus/Grafana observability; sustained **X jobs/sec across Y workers** (p99 end-to-end **Z ms**) and recovered from repeated SIGKILLed workers with zero lost jobs.
4. `docs/INTERVIEW_PREP.md`: about 25 questions. **The learner answers each first**; the AI then refines the answer and writes both down. Cover at least: why `SKIP LOCKED`; at-least-once vs exactly-once; what happens when a worker dies mid-job; choosing a lease duration; clock skew; fencing tokens; backoff and jitter; poison pills; error classification; leader-election pitfalls; the DAG lost-update problem; Redis/Postgres consistency; which metrics matter and why; how you benchmarked; what the bottleneck was; how you'd scale past one Postgres (partitioning by queue, sharding, or moving to Kafka); retention of finished jobs; what you'd do differently.
5. Final cleanup: `make lint`, `make test`, `buf lint`, remove dead code, make log messages consistent, and document every Makefile target in the README.

**Verify.** A fresh clone builds and runs by following only the README (the AI walks through it from scratch in a temporary directory).

**Check your understanding.** A **15-minute mock interview**: the AI plays the interviewer and asks follow-up questions; the learner answers; the AI gives specific feedback at the end.

**Done when.** The README, architecture doc, benchmarks, resume bullet and interview prep are complete, and the learner can explain any part of the system without notes.

---
## Appendix A: Glossary

| Term | Meaning in this project |
|---|---|
| **Job** | One unit of work: a `type` (which handler runs it) plus a JSON `payload`. |
| **Queue** | A named stream of jobs (`default`, `critical`, ...). Workers choose which queues to pull from. |
| **Claim** | Atomically moving a job from `pending` to `running` and giving one worker a lease on it. |
| **Lease** | Time-limited ownership of a running job (`lease_owner`, `lease_expires_at`). |
| **Heartbeat** | A worker renewing its lease while the job is still running. |
| **Reaper** | A background loop that returns jobs with expired leases to `pending` (or `dead`). |
| **Fencing token** | A number that increases with every new owner; writes carrying an old token are rejected. Here: `attempts`. |
| **Zombie worker** | A worker that lost its lease (e.g. after a long pause) but is still running the job. |
| **At-least-once** | Every job runs one or more times; never zero. |
| **Effectively-once** | At-least-once delivery + idempotent handlers = the effect happens once. |
| **Idempotent** | Doing it twice has the same effect as doing it once. |
| **Idempotency key** | A client-chosen unique key that makes a request safe to retry. |
| **Backoff** | Waiting longer between successive retries. |
| **Jitter** | Randomizing retry delays so clients don't retry in lockstep. |
| **Thundering herd** | Many clients retrying at the same moment and overloading a service. |
| **Poison message / pill** | A job that fails (or crashes workers) every time it runs. |
| **DLQ (dead-letter queue)** | Where jobs go after exhausting retries; here, `state = 'dead'`. |
| **Replay** | Moving a dead job back to `pending` after fixing the root cause. |
| **Backpressure** | The consumer controlling the rate of work (workers fetch only what they can run). |
| **Semaphore** | A counter limiting how many things run at once; here, a buffered channel. |
| **Graceful shutdown / drain** | Stop taking work, finish in-flight work, hand back the rest, then exit. |
| **Advisory lock** | A Postgres lock on an application-chosen number, used here for leader election. |
| **Leader election** | Choosing one replica to perform singleton work (cron, promoter, reconciler). |
| **DAG** | Directed acyclic graph: the shape of a workflow's dependencies. |
| **Topological sort** | An ordering of DAG nodes where each node comes after its dependencies. |
| **Lost update** | Two concurrent read-modify-write cycles where one overwrites the other. |
| **MVCC** | Postgres's multi-version concurrency control: updates create new row versions; vacuum removes dead ones. |
| **Partial index** | An index over only the rows matching a `WHERE` clause. |
| **Keyset pagination** | Paging with `WHERE id > last_seen ORDER BY id LIMIT n` instead of `OFFSET`. |
| **Cardinality** | The number of distinct label combinations (time series) a metric has. |
| **Little's Law** | Items in a system = arrival rate × time each spends in the system. |
| **Dual write** | Writing the same fact to two systems without a shared transaction. |
| **Reconciler / anti-entropy** | A loop that finds and repairs differences between a source of truth and a derived copy. |

---

## Appendix B: Troubleshooting

| Symptom | Likely cause and fix |
|---|---|
| `protoc-gen-go: program not found or is not executable` | `$(go env GOPATH)/bin` isn't on `PATH`. Add it to `~/.zshrc` and open a new terminal. |
| `Bind for 0.0.0.0:5433 failed: port is already allocated` | Something else uses the port. Stop it, or change the host port in `docker-compose.yml` and `.env`. |
| Tests fail with `connection refused` | Docker services aren't running: `make up`. |
| `there is no unique or exclusion constraint matching the ON CONFLICT specification` | The `ON CONFLICT` target is missing the partial index's `WHERE idempotency_key IS NOT NULL`. |
| `new row for relation "jobs" violates check constraint "jobs_lease_check"` | A transition forgot to set or clear `lease_owner` / `lease_expires_at`. The invariant just caught a real bug. |
| `... violates check constraint "jobs_completed_check"` | A transition into or out of a terminal state forgot `completed_at`. |
| `... violates check constraint "jobs_blocked_deps_check"` | A workflow transition left `pending_dependencies` inconsistent with `state` (e.g. cancelling without setting it to 0). |
| `conn busy` from pgx | A single `pgx.Conn` used from several goroutines at once. Use the pool, or one connection per goroutine. |
| `import cycle not allowed in test` | A test in `package store` imports `testutil`. Change it to `package store_test`. |
| `fatal error: all goroutines are asleep - deadlock!` | Every goroutine is blocked: typically an unbuffered send nobody receives, or a `WaitGroup` that's never `Done`. |
| `WARNING: DATA RACE` | Read both stack traces: one is the write, one the conflicting access. Protect the shared value with a mutex or atomic, or stop sharing it. |
| gRPC `Unavailable: connection refused` | The server isn't running, or `JQ_SERVER_ADDR` is wrong. |
| bufconn tests hang or fail to connect | The target must be `passthrough:///bufnet`, not `bufnet`. |
| `ERROR: deadlock detected` (SQLSTATE `40P01`) | Two transactions locked rows in different orders. Lock in a consistent order (`ORDER BY id ... FOR UPDATE`), and retry the transaction if it still happens. |
| Grafana dashboard empty | Check `http://localhost:9090/targets`; on Linux make sure `extra_hosts: host-gateway` is set; check `docker compose logs grafana`. |
| `buf breaking` fails | You changed or removed an existing field number or type. Add a new field instead. |

---

## Appendix C: Further reading (primary sources)

- PostgreSQL docs, `SELECT ... FOR UPDATE / SKIP LOCKED`: https://www.postgresql.org/docs/current/sql-select.html#SQL-FOR-UPDATE-SHARE
- PostgreSQL docs, explicit locking and advisory locks: https://www.postgresql.org/docs/current/explicit-locking.html
- PostgreSQL docs, `EXPLAIN`: https://www.postgresql.org/docs/current/using-explain.html
- Brandur Leach, "Postgres Job Queues & Failure By MVCC": https://brandur.org/postgres-queues
- Martin Kleppmann, "How to do distributed locking" (fencing tokens): https://martin.kleppmann.com/2016/02/08/how-to-do-distributed-locking.html
- Martin Kleppmann, *Designing Data-Intensive Applications* (O'Reilly), chapters on transactions, distributed-systems trouble, and stream processing.
- AWS Architecture Blog, "Exponential Backoff And Jitter": https://aws.amazon.com/blogs/architecture/exponential-backoff-and-jitter/
- Go blog, "Go Concurrency Patterns: Pipelines and cancellation": https://go.dev/blog/pipelines
- Go blog, "Go Concurrency Patterns: Context": https://go.dev/blog/context
- Go docs, Data Race Detector: https://go.dev/doc/articles/race_detector
- `errgroup` package docs: https://pkg.go.dev/golang.org/x/sync/errgroup
- gRPC Go quick start: https://grpc.io/docs/languages/go/quickstart/
- Protocol Buffers language guide (proto3): https://protobuf.dev/programming-guides/proto3/
- Buf docs: https://buf.build/docs/
- Prometheus, metric and label naming: https://prometheus.io/docs/practices/naming/
- Prometheus, histograms and summaries: https://prometheus.io/docs/practices/histograms/
- Google SRE book, "Monitoring Distributed Systems": https://sre.google/sre-book/monitoring-distributed-systems/
- Redis sorted sets: https://redis.io/docs/latest/develop/data-types/sorted-sets/
- Stripe API docs, idempotent requests: https://docs.stripe.com/api/idempotent_requests

---

## Appendix D: Plan changes

The AI records every approved deviation from this plan here.

| Date | Step | Change | Reason |
|---|---|---|---|
| | | | |
