# React 17 Upgrade Checklist (Frontend)

Last updated: 2026-02-19 13:30 EST

Purpose: status tracker for the React 17 frontend upgrade. Completed items stay crossed out; this file tracks the remaining React 17 scope plus post-upgrade cleanup.

## Context

- Primary issue: https://github.com/kubeflow/pipelines/issues/11594
- Related context:
  - https://github.com/kubeflow/pipelines/issues/10098 (closed; Vite/Vitest migration completed)
  - https://github.com/kubeflow/pipelines/issues/5118

## Branch Scope

- `react-17-recharts-pr` merged into `master` as PR #12829 on 2026-02-18.
- `react-17-core-pr` is the remaining stacked branch and contains the final React 17 core/type bump.

## Completed Milestones

- [x] ~~PR #12754 (Vite/Vitest migration) merged~~
- [x] ~~PR #12756 (frontend server ESM + dev workflow) merged~~
- [x] ~~PR #12793 (MUI v4 baseline) merged~~
- [x] ~~PR #12829 (`react-vis`/`react-svg-line-chart` -> `recharts` + inline SVG) merged~~
- [x] ~~CRA/Craco-era tooling removed (`react-scripts`, `@craco/craco`)~~
- [x] ~~Enzyme and `jest-environment-jsdom-sixteen` removed from frontend deps~~
- [x] ~~Frontend server tests migrated from Jest to Vitest (`frontend/server/package.json` uses `vitest run`)~~
- [x] ~~Resolved `recharts` transitive peer mismatch via `overrides.recharts.react-redux=8.1.3`~~
- [x] ~~Stale Vite planning docs removed (`frontend/docs/vite-migration-plan.md`, `frontend/docs/vite-migration-review.md`)~~

## Current Baseline (this branch)

- React/core versions:
  - `react@^17.0.2`
  - `react-dom@^17.0.2`
  - `react-test-renderer@^17.0.2`
  - `@types/react@^17.0.0`
  - `@types/react-dom@^17.0.0`
  - `@types/react-test-renderer@^17.0.0`
- Build/test stack:
  - `vite@^7.3.1`, `vitest@^4.0.17`
  - `frontend/server` uses Vitest (`vitest run`)
- React-17 compatibility code updates in this branch:
  - `frontend/src/components/CustomTable.tsx`
  - `frontend/src/components/graph/SubDagLayer.tsx`

## Active Plan (remaining to complete React 17)

- [x] ~~Rebase `react-17-core-pr` on latest `master` after PR #12829 merge~~
- [x] ~~Run final verification on this branch and capture results~~:
  - `npm run test:ci`
  - `npm run build`

## Latest Verification Notes (2026-02-18)

- `npm run format:check` passed.
- `npm run lint` passed.
- `npm run typecheck` passed.
- `npm run test:ui:coverage:loop` (`--maxWorkers 4`) had 7 failures that were mostly 5s timeouts under local load.
- Serial rerun of the same failing suites passed:
  - `npx vitest run --maxWorkers 1 src/pages/NewExperiment.test.tsx src/pages/ExperimentDetails.test.tsx src/pages/NewRun.test.tsx src/pages/NewRunV2.test.tsx`
- `npm run test:server:coverage` passed.
- `npm run build` passed.

## Post-Upgrade Cleanup TODO (non-blocking)

- [ ] Reduce remaining `react-dom/test-utils` usage in UI tests:
  - `frontend/src/components/UploadPipelineDialog.test.tsx`
  - `frontend/src/components/SideNav.test.tsx`
  - `frontend/src/components/viewers/Tensorboard.test.tsx`
  - `frontend/src/components/viewers/VisualizationCreator.test.tsx`
  - `frontend/src/mlmd/LineageActionBar.test.tsx`
- [x] ~~Address remaining legacy lifecycle warning source (`re-resizable` via `SidePanel`)~~
- [ ] Remove `snapshot-diff` (replace with Vitest-native assertions)
- [ ] Decide whether to keep `eslint-config-react-app` or migrate to non-CRA ESLint base
- [ ] Recheck and resolve reproducible `TablePagination` DOM nesting warnings in UI tests

## Verification Commands

- `npm run test:ci`
- `npm run lint`
- `npm run test`
- `npm run build`
- `npm run test:server:coverage` (if server code/tests are touched)
- `npm run format:check` (if formatting touched)

## Notes

- `tasks/todo.md` is retired; use this checklist as the canonical plan.
