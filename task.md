# Korion — Task Tracker

Mirrors the phased plan in [`docs/PLAN.md`](docs/PLAN.md) (see `decisions.md`
for the architectural decisions behind each phase). Update status here as
work progresses — this file is the durable, repo-committed source of truth
for progress across sessions (the in-session Task tool is ephemeral).

**Starting a new session?** Point Claude at these three files first:
`docs/PLAN.md` (full phase detail), `task.md` (this file, current status),
`decisions.md` (constraints already decided — don't re-litigate). E.g.:
"Continue Korion per docs/PLAN.md — implement Phase N."

Status legend: `[ ]` pending · `[~]` in progress · `[x]` done

## v0.1 — Topology engine (no ARIA)

- [x] Phase 0 — Repo scaffold & governance hygiene
- [ ] Phase 1 — API types + CRD generation (PlatformMap, PlatformAgent)
- [ ] Phase 2 — K8s discovery vertical slice + read API
- [ ] Phase 3 — Graph builder hardening + frozen contract
- [ ] Phase 4 — Frontend skeleton against mock data
- [ ] Phase 5 — Wire canvas to real controller
- [ ] Phase 6 — Remaining five discovery engines (argocd, istio, kyverno, github, prometheus)
- [ ] Phase 7 — Helm chart
- [ ] Phase 8 — SuperHeros v0.1 acceptance validation

## v0.2/v0.3 — ARIA

- [ ] Phase 9 — ARIA foundation: FastAPI service, models, context builder (no LLM yet)
- [ ] Phase 10 — ARIA alert enrichment
- [ ] Phase 11 — ARIA health advisor + canary decision
- [ ] Phase 12 — Helm integration for ARIA + dashboard wiring

## Explicitly out of scope

- Kforge (separate IDP project)
- Cloud-provider-specific detection (EKS/GKE/AKS) until actually needed
- Playwright E2E / Storybook
- "Runtime Service Map" traffic panel (seen in one mockup, not in CLAUDE.md's layout)
