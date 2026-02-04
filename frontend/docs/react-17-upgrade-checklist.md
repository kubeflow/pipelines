# React 17 Upgrade Checklist (Frontend)

Last updated: 2026-02-15 19:41 EST

Purpose: status tracker for the React 17 frontend upgrade. Completed items stay crossed out; this file tracks only remaining actionable cleanup.

## Context
- Primary issue: https://github.com/kubeflow/pipelines/issues/11594
- Related context:
  - https://github.com/kubeflow/pipelines/issues/10098 (closed; Vite/Vitest migration completed)
  - https://github.com/kubeflow/pipelines/issues/5118

## Branch Scope Note
- This file is shared across the stacked React 17 branches.
- `react-17-recharts-pr` is the lower stack branch and intentionally keeps React 16.
- The React core/version bump is applied in the stacked `react-17-core-pr` branch.

## Completed Milestones
- [x] ~~PR #12754 (Vite/Vitest migration) merged~~
- [x] ~~PR #12756 (frontend server ESM + dev workflow) merged~~
- [x] ~~MUI v4 baseline present (`@material-ui/core@^4.12.4`, `@material-ui/icons@^4.11.3`)~~
- [ ] React core upgraded to 17.x (`react`, `react-dom`, `react-test-renderer`)
- [ ] React type packages aligned to 17.x (`@types/react`, `@types/react-dom`, `@types/react-test-renderer`)
- [x] ~~UI tests migrated to Vitest + Testing Library~~
- [x] ~~Frontend server tests migrated from Jest to Vitest (`frontend/server/package.json` uses `vitest run`)~~
- [x] ~~`react-vis`/`react-svg-line-chart` replaced with `recharts` + inline SVG~~
- [x] ~~ROC tooltip nearest-point behavior restored for mismatched series x-values (with regression test)~~
- [x] ~~ROC chart data preprocessing memoized by `configs` identity to avoid recompute on zoom/hover rerenders~~
- [x] ~~CRA/Craco-era tooling removed (`react-scripts`, `@craco/craco`)~~
- [x] ~~Enzyme and `jest-environment-jsdom-sixteen` removed from frontend deps~~

## Current Baseline (verified from branch)
- React/core versions:
  - `react@^16.12.0`
  - `react-dom@^16.12.0`
  - `react-test-renderer@^16.5.2`
  - `@types/react@^16.9.22`
  - `@types/react-dom@^16.9.5`
  - `@types/react-test-renderer@^16.0.2`
- Build/test stack:
  - `vite@^7.3.1`, `vitest@^4.0.17`
  - `frontend test:ci` runs format/lint/typecheck/UI coverage/server coverage
  - `frontend/server` uses Vitest (`vitest run`)
- Remaining notable dependencies:
  - `re-resizable@^4.11.0` (legacy lifecycle warnings risk)
  - `eslint-config-react-app@^7.0.1`
  - `snapshot-diff@^0.6.1` (Jest-era utility)

## Active Plan (remaining work)
- [ ] Apply stacked React 17 core bump branch (`react-17-core-pr`) to move this stack to React 17:
  - `react`, `react-dom`, `react-test-renderer`
  - `@types/react`, `@types/react-dom`, `@types/react-test-renderer`
- [x] ~~Resolved `recharts` transitive peer mismatch on this branch~~:
  - nested dep now pinned via override to `recharts -> react-redux@8.1.3` in lockfile
  - still requires full CI-equivalent verification run before merge
- [ ] Reduce remaining `react-dom/test-utils` usage in UI tests (currently 5 files):
  - `frontend/src/components/UploadPipelineDialog.test.tsx`
  - `frontend/src/components/SideNav.test.tsx`
  - `frontend/src/components/viewers/Tensorboard.test.tsx`
  - `frontend/src/components/viewers/VisualizationCreator.test.tsx`
  - `frontend/src/mlmd/LineageActionBar.test.tsx`
- [ ] Address remaining React 16-era warning sources:
  - `re-resizable` legacy lifecycle warnings used by `SidePanel`
  - any remaining `React.createFactory` warnings seen during full test runs
- [ ] Remove `snapshot-diff` (replace with Vitest-native assertions where still needed)
- [ ] Decide whether to keep `eslint-config-react-app` or move to a non-CRA ESLint base
- [ ] Recheck and resolve any reproducible `TablePagination` DOM nesting warnings in UI tests
- [ ] Run full verification and record results in PR description:
  - `npm run test:ci`
  - `npm run build`

## Verification Commands
- `npm run test:ci`
- `npm run lint`
- `npm run test`
- `npm run build`
- `npm run test:server:coverage` (if server code/tests are touched)
- `npm run format:check` (if formatting touched)

## Notes
- `tasks/todo.md` is retired; use this checklist as the canonical plan.
- `frontend/docs/vite-migration-plan.md` and `frontend/docs/vite-migration-review.md` were intentionally removed as stale.
- ROC chart-data memoization currently keys off `configs` reference identity; if upstream mutates configs in place, cache invalidation would require follow-up.
