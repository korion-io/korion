# Phase 6 Summary — Remaining five discovery engines

Status: complete. See `docs/PLAN.md` for the full phase spec and
`decisions.md` for the prior architectural decisions (dynamic-client vs
vendored SDKs, vendor-neutral Kyverno source, Loki-belongs-to-ARIA) this
phase implements.

This phase adds the five discovery engines that turn Korion from a
plain-K8s vertical slice into the full "GitHub Actions → ArgoCD → K8s →
Istio → Kyverno → Prometheus" pipeline view. Every engine implements the
same `discovery.Discoverer` contract frozen in Phase 2 and emits generic
`graph.Node`/`graph.Edge` values whose `Node.Type` strings match the
anticipatory `brandColors` table Phase 3 pre-populated — so no
`internal/graph` change was needed, the color table Just Worked.

## What was built

### Shared prerequisite

- `internal/discovery/detect.go` — `GroupVersionAvailable(disco,
  groupVersion)` wraps the discovery client's
  `ServerResourcesForGroupVersion`, and the `ErrCRDNotInstalled` sentinel.
  The three CRD-backed engines (ArgoCD, Istio, Kyverno) call this before
  touching the dynamic client; when the group/version is absent they return
  `DiscoveryResult{Err: ErrCRDNotInstalled}` with zero nodes. The controller
  (unchanged from Phase 2) already carries that error into a
  `<Source>Detected=False` condition rather than failing the reconcile —
  this is the concrete answer to "what if ArgoCD/Istio/Kyverno aren't
  installed."

### The five engines (all in `internal/discovery/`)

- **`argocd.go`** — `dynamic.Interface` against `argoproj.io/v1alpha1`
  `applications` (list all namespaces, filter to those whose
  `spec.destination.namespace` matches `spec.namespace`). Emits an
  `argocd-application` node per app with sync status, health status,
  revision, last-sync message/time, and repoURL pulled via
  `unstructured.NestedString`. ArgoCD's health vocabulary
  (Healthy/Degraded/Missing/…) is mapped down to Korion's coarse
  healthy/degraded/unknown `Node.Status`.
- **`istio.go`** — dynamic client against `networking.istio.io/v1beta1`
  `virtualservices` + `destinationrules` in `spec.namespace`. Emits an
  `istio-virtualservice` and `istio-destinationrule` node each, **and**
  additionally emits a bare enrichment node keyed by the underlying
  Service's stable ID (`service/<ns>/<host>`) carrying
  `istioTrafficWeights` (a `subset → weight` map extracted from
  `spec.http[].route[]`). Because that ID is the same one `K8sDiscoverer`
  already produces, `graph.Merge`'s `joinNodes` folds the canary weights
  onto the existing Service node instead of creating a duplicate — this is
  exactly the cross-source enrichment Phase 3's `Merge` hardening was built
  for, now exercised by a real second source.
- **`kyverno.go`** — dynamic client against the vendor-neutral
  `wgpolicyk8s.io/v1alpha2` `policyreports` (namespaced) +
  `clusterpolicyreports` (cluster-scoped), per the confirmed
  policy-engine-neutral decision. Emits a report node per report with its
  pass/fail/warn/error/skip summary, and walks each report's `results[]`:
  for any result targeting a `Deployment`, it tallies fail/error/warn counts
  onto that Deployment's stable node ID (`deployment/<ns>/<name>`) as an
  enrichment node (`policyViolations`, `policyViolationsFail/Error/Warn`) —
  again joined onto the K8s-discovered Deployment node by `Merge`, giving
  the canvas its per-service violation-count badge.
- **`github.go`** — `google/go-github/v68` client authenticated with a token
  resolved from `spec.tools.github.tokenSecretRef`. Parses owner/repo from
  `spec.repository`, emits a `github-repository` node plus a
  `github-actions-workflow` node per workflow, each enriched with its latest
  run's status/conclusion/URL/branch/SHA. Run conclusion is mapped to
  `Node.Status` (success→healthy, failure/timed_out/cancelled→degraded,
  in-progress→unknown). Unlike the CRD engines, GitHub is explicit user
  configuration, so a disabled tool is a silent no-op but a
  misconfiguration (missing `tokenSecretRef`, unparseable repo) is a
  surfaced `DiscoveryResult.Err`.
- **`prometheus.go`** — `prometheus/client_golang/api` +
  `api/prometheus/v1` instant PromQL queries for error rate, p99 latency,
  CPU (millicores), and memory (Mi), each grouped `by (service)` so one
  query yields every service at once. Results enrich the matching
  `service/<ns>/<name>` node's metadata (`errorRatePct`, `latencyP99Ms`,
  `cpuUsageM`, `memoryUsageMi`). A failed query for one metric records the
  first error but never blocks the other three.

### Supporting

- `internal/discovery/secrets.go` — `SecretResolver` interface +
  `ClientsetSecretResolver` (the real client-go-backed implementation wired
  into `main.go`). Keeps `GitHubDiscoverer` depending on a one-method
  interface so its unit tests use a stub, not a fake clientset.
- `cmd/manager/main.go` — builds a `dynamic.NewForConfig` client and
  registers all five new discoverers alongside `K8sDiscoverer`. ArgoCD/
  Istio/Kyverno share the one dynamic client + `clientset.Discovery()`;
  GitHub gets the `ClientsetSecretResolver`; Prometheus is constructed with
  its zero value (its `NewAPI` field defaults to the real HTTP client).
- `internal/controller/platformmap_controller.go` — RBAC kubebuilder
  markers expanded: `get` on `secrets`, and `get;list;watch` on
  `argoproj.io applications`, `networking.istio.io virtualservices/
  destinationrules`, and `wgpolicyk8s.io policyreports/
  clusterpolicyreports`. No reconcile-logic change — the Phase 2 loop was
  already source-agnostic.
