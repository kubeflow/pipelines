# October 2 release qualification

Large generated reports are preserved in the [evidence archive](../evidence-archive.md).
Original hash manifests apply to the extracted archive.

## Performance and first editor acceptance

[Hosted performance run 37064291277](https://github.com/kubeflow/pipelines/actions/runs/37064291277)
passed at implementation head `71543eb922a82b8716cb58f139618052866eece7`, tested as
merge `00d243a9d92888c2746a7084bec45f36850ced5f`. The merge parents independently
verify the implementation head and base `463f15ff9df3be4b5720a3d9473d033a19509a65`.

The first editor gates are an engineering acceptance decision: bound cold lazy
loading while preserving the smaller initial bundle. Both use seven-sample
medians under 4× CPU throttling, 150 ms network latency, 1.6 Mbps download and
0.75 Mbps upload, at 1280×720/DPR 1. Fresh browser contexts retain warm browser
process and runner file/OS caches. These are laboratory fixture measurements,
not deployed backend performance or field percentiles.

| Measurement                                                          | Candidate median | Blocking limit |
| -------------------------------------------------------------------- | ---------------: | -------------: |
| First editor display: complete read-only model, fonts and two frames |       3,354.3 ms |       4,000 ms |
| First initialized YAML worker: complete model round trip             |       3,706.3 ms |       5,000 ms |

The same-run eager legacy editor displays in **291.6 ms**. The candidate's cold
first open remains slower; this accepted tradeoff moves editor code off the
initial route. The immutable legacy build omits its YAML worker, so its recorded
HTTP 404 cannot support a worker-ready timing comparison. Candidate worker HTTP
200, initialization and exact model hashes are required in every sample.

All other existing blocking budgets also pass:

- Main-route median readiness improves 21.3–22.5% against the same-run legacy
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

[Hosted deployment run 37064291781](https://github.com/kubeflow/pipelines/actions/runs/37064291781)
passes both standalone and real authenticated embedded multi-user rehearsals at
implementation head `71543eb922a82b8716cb58f139618052866eece7`, tested merge
`00d243a9d92888c2746a7084bec45f36850ced5f`. Standalone passes all 18 checks and
multi-user all 21, with no skipped phases. Each exercises legacy → candidate →
restored legacy UI/Express images against one unchanged candidate backend.

Independent verification covers immutable image/config identities, exact runtime
index and child-manifest hashes, all 6 legacy and 22 candidate asset files,
unchanged backend/RBAC/authentication/signing state, preserved resources and
preferences, successful new runs/logs, schedules, artifacts and old signed
TensorBoard access. Rollback restores the exact captured legacy UI template.
Downloaded artifact archive hashes also match GitHub's published digests.

Multi-user additionally verifies real Dex/OAuth2 sessions, three namespace-change
checks with prior responses fully drained, normal cross-user denial, six explicit
SubjectAccessReview allow/deny cases, and seven mesh workloads. Canonical profile
roles, owner bindings and mesh setup finish before baseline; these remain
unchanged during UI replacement and restoration. This does not establish CNI
NetworkPolicy enforcement or every downstream deployment/storage configuration.

| Mode | Upgrade recovery upper bound | Rollback recovery upper bound |
| --- | ---: | ---: |
| Standalone | 43.114 s | 36.473 s |
| Embedded multi-user | 42.154 s | 40.709 s |

The Recreate transitions produce a measured interruption; this is not a
zero-downtime claim. Recovery is observed by 200 ms polling with a 2 s request
deadline. Raw availability observations are retained.

Sanitized raw records, source/job provenance, asset and image manifests, runtime
manifest bytes, phase invariants and independent audits live in
`deployment-standalone/` and `deployment-multiuser/`. They exclude credentials,
browser storage, private authentication snapshots and signed URLs. Full static
builds and screenshots remain in the seven-day Actions artifacts.

The two gates covered here—editor acceptance and deployment/rollback
qualification—are complete. Browser/provider/device gaps and release delivery
remain separate work tracked in issue #14572; no KEP approval is required.
