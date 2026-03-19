# GitHub Issues: React 18/19 Upgrade & Testing Modernization

**Tracking Issue**: React 18/19 Frontend Upgrade - Modernize Kubeflow Pipelines UI
**Repository**: [kubeflow/pipelines](https://github.com/kubeflow/pipelines)

**Strategy**: Deps-first where possible, bump-last where necessary. The React 17 staging work is complete, the React 18 core bump is now on `master`, and the remaining near-term work is cleanup around the test stack before moving deeper into React 18 stabilization and React 19.
**Canonical source**: This checklist is the canonical execution plan and supersedes earlier draft planning notes.

## Status at a glance (updated March 18, 2026 ET)

- [x] ~~#1 Prereq warning/test cleanup~~ (`PRs`: [#12855](https://github.com/kubeflow/pipelines/pull/12855), [#12856](https://github.com/kubeflow/pipelines/pull/12856), [#12858](https://github.com/kubeflow/pipelines/pull/12858), [#12872](https://github.com/kubeflow/pipelines/pull/12872))
- [x] ~~#2 Add React peer compatibility gate~~ (`PR`: [#12881](https://github.com/kubeflow/pipelines/pull/12881))
- [x] ~~#3 Storybook modernization~~ (`issue`: [#12891](https://github.com/kubeflow/pipelines/issues/12891); `PR`: [#12940](https://github.com/kubeflow/pipelines/pull/12940))
- [x] ~~#4 react-query -> @tanstack/react-query~~ (`issue`: [#12892](https://github.com/kubeflow/pipelines/issues/12892); `PR`: [#12946](https://github.com/kubeflow/pipelines/pull/12946))
- [x] ~~#5 react-flow-renderer -> @xyflow/react~~ (`issue`: [#12893](https://github.com/kubeflow/pipelines/issues/12893); `PR`: [#12945](https://github.com/kubeflow/pipelines/pull/12945))
- [x] ~~#6 Material-UI v4 -> MUI v5~~ (`issue`: [#12894](https://github.com/kubeflow/pipelines/issues/12894); `PR`: [#12925](https://github.com/kubeflow/pipelines/pull/12925))
- [x] ~~#7 JSX runtime and test modernization~~ (`issue`: [#12895](https://github.com/kubeflow/pipelines/issues/12895); `PR`: [#13019](https://github.com/kubeflow/pipelines/pull/13019))
- [x] ~~#8 Remaining React 17 ecosystem deps~~ (`issue`: [#12896](https://github.com/kubeflow/pipelines/issues/12896); `PR`: [#13025](https://github.com/kubeflow/pipelines/pull/13025))
- [x] ~~#9 Upgrade React to v18~~ (`issue`: [#12897](https://github.com/kubeflow/pipelines/issues/12897); `PR`: [#13070](https://github.com/kubeflow/pipelines/pull/13070))
- [ ] #9.5 Finish React 18 test-stack cleanup (`issue`: none yet; follow-up split out of closed [#12895](https://github.com/kubeflow/pipelines/issues/12895); `PR`: none yet)
- [ ] #10 Stabilize React 18 runtime (`issue`: [#12898](https://github.com/kubeflow/pipelines/issues/12898); `PR`: none yet)
- [ ] #11 React 18.3 deprecation checkpoint (`issue`: [#12899](https://github.com/kubeflow/pipelines/issues/12899); `PR`: none yet)
- [ ] #12 Dependency sweep for React 19 (`issue`: [#12900](https://github.com/kubeflow/pipelines/issues/12900); `PR`: none yet)
- [ ] #13 Upgrade React to v19 (`issue`: [#12901](https://github.com/kubeflow/pipelines/issues/12901); `PR`: none yet)
- [ ] #14 Enable StrictMode in dev/test (`issue`: [#12902](https://github.com/kubeflow/pipelines/issues/12902); `PR`: none yet)
- [ ] #15 Update documentation for the post-upgrade stack (`issue`: [#12903](https://github.com/kubeflow/pipelines/issues/12903); `PR`: none yet)

**Current focus**:
- `master` already runs React 18 and `createRoot()` via [#13070](https://github.com/kubeflow/pipelines/pull/13070), so the next real milestone is `#9.5`: upgrade `@testing-library/react` off v12 and remove the last React 18 compatibility workaround.
- That workaround is still visible in source today: `frontend/docs/react-peer-allowlist.json` still allowlists `@testing-library/react@12.1.5` for React 18, `frontend/.npmrc` still sets `legacy-peer-deps=true`, and `frontend/src/vitest.setup.ts` / `frontend/src/TestUtils.ts` still filter `ReactDOM.render()` / `unmountComponentAtNode()` deprecation noise emitted by testing-library v12.
- After `#9.5`, move to `#10` React 18 stabilization and `#11` explicit React 18.3 warning review.
- No open PRs were found for `#9.5` or `#10` through `#15`.

**How to contribute**: `#1` through `#9` are complete. The next actionable work is `#9.5`, followed by `#10`. Every PR should pass `npm run test:ci` and `npm run build` before merge.

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
- `npm run check:react-peers`
- `npm run check:react-peers:18`
- `npm run check:react-peers:19`

This CI guardrail prevents new dependency additions from silently breaking the targeted React major.

---

## 3. ~~Upgrade Storybook 6~~ Completed via Storybook 10 ([#12891](https://github.com/kubeflow/pipelines/issues/12891), [#12940](https://github.com/kubeflow/pipelines/pull/12940))

**Labels**: `area/frontend`, `priority/p1`, `kind/chore`, `good first issue`
**Depends on**: #2

**Status**:
Completed by [#12940](https://github.com/kubeflow/pipelines/pull/12940). The original Storybook 7 target was superseded by a direct upgrade from Storybook 6 to Storybook 10 on the Vite builder.

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
Completed by [#12946](https://github.com/kubeflow/pipelines/pull/12946). The repo now uses `@tanstack/react-query` v4.

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
The shipped work is complete: [#13019](https://github.com/kubeflow/pipelines/pull/13019) enabled the modern JSX transform, removed `react-test-renderer`, removed `react-dom/test-utils` imports, and added coverage-baseline tooling. The part of the original issue that called for upgrading `@testing-library/react` to a React 18-native version did not land before [#12895](https://github.com/kubeflow/pipelines/issues/12895) was closed, so that remaining work is now tracked here as `#9.5`.

**Description**:
Land the React 17-safe JSX and test modernization work, then carry the remaining React 18-specific testing-library cleanup as a follow-up once React 18 is on `master`.

**Acceptance Criteria**:
- [x] `react-jsx` is enabled in `tsconfig.json`
- [x] `react-test-renderer` / `@types/react-test-renderer` are removed
- [x] No `react-dom/test-utils` imports remain
- [x] Coverage baseline tooling is available

---

## 8. ~~Update remaining ecosystem dependencies~~ Completed ([#12896](https://github.com/kubeflow/pipelines/issues/12896), [#13025](https://github.com/kubeflow/pipelines/pull/13025))

**Labels**: `area/frontend`, `priority/p2`, `kind/chore`, `good first issue`
**Depends on**: #7

**Status**:
Completed by [#13025](https://github.com/kubeflow/pipelines/pull/13025). The repo now uses `markdown-to-jsx` v7, `react-dropzone` v14, and `react-textarea-autosize` 8.5.9. React 17 peer compatibility is fully green with an empty allowlist.

**Description**:
Upgrade the remaining React 17-limited ecosystem packages and drive the React 17 peer gate to green before the React 18 bump.

**Current note after #13070**:
The React 18 core bump is now complete. The only remaining React 18 peer-gate exception is `@testing-library/react@12.1.5`, which is intentionally allowlisted until `#9.5`.

**Acceptance Criteria**:
- [x] `node scripts/check-react-peers.mjs --target 17` passes with an empty allowlist
- [x] `npm run test:ci && npm run build` pass
- [x] Markdown rendering and file upload continue to work

---

## 9. ~~Upgrade React to v18~~ Completed ([#12897](https://github.com/kubeflow/pipelines/issues/12897), [#13070](https://github.com/kubeflow/pipelines/pull/13070))

**Labels**: `area/frontend`, `priority/p1`, `kind/feature`
**Depends on**: #8

**Status**:
Completed by [#13070](https://github.com/kubeflow/pipelines/pull/13070), merged on March 19, 2026 UTC. `master` now uses React 18, ReactDOM 18, `createRoot()` in `frontend/src/index.tsx`, and a default peer gate target of React 18 in `frontend/package.json`.

**Current caveat**:
`#9` landed with one explicit compatibility shim that still needs cleanup in `#9.5`:
- `frontend/docs/react-peer-allowlist.json` allowlists `@testing-library/react@12.1.5` for React 18.
- `frontend/.npmrc` sets `legacy-peer-deps=true` because npm would otherwise reject the testing-library v12 peer range.
- `frontend/src/vitest.setup.ts` and `frontend/src/TestUtils.ts` filter React 18 deprecation warnings emitted by testing-library v12 internals.

**Description**:
Bump `react` and `react-dom` to v18, migrate the entrypoint to `createRoot()`, fix the type-level React 18 breakages, and keep the app running cleanly on `master`.

**Acceptance Criteria**:
- [x] `npm run test:ci && npm run build` pass
- [x] `npm run check:react-peers:18` passes
- [x] Manual smoke testing was completed during [#13070](https://github.com/kubeflow/pipelines/pull/13070)
- [x] Snapshot updates were regenerated and reviewed in the React 18 PR

---

## 9.5. Finish React 18 test-stack cleanup (follow-up from closed [#12895](https://github.com/kubeflow/pipelines/issues/12895))

**Labels**: `area/frontend`, `priority/p2`, `kind/chore`
**Depends on**: #9

**Status**:
Not started as standalone follow-up work.

**Description**:
Upgrade `@testing-library/react` from v12 to a React 18-native release, remove the last React 18 peer-compatibility exception, and delete the temporary shims added in [#13070](https://github.com/kubeflow/pipelines/pull/13070). This includes:
- removing the React 18 allowlist entry from `frontend/docs/react-peer-allowlist.json`
- removing `legacy-peer-deps=true` from `frontend/.npmrc`
- removing the testing-library-v12-specific warning filtering from `frontend/src/vitest.setup.ts` and `frontend/src/TestUtils.ts`
- fixing any remaining `@xyflow/react` / `d3-drag` test behavior that only reproduces once the newer testing-library stack is in place

**Acceptance Criteria**:
- [ ] `npm run check:react-peers:18` passes with an empty allowlist
- [ ] `npm ci` no longer depends on `legacy-peer-deps=true` for the frontend
- [ ] React 18 test runs no longer emit the currently filtered `ReactDOM.render()` / `unmountComponentAtNode()` deprecation warnings
- [ ] `npm run test:ci && npm run build` pass

---

## 10. Stabilize React 18 runtime ([#12898](https://github.com/kubeflow/pipelines/issues/12898))

**Labels**: `area/frontend`, `priority/p1`, `kind/bug`
**Depends on**: #9.5

**Status**:
Not started.

**Description**:
With the React 18 core bump already landed, this phase is now about removing regressions and flaky behavior that remain after the test-stack cleanup. Audit automatic batching assumptions, compare bundle size against the pre-React-18 baseline, and smoke test both single-user and multi-user deployments.

**Acceptance Criteria**:
- [ ] `npm run test:ci` is stable with zero flaky tests
- [ ] Visual regression comparison is clean
- [ ] Bundle size remains within 5% of the pre-upgrade baseline
- [ ] Both single-user and multi-user modes function correctly

---

## 11. React 18.3 deprecation checkpoint ([#12899](https://github.com/kubeflow/pipelines/issues/12899))

**Labels**: `area/frontend`, `priority/p2`, `kind/chore`
**Depends on**: #10

**Status**:
Not started as an explicit audit step.

**Current note**:
The current lockfile already resolves `react-dom@18.3.1` under the `^18.2.0` range in `package.json`, so this item is now less about the first 18.3 install and more about explicitly auditing, documenting, and clearing any React 19 deprecation warnings before the React 19 bump.

**Description**:
Make the React 18.3 state explicit in package metadata, run the full verification suite, document all React 19 deprecation warnings, and either fix or track them before proceeding.

**Acceptance Criteria**:
- [ ] All React 19 deprecation warnings are documented
- [ ] Warnings are addressed or tracked before `#13`
- [ ] `npm run test:ci` passes

---

## 12. Dependency sweep for React 19 ([#12900](https://github.com/kubeflow/pipelines/issues/12900))

**Labels**: `area/frontend`, `priority/p2`, `kind/chore`
**Depends on**: #11

**Status**:
Not started.

**Description**:
Run `npm run check:react-peers:19`, upgrade any remaining React 19-incompatible dependencies, and reduce the peer-gate output to only the expected React core blocker that is resolved in `#13`.

**Current `check:react-peers:19` blockers (verified against `origin/master` on March 18, 2026 ET)**:
- `@testing-library/react@12.1.5` (`react-dom=<18.0.0`, `react=<18.0.0`) - to clear in `#9.5`
- `react-ace@10.1.0` (`react-dom=... || ^18.0.0`, `react=... || ^18.0.0`) - to clear in `#12`
- `react-dom@18.3.1` (`react=^18.3.1`) - expected until `#13`
- Transitive: `react-redux@8.1.3` (`react-dom=^16.8 || ^17.0 || ^18.0`, `react=^16.8 || ^17.0 || ^18.0`)

**Acceptance Criteria**:
- [ ] `npm run check:react-peers:19` is reduced to only the expected `react-dom` blocker for `#13`
- [ ] No non-core React 19 blockers remain
- [ ] `npm run test:ci` passes

---

## 13. Upgrade React to v19 ([#12901](https://github.com/kubeflow/pipelines/issues/12901))

**Labels**: `area/frontend`, `priority/p1`, `kind/feature`
**Depends on**: #12

**Status**:
Not started.

**Description**:
Bump `react`, `react-dom`, `@types/react`, and `@types/react-dom` to v19, then handle the small set of React 19-specific source changes still visible in the repo today. Known examples include the `forwardRef`-based test mocks still present in:
- `frontend/src/pages/RecurringRunList.test.tsx`
- `frontend/src/pages/ArchivedRuns.test.tsx`
- `frontend/src/pages/ArchivedExperiments.test.tsx`
- `frontend/src/pages/AllRunsList.test.tsx`

Regenerate and review the affected snapshots after the bump.

**Acceptance Criteria**:
- [ ] `npm run test:ci && npm run build` pass
- [ ] `npm install` completes without peer-dependency warnings
- [ ] Manual smoke testing shows zero new console warnings or deprecation messages
- [ ] `npm run check:react-peers:19` passes

---

## 14. Enable StrictMode in dev/test ([#12902](https://github.com/kubeflow/pipelines/issues/12902))

**Labels**: `area/frontend`, `priority/p3`, `kind/chore`
**Depends on**: #13

**Status**:
Not started. `frontend/src/index.tsx` still renders without `<StrictMode>`.

**Description**:
Enable `<StrictMode>` in development and test rendering paths, fix double-invoke side effects, and keep production behavior unchanged.

**Acceptance Criteria**:
- [ ] `npm run test:ci` passes with strict mode enabled
- [ ] `npm start` runs without strict-mode-related warnings
- [ ] Production builds do not include strict mode

---

## 15. Update documentation for the post-upgrade stack ([#12903](https://github.com/kubeflow/pipelines/issues/12903))

**Labels**: `area/frontend`, `priority/p2`, `kind/documentation`, `good first issue`
**Depends on**: #13 (or inline updates in prior PRs)

**Status**:
Not started. Some docs are already stale today; for example, `AGENTS.md` still describes the frontend as React 17 with Material-UI v3 and says the current peer gate target is React 17.

**Description**:
Update the top-level docs once the stack is settled:
- `AGENTS.md`
- `frontend/CONTRIBUTING.md`
- `frontend/README.md`
- any remaining upgrade checklists or archived planning docs

The final state should reflect the post-upgrade stack without leaving references to removed tools or old migration-only instructions.

**Acceptance Criteria**:
- [ ] All referenced versions match `frontend/package.json`
- [ ] No stale references to Enzyme, `react-test-renderer`, or pre-upgrade React guidance remain
- [ ] A new contributor can follow the frontend docs without hitting outdated instructions

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
                   #7  JSX + Tests (React 17-safe portion) [done]
                   |
                   #8  Ecosystem Deps [done]
                   |
                   #9  React 18 Core [done]
                   |
                   #9.5 Test-Stack Cleanup [next]
                   |
                   #10 React 18 Stabilization
                   |
                   #11 React 18.3 Checkpoint
                   |
                   #12 React 19 Dependency Sweep
                   |
                   #13 React 19 Core
                   |
                   #14 StrictMode
                   |
                   #15 Documentation
```

**Parallelizable**:
The old React 17 staging tranche is complete. The practical next start point is `#9.5`, followed by `#10`. `#15` remains a good first issue once the stack stops moving.
