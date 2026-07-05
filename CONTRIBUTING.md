# Contributing to Korion

Thanks for your interest in contributing. Korion is early-stage and moving
fast — please read `CLAUDE.md` first for the full architecture and
non-negotiable design decisions before proposing changes that touch them.

## Development setup

See the "Build and Run Commands" section in `CLAUDE.md` for the full command
reference (`make generate`, `make test`, `make run`, frontend `npm run dev`,
ARIA `uvicorn main:app --reload`, etc.).

Prerequisites: Go 1.22+, Node 18+, Python 3.12+, Docker, `kind`, `helm`,
`kubectl`, `controller-gen`, `kubebuilder`.

## License headers

Every `.go`, `.py`, and `.ts`/`.tsx` file must carry an Apache 2.0 license
header. Run `make license-check` before submitting a PR; CI enforces this.

## Pull requests

- Keep PRs scoped to one logical change.
- Add or update tests for any behavior change (`internal/graph/builder_test.go`
  is a good example of the expected style for Go; Vitest for the frontend;
  `pytest` for ARIA).
- Reference the relevant phase from `task.md` in your PR description if
  applicable, and update `task.md`/`decisions.md` if the PR completes a phase
  or makes a new architectural call.

## Code of conduct

Be respectful and constructive. A formal CODE_OF_CONDUCT.md will be added
ahead of the CNCF Sandbox submission.
