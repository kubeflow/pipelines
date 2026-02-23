# React 18 and 19 Upgrade Plan (Frontend)

Last updated: 2026-02-23

Purpose: executable PR-by-PR plan to move frontend React from 17 -> 18 and then 18 -> 19 while keeping dependency support explicit and auditable.

## Current baseline
- React stack in `frontend/package.json` is still on 17.x (`react`, `react-dom`, `@types/react`, `@types/react-dom`, `react-test-renderer`).
- App bootstraps via `ReactDOM.render` in `frontend/src/index.tsx`.
- Multiple direct dependencies currently declare React-17-limited peer ranges (for example: MUI v4, Storybook 6.3, `react-flow-renderer@9`, `react-query@3`).

## Open PR coverage map (as of 2026-02-23)
- Covered now:
  - #12856: removes remaining `react-dom/test-utils` imports and `snapshot-diff`.
  - #12858: upgrades `re-resizable` warning source (`SidePanel`) and adds regression coverage.
  - #12855: adds DOM nesting warning guard coverage for paged table path.
  - #12872: removes dead dependencies and lockfile noise (hygiene benefit).
- Not covered by open PRs:
  - React peer-compatibility gate (this PR-B).
  - Storybook major upgrade.
  - `react-query` -> `@tanstack/react-query`.
  - `react-flow-renderer` -> `@xyflow/react`.
  - MUI v4 -> MUI (`@mui/*`) migration.
  - React core bump to 18 and later 19.
  - JSX runtime modernization and post-upgrade strict-mode hardening.

## PR-B (this change): lockfile React peer compatibility gate
- Added script: `frontend/scripts/check-react-peers.mjs`.
- Added npm commands:
  - `npm run check:react-peers` (target 17, enforced now).
  - `npm run check:react-peers:18`.
  - `npm run check:react-peers:19`.
- Wired into:
  - `frontend/package.json` `test:ci` (which is already invoked by `.github/workflows/frontend.yml`).
- Policy:
  - Keep target at 17 until 17->18 migration starts.
  - Flip CI command from `check:react-peers` to `check:react-peers:18` in the React 18 core bump PR.
  - Flip CI command to `check:react-peers:19` in the React 19 core bump PR.

## Phase 1: React 17 -> 18 (stacked PRs)
1. Prereq cleanup merge window
- Merge and rebase around #12855, #12856, #12858, and optionally #12872.
- Goal: reduce warning/test noise before dependency migrations.

2. Storybook modernization
- Upgrade Storybook from 6.3 stack to a React-18/19-supported major.
- Keep this isolated because Storybook migration has its own breaking changes.

3. Query layer migration
- Replace `react-query` v3 with `@tanstack/react-query` (v5 target).
- Update imports/usages and test wrappers.

4. Flow graph migration
- Replace `react-flow-renderer` v9 with `@xyflow/react`.
- Update graph code, stories, and dependent tests.

5. MUI migration
- Migrate `@material-ui/*` v4 usage to `@mui/*`.
- Update theme/styling code (`frontend/src/Css.tsx` and related files).

6. JSX/test runtime modernization
- Move to modern JSX runtime configuration.
- Remove/replace any remaining `react-test-renderer` usage.

7. React 18 core bump
- Update `react`, `react-dom`, and type packages to 18-compatible versions.
- Replace `ReactDOM.render` with `createRoot` in `frontend/src/index.tsx`.
- Switch React peer gate to target 18.

8. React 18 stabilization
- Fix regressions and flaky tests under React 18.
- Keep strict mode enablement as a follow-up hardening step.

## Phase 2: React 18 -> 19 (stacked PRs)
1. React 18 latest checkpoint
- Move to latest React 18.x first and clear deprecation warnings.

2. Dependency compatibility sweep for React 19
- Run `npm run check:react-peers:19`.
- Upgrade any remaining deps that still exclude 19.

3. React 19 core bump
- Update `react`, `react-dom`, type packages, and test renderer dependencies as needed.
- Run React 19 codemods and deprecation cleanup.
- Switch React peer gate target to 19.

4. Strict mode hardening (dev/test)
- Enable strict mode in development/test tree and resolve double-invoke side effects.
- Keep production behavior unchanged unless explicitly needed.

## Verification checklist for each PR
- `cd frontend && npm run format:check`
- `cd frontend && npm run lint`
- `cd frontend && npm run typecheck`
- `cd frontend && npm run check:react-peers` (or `:18` / `:19` for target phase)
- `cd frontend && npm run test:ci`
- `cd frontend && npm run build`
- `cd frontend && npm run build:storybook`
