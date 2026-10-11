# UI modernization layout stability

The [supported-browser checkpoint](supported-browser-qualification.md) adds actual branded-browser evidence on identical emitted assets. It does not replace the measurements or source identities below.

This report records the layout-stability follow-up at application source `e79f8d423e6b118e5df94815ee2f36f68570a9a9`. It includes the layout changes from `4ee7610b739ef57c54155e89c9f3111adee2118f` and the subsequent short-viewport correction. The [previous performance qualification](performance-qualification.md) retains results for `b358c3b4dc6bcf1b47c2ae399b5e85dfc3df0269`; those observations remain historical and are not measurements of this build. [Workflow coverage](workflows.md) and [browser/accessibility qualification](browser-accessibility-qualification.md) describe the wider migration and its remaining release gates.

The practical question is whether the same requested content and interactions remain at least as responsive while avoiding the observed loading and filtering shifts. A smaller bundle or a successful functional smoke test alone cannot answer that question. The measurements below must be evaluated separately for content readiness, interaction readiness and layout stability; they do not establish an intrinsic framework speedup.

## Confirmed shifts and corrections

- **Runs loading and filtering:** the full Runs page keeps its table footer in place while results change from loading to ten rows, one row, no matches and back. Run columns now have stable widths, with the run-name column taking the remaining space. The table retains horizontal scrolling and its existing selection, sorting, paging and request controls. The fixed page-height treatment is scoped to the full Runs page, so nested resource tables retain their existing page flow.
- **Comparison overview and empty sections:** a comparison already knows the requested run IDs. Its overview reserves that row footprint while the existing reads resolve; it does not fabricate data rows. Loading and genuinely empty parameter/scalar sections share the same minimum placeholder height. Refresh preserves the existing data while pending and replaces it with the completed response, with no change to API or selection semantics.
- **Run summary:** stable columns and a minimum value footprint prevent completed runtime-task counts from moving adjacent fields. The summary switches to two columns below 1240 pixels and one column at 480 pixels, with wrapping permitted for long values. The wider breakpoint leaves room for classic scrollbars beside an expanded navigation rail.
- **Short Run Details viewports:** the taller reserved summary exposed a real regression at 780×437: the Detail panel could shrink to zero height. The run-page wrapper now honors its intrinsic minimum height so the existing outer page can scroll. The graph retains its 380-pixel minimum, and inspector/log scrolling and virtualization are unchanged. Regression checks cover readable, reachable Detail values at both 780×437 and 375×437, followed by opening Graph and task logs and returning to navigation.
- **Sidebar metadata and fonts:** expanded navigation reserves optional version, cluster and project rows while keeping unavailable values absent. Long values may wrap and grow; collapsed navigation has no metadata reservation. The four critical font faces are preloaded through Vite-rewritten relative asset URLs. This changes when those fonts are requested; they remain separate from the initial JS/CSS bundle totals.

These changes preserve the existing namespace, query, mutation and persisted-preference contracts. They do not add a backend state filter, global resource statistics or a data migration. The deferred Runs status-filter dependency remains described in [workflow compatibility notes](workflows.md).

## Measurement method

