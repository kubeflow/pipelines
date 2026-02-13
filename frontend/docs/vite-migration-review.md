# Vite Migration Review (Updated)

Post-migration analysis of the CRA to Vite + Vitest migration.

**Scope:** 290 files changed, 123,816 insertions, 146,097 deletions (net -22,281 lines)

---

## Executive Summary

This migration exceeded initial scope but delivered a comprehensive modernization:
- CRA → Vite 7 (latest)
- Jest + Enzyme → Vitest 4 + Testing Library (UI tests)
- TypeScript 3.8 → 4.9.5
- react-router 4 → 5.3.4

The planning document is exceptionally detailed, tracking 50+ "unplanned" items with rationale for each. The test migration eliminated flaky timer-based tests and reduced snapshot file sizes significantly.

---

## What Was Done Well

### 1. Exceptional Documentation
The `vite-migration-plan.md` is exemplary:
- Every unplanned scope expansion is documented with context
- External references cited for decisions (Testing Library principles, Vitest docs, React lifecycle deprecations)
- Clear delineation between "done now" vs "deferred to follow-up"
- Risks tracked with mitigations

### 2. Vitest Migration Completed (High Risk, Well Executed)
The previous review flagged Vitest as "Low priority, high risk." The agent chose to do it anyway and executed it cleanly:
- All UI tests now use standard `.test.tsx` naming
- Proper `vitest.config.mts` separate from `vite.config.mts`
- Clean setup file with necessary jsdom polyfills (Worker, localStorage, URL.createObjectURL)
- Coverage properly configured with sensible excludes
- `afterEach(cleanup)` ensures test isolation

### 3. Test Stability Improvements
The migration addressed root causes rather than masking symptoms:
- Timer-driven tests stabilized (RunDetails auto-refresh, Tensorboard polling) with explicit mocks and awaits
- Vitest-ready test utilities (fake-timer-aware `flushPromises`, `vi`-based helpers, constructible `ResizeObserver` mock)
- Tensorboard guards against setState after unmount via `_isMounted` + `setStateSafe`
- Explicit async waits added for slow CI/coverage paths to avoid flake without skipping assertions
- jsdom polyfills added (Worker, localStorage, URL.createObjectURL) to remove environment noise

### 4. Forward-Looking Version Choices
- Vite 7 (latest major, ESM-only)
- Vitest 4 (latest major)
- `build.target = 'es2015'` + `browserslist: supports es6-module` codifies modern-only browser support (breaking)
- Config files use `.mts` extension for ESM compatibility

### 5. Snapshot Reduction
The Testing Library migration dramatically reduced snapshot sizes:
- Snapshots test rendered output, not implementation details
- Net reduction of ~22k lines indicates less brittle tests
- URL escaping changes are serialization-only (verified non-breaking)

---

## Breaking Changes

- **Browser support is now modern-only** (ESM-capable browsers). CRA previously used a broad `browserslist` and
  polyfills; Vite does not. We explicitly align `browserslist` to `supports es6-module` and set
  `build.target = 'es2015'` to match this stance.
  Expected impact is low but should be called out in PR/release notes.

### PR / Release Notes Snippet

Breaking: Frontend browser support is now modern-only (ESM-capable browsers). We align `browserslist` to
`supports es6-module` and set `build.target = 'es2015'` to match Vite’s output; legacy browsers are no longer targeted.

### 6. Dependency Upgrades Handled Strategically
- `react-router` 4→5.3.4: Reduces lifecycle warnings, maintains API compatibility
- `react-vis` 1.11→1.12.1: Aligns with Node 22
- `re-resizable` 4.9→4.11: Minor bump within v4

---

## Remaining Technical Debt

### 1. Server Tests Still on Jest (Planned Phase 8)

**Location:** `frontend/server/` tests

**Status:** Intentionally deferred. Plan notes:
> "Keep Jest for `frontend/server` tests until a dedicated Vitest node config is in place."

**Impact:** Two test runners in CI (`npm run test:ui:coverage` + `npm run test:server:coverage`)

**Recommendation:** Complete Phase 8 in a follow-up PR to eliminate Jest entirely.

---

### 2. CommonJS Workaround Persists

