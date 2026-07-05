# Phase 4 Summary — Frontend skeleton against mock data

Status: complete. See `docs/PLAN.md` for the full phase spec and
`decisions.md` for the seven decisions/findings made this phase and why.

This phase builds the full mockup dashboard layout in the browser, wired to
static mock data rather than the live controller — proving the frontend
architecture (canvas, panels, filters, cross-panel state, data-fetching
interface) independently of the five remaining discovery engines (Phase 6).

## What was built

- `ui/` scaffolded via `npm create vite@latest -- --template react-ts`, then
  pinned to **React 18** (CLAUDE.md's stated version; the current Vite
  template defaults to React 19 — see `decisions.md`).
- **Tailwind v4** wired via `@tailwindcss/vite`, dark-only theme tokens
  (`--color-korion-bg`, `-cyan`, `-violet`, health-status colors) declared
  once in `ui/src/index.css`'s `@theme` block — no `tailwind.config.js`, no
  `dark:` variants anywhere, per CLAUDE.md's "Dark theme ONLY" rule.
- `ui/src/api/types.ts` — hand-written TS types (`GraphNode`, `GraphEdge`,
  `Graph`) mirroring `internal/graph/types.go` field-for-field, plus
  `DeploymentEvent`/`PolicyViolation`/`PolicySummary`/`PlatformMapView` types
  for the panels that aren't part of the frozen graph contract yet.
- `ui/src/api/colors.ts` — `brandColorFor`/`healthDotColorFor`, mirroring
  `internal/graph/colors.go`'s table (a second, independent copy that the
  frontend can use for mock data ahead of the Go builder always stamping
  `brandColor` server-side).
- `ui/src/api/nodeCategory.ts` — `categoryForNodeType`, grouping frozen
  `Node.type` strings into the Sidebar's filter categories (All/GitHub/
  Docker/ArgoCD/Kubernetes/Istio/Kyverno/Prometheus).