The original application baseline is `02cbc725ac9ddcd950f4400d8355dd78bfcd6c57`; the candidate is `e79f8d423e6b118e5df94815ee2f36f68570a9a9`. The final [bundle inventory](https://github.com/kubeflow/pipelines/blob/d9aed93d6184be1f65d36066b40eb92028a2aa25/snapshots/54f78798fea94aecc64006a8296a9473f453d55a/frontend/docs/ui-modernization/layout-stability/bundle-inventory.json), [observations](https://github.com/kubeflow/pipelines/blob/d9aed93d6184be1f65d36066b40eb92028a2aa25/snapshots/a058b10903912eb8bd29fc15a24b63b8e2722987/frontend/docs/ui-modernization/layout-stability/performance-evidence.json) and [protocol](https://github.com/kubeflow/pipelines/blob/d9aed93d6184be1f65d36066b40eb92028a2aa25/snapshots/54f78798fea94aecc64006a8296a9473f453d55a/frontend/docs/ui-modernization/layout-stability/protocol.json) identify the emitted assets and exact sampling provenance. Earlier `4ee7610b` inventories and diagnostic observations are not final-build evidence.

Apple M2/16 GiB, macOS 26.6.2/arm64; headless Chrome 154, 1280×720/DPR1/light, CPU 4× slowdown and Fast 4G. All baseline samples were rerun alongside the candidate: three interleaved trials per build for three loads, filtering and run/task navigation, totaling 30 observations. Every trial uses a fresh isolated context; the browser process, OS caches and local native fixture API remain warm. No task-owned builds, tests, browser audits or compression run during sampling. Native fixtures and instrumentation are unchanged from the prior protocol. DevTools summaries and browser entries are retained; raw traces were not exported. These are lab observations, not field performance or deployed-backend load results.

Both builds use the same populated-route and successful-API predicates, followed by two animation frames. Fonts readiness is recorded separately. Runs requires four known links plus experiment metadata; Details requires the known visible task node and run/task reads; Compare requires both selected links, successful reads and the honest empty-parameters state. Filtering includes the existing debounce and ends after the matching result is rendered; run/task navigation ends at the same known graph node and inspector controls. Retained browser layout-shift observations and DevTools summaries preserve their actual capture windows and rounding. Lab interaction-to-next-paint values are not field INP or population percentiles.

Median (minimum–maximum), milliseconds except CLS. Lower is better.

| Route         | Legacy content ready, ms | New content ready, ms | Change |    Legacy observed CLS |       New observed CLS |
| ------------- | -----------------------: | --------------------: | -----: | ---------------------: | ---------------------: |
| Runs          |      2,405 (2,403–2,444) |   1,739 (1,731–1,783) | -27.7% | 0.0085 (0.0085–0.0085) | 0.0000 (0.0000–0.0000) |
| Run Details   |      2,521 (2,506–2,531) |   2,076 (2,074–2,347) | -17.6% | 0.0014 (0.0014–0.0014) | 0.0000 (0.0000–0.0000) |
| Empty Compare |      2,332 (2,332–2,348) |   1,852 (1,850–1,872) | -20.6% | 0.0443 (0.0443–0.0443) | 0.0000 (0.0000–0.0000) |

| Route         |      Legacy LCP, ms |         New LCP, ms |
| ------------- | ------------------: | ------------------: |
| Runs          | 2,023 (2,013–2,036) | 1,740 (1,733–1,784) |
| Run Details   | 1,939 (1,899–1,948) | 1,900 (1,893–2,149) |
| Empty Compare | 2,283 (2,275–2,286) | 1,430 (1,427–1,442) |

| Interaction endpoint              |    Legacy, ms |       New, ms | Change |
| --------------------------------- | ------------: | ------------: | -----: |
| Filter result, including debounce | 746 (736–752) | 547 (547–551) | -26.6% |
| Run click → known task node       | 623 (622–624) | 590 (574–591) |  -5.4% |
| Task click → inspector controls   | 127 (124–135) |    88 (80–88) | -31.2% |

| Interaction trace   | Legacy lab INP, ms | New lab INP, ms | Legacy reported CLS | New reported CLS |
| ------------------- | -----------------: | --------------: | ------------------: | ---------------: |
| Filtering           |         54 (49–55) |      27 (27–27) |    0.00 (0.00–0.00) | 0.01 (0.01–0.01) |
| Run/task navigation |      137 (134–153) |    103 (90–104) |    0.00 (0.00–0.00) | 0.00 (0.00–0.00) |

Load CLS uses retained `LayoutShift` entries through the recorded observer snapshot after fonts readiness, excluding recent input and taking the largest session window (under one second between shifts and under five seconds overall). Full precision is retained in JSON. Interaction CLS and LCP use separately rounded DevTools summaries; lab INP is the maximum insight value per trace. The JSON also retains FCP, fonts-ready, all API/resource entries, interaction marks and observation endpoints. LCP targets differ: the legacy sidebar cluster label versus redesigned run/title content. The separate DevTools and browser paint paths are not reconciled; they do not establish paint ordering.

All nine candidate load observations contain no layout-shift entries. The preceding candidate recorded median observed CLS of 0.023850 on Runs, 0.004198 on Details and 0.061255 on empty Compare; those are historical diagnostic values, not contemporaneous samples. The new matched comparison shows content-readiness medians 27.7%, 17.6% and 20.6% below legacy, respectively. Filtering improves 26.6%, run opening 5.4%, and task opening 31.2%. Details LCP has overlapping ranges and one slower candidate sample; the lower median does not establish a universal improvement.

**Filtering retains a residual difference:** observed CLS is 0.013583 (0.013583–0.013583) for the candidate versus 0.003459 (0.003459–0.003459) for legacy. DevTools rounds these to 0.01 and 0.00. In all three candidate samples, the only shift source is the same keyed matching row moving from y346.5 to y218.5 after its two preceding 64-pixel rows are removed; width, height and horizontal position are unchanged. The debounce plus response/render time puts this motion beyond the recent-input exclusion. Column headers, pagination and sidebar controls remain fixed. Legacy instead records a smaller footer movement. Retaining the matching row and compacting the filtered results is expected behavior; remounting it or delaying presentation merely to suppress CLS is not part of this fix. This residual remains subject to the issue's explicit performance-budget acceptance, rather than being represented as zero shift or an across-the-board performance pass.

The result is improved loading stability with the measured readiness gains preserved. It applies to these fixtures and settings, not every dataset or field percentile. Arbitrary long metadata and unknown populated content may legitimately grow.

## Representative larger workloads

Three fresh-context samples each on the same final application, using Playwright Chromium 145 without CPU/network throttling and warm local route fixtures. Host-side timings include automation overhead, so these are separate from the native comparison and do not claim a baseline speedup or scale ceiling. Retained [graph evidence](https://github.com/kubeflow/pipelines/blob/d9aed93d6184be1f65d36066b40eb92028a2aa25/snapshots/54f78798fea94aecc64006a8296a9473f453d55a/frontend/docs/ui-modernization/layout-stability/graph-chromium.json) covers 200 executable tasks, 201 rendered nodes, 363 edges, two task pages and settled geometry. [Comparison evidence](https://github.com/kubeflow/pipelines/blob/d9aed93d6184be1f65d36066b40eb92028a2aa25/snapshots/54f78798fea94aecc64006a8296a9473f453d55a/frontend/docs/ui-modernization/layout-stability/comparison-chromium.json) covers two runs, typed zero/false/empty parameters, scalar zero, 111 classification artifacts, 100/11 paging, selected provenance and real curve expansion. Both records retain fixture, harness, index and loaded-asset hashes.

| Endpoint                            | Median (minimum–maximum), ms |
| ----------------------------------- | ---------------------------: |
| 200-task graph ready                |                628 (619–654) |
| Task activation → inspector         |                   61 (60–65) |
| Fit View → settled changed viewport |                114 (113–114) |
| Populated comparison ready          |                209 (208–336) |
| Classification tab → three curves   |                150 (133–150) |
| Off-page selection → four curves    |                   71 (71–72) |
| Expand → larger rendered chart      |                   94 (92–94) |

## Verification and asset identity

At application source `e79f8d423e6b118e5df94815ee2f36f68570a9a9`, UI coverage passes **1,756 tests in 133 files**. Formatting, UI/server lint, application/mock TypeScript, React peer checks, repository hooks, production build and Storybook build pass. This slice does not change the Express server or dependency manifests/lockfiles; the retained **1,052 server tests** remain attributed to their previously recorded source rather than described as a new server-suite run.

The production matrix adds three named scenarios to the previous 44-per-engine inventory:

1. [Runs](../../scripts/ui-modernization-runs.smoke.mjs): held initial data and metadata; stable columns, footer and sidebar controls through ten/one/zero/ten results; exact result IDs and page-token reset; collapsed/narrow navigation.
2. [Run Details](../../scripts/ui-modernization-run-details.smoke.mjs): held task data, stable summary geometry when counts arrive, exactly one task request, and containment at 1200 and 375 pixels.
3. [Comparison](../../scripts/ui-modernization-comparison.smoke.mjs): held initial reads and refresh, stable overview/section geometry, honest empty panels, retained selection/order and query contracts, and completed-response content replacing the previous names.

An existing Run Details scenario is also strengthened with nonzero Detail-panel height and native scroll/hit-testing at 780×437 and 375×437. It retains its semantic-value, graph/log and navigation assertions; it does not add another case.

The [final production browser matrix](https://github.com/kubeflow/pipelines/blob/d9aed93d6184be1f65d36066b40eb92028a2aa25/snapshots/54f78798fea94aecc64006a8296a9473f453d55a/frontend/docs/ui-modernization/layout-stability/browser-matrix.json) passes **141/141** cases: 47/47 in chromium 145.0.7632.6, 47/47 in firefox 146.0.1, 47/47 in webkit 26.0. Its asset hashes match the measured build. Fresh reviewed screenshots include [Runs light](https://github.com/kubeflow/pipelines/blob/d9aed93d6184be1f65d36066b40eb92028a2aa25/snapshots/54f78798fea94aecc64006a8296a9473f453d55a/frontend/docs/ui-modernization/layout-stability/runs-light.png), [Runs dark](https://github.com/kubeflow/pipelines/blob/d9aed93d6184be1f65d36066b40eb92028a2aa25/snapshots/54f78798fea94aecc64006a8296a9473f453d55a/frontend/docs/ui-modernization/layout-stability/runs-dark.png), [Run Details graph](https://github.com/kubeflow/pipelines/blob/d9aed93d6184be1f65d36066b40eb92028a2aa25/snapshots/54f78798fea94aecc64006a8296a9473f453d55a/frontend/docs/ui-modernization/layout-stability/run-details-graph.png), [populated comparison](https://github.com/kubeflow/pipelines/blob/d9aed93d6184be1f65d36066b40eb92028a2aa25/snapshots/54f78798fea94aecc64006a8296a9473f453d55a/frontend/docs/ui-modernization/layout-stability/comparison-light.png) and [narrow inspector](https://github.com/kubeflow/pipelines/blob/d9aed93d6184be1f65d36066b40eb92028a2aa25/snapshots/54f78798fea94aecc64006a8296a9473f453d55a/frontend/docs/ui-modernization/layout-stability/run-details-narrow-inspector.png).

Entry JS/CSS only; offline per-file gzip level 9 and Node-default Brotli. Fonts, deferred editor/workers and other emitted files are itemized separately in the inventory; these are not HTTP transfer sizes.

| Build                                | Raw bytes | gzip bytes | Brotli bytes |
| ------------------------------------ | --------: | ---------: | -----------: |
| Original legacy baseline             | 2,832,056 |    785,657 |      635,011 |
| Prior qualified candidate `b358c3b4` | 2,141,724 |    596,193 |      488,701 |
| Layout-stable candidate `e79f8d42`   | 2,143,872 |    596,673 |      489,101 |

Initial gzip remains **24.1% below legacy** and **20.9% below retirement**, adding 480 bytes relative to the prior qualification. The four preloaded fonts were already used by these routes; the Runs regression asserts four distinct font URLs and exactly four requests. The pass adds no dependency or data request.

The earlier actual Firefox 128 seven-scenario run and hosted frontend/four Kubernetes integration lanes passed on `b358c3b4`, as recorded in [browser/accessibility qualification](browser-accessibility-qualification.md). The scoped Lighthouse snapshots retain their individual `a2ea3e85` and `b80da474` source/asset identities, accessibility score 100 and reviewed experimental warnings. None of those prior results is relabeled as an `e79f8d42` rerun.

Fresh hosted qualification at `e79f8d42` also passes [hosted frontend checks](https://github.com/kubeflow/pipelines/actions/runs/36487388896/job/109147489374), [Kubernetes 1.33.12 TLS on](https://github.com/kubeflow/pipelines/actions/runs/36487389231/job/109147490404), [Kubernetes 1.33.12 TLS off](https://github.com/kubeflow/pipelines/actions/runs/36487389231/job/109147490570), [Kubernetes 1.36.1 TLS on](https://github.com/kubeflow/pipelines/actions/runs/36487389231/job/109147490786), [Kubernetes 1.36.1 TLS off](https://github.com/kubeflow/pipelines/actions/runs/36487389231/job/109147490431). The [retained check snapshot](https://github.com/kubeflow/pipelines/blob/d9aed93d6184be1f65d36066b40eb92028a2aa25/snapshots/54f78798fea94aecc64006a8296a9473f453d55a/frontend/docs/ui-modernization/layout-stability/hosted-checks.json) also records successful pre-commit, DCO and aggregate CI. This does not close deployment-specific authorization or rollback gates.

## Remaining acceptance criteria

- Agree workload-specific performance and layout-stability budgets and explicitly disposition the final measurements; a low absolute CLS or a faster readiness median alone does not close every criterion.
- Retain the distinction between deterministic fixture coverage and real-backend performance. The graph/comparison records above have their own workload scope; editor-first-open latency remains unmeasured.
- Complete actual supported-browser and device checks under the [approved browser policy](../browser-support.md). Policy acceptance is complete; the recorded Playwright engines do not qualify all newly declared stable/enterprise channels or annual Safari/iOS releases.
- Complete real assistive-technology/device review. Scoped automated Lighthouse, forced-colors and keyboard checks do not establish blanket WCAG conformance.
- Qualify the remaining standalone/embedded authorization and deployment modes, and rehearse previous UI → candidate UI → previous UI against the same compatible backend using immutable image identities and retained data/preferences/signing state.

KFP Local requires no migration for this UI slice; no SDK, pipeline-format, backend API or persisted-data change is introduced. The frontend image still includes Express, so static-bundle evidence alone does not complete deployment or rollback qualification. [Issue #14572](https://github.com/kubeflow/pipelines/issues/14572) tracks these open gates alongside [KEP PR #14574](https://github.com/kubeflow/pipelines/pull/14574) and [implementation PR #14584](https://github.com/kubeflow/pipelines/pull/14584).
