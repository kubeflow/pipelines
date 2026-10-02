# Remaining release qualification

Status: completing hosted deployment and editor qualification, October 2, 2026.
The maintainer accepted timing, payload and layout budgets on September 30.
The separate first-editor limits below were adopted on October 2 to complete
editor qualification; the slower cold-open tradeoff remains explicit.
They are blocking project criteria, not field-percentile claims. Unavailable provider/device rows remain explicit
follow-ups under the existing CI-only delivery plan; they do not count as qualified.

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
Use seven interleaved pairs, alternating order, fresh contexts, identical fixtures,
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

The implementation lives in `.github/workflows/frontend-performance-qualification.yml`
and `frontend/scripts/run-performance-qualification.mjs`. Three immutable build
artifacts carry source and per-file SHA256 manifests. The matched runner uses a
new explicit profile: 150 ms latency, 1.6 Mbit/s download, 0.75 Mbit/s upload,
1280×720/DPR1 and 4x CPU slowdown. Six cases per pair include first editor open,
for 84 retained matched trial records. Each matched trial exports its raw Chromium
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

`frontend/scripts/performance-budgets.json` records status `accepted`. Measurement,
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
