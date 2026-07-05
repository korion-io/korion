# Phase 8 Summary — SuperHeros v0.1 acceptance validation

Status: complete. See `docs/PLAN.md` for the full phase spec and
`decisions.md` for prior architectural decisions. This phase closes out v0.1:
it proves the CLAUDE.md acceptance criteria hold end to end against a
Helm-installed controller on a real cluster, and turns that proof into a
repeatable, automated harness rather than a one-off manual walk-through.

## The core decision: hermetic fixture, not the full production stack

Phase 8's spec is "apply `platformmap-superheros.yaml` to the SuperHeros
cluster and walk the v0.1 checklist." Standing up the *entire* production stack
(ArgoCD + Istio + Kyverno + Prometheus + Loki controllers + 6 real services +
a live GitHub token) on demand is neither reproducible nor CI-friendly.

The key enabling insight: Korion's ArgoCD/Istio/Kyverno discovery engines read
**custom resources** through the dynamic client after a discovery-client
existence check (`internal/discovery/detect.go`) — they never talk to those
tools' controllers. So installing just the **CRDs + sample CRs** reproduces the
exact discovery path faithfully and hermetically, with no heavyweight control
planes. That is what the acceptance harness does. Only two criteria genuinely
need external systems — GitHub Actions last-run (needs an API token) and
Prometheus live metrics (needs a reachable endpoint) — and those are documented
as a live-cluster checklist rather than forced into the hermetic run.

## What was built (all under `test/e2e/`)

### SuperHeros fixtures (`test/e2e/fixtures/`)

- **`00-crds.yaml`** — minimal CRDs for `argoproj.io/v1alpha1 Application`,
  `networking.istio.io/v1beta1 VirtualService`/`DestinationRule`, and
  `wgpolicyk8s.io/v1alpha2 PolicyReport`/`ClusterPolicyReport`. Each uses
  `x-kubernetes-preserve-unknown-fields` and declares **no** status
  subresource, so the sample CRs' inline `status`/`summary`/`results` blocks
  persist through `kubectl apply` (a status subresource would strip them).
- **`10-workloads.yaml`** — the SuperHeros namespace: 7 Deployments (frontend,
  catalog-v1/v2/v3, inventory, orders, payment) + 5 Services. All run
  `registry.k8s.io/pause:3.9` (present in the Kind node image, no pull, Ready
  immediately). The catalog Service selects `app: catalog`, fanning out to all
  three versioned Deployments → 3 `routes-to` edges.
