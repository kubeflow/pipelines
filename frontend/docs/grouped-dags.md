# Inline sub-DAG graphs

Pipeline and run graphs expand sub-DAGs into labeled boxes by default. The header's
chevron collapses or expands only that instance (also accessible with Enter/Space).
Selecting a task, artifact, or group still opens its details. Focused navigation and
existing task deep links remain available; opening a group is no longer required to
see its contents.

- Nested groups, conditions, and loop bodies share the same layout.
- Run graphs show iteration boxes once runtime iteration metadata is available.
  Before then, they show the declarative loop body; a zero-iteration loop is empty.
- Edges between scopes terminate at the group boundary. Exported artifacts remain
  in the containing scope, while internal artifacts stay inside their group.
- Full instance paths identify nested nodes, so reused components and iterations do
  not collide. Details panels receive the original local ID and its owning scope.
- Collapse choices and drag positions survive task polling while the canvas is
  mounted. Toggling a group reflows the graph and clears manual positions. Remounting
  the canvas (including switching tabs or pipeline versions) restores the expanded
  default; preferences are not persisted in the URL or browser storage.
- Large graphs can be zoomed out to 5%. All visible scopes are laid out; there is no
  server-side pagination or lazy loading of expanded groups in this change.
- A malformed child scope displays an inline warning rather than breaking the page.
  Nesting beyond 64 layers is likewise reported instead of recursing indefinitely.

## Reproduce the visual checks

Use the Node/npm versions pinned by `frontend/.nvmrc` and `package.json`.

```sh
cd frontend
npm ci
npm run build:tailwind
npm run storybook -- --ci --no-open
```

In another terminal:

```sh
cd frontend
node scripts/grouped-dag.smoke.mjs
```

The Playwright smoke check exercises the **v2/GroupedDag** stories using the real
`DagCanvas`, verifies keyboard collapse, expansion, selection scope, independent
runtime iterations, and viewport fit, then writes six screenshots to
`.visual/grouped-dags/` (git-ignored). Set `STORYBOOK_URL` to change the server URL or
pass an output directory as the first argument. Install Chromium with
`npx playwright install chromium` if necessary.

The nested-artifact, conditional, and runtime examples use small IR/task fixtures
in `src/data/test/groupedFlow.ts`; runtime statuses are fixture data, not a live
cluster. The nested-loop example uses the existing compiler fixture in
`src/data/test/pipeline_with_loops_and_conditions.yaml`. The exit-handler example
uses a copy of `test_data/sdk_compiled_pipelines/valid/pipeline_as_exit_task.yaml`.
