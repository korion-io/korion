# Phase 5 Summary — Wire canvas to the real controller

Status: complete. See `docs/PLAN.md` for the full phase spec and
`decisions.md` for the design decisions made this phase and why.

This is the first true end-to-end demo: the React canvas now reads live
topology from the Phase 2 Go read API instead of Phase 4's static mock
fixture, closing the loop from `kubectl apply` to a rendered node on screen.

## What was built

- `ui/src/api/client.ts` — `getPlatformMap` now does a real `fetch` against
  `GET {API_BASE_URL}/api/v1/platformmaps/{namespace}/{name}` and transforms
  the raw `PlatformMap` custom resource JSON (metadata/spec/status, matching
  `api/v1alpha1/platformmap_types.go`) into the `PlatformMapView` shape the
  UI already renders from. `API_BASE_URL` defaults to
  `http://localhost:8082` (the manager's default `--api-bind-address`) in dev
  and `''` (same-origin, behind the Helm Service) in production, overridable
  via `VITE_API_BASE_URL`.
- `PlatformMapNotFoundError` — a distinct error class thrown on a 404
  response, so the UI can render "apply a PlatformMap" guidance instead of a
  generic failure message for the one case that's actually an expected part
  of the workflow (nothing applied yet, or a typo'd namespace/name).
- `ui/src/api/types.ts` — `PlatformMapView` gained `conditions:
  PlatformMapCondition[]` and `lastDiscoveryTime: string | null` (both real
  fields off `PlatformMap.status`), and dropped `cluster: string` -- a
  Phase-4 field invented for the mockup header that has no backing field
  anywhere in the `PlatformMap` CRD. See `decisions.md` for why this was
  replaced with `lastDiscoveryTime` rather than kept as dead data.
- `ui/src/App.tsx`:
  - `NAMESPACE`/`NAME` are now read from `VITE_PLATFORMMAP_NAMESPACE`/
    `VITE_PLATFORMMAP_NAME`, defaulting to the SuperHeros sample
    (`superheros`/`superheros-platform`) so the app targets the canonical
    test case out of the box, overridable for pointing a dev build at a
    different cluster/PlatformMap without a code change.
  - Three distinct non-happy-path states, per the phase spec's "loading/
    error/CRD not yet reconciled" requirement:
    1. `isLoading` — "Loading topology…" (unchanged from Phase 4).
    2. `isError` + `PlatformMapNotFoundError` — a specific message pointing
       at `config/samples/platformmap-superheros.yaml`.
    3. `isError`, other — generic failure message, now asking whether the
       controller's read API is reachable (distinguishes "wrong resource"
       from "backend down/unreachable").
    4. `data` but `lastDiscoveryTime === null` — the PlatformMap exists but
       the controller hasn't completed its first reconcile yet; canvas/
       panels are withheld until then rather than rendering an empty graph
       that looks like "no services found."
  - Header now shows `{namespace}/{name} · last discovered {time}` instead
    of the fabricated `cluster` field.
- `ui/src/vite-env.d.ts` — declares `ImportMetaEnv` for the three new
  `VITE_*` variables (`vite/client`'s base type is an empty interface;
  nothing declared these before Phase 5 needed configurable values).
- `ui/src/api/client.test.ts` — new unit tests: successful transform
  (topology/conditions/lastDiscoveryTime populated, deploymentEvents/
  policySummary zeroed since Phase 6 doesn't exist yet), defaults when
  `status` is entirely absent, `PlatformMapNotFoundError` on 404, generic
  `Error` on other non-OK statuses. Uses `vi.stubGlobal('fetch', ...)`, no
  new test-only dependency.
- Deleted `ui/src/api/fixtures/mockPlatformMap.ts` — the Phase 4 mock
  dataset became fully unused once `client.ts` stopped importing it (no test
  or component referenced it directly); kept dead code out per the "delete
  what's certainly unused" guidance rather than leaving it as an inert
  fallback nobody asked for. See `decisions.md` for the reasoning.
- `ui/src/api/fixtures/sample-topology.json` and its contract test are
  untouched — that fixture guards the Go/TS `Graph` shape, not the
  `PlatformMapView` wrapper, and is unaffected by this phase.

## Verification performed

- `npm run build` (`tsc -b && vite build`) — clean, no type errors.
- `npm test` (`vitest run`) — 11/11 tests passing across 4 files (7 prior +
  4 new in `client.test.ts`).
- `mingw32-make license-check` — clean after `git add -A ui` (new/deleted
  files staged first, per the Phase 4 finding that `license-check` is blind
  to untracked files).
- **Real end-to-end verification against a fresh Kind cluster**
  (`kind create cluster --name korion-dev`), matching the phase's own verify
  step ("apply a PlatformMap ... with 2-3 plain Deployments → real nodes
  appear on the real canvas"):
  1. `mingw32-make install` — both CRDs applied.
  2. Applied a `demo` namespace with three Deployments/Services (`frontend`
     1/1, `catalog` 2/2, `orders` 1/1) and a `PlatformMap`
     (`demo/demo-platform`, `autoDiscover: true`, `refreshInterval: 15s`).
  3. `go run ./cmd/manager` against the Kind kubeconfig.
  4. Polled `curl http://localhost:8082/api/v1/platformmaps/demo/demo-platform`
     — `status.topology` populated within ~1s of the manager starting (6
     nodes: 3 deployments all `healthy`, 3 services; 3 `routes-to` edges;
     `K8sDetected: True`; `lastDiscoveryTime` set) — well inside the 60s
     acceptance budget reused from Phase 2.
  5. Started `npm run dev` with `VITE_PLATFORMMAP_NAMESPACE=demo
     VITE_PLATFORMMAP_NAME=demo-platform` and screenshotted the running page
     via headless Chrome (`chrome.exe --headless=new --screenshot=...`,
     avoiding the puppeteer-core scratch dependency Phase 4 needed since
     Chrome's own CLI screenshot flag is sufficient for a static capture):
     confirmed all 6 real nodes rendered on the canvas with correct health
     dots, the header reading `demo/demo-platform · last discovered
     12:48:27 AM`, and the Deployment Timeline / Policy panels honestly
     empty (zeros, not fabricated data) since Phase 6's ArgoCD/GitHub/Kyverno
     engines don't exist yet.
  6. Restarted the dev server pointed at a nonexistent
     `demo/does-not-exist` PlatformMap and screenshotted the not-found
     state: confirmed the specific `PlatformMapNotFoundError` message
     rendered instead of a generic failure.
  7. Cluster torn down (`kind delete cluster`) after verification.

## Design decisions this phase (full detail in `decisions.md`)

1. Dropped `PlatformMapView.cluster` (a Phase-4 field with no backing data
   in the real CRD) in favor of surfacing the real `lastDiscoveryTime` in
   the header.
2. `lastDiscoveryTime === null` is the signal for "not yet reconciled,"
   distinct from "reconciled but the namespace happens to be empty" (which
   would have a real timestamp and zero nodes) -- the UI only withholds the
   canvas for the former.
3. Deleted the now-fully-unused `mockPlatformMap.ts` rather than keeping it
   as an unrequested mock-mode fallback.
4. Headless Chrome's own `--screenshot` CLI flag substituted for the
   puppeteer-core scratch script Phase 4 used, since a static single-page
   capture doesn't need a scripted browser automation library.

## Not done in this phase (by design)

- `deploymentEvents` and `policySummary` are always empty/zeroed — real data
  requires ArgoCD/GitHub discovery (timeline) and Kyverno discovery (policy
  summary), both Phase 6.
- No per-source condition UI (e.g. an "ArgoCD not detected" badge) — the
  `conditions` field is threaded through the type/client but not yet
  rendered anywhere; revisit once Phase 6 gives it more than one possible
  value (`K8sDetected` alone isn't informative enough to justify UI real
  estate yet).
- No RBAC/in-cluster Deployment for the controller — this phase's Kind
  verification ran `go run ./cmd/manager` locally against an admin
  kubeconfig, same as Phase 2; Helm packaging is Phase 7.
- No route/URL-param-driven namespace/name selection — still a single
  build-time env-configured PlatformMap, not a picker UI.