- **`20-argocd.yaml`** — an `Application` in the `argocd` namespace targeting
  `superheros`, `Synced`/`Healthy`, with `revision` + `reconciledAt` (the
  Deployment Timeline's source data).
- **`30-istio.yaml`** — catalog `VirtualService` splitting 20/30/50 across
  subsets v1/v2/v3, plus the matching `DestinationRule`.
- **`40-kyverno.yaml`** — a `PolicyReport` with 2 failing policies on the
  payment Deployment (→ `policyViolations: 2`) and 1 warning on catalog-v3,
  plus a `ClusterPolicyReport`.
- **`50-platformmap.yaml`** — the e2e PlatformMap. Deliberately **not** the
  production sample: GitHub + Prometheus are disabled so the run is hermetic;
  ArgoCD/Istio/Kyverno are gated only by CRD presence and run for real.
  `refreshInterval: 15s`.

### Assertion logic (`test/e2e/assert_acceptance.py` + test)

- `assert_acceptance.py` — pure logic over a PlatformMap object (from the read
  API or `kubectl get -o json`). Walks the CLAUDE.md v0.1 checklist over
  `.status.topology` + `.status.conditions` and prints a PASS/FAIL/SKIP table,
  exiting non-zero on any FAIL. No cluster dependency, so it is independently
  testable and reused unchanged by the harness. Checks node IDs/types/metadata
  against the exact strings the Go engines emit (`deployment/ns/name`,
  `argocd-application/...`, `istioTrafficWeights`, `policyViolations`, the four
  `<Source>Detected` conditions).
- `assert_acceptance_test.py` — 5 unit tests over a synthetic PlatformMap that
  mirrors the fixtures (good case all-pass + GitHub skip; unreconciled;
  missing istio weights; missing kyverno violations; ArgoCD condition False).
  Runs standalone (`python assert_acceptance_test.py`) or under pytest, no
  third-party dep.

### The harness (`test/e2e/run-acceptance.sh`)

Kind cluster → build + `kind load` controller image → install optional-tool
CRDs → `helm install` Korion (`ui.enabled=false`) → apply fixtures → apply
PlatformMap → port-forward the controller read API → poll until the first
discovery completes (90s ceiling, 60s acceptance budget) → run the asserter.
Flags: `--keep`, `--skip-build`, `--cluster`, `--image`. Cleanup (port-forward
kill + cluster delete) via an `EXIT` trap.

- **`README.md`** — how to run it and why the fixture approach is faithful.
- **`ACCEPTANCE.md`** — the v0.1 criteria table (automated vs. live) plus the
  exact live-cluster steps for the GitHub + Prometheus criteria the hermetic
  run skips, and the Phase 8 tuning notes.

### Supporting changes

- **`Makefile`** — new `docker-build-ui`/`docker-push-ui` targets (the UI image
  had no Makefile target, flagged as a Phase 8 add in the Phase 7 summary) and
  an `e2e` target that runs the harness.
- **`.github/workflows/ci.yml`** — new `e2e` job: installs kind + helm +
  python, runs the asserter unit tests, then the full acceptance harness on the
  ubuntu runner. This is the plan's "automate later as a CI e2e job."
- **`config/samples/platformmap-superheros.yaml`** — the two provisional bits
  Phase 6 flagged for Phase 8 tuning are now addressed as config: an explicit
  Prometheus `url` example (kube-prometheus-stack's
  `prometheus-operated.monitoring:9090`, since the same-namespace fallback is
  almost never right) and a `kubectl create secret` hint for the GitHub token.
- **`.gitignore`** — generalized the `__pycache__` ignore beyond `aria/`.

## Verification performed

- **Full acceptance harness on a fresh Kind cluster (the real Phase 8 proof):**
  `bash test/e2e/run-acceptance.sh` built `korion/korion:e2e`, created the
  cluster, `helm install`ed Korion, applied all fixtures, and asserted the
  checklist. Result: **14 passed, 0 failed, 1 skipped** (GitHub, as designed).
  First discovery completed in **~1s** — 17 nodes / 7 edges — far inside the
  60s budget. Every hermetically reproducible v0.1 criterion held:
  - 6 microservices (7 Deployments + 5 Services) on the canvas;
  - `argocd-application/argocd/superheros` node `Synced`/`Healthy`;
  - catalog Service carrying `istioTrafficWeights {v1:20, v2:30, v3:50}` plus
    the VirtualService/DestinationRule nodes;
  - `policyViolations: 2` folded onto the payment Deployment node + a
    PolicyReport node;
  - frontend node carrying image/replica metadata for ServiceDetails;
  - ArgoCD `revision` + `lastSyncTime` for the Deployment Timeline;
  - `K8sDetected`/`ArgoCDDetected`/`IstioDetected`/`KyvernoDetected` all True.
- A second run with `--skip-build` (reusing the cluster) confirmed the script's
  own exit code is **0** and its `EXIT` trap tears the cluster down cleanly.
- `python test/e2e/assert_acceptance_test.py` — 5/5 pass, both from the repo
  root (as CI invokes it) and from `test/e2e/`.
- `helm lint helm/korion` — 0 failures.
- `mingw32-make license-check` — clean after `git add -A` (new `.py` files
  carry the Apache header; the `.sh` carries one too, though it's outside the
  check's `.go/.py/.ts` glob).
- No Go sources changed this phase, so the Go build/test surface is unaffected.

## Design decisions / notes

1. **Hermetic fixture over full stack** (above). The automated run validates
   the discovery→graph→status→read-API path for K8s/ArgoCD/Istio/Kyverno for
   real; GitHub/Prometheus live verification is the documented manual step.
2. **`pause:3.9` for workload containers.** Using the Kind-node-baked pause
   image keeps the run offline and fast; the fixture proves topology discovery,
   not service behavior, so real service images would only add pull flakiness.
   Cost: the ServiceDetails `image` metadata shows `pause:3.9` rather than a
   realistic tag — cosmetic, and not asserted beyond "image is present."
3. **Assertions in Python, not jq.** `jq` isn't reliably present on the Windows
   Git Bash host; `python` is (and is on CI). A single asserter module is also
   unit-testable in isolation, which a pipeline of `jq` filters is not.
4. **ArgoCD/Istio/Kyverno ignore their `spec.tools.*.enabled` flag** — those
   engines gate only on CRD presence (existing Phase 6 behavior); only
   GitHub/Prometheus honor the per-tool enable flag. So the e2e PlatformMap
   disables GitHub/Prometheus (effective) while ArgoCD/Istio/Kyverno run purely
   because their CRDs are installed. This matched behavior exactly and needed no
   code change.
5. **Prometheus fallback address left as-is in code, fixed in config.** The
   same-namespace guess in `prometheus.go` is documented as best-effort; the
   real fix is setting `spec.tools.prometheus.url`, which the production sample
   now does by example. Changing the hard-coded default without a real cluster
   to confirm the "right" Service name would be speculation.

## Not done in this phase (by design)

- **No live GitHub/Prometheus assertion in CI.** Both need secrets/endpoints a
  hermetic run can't provide; they're covered by `ACCEPTANCE.md`'s manual
  steps. A future enhancement could add a mock Prometheus HTTP server (canned
  `/api/v1/query` vectors) to the fixture to exercise the metrics enrichment
  path offline.
- **No code tuning of `discoveryTimeout`/`refreshInterval`.** With all engines
  concurrent under the 10s per-engine timeout, worst case is ~10s vs. the 60s
  budget; observed discovery was ~1s. Nothing to tune without evidence of a
  real long pole, which needs a live GitHub/Prometheus run.
- **v0.1 is now complete.** Next is Phase 9 (ARIA foundation: FastAPI service,
  Pydantic models, context builder — no LLM yet), the start of v0.2.
