# Phase 2 Summary — K8s discovery vertical slice + read API

Status: complete. See `docs/PLAN.md` for the full phase spec and
`decisions.md` for the two design deviations made this phase and why.

This is the single most important phase in the plan: it proves the whole
architecture end to end — CRD → controller → discovery → `.status.topology`
→ read API → JSON — using only plain Kubernetes Deployments/Services, before
any of the five optional-tool discovery engines exist.

## What was built

- `internal/graph/types.go` — `Node{ID, Type, Label, Status, Metadata}`,
  `Edge{From, To, Type, Label}`, `Graph{Nodes, Edges}`. Provisional shape,
  intentionally not yet frozen with `BrandColor`/a committed fixture — that's
  Phase 3.
- `internal/graph/builder.go` — `Merge(parts ...Graph) Graph`, deduping nodes
  by ID (later source wins on conflicts), tolerant of empty/nil parts.
  `internal/graph` still imports nothing from `internal/discovery` (see
  `decisions.md` for why `Merge`'s signature isn't the literal
  `Merge(...DiscoveryResult)` `docs/PLAN.md` suggested — that would have
  forced the exact import the same paragraph forbids).
- `internal/discovery/k8s.go` — the shared `DiscoveryResult`/`Discoverer`
  contract every engine will implement, plus `K8sDiscoverer`: lists
  Deployments and Services in `spec.namespace` via a `client-go` typed
  clientset, maps them to `graph.Node`s (health derived from
  `readyReplicas` vs desired), and adds a `routes-to` edge from a Service to
  a Deployment when the Service's selector matches the Deployment's pod
  template labels.
- `internal/controller/platformmap_controller.go` — `PlatformMapReconciler`:
  fetches the `PlatformMap`, runs all registered `Discoverer`s concurrently
  via `golang.org/x/sync/errgroup` (each under its own 10s
  `context.WithTimeout`), merges results with `graph.Merge`, writes
  `.status.topology` (JSON `RawExtension`), sets a `<Source>Detected`
  condition per discoverer (via `k8s.io/apimachinery/pkg/api/meta.SetStatusCondition`),
  and requeues at `spec.refreshInterval` (falling back to 30s if unset). A
  discoverer's error never fails the reconcile — it's carried into that
  source's condition instead, which is what lets optional tools (ArgoCD,
  Istio, Kyverno in Phase 6) be absent without breaking discovery.
  Immediate reconcile on `PlatformMap` create is controller-runtime's default
  watch behavior — no extra code needed.
- `internal/api/server.go` — `Server`, a `manager.Runnable` serving
  `GET /api/v1/platformmaps/{namespace}/{name}` off the manager's cached
  client (Go 1.22+ `net/http` pattern routing), with a permissive CORS
  wrapper for the `ui/` dev server. Returns 404 for a missing PlatformMap,
  200 + full object JSON otherwise. No K8s credentials are ever reachable
  from this endpoint.
- `cmd/manager/main.go` — the controller entrypoint: builds the scheme
  (client-go + `korion.io/v1alpha1`), constructs the manager, wires up a
  `kubernetes.Interface` for `K8sDiscoverer`, registers
  `PlatformMapReconciler` and the read API `Server` (via `mgr.Add`), adds
  healthz/readyz checks, and starts the manager.
- `config/rbac/role.yaml` — generated from `+kubebuilder:rbac` markers on
  `platformmap_controller.go`: read/watch on `services`, `deployments`, and
  `get/list/watch` + `status` `get/update/patch` on `platformmaps`.
- `Makefile`'s `CONTROLLER_GEN_PATHS` extended (semicolon-joined, per the
  Phase 1 finding) to include `internal/controller`, `internal/discovery`,
  `internal/graph`.

## Verification performed

