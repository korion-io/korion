# Phase 0 Summary — Repo scaffold & governance hygiene

Status: complete. See `docs/PLAN.md` for the full phase spec and
`decisions.md` for the environment/tooling issues found and fixed along the
way.

## What was built

- `go.mod` — module `github.com/korion-io/korion`.
- `.gitignore` — Go/Node/Python/editor/secrets patterns.
- `LICENSE` — full Apache License 2.0 text.
- `README.md` — thin problem-statement overview linking out to `CLAUDE.md`,
  `docs/PLAN.md`, `task.md`, `decisions.md`.
- `GOVERNANCE.md`, `SECURITY.md`, `CONTRIBUTING.md` — CNCF-Sandbox-minimum
  governance docs.
- `Makefile` — every target named in `CLAUDE.md`'s "Build and Run Commands"
  section (`generate`, `manifests`, `test`, `build`, `run`, `docker-build`,
  `docker-push`, `install`, `uninstall`, `deploy`, `helm-sync-crds`), plus
  `license-check`/`license-fix` and `controller-gen`/`addlicense` bootstrap
  targets.
- `hack/boilerplate.go.txt` — Apache 2.0 header template consumed by
  `controller-gen` for generated Go files.
- `.github/workflows/ci.yml` — four jobs (`license-check`, `go`, `frontend`,
  `aria`), each degrading gracefully (skip, not fail) until its layer's files
  exist.
- `task.md`, `decisions.md`, `docs/PLAN.md` — durable, repo-committed
  tracking (phase checklist, decision log, full copy of the approved plan)
  so progress and rationale survive across sessions instead of living only
  in local plan-mode state.

## Verification performed

- `mingw32-make license-check` — passed (no `.go`/`.py`/`.ts` files tracked
  yet, so the check is a no-op by design at this phase).
- `git init`, staged, committed (`e420c34`), and pushed to a newly created
  `korion-io/korion` GitHub repo (public), with `gc-ghub` confirmed as an
  active admin member of the `korion-io` org before push.

## Environment/tooling issues found and fixed

Full detail in `decisions.md`; summarized here:

1. **No bare `make` on this Windows host's Git Bash PATH** — `mingw32-make`
   works as a substitute (WSL2 `Ubuntu-22.04` also available as a fallback).
2. **Project path contains spaces and an em-dash**
   (`Project- Korion — K8s-Native Platform Topology Engine`) — broke
   unquoted `test`/shell invocations in the Makefile (`test -s $(ADDLICENSE)`
   etc. expanded to multiple words). Fixed by quoting every path variable
   used in a Makefile recipe.
3. **`github.com/google/addlicense`** was chosen and installed as the
   license-header enforcement tool (over `apache/skywalking-eyes`) — no
   issues encountered.

## Not done in this phase (by design)

- No Go/Python/TypeScript source files — this phase is scaffold-only.
- CI's `frontend`/`aria` jobs have nothing to build yet; they no-op until
  Phase 4 (`ui/`) and Phase 9 (`aria/`) respectively.
- No CRDs, no controller, no discovery engines — all later phases.
