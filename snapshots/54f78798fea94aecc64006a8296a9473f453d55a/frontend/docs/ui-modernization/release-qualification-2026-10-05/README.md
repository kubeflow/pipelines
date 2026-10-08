# October 5 cold editor optimization

Large generated reports are preserved in the [evidence archive](../evidence-archive.md).
Original hash manifests apply to the extracted archive.

Build-time gzip sidecars reduce transfer cost without changing the lazy-loading
boundary or original application code. Express negotiates gzip for public static
JavaScript/CSS, preserving identity fallback, MIME types, conditional responses,
and prefixed asset URLs. No dependency or editor prefetch is added. HTML, APIs,
proxies, user artifacts and source maps are outside the compression handler.

The tested implementation is `ac310c997da668c012f30ac493b0e9ac1ab2f955`, tested as
merge `1db95b6e2fec3899c6a5bf6a54976c19208e14da` with base
`e3c93651a44fa718ab079010bb332318854e0491`. The immutable previous modern build is
`439304aa4ed3a20ae5a7c004b96936cb65c3aff0`. The publication follow-up adds evidence
and CI trigger coverage only; it changes no application or measurement code.

## Matched performance

[Hosted run 37264911581](https://github.com/kubeflow/pipelines/actions/runs/37264911581)
passes all 29 acceptance comparisons. Seven samples per build/case rotate all six
legacy/previous/candidate orders: 126 matched observations plus 28 larger-workload
observations against the immutable scaling checkpoint. Conditions remain 4× CPU,
150 ms latency, 1.6 Mbps download, 0.75 Mbps upload, 1280×720/DPR 1, fresh browser
contexts and warm browser/runner caches. Large-workload sampling is unthrottled.

| Median | Previous modern | Candidate | Reduction |
| --- | ---: | ---: | ---: |
| Cold editor display | 3,376.4 ms | 1,399.1 ms | 58.6% |
| Initialized YAML worker | 3,724.6 ms | 1,493.3 ms | 59.9% |
| Runs readiness | 13,485.5 ms | 5,516.1 ms | 59.1% |
| Run Details readiness | 13,582.7 ms | 5,628.2 ms | 58.6% |
| Compare readiness | 13,690.8 ms | 5,688.1 ms | 58.5% |

Filtering, run opening, task opening and larger workloads also pass their existing
relative budgets. Entry JS/CSS gzip remains **596,806 bytes**, unchanged from the
previous modern build and 24.0% below original legacy. All six original JS/CSS
files are byte-identical between previous and candidate; the six added gzip files
decode exactly to those originals. Editor JS transfers **143,418 bytes instead of
531,379**; its YAML worker transfers **23,481 instead of 76,346**. The sole source-map
difference is two generated Vite worker URL placeholder IDs, with no emitted-code
change. All 21 candidate initial-load observations have zero measured CLS; filter
CLS remains 0.013583 from verified row compaction with controls within one pixel.

The independent audit recalculates all 29 comparisons and checks 126 raw trace
hashes and parsed contents. It cross-checks 294 static transfers against build
inventories and trace response headers/body lengths, with no cache or service-worker
substitution. Editor trace endpoints match the reported measurements. All 35
candidate non-editor samples request no editor or worker assets.

These gains measure actual gzip delivery by the production static handler against
previous builds without gzip sidecars, under the same hosted conditions. They are
not a framework speedup or a field-performance guarantee. Deployments whose ingress
already compresses these assets may see a smaller incremental improvement. The
legacy eager editor still displays sooner (331.1 ms) because its code loaded with
the main page; its missing YAML worker has no equivalent worker-ready timing.
The candidate retains lazy loading and both accepted 4-second display/5-second
worker limits; no acceptance threshold was weakened.

## Behavior and deployment

[Frontend run 37264911489](https://github.com/kubeflow/pipelines/actions/runs/37264911489)
passes 1,763 UI tests, 1,183 server tests and 432 browser checks across nine
Windows/macOS/Linux and Chromium/Firefox/WebKit combinations. The server coverage
includes gzip negotiation, identity fallback, headers, validators, HEAD/ranges,
prefixed paths and exclusions. All nine browser lanes verify the same 28 assets.

[Deployment run 37264911764](https://github.com/kubeflow/pipelines/actions/runs/37264911764)
passes all 18 standalone and 21 authenticated multi-user checks through
legacy → candidate → restored legacy. Independent audits verify image/index/config
identity, asset hashes and sidecar decoding, unchanged backend resources/pods,
preserved authentication/signing state, data/preferences and exact rollback.
Recreate interruptions remain approximately 38–43 seconds; this optimization does
not change the deployment strategy. Deployment reports validate decoded asset bytes;
wire encoding is separately verified by the performance run and server HTTP tests.

[Broader browser run 37264911580](https://github.com/kubeflow/pipelines/actions/runs/37264911580)
passes 14 of 17 lanes, including desktop Safari 26/27 and iPhone 27. iPad 27 hits
Appium startup failure, iPhone 26 cannot connect Safari's debugger, and iPad 26
fails native keyboard dismissal. These remain explicit mobile automation gaps.
The unrelated whole-PR pre-commit gate fails on `isort` in
`.github/resources/scripts/update_go_version.py`, whose bytes match the base.
All changed-file hooks and the 18 trigger-inventory tests pass.

## Retained evidence

The raw performance observations, effective protocol, static and independent
performance audits, browser audit/status and sanitized deployment records are
retained here. `record-hashes.json` covers all retained files after formatting;
raw runtime-manifest digests are preserved exactly. Full Chromium traces, build
artifacts, screenshots and logs remain available from Actions for seven days.
Retained hashes establish identity but cannot retrieve expired artifact bytes.
The earlier [October 2 qualification](../release-qualification-2026-10-02/README.md)
remains historical evidence. Issue #14572 is authoritative; no KEP is required.
