# Performance qualification

Latest follow-up: [loading-layout stability](layout-stability.md) records the `e79f8d42` fixes, 141-case browser matrix and repeated matched measurements. The source-specific results below remain historical evidence.

This report supersedes the timing conclusions in the [retirement checkpoint](performance-comparison.md), while retaining that checkpoint as historical evidence. It compares original baseline source `02cbc725ac9ddcd950f4400d8355dd78bfcd6c57` with candidate `b358c3b4dc6bcf1b47c2ae399b5e85dfc3df0269` under a matched protocol. [Browser/accessibility qualification](browser-accessibility-qualification.md) records separate coverage and remaining release gates.

## Changes and diagnosis

The old sidebar cluster label was the largest-painted element on Runs and Details; the redesigned page paints a run ID/title as LCP. LCP is retained, but it is not an equivalent requested-content endpoint across these layouts. The new protocol also measures the same populated route and successful API responses in both builds, with instrumentation installed before navigation.

The candidate defers Ace and its modes until an editor is opened, starts independent experiment metadata alongside the Runs request, and reserves the breadcrumb/header footprint while Details and Compare metadata load. Editor code is deferred, not eliminated: first editor-open latency is outside these timings. Request generations and error handling remain covered; no API or persisted-data migration is introduced.

The first post-audit measurements detected a new header regression: removing an unnamed breadcrumb also removed the pending header, moving Details content down 108 pixels when it appeared. Those observations were retained as diagnosis, not combined with the corrected candidate's results. Explicit pending-route header space preserves semantics and loading geometry; held-response browser assertions cover both routes. Runs still moves the table footer when loading rows become available; sidebar metadata and late fonts can also move content. Remaining shifts and budget acceptance are reported below.

## Native-fixture method

Apple M2 with 16 GiB RAM, macOS 26.6.2/arm64. Headless Chrome 154, viewport 1280×720/DPR1/light, CPU 4× slowdown and Fast 4G. Three trials per route and interaction per build, interleaved baseline/candidate; each uses a fresh isolated context. The browser process, OS caches and local native fixture API remain warm. No concurrent task-owned application builds, test suites, browser audits or compression runs during sampling. These are laboratory observations, not field INP, population percentiles or deployed-backend performance.

For Runs, readiness requires the four known row links and successful experiment metadata. Details requires its known visible task node and successful run/task APIs. Compare requires its two selected run links, successful requests and the honest empty-parameters state. Content readiness ends two animation frames after that shared predicate; fonts-ready adds `document.fonts.ready` and two frames. The same observer runs in both builds. This comparison fixture has empty parameter/scalar panels; the populated comparison has separate timings below.

Filtering measures the final input event through the single matching row and two frames, including the existing 300 ms debounce. Run navigation starts with the same four unfiltered rows in both builds and ends at the known graph node. Task navigation ends at the inspector Task Details controls, not loaded logs. DevTools trace summaries and browser entries are retained; raw traces were not exported. Reported lab INP is the maximum insight value per recorded interaction, not a field metric.

