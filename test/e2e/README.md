# Korion e2e — SuperHeros v0.1 acceptance (Phase 8)

This directory holds the automatable form of Phase 8's acceptance
walk-through: a hermetic Kind + Helm run that applies a SuperHeros-shaped
fixture and asserts the CLAUDE.md v0.1 acceptance criteria against the
controller's read API.

## What it does

`run-acceptance.sh`:

1. Creates a Kind cluster (`korion-e2e`).
2. Builds the controller image and `kind load`s it.
3. Installs the optional-tool CRDs (`fixtures/00-crds.yaml`).
4. `helm install`s Korion (`ui.enabled=false` — this run asserts the API).
5. Applies the SuperHeros fixtures: 7 Deployments + 5 Services, an ArgoCD
   `Application`, an Istio `VirtualService`/`DestinationRule` (catalog canary
   20/30/50), and Kyverno `PolicyReport`/`ClusterPolicyReport`.
6. Applies the PlatformMap and polls the read API until the first discovery
   completes (60s acceptance budget; observed ~1s).
7. Runs `assert_acceptance.py` over the returned PlatformMap JSON.

```bash
make e2e                      # or:
bash test/e2e/run-acceptance.sh
bash test/e2e/run-acceptance.sh --keep         # leave the cluster up to poke at
bash test/e2e/run-acceptance.sh --skip-build   # reuse an already-loaded image
```

Requires `kind`, `kubectl`, `helm`, `docker`, and `python` on `PATH`.

## Why the fixture, not the real SuperHeros cluster

Korion's ArgoCD/Istio/Kyverno engines read **custom resources** through the
dynamic client after a discovery-client existence check
(`internal/discovery/detect.go`) — they never talk to those tools'
controllers. So installing just the CRDs + sample CRs reproduces the exact
discovery path faithfully and hermetically, without standing up ArgoCD, Istio,
and Kyverno control planes.

The two criteria that genuinely need external systems — **GitHub Actions**
last-run status (needs an API token) and **Prometheus** live metrics (needs a
reachable endpoint) — are disabled in `fixtures/50-platformmap.yaml` so the
automated run stays deterministic. They are covered by the manual live-cluster
steps in [`ACCEPTANCE.md`](ACCEPTANCE.md).

## Testing the asserter itself

`assert_acceptance.py` is pure logic over a PlatformMap object, unit-tested
without a cluster:

```bash
python test/e2e/assert_acceptance_test.py     # standalone, no pytest needed
pytest test/e2e/assert_acceptance_test.py     # or under pytest
```
