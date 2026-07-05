# Korion — Architectural Decisions Log

Append-only log of decisions made during planning/implementation, newest last.
For the full phased plan these decisions support, see the plan history for
"Korion — Phased Implementation Plan (Zero → v0.1)".

## 2026-07-05 — Project scope and independence

- Korion is built as an independent project — no dependency on kagent (generic
  agent framework). Reason: kagent is a framework for others to build products;
  Korion is the product. A kagent dependency would make Korion a plugin, which
  weakens the CNCF Sandbox submission story and constrains architecture to
  kagent's boundaries. See `CLAUDE.md` rule #2.
- Kforge (a separate IDP project, CRD-based developer self-service) was
  discussed in the same design session but is explicitly **out of scope** for
  Korion. It may integrate with Korion later (Korion would auto-discover
  services Kforge provisions), but no Kforge code lives in this repo.
- ARIA (the PlatformAgent AI layer) **is in scope** for Korion, planned in full
  as Phases 9-12, not deferred as an afterthought — it's core to the "system
  specific AI, not generic K8s advice" differentiator.

## 2026-07-05 — Go module path

- Module: `github.com/korion-io/korion`, matching the confirmed `korion-io`
  GitHub org (kept separate from the user's personal `gc-ghub` account per
  CNCF governance requirements — a project can't be CNCF-submitted under a
  personal account).

## 2026-07-05 — Frontend-to-cluster transport

- **Decision:** the React frontend does NOT talk to the K8s API server
  directly. The Go manager binary exposes a small read-only HTTP endpoint
  (`GET /api/v1/platformmaps/{namespace}/{name}`) backed by its in-memory
  controller-runtime cache, with CORS for local dev / same-origin in
  production behind the Helm-installed Service.
- **Why:** a browser cannot safely hold K8s bearer-token credentials. Routing
  through a thin API in the controller means "the frontend reads from
  PlatformMap status" (CLAUDE.md rule #5) stays literally true without ever
  exposing cluster credentials to client-side JS. The alternative considered
  (direct K8s API server access via kubectl-proxy/oauth2-proxy) was rejected
  as pushing RBAC/auth complexity into the browser layer for no real benefit.
- **Confirmed with user:** yes, via AskUserQuestion during planning.

## 2026-07-05 — Kyverno discovery source

- **Decision:** `internal/discovery/kyverno.go` reads the vendor-neutral
  `wgpolicyk8s.io/v1alpha2 PolicyReport`/`ClusterPolicyReport` CRDs (which
  Kyverno populates), not a Kyverno-specific CRD.
- **Why:** keeps the discovery engine policy-engine-neutral — consistent with
  `CLAUDE.md` rule #9 (multi-cloud/vendor-neutral from v0.1) and would let
  Korion support other policy engines (e.g. OPA/Gatekeeper) later without
  changing this engine.
- **Confirmed with user:** yes, via AskUserQuestion during planning.

## 2026-07-05 — ArgoCD/Istio discovery via dynamic client, not vendored SDKs

- **Decision:** `argocd.go` and `istio.go` use `k8s.io/client-go/dynamic`
  against the known GVRs (`argoproj.io/v1alpha1` Applications,
  `networking.istio.io/v1beta1` VirtualService/DestinationRule) via
  `unstructured.NestedString/NestedMap`, rather than importing the full
  `argo-cd` or Istio Go client modules.
- **Why:** avoids heavy transitive dependencies and coupling Korion's release
  cadence to ArgoCD's/Istio's internal Go API stability — these tools are
  themselves just CRDs from Korion's point of view.

## 2026-07-05 — Loki discovery lives in ARIA, not the Go controller

- **Decision:** there is no `internal/discovery/loki.go`. Recent error log
  pattern discovery (`aria/collectors/loki_collector.py`) is ARIA's exclusive
  responsibility.
- **Why:** matches `CLAUDE.md`'s repo structure (Loki collector is listed only
  under `aria/collectors/`), and log freshness (last 30 min) is more relevant
  at ARIA's diagnosis-time than at the controller's topology-refresh cadence.

## 2026-07-05 — PlatformAgent CRD scaffolded early, reconciler wired late

- **Decision:** the `PlatformAgent` CRD schema (`api/v1alpha1/platformagent_types.go`)
  is scaffolded in Phase 1 alongside `PlatformMap`, but
  `internal/controller/platformagent_controller.go` stays unregistered/no-op
  until Phase 12.
- **Why:** the repo structure and sample YAMLs in `CLAUDE.md` reference the
  full `PlatformAgent` schema from the start; scaffolding it early avoids
  reshaping the API group later. The reconciler itself has nothing useful to
  do until ARIA (Phases 9-11) exists to be reconciled against.

## 2026-07-05 — Windows host has no bare `make`, use `mingw32-make`

- **Finding:** this dev machine's Git Bash has no `make` on PATH, but
  `mingw32-make` works. WSL2's `Ubuntu-22.04` distro also exists (currently
  stopped) as a fallback if `mingw32-make` proves insufficient for later
  phases (e.g. `envtest` setup in Phase 2).
- **Why it matters:** the project path itself contains spaces and an em-dash
  (`Project- Korion — K8s-Native Platform Topology Engine`), which broke
  unquoted `test`/shell invocations in the Makefile until path variables were
  quoted. Keep all Makefile recipe path variables quoted going forward.

## 2026-07-05 — Tracking artifacts

- **Decision:** `task.md` (phase/status tracker) and `decisions.md` (this
  file) live at the repo root as durable, git-committed tracking — separate
  from the ephemeral in-session Task tool, which doesn't survive across
  sessions.
