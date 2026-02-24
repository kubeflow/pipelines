# GitHub Issues: React 18/19 Upgrade & Testing Modernization

**Tracking Issue**: React 18/19 Frontend Upgrade — Modernize Kubeflow Pipelines UI
**Repository**: [kubeflow/pipelines](https://github.com/kubeflow/pipelines)

**Strategy**: Deps-first, bump-last. All dependencies are migrated to React 18-compatible versions on the stable React 17 runtime before bumping React. This plan consolidates [jeffspahr's upgrade plan](https://github.com/jeffspahr/jeffspahr-pipelines/blob/e02da33/frontend/docs/react-18-19-upgrade-plan.md) with detailed migration mechanics and quality guardrails contributed by the community.

**How to contribute**: Each issue below maps to one independently mergeable PR. Issues #3–#6 can be worked on in parallel by different contributors. Pick an unassigned issue, comment to claim it, and open a PR following the [contribution guide](https://github.com/kubeflow/pipelines/blob/master/frontend/CONTRIBUTING.md). Every PR must pass `npm run test:ci` and `npm run build` before merge.

---

## 1. Prereq warning/test cleanup

**Labels**: `area/frontend`, `priority/p0`, `kind/cleanup`
**Assignee**: @jeffspahr
**Depends on**: Nothing

**Description**:
Land cleanup PRs ([#12855](https://github.com/kubeflow/pipelines/pull/12855), [#12856](https://github.com/kubeflow/pipelines/pull/12856), [#12858](https://github.com/kubeflow/pipelines/pull/12858), [#12872](https://github.com/kubeflow/pipelines/pull/12872)) that remove `react-dom/test-utils` imports, `snapshot-diff`, upgrade `re-resizable`, add DOM nesting warning coverage, and remove dead dependencies. Goal: reduce warning/test noise before dependency migrations begin.

---

## 2. Add React peer compatibility gate

**Labels**: `area/frontend`, `priority/p0`, `kind/infra`
**Assignee**: @jeffspahr
**Depends on**: #1

**Description**:
Add `check-react-peers.mjs` script ([#12881](https://github.com/kubeflow/pipelines/pull/12881)) and wire it into `npm run test:ci`. Introduces `npm run check:react-peers` (target 17), `check:react-peers:18`, and `check:react-peers:19`. The gate enforces that all dependencies declare peer ranges supporting the target React version. An allowlist tracks known exceptions with documented justifications.

This CI guardrail ensures no new dependency can silently break React version compatibility.

---

## 3. Upgrade Storybook 6 to 7

**Labels**: `area/frontend`, `priority/p1`, `kind/chore`, `good first issue`
**Depends on**: #2

**Description**:
Upgrade Storybook from 6.3.x to 7.x while still running on React 17. Replace the Webpack builder with the Vite builder (`@storybook/builder-vite`, `@storybook/react-vite`) to align with the project's existing build tooling. Remove transitive React 17-incompatible deps (`@reach/router`, `create-react-context`). Remove corresponding peer gate allowlist entries. Run automigration (`npx storybook upgrade`), update `.storybook/main.js` and `.storybook/preview.js`, migrate 8 story files to CSF 3 if needed.

**Acceptance Criteria**:
- [ ] `npm run storybook` renders all 8 stories without errors
- [ ] `npm run build:storybook` succeeds
- [ ] `npm run test:ci` passes
- [ ] Peer gate allowlist entries for Storybook removed

---

## 4. Migrate react-query to @tanstack/react-query

**Labels**: `area/frontend`, `priority/p1`, `kind/chore`
**Depends on**: #2

**Description**:
Replace `react-query` v3 with `@tanstack/react-query` v5. Update all query hooks (`useQuery`, `useMutation`), `QueryClientProvider` setup, cache configuration, and test wrappers to the new API. Remove peer gate allowlist entry for `react-query`.

**Acceptance Criteria**:
- [ ] Data fetching, polling, and cache behavior work identically to pre-migration
- [ ] `npm run check:react-peers` passes without `react-query` in allowlist
- [ ] `npm run test:ci` passes

---

## 5. Migrate react-flow-renderer to @xyflow/react

**Labels**: `area/frontend`, `priority/p1`, `kind/chore`
**Depends on**: #2

**Description**:
Replace deprecated `react-flow-renderer` v9 with `@xyflow/react`. Update DAG visualization components, node/edge type definitions, event handlers, and related stories/tests. Remove peer gate allowlist entry.

**Acceptance Criteria**:
- [ ] Pipeline graph renders correctly with drag, zoom, and pan interactions
- [ ] `npm run check:react-peers` passes without `react-flow-renderer` in allowlist
- [ ] `npm run test:ci` passes

---

## 6. Migrate Material-UI v4 to MUI v5

**Labels**: `area/frontend`, `priority/p1`, `kind/chore`
**Depends on**: #2

**Description**:
Migrate `@material-ui/core` and `@material-ui/icons` to `@mui/material` and `@mui/icons-material` v5 with Emotion (`@emotion/react`, `@emotion/styled`) as the styling engine.

Run MUI codemods in order:
1. `npx @mui/codemod v5.0.0/preset-safe frontend/src`
2. `npx @mui/codemod v5.0.0/variant-prop frontend/src`
3. `npx @mui/codemod v5.0.0/top-level-imports frontend/src`

Manually migrate theme files (`src/Css.tsx`, `src/mlmd/Css.tsx`): convert `overrides` to `components.styleOverrides` format. Migrate `withStyles` HOC to `styled` in `src/atoms/CardTooltip.tsx`. Verify codemods did NOT touch generated directories (`src/apis/`, `src/apisv2beta1/`, `src/third_party/`). Run `npm run format` after codemods. Capture visual regression screenshots before/after.

**Impact**: 64 files import `@material-ui/core`, 25 import `@material-ui/icons`, 2 theme files, 1 `withStyles` file.

**Acceptance Criteria**:
- [ ] Visual parity on all pages (Pipeline List, Run Details, Experiment List, Side Nav)
- [ ] `npm run test:ci && npm run build` pass
- [ ] No codemod changes in generated directories
- [ ] Theme colors, typography, and spacing unchanged

---

## 7. Modernize JSX runtime and test utilities

**Labels**: `area/frontend`, `priority/p2`, `kind/chore`
**Depends on**: #5, #6

**Description**:
Switch to modern JSX transform (`react-jsx` in tsconfig). Upgrade `@testing-library/react` from v11 to v14 and `@testing-library/user-event` from v13 to v14. Remove `react-test-renderer` and `@types/react-test-renderer`.

Migrate 5 files from `react-test-renderer` to `render()` + `asFragment()`:
- `src/atoms/BusyButton.test.tsx`
- `src/atoms/Hr.test.tsx`
- `src/atoms/IconWithTooltip.test.tsx`
- `src/components/viewers/ROCCurve.test.tsx`
- `src/components/viewers/VisualizationCreator.test.tsx`

Migrate remaining `act` imports from `react-dom/test-utils` to `@testing-library/react` in 5 files:
- `src/mlmd/LineageActionBar.test.tsx`
- `src/components/viewers/VisualizationCreator.test.tsx`
- `src/components/viewers/Tensorboard.test.tsx`
- `src/components/UploadPipelineDialog.test.tsx`
- `src/components/SideNav.test.tsx`

Capture coverage baseline before and compare after -- coverage must not decrease.

**Acceptance Criteria**:
- [ ] `npm run test:ci` passes with zero deprecation warnings
- [ ] No file imports `React` solely for JSX (unused import)
- [ ] Line/branch coverage not decreased from baseline

---

## 8. Update remaining ecosystem dependencies

**Labels**: `area/frontend`, `priority/p2`, `kind/chore`, `good first issue`
**Depends on**: #7

**Description**:
Upgrade remaining deps with React 17-limited peer ranges: `markdown-to-jsx` v6 to v7, `react-dropzone` v5 to v14, any `re-resizable` residuals, `snapshot-diff` update. Review and update `package.json` `overrides` and `resolutions` entries -- remove stale ones, add new ones as needed. Drive peer gate allowlist to empty.

**Acceptance Criteria**:
- [ ] `npm run check:react-peers` passes with empty allowlist
- [ ] `npm run test:ci && npm run build` pass
- [ ] Markdown rendering and file upload work identically to pre-migration

---

## 9. Upgrade React to v18

**Labels**: `area/frontend`, `priority/p1`, `kind/feature`
**Depends on**: #8

**Description**:
Bump `react` and `react-dom` to ^18.2.0. Bump `@types/react` and `@types/react-dom` to ^18. Migrate `ReactDOM.render()` to `createRoot()` in `src/index.tsx`. Fix TypeScript errors from `@types/react@18`. Flip peer gate to `check:react-peers:18`. Regenerate all snapshots (`npm test -- -u`) and review every diff (~55 files, ~280 assertions) for correctness. Capture visual regression screenshots.

**Acceptance Criteria**:
- [ ] `npm run test:ci && npm run build` pass
- [ ] `npm run check:react-peers:18` passes
- [ ] Manual smoke test of all primary pages -- zero console errors
- [ ] Every regenerated snapshot diff reviewed for correctness

---

## 10. Stabilize React 18 runtime

**Labels**: `area/frontend`, `priority/p1`, `kind/bug`
**Depends on**: #9

**Description**:
Resolve any regressions or flaky tests introduced by the React 18 runtime. Address automatic batching behavior changes if any components depend on intermediate render states. Compare bundle size against pre-upgrade baseline (< 5% increase). Smoke test both single-user and multi-user deployment modes.

**Acceptance Criteria**:
- [ ] `npm run test:ci` stable with zero flaky tests
- [ ] Visual regression clean
- [ ] Bundle size within 5% of baseline
- [ ] Both single-user and multi-user modes functional

---

## 11. React 18.3 deprecation checkpoint

**Labels**: `area/frontend`, `priority/p2`, `kind/chore`
**Depends on**: #10

**Description**:
Bump `react` and `react-dom` to ^18.3.0. Run `npm run test:ci` and review test output for React 19 deprecation warnings. Document each warning found. Address any deprecation warnings before proceeding to React 19. This is a low-cost step that surfaces React 19 issues early.

**Acceptance Criteria**:
- [ ] All React 19 deprecation warnings documented
- [ ] Warnings addressed or tracked for resolution in subsequent issues
- [ ] `npm run test:ci` passes

---

## 12. Dependency sweep for React 19

**Labels**: `area/frontend`, `priority/p2`, `kind/chore`
**Depends on**: #11

**Description**:
Run `npm run check:react-peers:19` and upgrade any remaining deps that declare peer ranges excluding React 19. Drive the peer gate to green for React 19.

**Acceptance Criteria**:
- [ ] `npm run check:react-peers:19` passes
- [ ] `npm run test:ci` passes

---

## 13. Upgrade React to v19

**Labels**: `area/frontend`, `priority/p1`, `kind/feature`
**Depends on**: #12

**Description**:
Bump `react` and `react-dom` to ^19. Bump `@types/react` and `@types/react-dom` to ^19. Migrate 4 `forwardRef` test mocks to ref-as-prop pattern:
- `src/pages/RecurringRunList.test.tsx` (line 38)
- `src/pages/ArchivedRuns.test.tsx` (line 31)
- `src/pages/ArchivedExperiments.test.tsx` (line 30)
- `src/pages/AllRunsList.test.tsx` (line 31)

Fix TypeScript errors from `@types/react@19`. Flip peer gate to `check:react-peers:19`. Regenerate and review all snapshots.

**Acceptance Criteria**:
- [ ] `npm run test:ci && npm run build` pass
- [ ] `npm install` has zero peer dependency warnings
- [ ] Manual smoke test with zero console warnings or deprecation messages
- [ ] `npm run check:react-peers:19` passes

---

## 14. Enable StrictMode in dev/test

**Labels**: `area/frontend`, `priority/p3`, `kind/chore`
**Depends on**: #13

**Description**:
Enable `<StrictMode>` in the development and test rendering trees. Identify and fix double-invoke side effects. Production behavior remains unchanged -- strict mode is NOT active in production builds. This is deferrable without blocking the React 19 milestone if it surfaces a large number of issues.

**Acceptance Criteria**:
- [ ] `npm run test:ci` passes with strict mode enabled
- [ ] Dev server (`npm start`) runs without strict-mode-related warnings
- [ ] Production builds do NOT include strict mode

---

## 15. Update documentation for React 19 stack

**Labels**: `area/frontend`, `priority/p2`, `kind/documentation`, `good first issue`
**Depends on**: #13 (or included inline in prior PRs)

**Description**:
Update `AGENTS.md` "Frontend development" section with React 19, MUI v5, @testing-library/react v14, Storybook 7, @tanstack/react-query. Update `frontend/CONTRIBUTING.md` testing FAQ -- remove references to Enzyme, `react-test-renderer`, React 16-specific patterns. Update `frontend/README.md` stack description. Archive or update `frontend/docs/react-17-upgrade-checklist.md`.

**Acceptance Criteria**:
- [ ] All referenced versions match actual installed versions in `package.json`
- [ ] No references to Enzyme, `react-test-renderer`, or React 16-specific patterns remain
- [ ] New contributor can follow `CONTRIBUTING.md` without encountering outdated instructions

---

## Dependency Graph

```
#1 Prereq Cleanup (@jeffspahr)
 └── #2 Peer Gate (@jeffspahr)
      ├── #3 Storybook 7 ──────────┐
      ├── #4 react-query ──────────┤  (parallelizable)
      ├── #5 react-flow ───────────┤
      └── #6 MUI v5 ──────────────┘
                                    │
                              #7 JSX + Tests
                                    │
                              #8 Ecosystem Deps
                                    │
                              #9 React 18 Core
                                    │
                              #10 React 18 Stabilization
                                    │
                              #11 React 18.3 Checkpoint
                                    │
                              #12 Dep Sweep for 19
                                    │
                              #13 React 19 Core
                                    │
                              #14 Strict Mode
                                    │
                              #15 Documentation
```

**Parallelizable**: Issues #3, #4, #5, #6 can be worked on in parallel by different contributors -- they touch different files with no mutual dependencies. All four must be merged before #7 can begin.

**Good first issues**: #3 (Storybook upgrade -- `npx storybook upgrade` does most of the work), #8 (ecosystem deps -- small, focused package bumps), #15 (documentation -- no code logic changes).
