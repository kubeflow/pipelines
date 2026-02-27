# Migration Lessons

## 2026-02-27

- Failure mode: `SidePanel` crashed in tests/runtime because `Slide` could not call `getBoundingClientRect` on a non-DOM child (`Resizable` wrapper).
  - Detection signal: `TypeError: node.getBoundingClientRect is not a function` from MUI transition flow.
  - Prevention rule: For MUI transition components (`Slide`, `Grow`, etc.), ensure the direct child resolves to a DOM node or a ref-forwarding component that exposes a DOM element.

- Failure mode: Existing tests used brittle selectors that changed after MUI v5 migration (e.g., `button` role queries and `title` attributes on selects).
  - Detection signal: multiple frontend tests failed to locate controls after migration despite unchanged user behavior.
  - Prevention rule: Prefer semantic and stable queries (`combobox`, `aria-label`, visible text) over implementation-driven selectors (`title`, legacy role assumptions).

- Failure mode: Four long-running UI tests intermittently exceeded the default 5s timeout in coverage mode after migration.
  - Detection signal: `Test timed out in 5000ms` in `CompareV2`, `NewPipelineVersion`, `NewRun`, and `NewRunV2` during `npm run test:ci`.
  - Prevention rule: Add explicit per-test timeout for known long flows and validate under full CI coverage mode before marking migration complete.