- **Unit tests**, all passing:
  - `internal/graph/builder_test.go` — empty input, single source,
    multi-source merge/dedupe (later wins), an erroring/empty source not
    blocking the rest.
  - `internal/discovery/k8s_test.go` — using `k8s.io/client-go/kubernetes/fake`:
    correct nodes/edges/health status for a healthy and a degraded
    Deployment, a Service-selector-to-Deployment edge, namespace scoping,
    an empty namespace, and a simulated `List` API error surfacing as
    `DiscoveryResult.Err` with zero nodes.
  - `internal/controller/platformmap_controller_test.go` — using
    `sigs.k8s.io/controller-runtime/pkg/client/fake` with a stub
    `Discoverer`: topology gets merged into status, one source erroring
    doesn't block another source's nodes from landing (and produces the
    right `False`/`True` conditions), `autoDiscover: false` skips discovery
    entirely, and a not-found `PlatformMap` reconciles cleanly with no
    error/requeue.
  - `internal/api/server_test.go` — 200 + correct JSON body for a found
    PlatformMap, 404 for a missing one, CORS preflight headers present.
- `go build ./...`, `go vet ./...`, `gofmt -l .` — clean.
- `mingw32-make license-check` — clean.
- **Real end-to-end verification against a Kind cluster** (`kind create
  cluster --name korion-dev`):
  1. `mingw32-make install` — both CRDs applied.
  2. Applied a `demo` namespace with two Deployments/Services
     (`frontend`: 1/1 ready, `catalog`: 2/2 ready) and a minimal
     `PlatformMap` (`autoDiscover: true`, `refreshInterval: 30s`).
  3. `go run ./cmd/manager` against the Kind kubeconfig (cluster-admin
     creds, so no RBAC binding was needed for this local-process check —
     applying the generated ClusterRole/Binding is Phase 7's Helm job).
  4. `curl http://localhost:8082/api/v1/platformmaps/demo/demo-platform`
     returned, ~1s after the controller started, all 4 nodes (2
     deployments correctly marked `healthy`, 2 services), 2 correct
     `routes-to` edges, a `K8sDetected: True` condition, and a populated
     `lastDiscoveryTime` — well inside the 60s acceptance budget.
  5. `curl .../demo/does-not-exist` returned 404 as expected.
  6. Cluster torn down (`kind delete cluster`) after verification.

## Design deviations from the plan (see `decisions.md` for full rationale)

1. **`envtest` substituted with a fake-client reconcile test** for the
   automated suite, with the real integration proof done manually against
   Kind instead. `docs/PLAN.md`'s own risk log already flagged `envtest` as
   fragile on native Windows; since a Kind cluster was needed for manual
   verification anyway, that's where the true end-to-end check happened.
2. **`graph.Merge(parts ...Graph) Graph`**, not
   `Merge(results ...DiscoveryResult) Graph`. The literal signature in
   `docs/PLAN.md`'s Phase 3 section would require `internal/graph` to import
   `internal/discovery` (where `DiscoveryResult` lives), contradicting the
   same paragraph's "`graph` never imports discovery" rule. `Graph` is a
   type `graph` already owns, so `Merge` stays fully decoupled; the
   controller converts each `DiscoveryResult` to a `Graph` before calling
   it.
3. **`K8sDiscoverer.Name()` returns `"K8s"`** (not lowercase `"k8s"`) so its
   status condition reads `K8sDetected`, matching the casing convention
   `CLAUDE.md`/`docs/PLAN.md` use for other engines (`ArgoCDDetected`,
   `IstioDetected`).

## Not done in this phase (by design)

- The five remaining discovery engines (ArgoCD, Istio, Kyverno, GitHub,
  Prometheus) and `internal/discovery/detect.go`'s CRD-presence check —
  Phase 6.
- `BrandColor` on `graph.Node` and the frozen/committed topology fixture —
  Phase 3.
- No frontend yet — Phases 4/5.
- No RBAC `ServiceAccount`/`ClusterRoleBinding` or Deployment manifest for
  running the controller in-cluster — that's Helm's job in Phase 7; this
  phase's manual verification ran the manager as a local process against an
  admin kubeconfig, which is sufficient to prove the reconcile/API logic.
- The controller only watches `PlatformMap` itself, not the Deployments/
  Services it discovers — staleness between discoveries is bounded by
  `spec.refreshInterval` rather than reacting to live changes. Documented as
  a deliberate Phase 2 simplification in code comments; revisit if 30s
  proves too coarse.
