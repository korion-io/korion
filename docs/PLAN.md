# Korion — Phased Implementation Plan (Zero → v0.1)

> This is the durable, repo-committed copy of the approved implementation
> plan. It was originally produced and approved via Claude Code plan mode; see
> `decisions.md` for the two architectural decisions confirmed during
> planning and `task.md` for live phase status. Keep this file in sync if the
> plan changes — it, not `~/.claude/plans/`, is the source of truth once
> committed, since plan-mode files are local to one machine/session and are
> not part of the repo.

## Context

Korion is a greenfield, K8s-native platform topology engine — it auto-discovers a
cluster's full DevOps stack (GitHub Actions → ArgoCD → K8s → Istio → Kyverno →
Prometheus) and renders it as a live, interactive React Flow diagram, replacing the
6+ separate dashboard tabs engineers currently juggle. The full architecture, CRDs,
tech stack, repo layout, and non-negotiable design rules are already finalized in
`CLAUDE.md` at the repo root (written after an extensive design discussion captured
in `ref-docs/Korion Project Discussion - Claude.pdf`). The `ref-docs/` mockups
(especially `file_000000006c407208be1de7ae77fa6da5.png`) show the exact dashboard
layout to build toward: left nav, main topology canvas, right `ServiceDetails`
panel, bottom-left `Deployment Timeline`, bottom-right `Policy & Security` panel.

The repo started with **only** `CLAUDE.md` and `ref-docs/` — no code existed. This
plan starts from true zero and takes the project to the v0.1 acceptance criteria
already defined in `CLAUDE.md`: applying `platformmap-superheros.yaml` to a Kind
cluster running the SuperHeros test platform must, within 60 seconds, show all 6
services on the canvas with live ArgoCD/GitHub/Istio/Kyverno state.

A separate project ("Kforge", an IDP) was discussed in the same design session but
is explicitly **out of scope** — this plan covers Korion only. **ARIA (the
PlatformAgent AI layer) is in scope and is planned in full below** (Phases 9-12) —
it is core to Korion's differentiation (system-specific AI, not generic K8s
advice), not an optional add-on. The sequencing still builds the topology engine
first (Phases 0-8, v0.1) because ARIA's context assembly depends on
`.status.topology` already existing and being reliable — but ARIA's own phases are
fully planned now, not deferred as an afterthought.

**Local environment verified at plan time:** Go 1.25.4, Docker 25.0.3, kubectl
1.34.1, Helm 3.12.1, kind 0.30.0, Node 24.11.1, Python 3.13.7, git 2.44.0 all
present. `controller-gen` and the `kubebuilder` CLI were not installed —
installing these is the first Phase 1 task. Note: bare `make` is not on PATH on
this Windows host; use `mingw32-make` (see `decisions.md`).

**Two architectural decisions confirmed with the user (full rationale in
`decisions.md`):**
1. The frontend will **not** talk to the K8s API server directly. The Go manager
   binary exposes a small read-only HTTP endpoint (e.g.
   `GET /api/v1/platformmaps/{namespace}/{name}`) backed by its in-memory
   controller-runtime cache, with CORS for local dev / same-origin in production
   behind the Helm-installed Service. No K8s credentials ever reach the browser.
2. `internal/discovery/kyverno.go` reads the vendor-neutral
   `wgpolicyk8s.io/v1alpha2 PolicyReport` CRD (which Kyverno populates) rather than
   a Kyverno-specific CRD — keeps discovery policy-engine-neutral, consistent with
   the multi-cloud/vendor-neutral principle already stated in `CLAUDE.md`.

## Guiding sequencing principle

Build one **vertical slice** through all four layers first — CRD → controller →
`.status.topology` → HTTP read API → React Flow canvas — using only the simplest
discovery source (plain K8s Deployments/Services, no ArgoCD/Istio/Kyverno/GitHub/
Prometheus required). This proves the whole architecture and freezes the JSON
contract between Go and TypeScript with the fewest moving parts. Only after that
slice is demoable against a bare Kind cluster do the remaining five discovery
engines get built in parallel, and the frontend gets polished to match the full
mockup.

---

## Phase 0 — Repo scaffold & governance hygiene ✅ complete

