# React 17 Upgrade Checklist (Frontend)

Last updated: 2026-02-15

Purpose: capture a minimal, auditable checklist for upgrading the frontend from React 16 to React 17 while keeping risk low and changes shippable.

## Prerequisites (must be true before upgrading)
- PR #12754 (Vite/Vitest migration) merged into the target branch.
- PR #12756 (frontend server ESM + dev workflow) merged into the target branch.

If either prerequisite is not true in the target branch, stop and rebase before proceeding.

Current status on 2026-02-15:
- PR #12754 merged on 2026-02-09.
- PR #12756 merged on 2026-02-06.

## Baseline Snapshot (fill in before starting)
- React core: `react`, `react-dom`, `@types/react`, `@types/react-dom`, `react-test-renderer` versions.
- Build/dev stack: confirm Vite/Vitest scripts are present and `react-scripts` is not used for client builds.
- Test stack: jest config, `@testing-library/*`, Enzyme + adapter version, jsdom environment.
- Storybook: version and preset used.
- Node version: `.nvmrc`.
- Frontend server build: confirm ESM entrypoint and build output expectations.

Suggested quick checks:
- `rg -n "react-scripts|vite|vitest" frontend/package.json`
- `rg -n "enzyme|adapter" frontend/package.json`
- `rg -n "componentWill|UNSAFE_|findDOMNode" frontend/src`

## Compatibility Audit (before changing React)
Dependencies
- Identify packages with React 16-only peer ranges and plan upgrades/replacements.
- Specifically review: `enzyme-adapter-react-16`, `@types/enzyme-adapter-react-16`, Storybook React preset, `@material-ui/core` v3, `react-virtualized`, `react-vis`, `react-flow-renderer`, `react-dropzone`, `react-ace`, and `react-query`.

Code patterns
- Legacy lifecycles: `componentWillMount`, `componentWillReceiveProps`, `componentWillUpdate` and `UNSAFE_` variants.
- `findDOMNode` usage.
- Assumptions about SyntheticEvent pooling.
- Tests relying on `react-dom/test-utils` behaviors.

## Verification Commands (baseline and after each slice)
- `npm run lint`
- `npm run test`
- `npm run build`
- `npm run test:server:coverage` (only if server changes)
- `npm run format:check` (if formatting touched)

## Notes
- Keep each change small and independently shippable.
- Prefer dependency updates that are React 17-compatible while still running on React 16 until the core bump is done.
