# UI modernization performance comparison

**Historical retirement checkpoint:** newer application source, repeated measurements and compatibility checks are recorded in [performance qualification](performance-qualification.md) and [browser/accessibility qualification](browser-accessibility-qualification.md). The results below remain tied to their original source and assets.

Retiring the unused presentation code reduces the initial compressed payload below the original UI baseline and improves the measured load medians relative to the [pre-retirement candidate](https://github.com/kubeflow/pipelines/blob/5008b24d93387dba1a2c71d79f3756ad67fee10f/frontend/docs/ui-modernization/performance-comparison.md). It does **not** establish overall performance parity: Runs and Run Details LCP remain slower than baseline, filtering readiness remains slower, and layout shifts remain higher. Performance acceptance and workload-specific budgets remain open.

Application source: `2e52a3fa634fa0805511a37c982f8d13f690b868`. The [browser matrix](workflows/browser-matrix.json), reviewed screenshots and measurements identify the same entry assets: `index-CsUTqcBF.js` and `index-8lXTuxuW.css`. [Machine-readable measurements](https://github.com/jeffspahr/jeffspahr-pipelines/blob/a058b10903912eb8bd29fc15a24b63b8e2722987/frontend/docs/ui-modernization/workflows/performance-evidence.json) retain hashes, individual trials, browser performance entries, readiness assertions and DevTools summary excerpts. [Bundle inventory](workflows/bundle-sizes.json) and [measurement functions](workflows/performance-protocol.json) accompany the results.

## Method and comparison scope

The [published baseline](https://github.com/kubeflow/pipelines/blob/339670f5e/frontend/docs/ui-modernization/performance-baseline.md) uses application source `02cbc725ac9ddcd950f4400d8355dd78bfcd6c57` and harness `4699581fa9b4e2951105200724aa56e92c841791`. Both builds use unchanged native baseline API fixtures: four available succeeded runs, four pipelines and experiments, native tasks/artifacts for the selected run, and a two-run comparison with no runtime parameters or scalar metrics. Fixture commit fields are fixture data, not running source identities.

Settings: HeadlessChrome 154.0.0.0, Chrome DevTools MCP 1.10.1, CPU slowdown 4×, Fast 4G, 1280×720 CSS pixels, DPR 1, light theme, en-US and America/New_York; Node 24.14.0. Vite preview serves production assets with the native mock API over HTTP/1.1 loopback. Each load starts in a unique isolated context at `about:blank`, with tracing before its first navigation. Browser process, OS caches and API server remain warm. Task-owned builds, tests and other browser runs were stopped during sampling. This is not a cold process/backend or production deployment experiment.

Three loads per route verify expected names/API 200 responses or the known visible task node, then wait for fonts and two animation frames. FCP comes from browser paint entries; LCP/CLS and lab interaction-to-next-paint (INP) come from DevTools summaries. Candidate raw trace export/reconciliation was unavailable; only summaries and browser entries were retained. The baseline retains filtered trace extracts and reconciled interaction timings. On Compare, reported LCP is 1–2 ms below browser FCP; these separate reporting paths are not reconciled and do not establish paint ordering. These are lab observations, not field INP or population percentiles.

## Fresh loads

Median (minimum–maximum), milliseconds except CLS; three samples per route and build.

| Route | Baseline FCP | Retired candidate FCP | Baseline LCP | Retired candidate LCP | LCP median delta | Baseline → candidate CLS |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Runs | 1,836 (1,808–1,856) | 1,616 (1,604–1,692) | 2,002 (1,972–2,053) | 2,130 (2,107–2,148) | +6.4% | 0.01 → 0.02 |
| Run Details | 1,736 (1,732–1,748) | 1,556 (1,548–1,564) | 1,912 (1,910–1,919) | 2,190 (2,182–2,212) | +14.5% | 0.00 → 0.01 |
| Compare, empty parameters/metrics | 1,936 (1,888–1,968) | 1,648 (1,640–1,660) | 2,329 (2,300–2,364) | 1,646 (1,639–1,659) | -29.3% | 0.04 → 0.06 |

The preceding candidate's median LCP was 2,266 ms for Runs, 2,323 ms for Run Details and 1,847 ms for empty Compare. Retirement reduces those medians to 2,130, 2,190 and 1,646 ms respectively, but the Runs and Details samples still exceed their baseline ranges. The largest painted elements can differ after a redesign. Empty comparison results do not establish populated-table/chart performance.

Browser-reported transfers include the document, observed assets and API responses; they differ from offline compression:

| Route | Baseline transfer bytes | Candidate transfer bytes | Candidate entries including document |
| --- | ---: | ---: | ---: |
| Runs | 962,495 | 992,572 | 13 |
| Run Details | 855,865 | 885,942 | 14 |
| Compare, empty parameters/metrics | 898,575 | 928,652 | 18 |

Totals are identical within each route's three samples. Encoded/decoded body sizes are retained separately; cached repeated responses can contribute body size with zero transfer size. A smaller offline JS/CSS payload does not by itself guarantee lower total browser transfers, which include fonts and other assets.

## Filtering and navigation

Three separately loaded Runs pages receive physical filter input. Readiness runs from the last `xgboost` input event to one matching row plus two animation frames, including the existing 300 ms debounce and request wait. Median readiness is **725.5 (724.2–727.9) ms**, versus baseline **693.8 (693.7–695.6) ms** (+4.6%). Lab INP is **28 (27–32) ms**, versus **45 (42–45) ms**; filtering CLS remains **0.03**, versus **0.00**. Lower input latency does not remove the readiness or layout-shift regressions.

Three physical run/task navigation trials measure click-handler start to the known task node or visible Task Details controls, then two animation frames. Task readiness does not imply loaded logs/artifacts.

| Interaction | Candidate median (range), ms | Baseline exploratory observation, ms |
| --- | ---: | ---: |
| Open run from Runs | 567.2 (565.8–567.8) | 589.4 |
| Open task controls | 93.7 (76.0–95.8) | 121.3 |
| Maximum reported INP across each trace's insight sets | 110 (93–111) | 135 |

The baseline navigation trace is one exploratory observation starting from a list filtered to one run; these three candidate trials start from four unfiltered runs. The starting states therefore differ, and the baseline is not a repeated distribution, so this does not establish a statistically reliable navigation improvement. Individual insight-set INPs and API statuses are retained; the maximum is used for each trace.

## Bundle sizes

Per-file offline gzip level 9 and Node Brotli defaults match the baseline method. These are encoded-file sizes, not observed HTTP transfers.

| Entry | Baseline raw bytes | Candidate raw bytes | Baseline gzip bytes | Candidate gzip bytes | Baseline Brotli bytes | Candidate Brotli bytes |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| JavaScript | 2,803,952 | 2,631,014 | 780,136 | 737,377 | 630,132 | 600,695 |
| CSS | 28,104 | 96,222 | 5,521 | 16,704 | 4,879 | 14,643 |

Combined initial JS/CSS gzip is **754,081 bytes**, down **31,576 bytes (4.0%)** from baseline and **78,467 bytes (9.4%)** from the pre-retirement candidate. JavaScript gzip is 5.5% below baseline; CSS remains 11,183 gzip bytes larger. The lockfile removes 74 packages with no retained version changes. There are no remaining MUI/Emotion/typestyle dependencies or application imports. Inventory separates emitted editor workers and font formats from entry assets, excluding sourcemaps/analyzer output; not every worker is loaded on first visit.

## Open acceptance gates

- Reduce or explicitly accept the remaining Runs/Details LCP, filtering-readiness and layout-shift regressions against agreed budgets.
- Repeat representative large-run/graph and populated-comparison timings with enough trials to assess variability. The 200-task functional fixture is not a repeated scalability benchmark.
- Qualify accepted minimum browser versions, actual Safari/iOS and representative assistive technology.
- Complete final-source hosted deployment qualification and the unchanged-backend upgrade/use/rollback rehearsal with immutable UI/Express images and retained data/preferences/signing key.

These measurements do not establish a framework-wide speedup, release readiness or live rollback success.
