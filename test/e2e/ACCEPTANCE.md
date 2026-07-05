# Korion v0.1 acceptance checklist — SuperHeros platform

The v0.1 acceptance criteria from `CLAUDE.md`, mapped to how each is verified.
The **Automated** column is what `run-acceptance.sh` asserts on every run (see
[`README.md`](README.md)); the **Live** notes cover the two criteria that need
a real GitHub token / Prometheus endpoint and are therefore validated against
an actual SuperHeros cluster rather than the hermetic Kind fixture.

Target: within **60 seconds** of applying the PlatformMap, the criteria below
hold. Observed on the Kind harness: first discovery completes in **~1s**
(17 nodes / 7 edges), well inside the budget.

| # | v0.1 criterion (CLAUDE.md) | Automated | How |
|---|----------------------------|-----------|-----|
| 1 | All 6 services as nodes | ✅ | 7 Deployments (frontend, catalog-v1/v2/v3, inventory, orders, payment) + 5 Services present |
| 2 | ArgoCD node shows sync status | ✅ | `argocd-application/argocd/superheros` node, `syncStatus=Synced`, `healthStatus=Healthy` |
| 3 | GitHub Actions node shows last run | ⏸️ live | disabled in the hermetic run — see below |
| 4 | catalog v1/v2/v3 Istio traffic weights | ✅ | catalog Service node carries `istioTrafficWeights {v1:20, v2:30, v3:50}` |
| 5 | Kyverno violation badge on affected nodes | ✅ | payment Deployment node carries `policyViolations: 2` |
| 6 | Node click → ServiceDetails with live metrics | ✅ / ⏸️ | node metadata (image, replicas) present; live Prometheus metrics need an endpoint |
| 7 | Deployment Timeline shows last ArgoCD syncs | ✅ | ArgoCD node carries `revision` + `lastSyncTime` (timeline source data) |

## Live-cluster verification for the two external-dependency criteria

Run against a real SuperHeros cluster (Kind or existing), where ArgoCD, Istio,
Kyverno, Prometheus, and GitHub are actually installed. Use the production
sample, not the e2e fixture:

```bash
helm upgrade --install korion ./helm/korion -n korion-system --create-namespace
kubectl create ns superheros    # if not already present

# (3) GitHub Actions last-run status — needs a PAT with repo + actions:read
kubectl -n superheros create secret generic korion-github-secret \
  --from-literal=token=<GITHUB_PAT>

# (6) Prometheus metrics — set spec.tools.prometheus.url to your query endpoint
#     (edit config/samples/platformmap-superheros.yaml; the sample now carries
#      a kube-prometheus-stack example URL)

kubectl apply -f config/samples/platformmap-superheros.yaml
kubectl -n korion-system port-forward svc/korion-ui 8080:80   # or svc/korion-controller 8082:8082
```

Then confirm in the UI (or via the read API):

- **(3)** a `github-actions-workflow` node per workflow, each showing its
  latest run conclusion (success → healthy, failure → degraded), and
  `GitHubDetected=True` in `kubectl get platformmap superheros-platform -n
  superheros -o jsonpath='{.status.conditions}'`.
- **(6)** the ServiceDetails panel shows non-zero `errorRatePct`,
  `latencyP99Ms`, `cpuUsageM`, `memoryUsageMi` on services with traffic, and
  `PrometheusDetected=True`.

If a tool is genuinely absent, discovery degrades gracefully: the
corresponding `<Source>Detected` condition goes `False` with a reason, and the
rest of the graph still builds — never a whole-reconcile failure.

## Tuning notes (Phase 8)

- **Discovery budget.** Each engine runs concurrently under a 10s timeout
  (`discoveryTimeout`), so even with all six live the worst case is ~10s,
  comfortably inside the 60s criterion. The sample `refreshInterval` is 30s.
- **Prometheus endpoint.** The same-namespace fallback address is almost never
  correct; set `spec.tools.prometheus.url` explicitly. The PromQL templates
  (`internal/discovery/prometheus.go`) assume standard
  `http_requests_total` / `http_request_duration_seconds_bucket` /
  `container_*` metric names grouped by a `service` label — adjust to the real
  SuperHeros metric labels if they differ.
- **GitHub / Prometheus are the long poles** (external HTTP), which is exactly
  why they sit behind the per-engine timeout and never block the K8s/ArgoCD/
  Istio/Kyverno half of the graph.
