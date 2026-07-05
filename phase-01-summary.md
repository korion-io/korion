# Phase 1 Summary — API types + CRD generation

Status: complete. See `docs/PLAN.md` for the full phase spec and
`decisions.md` for the environment/tooling issues found and fixed along the
way.

## What was built

- `api/v1alpha1/groupversion_info.go` — `korion.io/v1alpha1` scheme
  registration.
- `api/v1alpha1/platformmap_types.go` — `PlatformMap` spec (repository,
  namespace, autoDiscover, per-tool `ToolsConfig`/`ToolConfig`/
  `GitHubToolConfig`, `refreshInterval`) and status (opaque `Topology`
  `RawExtension` placeholder pending the Phase 3 graph-schema freeze,
  `conditions`, `lastDiscoveryTime`).
- `api/v1alpha1/platformagent_types.go` — full `PlatformAgent` spec
  (`platformMap` ref, `autonomyLevel`, `llmProvider`, `features`,
  `runbooks`) transcribed from `CLAUDE.md`'s sample. Scaffolded but inert —
  no reconciler registered until Phase 12, per the existing decision log.
- `api/v1alpha1/zz_generated.deepcopy.go` — generated via `controller-gen`.
- `config/crd/bases/korion.io_platformmaps.yaml`,
  `korion.io_platformagents.yaml` — generated CRD manifests.
- `config/samples/platformmap-superheros.yaml`,
  `platformagent-superheros.yaml` — transcribed verbatim from `CLAUDE.md`.

Types were hand-written rather than scaffolded via `kubebuilder init`/
`kubebuilder create api`, to avoid the generator overwriting Phase 0's
already-in-place `Makefile`/`go.mod`/directory layout. `controller-gen`
(already wired into the Makefile in Phase 0) produces the deepcopy code and
CRD YAML from kubebuilder markers on the hand-written types, which satisfies
the same phase goal (installable CRDs, generated deepcopy/manifests) with
less risk on this repo.

No `config/rbac/*.yaml` was generated — `controller-gen`'s RBAC generator
only emits output from `+kubebuilder:rbac` markers on controller code, which
doesn't exist until Phase 2. Generating a placeholder now would be
scaffolding ahead of the code it's meant to describe.

## Verification performed

- `go build ./...`, `go vet ./...` — clean.
- `mingw32-make test` (`go test ./...`) — clean (no test files yet for a
  pure-types package, expected).
- `mingw32-make license-check` — clean; also spot-checked directly with
  `addlicense -check` since the Makefile target only scans `git ls-files`
  (files were untracked at check time).
- Real end-to-end verification against a Kind cluster (`kind create cluster
  --name korion-dev`):
  - `make install` applied both CRDs cleanly.
  - `kubectl apply -f config/samples/platformmap-superheros.yaml` and
    `platformagent-superheros.yaml` (with dummy Secrets for the referenced
    `secretRef`s) both succeeded.
  - `kubectl get platformmap/platformagent -o yaml` round-tripped the full
    spec correctly, confirming the OpenAPI schema accepts the real sample
    shape.
  - `make uninstall` and `kind delete cluster` cleaned up afterward.

## Environment/tooling issues found and fixed

Full detail in `decisions.md`; summarized here:

1. **`mingw32-make` mis-transcodes this repo's em-dash in `$(shell pwd)`** —
   silently installed `controller-gen`/`addlicense` into a stray mojibake
   sibling directory instead of the real `bin/`. Fixed by switching
   `Makefile`'s `LOCALBIN` to `$(CURDIR)`. Recovered binaries were moved
   into the real `bin/` and the stray directory was deleted.
2. **`controller-gen` v0.21 fatally errors on `...`-glob directories with no
   `.go` files directly in them** (e.g. `./api/...`, since `api/` itself has
   no files, only its `v1alpha1/` subpackage does). Fixed by introducing a
   `CONTROLLER_GEN_PATHS` Makefile variable pointing at explicit leaf package
   paths, used by both `generate` and `manifests`. Must be extended
   (semicolon-joined) as `internal/controller`, `internal/discovery`, and
   `internal/graph` come online in later phases.
3. **`go.mod`'s `go` directive bumped to `1.26.0`** — required by
   `k8s.io/api@v0.36.2` (confirmed by attempting to force it back to
   `1.25.4`, which fails the build with an explicit version-floor error).
   `.github/workflows/ci.yml`'s two `setup-go` pins were bumped from `1.22`
   to `1.26` to match.

## Not done in this phase (by design)

- `internal/controller/platformmap_controller.go` reconciler — Phase 2.
- `internal/controller/platformagent_controller.go` reconciler — Phase 12.
- `config/rbac/*.yaml` — follows from Phase 2's controller RBAC markers.
