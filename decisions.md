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

## 2026-07-05 — mingw32-make mis-transcodes the em-dash in $(shell pwd)

- **Finding:** `mingw32-make controller-gen` (Phase 0's `LOCALBIN ?= $(shell
  pwd)/bin`) silently installed `controller-gen`/`addlicense` into a stray
  sibling directory with a mojibake name (`Project- Korion â€” K8s-Native...`)
  instead of the real repo path, because the subshell `pwd` mis-transcoded
  the em-dash (U+2014) in this repo's path. `bin/` appeared to not exist even
  though `make` reported success.
- **Fix:** `Makefile`'s `LOCALBIN` now uses `$(CURDIR)` (make's own working
  directory, not a subshell round-trip) instead of `$(shell pwd)`. The stray
  directory's binaries were recovered and moved into the real `bin/`, and the
  stray directory was deleted.
- **Why it matters:** any future `$(shell pwd)`-style construct in this
  Makefile risks the same silent misdirection on this host. Prefer `$(CURDIR)`
  or relative paths. Note `mingw32-make` still garbles the em-dash when
  *echoing* the command line to the terminal (cosmetic only — the actual
  argument passed to the executable is correct, confirmed by controller-gen's
  own error messages showing the correct path).

## 2026-07-05 — controller-gen v0.21 fatally errors on empty parent dirs in "..." globs

- **Finding:** `controller-gen ... paths="./api/..."` (and `paths="./..."`)
  fails with `no Go files in <path>/api` even though `api/v1alpha1/` has
  buildable Go files — controller-gen v0.21 treats any directory matched by a
  `...` glob that has zero `.go` files directly in it (like the parent `api/`,
  which only ever holds the `v1alpha1/` subpackage) as a fatal package-load
  error, not a skippable one.
- **Fix:** `Makefile` now defines `CONTROLLER_GEN_PATHS` (currently
  `./api/v1alpha1/...`) and passes it explicitly to both `generate` and
  `manifests` instead of a repo-wide `./...` or `./api/...` glob. Extend this
  variable (semicolon-joined) as new packages with kubebuilder markers land —
  `internal/controller` in Phase 2, `internal/discovery`/`internal/graph` in
  Phase 2/3, etc. — always pointing at the leaf package, never an empty
  parent.
- **Why it matters:** repo root and `api/`/`internal/` themselves will never
  contain `.go` files directly per `CLAUDE.md`'s layout, so a naive `./...` in
  `generate`/`manifests` will keep breaking as more packages are added unless
  this pattern is followed.

## 2026-07-05 — go.mod bumped to go 1.26.0 by k8s.io/api

- **Finding:** adding `k8s.io/api`, `k8s.io/apimachinery`, and
  `sigs.k8s.io/controller-runtime` (all `@latest` as of this date) forced
  `go.mod`'s `go` directive from `1.25.4` to `1.26.0` — `k8s.io/api@v0.36.2`
  itself declares `go >= 1.26.0`. Confirmed by attempting `go mod edit
  -go=1.25.4`, which then fails `go build` with that exact message.
- **Fix:** kept `go 1.26.0` in `go.mod` (locally installed 1.25.4 auto-fetches
  the 1.26.0 toolchain via `GOTOOLCHAIN=auto`, transparent but requires
  network on first build). Bumped `.github/workflows/ci.yml`'s two
  `setup-go` `go-version` pins from `1.22` to `1.26` to match.
- **Why it matters:** `CLAUDE.md` says "Go 1.22+" as a floor, not a ceiling —
  this isn't a violation, just a note that the effective minimum has moved
  with the k8s.io dependency graph. Future `go get -u`/`@latest` runs may
  bump this further; keep CI's pin in sync when it does.

## 2026-07-05 — Phase 2 controller test substitutes fake client for envtest

- **Decision:** `internal/controller/platformmap_controller_test.go` exercises
  `Reconcile()` against `sigs.k8s.io/controller-runtime/pkg/client/fake` with
  a stub `discovery.Discoverer`, not a real `envtest` (etcd + kube-apiserver)
  integration test as `docs/PLAN.md`'s Phase 2 verification step describes.
  Real end-to-end proof against an actual cluster was instead done manually:
  `kind create cluster`, apply CRDs + two Deployments/Services + a
  `PlatformMap`, run `go run ./cmd/manager` against the Kind kubeconfig, curl
  the read API, confirm the JSON topology and conditions are correct.
- **Why:** `docs/PLAN.md`'s own risk log (risk 7) already flagged
  `envtest`'s `setup-envtest` binary as "occasionally fiddly on native
  Windows." Given a real Kind cluster was going to be used for manual
  verification anyway (also required by the plan), doing the true end-to-end
  proof there and keeping the automated test on the fast, dependency-free
  fake client was the better tradeoff for this phase. `envtest` remains an
  option to introduce later if a scenario genuinely needs a real API server
  (e.g. CRD defaulting/validation webhooks) that the fake client can't
  exercise.

## 2026-07-05 — graph.Merge takes []Graph, not discovery.DiscoveryResult

- **Decision:** `internal/graph/builder.go`'s `Merge` has the signature
  `Merge(parts ...Graph) Graph`, not `Merge(results ...DiscoveryResult) Graph`
  as one reading of `docs/PLAN.md`'s Phase 3 section could suggest.
  `internal/controller/platformmap_controller.go` converts each
  `discovery.DiscoveryResult` into a `graph.Graph{Nodes, Edges}` before
  calling `Merge`.
- **Why:** `docs/PLAN.md` states both that `Merge` takes `DiscoveryResult`
  *and* that `internal/graph` "never imports anything from
  internal/discovery" — `DiscoveryResult` is defined in
  `internal/discovery/k8s.go`, so a literal `Merge(...DiscoveryResult)`
  signature in the `graph` package would require exactly the import the same
  paragraph forbids. Taking `Graph` (a type `graph` already owns) resolves
  the contradiction while preserving the intent: `graph` stays
  discovery-agnostic, and any future caller (not just
  `internal/controller`) can build a `Graph` from whatever shape it has.

## 2026-07-05 — Phase 3: BrandColor keyed by literal Node.Type, not a prefix/family match

- **Decision:** `internal/graph/colors.go`'s `brandColors` table is keyed by
  exact `Node.Type` strings (e.g. `"k8s-deployment"`, `"k8s-service"`), not a
  derived "family" prefix (e.g. splitting on the first `-`). `BrandColorFor`
  falls back to a neutral `defaultBrandColor` (`#6B7280`) for any type not yet
  in the table.
