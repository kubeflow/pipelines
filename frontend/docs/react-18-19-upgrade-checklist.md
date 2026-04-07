# GitHub Issues: React 18/19 Upgrade & Testing Modernization

**Tracking Issue**: React 18/19 Frontend Upgrade - Modernize Kubeflow Pipelines UI
**Repository**: [kubeflow/pipelines](https://github.com/kubeflow/pipelines)

**Strategy**: Deps-first where possible, bump-last where necessary. The React 18 rollout, React 19 core bump, Strict Mode enablement, and post-upgrade documentation refresh are now complete on `master`.
**Canonical source**: This checklist is the canonical execution plan and supersedes earlier draft planning notes.

## Status at a glance (updated April 7, 2026 UTC)

- [x] ~~#1 Prereq warning/test cleanup~~ (`PRs`: [#12855](https://github.com/kubeflow/pipelines/pull/12855), [#12856](https://github.com/kubeflow/pipelines/pull/12856), [#12858](https://github.com/kubeflow/pipelines/pull/12858), [#12872](https://github.com/kubeflow/pipelines/pull/12872))
- [x] ~~#2 Add React peer compatibility gate~~ (`PR`: [#12881](https://github.com/kubeflow/pipelines/pull/12881))
- [x] ~~#3 Storybook modernization~~ (`issue`: [#12891](https://github.com/kubeflow/pipelines/issues/12891); `PR`: [#12940](https://github.com/kubeflow/pipelines/pull/12940))
- [x] ~~#4 react-query -> @tanstack/react-query~~ (`issue`: [#12892](https://github.com/kubeflow/pipelines/issues/12892); `PR`: [#12946](https://github.com/kubeflow/pipelines/pull/12946))
- [x] ~~#5 react-flow-renderer -> @xyflow/react~~ (`issue`: [#12893](https://github.com/kubeflow/pipelines/issues/12893); `PR`: [#12945](https://github.com/kubeflow/pipelines/pull/12945))
- [x] ~~#6 Material-UI v4 -> MUI v5~~ (`issue`: [#12894](https://github.com/kubeflow/pipelines/issues/12894); `PR`: [#12925](https://github.com/kubeflow/pipelines/pull/12925))
- [x] ~~#7 JSX runtime and test modernization~~ (`issue`: [#12895](https://github.com/kubeflow/pipelines/issues/12895); `PR`: [#13019](https://github.com/kubeflow/pipelines/pull/13019))
- [x] ~~#8 Remaining ecosystem deps before the core bump~~ (`issue`: [#12896](https://github.com/kubeflow/pipelines/issues/12896); `PR`: [#13025](https://github.com/kubeflow/pipelines/pull/13025))
- [x] ~~#9 Upgrade React to v18~~ (`issue`: [#12897](https://github.com/kubeflow/pipelines/issues/12897); `PR`: [#13070](https://github.com/kubeflow/pipelines/pull/13070))
- [x] ~~#9.5 Finish React 18 test-stack cleanup~~ (follow-up split out of closed [#12895](https://github.com/kubeflow/pipelines/issues/12895); `PRs`: [#13075](https://github.com/kubeflow/pipelines/pull/13075), [#13077](https://github.com/kubeflow/pipelines/pull/13077))
- [x] ~~#10 Stabilize React 18 runtime~~ (`issue`: [#12898](https://github.com/kubeflow/pipelines/issues/12898); `PRs`: [#13075](https://github.com/kubeflow/pipelines/pull/13075), [#13077](https://github.com/kubeflow/pipelines/pull/13077))
- [x] ~~#11 React 18.3 deprecation checkpoint~~ (`issue`: [#12899](https://github.com/kubeflow/pipelines/issues/12899); `PR`: [#13080](https://github.com/kubeflow/pipelines/pull/13080))
- [x] ~~#12 Dependency sweep for React 19~~ (`issue`: [#12900](https://github.com/kubeflow/pipelines/issues/12900); `PR`: [#13082](https://github.com/kubeflow/pipelines/pull/13082))
- [x] ~~#13 Upgrade React to v19~~ (`issue`: [#12901](https://github.com/kubeflow/pipelines/issues/12901); `PR`: [#13153](https://github.com/kubeflow/pipelines/pull/13153))
- [x] ~~#14 Enable StrictMode in dev/test~~ (`issue`: [#12902](https://github.com/kubeflow/pipelines/issues/12902); `PR`: [#13159](https://github.com/kubeflow/pipelines/pull/13159))
- [x] ~~#15 Update documentation for the post-upgrade stack~~ (`issue`: [#12903](https://github.com/kubeflow/pipelines/issues/12903); `PR`: [#13164](https://github.com/kubeflow/pipelines/pull/13164))

**Current focus**:
- The React 18/19 upgrade track is complete through `#15`.
- Remaining React 19 follow-up cleanup is limited to smaller polish items noted during `#13`, such as removing the temporary `frontend/src/react19-compat.d.ts` shim and clearing the remaining `act(...)` warning noise in `frontend/src/components/CustomTable.test.tsx`.

**How to contribute**: This checklist is now complete. Follow-up frontend work should track new issues directly and continue to pass `npm run test:ci` and `npm run build` before merge.

---

## 1. Prereq warning/test cleanup

**Labels**: `area/frontend`, `priority/p0`, `kind/cleanup`
**Assignee**: @jeffspahr
**Depends on**: Nothing

**Status**:
Completed via [#12855](https://github.com/kubeflow/pipelines/pull/12855), [#12856](https://github.com/kubeflow/pipelines/pull/12856), [#12858](https://github.com/kubeflow/pipelines/pull/12858), and [#12872](https://github.com/kubeflow/pipelines/pull/12872).

**Description**:
Land cleanup PRs that remove `react-dom/test-utils` imports, remove `snapshot-diff`, upgrade `re-resizable`, add DOM nesting warning coverage, and drop dead dependencies. Goal: reduce warning/test noise before dependency migrations begin.

---

## 2. Add React peer compatibility gate

**Labels**: `area/frontend`, `priority/p0`, `kind/infra`
**Assignee**: @jeffspahr
**Depends on**: #1

**Status**:
Completed via [#12881](https://github.com/kubeflow/pipelines/pull/12881).

**Description**:
Add `check-react-peers.mjs` and wire it into `npm run test:ci`. The repo now exposes:
- `npm run check:react-peers` (targets React 19 by default)
- `npm run check:react-peers:18`

> **Note**: `check:react-peers:19` was originally a separate alias but was removed after the React 19 upgrade landed; `check:react-peers` now targets React 19 directly.

This CI guardrail prevents new dependency additions from silently breaking the targeted React major.

---

## 3. ~~Upgrade Storybook 6~~ Completed via Storybook 10 ([#12891](https://github.com/kubeflow/pipelines/issues/12891), [#12940](https://github.com/kubeflow/pipelines/pull/12940))

**Labels**: `area/frontend`, `priority/p1`, `kind/chore`, `good first issue`
**Depends on**: #2

**Status**:
Completed by [#12940](https://github.com/kubeflow/pipelines/pull/12940). Earlier draft targets were superseded by a direct upgrade from Storybook 6 to Storybook 10 on the Vite builder.

**Description**:
Modernize Storybook while staying compatible with the staged pre-React-18 environment. Replace the old Webpack builder setup with `@storybook/react-vite`, clear obsolete transitive deps, and keep the story set rendering under the modern frontend toolchain.

**Acceptance Criteria**:
- [x] `npm run storybook` renders the migrated stories without errors
- [x] `npm run build:storybook` succeeds
- [x] `npm run test:ci` passes
- [x] Storybook no longer appears as a React peer gate blocker

---

## 4. Migrate react-query to @tanstack/react-query ([#12892](https://github.com/kubeflow/pipelines/issues/12892))

**Labels**: `area/frontend`, `priority/p1`, `kind/chore`
**Depends on**: #2

**Status**:
Completed by [#12946](https://github.com/kubeflow/pipelines/pull/12946). The repo now uses `@tanstack/react-query` v5.

**Description**:
Replace `react-query` v3 with TanStack Query while preserving existing data fetching, polling, and caching behavior.

**Acceptance Criteria**:
- [x] Data fetching, polling, and cache behavior remain intact
- [x] `react-query` is removed as a React peer blocker
- [x] `npm run test:ci` passes

---

## 5. Migrate react-flow-renderer to @xyflow/react ([#12893](https://github.com/kubeflow/pipelines/issues/12893))

**Labels**: `area/frontend`, `priority/p1`, `kind/chore`
**Depends on**: #2

**Status**:
Completed by [#12945](https://github.com/kubeflow/pipelines/pull/12945). The repo now uses `@xyflow/react`.

**Description**:
Replace deprecated `react-flow-renderer` v9 with `@xyflow/react`, updating the DAG visualization code, stories, and tests.

**Acceptance Criteria**:
- [x] Pipeline graph renders correctly with drag, zoom, and pan interactions
- [x] `react-flow-renderer` is removed as a React peer blocker
- [x] `npm run test:ci` passes

---

## 6. Migrate Material-UI v4 to MUI v5 ([#12894](https://github.com/kubeflow/pipelines/issues/12894))

**Labels**: `area/frontend`, `priority/p1`, `kind/chore`
**Depends on**: #2

**Status**:
Completed by [#12925](https://github.com/kubeflow/pipelines/pull/12925). The repo now uses `@mui/material`, `@mui/icons-material`, and Emotion.

**Description**:
Migrate `@material-ui/core` and `@material-ui/icons` to MUI v5, including theme migration and the few non-codemod-safe styling updates.

**Acceptance Criteria**:
- [x] Visual smoke comparison completed on the primary pages
- [x] `npm run test:ci && npm run build` pass
- [x] Generated directories remain untouched by migration codemods
- [x] Styling remains acceptable without migration-specific hacks

---

## 7. ~~Modernize JSX runtime and test utilities~~ Completed with one follow-up extracted ([#12895](https://github.com/kubeflow/pipelines/issues/12895), [#13019](https://github.com/kubeflow/pipelines/pull/13019))

**Labels**: `area/frontend`, `priority/p2`, `kind/chore`
**Depends on**: #3, #4, #5, #6

**Status**:
The shipped work is complete: [#13019](https://github.com/kubeflow/pipelines/pull/13019) enabled the modern JSX transform, removed the legacy test-renderer packages, removed `react-dom/test-utils` imports, and added coverage-baseline tooling. The part of the original issue that called for upgrading `@testing-library/react` to a React 18-native version did not land before [#12895](https://github.com/kubeflow/pipelines/issues/12895) was closed, so that remaining work is now tracked here as `#9.5`.

**Description**:
Land the pre-core-bump JSX and test modernization work, then carry the remaining React 18-specific testing-library cleanup as a follow-up once React 18 is on `master`.

**Acceptance Criteria**:
- [x] `react-jsx` is enabled in `tsconfig.json`
- [x] Legacy test-renderer packages are removed
- [x] No `react-dom/test-utils` imports remain
- [x] Coverage baseline tooling is available

---

## 8. ~~Update remaining ecosystem dependencies~~ Completed ([#12896](https://github.com/kubeflow/pipelines/issues/12896), [#13025](https://github.com/kubeflow/pipelines/pull/13025))

**Labels**: `area/frontend`, `priority/p2`, `kind/chore`, `good first issue`
**Depends on**: #7

**Status**:
Completed by [#13025](https://github.com/kubeflow/pipelines/pull/13025). The repo now uses `markdown-to-jsx` v7, `react-dropzone` v14, and `react-textarea-autosize` 8.5.9. The pre-core-bump React peer gate is fully green with an empty allowlist.

**Description**:
Upgrade the remaining ecosystem packages with outdated peer ranges and drive the pre-core-bump React peer gate to green before the React 18 bump.

**Current note after #9.5**:
The `@testing-library/react` allowlist exception has been cleared. The React 18 peer gate now passes with an empty allowlist.

**Acceptance Criteria**:
- [x] `npm run check:react-peers` passes with an empty allowlist
- [x] `npm run test:ci && npm run build` pass
- [x] Markdown rendering and file upload continue to work

---

## 9. ~~Upgrade React to v18~~ Completed ([#12897](https://github.com/kubeflow/pipelines/issues/12897), [#13070](https://github.com/kubeflow/pipelines/pull/13070))

**Labels**: `area/frontend`, `priority/p1`, `kind/feature`
**Depends on**: #8

**Status**:
Completed by [#13070](https://github.com/kubeflow/pipelines/pull/13070), merged on March 19, 2026 UTC. `master` now uses React 18, ReactDOM 18, `createRoot()` in `frontend/src/index.tsx`, and a default peer gate target of React 18 in `frontend/package.json`.

**Follow-up note**:
The React 18 core bump itself landed in [#13070](https://github.com/kubeflow/pipelines/pull/13070). The package-level test-stack cleanup then landed in [#13075](https://github.com/kubeflow/pipelines/pull/13075), and [#13077](https://github.com/kubeflow/pipelines/pull/13077) closed the remaining noisy-suite cleanup and final `NewRun` batching fix.

**Description**:
Bump `react` and `react-dom` to v18, migrate the entrypoint to `createRoot()`, fix the type-level React 18 breakages, and keep the app running cleanly on `master`.

**Acceptance Criteria**:
- [x] `npm run test:ci && npm run build` pass
- [x] `npm run check:react-peers:18` passes
- [x] Manual smoke testing was completed during [#13070](https://github.com/kubeflow/pipelines/pull/13070)
- [x] Snapshot updates were regenerated and reviewed in the React 18 PR

---

## 9.5. ~~Finish React 18 test-stack cleanup~~ Completed (follow-up from closed [#12895](https://github.com/kubeflow/pipelines/issues/12895), [#13075](https://github.com/kubeflow/pipelines/pull/13075), [#13077](https://github.com/kubeflow/pipelines/pull/13077))

**Labels**: `area/frontend`, `priority/p2`, `kind/chore`
**Depends on**: #9

**Status**:
Completed via [#13075](https://github.com/kubeflow/pipelines/pull/13075) and [#13077](https://github.com/kubeflow/pipelines/pull/13077). [#13075](https://github.com/kubeflow/pipelines/pull/13075) upgraded the testing stack and removed the React 18 compatibility shims; [#13077](https://github.com/kubeflow/pipelines/pull/13077) finished the remaining noisy-suite cleanup so the previously problematic targeted reruns are now clean of React 18 `act(...)` warnings.

**Changes made**:
- Upgraded `@testing-library/react` to `^16.3.2` and `@testing-library/dom` to `^10.4.1` in `frontend/package.json`
- Removed `legacy-peer-deps=true` from `frontend/.npmrc`
- Cleared the React 18 allowlist in `frontend/docs/react-peer-allowlist.json`
- Removed `ReactDOM.render`/`unmountComponentAtNode` deprecation warning suppression from `frontend/src/vitest.setup.ts`
- Removed `notifyManager.setNotifyFunction` act() workaround for React Query from `frontend/src/vitest.setup.ts`
- Removed `filterReactDeprecationWarnings` utility from `frontend/src/TestUtils.ts`
- Updated `frontend/src/components/Metric.test.tsx` and `frontend/src/pages/ExperimentDetails.test.tsx` to remove filter usage
- Updated `frontend/src/pages/ExperimentDetails.test.tsx`, `frontend/src/pages/NewPipelineVersion.test.tsx`, `frontend/src/pages/NewRun.test.tsx`, and `frontend/src/pages/RunDetails.test.tsx` to wrap the remaining direct instance-method and modal interaction flows in explicit `act()` + flush handling
- Regenerated all affected snapshots (95 snapshot updates across multiple files)

**Acceptance Criteria**:
- [x] `npm run check:react-peers:18` passes with an empty allowlist
- [x] `npm ci` no longer depends on `legacy-peer-deps=true` for the frontend
- [x] Targeted reruns of `ExperimentDetails`, `NewPipelineVersion`, `NewRun`, and `RunDetails` are clean of React 18 `act(...)` warnings
- [x] `npm run build` and the affected frontend suite verification pass

---

## 10. ~~Stabilize React 18 runtime~~ Completed ([#12898](https://github.com/kubeflow/pipelines/issues/12898), [#13075](https://github.com/kubeflow/pipelines/pull/13075), [#13077](https://github.com/kubeflow/pipelines/pull/13077))

**Labels**: `area/frontend`, `priority/p1`, `kind/bug`
**Depends on**: #9.5

**Status**:
Completed via [#13075](https://github.com/kubeflow/pipelines/pull/13075) and [#13077](https://github.com/kubeflow/pipelines/pull/13077). [#13075](https://github.com/kubeflow/pipelines/pull/13075) fixed the first React 18 automatic batching regressions and preserved the production bundle baseline; [#13077](https://github.com/kubeflow/pipelines/pull/13077) closed the remaining `NewRun` validation timing bug and cleaned the affected regression suites.

**Changes made**:
- **CompareV1.tsx**: Refactored `_loadParameters` and `_loadMetrics` to accept state as parameters instead of reading from `this.state` after batched `setStateSafe` calls.
- **ExperimentDetails.tsx**: Moved `_selectionChanged([])` into the `setStateSafe` callback to prevent reading stale `runStorageState`.
- **NewPipelineVersion.tsx**: Consolidated multiple `setState` calls in `componentDidMount` into a single call with `_validate()` in the callback.
- **NewRun.tsx**: Moved the embedded-pipeline and clone-form `_validate()` calls into `setStateSafe(..., callback)` so React 18 batching no longer races validation against stale state.
- **RunDetails.test.tsx**: Wrapped state assertions in `await waitFor()` to account for asynchronous state updates.
- **NewRun.test.tsx**: Wrapped state-changing calls in `await act()` and assertions in `await waitFor()`.
- **NewPipelineVersion.test.tsx**: Wrapped state-changing method calls in `await act()`.
- **ExperimentDetails.test.tsx**: Waited for the post-load UI state and wrapped the refresh path in explicit async flushing to match React 18 update timing.
- **RecurringRunDetailsV2FC.test.tsx**: Fixed test isolation issue.
- Regenerated all affected snapshots.

**Bundle size comparison** (production build, `npm run build`):
- Baseline JS: 4,499.35 kB (gzip: 1,004.36 kB)
- After changes JS: 4,499.35 kB (gzip: 1,004.36 kB)
- **Delta: 0%** (well within the 5% threshold)

**Acceptance Criteria**:
- [x] `npm run test:ci` is stable with zero flaky tests
- [x] Visual regression comparison is clean
- [x] Bundle size remains within 5% of the pre-upgrade baseline (0% change)
- [x] Both single-user and multi-user modes function correctly (manual smoke coverage was completed in [#13075](https://github.com/kubeflow/pipelines/pull/13075); [#13077](https://github.com/kubeflow/pipelines/pull/13077) only tightened one local `NewRun` validation path plus affected tests)

---

## 11. ~~React 18.3 deprecation checkpoint~~ Completed ([#12899](https://github.com/kubeflow/pipelines/issues/12899), [#13080](https://github.com/kubeflow/pipelines/pull/13080))

**Labels**: `area/frontend`, `priority/p2`, `kind/chore`
**Depends on**: #10

**Status**:
Completed by [#13080](https://github.com/kubeflow/pipelines/pull/13080), merged on March 19, 2026 UTC. `frontend/package.json` now pins `react`, `react-dom`, `@types/react`, and `@types/react-dom` to `^18.3.0`, and the React 19 deprecation audit reported no React-specific warnings. Issue [#12899](https://github.com/kubeflow/pipelines/issues/12899) was closed on March 19, 2026.

**Current note**:
The React 18.3 state is now explicit in package metadata. The audit run for [#13080](https://github.com/kubeflow/pipelines/pull/13080) found no React deprecation warnings in browser-console or test output; the only deprecation observed was Node's unrelated `util._extend` (`DEP0060`).

**Description**:
Make the React 18.3 state explicit in package metadata, run the full verification suite, document all React 19 deprecation warnings, and either fix or track them before proceeding.

**Acceptance Criteria**:
- [x] All React 19 deprecation warnings are documented
- [x] Warnings are addressed or tracked before `#13`
- [x] `npm run test:ci` passes

---

## 12. Dependency sweep for React 19 ([#12900](https://github.com/kubeflow/pipelines/issues/12900))

**Labels**: `area/frontend`, `priority/p2`, `kind/chore`
**Depends on**: #11

**Status**:
Completed via [#13082](https://github.com/kubeflow/pipelines/pull/13082), merged on March 20, 2026 UTC. `npm run check:react-peers` (targeting React 19) now passes with a single allowlisted React core blocker that remains until `#13`. Issue [#12900](https://github.com/kubeflow/pipelines/issues/12900) was closed on March 20, 2026.

**Description**:
Run `npm run check:react-peers` (targeting React 19), upgrade any remaining React 19-incompatible dependencies, and allowlist the expected React core blocker that is resolved in `#13`.

**Result**:
- Cleared `react-ace` by upgrading to `14.0.1`
- Cleared the transitive `react-redux` blocker by letting `recharts` resolve `react-redux@9.2.0`
- Temporarily allowlisted `react-dom@18.3.1::react=^18.3.1` in `frontend/docs/react-peer-allowlist.json` so the React 19 peer gate stayed green until `#13`

**Current state after `#13`**: That allowlist entry is gone; `frontend/docs/react-peer-allowlist.json` is `[]` for targets `17`, `18`, and `19`.

**Acceptance Criteria**:
- [x] `npm run check:react-peers` (targeting React 19) passed with only the expected allowlisted `react-dom` core blocker until `#13` landed
- [x] No non-core React 19 blockers remain
- [x] `npm run test:ci` passes

---

## 13. ~~Upgrade React to v19~~ Completed ([#12901](https://github.com/kubeflow/pipelines/issues/12901), [#13153](https://github.com/kubeflow/pipelines/pull/13153))

**Labels**: `area/frontend`, `priority/p1`, `kind/feature`
**Depends on**: #12

**Status**:
Completed by [#13153](https://github.com/kubeflow/pipelines/pull/13153), merged on March 27, 2026 UTC. `frontend/package.json` now pins `react`, `react-dom`, `@types/react`, and `@types/react-dom` to `^19`, and the default React peer gate target is React 19.

**Current note**:
The React 19 core bump is complete. Two smaller follow-up cleanup items remain from the review thread:
- remove the temporary `frontend/src/react19-compat.d.ts` shim by converting remaining `JSX.Element`-style annotations to `React.JSX.*` or inferred types
- clear the remaining `act(...)` warning noise reported from `frontend/src/components/CustomTable.test.tsx`

**Description** (historical):
Bump `react`, `react-dom`, `@types/react`, and `@types/react-dom` to v19, address any React 19-specific breakages, regenerate and review affected snapshots, and clear the temporary React core allowlist from the `#12` sweep.

**Acceptance Criteria**:
- [x] `npm run test:ci && npm run build` pass
- [x] `npm install` completes without peer-dependency warnings (and the `#12` core allowlist is removed)
- [x] Manual smoke testing completed as part of the React 19 rollout
- [x] `npm run check:react-peers` (targeting React 19) passes with an empty allowlist

---

## 14. Enable StrictMode in dev/test ([#12902](https://github.com/kubeflow/pipelines/issues/12902))

**Labels**: `area/frontend`, `priority/p3`, `kind/chore`
**Depends on**: #13

**Status**:
Completed by [#13159](https://github.com/kubeflow/pipelines/pull/13159). The non-production app bootstrap now renders inside `<React.StrictMode>`, and the Vitest global setup enables Testing Library `configure({ reactStrictMode: true })` so direct `render()` calls exercise the same Strict Mode behavior in tests.

**Description**:
Enable `<StrictMode>` in development and test rendering paths, fix double-invoke side effects, and keep production behavior unchanged.

**Acceptance Criteria**:
- [x] `npm run test:ci` passes with strict mode enabled
- [x] `npm start` runs without strict-mode-related warnings
- [x] Production builds do not include strict mode

---

## 15. Update documentation for the post-upgrade stack ([#12903](https://github.com/kubeflow/pipelines/issues/12903))

**Labels**: `area/frontend`, `priority/p2`, `kind/documentation`, `good first issue`
**Depends on**: #13 (or inline updates in prior PRs)

**Status**:
Completed by [#13164](https://github.com/kubeflow/pipelines/pull/13164). The repo-level and frontend contributor docs now describe the React 19 stack, the React 19 default peer gate, Storybook 10, MUI v5, TanStack Query v5, and the post-`#13159` Strict Mode expectations for dev and Vitest.

**Description**:
Update the top-level docs once the stack is settled:
- `AGENTS.md`
- `frontend/CONTRIBUTING.md`
- `frontend/README.md`
- any remaining upgrade checklists or archived planning docs

The final state should reflect the post-upgrade stack without leaving references to removed tools or old migration-only instructions.

**Acceptance Criteria**:
- [x] All referenced versions match `frontend/package.json`
- [x] No stale references to Enzyme, `react-test-renderer`, or pre-upgrade React guidance remain
- [x] A new contributor can follow the frontend docs without hitting outdated instructions

---

## Dependency Graph

```text
#1 Prereq Cleanup [done]
 \- #2 Peer Gate [done]
     |- #3  Storybook [done]
     |- #4  TanStack Query [done]
     |- #5  XYFlow [done]
     \- #6  MUI v5 [done]
                   |
                   #7  JSX + Tests (pre-core-bump portion) [done]
                   |
                   #8  Ecosystem Deps [done]
                   |
                   #9  React 18 Core [done]
                   |
                   #9.5 Test-Stack Cleanup [done]
                   |
                   #10 React 18 Stabilization [done]
                   |
                   #11 React 18.3 Checkpoint [done]
                   |
                   #12 React 19 Dependency Sweep [done]
                   |
                   #13 React 19 Core [done]
                   |
                   #14 StrictMode [done]
                   |
                   #15 Documentation [done]
```

**Parallelizable**:
`#1` through `#15` are complete. Future frontend cleanup can proceed independently of this upgrade track.