Empty-but-correct skeleton, no logic yet.

- `go.mod` (module `github.com/korion-io/korion`), `.gitignore`, `LICENSE`
  (Apache-2.0 full text).
- Thin `README.md` (problem statement + one mockup screenshot), `GOVERNANCE.md`,
  `SECURITY.md`, `CONTRIBUTING.md` — CNCF Sandbox expects these to exist even in
  minimal form.
- `Makefile` with the exact targets already named in `CLAUDE.md`'s "Build and Run
  Commands" section (`generate`, `manifests`, `test`, `build`, `docker-build`,
  `docker-push`, `install`, `run`, `deploy`, `uninstall`), most as no-ops initially.
- License-header enforcement via `google/addlicense` as a `make license-check`
  target + CI job — the mechanical guarantee that every `.go`/`.py`/`.ts` file
  carries the Apache 2.0 header, per `CLAUDE.md` rule #1.
- `.github/workflows/ci.yml` skeleton: Go build+vet job now, Node/Python jobs added
  as those layers come online.

**Verify:** `mingw32-make license-check` passes; CI runs green on a trivial commit.

---

## Phase 1 — API types + CRD generation

`PlatformMap` and `PlatformAgent` CRDs installable into a cluster, generated
deepcopy/manifests, no controller logic yet.

- Install `controller-gen` and `kubebuilder` CLI (currently missing locally).
- `kubebuilder init --domain korion.io --repo github.com/korion-io/korion`, then
  fold the generated scaffold into the exact directory names `CLAUDE.md` specifies
  (`api/v1alpha1/`, `internal/controller/`).
- `kubebuilder create api --group korion --version v1alpha1 --kind PlatformMap`
  and `--kind PlatformAgent`, producing:
  - `api/v1alpha1/platformmap_types.go` — spec (repository, namespace,
    autoDiscover, per-tool `enabled`/`url`/secretRef map, `refreshInterval` as
    `metav1.Duration`) and status (topology graph — shape frozen in Phase 3 —
    plus per-source `conditions []metav1.Condition` and `lastDiscoveryTime`).
  - `api/v1alpha1/platformagent_types.go` — full spec per the sample YAML in
    `CLAUDE.md` (platformMap ref, autonomyLevel, llmProvider, features,
    runbooks). Scaffolded now but **inert** — no controller registered against it
    until Phase 12, since ARIA itself doesn't run until then.
  - `api/v1alpha1/groupversion_info.go`.
- `make manifests` → `config/crd/bases/*.yaml`, `config/rbac/*.yaml` (start
  narrow — just what K8s-core discovery needs; expand per engine in Phase 6).
- `config/samples/platformmap-superheros.yaml` and `platformagent-superheros.yaml`
  transcribed from `CLAUDE.md`.

**Verify:** `make install` against a fresh `kind create cluster` succeeds;
`kubectl apply -f config/samples/platformmap-superheros.yaml` is accepted and
`kubectl get platformmap -o yaml` reflects it.

---

## Phase 2 — Vertical slice: K8s discovery → controller → status → read API

The single most important phase — proves the architecture end to end before any
breadth work.

- `internal/controller/platformmap_controller.go`: `Reconcile()` fetches the
  `PlatformMap`, runs discovery, writes `.status.topology` + `.status.conditions`,
  requeues at `spec.refreshInterval` — and reconciles immediately on create so the
  60-second acceptance criteria isn't waiting a full interval on first apply.
- `internal/discovery/k8s.go` implements a shared contract used by all six engines:
  ```go
  type DiscoveryResult struct {
      Source string
      Nodes  []graph.Node
      Edges  []graph.Edge
      Err    error // non-fatal; partial result still used
  }
  type Discoverer interface {
      Name() string
      Discover(ctx context.Context, pm *v1alpha1.PlatformMap) DiscoveryResult
  }
  ```
  K8s discoverer uses the `client-go` typed clientset scoped to `spec.namespace`,
  mapped into generic `graph.Node`s.
- Minimal `internal/graph/types.go` / `internal/graph/builder.go` — the JSON shape
  gets frozen here since Phase 4 (frontend) starts consuming it as a fixture
  immediately.