- Fixtures (`ui/src/api/fixtures/`):
  - `sample-topology.json` — literal copy of the Go-owned frozen fixture,
    guarded by `sample-topology.contract.test.ts` (reads the real
    `internal/graph/testdata/sample-topology.json` off disk and asserts deep
    equality, so the two can't drift).
  - `mockPlatformMap.ts` — a richer, hand-built 18-node SuperHeros dataset
    (GitHub → GitHub Actions → Docker Hub → ArgoCD → 6 K8s
    services/deployments including the catalog v1/v2/v3 canary trio → Istio
    traffic-split node → Kyverno node) plus mock `DeploymentEvent`/
    `PolicySummary` data, needed to demonstrate the *full* mockup layout —
    the two-node frozen fixture alone is too sparse for that.
- `ui/src/api/client.ts` — `getPlatformMap(namespace, name)` returns the mock
  fixture today; `usePlatformMap.ts` wraps it in a TanStack Query
  `useQuery` with a 30s `refetchInterval` matching the controller's default
  `spec.refreshInterval`. Swapping to a real `fetch` in Phase 5 changes only
  `client.ts`'s implementation, not any caller.
- `ui/src/state/useUIStore.ts` — a small zustand store (`selectedNodeId`,
  `activeFilter`) for cross-panel UI state, per `docs/PLAN.md`'s "not Redux"
  guidance.
- Components (`ui/src/components/`):
  - `TopologyCanvas/` — `@xyflow/react` canvas; `layout.ts` computes a
    BFS-depth-as-column layered layout (no `dagre` dependency yet); filters
    nodes/edges by `useUIStore`'s `activeFilter`; clicking a node/pane updates
    `selectedNodeId`.
  - `NodeTypes/` — one `ToolNode` component used for every tool type (border
    color from `node.brandColor`, a `HealthDot` colored by `node.status`),
    keyed by data rather than one component per type.
  - `Sidebar/` — filter buttons + health-status legend.
  - `ServiceDetails/` — right panel, Overview/Metrics/Logs/Events tabs;
    Overview renders the selected node's metadata; the other three are
    explicit placeholders pointing at the phase that will populate them
    (Prometheus in Phase 6, Loki/ARIA in Phase 9).
  - `DeploymentTimeline/` and `PolicyPanel/` — bottom-left event stream and
    bottom-right Kyverno pass/warn/fail summary + recent violations, both
    driven by `mockPlatformMap`'s mock data.
  - `App.tsx`/`main.tsx` — assembles the full layout (Sidebar | canvas +
    ServiceDetails | DeploymentTimeline + PolicyPanel) and wraps it in a
    `QueryClientProvider`.
- Tests: `NodeTypes.test.tsx` (border color from `brandColor`, fallback to
  the neutral default, health-dot status), `Sidebar.test.tsx` (default "All"
  filter, clicking a filter updates the shared store and `aria-pressed`
  state), `sample-topology.contract.test.ts` (frontend/Go fixture parity).
- `ui/src/setupTests.ts` — jest-dom matchers plus a `ResizeObserver` stub
  jsdom doesn't provide, which `@xyflow/react`'s `<ReactFlow>` needs (see
  `decisions.md`).

## Verification performed

- `npm run build` (`tsc -b && vite build`) — clean, no type errors.
- `npm test` (`vitest run`) — 7/7 tests passing across 3 files.
- `mingw32-make license-check` — clean after `git add ui` +
  `mingw32-make license-fix` (see `decisions.md` for why staging first was
  necessary — `git ls-files`-based tooling is blind to untracked files).
- **Manual visual verification** against
  `ref-docs/file_000000006c407208be1de7ae77fa6da5.png`: started `npm run dev`,
  drove headless local Chrome via a scratch `puppeteer-core` script (no
  `chromium-cli`/Playwright available on this Windows host) to screenshot
  three states — default view (18 nodes, sidebar, timeline, policy panel all
  rendering with correct brand colors and health dots), a node selected
  (ArgoCD node highlighted, ServiceDetails populated with its metadata), and
  the "Istio" sidebar filter applied (canvas correctly narrows to only the
  `istio-virtualservice` node and its edges disappear). No console/page
  errors in any state. This pass caught and fixed one real issue: React
  Flow's `<MiniMap>` rendered as a large blank panel overlapping nodes and
  isn't in the mockup — removed.

## Design decisions/findings this phase (full detail in `decisions.md`)

1. Pinned React 18 over Vite's current React 19 default, per CLAUDE.md.
2. Tailwind v4 CSS-first `@theme` config instead of `tailwind.config.js`.
3. BFS-layered canvas layout instead of adding a `dagre`/`elkjs` dependency.
4. Dropped React Flow's `<MiniMap>` — not in the mockup, visually broken.
5. Two-fixture strategy: frozen Go fixture (contract-tested) kept separate
   from a richer hand-built mock dataset (full-layout demo).
6. `license-check`/`license-fix` silently no-op on untracked files — stage
   new files first. Relevant again for Phase 6 and Phase 9's new file
   batches.
7. jsdom needs a `ResizeObserver` polyfill for any test mounting `<ReactFlow>`.

## Not done in this phase (by design)

- No live controller wiring — `client.ts` reads the mock fixture only;
  Phase 5 swaps in the real `fetch` against the Phase 2 read API.
- No loading/error/"CRD not yet reconciled" UI states beyond the generic
  `isLoading`/`isError` branches already in `App.tsx` — Phase 5 exercises
  these against a real API instead of TanStack Query's synthetic states.
- Metrics/Logs/Events tabs in `ServiceDetails` are explicit placeholders —
  populated by Phase 6 (Prometheus) and Phase 9 (Loki/ARIA) respectively.
- No Storybook/Playwright E2E, per `docs/PLAN.md`'s explicit out-of-scope
  list.
- The "Runtime Service Map" traffic panel from one mockup image — deferred
  per `docs/PLAN.md`, not part of v0.1.
