# Phase 3 Summary — Graph builder hardening + frozen contract

Status: complete. See `docs/PLAN.md` for the full phase spec and
`decisions.md` for the three design decisions made this phase and why.

This phase turns `internal/graph/builder.go` from Phase 2's minimal
dedupe-by-ID merge into a real, tested contract: the topology JSON shape is
now frozen behind a committed fixture, so Go discovery-engine work (Phase 6)
and frontend work (Phase 4) can both build against it independently without
drifting apart.

## What was built

- `internal/graph/types.go` — `Node` gains `BrandColor string
  \`json:"brandColor,omitempty"\``, documented as stamped only by `Merge`,
  never set by a `Discoverer` directly. Doc comments updated to state the
  shape is now frozen (was "provisional... Phase 3 freezes it").
- `internal/graph/colors.go` (new) — the single, fixed `brandColors` lookup
  table transcribed from `CLAUDE.md` rule #7, keyed by exact `Node.Type`
  strings (`k8s-deployment`, `k8s-service` populated now; ArgoCD/Istio/
  Kyverno/GitHub/Docker/Prometheus/Grafana/Loki entries pre-populated for
  Phase 6 to consume or correct once those engines' actual `Type` strings are
  finalized). `BrandColorFor(nodeType string) string` falls back to a neutral
  `defaultBrandColor` (`#6B7280`) for anything not yet in the table.
- `internal/graph/builder.go` — `Merge` hardened:
  - Every merged node gets `BrandColor` stamped from `BrandColorFor`.
  - Same-ID nodes across sources are now joined, not fully overwritten:
    `joinNodes` replaces scalar fields (`Type`/`Label`/`Status`) only when
    the later source provides a non-empty value, and shallow-merges
    `Metadata` maps key by key — a source contributing only partial detail
    (e.g. a future ArgoCD node enriching a K8s-discovered node with sync
    status) no longer erases fields an earlier source already set.
  - `dedupeEdges` collapses identical `From`/`To`/`Type`/`Label` edges
    reported by more than one source down to one.
- `internal/graph/builder_test.go` — extended with: BrandColor stamped for a
  known type and falling back to `defaultBrandColor` for an unknown one; a
  later source's partial metadata enriching rather than erasing an earlier
  source's fields; duplicate edges collapsing to one. Original Phase 2
  cases (empty input, single source, multi-source dedupe/later-wins,
  erroring source not blocking the rest) all still pass unchanged.
- `internal/graph/testdata/sample-topology.json` (new) — the frozen fixture:
  two nodes (`k8s-deployment`, `k8s-service`) with `brandColor` populated and
  one `routes-to` edge, matching a "catalog" service in the `superheros`
  namespace.
- `internal/graph/contract_test.go` (new) — `TestFrozenTopologyContract`
  builds an equivalent `Graph` from Go literals, runs it through `Merge`,
  and asserts the marshaled JSON is structurally identical (via
  unmarshal-to-`any` + `reflect.DeepEqual`, so key order is irrelevant) to
  the fixture.
- `internal/discovery/k8s_test.go` — added
  `TestK8sDiscoverer_MatchesFrozenTopologyContract`: runs the real
  `K8sDiscoverer.Discover` (fake clientset, a Deployment + Service shaped to
  match the fixture) through the same `graph.Merge`, and asserts its JSON
  matches the identical fixture file. This is the stronger of the two
  contract tests — it proves the actual Phase 2 discovery pipeline (not just
  hand-typed Go literals) produces the frozen shape.
- `api/v1alpha1/platformmap_types.go` — `PlatformMapStatus.Topology`'s doc
  comment updated to point at the now-frozen `internal/graph` shape and its
  fixture, instead of deferring to "a later phase." The field itself stays
  `*runtime.RawExtension` (unchanged) — freezing the JSON shape doesn't
  require the API package to import `internal/graph`, only that the shape
  within the opaque JSON is now stable and tested.
- `config/crd/bases/korion.io_platformmaps.yaml` — regenerated via
  `mingw32-make generate manifests`; only the `topology` field's
  `description` text changed (mirrors the doc-comment update above), no
  schema/behavior change.

## Verification performed

- `go build ./...`, `go vet ./...`, `gofmt -l -w` — clean (gofmt only
  re-aligned `colors.go`'s map literal).
- `go test ./...` — all packages green, including the two new contract
  tests and the extended `builder_test.go` table.
- `mingw32-make license-check` — clean (exit 0).
- `mingw32-make generate manifests` — regenerated deepcopy/CRD YAML; `git
  status` confirmed the only unexpected diff was the `topology` field's
  description text propagating from the Go doc-comment change, nothing else
  drifted.

## Design decisions made this phase (full rationale in `decisions.md`)

1. **BrandColor keyed by exact `Node.Type` string, not a prefix/family
   match.** A prefix scheme breaks down for multi-word families (e.g.
   `github-actions-workflow` vs `github-repository` both start with
   `github` but need different colors). An explicit table plus a safe
   fallback is simpler and never silently misassigns a color.
2. **`Merge` now joins same-ID nodes instead of fully overwriting them.**
   `docs/PLAN.md` describes `Merge` as "dedupes/joins," not just dedupes —
   once Phase 6 engines can describe an entity K8s discovery already
   produced a node for, partial enrichment (not full replacement) is the
   correct semantics. Exercised now via synthetic two-source test cases
   ahead of any real second source existing.
3. **The frozen fixture is verified two ways** — once against a hand-built
   `Graph` (`internal/graph/contract_test.go`) and once against the real
   `K8sDiscoverer` output (`internal/discovery/k8s_test.go`) — so the
   contract test proves the actual discovery pipeline matches the fixture,
   not just that the Go struct tags are internally consistent.

## Not done in this phase (by design)

- No frontend/TypeScript work — Phase 4 consumes
  `internal/graph/testdata/sample-topology.json` as its own fixture, hasn't
  started yet.
- No new discovery engines — the `brandColors` table's ArgoCD/Istio/
  Kyverno/GitHub/Docker/Prometheus/Grafana/Loki entries are anticipatory;
  Phase 6 must confirm or correct the exact `Node.Type` strings each engine
  actually emits.
- `PlatformMapStatus.Topology` stays an untyped `RawExtension` — no change
  to the CRD's Go-typed status shape, only to its documentation. This
  matches the existing Phase 1 rationale (API package shouldn't import
  `internal/graph`) and wasn't revisited.