- Discoverers run concurrently via `golang.org/x/sync/errgroup`, each with its own
  `context.WithTimeout` (e.g. 10s) — the pattern every later engine follows, and
  the direct mechanism for not blowing the 60s budget once 5-6 sources are live.
- Read API: a small `net/http` mux (`internal/api/server.go`), started as a
  controller-runtime `manager.Runnable`, serving
  `GET /api/v1/platformmaps/{namespace}/{name}` off `mgr.GetClient()`'s cache,
  with CORS for the `ui/` dev server.

**Verify:** unit tests for `k8s.go` via `k8s.io/client-go/kubernetes/fake`;
`envtest`-based integration test applying a `PlatformMap` + two Deployments and
asserting `.status.topology` populates; manual `curl` against a real Kind cluster
returns matching JSON.

---

## Phase 3 — Graph builder hardening + frozen contract

`internal/graph/builder.go` becomes a pure, discovery-agnostic merge function with
real test coverage; the topology JSON schema is frozen so Go and frontend work can
proceed in parallel from here.

- Finalize `Node{ID, Type, Label, BrandColor, Status, Metadata map[string]any}` and
  `Edge{From, To, Type, Label}`. `BrandColor` is populated by the builder from the
  fixed brand-color table in `CLAUDE.md` (ArgoCD `#EF7B4D`, K8s `#326CE5`, Istio
  `#466BB0`, Kyverno `#1E40AF`, Prometheus `#EA580C`, GitHub Actions `#3B82F6`,
  Docker `#2496ED`) keyed off `Node.Type` — one lookup table, not scattered
  constants.
- `Merge(results ...DiscoveryResult) Graph` dedupes/joins nodes by a stable key
  (namespace+service-name), tolerates any subset of sources being absent or
  erroring, and never imports anything from `internal/discovery` — this is what
  keeps the builder decoupled from any specific discovery engine's shape.
- `internal/graph/builder_test.go`: table-driven tests — single-source graph,
  multi-source merge/join, a source erroring (graph still builds from the rest),
  empty input.
- Freeze the JSON shape as a committed fixture (`internal/graph/testdata/
  sample-topology.json`) that both the Go structs and the hand-written TS types in
  Phase 4 must match — a lightweight contract test, not codegen, for v0.1.

**Verify:** `go test ./internal/graph/...` green with meaningful coverage; frozen
fixture committed for Phase 4 to consume.

---

## Phase 4 — Frontend skeleton against mock data (parallel to Phase 6)

Full mockup layout rendered in-browser, wired to the frozen Phase 3 fixture — not
yet to a live controller, so this doesn't block on the remaining discovery engines.

- `ui/` scaffold: Vite + React 18 + TypeScript, Tailwind configured with the fixed
  dark-only palette (`#040912` bg, `#00C8FF` cyan, `#8B5CF6` violet) as theme
  tokens — no `dark:` variants needed since light mode is explicitly excluded.
- `@xyflow/react` (current maintained package, successor to legacy `reactflow`)
  for `ui/src/components/TopologyCanvas/`.
- `ui/src/components/NodeTypes/` — one custom node component per tool type, border
  color from the shared brand-color table, health-dot legend
  (green/yellow/red/grey) matching the mockup.
- `ui/src/components/Sidebar/` (left nav/filters), `ServiceDetails/` (right panel,
  tabs: Overview/Metrics/Logs/Events), `DeploymentTimeline/` (bottom-left event
  stream), `PolicyPanel/` (bottom-right Kyverno pass/warn/fail counts).
- **TanStack Query** for the future polling fetch (`usePlatformMap.ts`) with
  `refetchInterval` mirroring `spec.refreshInterval`; lightweight state (zustand or
  React context — not Redux) for cross-panel UI state (selected node → drives
  `ServiceDetails`; active filter → drives canvas visibility).
- `ui/src/api/client.ts` reads the static fixture behind the same interface it
  will use for the real API in Phase 5 — swapping is a one-line change.

**Verify:** `npm run dev`, visual comparison against
`ref-docs/file_000000006c407208be1de7ae77fa6da5.png`; component tests for
`NodeTypes` border colors and `Sidebar` filter logic.

---

## Phase 5 — Wire canvas to the real controller