- `config/rbac/role.yaml` — regenerated via `mingw32-make manifests` to
  match the new markers.
- `go.mod`/`go.sum` — added `github.com/google/go-github/v68`, promoted
  `github.com/prometheus/client_golang` and `github.com/prometheus/common`
  to direct dependencies (`go mod tidy`).

## Verification performed

- **Unit tests**, all passing (`go test -count=1 ./...` green across all
  packages):
  - `detect_test.go` — group/version reported unavailable against an empty
    fake discovery client.
  - `argocd_test.go` — namespace-scoped filtering (only apps whose
    destination namespace matches), sync/health/revision metadata,
    health→status mapping, and the `ErrCRDNotInstalled` path when the CRD is
    absent. Uses `dynamicfake.NewSimpleDynamicClientWithCustomListKinds` +
    a `discoveryfake.FakeDiscovery` helper (`discoveryClientWithGroupVersions`)
    shared by all three CRD engine tests.
  - `istio_test.go` — VirtualService/DestinationRule nodes plus a
    traffic-weight enrichment node with the correct `v1/v2/v3` split
    (20/30/50), and the CRD-absent path.
  - `kyverno_test.go` — report node + per-Deployment violation tally
    (catalog: 2 fails → `policyViolations: 2`; orders: 1 warn, 0
    fail/error), and the CRD-absent path.
  - `github_test.go` — an `httptest.Server` serving the workflows +
    workflow-runs endpoints behind an injected `BaseURL`, asserting
    repo+workflow node counts and success→healthy / failure→degraded status;
    plus disabled-tool no-op and missing-`tokenSecretRef` error cases.
  - `prometheus_test.go` — an `httptest.Server` serving `/api/v1/query`
    with per-metric fixtures, asserting all four metrics land on the right
    service nodes; plus disabled-tool no-op and unreachable-endpoint error
    cases. The engine's `NewAPI` field is overridden to point the client at
    the test server.
- `go build ./...`, `go vet ./...`, `gofmt -l .` — all clean.
- `mingw32-make license-check` — clean (all new files carry the Apache 2.0
  header).
- `mingw32-make generate manifests` — regenerated; the only manifest diff
  was `config/rbac/role.yaml`'s new rules. No CRD schema drift (this phase
  added no API-type fields).

## Design decisions / notes for later phases

1. **Enrichment-node pattern for cross-source data.** Istio traffic weights,
   Kyverno violation counts, and Prometheus metrics are all emitted as bare
   `graph.Node`s carrying only an ID (matching an existing K8s node) plus
   `Metadata` — never as standalone nodes. `graph.Merge`/`joinNodes` folds
   them onto the K8s-discovered Service/Deployment node. This keeps each
   engine ignorant of the others and relies entirely on the stable-ID join
   contract frozen in Phases 2/3. The Phase 3 `Merge` metadata-join logic,
   previously exercised only by synthetic two-source tests, is now driven by
   three real independent sources.
2. **`PrometheusDiscoverer` builds its client per-`Discover`, not once at
   startup.** Unlike the CRD engines (which query the one in-cluster dynamic
   client), a Prometheus endpoint can vary per PlatformMap via
   `spec.tools.prometheus.url`. The engine holds a `NewAPI func(address)`
   field (defaulting to the real HTTP client, overridable in tests) and
   calls it fresh each reconcile. `prometheusAddress` falls back to
   `http://prometheus.<ns>.svc.cluster.local:9090` when no URL is set — a
   deliberate simplification flagged for tuning in Phase 8, since the real
   Service name/namespace varies by install method.
3. **CRD engines vs configured tools handle "absent" differently, by
   design.** ArgoCD/Istio/Kyverno absence is discovered at runtime
   (`GroupVersionAvailable` → `ErrCRDNotInstalled`) and is a normal,
   expected `False` condition. GitHub/Prometheus are explicit user config:
   `enabled: false` is a silent no-op, but an enabled-but-misconfigured tool
   surfaces a real error condition. Both paths are non-fatal to the overall
   reconcile.
4. **The PromQL query templates are provisional.** They use standard
   metric names (`http_requests_total`, `http_request_duration_seconds_bucket`,
   `container_cpu_usage_seconds_total`, `container_memory_working_set_bytes`)
   grouped by a `service` label. Real SuperHeros metric names/labels must be
   confirmed and the templates tuned in Phase 8.

## Not done in this phase (by design)

- **No Loki engine.** Per `decisions.md`, recent-error-log discovery is
  ARIA's exclusive responsibility (`aria/collectors/loki_collector.py`,
  Phase 9), not a Go engine — the `loki-log-source` brand color in the
  table stays anticipatory.
- **No real-cluster end-to-end verification of the new engines.** Unlike
  Phase 2 (which stood up Kind + plain Deployments), these five engines were
  verified against fake dynamic clients / `httptest.Server`s, since a real
  proof needs ArgoCD + Istio + Kyverno + Prometheus + a GitHub token all
  installed — that full-stack validation is Phase 8's SuperHeros acceptance
  walk-through against the actual cluster.
- **No frontend changes.** The new node types already have brand colors and
  the frozen graph contract is unchanged, so the Phase 4/5 canvas renders
  them as-is; any tool-specific `ServiceDetails` polish is out of scope here.
- **RBAC is a broad ClusterRole.** The generated role grants cluster-wide
  read on the discovered resources; scoping/namespacing and the
  ServiceAccount/binding + Deployment manifests are Helm's job in Phase 7.
