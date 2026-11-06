# Run inspection, comparison and search

This historical checkpoint records the inspection, comparison and search slice after [shell and Runs](runs.md), toward the coordinated cutover in [KEP PR #14574](https://github.com/kubeflow/pipelines/pull/14574). It preserved existing data clients, run/task polling, graph mapping and mutations while graph/editor and some viewers still had temporary light boundaries. Those presentation boundaries have since been migrated; see [Workflows and qualification](workflows.md) for the current state and remaining release gates. Screenshots and test totals below describe this earlier checkpoint.

## Implemented behavior at this checkpoint

- Run Details uses the shared toolbar, confirmation and notification contracts. Metadata links preserve encoded pipeline/version IDs. The runtime-task summary uses the complete paginated task response and excludes structural root/DAG/loop records. Missing data is not reported as a global zero count.
- The failure banner selects an actual failed runtime task and opens its Logs tab through the existing `task` query and nested graph mapping. It preserves other query parameters, URL state and browser history. No exit code or Kubernetes event feed is inferred.
- The task inspector retains Input/Output, Task Details and Logs, including artifacts, parameters, pods, volumes and task state history. It is resizable on desktop and becomes a modal below 900px, with keyboard focus trapping and return.
- Log presentation follows light/dark tokens while preserving fixed 15px virtualized rows, horizontal scrolling, bounded overscan, follow-tail pause/resume, executor/artifact/driver source distinctions, retry handling and stale-response protection.
- Comparison retains selected-run order, encoded links, missing/empty/zero distinctions, artifact provenance and rich visualizations. ROC paging/search retains selections outside the displayed option page. Fullscreen visualizations use the shared modal.
- Search is available from the rail, embedded layout and Ctrl/Cmd+K. After a 275ms debounce and at least two characters it requests one page of at most five results from each of Runs, Pipelines and Experiments in the active namespace. Closing, unmounting or changing scope/query aborts old requests; stale results cannot replace the current query. Partial failures, retry, empty results and archived labels are explicit. No history scan or per-row fetch is added.
- Shared controls now include semantic inspection tabs/fields, modal dialogs, alerts, accessible text fields, and a resource-table presentation supporting expansion, radio pickers and selection limits. The existing table controller retains filtering, sorting, paging and preference storage.

## Browser review

![Run Details and task logs in dark mode](https://github.com/kubeflow/pipelines/blob/d9aed93d6184be1f65d36066b40eb92028a2aa25/snapshots/54f78798fea94aecc64006a8296a9473f453d55a/frontend/docs/ui-modernization/inspection/run-details-dark.png)
![Namespace-scoped command search in dark mode](https://github.com/kubeflow/pipelines/blob/d9aed93d6184be1f65d36066b40eb92028a2aa25/snapshots/54f78798fea94aecc64006a8296a9473f453d55a/frontend/docs/ui-modernization/inspection/command-palette-dark.png)

These historical Chromium captures use synthetic fixtures at 1440×900. The white graph shows the temporary boundary at that checkpoint; the [current graph presentation and qualification](workflows.md#graph-editor-and-runtime-contracts) use light/dark tokens.

## Historical verification

Run with the repository-pinned toolchain from `frontend`:

```sh
npm run test:ci
npm run test:bundle
```

The frozen slice passed 141 UI files / 1,802 tests and 29 server files / 1,052 tests with coverage. Formatting, UI/server lint, app/mock TypeScript, React peer compatibility and the production build pass. Existing behavioral tests are retained; snapshot updates cover intended presentation changes.

The production browser suite at this checkpoint had 15 scenarios: startup, six Runs/search checks and eight Run Details checks. It covers nested task links, copied URLs, back/forward navigation, deleted-version read paths, initial/refresh failure recovery, exact-once retry, stale logs, source fallbacks, horizontal/follow-tail behavior, failed-task navigation, desktop/narrow focus, prefixed embedding and namespace changes. The run-inspection harness was wired into the existing `test:bundle` CI entry point. Set `KFP_RUN_DETAILS_SCREENSHOT_DIR` and `KFP_RUNS_SCREENSHOT_DIR` to retain review captures.

The original six run-inspection scenarios also passed against the prior production bundle. For the deterministic five-node/six-edge fixture, old/new graph identities match; maximum node-geometry change was 0.00101px and maximum edge-coordinate change was 0.000718px, within the existing 0.05px jitter tolerance. This is geometry/workflow evidence, not qualification of the future graph redesign or large-graph performance.

The evidence above uses local synthetic data and Chromium. Subsequent comparison, graph and workflow checks and all remaining qualification gates are tracked in [Workflows and qualification](workflows.md); this historical list must not be used as the current completion report. No backend schema, API payload or Python `kfp.local` behavior changes in this slice.
