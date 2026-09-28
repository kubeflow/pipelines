# UI modernization performance comparison

The candidate does **not** establish performance parity. In this small native-fixture workload, first content paints earlier, but median largest-contentful paint (LCP) regresses on Runs and Run Details. The empty Compare view has a lower LCP with more layout shift. The initial compressed payload grows. These mixed results leave performance acceptance and workload-specific budgets open; adopting the new presentation stack is not evidence of a speedup.

Application source: `b3f82500fea54d00b65a314f9507ceebe98a0f25`. The [browser matrix](workflows/browser-matrix.json), screenshots and measurements use the same entry assets: `index-ybJg2BvL.js` and `index-EXijewuy.css`. [Machine-readable measurements](workflows/performance-evidence.json) include exact hashes, individual trials, browser performance entries, API readiness assertions and retained DevTools summary excerpts. [Bundle inventory](workflows/bundle-sizes.json) and [browser measurement functions](workflows/performance-protocol.json) accompany the results.

## Method and comparison scope

The [published baseline](https://github.com/kubeflow/pipelines/blob/339670f5e/frontend/docs/ui-modernization/performance-baseline.md) uses application source `02cbc725ac9ddcd950f4400d8355dd78bfcd6c57` and harness `4699581fa9b4e2951105200724aa56e92c841791`. Both builds use unchanged native baseline API fixtures: four available succeeded runs, four pipelines and experiments, native tasks/artifacts for the selected run, and a two-run comparison with no runtime parameters or scalar metrics. Fixture commit fields are fixture data, not the identity of the running candidate.

Settings: HeadlessChrome 154.0.0.0, Chrome DevTools MCP 1.10.1, CPU slowdown 4×, Fast 4G, 1280×720 CSS pixels, DPR 1, light theme, en-US and America/New_York. Builds run through Vite preview with the native mock API over HTTP/1.1 loopback. Node is 24.14.0. Each load uses a uniquely named isolated browser context starting at `about:blank`; tracing begins before the first navigation. Browser process, OS caches and API server remain warm. Task-owned builds, tests and other browser runs were stopped during sampling. This is not a cold process/backend experiment or a production deployment measurement.

Three fresh loads per route verify actual expected names/API 200 responses or the known visible task node, then wait for fonts and two animation frames. FCP comes from browser paint entries; LCP/CLS and lab interaction-to-next-paint (INP) come from DevTools summaries. For the candidate, only summaries and browser entries were retained: raw trace export and raw-event reconciliation were unavailable. The linked baseline retains filtered trace-event extracts and reconciled interaction timings. On Compare, tool-reported LCP is 1–3 ms below browser-reported FCP; the separate reporting paths are not reconciled, so these tiny differences do not establish paint ordering. These are lab observations, not field INP or population percentiles.

## Fresh loads

Values are median (minimum–maximum), in milliseconds except CLS. Each route has three baseline and three candidate samples.

| Route                             |        Baseline FCP |       Candidate FCP |        Baseline LCP |       Candidate LCP | LCP median delta | Baseline → candidate CLS |
| --------------------------------- | ------------------: | ------------------: | ------------------: | ------------------: | ---------------: | -----------------------: |
| Runs                              | 1,836 (1,808–1,856) | 1,748 (1,744–1,752) | 2,002 (1,972–2,053) | 2,266 (2,263–2,270) | +264 ms / +13.2% |              0.01 → 0.02 |
| Run Details                       | 1,736 (1,732–1,748) | 1,676 (1,668–1,680) | 1,912 (1,910–1,919) | 2,323 (2,299–2,606) | +411 ms / +21.5% |              0.00 → 0.01 |
| Compare, empty parameters/metrics | 1,936 (1,888–1,968) | 1,848 (1,812–1,868) | 2,329 (2,300–2,364) | 1,847 (1,809–1,867) | −482 ms / −20.7% |              0.04 → 0.06 |

The Run Details samples have a wider range than Runs; all three exceed the published baseline range. First paint and largest paint measure different milestones, and the redesigned pages need not have the same largest element. The empty comparison does not establish populated-table/chart performance.

Browser-reported total transfer sizes include the document, observed assets and API responses, and differ from offline bundle compression:

| Route       | Baseline transfer bytes | Candidate transfer bytes | Candidate entries including document |
| ----------- | ----------------------: | -----------------------: | -----------------------------------: |
| Runs        |                 962,495 |                1,071,152 |                                   13 |
| Run Details |                 855,865 |                  964,522 |                                   14 |
| Compare     |                 898,575 |                1,007,232 |                                   18 |

These totals are identical across the three samples of each route. Encoded/decoded body sizes are retained separately in the JSON; cached repeated responses can contribute body size with zero transfer size.

## Filtering and navigation

Filtering uses three separately loaded Runs pages and a physical input action. It measures the last `xgboost` input event to one matching row plus two animation frames; the existing 300 ms debounce and request wait are included. Median filter readiness is **728.2 ms (726.1–728.4)** versus baseline **693.8 ms (693.7–695.6)**: +34.4 ms / +5.0%. Reported lab INP is **37 ms (24–67)** versus **45 ms (42–45)**, while filtering CLS increases from **0.00 to 0.03**. The lower median input latency does not remove the readiness or layout-shift regressions.

Three physical run/task navigation trials measure click-handler start to the known task node's presence or visible Task Details controls, then two animation frames. Task-panel readiness does not mean logs/artifacts have all loaded.

| Interaction                                                | Candidate median (range), ms | Baseline exploratory observation, ms |
| ---------------------------------------------------------- | ---------------------------: | -----------------------------------: |
| Open run from Runs                                         |          573.8 (571.5–589.9) |                                589.4 |
| Open task controls                                         |             79.3 (79.0–83.9) |                                121.3 |
| Maximum tool-reported INP across each trace's insight sets |                   93 (90–96) |                                  135 |

The baseline navigation trace is one exploratory observation, not a repeated distribution. The candidate trials extend coverage but do not justify a statistically established navigation improvement. Individual insight-set INPs and API statuses are retained; selecting only the faster insight set would misrepresent the trace.

## Bundle sizes

Per-file offline gzip level 9 and Node Brotli defaults match the baseline method. These are encoded-file comparisons, not observed HTTP transfer sizes.

| Initial entry | Baseline raw bytes | Candidate raw bytes | Baseline gzip bytes | Candidate gzip bytes | Baseline Brotli bytes | Candidate Brotli bytes |
| ------------- | -----------------: | ------------------: | ------------------: | -------------------: | --------------------: | ---------------------: |
| JavaScript    |          2,803,952 |           2,883,416 |             780,136 |              815,796 |               630,132 |                662,163 |
| CSS           |             28,104 |              96,508 |               5,521 |               16,752 |                 4,879 |                 14,667 |

Combined initial JS/CSS gzip grows **46,891 bytes (+6.0%)**. JavaScript gzip grows 4.6%; CSS adds 11,231 gzip bytes. The inventory separates emitted YAML/JSON workers and font formats from entry assets and excludes sourcemaps/analyzer output. Workers are not all necessarily loaded on a first visit. Retained legacy table/run-list branches and presentation dependencies remain in this candidate; their eventual removal requires a new build and fresh measurements rather than a predicted saving.

## Open acceptance gates

- Investigate/reduce the Runs and Run Details LCP regressions, filtering delay/layout shift and payload growth; agree explicit budgets before accepting the cutover.
- Repeat representative large-run/graph and populated-comparison timings with enough trials to assess variability. The 200-task fixture currently proves 201 node/363 directed-edge identities and interaction geometry; its browser checks are not a repeated scalability benchmark.
- Qualify the accepted minimum browser versions, actual Safari/iOS and representative assistive technology. Current-engine functional checks do not establish these results.
- Requalify the final dependency-retirement build and hosted deployment modes, then perform the unchanged-backend upgrade/use/rollback rehearsal.

No framework-wide speed claim, release qualification or live rollback result follows from these measurements.
