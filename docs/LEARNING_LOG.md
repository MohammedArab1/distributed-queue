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
