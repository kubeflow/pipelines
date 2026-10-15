# Shell integration and Runs

This historical checkpoint records the first shell and Active/Archived Runs integration after the [foundation](foundation.md), in [PR #14584](https://github.com/kubeflow/pipelines/pull/14584) for [issue #14572](https://github.com/kubeflow/pipelines/issues/14572). Other routes still used temporary light presentation at this point. See [Workflows and qualification](workflows.md) for their completed presentation migration and current release gates. The screenshots and totals below belong to this earlier slice.

## Behavior and design at this checkpoint

- Retain hash routes, encoded IDs, legacy execution redirects, browser history and page-owned state. Run Details keeps its query-driven task selection; namespace changes retain the existing list reset behavior without resetting the shell.
- Adapt existing build/GKE/namespace contexts and deployment flags to the shell. Marketplace keeps Getting Started, shared pipelines remain deployment-dependent, and embedded Runs exposes theme selection when `HIDE_SIDENAV` hides the rail. Preserve existing navigation preference storage and hosted browser-test selectors.
- Present Active and Archived views at their existing URLs. A row opens its run; its checkbox selects it. Keyboard users can operate the checkbox or run link independently. Related pipeline, experiment and recurring-run links retain their destinations; deleted versions remain distinguishable from request failures.
- Keep the existing `RunList` API/enrichment path and `CustomTable` paging/filter/sort controller. An optional render adapter supplies the semantic table only to the migrated views; experiment/comparison consumers retain their current presentation.
- Preserve Compare (2–10 runs), Clone (one run), Archive, Restore, Delete and Refresh through the existing action callbacks. Confirmations retain cancellation, error details and partial-success behavior. Notifications pause for focus, hover and inactive windows. Modal portals use the resolved theme and return keyboard focus to their trigger.
- Keep name-only server filtering, sorting, token pagination, page-size preferences and selected IDs. Filtering starts at the first-page cursor; superseded requests cannot replace newer rows, errors or paging state. Failed reads retain available rows and show explicit failure copy; successful recovery clears the page banner.
- Render every current runtime state with text and color, including Canceling, Paused, Skipped and Unknown. Contrast checks cover normal, hovered and selected row surfaces in both themes.

The handoff's aggregate/status counts, recent-failure summary and history bars are omitted because this request path deliberately uses `skip_count=true`. The filter says **Filter runs by name**; it does not claim pipeline/experiment search. At this checkpoint, status chips, command search, notification history and additional workflows were follow-up items. Subsequent delivered scope and deliberate omissions are tracked in the [current report](workflows.md). No new data aggregation or per-row request path is introduced; existing version enrichment stays deduplicated by pipeline/version pair.

## Browser evidence

The deterministic [production browser harness](../../scripts/ui-modernization-runs.smoke.mjs) serves the emitted bundle through Playwright request interception. Each scenario gets fresh fixture state: 18 runs across available/archived storage states, mixed runtime states, token pagination, related resources and simulated failures. Requests stay on the fixture origin; unexpected endpoints fail the test.

Its five scenarios exercise:

1. Paging, filtering from page two, sorting, page-size changes, visible selection and one lookup per distinct referenced version.
2. Keyboard selection, independent row/resource-link navigation, compare/clone destinations and selection preservation on refresh.
3. Archive cancellation/confirmation, exactly-once mutations, partial forbidden writes with failed selection retained, restore and delete.
4. Failed reads and recovery, light/dark presentation, initial/trapped/restored modal focus, and narrow layout overflow.
5. An iframe under `/pipelines/`, namespace propagation/switching, selection reset and theme selection with hidden navigation.

These Chromium screenshots use synthetic fixtures at 1440×900. Finite transitions are completed during capture.

![Active Runs in light theme](https://github.com/kubeflow/pipelines/blob/d9aed93d6184be1f65d36066b40eb92028a2aa25/snapshots/54f78798fea94aecc64006a8296a9473f453d55a/frontend/docs/ui-modernization/runs/runs-light.png)
![Active Runs in dark theme](https://github.com/kubeflow/pipelines/blob/d9aed93d6184be1f65d36066b40eb92028a2aa25/snapshots/54f78798fea94aecc64006a8296a9473f453d55a/frontend/docs/ui-modernization/runs/runs-dark.png)
![Archive confirmation in dark theme](https://github.com/kubeflow/pipelines/blob/d9aed93d6184be1f65d36066b40eb92028a2aa25/snapshots/54f78798fea94aecc64006a8296a9473f453d55a/frontend/docs/ui-modernization/runs/runs-dark-archive-dialog.png)

## Historical verification

On Node 24.14.0 and npm 11.17.0, `test:ci` passed: 137 UI files / 1,770 tests and 29 server files / 1,052 tests with coverage, formatting, UI/server lint, TypeScript and React peer checks. Production and Storybook builds, the production startup smoke and all five Runs browser scenarios also passed. The token suite at this checkpoint verified 54 contrast pairs.

Use the pinned Node/npm versions and run from `frontend`:

```sh
npm run test:ci
npm run test:bundle
npm run build:storybook
```

At this checkpoint `test:bundle` built the application and ran the startup and Runs browser tests serially; the existing frontend CI job installs Chromium and executes this command. Set `KFP_RUNS_SCREENSHOT_DIR` when running the Runs harness to retain its three screenshots. Storybook includes **Modernization → Runs table** light/dark, loading and empty examples.

The full application browser harness verifies UI/API contracts against controlled responses. It does not establish real cluster authentication, authorization, ingress behavior or a supported minimum browser version. The hosted frontend integration specs retain their real cluster assertions with updated semantic selectors.

The [current report](workflows.md#final-qualification-checklist) supersedes this checkpoint’s remaining-work list. Before cutover, qualify deployment parity and the declared browser floor, compare representative performance against [baseline PR #14576](https://github.com/kubeflow/pipelines/pull/14576), and rehearse upgrade/use/rollback with identified compatible UI/backend images. This slice changes no backend API or schema and no Python `kfp.local` behavior or artifacts. The temporary mixed presentation is not a second released theme or a rollback mechanism.
