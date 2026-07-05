# Korion UI

React + TypeScript + Vite frontend for Korion's topology dashboard. See
`CLAUDE.md` and `docs/PLAN.md` at the repo root for the full design and
phase plan.

```bash
npm install
npm run dev       # http://localhost:5173, Phase 4: renders against ui/src/api/fixtures
npm run build
npm test
```

`src/api/client.ts` currently reads a static mock fixture
(`src/api/fixtures/mockPlatformMap.ts`). Phase 5 swaps it for a real fetch
against the Go controller's read API without changing any callers.
