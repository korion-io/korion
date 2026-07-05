# Phase 7 Summary — Helm chart

Status: complete. See `docs/PLAN.md` for the full phase spec and
`decisions.md` for prior architectural decisions (browser → controller
read-API path, vendor-neutral discovery) this phase packages for real
deployment.

This phase turns Korion from "runs via `go run` / `npm run dev` against a
kubeconfig" into a single-command install:
`helm install korion ./helm/korion -n korion-system --create-namespace`
deploys the controller + frontend end to end, with ARIA off by default. It
also fills in the two build prerequisites the earlier phases had deferred —
the controller and UI container images — since the chart can't run without
them.

## What was built

### Container images (the missing build prerequisites)

- **`Dockerfile`** (repo root) — the controller image the Makefile's
  `docker-build` target always referenced but that never existed. Multi-stage:
  `golang:1.26` builds a fully static (`CGO_ENABLED=0`, `-ldflags="-s -w"`)
  `manager` binary, shipped on `gcr.io/distroless/static:nonroot` (no shell,
  no package manager, non-root uid 65532). `.dockerignore` trims the context
  to just the Go sources it COPYs (`cmd/`, `api/`, `internal/`, go.mod/sum).
- **`ui/Dockerfile`** — Vite static build (`node:24-alpine`) served by
  `nginx:1.27-alpine`. The baked `ui/nginx.default.conf` serves the SPA with
  history-API fallback for standalone `docker run`; in cluster the Helm chart
  mounts a fuller config over it (see below). `ui/.dockerignore` keeps host
  `node_modules`/`dist` out of the build context.

### Helm chart (`helm/korion/`)

- **`Chart.yaml`** — `apiVersion: v2`, `version: 0.1.0`, `appVersion: 0.1.0`.
  Image tags default to `.Chart.AppVersion` when unset, so bumping the release
  is a one-line change.
- **`values.yaml`** — `controller.*` (image, replicas, ports, resources,
  hardened security contexts, scheduling), `ui.enabled`/`ui.image`/
  `ui.service`/`ui.ingress`, `discovery.tools.*.enabled` (the five optional
  engines), and `aria.enabled: false` (explicit, inert pre-v0.2). Documented
  inline so `helm show values` is self-explanatory.
- **`templates/_helpers.tpl`** — standard name/label helpers plus
  component-scoped variants (`korion.controller.*`, `korion.ui.*`) and derived
  image refs (`korion.controller.image` folds in the appVersion fallback).
- **Controller resources** — `serviceaccount.yaml`, `clusterrole.yaml`
  (rules transcribed verbatim from the generated `config/rbac/role.yaml`, with
  a comment naming that file + the kubebuilder markers as the source of truth),
  `clusterrolebinding.yaml`, `controller-deployment.yaml` (args, liveness/
  readiness probes on `/healthz` + `/readyz`, security context), and
  `controller-service.yaml` (ClusterIP exposing only the `:8082` read API).
