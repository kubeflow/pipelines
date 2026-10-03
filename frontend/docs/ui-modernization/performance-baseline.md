# UI modernization performance baseline

These controlled browser measurements extend the [existing UI baseline](README.md). They characterize the current application with its native mock fixtures before presentation changes. They do not qualify a deployed backend, a browser support policy or a performance budget.

## Provenance and method

Application source and lockfiles are at [`02cbc725ac9ddcd950f4400d8355dd78bfcd6c57`](https://github.com/kubeflow/pipelines/commit/02cbc725ac9ddcd950f4400d8355dd78bfcd6c57); the accompanying baseline harness revision is `4699581fa`. The production JavaScript, CSS and HTML hashes match [bundle-sizes.json](evidence/bundle-sizes.json). The harness revision changes test configuration, documentation and fixture/capture support; application source is unchanged.

- Browser: Chrome `154.0.0.0`, as reported by the browser user agent; Chrome DevTools MCP `1.10.1`.
- Toolchain: Node `24.14.0`, npm `11.17.0`, Vite `8.0.16`; production build served by Vite preview with the native mock API.
- Host: macOS ARM64. Viewport: 1280 × 720 CSS pixels, device pixel ratio 1; locale `en-US`, timezone `America/New_York`. Normal animation and browser time are retained. This differs from the screenshot baseline's pinned Chromium, UTC timezone and reduced motion.
- Emulation: 4× CPU slowdown and the DevTools **Fast 4G** network preset. These are browser emulation settings, not measurements of a physical device or cellular connection.
- Each load starts in a uniquely named isolated browser context at `about:blank`. Emulation is applied before recording. Tracing starts with `reload: false` and `autoStop: false`, followed by the first navigation to the target. Tracing stops after the route's loaded data is verified.
- Each load sample has a fresh browser context and application session. The browser process, operating-system caches and mock server remain warm. Three samples per route were recorded; summaries retain individual observations and report median/range rather than a single best result.

The trace tool's `reload: true` option visits `about:blank` and then navigates normally to the previous URL; it does not disable browser caching. It is therefore not used to establish a fresh-context first visit. Automation command duration includes tool overhead and is not an application response-time metric.

## Workloads and readiness

| Workload    | Route and required loaded state                                                                                                                                                                                                                   |
| ----------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Runs        | `/#/runs`; all four run-name links are present, including `mock-run-0` and `e0115ac1-0479-4194-a22d-01e65e09a32b`; scoped API requests succeed.                                                                                                   |
| Run details | `/#/runs/details/e0115ac1-0479-4194-a22d-01e65e09a32b`; the DAG contains `task.chicago-taxi-trips-dataset`, and the run/tasks requests succeed.                                                                                                   |
| Compare     | `/#/compare?runlist=mock-run-0,e0115ac1-0479-4194-a22d-01e65e09a32b`; both expected run names and the empty-parameters message are visible, and run/task/experiment requests succeed. The empty message alone is insufficient readiness evidence. |

The [fixed data](../../mock-backend/fixed-data.ts) contains four succeeded, available runs, four pipelines and four experiments. The [native task/artifact fixtures](../../mock-backend/mock-api-middleware.ts) provide three tasks for one run and one Dataset artifact. Comparison has no populated runtime parameters or scalar metrics. These workloads do not stress pagination, large DAGs, large comparisons, running-task polling, log streaming or artifact viewers.

## Results

The [load observations](evidence/performance-loads.json) retain all nine samples, environment settings, network observations, metric definitions and original trace hashes. Times are milliseconds; ranges cover three trials per route.

| Route       | FCP median (range), ms | LCP median (range), ms | Reported CLS, all trials |
| ----------- | ---------------------: | ---------------------: | -----------------------: |
| Runs        |    1,836 (1,808–1,856) |    2,002 (1,972–2,053) |                     0.01 |
| Run details |    1,736 (1,732–1,748) |    1,912 (1,910–1,919) |                     0.00 |
| Compare     |    1,936 (1,888–1,968) |    2,329 (2,300–2,364) |                     0.04 |

CLS is the tool's value rounded to two decimal places; `0.00` does not prove zero layout shift. All observed load requests returned HTTP 200 with no failed trace requests. Browser-reported transfer sizes and body sizes are preserved separately, including repeated Compare requests; those fields alone do not establish why a response transferred fewer bytes.

Three separately reloaded [filter trials](evidence/performance-filter.json), each starting with four loaded runs after a document reload, entered `xgboost` and verified one resulting row. Last matching input event to result readiness measured **693.8 ms median (693.7–695.6 ms)**. Readiness includes the existing 300 ms debounce, request/render time and two animation frames after the matching row appears. The trace tool's separate **observed lab INP was 45 ms median (42–45 ms)**. Result readiness and input responsiveness measure different intervals.

One additional [run/task navigation observation](evidence/performance-navigation.json) measured **589.4 ms** from clicking the filtered run link to the expected DAG node plus two animation frames, then **121.3 ms** from clicking that task to its side-panel controls plus two animation frames. The panel check establishes shell visibility, not loaded logs or the Task Details tab contents. The trace reported **135 ms observed lab INP** across these two clicks. This single observation is exploratory and establishes no distribution or stability guarantee.

Filtered trace event extracts accompany [loads](evidence/performance-load-traces.json.gz), [filtering](evidence/performance-filter-traces.json.gz) and [navigation](evidence/performance-navigation-trace.json.gz). Their JSON manifests describe retained events, omitted data and hashes of the original recordings. These compact extracts support inspection of metric, timing and request observations; they are not complete recordings for replaying all DevTools analyses. No field percentile, performance budget or pass/fail threshold is established.

## Reproduce the measurements

Start the production preview and native mock API using the [baseline commands](README.md#reproduce-tests-and-capture), and verify the emitted asset hashes before comparing results. Apply the browser, viewport, CPU, network, locale and timezone settings above. For each of the three route samples, create a new isolated context on `about:blank`, start a trace without automatic reload/stop, navigate to the target, verify its loaded-state conditions, then stop and save the recording. Allow fonts and two animation frames to settle before capturing browser timing entries. Keep the graph's natural initial framing; the performance samples do not invoke Fit View.

For filter trials, reload Runs between trials and assert an empty filter and four run-name links. Attach temporary browser-side instrumentation before the trusted input action: mark `filter-start` on the input event whose value is `xgboost`; observe the DOM until exactly one run-name link contains `xgboost`; after two animation frames, mark `filter-ready` and measure `filter-results` between the marks. Start tracing, fill the filter, verify the final value and row count, and save the trace. Use browser timing entries, not elapsed automation call time. Reject a trial if an action fails or either state check fails.

For the exploratory navigation trace, start from the loaded one-row filter result. Mark the run link's click event, then mark readiness after `[data-testid="DagCanvas"] .react-flow__node[data-id="task.chicago-taxi-trips-dataset"]` exists and two animation frames pass. Next mark the task's click event, then mark readiness after the panel's `Task Details` control appears and two animation frames pass. Keep these action-to-readiness measures separate from the trace's observed INP. Instrumentation is temporary in the browser and does not change application source.

## Interpretation and follow-up candidates

The production output contains one JavaScript asset of **2,803,952 decoded bytes** and one CSS asset of **28,104 decoded bytes**. Those sizes are not HTTP transfer sizes. [Router](../../src/components/Router.tsx) statically imports the page components, and the [entry point](../../src/index.tsx) imports Tailwind and graph CSS globally. Evaluating route-based loading and which styles are needed initially is a plausible modernization opportunity; any improvement must be measured while preserving deep links and loading behavior.

The recorded resource events identify the stylesheet as render blocking. Evaluate initial CSS delivery alongside JavaScript loading; potential savings require a measured change before becoming an improvement claim.

The [run-list loading path](../../src/pages/RunList.tsx) awaits the runs response, then experiment metadata, then referenced pipeline versions before assigning the displayed rows. This explains the observed runs-to-experiments request chain. Investigate reducing serial waits or rendering independently available information while preserving errors and recovery behavior. Pipeline-version requests are already deduplicated and executed concurrently per unique reference; this is not a request-per-row finding.

Vite preview and the production UI server have different delivery behavior. Installed Vite preview enables gzip compression and serves static assets with ETags and `Cache-Control: no-cache`. The [UI server](../../server/app.ts) uses Express static defaults without configured compression middleware or custom static cache options; its static responses default to `public, max-age=0`, ETag and Last-Modified. An ingress may change compression or caching. The [production HTML handler](../../server/handlers/index-html.ts) also injects runtime deployment flags and, for Kubeflow embedding, the dashboard client script. Preview measurements do not exercise that deployed path.

The [build configuration](../../vite.config.mts) retains relative asset paths, the existing `es2015` target and sourcemaps. This profiling work does not change browser compatibility requirements.

## Qualification still required

- Extend the initial filter and run/task observations to repeated task navigation, selection, populated comparison controls and larger workloads. Preserve the distinction between input responsiveness and asynchronous result readiness.
- Add realistic list, graph and comparison sizes and the pending state/permission fixtures before setting regression budgets.
- Agree a versioned browser floor and validate the supported browser matrix.
- Repeat representative workloads against the actual UI server, embedded and standalone deployments, and a seeded backend with recorded immutable identities.
- Keep field INP distinct from lab interaction observations. A few scripted actions do not establish user-population responsiveness or a field percentile.

CPU/network emulation and the small fixture set define the comparison conditions for this setup; they do not predict all production workloads. LCP and layout shifts describe the observed navigation, while loaded-data checks establish that the intended UI state was reached. Neither alone proves workflow parity.
