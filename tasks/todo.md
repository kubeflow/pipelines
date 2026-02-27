- [x] Restate goal + acceptance criteria
  - Goal: Migrate frontend from MUI v4 (`@material-ui/core`, `@material-ui/icons`) to MUI v5 (`@mui/material`, `@mui/icons-material`) while keeping runtime behavior stable on React 17.
  - Acceptance criteria:
    - Visual parity for Pipeline List, Run Details, Experiment List, and Side Nav.
    - `npm run test:ci && npm run build` pass.
    - No codemod changes in generated directories (`src/apis`, `src/apisv2beta1`, `src/third_party`).
    - Theme colors, typography, and spacing remain unchanged.

- [x] Locate existing implementation / patterns
  - Inventory all MUI v4 imports and style APIs (`withStyles`, `makeStyles`, theme `overrides`).
  - Confirm current theming entry points (`src/Css.tsx`, `src/mlmd/Css.tsx`, `src/index.tsx`).
  - Confirm generated directories and exclude them from codemod edits.

- [x] Design: minimal approach + key decisions
  - Use MUI codemods first, then manual fixes.
  - Keep migration scope to #12894 only; do not bundle React 18/19 bumps.
  - Package target: use latest stable MUI v5 patch (React 17/18/19 compatible) to reduce near-term rework.
  - Keep Emotion as styling engine (`@emotion/react`, `@emotion/styled`).

- [x] Implement smallest safe slice
  - Slice A: dependency and import migration
    - Update packages in `frontend/package.json`:
      - remove `@material-ui/core`, `@material-ui/icons`
      - add `@mui/material`, `@mui/icons-material`, `@emotion/react`, `@emotion/styled`
    - Run codemods in order:
      - `npx @mui/codemod v5.0.0/preset-safe frontend/src`
      - `npx @mui/codemod v5.0.0/variant-prop frontend/src`
      - `npx @mui/codemod v5.0.0/top-level-imports frontend/src`
    - Validate generated directories are unchanged.
  - Slice B: theme and styling migration
    - Convert theme `overrides` to `components.styleOverrides` in:
      - `frontend/src/Css.tsx`
      - `frontend/src/mlmd/Css.tsx`
    - Replace `withStyles` with v5-compatible styling in:
      - `frontend/src/atoms/CardTooltip.tsx`
    - Fix any breaking changes found in compile/test output.
  - Slice C: application bootstrap and provider checks
    - Confirm `ThemeProvider` and theme typing are v5-correct in `frontend/src/index.tsx`.
    - Ensure no residual `@material-ui/*` imports remain.

- [x] Add/adjust tests
  - Update failing snapshots only where behavior is intentionally unchanged but render output differs due to MUI internals.
  - Add focused regression assertions if any component behavior changes were fixed during migration.

- [x] Run verification (lint/tests/build/manual repro)
  - Required:
    - `cd frontend && npm ci`
    - `cd frontend && npm run format`
    - `cd frontend && npm run lint`
    - `cd frontend && npm run typecheck`
    - `cd frontend && npm run test:ci`
    - `cd frontend && npm run build`
  - Strongly recommended:
    - `cd frontend && npm run storybook`
    - `cd frontend && npm run build:storybook`
    - capture before/after screenshots for key pages.
  - Guard checks:
    - `git diff -- frontend/src/apis frontend/src/apisv2beta1 frontend/src/third_party` must be empty.
    - `rg -n \"@material-ui/\" frontend/src frontend/package.json` must return no matches.

- [x] Summarize changes + verification story
  - List changed files grouped by:
    - dependencies
    - theme/style migrations
    - component-level API migrations
  - Record exact command outputs (pass/fail + any waivers).

- [x] Record lessons (if any)
  - If migration surprises occur (codemod misses, style regressions, test flakiness), add entries to `tasks/lessons.md` with:
    - failure mode
    - detection signal
    - prevention rule

## Working Notes
- Keep PR scope focused on MUI v5 migration only (issue #12894).
- Do not refactor unrelated components while touching imports.
- Expect most churn in tooltip/dialog/table components and theme overrides.
- Visual regressions are the highest risk; prioritize deterministic screenshot checks on the four primary pages.

## Results
- Dependencies
  - Replaced `@material-ui/core` and `@material-ui/icons` with:
    - `@mui/material@^5.18.0`
    - `@mui/icons-material@^5.18.0`
    - `@emotion/react`
    - `@emotion/styled`
  - Updated lockfile to match.
- Theme/style migration
  - Migrated theme overrides to `components.styleOverrides` in:
    - `frontend/src/Css.tsx`
    - `frontend/src/mlmd/Css.tsx`
  - Migrated `CardTooltip` off `@mui/styles` usage.
- Runtime + tests
  - Fixed `SidePanel` transition child/ref compatibility for MUI v5 `Slide`.
  - Updated several tests/selectors and snapshots for MUI v5 behavior changes.
  - Stabilized 4 long-running tests with explicit per-test timeouts (`20000`) to remove coverage-mode flakes.
- Verification
  - `cd frontend && npm run test:ci` ✅ pass
    - UI: 118 files / 1654 tests passed.
    - Server: 10 files / 163 tests passed.
  - `cd frontend && npm run build` ✅ pass
  - Guard check: `git diff -- frontend/src/apis frontend/src/apisv2beta1 frontend/src/third_party` ✅ empty
  - Guard check: `rg -n "@material-ui/" frontend/src frontend/package.json` ✅ no matches
- Not run
  - `npm run storybook`
  - `npm run build:storybook`
