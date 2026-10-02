# October 2 release qualification

## Performance and first editor acceptance

[Hosted performance run 37057281124](https://github.com/kubeflow/pipelines/actions/runs/37057281124)
passed at implementation head `998115570cd27b2b6a2b67bf4093e9a746c09017`, tested as
merge `295cf388a0d0845a0ffb34367d8a85ce9c2e80f8`. The merge parents independently
verify the implementation head and base `0a34391b09303c03eae17116c5003e62b9d9f881`.

The first editor gates are an engineering acceptance decision: bound cold lazy
loading while preserving the smaller initial bundle. Both use seven-sample
medians under 4× CPU throttling, 150 ms network latency, 1.6 Mbps download and
0.75 Mbps upload, at 1280×720/DPR 1. Fresh browser contexts retain warm browser
process and runner file/OS caches. These are laboratory fixture measurements,
not deployed backend performance or field percentiles.

| Measurement                                                          | Candidate median | Blocking limit |
| -------------------------------------------------------------------- | ---------------: | -------------: |
| First editor display: complete read-only model, fonts and two frames |       3,352.5 ms |       4,000 ms |
| First initialized YAML worker: complete model round trip             |       3,688.0 ms |       5,000 ms |

The same-run eager legacy editor displays in **275.6 ms**. The candidate's cold
first open remains slower; this accepted tradeoff moves editor code off the
initial route. The immutable legacy build omits its YAML worker, so its recorded
HTTP 404 cannot support a worker-ready timing comparison. Candidate worker HTTP
200, initialization and exact model hashes are required in every sample.

All other existing blocking budgets also pass:

- Main-route median readiness improves 21.2–22.4% against the same-run legacy
  build. All route, interaction and larger-workload timing medians remain within
  their reference plus max(10%, 50 ms).
- Entry JS/CSS gzip is 596,806 bytes, 24.0% below legacy and within the 80% limit.
- All 21 candidate initial-load observations have zero measured CLS.
- All seven filter observations have CLS 0.013583, below 0.02. Independent
  geometry checks verify upward compaction of the matching result row and
  controls stable within one CSS pixel.

The independent audit verifies 84 matched observations, 28 larger-workload
observations, all 84 raw trace hashes and parsed trace contents, and all 50
emitted files across the three builds. It independently recalculates timing
medians/limits, CLS session windows, filter geometry and entry gzip sizes (using
the source-pinned Node runtime). Source, effective protocol, measurement harness,
fixture equality for larger workloads and emitted asset identities are checked.

`performance.json` retains the raw observations, source/build identities,
per-trace hashes and all comparisons. `hosted-protocol.json` retains the effective
protocol. `performance-audit.json` records the independent verification results.
`record-hashes.json` verifies these retained records. The reports use test fixture
data and contain no deployment credentials.

GitHub Actions retains the larger raw traces and static build artifacts for
**seven days**. The committed hashes establish identity of downloaded files;
they do not make expired artifacts retrievable. The sanitized records above
remain available in the repository after artifact expiration.

## Deployment and rollback

Standalone qualification passes in [run 37060216956](https://github.com/kubeflow/pipelines/actions/runs/37060216956),
implementation head `aa925258a3ae78faca5541cb7957570b1ea9526e`, tested merge
`2388b2182a881e236f2f959e2f5f304f7f289d5e`. All 18 checks pass through legacy,
candidate and restored legacy phases. Independent verification covers immutable
image/config identities, exact runtime index and child-manifest hashes, static
assets, unchanged backend/RBAC/signing state, preserved resources/preferences,
old signed TensorBoard access, and exact restoration of the captured UI template.

The Recreate transitions produce a measured interruption; recovery upper bounds
are 42.613 seconds for upgrade and 38.064 seconds for rollback. This is not a
zero-downtime claim. Sanitized records and their independent audit are retained
in `deployment-standalone/`; full static builds remain in the seven-day artifact.

Authenticated multi-user qualification remains pending. Standalone success does
not establish its namespace isolation or authorization behavior. Its result and
verified state evidence will be added after the hosted rehearsal completes.
