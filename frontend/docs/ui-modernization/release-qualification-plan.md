# Remaining release qualification

Broad qualification tooling is maintained in [qualification PR #14757](https://github.com/kubeflow/pipelines/pull/14757). Commands and results in this document refer to that tooling or the explicitly recorded historical source; they are not additional presubmit gates in the application PR. See [the evidence index](evidence-archive.md) for retained records.

Status: hosted deployment/rollback and editor qualification passed October 2, 2026.
See [retained results](https://github.com/kubeflow/pipelines/blob/d9aed93d6184be1f65d36066b40eb92028a2aa25/snapshots/54f78798fea94aecc64006a8296a9473f453d55a/frontend/docs/ui-modernization/release-qualification-2026-10-02/README.md) for exact sources,
measurements, image/state proofs and remaining coverage limits.
The maintainer accepted timing, payload and layout budgets on September 30.
The separate first-editor limits below were adopted on October 2 to complete
editor qualification; the slower cold-open tradeoff remains explicit.
They are unchanged acceptance criteria, not field-percentile claims. Unavailable provider/device rows remain explicit
follow-ups under the existing CI-only delivery plan; they do not count as qualified.

## Qualification cadence and promotion

Required frontend presubmits retain unit/server tests, lint, type checks, the
production build and Linux Chromium. Broader browser coverage, performance and
deployment/rollback are provided by [qualification PR #14757](https://github.com/kubeflow/pipelines/pull/14757). After that PR lands, they run weekly and by manual dispatch. The application PR alone does not install these workflows.

These three qualification workflows are observational and nonblocking for PRs
and releases. Run them against the intended release candidate for advisory
evidence; do not treat historical success as current-candidate qualification.
[Issue #14754](https://github.com/kubeflow/pipelines/issues/14754) tracks flake
investigation, repeatability, first-attempt failure reporting, deduplicated
alerts, overdue-run detection and triage ownership. After #14757 lands, the daily CI health report
will include these workflows; the remaining operational mechanisms are follow-up work. Promotion to
required gates needs demonstrated reliability and a separate maintainer decision.
Keep failed evidence visible and triage confirmed product regressions. Browser
support, the budgets below and deployment/rollback assertions are unchanged;
nonblocking workflows still report failed checks as failures.

## Available hosted browser coverage

- Instrument the initial production-bundle smoke test with bounded stage evidence.
  Preserve the 30-second deadline and capture startup stage evidence. The
  September 30 engine runs pass this check on all nine OS/engine lanes, including
  Windows Firefox; vendor and mobile qualification remain separate.
- Add exact Firefox stable and ESR Linux lanes using official checksums and the
  existing native suite. Keep workstation guards and exact identity checks.
- Extend native Firefox/Apple fixtures with experiment creation failure/retry,
  typed run creation, and recurring-run creation/toggle recovery. Allow only
  explicit in-memory fixture mutations, record requests, and reject others.
- Add Safari 27 and iPhone/iPad iOS 27.0 lanes using the hosted `xcode-27` image,
  alongside the existing 26.x lanes. The September 21
  [runner inventory](https://github.com/actions/runner-images/blob/main/images/macos/xcode-27-arm64-Readme.md)
  lists Xcode 27.0, Safari 27.0 and those simulator runtimes. Require exact observed
  identities; the inventory alone is not a passing qualification result.
  Neither available annual family necessarily supplies its latest supported patch.
- Keep unavailable vendor versions and physical devices explicit. Proposed
  disposition: complete available CI coverage and retain provider-dependent work
  for the previously requested BrowserStack investigation. No account or purchase is assumed.

## Accepted performance budgets

Measure current and pinned original legacy source
`02cbc725ac9ddcd950f4400d8355dd78bfcd6c57` on the same disposable hosted runner.
Also rebuild the pre-compression modern control `dc1b3cfbaba17444408c9a59eb8ab9354b2dfd79`.
This is the rebased equivalent of the previously qualified `439304aa4ed3a20ae5a7c004b96936cb65c3aff0`,
with the same application code and candidate dependency lockfile. Master’s React Query
5.104.0 and XYFlow 12.12.0 updates increased candidate entry gzip by 62 bytes
(596,806 to 596,868), so the old dependency set is no longer a matched control for
the additional zero-growth compression check. Both original legacy and graph checkpoint
references, all accepted budgets, and the zero-growth assertion remain unchanged.
Retain historical results under their original source identities; this control update
requires a fresh hosted comparison and does not retroactively qualify the new dependencies.
Use seven interleaved trials per build, rotating all six build orders, fresh contexts, identical fixtures,
4x CPU slowdown, explicit numeric network conditions and retained raw traces.
Finish builds before sampling. Preserve all samples and failed attempts.
The earlier local protocol did not retain numeric Fast 4G settings, so this is a
new hosted protocol rather than an exact reproduction of those measurements.

| Criterion                                                     | Accepted budget                                                                                                     |
| ------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------- |
| Runs, Details, Compare readiness; filter/run/task interaction | Candidate median <= baseline median + max(10% of baseline median, 50 ms)                                            |
| Entry JS/CSS gzip                                             | <=80% of freshly built original legacy                                                                              |
| Initial route loading CLS                                     | Every sample <=0.005                                                                                                |
| Filter CLS                                                    | Every sample <=0.02, only with the conditions below                                                                 |
| Larger graph/comparison workloads                             | Same relative timing budget against a fresh build of measured checkpoint `e79f8d423e6b118e5df94815ee2f36f68570a9a9` |
| First editor open                                             | Candidate median <=4,000 ms for complete editor display; <=5,000 ms for YAML worker readiness                                               |

The filter allowance accepts only expected result-row compaction: the same keyed
matching row moves upward when preceding results disappear. Headers, filter,
pagination and sidebar controls must remain within one CSS pixel, and retained
layout-shift sources must identify row compaction alone. Historical candidate CLS
was 0.013583 versus legacy 0.003459; this difference remains visible in the report.
Do not remount rows or delay results to manipulate the metric.

Retain browser/OS/source identity, fixture/harness/build hashes, all samples,
API/readiness evidence, full-precision layout-shift entries and raw traces.
Use the existing large-graph and populated-comparison samplers with seven samples.
Validate aggregation, threshold boundaries, missing data and evidence integrity
with browser-free tests; actual results must come from hosted execution.

The performance workflow and runner live in [qualification PR #14757](https://github.com/kubeflow/pipelines/pull/14757), not in the application checkout. Run them only from a checkout combining that tooling with its #14584 application dependency. Four immutable build
artifacts carry source and per-file SHA256 manifests. The matched runner uses a
new explicit profile: 150 ms latency, 1.6 Mbit/s download, 0.75 Mbit/s upload,
1280×720/DPR1 and 4x CPU slowdown. Six cases per build include first editor open,
for 126 retained matched trial records across legacy, previous modern and candidate.
The previous modern build also gates main-page and interaction timings, editor display
and worker readiness under the same relative timing allowance; candidate entry gzip
bytes must not exceed previous modern bytes. The fourth build is the scaling checkpoint.
Public static JS/CSS are served through the production gzip-sidecar handler for all
builds. Immutable older builds have no sidecars and retain identity transfer. Every
trial checks actual response encoding, length, Vary and no-store headers against
the build inventory. Compression stays build-time and does not preload the editor. Each matched trial exports its raw Chromium
trace; layout observations include a declared 500 ms settling interval. Hosted
content readiness requires the route/API predicate after fonts and two confirming
frames. Earlier transient readiness is retained separately; loss after confirmation
fails the trial even if it recovers before capture. The effective hosted protocol
and its hash are retained separately from the unchanged historical protocol. Editor
model readiness requires the complete fixture YAML in read-only Ace, fonts and
two animation frames on both builds. The candidate must additionally load its
YAML worker successfully and return the same complete model through a worker
round trip; that worker-ready duration is reported separately. The immutable
legacy build omits `worker-yaml.js`; each baseline editor trial must retain its
actual 404 and complete model evidence. This known baseline failure does not
waive candidate worker correctness or permit other missing assets/errors, and
there is no equivalent fully worker-ready legacy timing.

The larger-workload samplers run seven samples per build without throttling,
sequentially after matched sampling. Graph uses checkpoint then candidate;
comparison reverses that order. Their existing raw readiness records retain
fixture, harness, browser, index and loaded-asset identities. Their automation
measurements remain separate from the throttled in-page timings.

The machine-readable `performance-budgets.json` in [qualification PR #14757](https://github.com/kubeflow/pipelines/pull/14757) records status `accepted`; the budgets described here remain the policy reference when only the application PR is checked out. Measurement,
fixture and evidence-integrity errors fail CI immediately, as do exceeded accepted
budgets, including the separate absolute first-editor limits. The editor limits
apply to the same cold-context profile and do not assert legacy timing parity.
Retain the original legacy display timing and candidate worker timing separately.
The artifact
`frontend-performance-evidence/performance.json` records the run, attempt, tested
merge/source, PR head, every sample, comparisons and limitations.

## Real deployment and UI-only rollback

Use disposable hosted Kind clusters for standalone and authenticated embedded
multi-user configurations. The latter needs real Dex/OAuth2 Proxy/Dashboard login;
existing tests that inject identity headers do not establish browser authentication.
Install the pinned upstream aggregate roles used by profile owners, verify their
canonical namespace bindings, and require owner access plus cross-owner denial
before the browser rehearsal. Preserve this authorization configuration through
both UI transitions.
Use the repository's pinned Kubeflow manifests and disposable test users.

Build complete legacy and candidate frontend containers from their own sources
and lockfiles. Their Express server and Dockerfile are currently identical; still
prove compatibility against the same candidate backend through the rehearsal.
Register both images in a disposable CI registry and record actual OCI manifest
digests, archive hashes, config IDs and running pod image IDs. If the runtime
reports a preloaded manifest alias, read the exact reported SHA256 manifest from
the node's containerd content store. Verify the raw manifest hash, image-manifest
schema and qualified archive config digest; a mutable alias is insufficient. An
index must identify exactly one distinct image child; verify both content hashes
and the child descriptor size and media type, and retain the exact bytes.

Before the multi-user baseline, enable and verify native Istio sidecars on the
UI, API, required backend callers, MySQL and SeaweedFS. Their existing
DestinationRules require mutual TLS; preserve their authentication and authorization policies. Require
the proxy to start before network-wait initializers, retain its observed image
identity, and check its readiness across UI replacement and restoration. All
setup finishes before the invariant snapshot and measured UI-only transitions.
This lane does not establish CNI NetworkPolicy enforcement.

1. Start the legacy UI; establish sessions/preferences and create a pipeline,
   experiment, successful run, future/disabled schedule and inspectable artifacts.
2. Replace only the UI image. Exercise baseline resources, new run/schedule
   creation, logs/artifacts/TensorBoard, preferences and embedded namespace changes.
3. Restore the captured legacy UI pod template and immutable image. Preserve the
   same browser context and backend. Inspect candidate-created resources and
   create another working run through the restored UI.

Require unchanged backend pod templates/UIDs, authorization configuration,
service accounts, ConfigMaps, and TensorBoard signing state. Verify data and
preferences survive, old signed access still works, and normal authenticated
sessions have the expected namespace access. Check deep-link refresh, prefixed
assets and absence of mixed asset generations. Measure interruption during both
Recreate transitions; no zero-downtime claim is assumed.

Upload sanitized image/deployment/resource evidence, JUnit, browser errors and
cluster diagnostics. Exclude credentials, cookies, tokens and Secret
contents; record signing-state equality without publishing its value.

## Evidence retention

The repository caps Actions artifacts at seven days. Preserve sanitized raw
measurement records, effective protocol, source/build/image identities and
independent audit results in this repository before expiry. Large traces,
screenshots and complete logs remain downloadable for the configured seven-day
window; published reports must state that limit and retain their verified hashes.

## Completion

Publish measured results and qualification budgets to issue #14572 and
implementation PR #14584. Maintainers confirmed no KEP is required; the issue
holds the complete design, compatibility and release requirements. Keep
unavailable coverage and the measured first-editor tradeoff explicit.
Real deployment and rollback must pass without backend changes or data repair.