- **Why:** a prefix-derived scheme breaks down for multi-word families
  (`"github-actions-workflow"` vs `"github-repository"` want different
  colors but share the `github` prefix), and CLAUDE.md's brand-color table
  itself is a flat, explicit list, not a hierarchy. An exact-match table with
  a safe fallback is simpler and never silently misassigns a color; it costs
  one line per new Node.Type introduced in Phase 6, which is an acceptable
  tradeoff for correctness.

## 2026-07-05 — Phase 3: Merge joins same-ID nodes across sources instead of full overwrite

- **Decision:** `graph.Merge`'s conflict handling changed from "later source's
  Node fully replaces the earlier one" (Phase 2) to `joinNodes`: scalar
  fields (`Type`, `Label`, `Status`) are replaced only when the later source
  provides a non-empty value, and `Metadata` maps are shallow-merged key by
  key. Duplicate edges (identical `From`/`To`/`Type`/`Label` reported by more
  than one source) now collapse to one via `dedupeEdges`.
- **Why:** `docs/PLAN.md`'s Phase 3 section describes `Merge` as
  "dedupes/joins nodes by a stable key," not just dedupes -- once ArgoCD/
  Istio/Kyverno discovery (Phase 6) can describe an entity K8s discovery
  already produced a node for, a source that only contributes partial detail
  (e.g. ArgoCD adding sync status) must enrich that node, not silently erase
  the K8s-sourced replica counts by fully overwriting it. This is exercised
  now, ahead of Phase 6, via `internal/graph/builder_test.go`'s synthetic
  two-source metadata-join and edge-dedup cases.

## 2026-07-05 — Phase 3: frozen fixture verified against both a hand-built Graph and the real K8sDiscoverer

- **Decision:** `internal/graph/testdata/sample-topology.json` is guarded by
  two tests: `internal/graph/contract_test.go`'s `TestFrozenTopologyContract`
  (a hand-built `Graph` through `Merge`, structurally compared to the
  fixture via unmarshal-to-`any` + `reflect.DeepEqual`, so key order doesn't
  matter) and `internal/discovery/k8s_test.go`'s
  `TestK8sDiscoverer_MatchesFrozenTopologyContract` (a fake-clientset-backed
  `K8sDiscoverer.Discover` run through the same `graph.Merge`, compared to
  the identical fixture).
- **Why:** a fixture only checked against hand-typed Go literals proves the
  struct tags are self-consistent, but not that the actual Phase 2 discovery
  pipeline produces that shape. Having both closes that gap without
  `internal/graph` importing `internal/discovery` (the second test lives in
  `internal/discovery`, which already depends on `internal/graph`, not the
  reverse).

## 2026-07-05 — Tracking artifacts

- **Decision:** `task.md` (phase/status tracker) and `decisions.md` (this
  file) live at the repo root as durable, git-committed tracking — separate
  from the ephemeral in-session Task tool, which doesn't survive across
  sessions.

## 2026-07-05 — Phase 4: pin React to 18, not the Vite template's default 19

- **Finding:** `npm create vite@latest ui -- --template react-ts` scaffolds
  React 19 by default as of this date. `CLAUDE.md`'s tech stack explicitly
  pins "React 18 + TypeScript."
- **Fix:** immediately after scaffolding, `npm install react@^18 react-dom@^18
  @types/react@^18 @types/react-dom@^18` to force the resolution down to
  18.3.1. npm prints ERESOLVE peer-dependency warnings (several deps declare
  `peer react@"^18 || ^19"` or similar) but resolves cleanly since 18 is
  within every stated peer range.