- **UI resources** (all gated on `.Values.ui.enabled`) —
  `ui-configmap.yaml` (nginx config that serves the SPA **and** reverse-
  proxies `/api/` to the controller Service's cluster DNS name),
  `ui-deployment.yaml` (mounts that ConfigMap over
  `/etc/nginx/conf.d/default.conf` via `subPath`, with a `checksum/config`
  pod annotation so a config change rolls the pods), `ui-service.yaml`, and
  `ui-ingress.yaml` (fully gated on `ui.ingress.enabled`, defaults off).
- **`NOTES.txt`** — post-install guidance (apply a PlatformMap, port-forward
  or Ingress URL, enabled-engine summary, an explicit note when someone sets
  the inert `aria.enabled=true`).
- **`crds/`** — both generated CRDs, delivered via Helm's `crds/` convention
  (installed once, never templated), populated by the pre-existing
  `make helm-sync-crds` target.

### Controller change to make `discovery.tools.*.enabled` real (not dead config)

`cmd/manager/main.go` gained five boolean flags (`--enable-argocd`,
`--enable-istio`, `--enable-kyverno`, `--enable-github`,
`--enable-prometheus`, all defaulting true); the discoverer slice is now built
conditionally instead of hardcoded. The chart wires these from
`discovery.tools.*.enabled`, so a source disabled in values is never
registered in the controller at all. This is a **cluster-wide** kill switch
that complements — does not replace — the **per-PlatformMap**
`spec.tools.*.enabled` toggles Phase 6 already honors. K8s core discovery has
no flag; it is always on. Without this wiring the values keys would be
documentation with no effect, so the flags exist to back them.

### CI

- New **`helm`** job (`azure/setup-helm@v4`): `helm lint helm/korion` plus
  `helm template` rendered three ways — default `--include-crds`,
  `ui.enabled=false`, and `ui.ingress.enabled=true` + a disabled engine. This
  is the plan's "helm template snapshot test": it fails on any template error
  or bad values path without pinning brittle golden output.
- Frontend job switched from `npm ci` to `npm install --no-audit --no-fund`
  and bumped Node 20 → 24 — see the lockfile note below.

## Verification performed

- `go build ./...`, `go vet ./...`, `gofmt -l` — clean after the main.go
  change. `go test ./...` — all packages green.
- `mingw32-make license-check` — clean (exit 0) after staging the new files.
- `helm lint helm/korion` — 0 failures (only the cosmetic "icon is
  recommended" info).
- `helm template` rendered and eyeballed for: default install, `ui.enabled=
  false` (no `ui-*` resources emitted), a disabled engine (`--enable-github=
  false` in the controller args), `ui.ingress.enabled=true` (Ingress rendered),
  and `--include-crds` (both CRDs present).
- **Full real end-to-end install on a fresh Kind cluster** (the plan's Phase 7
  verify, exceeded):
  1. `docker build` both images (`korion/korion:0.1.0`,
     `korion/korion-ui:0.1.0`); `kind create cluster --name korion-p7`;
     `kind load docker-image` both.
  2. `helm install korion helm/korion -n korion-system --create-namespace` —
     `STATUS: deployed`, NOTES rendered correctly.
  3. Both Deployments rolled out; `korion-controller` and `korion-ui` pods
     `1/1 Running`.
  4. Controller logs grepped for `forbidden`/`cannot list`/`denied` — **none**
     (RBAC from the chart's ClusterRole is sufficient); zero Warning events.
  5. Applied a `demo` namespace (2 Deployments + 2 Services) + a PlatformMap;
     within one refresh the status showed `K8sDetected=True`, 4 topology nodes
     (frontend/orders deployments + services), `lastDiscoveryTime` set, and the
     optional CRD engines correctly `...Detected=False` — proving the
     installed SA can actually discover, not just start.
  6. Port-forwarded `svc/korion-ui` and confirmed via curl: `/` serves the SPA
     (`<title>Korion</title>`, HTTP 200); `/api/v1/platformmaps/demo/
     demo-platform` proxies through nginx to the controller and returns the
     PlatformMap JSON (HTTP 200); a bogus name returns HTTP 404 passed through
     from the Go API — the same-origin production path works.
  7. `kind delete cluster` after verification.

## Design decisions / notes for later phases

1. **Windows-authored lockfile vs. Linux `npm ci`.** The committed
   `ui/package-lock.json` is generated on this Windows host, which never
   records the Linux-only optional native subtrees (`@emnapi/*`, the
   `@tailwindcss/oxide` native binding). Strict `npm ci` therefore fails its
   sync check on **any** Linux build — both the UI Docker image and the CI
   frontend job (a latent issue that predates this phase and would have bitten
   Phase 8's CI). Regenerating the lock on Windows can't fix it (npm only fully
   resolves the current platform's optional deps). Fix: both the `ui/Dockerfile`
   and the CI frontend job use `npm install --no-audit --no-fund` (lock-aware
   but tolerant) instead of `npm ci`. Documented in both files. If strict
   reproducibility is later wanted, regenerate the lock from a Linux
   environment (CI) and commit that.
2. **RBAC is transcribed into the chart, not `include`d from `config/rbac/`.**
   Helm can't read files outside the chart dir, so `clusterrole.yaml` carries a
   hand-copied version of the generated rules with a comment pointing at the
   real source of truth (the kubebuilder markers /
   `config/rbac/role.yaml`). These must be kept in sync by hand when a future
   engine's RBAC markers change. A `make helm-sync-rbac` analogue to
   `helm-sync-crds` is a reasonable future add if this drifts.
3. **nginx resolves the controller Service at startup.** `ui-configmap.yaml`
   uses a literal upstream (`<controller>.<ns>.svc.cluster.local`), not an
   nginx `resolver` + variable. The Service object is created by the same
   chart, so its DNS A record exists regardless of controller pod readiness —
   nginx starts cleanly. If startup ordering ever becomes a problem, switch to
   a `resolver` directive pointing at kube-dns.
4. **`aria.enabled` is inert but present.** Setting it true today creates no
   ARIA pods (NOTES.txt says so explicitly). The flag exists so enabling ARIA
   in Phase 12 is a values change, not a chart restructure — matching the
   plan's "v0.1 default deploys cleanly with zero ARIA resources."
5. **Image references default to `korion/korion` + `korion/korion-ui` at the
   chart appVersion.** These are placeholder repositories; real published
   images (and a registry) are a release-engineering concern, not a v0.1
   blocker. `make docker-build`/`docker-push` use the `IMG` var for the
   controller; the UI image has no Makefile target yet (built directly with
   `docker build ui/` this phase) — a `docker-build-ui` target is a reasonable
   Phase 8 add.

## Not done in this phase (by design)

- **No published/pushed images.** The images were built locally and
  `kind load`ed for verification; pushing to a registry (and a
  `docker-build-ui` Makefile target) is deferred — not needed to prove the
  chart.
- **No ARIA templates.** ARIA packaging is Phase 12; only the inert
  `aria.enabled` flag exists now.
- **No PodDisruptionBudget / HPA / NetworkPolicy / ServiceMonitor.** The
  controller runs a single replica with no leader election (unchanged from
  Phase 2), so a PDB/HPA would be premature; a Prometheus `ServiceMonitor` for
  the controller's own `:8080` metrics and a NetworkPolicy are sensible
  hardening adds for a later CNCF-readiness pass, out of scope for v0.1.
- **RBAC not yet namespace-scoped.** The chart ships the same cluster-wide
  read ClusterRole the generated manifests define — appropriate since
  discovery spans arbitrary namespaces. Tightening (e.g. per-namespace Roles)
  is a future hardening decision, not a v0.1 requirement.
- **No SuperHeros validation.** Applying the real
  `platformmap-superheros.yaml` against the full ArgoCD/Istio/Kyverno/
  Prometheus/GitHub stack is Phase 8; this phase verified the chart mechanics
  with plain Deployments + the same partial-discovery behavior Phase 5 used.
