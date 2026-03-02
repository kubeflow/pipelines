# GitHub Issues: React 18/19 Upgrade & Testing Modernization

**Tracking Issue**: React 18/19 Frontend Upgrade — Modernize Kubeflow Pipelines UI
**Repository**: [kubeflow/pipelines](https://github.com/kubeflow/pipelines)

**Strategy**: Deps-first, bump-last. All dependencies are migrated to React 18-compatible versions on the stable React 17 runtime before bumping React.
**Canonical source**: This checklist is the canonical execution plan and supersedes earlier draft planning notes.

**How to contribute**: Each issue below maps to one independently mergeable PR. As of March 2, 2026, #1, #2, #3, and #6 are complete. The remaining parallelizable dependency migrations before the React bump are #4 and #5 ([#12892](https://github.com/kubeflow/pipelines/issues/12892) and [#12893](https://github.com/kubeflow/pipelines/issues/12893)), and both already have active PRs: [#12946](https://github.com/kubeflow/pipelines/pull/12946) for #4 and [#12945](https://github.com/kubeflow/pipelines/pull/12945) for #5. Coordinate on those PRs before opening duplicate work. Every PR must pass `npm run test:ci` and `npm run build` before merge.

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

## 3. ~~Upgrade Storybook 6 to 7~~ Completed via Storybook 10 ([#12891](https://github.com/kubeflow/pipelines/issues/12891), [#12940](https://github.com/kubeflow/pipelines/pull/12940))

**Labels**: `area/frontend`, `priority/p1`, `kind/chore`, `good first issue`
**Depends on**: #2

**Status**:
Completed by [#12940](https://github.com/kubeflow/pipelines/pull/12940). The original Storybook 7 target was superseded by a direct upgrade from Storybook 6 to Storybook 10 using the Vite builder, which clears the same React-compatibility blockers with fewer intermediate steps.

**Description**:
Upgrade Storybook from 6.3.x to a React-17-compatible modern release while staying on the React 17 runtime. Replace the legacy builder setup with the Vite builder (`@storybook/react-vite`), remove transitive React-incompatible deps (`@reach/router`, `create-react-context`), remove the corresponding peer gate allowlist entries, and migrate story config/files as needed.

**Acceptance Criteria**:
- [x] `npm run storybook` renders the migrated stories without errors
- [x] `npm run build:storybook` succeeds
- [x] `npm run test:ci` passes
- [x] Peer gate allowlist entries for Storybook removed

---

## 4. Migrate react-query to @tanstack/react-query ([#12892](https://github.com/kubeflow/pipelines/issues/12892))

**Labels**: `area/frontend`, `priority/p1`, `kind/chore`
**Depends on**: #2

**Status**:
In progress via [#12946](https://github.com/kubeflow/pipelines/pull/12946). Because the repo is still intentionally staged on React 17 before the React 18 bump, this step should use the latest TanStack React Query version compatible with React 17 (currently v4), not v5.

**Description**:
Replace `react-query` v3 with `@tanstack/react-query` while staying on the React 17 runtime. Update all query hooks (`useQuery`, `useMutation`), `QueryClientProvider` setup, cache configuration, and test wrappers to the TanStack API. Remove `react-query` as a React 18/19 peer-gate blocker.

**Acceptance Criteria**:
- [ ] Data fetching, polling, and cache behavior work identically to pre-migration
- [ ] `npm run check:react-peers:18` no longer reports `react-query` as a direct blocker
- [ ] `npm run test:ci` passes

---

## 5. Migrate react-flow-renderer to @xyflow/react ([#12893](https://github.com/kubeflow/pipelines/issues/12893))

**Labels**: `area/frontend`, `priority/p1`, `kind/chore`
**Depends on**: #2

**Status**:
In progress via [#12945](https://github.com/kubeflow/pipelines/pull/12945).

**Description**:
Replace deprecated `react-flow-renderer` v9 with `@xyflow/react`. Update DAG visualization components, node/edge type definitions, event handlers, and related stories/tests. Remove peer gate allowlist entry.

**Acceptance Criteria**:
- [ ] Pipeline graph renders correctly with drag, zoom, and pan interactions
- [ ] `npm run check:react-peers:18` no longer reports `react-flow-renderer` as a direct blocker
- [ ] `npm run test:ci` passes

---

## 6. Migrate Material-UI v4 to MUI v5 ([#12894](https://github.com/kubeflow/pipelines/issues/12894))

**Labels**: `area/frontend`, `priority/p1`, `kind/chore`
**Depends on**: #2

**Status**:
Completed by [#12925](https://github.com/kubeflow/pipelines/pull/12925); issue [#12894](https://github.com/kubeflow/pipelines/issues/12894) was closed on March 2, 2026. Exact pixel parity was intentionally not required; visual smoke comparisons found only minor button/icon rendering differences from MUI v5 internals, and those new visuals were accepted to avoid migration-specific styling hacks.

**Description**:
Migrate `@material-ui/core` and `@material-ui/icons` to `@mui/material` and `@mui/icons-material` v5 with Emotion (`@emotion/react`, `@emotion/styled`) as the styling engine.

Run MUI codemods in order:
1. `npx @mui/codemod v5.0.0/preset-safe frontend/src`
2. `npx @mui/codemod v5.0.0/variant-prop frontend/src`
3. `npx @mui/codemod v5.0.0/top-level-imports frontend/src`

Manually migrate theme files (`src/Css.tsx`, `src/mlmd/Css.tsx`): convert `overrides` to `components.styleOverrides` format. Migrate `withStyles` HOC to `styled` in `src/atoms/CardTooltip.tsx`. Verify codemods did NOT touch generated directories (`src/apis/`, `src/apisv2beta1/`, `src/third_party/`). Run `npm run format` after codemods. Capture visual regression screenshots before/after.

**Impact**: 64 files import `@material-ui/core`, 25 import `@material-ui/icons`, 2 theme files, 1 `withStyles` file.

**Acceptance Criteria**:
- [x] ~~Visual parity on all pages (Pipeline List, Run Details, Experiment List, Side Nav)~~ Visual smoke comparison completed on all primary pages; only minor cosmetic MUI v5 rendering deltas were observed and intentionally accepted
- [x] `npm run test:ci && npm run build` pass
- [x] No codemod changes in generated directories
- [x] ~~Theme colors, typography, and spacing unchanged~~ Theme styling is preserved aside from the accepted MUI v5 cosmetic rendering differences

---

## 7. Modernize JSX runtime and test utilities ([#12895](https://github.com/kubeflow/pipelines/issues/12895))

**Labels**: `area/frontend`, `priority/p2`, `kind/chore`
**Depends on**: #3 ([#12891](https://github.com/kubeflow/pipelines/issues/12891)), #4 ([#12892](https://github.com/kubeflow/pipelines/issues/12892)), #5 ([#12893](https://github.com/kubeflow/pipelines/issues/12893)), #6 ([#12894](https://github.com/kubeflow/pipelines/issues/12894))

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

## 8. Update remaining ecosystem dependencies ([#12896](https://github.com/kubeflow/pipelines/issues/12896))

**Labels**: `area/frontend`, `priority/p2`, `kind/chore`, `good first issue`
**Depends on**: #7 ([#12895](https://github.com/kubeflow/pipelines/issues/12895))

**Description**:
Upgrade remaining deps with React 17-limited peer ranges: `markdown-to-jsx` v6 to v7, `react-dropzone` v5 to v14, any `re-resizable` residuals, `snapshot-diff` update. Review and update `package.json` `overrides` and `resolutions` entries -- remove stale ones, add new ones as needed. Drive peer gate allowlist to empty.

**Current peer-gate blockers for React 18 to clear across #4, #5, #7, #8, and #9 (as of March 2, 2026):**
- `react-dom@17.0.2` (`react=17.0.2`) - handled in #9 ([#12897](https://github.com/kubeflow/pipelines/issues/12897))
- `react-flow-renderer@9.6.5` (`react-dom=16 || 17`, `react=16 || 17`) - handled in #5 ([#12893](https://github.com/kubeflow/pipelines/issues/12893))
- `react-query@3.16.0` (`react=^16.8.0 || ^17.0.0`) - handled in #4 ([#12892](https://github.com/kubeflow/pipelines/issues/12892))
- `react-test-renderer@17.0.2` (`react=17.0.2`) - handled in #7 ([#12895](https://github.com/kubeflow/pipelines/issues/12895))
- `react-textarea-autosize@8.3.3` (`react=^16.8.0 || ^17.0.0`) - handled in #8
- Current transitive blockers still reported by `npm run check:react-peers:18`: `react-redux@7.2.4`, `use-composed-ref@1.1.0`, `use-isomorphic-layout-effect@1.1.1`, `use-latest@1.2.0`

**Acceptance Criteria**:
- [ ] `npm run check:react-peers` passes with empty allowlist
- [ ] `npm run check:react-peers:18` direct blockers are reduced to only planned React core bump items
- [ ] `npm run test:ci && npm run build` pass
- [ ] Markdown rendering and file upload work identically to pre-migration

---

## 9. Upgrade React to v18 ([#12897](https://github.com/kubeflow/pipelines/issues/12897))

**Labels**: `area/frontend`, `priority/p1`, `kind/feature`
**Depends on**: #8 ([#12896](https://github.com/kubeflow/pipelines/issues/12896))

**Description**:
Bump `react` and `react-dom` to ^18.2.0. Bump `@types/react` and `@types/react-dom` to ^18. Migrate `ReactDOM.render()` to `createRoot()` in `src/index.tsx`. Fix TypeScript errors from `@types/react@18`. Flip peer gate to `check:react-peers:18`. Regenerate all snapshots (`npm test -- -u`) and review every diff (~55 files, ~280 assertions) for correctness. Capture visual regression screenshots.

**Acceptance Criteria**:
- [ ] `npm run test:ci && npm run build` pass
- [ ] `npm run check:react-peers:18` passes
- [ ] Manual smoke test of all primary pages -- zero console errors
- [ ] Every regenerated snapshot diff reviewed for correctness

---

## 10. Stabilize React 18 runtime ([#12898](https://github.com/kubeflow/pipelines/issues/12898))

**Labels**: `area/frontend`, `priority/p1`, `kind/bug`
**Depends on**: #9 ([#12897](https://github.com/kubeflow/pipelines/issues/12897))

**Description**:
Resolve any regressions or flaky tests introduced by the React 18 runtime. Address automatic batching behavior changes if any components depend on intermediate render states. Compare bundle size against pre-upgrade baseline (< 5% increase). Smoke test both single-user and multi-user deployment modes.

**Acceptance Criteria**:
- [ ] `npm run test:ci` stable with zero flaky tests
- [ ] Visual regression clean
- [ ] Bundle size within 5% of baseline
- [ ] Both single-user and multi-user modes functional

---

## 11. React 18.3 deprecation checkpoint ([#12899](https://github.com/kubeflow/pipelines/issues/12899))

**Labels**: `area/frontend`, `priority/p2`, `kind/chore`
**Depends on**: #10 ([#12898](https://github.com/kubeflow/pipelines/issues/12898))

**Description**:
Bump `react` and `react-dom` to ^18.3.0. Run `npm run test:ci` and review test output for React 19 deprecation warnings. Document each warning found. Address any deprecation warnings before proceeding to React 19. This is a low-cost step that surfaces React 19 issues early.

**Acceptance Criteria**:
- [ ] All React 19 deprecation warnings documented
- [ ] Warnings addressed or tracked for resolution in subsequent issues
- [ ] `npm run test:ci` passes

---

## 12. Dependency sweep for React 19 ([#12900](https://github.com/kubeflow/pipelines/issues/12900))

**Labels**: `area/frontend`, `priority/p2`, `kind/chore`
**Depends on**: #11 ([#12899](https://github.com/kubeflow/pipelines/issues/12899))

**Description**:
Run `npm run check:react-peers:19` and upgrade any remaining deps that declare peer ranges excluding React 19. Drive the peer gate to green for React 19.

**Current peer-gate blockers for React 19 (as of March 2, 2026):**
- `react-ace@10.1.0` (peer ranges currently cap at React 18)
- `react-dom@17.0.2` / `react-test-renderer@17.0.2` - handled in #9 ([#12897](https://github.com/kubeflow/pipelines/issues/12897)) / #7 ([#12895](https://github.com/kubeflow/pipelines/issues/12895))
- `react-flow-renderer@9.6.5` - handled in #5 ([#12893](https://github.com/kubeflow/pipelines/issues/12893))
- `react-query@3.16.0` - handled in #4 ([#12892](https://github.com/kubeflow/pipelines/issues/12892))
- `react-textarea-autosize@8.3.3` - handled in #8
- Current transitive blockers still reported by `npm run check:react-peers:19`: `react-redux@7.2.4`, `react-redux@8.1.3`, `react-shallow-renderer@16.15.0`, `use-composed-ref@1.1.0`, `use-isomorphic-layout-effect@1.1.1`, `use-latest@1.2.0`

**Acceptance Criteria**:
- [ ] `npm run check:react-peers:19` passes
- [ ] No direct dependencies remain in `check:react-peers:19` output
- [ ] `npm run test:ci` passes

---

## 13. Upgrade React to v19 ([#12901](https://github.com/kubeflow/pipelines/issues/12901))

**Labels**: `area/frontend`, `priority/p1`, `kind/feature`
**Depends on**: #12 ([#12900](https://github.com/kubeflow/pipelines/issues/12900))

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

## 14. Enable StrictMode in dev/test ([#12902](https://github.com/kubeflow/pipelines/issues/12902))

**Labels**: `area/frontend`, `priority/p3`, `kind/chore`
**Depends on**: #13 ([#12901](https://github.com/kubeflow/pipelines/issues/12901))

**Description**:
Enable `<StrictMode>` in the development and test rendering trees. Identify and fix double-invoke side effects. Production behavior remains unchanged -- strict mode is NOT active in production builds. This is deferrable without blocking the React 19 milestone if it surfaces a large number of issues.

**Acceptance Criteria**:
- [ ] `npm run test:ci` passes with strict mode enabled
- [ ] Dev server (`npm start`) runs without strict-mode-related warnings
- [ ] Production builds do NOT include strict mode

---

## 15. Update documentation for React 19 stack ([#12903](https://github.com/kubeflow/pipelines/issues/12903))

**Labels**: `area/frontend`, `priority/p2`, `kind/documentation`, `good first issue`
**Depends on**: #13 ([#12901](https://github.com/kubeflow/pipelines/issues/12901)) (or included inline in prior PRs)

**Description**:
Update `AGENTS.md` "Frontend development" section with React 19, MUI v5, @testing-library/react v14, Storybook 7, @tanstack/react-query. Update `frontend/CONTRIBUTING.md` testing FAQ -- remove references to Enzyme, `react-test-renderer`, React 16-specific patterns. Update `frontend/README.md` stack description. Ensure `frontend/docs/react-17-upgrade-checklist.md` remains removed (or archive/update if it is reintroduced).

**Acceptance Criteria**:
- [ ] All referenced versions match actual installed versions in `package.json`
- [ ] No references to Enzyme, `react-test-renderer`, or React 16-specific patterns remain
- [ ] New contributor can follow `CONTRIBUTING.md` without encountering outdated instructions

---

## Dependency Graph

```
#1 Prereq Cleanup (@jeffspahr)
 └── #2 Peer Gate (@jeffspahr)
      ├── #3  Storybook 7 (#12891) ─────────┐
      ├── #4  react-query (#12892) ──────────┤  (parallelizable)
      ├── #5  react-flow (#12893) ───────────┤
      └── #6  MUI v5 (#12894) ──────────────┘
                                              │
                              #7  JSX + Tests (#12895)
                                              │
                              #8  Ecosystem Deps (#12896)
                                              │
                              #9  React 18 Core (#12897)
                                              │
                              #10 React 18 Stabilization (#12898)
                                              │
                              #11 React 18.3 Checkpoint (#12899)
                                              │
                              #12 Dep Sweep for 19 (#12900)
                                              │
                              #13 React 19 Core (#12901)
                                              │
                              #14 Strict Mode (#12902)
                                              │
                              #15 Documentation (#12903)
```

**Parallelizable**: Issues #3 ([#12891](https://github.com/kubeflow/pipelines/issues/12891)), #4 ([#12892](https://github.com/kubeflow/pipelines/issues/12892)), #5 ([#12893](https://github.com/kubeflow/pipelines/issues/12893)), #6 ([#12894](https://github.com/kubeflow/pipelines/issues/12894)) can be worked on in parallel by different contributors -- they touch different files with no mutual dependencies. All four must be merged before #7 ([#12895](https://github.com/kubeflow/pipelines/issues/12895)) can begin.

**Good first issues**: #3 ([#12891](https://github.com/kubeflow/pipelines/issues/12891) — Storybook upgrade — `npx storybook upgrade` does most of the work), #8 ([#12896](https://github.com/kubeflow/pipelines/issues/12896) — ecosystem deps — small, focused package bumps), #15 ([#12903](https://github.com/kubeflow/pipelines/issues/12903) — documentation — no code logic changes).
