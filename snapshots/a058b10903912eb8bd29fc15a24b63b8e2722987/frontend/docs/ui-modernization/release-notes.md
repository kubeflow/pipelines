# UI modernization: release and migration notes

These notes prepare [PR #14584](https://github.com/kubeflow/pipelines/pull/14584)
for review. They do not announce a published release. The
[tracking issue](https://github.com/kubeflow/pipelines/issues/14572) owns the design,
compatibility requirements and current release checklist; no KEP is required.
Source-specific results are retained in the [rebased qualification report and dated matrix](release-qualification-2026-10-05-rebased/README.md).
Subsequent harness changes and eventual release-image checks must be assessed separately.

## What changes for users

The UI adopts a consistent light/dark theme, responsive navigation, keyboard
command search, resource tables, run/task inspection and themed pipeline graphs.
Pipelines, experiments, recurring runs, creation forms, artifacts, comparison and
rich viewers use the same shared controls. Existing resource URLs, namespace
selection, mutations, paging, sorting and persisted table preferences remain part
of the compatibility contract. The theme adds the `kfp.theme` preference; existing
preferences are not destructively migrated.

The YAML/JSON editor loads on demand. The production server now negotiates gzip
for precompressed JavaScript, CSS and editor workers, with original-file fallback.
This reduces transfer cost without loading the editor on the main pages. In the
matched October 5 hosted experiment, median first-editor display fell from
3,376 to 1,399 ms and worker readiness from 3,725 to 1,493 ms. Entry JS/CSS gzip
remained 596,806 bytes, with identical original asset bytes. These are controlled
lab results; deployments that already compress assets at ingress can see smaller
gains. See the qualification report for network/CPU conditions and all budgets.

The existing name and archive filters remain available. The proposed new Runs
status selector is deferred until backend filtering matches the effective status
shown for historical runs, including paused and canceled states. Global status
counts and the proposed 48-hour failure summary are omitted: a fetched page or
bounded recent-run sample is not a global total. These are explicit product-scope
follow-ups, not features advertised by this release. See
[workflow compatibility](workflows.md#runs-status-filter-compatibility-dependency).

## Browser requirements

Use the [approved browser policy](../browser-support.md): the latest two stable
major releases of desktop Chrome, Edge and Firefox; current Chrome/Edge Extended
Stable; current Firefox ESR and the outgoing ESR during its supported overlap;
and current/previous annual Safari on supported macOS and iOS/iPadOS, at latest
available patches. Older browsers are no longer qualification targets, although
there is no user-agent block. Conservative compiler targets are not a promise to
support every older browser that can parse the bundle.

Publish a dated exact browser/OS matrix for the release. Engine tests do not
substitute for branded Safari, enterprise channels or devices. The
[hosted coverage report](hosted-browser-qualification.md) and machine-readable
[catalog](../../scripts/qualified-browsers.json) identify tested scope and missing
slots. A successful test of one patch does not qualify a later vendor patch.

## Operator migration and rollback

This is a coordinated replacement of the frontend application and its Express
server. There is no backend API, database, pipeline-format or artifact migration.
No alternate legacy-renderer switch remains. Upgrade the UI image against the
same compatible backend using the deployment's existing configuration.

Before rollout, retain immutable previous/candidate UI image identities and the
existing deployment configuration. Preserve root and `/pipeline/` entry points,
hash routes, namespace context, runtime flags, service ports, probes, security
context and the TensorBoard signing key. Preserve authentication and dashboard
integration for multi-user installations. The backend must remain unchanged for
a UI-only rollback.

Ship the generated `.gz` sidecars with their original assets. The server selects
the representation through `Accept-Encoding` and sets `Vary: Accept-Encoding`;
intermediary caches must preserve that distinction. Do not rewrite compressed
bytes or strip their encoding header. Identity fallback remains available.
HTML, API responses and artifact/proxy responses are outside this static-asset
compression change.

The existing `Recreate` rollout strategy interrupts UI availability during both
upgrade and rollback. Hosted rehearsals observed roughly 38–43 seconds; this is
not a maximum downtime guarantee. Plan a maintenance window rather than assuming
zero downtime. To roll back, restore the recorded compatible previous UI image
and configuration, preserving backend data, browser preferences, authentication
and signing state. Recheck list/create, task/log/artifact access and viewer links
through the actual entry path. Restore the UI only; no data repair or preference
reset should be necessary. The retained standalone and authenticated multi-user
rehearsals document those invariants and their tested image identities.

## KFP Local Considerations

Python `kfp.local` execution, local output artifacts, compilation and SDK behavior
are unchanged. No local SDK upgrade, recompilation, artifact conversion or local
state migration is required by this UI change. A local cluster serving the KFP UI
uses the same browser and frontend-image requirements as any other installation;
that deployment is distinct from Python local execution.

## Release-readiness disposition

| Item                                                                                                                  | Disposition and reason                                                                                                                                                                                                                                                                                                   |
| --------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Current-head frontend, repository formatting, performance and deployment/rollback checks                              | Required before calling the implementation qualified. Historical passing results retain their own source identities and do not turn an unfinished or failed current check green.                                                                                                                                         |
| Supported-browser failures affecting load, core workflows, data integrity, keyboard access or reachable content       | Release blockers under the approved policy. Fix and rerun the affected lane; do not mask failures or substitute another engine.                                                                                                                                                                                          |
| Missing exact versions, enterprise distributions and unavailable device/OS environments                               | Explicit coverage follow-ups under the approved CI-only plan; they remain unqualified. They do not add a release blocker under that disposition, but prevent claiming the complete browser policy is verified. This does not excuse a failure in an available configured lane or narrow the support policy.              |
| Keyboard/focus, contrast, reduced motion, forced colors, dialogs and responsive/zoom behavior exercised by automation | Preserve passing assertions and source-specific evidence; failures affecting access to core content block release. Simulated conditions must be labeled.                                                                                                                                                                 |
| Untested screen-reader, speech-control, physical-device, IME and predictive-input behavior                            | Disclosed limitations and automated-coverage follow-ups. The agreed plan does not require manual sessions and makes no blanket WCAG or assistive-technology conformance claim. Known core accessibility defects still block release.                                                                                     |
| BrowserStack access                                                                                                   | Investigate exact inventory, CI integration, eligibility and retention. This is a route to close coverage gaps, not a prerequisite product dependency or authorization to purchase/apply for access.                                                                                                                     |
| Status selector and unavailable aggregate statistics                                                                  | Explicit scope deferrals with the compatibility/data rationale above. Do not show guessed counts or change historical run-state semantics to close a design checkbox.                                                                                                                                                    |
| Downstream deployment variants and final published images                                                             | The retained standalone and authenticated dashboard rehearsals establish their recorded scope only. Release owners must validate the actual published image identities and supported deployment/architecture contract through the ordinary release process; untested downstream customization is not silently qualified. |

### BrowserStack investigation

The [open-source program](https://www.browserstack.com/open-source) advertises
Automate and Percy, five users, five parallel sessions and project-lifetime
membership. Official documentation describes
[GitHub Actions integration](https://www.browserstack.com/docs/automate/playwright/github-actions)
and [real iOS Safari automation](https://www.browserstack.com/docs/automate/playwright/playwright-ios/nodejs).
These establish a feasible CI path, not confirmed KFP eligibility or exact device
and browser inventory. Follow up on eligibility, required enterprise/Safari
versions, physical device availability, fixture tunneling and artifact retention
before proposing integration. No account application, activation or purchase has
been made; pending access is not qualified coverage.

After review and all required gates, the release owner must link these notes and
the dated tested matrix from the eventual release notes, verify the published
images using the [release process](../../../release/README.md), and retain results.
Preparing these notes does not merge a PR, publish images or deploy a release.
