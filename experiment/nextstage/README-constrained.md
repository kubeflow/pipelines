# Positive four-seat shared-priority experiment

`constrained-fixed-four-seat-bulk-list-v3` is a separate experiment. All completed
stock-pressure samples and the two unsuccessful stock-policy calibrations remain
unchanged. This scenario deliberately changes APF policy; it is not evidence that
stock Kubernetes defaults have four seats or that production defaults should
change.

KFP source, seven image identities, fixtures, source SDK, Kubernetes1.36.1 image,
10+10 server inflight flags, generator schedules and request deadlines remain
pinned. Existing bootstrap APF policy specs remain unchanged. Preparation creates
exactly one Limited priority and one FlowSchema. The priority uses60 nominal
shares,0% lending,0% borrowing,128 queues,hand size6 and queue length50. The stock
share sum245 predicts ceil(20*60/305)=4 seats; actual nominal/current seat metrics
must both equal4. Prediction is not accepted as runtime evidence.

The schema matches only the resolved API-server service account, the independent
quiet competitor and the background service account. It uses ByUser, so KFP
end-user requests passing through the API-server service account share one flow.
This tests fairness across those service accounts, not per-tenant APF fairness.
Other pipeline/controller identities retain stock classification. No RBAC grants
or authorization bypasses are added for KFP. Policies and UIDs are captured and
rechecked; any undeclared change fails the experiment.

The background namespace contains1024 ConfigMaps with exactly1KiB payload each,
plus the named probe and Kubernetes root-CA object. Only background LIST covers
the namespace, bounded at2048 objects with continuation refused. GET and serialized
UPDATE still target the named probe. Quiet traffic remains20RPS with named-probe
LIST. Generator drops, update-slot drops and actual HTTP attempts are retained;
nominal offered rate is never reported as achieved throughput.

Calibration retains four bounded steps:1worker/16RPS,4/64,16/256,64/1000. Each has
33 seconds of observations,5 seconds launch lead and a60-second subprocess bound;
all calibration fits370 seconds. Selection requires ten contiguous complete
one-second observations queued at the actual positive four-seat capacity. An
observed nominal/current capacity other than4 refuses selection. Isolated missing
samples are retained; more than five consecutive missing/failed health samples
or counter resets refuse selection. If calibration fails, no pair starts and the
result remains inconclusive. Neither the criterion nor pressure is changed silently.

The first qualifying rate/worker pair is frozen for fresh20/−1 KFP arms. Background
pressure occurs only during sustained and three bursts (210 seconds total,
maximum210000 offers), and stops in recovery. The harness concurrency bound is
203:64 KFP reads,64 background,64 quiet,4 serial pipeline lifecycle callers,
6 metrics/health requests and1 authorization probe. Generated driver/controller
traffic is additional. Two active pipelines per tenant permits all three fixtures
to be exercised; actual runtime evidence must still prove their execution.

Actual KFP classification requires Metadata audit records joined with the pinned
API-server's JSON HTTP completion records and immutable APF policy snapshots.
Only sanitized projections are retained. Configuration, same-account surrogate
headers or an incomplete join do not certify actual request classification.
Collection gaps and bounded-log rotation remain explicit failures.

State-aware container health distinguishes successful Job completion from an
unready service using current state, exit code, pod phase and batch/v1 Job
controller identity. Restart, identity and OOM checks remain strict. Previously
captured readiness-only failures cannot be retroactively cleared.

A single pair cannot establish production default safety, three-pair
reproducibility, minimum supported Kubernetes behavior, APF-disabled fallback,
per-RPC outgoing retry cessation or fixture output correctness. All unmet gates
remain explicit even if positive-seat overload and real KFP progress are measured.