**Location:** `frontend/vite.config.mts:57-59`

```ts
commonjsOptions: {
  include: [/node_modules/, /src\/generated/, /src\/third_party\/mlmd\/generated/],
}
```

**Status:** Functional workaround. Generated code uses CommonJS.

**Root cause:** swagger-codegen and protobuf generators emit CJS.

**Recommendation:** Regenerate with ESM-compatible settings or accept as permanent workaround (generated code is rarely touched).

---

### 3. React 16 Lifecycle Warnings (Deferred)

**Sources observed in test runs:**
- `re-resizable@4.11.0` uses `componentWillReceiveProps`
  - Fix: upgrade to v6+ (requires React 16.13.1+)
- `React.createFactory` deprecation warnings from legacy React 16 patterns
  - Fix: upgrade React and/or dependent libraries that still call `createFactory`

**Why deferred:** Both fixes require React upgrade or library replacement, expanding scope beyond tooling migration.

**Recommendation:** Track as separate initiative. Warnings are noise but don't affect functionality.

---

### 4. Vite CJS Node API Deprecation

**Status:** Potential warning if any tooling still uses the deprecated CJS Node API (not observed in current test runs).

**Impact:** Console noise only if it appears.

**Recommendation:** If it surfaces (e.g., Storybook tooling), resolve by moving to ESM-only APIs or a Vite-native builder.

---

### 5. Storybook Uses Webpack 5 (Not Vite)

**Current:** `@storybook/builder-webpack5` + `@storybook/manager-webpack5`

**Opportunity:** Storybook 7+ supports `@storybook/builder-vite` natively.

**Recommendation:** Evaluate Storybook 7 migration as separate effort. Current setup is stable.

---

## Verification Completed

### Environment Variable Audit - PASSED
```bash
grep -r "REACT_APP_" frontend/src/
# Result: No matches found

grep -r "process\.env\." frontend/src/
# Result: Only TZ timezone setting in Vitest setup/tests (expected)
```

### TypeScript Compilation
The migration includes TS 3.8 → 4.9.5. No type errors reported in CI.

**Recommendation:** Run `npx tsc --noEmit` as validation before next major upgrade.

---

## Architecture After Migration

```
Build Pipeline:
  Vite 7 (esbuild) → ESM output → frontend/build/

UI Tests:
  Vitest 4 + Testing Library + jsdom
  Config: frontend/vitest.config.mts
  Pattern: src/**/*.{test,spec}.{ts,tsx}

Server Tests (temporary):
  Jest (pending Phase 8)
  Location: frontend/server/

Storybook:
  @storybook/react 6.3 + webpack5 builder
  Config: frontend/.storybook/main.js

Dev Server:
  Vite dev server on :3000
  Proxy to :3001 for API routes
```

---

## Risk Assessment

| Item | Risk | Status |
|------|------|--------|
| UI test stability | High | Mitigated - Vitest + proper async handling |
| Build output parity | Medium | Mitigated - explicit modern-only target (`build.target = 'es2015'`, `supports es6-module`) |
| Proxy configuration | Medium | Verified - explicit path list |
| TypeScript upgrade | Medium | Mitigated - `npm run typecheck` restored in build/test:ci |
| Dual test runners | Low | Temporary, Phase 8 planned |
| Lifecycle warnings | Low | Cosmetic, deferred intentionally |

---

## Recommended Follow-up Work (Priority Order)

1. **Phase 8: Server test migration** (Medium priority)
   - Move `frontend/server` tests to Vitest
   - Remove Jest from devDependencies
   - Single test runner reduces maintenance

2. **Storybook 7 evaluation** (Low priority)
   - Evaluate `@storybook/builder-vite`
   - Would eliminate webpack from frontend entirely

3. **Build warning cleanup** (Low priority)
   - `google-protobuf` / `protobufjs` eval warnings
   - Main chunk size warning (>500 kB)
   - Address when doing performance or dependency upgrade work

---

## Manual UI Smoke Checklist (runtime dependency upgrades)

These checks mitigate the risk from `react-router`, `react-vis`, and `re-resizable` upgrades that can impact
runtime behavior and visuals beyond unit/snapshot coverage.