- **Why it matters:** re-running `npm create vite@latest` in a future phase
  (or `npm update`) could silently reintroduce React 19 -- verify the
  installed version against `CLAUDE.md` after any frontend scaffolding step.

## 2026-07-05 — Phase 4: Tailwind v4 CSS-first config, not tailwind.config.js

- **Decision:** styling uses Tailwind v4.3 (current as of this date) via the
  `@tailwindcss/vite` plugin and a single `@theme { --color-korion-*: ... }`
  block in `ui/src/index.css`, not a v3-style `tailwind.config.js` color
  palette.
- **Why:** v4's CSS-first configuration is simpler for a small, fixed token
  set (CLAUDE.md's dark palette + tool brand colors) -- no separate JS config
  file, and the tokens are consumed directly as `bg-korion-bg`/
  `text-korion-cyan` utilities. `color-scheme: dark` is set once in `:root`;
  no `dark:` variants exist anywhere, per CLAUDE.md's "Dark theme ONLY" rule.

## 2026-07-05 — Phase 4: BFS-layered canvas layout instead of dagre

- **Decision:** `ui/src/components/TopologyCanvas/layout.ts` assigns node
  positions via a simple BFS-depth-as-column, order-as-row algorithm, not a
  dedicated graph-layout library (e.g. `dagre`, `elkjs`).
- **Why:** the mock topology is small and roughly a DAG (pipeline stages ->
  services), so a lightweight layered layout gives an acceptable left-to-right
  flow without a new dependency. Revisit if Phase 6's real discovery output
  produces graphs irregular enough (deep cycles, very high fan-out) that this
  starts producing overlapping or unreadable layouts.

## 2026-07-05 — Phase 4: React Flow MiniMap dropped from the canvas

- **Finding:** `@xyflow/react`'s `<MiniMap>` rendered as a large, mostly-blank
  panel overlapping the bottom-right nodes in manual visual verification
  (screenshotted via a scratch puppeteer-core script driving local Chrome,
  since no `chromium-cli`/Playwright was available on this Windows host).
  `ref-docs/file_000000006c407208be1de7ae77fa6da5.png`'s mockup has no
  minimap either -- only zoom +/-, a lock, and a fullscreen icon.
  Fix: removed `<MiniMap>` entirely; kept `<Background>` and `<Controls>`.

## 2026-07-05 — Phase 4: frontend contract fixture strategy

- **Decision:** `ui/src/api/fixtures/sample-topology.json` is a literal copy
  of the Go-owned `internal/graph/testdata/sample-topology.json`, guarded by
  `ui/src/api/fixtures/sample-topology.contract.test.ts` (a vitest test that
  reads the real Go fixture off disk via relative path and asserts deep
  equality) so the two can't silently drift. `client.ts`'s actual mock data
  source is a separate, hand-built `mockPlatformMap.ts` -- a richer SuperHeros
  dataset (18 nodes covering every discovery-engine node type from
  `internal/graph/colors.go`, plus mocked `DeploymentEvent`/`PolicySummary`
  data that isn't part of the frozen graph contract at all) needed to render
  the *full* mockup layout, which the frozen fixture's two nodes are too
  sparse to demonstrate.
- **Why:** keeps "prove we match the frozen contract" and "have enough data
  to build/review the full UI" as two separate, independently-verifiable
  concerns instead of stretching one fixture to do both jobs.

## 2026-07-05 — Phase 4: license-check/license-fix silently no-op on untracked files

- **Finding:** `make license-check`/`license-fix` build their file list from
  `git ls-files '*.go' '*.py' '*.ts' '*.tsx'` -- files that are untracked
  (never `git add`-ed) are invisible to `git ls-files` and silently excluded,
  so running `license-fix` against a freshly scaffolded `ui/` (all untracked)
  produced zero output and added no headers, with no error to indicate why.
- **Fix:** `git add ui` before running `license-fix`/`license-check` so new
  files are visible to `git ls-files`.
- **Why it matters:** the same silent no-op will recur for any future phase
  that scaffolds a batch of new files (Phase 6 engines, Phase 9's `aria/`) --
  stage new files before trusting a clean `license-check` result.

## 2026-07-05 — Phase 4: jsdom needs a ResizeObserver polyfill for React Flow tests

- **Finding:** `@xyflow/react`'s `<ReactFlow>` calls `ResizeObserver` to
  measure its container, which jsdom (vitest's test environment) doesn't
  implement -- component tests rendering `<ReactFlow>`/`<ReactFlowProvider>`
  threw `ReferenceError: ResizeObserver is not defined` inside a passive
  effect.
- **Fix:** `ui/src/setupTests.ts` installs a no-op `ResizeObserverStub` on
  `globalThis` before tests run.
- **Why it matters:** any future test that mounts `<ReactFlow>` (directly or
  via `TopologyCanvas`) depends on this stub already being in place via
  `vite.config.ts`'s `test.setupFiles`.