Median (minimum–maximum), milliseconds except CLS; three samples per build/case. Lower is better. All individual values, browser entries and API readiness checks are in [the retained observations](https://github.com/jeffspahr/jeffspahr-pipelines/blob/a058b10903912eb8bd29fc15a24b63b8e2722987/frontend/docs/ui-modernization/qualification/performance-evidence.json); [the protocol](qualification/protocol.json) retains the instrumentation.

| Route         | Baseline content ready | Candidate content ready | Median change |
| ------------- | ---------------------: | ----------------------: | ------------: |
| Runs          |    2,387 (2,381–2,408) |     1,756 (1,748–1,759) |        -26.4% |
| Run Details   |    2,538 (2,490–2,544) |     2,155 (2,138–2,170) |        -15.1% |
| Empty Compare |    2,320 (2,315–2,329) |     1,899 (1,894–1,931) |        -18.2% |

| Route         |        Baseline LCP |       Candidate LCP |  Baseline observed CLS | Candidate observed CLS |
| ------------- | ------------------: | ------------------: | ---------------------: | ---------------------: |
| Runs          | 2,009 (2,005–2,009) | 1,758 (1,750–1,761) | 0.0085 (0.0085–0.0085) | 0.0239 (0.0239–0.0239) |
| Run Details   | 1,940 (1,912–1,970) | 1,956 (1,954–1,974) | 0.0014 (0.0014–0.0014) | 0.0042 (0.0042–0.0042) |
| Empty Compare | 2,276 (2,267–2,279) | 1,402 (1,397–1,414) | 0.0443 (0.0443–0.0443) | 0.0613 (0.0613–0.0613) |

LCP uses DevTools summary values. The table’s CLS uses retained `LayoutShift` observations through the recorded observer snapshot after fonts readiness, excluding recent input and taking the largest session window (under 1 second between shifts and under 5 seconds overall). Separately rounded DevTools CLS values remain in the JSON; no raw-trace reconciliation is claimed. FCP, fonts-ready and transfer sizes are retained alongside these endpoints.

| Interaction endpoint              |      Baseline |     Candidate | Median change |
| --------------------------------- | ------------: | ------------: | ------------: |
| Filter result, including debounce | 738 (737–748) | 545 (544–566) |        -26.1% |
| Run click → known task node       | 575 (575–577) | 544 (542–557) |         -5.4% |
| Task click → inspector controls   | 117 (111–121) |    75 (72–78) |        -35.6% |

| Lab interaction-to-next-paint          |      Baseline |  Candidate |
| -------------------------------------- | ------------: | ---------: |
| Filtering                              |    49 (42–64) | 25 (23–26) |
| Run/task navigation, maximum per trace | 127 (123–132) | 89 (84–90) |

Filtering reported CLS is 0.00 (0.00–0.00) on the baseline and 0.03 (0.03–0.03) on the candidate. Input paint, result readiness and layout stability remain separate acceptance criteria.

The matched content-readiness medians improve on all three routes, as do filtering and run/task opening. Details LCP remains slightly higher, with overlapping sample ranges; LCP targets differ between layouts. Observed CLS remains higher on all three routes, and filtering CLS remains higher. These are separate criteria rather than an overall pass/fail score. DevTools reports empty-Compare LCP a few milliseconds below browser FCP; the independent reporting paths are not reconciled and do not establish paint ordering.

## Representative larger workloads

These separate candidate-only runs use Playwright Chromium, a fresh context per trial, a warm browser process and local route fixtures, with **no CPU or network throttling**. Host-side timings include automation overhead. They are not directly comparable with the native-fixture table above or a baseline speedup. Each retained record identifies source, harness, fixture, index and loaded asset hashes.

| Workload endpoint                           | Median (minimum–maximum), ms |
| ------------------------------------------- | ---------------------------: |
| 200-task graph ready                        |                624 (595–630) |
| Task keyboard activation → inspector ready  |                   62 (62–78) |
| Fit View → settled changed viewport         |                114 (113–114) |
| Populated comparison tables ready           |                209 (194–333) |
| Classification tab → three rendered curves  |                133 (133–148) |
| Off-page selection → four curves/provenance |                   70 (55–87) |
| Expand → larger rendered chart              |                   94 (93–95) |

The graph-only timing harness follow-up `e54a5a4f4` changes the setup to Zoom In because the initial large graph is already at minimum zoom. The retained three samples follow that correction and use the unchanged `b358c3b4` application assets; the graph record retains the exact harness hash. The failed setup attempt produced no retained graph timing result. The populated-comparison harness is unchanged from `b358c3b4`.

The [graph fixture](qualification/graph-chromium.json) contains 200 executable tasks plus a structural root, 201 rendered nodes including an artifact, 363 edges and two task-list pages. Readiness requires complete identities, stable world geometry and fonts; selection uses the real task button; Fit View must change and settle the viewport. The [comparison fixture](qualification/comparison-chromium.json) contains two runs, typed zero/false/empty parameters, scalar zero and 111 classification artifacts. It proves 100/11 option pagination, retained off-page selection/provenance, actual three→four curve rendering and chart expansion. This is representative client coverage, not a measured scale ceiling or backend load test.

## Bundle and acceptance

Entry JS/CSS only; offline per-file gzip level 9 and Node Brotli defaults, not HTTP transfer sizes. Fonts, deferred editor/workers and other assets remain itemized in [the complete bundle inventory](qualification/bundle-sizes.json).

| Build                 | Raw bytes | gzip bytes | Brotli bytes |
| --------------------- | --------: | ---------: | -----------: |
| Original baseline     | 2,832,056 |    785,657 |      635,011 |
| Retirement checkpoint | 2,727,236 |    754,081 |      615,338 |
| Qualified candidate   | 2,141,724 |    596,193 |      488,701 |

Initial JS/CSS gzip changes by **-24.1% against the original baseline** and **-20.9% against retirement**. No native load sample requested the lazy Editor chunk. This result combines presentation retirement, deferred editor code and other implementation changes; it does not isolate an intrinsic framework effect.

These observations qualify the recorded fixtures and build. They do not establish a maximum supported graph size, rich-comparison capacity, a performance SLA, or an across-the-board framework speedup. Maintainers still need agreed budgets and explicit disposition of remaining shifts/regressions. Actual minimum-browser/device, assistive-technology, deployment authorization and same-backend rollback gates remain in #14572.