- Swap `ui/src/api/client.ts` to call the Phase-2 HTTP read API instead of the
  static fixture; `usePlatformMap.ts` polling wired for real.
- Handle loading/error/"CRD not yet reconciled" states in the UI.

**Verify (first true end-to-end demo):** apply a `PlatformMap` to a Kind cluster
with 2-3 plain Deployments → real nodes appear on the real canvas within
`refreshInterval`, no ArgoCD/Istio/Kyverno/GitHub/Prometheus involved yet.

---

## Phase 6 — Remaining five discovery engines (parallelizable, one PR each)

Shared prerequisite: `internal/discovery/detect.go` uses the K8s discovery client
(`clientset.Discovery().ServerResourcesForGroupVersion(...)`) to check whether a
CRD group/version is installed before querying it. If absent, the engine returns a
non-fatal `DiscoveryResult{Err: ErrCRDNotInstalled}`, and the controller sets a
condition like `ArgoCDDetected=False` rather than failing the whole reconcile —
this is the concrete answer to "what if ArgoCD/Istio/Kyverno aren't installed."

- **`internal/discovery/argocd.go`** — `k8s.io/client-go/dynamic` client against
  `argoproj.io/v1alpha1` `applications` (not the full `argo-cd` Go module — avoids
  heavy transitive deps and coupling to ArgoCD's release cadence), reading sync
  status, health status, last commit SHA/message, revision via
  `unstructured.NestedString/NestedMap`.
- **`internal/discovery/istio.go`** — same dynamic-client approach against
  `networking.istio.io/v1beta1` `VirtualService`/`DestinationRule`, extracting
  traffic-weight subsets for the catalog v1/v2/v3 canary nodes.
- **`internal/discovery/kyverno.go`** — dynamic client against
  `wgpolicyk8s.io/v1alpha2` `PolicyReport`/`ClusterPolicyReport` (per the
  confirmed vendor-neutral decision).
- **`internal/discovery/github.go`** — `google/go-github` client, token from
  `tokenSecretRef` Secret, lists workflow runs, maps last-run conclusion per
  service.
- **`internal/discovery/prometheus.go`** — `prometheus/client_golang/api` +
  `api/prometheus/v1`, instant PromQL queries for error rate/p99 latency/CPU/mem
  per service.
- Note: Loki discovery belongs to ARIA (`aria/collectors/loki_collector.py`), not
  a Go engine — consistent with `CLAUDE.md`'s repo structure, which lists no
  `internal/discovery/loki.go`.
- RBAC in `config/rbac/role.yaml` expands incrementally, one engine's exact verbs
  per PR — keeps the ClusterRole auditable.

**Verify per engine:** fake dynamic client
(`k8s.io/client-go/dynamic/fake`) + hand-written fixture YAML in
`internal/discovery/testdata/` (minimal ArgoCD `Application`, Istio
`VirtualService`, Kyverno-populated `PolicyReport`) — engines testable without a
real control plane. `github.go`/`prometheus.go` tested via an injected
`httptest.Server` behind an interface (not a hardcoded `*http.Client`).

---

## Phase 7 — Helm chart

`helm install korion ./helm/korion -n korion-system --create-namespace` deploys
controller + frontend end to end (ARIA off by default).

- `helm/korion/Chart.yaml`, `values.yaml` (`controller.image`, `ui.image`,
  `ui.enabled`, `aria.enabled: false` explicitly pre-v0.2,
  `discovery.tools.*.enabled` mirroring the CRD's per-tool toggles).
- Controller Deployment+ServiceAccount+ClusterRole/Binding (synced from
  `config/rbac/`), UI Deployment+Service(+optional Ingress). CRDs delivered via
  Helm's `crds/` directory convention. Add a `make helm-sync-crds` target that
  copies `config/crd/bases/*` into `helm/korion/crds/` after every
  `make manifests` so the two never drift.
- `helm lint` + `helm template` snapshot test in CI.

**Verify:** fresh Kind cluster, `helm install`, controller pod running, no
RBAC-denied events, UI reachable via port-forward.

---

## Phase 8 — SuperHeros validation (v0.1 acceptance)

- Apply `config/samples/platformmap-superheros.yaml` against the SuperHeros
  cluster (Kind or the user's existing instance).
- Walk the `CLAUDE.md` v0.1 checklist: 6 service nodes, ArgoCD sync status, GitHub
  Actions last-run status, Istio traffic weights on catalog v1/v2/v3, Kyverno
  violation badges, `ServiceDetails` on click, `DeploymentTimeline` last 3
  syncs — all within 60s of apply.
- Tune `refreshInterval` and per-engine timeouts based on observed real-world
  latency (GitHub API and Prometheus range queries are the likely long poles).

**Verify:** manual checklist walk-through; consider automating later as a CI e2e
job (Kind + `helm install` + apply sample + poll the read API + assert
node/field counts) once Phase 7 lands.

---

## Phase 9 — ARIA foundation: FastAPI service, models, context builder (v0.2 start)

**Goal:** `aria/` exists as a real service that can assemble a full, correct
`PlatformContext` snapshot and expose it — but does not call an LLM yet. This
phase is deliberately isolated from prompt/LLM work so the hardest part (context
assembly under the 5-second budget) is proven and testable on its own first.

- `aria/requirements.txt`: FastAPI, `anthropic` SDK, `kubernetes` Python client,
  `httpx`, `asyncpg`, `pydantic`.
- `aria/models.py` — Pydantic models transcribed exactly from `CLAUDE.md`'s
  `PlatformContext`/`ServiceHealth`/`GitOpsState`/`DeploymentEvent`/
  `PolicyViolation`/`LogPattern`/`CIRunStatus`/`ClusterResources` schema. This
  schema is the contract for every phase after this one — freeze it here.
- `aria/context_builder.py` — assembles `PlatformContext` primarily by **reading
  the already-aggregated `.status.topology`** from the Phase 2 HTTP read API
  (`GET /api/v1/platformmaps/{ns}/{name}`), not by re-querying ArgoCD/Prometheus/
  GitHub from scratch. This is the single biggest lever for hitting the 5-second
  budget, since the Go controller already did that aggregation work.
- `aria/collectors/` — thin async collectors only for data that must be *fresher*
  than the controller's `refreshInterval` allows: `loki_collector.py` (recent
  error log patterns, last 30 min — this is ARIA's exclusive discovery
  responsibility per `CLAUDE.md`'s repo layout, no Go equivalent exists) and any
  other time-sensitive source identified during implementation. Each collector is
  called via `asyncio.gather` alongside the topology read, with a per-collector
  timeout; a failing collector yields a partial context with that source marked
  unavailable — never blocks the others (per `CLAUDE.md` rule).
- `aria/main.py` — FastAPI app with a debug endpoint (e.g.
  `GET /debug/context/{namespace}/{platform_map}`) that returns the assembled
  `PlatformContext` as JSON, with no LLM call — this is what phase verification
  exercises.

**Verify:** unit tests asserting `context_builder.py` assembles correctly from a
mocked topology-API response and mocked collectors; a timing test asserting
end-to-end assembly completes under 5 seconds against a real (or Kind-hosted)
Korion controller; a test asserting a collector failure still yields a usable
partial context rather than raising.

---

## Phase 10 — ARIA alert enrichment (v0.2 completion)

**Goal:** the first real LLM-backed capability — Alertmanager fires, ARIA
produces a structured, system-specific SRE analysis, and it lands in Slack.

- `aria/agent.py` — Anthropic Python SDK client built from scratch (no kagent
  dependency, per `CLAUDE.md` rule #2): takes a `PlatformContext` + a prompt
  template, calls Claude, parses the structured JSON response.
- `aria/prompt_templates/alert_enrichment.txt` — follows the `CLAUDE.md`
  structure exactly: role definition → context injection (full `PlatformContext`
  as JSON) → task specification → required output fields (`root_cause`,
  `explanation`, `confidence`, `kubectl_commands`, `prevention`, `severity`) →
  explicit constraint to reason only from provided context and never give
  generic K8s advice.
- Confidence-score enforcement: the response-construction layer (not the LLM's
  discretion) enforces that `confidence < 60` always sets a
  `requires_approval: true` flag regardless of `spec.autonomyLevel` — per
  `CLAUDE.md` rule #8, this is a hard rule, not a suggestion to the model.
- `aria/main.py` — `POST /webhooks/alertmanager` endpoint: receives the
  Alertmanager payload, builds context (Phase 9), calls `agent.py`, posts the
  structured analysis to Slack via `slackWebhookSecretRef`.
- `aria/models.py` gains the `AlertEnrichmentResponse` schema matching the
  prompt's required output fields.
- Only read-only kubectl operations are ever suggested/run (`get pods`,
  `get events`, `describe deployment`, `get logs`) — ARIA never runs
  `kubectl apply`; any remediation suggestion is framed as "commit this change to
  Git for ArgoCD to reconcile," per `CLAUDE.md` rule #3. This constraint lives in
  the prompt template's constraints section AND is enforced by never wiring any
  write-capable kubectl/K8s client into `agent.py` in the first place — no apply
  capability exists to misuse.

**Verify:** integration test firing a synthetic Alertmanager webhook payload
against a Kind cluster with a known induced failure (e.g. a CrashLoopBackOff
Deployment) and asserting the Slack message contains a root cause referencing the
actual failing service by name (proving system-specific, not generic, output);
a unit test asserting confidence < 60 always sets `requires_approval`.

---

## Phase 11 — ARIA health advisor + canary decision (v0.3)

**Goal:** the two remaining proactive ARIA capabilities from `CLAUDE.md`, both
reusing the Phase 9/10 foundation (context builder, `agent.py`, confidence
enforcement) — no new architectural machinery, just new prompt templates and
triggers.

- `aria/prompt_templates/health_advisor.txt` — scheduled scan (cron per
  `spec.features.healthAdvisor.schedule`) producing a platform health report:
  critical findings, warnings, cost observations, architecture recommendations.
  Delivered to Slack + surfaced in the dashboard (a `GET /api/v1/health-reports`
  style endpoint the frontend can poll — exact shape decided during
  implementation).
- A lightweight scheduler in `aria/main.py` (APScheduler or a simple asyncio loop
  reading the cron expression) triggers the health advisor at
  `spec.features.healthAdvisor.schedule`.
- `aria/prompt_templates/canary_decision.txt` — reads Istio traffic weights and
  Prometheus error-rate/latency figures already present in `.status.topology`
  for the versioned services (catalog v1/v2/v3), recommends promote/hold/rollback
  with reasoning tied to the actual observed metrics per version.
- `aria/main.py` gains a manual-trigger endpoint for canary decisions (matching
  `spec.features.canaryDecision`) and for on-demand SRE diagnostics
  (`spec.features.sreDiagnostics`) — the fourth ARIA capability from `CLAUDE.md`,
  naturally falling out of the same `agent.py` + context builder once a
  dedicated `sre_diagnosis.txt` prompt template is added.
- `aria/learning_store.py` — Postgres via `asyncpg`. Minimal schema: incidents,
  actions_taken, outcomes. Every ARIA diagnosis/enrichment/decision gets recorded
  here; future prompt calls can optionally reference recent relevant history
  (exact retrieval strategy — e.g. "last N incidents for this service" — decided
  during implementation, kept simple for v0.3).

**Verify:** unit tests per prompt template's JSON-parsing/validation; a scheduled
health-advisor test using a fake clock/trigger; a canary-decision test against
fixture Istio+Prometheus data (known good/bad traffic split) asserting the
recommended action matches expectations; `learning_store.py` tested against a
real Postgres in Docker (e.g. via `testcontainers` or a docker-compose dev
service) confirming writes/reads round-trip.

---

## Phase 12 — Helm integration for ARIA + full dashboard wiring

**Goal:** `aria.enabled: true` becomes a real, supported Helm configuration, and
the frontend surfaces ARIA output (not just topology) — the point at which
Korion + ARIA are a single installable, demoable product.

- `helm/korion/templates/aria-deployment.yaml`, ARIA `Service`, `Secret`
  templates for `apiKeySecretRef` / `slackWebhookSecretRef` / Postgres
  connection, `values.yaml` gains `aria.enabled`, `aria.image`,
  `aria.postgres.*`.
- `internal/controller/platformagent_controller.go` — the previously-inert
  `PlatformAgent` reconciler now does real work: validates the referenced
  `PlatformMap` exists, surfaces `PlatformAgent` status/conditions (e.g. "ARIA
  reachable", "last enrichment time") so the Go controller and ARIA stay loosely
  coupled through the K8s API rather than direct network assumptions.
- Frontend: a lightweight surface for ARIA output — e.g. alert-enrichment
  results and health-advisor reports rendered alongside `ServiceDetails` or as a
  new small panel — enough to demonstrate the "system-specific AI, not generic
  advice" differentiator end-to-end in the UI, without building the full
  "Provisioning"-style panel envisioned for the later Kforge integration (out of
  scope here).

**Verify:** fresh Kind cluster, `helm install ... --set aria.enabled=true`, fire a
synthetic alert, confirm the enrichment appears in both Slack and the dashboard;
confirm `aria.enabled=false` (the v0.1 default) still deploys cleanly with zero
ARIA-related pods/errors.

---

## Explicitly out of scope for this plan

- Kforge (the separate IDP project) — not part of Korion.
- No cloud-provider-specific detection (EKS/GKE/AKS) until actually needed —
  goes behind a `CloudProvider` interface if/when it is.
- No Playwright E2E or Storybook yet.
- The "Runtime Service Map" traffic panel seen in one mockup but not in
  `CLAUDE.md`'s stated layout — optional/deferred, not part of v0.1 acceptance.

## Key risks and how each phase addresses them

1. Optional CRDs (ArgoCD/Istio/Kyverno) absent from a cluster → discovery-client
   existence check + per-source status conditions, never a whole-reconcile
   failure (Phase 6).
2. 60s acceptance criteria vs refresh tuning → immediate reconcile on create,
   concurrent discoverers with per-engine timeouts (Phase 2/6).
3. ARIA's 5s context budget (Phase 9) → read pre-aggregated topology instead of
   re-querying every tool from scratch; isolate context-assembly proof from
   LLM/prompt work so the budget is validated before any prompt engineering
   begins.
4. Graph builder/discovery coupling → generic `DiscoveryResult` contract frozen
   in Phase 3, builder never imports `internal/discovery`.
5. Frontend/backend contract drift → fixture-based contract test (Phase 3/4)
   instead of premature codegen.
6. Browser → K8s auth path → thin read-only HTTP API in the manager binary
   (confirmed), never exposing K8s credentials to the browser.
7. Windows dev host → `envtest`'s `setup-envtest` binary is occasionally fiddly
   on native Windows; recommend running Go/Kind work through WSL2 or Git Bash
   consistently, since Docker Desktop's WSL2 backend is already implied by the
   verified toolchain. Note: `mingw32-make` substitutes for a missing bare
   `make`; a WSL2 `Ubuntu-22.04` distro also exists as fallback.

## Critical files to be created (representative, not exhaustive)

- `CLAUDE.md` (existing — authoritative spec, re-read before each phase)
- `api/v1alpha1/platformmap_types.go`, `platformagent_types.go`
- `internal/controller/platformmap_controller.go`
- `internal/discovery/k8s.go`, `detect.go`, `argocd.go`, `istio.go`,
  `kyverno.go`, `github.go`, `prometheus.go`
- `internal/graph/types.go`, `builder.go`, `builder_test.go`
- `internal/api/server.go`
- `ui/src/components/TopologyCanvas/`, `NodeTypes/`, `ServiceDetails/`,
  `DeploymentTimeline/`, `PolicyPanel/`, `Sidebar/`
- `ui/src/hooks/usePlatformMap.ts`, `ui/src/api/client.ts`
- `helm/korion/Chart.yaml`, `values.yaml`, `templates/`, `crds/`
- `config/samples/platformmap-superheros.yaml`,
  `platformagent-superheros.yaml`
- `aria/models.py`, `context_builder.py`, `agent.py`, `learning_store.py`,
  `main.py`
- `aria/collectors/loki_collector.py`
- `aria/prompt_templates/alert_enrichment.txt`, `health_advisor.txt`,
  `canary_decision.txt`, `sre_diagnosis.txt`
- `internal/controller/platformagent_controller.go`