Run locally after `npm run start:proxy-and-server` (or against a deployed env):
- **Navigation + routing (react-router)**:
  - Run list → Run details; verify back/forward works and URLs update correctly.
  - Pipeline list → Pipeline details; check route params render correct entity.
  - Experiment list → Experiment details; verify breadcrumbs link to correct pages.
- **Charts/visualizations (react-vis)**:
  - Run details → Visualizations tab; confirm charts render and tooltips/axes appear.
  - Compare (v1/v2) → Confusion matrix / ROC / metrics plots render correctly.
- **Resizable panels (re-resizable)**:
  - Run details side panel: drag resize handle; ensure layout updates without jitter.
  - Any viewer with resizable container (e.g., Tensorboard) responds to resize.

Document results in the PR description if any issues are found.

---

## Local Visual Diff Helper (optional)

This repo now includes a local-only helper to capture and diff screenshots without storing binaries in git.
Reviewers can run it locally to compare a baseline commit to this branch.

Recommended usage (spins up baseline + current on separate ports and writes side-by-side PNGs):

```bash
bash frontend/scripts/visual-compare-run.sh
```

Outputs:
- `frontend/.visual/side-by-side/` (side-by-side PNGs)
- `frontend/.visual/report.html` (optional HTML report)

Manual alternative (UI must be running on `http://localhost:3000`, e.g. `npm run start` or
`npm run start:proxy-and-server`):

1. Run the baseline UI from a separate worktree so the helper stays available on this branch:
   - `git worktree add ../pipelines-baseline <base-commit>`
   - `cd ../pipelines-baseline/frontend && npm ci && npm run start`
2. Capture baseline screenshots from this branch:
   - `cd frontend && npm run visual:baseline -- --base-url http://localhost:3000`
3. Stop the baseline UI, then start the UI from this branch:
   - `cd frontend && npm run start`
   - `cd frontend && npm run visual:current -- --base-url http://localhost:3000`
4. Generate diffs and open the report:
   - `cd frontend && npm run visual:diff`
   - Side-by-side images are written to `frontend/.visual/side-by-side/`
   - (Optional) Open `frontend/.visual/report.html` for the HTML report

Notes:
- Edit `frontend/scripts/visual-compare.routes.json` or pass `--routes <path>` to expand coverage.
- To add more viewports, pass `--viewports "1280x720,768x1024,375x812"`.
- Outputs are written under `frontend/.visual/` and ignored by git.

---

## Known Warnings From Latest Test Runs

- **React.createFactory deprecations** across many UI tests.
  - Source: legacy React 16 + recompose/react-vis usage.
  - Status: deferred (tracked with React/MUI/visualization upgrades).
- **React 16 lifecycle warnings** (`componentWillReceiveProps`) from `re-resizable`.
  - Status: deferred (requires dependency upgrade + React patch bump).
- **Node deprecation warnings** during tests: `punycode`, `url.parse`, `util._extend`, `Buffer()`.
  - Source: transitive dependencies in server tests.
  - Status: deferred to dependency refresh (post-migration).
- **ts-jest config deprecation warnings** during server tests.
  - Source: `frontend/server` Jest runner still uses `ts-jest` (Phase 8 migration).
  - Status: will be removed when server tests move to Vitest.
- **Console error logs in server tests** (auth/minio/aws negative cases).
  - Status: expected in current tests; no failures observed.

4. **React 16 lifecycle warnings** (Low priority)
   - Track with React upgrade initiative
   - Not blocking functionality

5. **Generated code ESM conversion** (Low priority)
   - Only if regeneration is needed for other reasons
   - Current workaround is stable

---

## Summary

This is a high-quality migration that went beyond the original scope to address systemic test instability. Key strengths:

1. **Thorough documentation** - Every decision tracked with rationale
2. **Root cause fixes** - Test flakiness addressed at source, not masked
3. **Forward-looking choices** - Latest Vite/Vitest majors avoid immediate tech debt
4. **Strategic deferral** - React lifecycle warnings correctly pushed to separate initiative

The migration is production-ready. The remaining items (Phase 8, Storybook 7) are improvements, not blockers.

**Verdict:** Approve for merge. Create follow-up tickets for Phase 8 and Storybook evaluation.
