# Separate constrained-control-plane APF scenario

This is a new experiment identity, `constrained-stock-apf-bulk-list-v2`.
It preserves the completed stock-pressure samples and the v1 calibration failure.
V1 reached queue depth9 but only one consecutive queued sample, so no pair ran. KFP source, seven image
config digests, fixture YAML and source SDK remain pinned to the existing exact
merge. The only control-plane change is `max-requests-inflight=10` and
`max-mutating-requests-inflight=10`. Bootstrap APF specs must exactly match the
archived v1.36.1 stock specs. Generated Kind configuration and actual API-server
flags are both captured. Never configure zero seats or a protected priority.

The background service account has namespace-scoped LIST permission only for
ConfigMaps and GET/UPDATE permission only on its named probe ConfigMap. Its owned
namespace contains1024 synthetic ConfigMaps with exactly1KiB data each, plus the
probe and Kubernetes root-CA object. Each background LIST requests the namespace
with limit2048; continuation or more than2048 returned objects fails explicitly.
GET and serialized UPDATE remain named-probe operations; the fixed20-RPS quiet
competitor still lists only its named probe. Seed counts and payload hashes are
verified and captured before calibration. Actual response
header UIDs must map both identities to the same Limited priority. This does not
prove actual KFP client classification; that remains a separate unmet gate.

Before a pair, bounded33-second calibration steps try `(workers,RPS)` of
`(1,16),(4,64),(16,256),(64,1000)`, with30 seconds no calibration traffic between
steps. The subprocess bound is60 seconds (5-second lead,33-second timeline,
12-second observer context tail and10-second durable-output margin), within the
unchanged370-second calibration bound. The first step with ten contiguous complete one-second samples showing
queue>0 and executing seats>=observed limit>=1 is frozen. Policy specs must remain
unchanged, no counter reset is permitted, and more than five consecutive missing
or failed health samples blocks selection. Isolated missing observations remain
recorded. Selecting pressure never constitutes complete observation or acceptance.
If the bounded steps cannot prove pressure, no pair starts and the result is
inconclusive; rates or seats must not be adjusted silently.

Background pressure runs only during sustained and the three burst windows:
210 seconds total, at most210,000 offers. The same selected schedule applies to
both20 and−1. No excess background traffic occurs in normal, rest, recovery,
probe or drain windows. Quiet traffic remains20 RPS throughout. The separate
workload variant permits two active pipelines per tenant, eight total, so a
completed flip-coin can free a slot for oneof while collected-parameters is still
running. All active runs are polled and drained; no ambiguous CreateRun retries.
Fixture presence and output correctness still require captured runtime evidence.

Explicit upper budget:64 KFP read callers +64 background +64 quiet Kubernetes
callers +4 serial pipeline-lifecycle callers +6 admin metrics/health requests
+1 authorization probe =203 harness requests in flight. Real driver/controller
traffic is additional and measured; preflight/postflight operations and
port-forward setup are outside load windows. Fixed offered rates and generator
scheduling/inflight drops are retained; successful throughput is not offered load.

The scenario-only background trace is gzip-streamed, bounded at128 MiB compressed,
512 MiB uncompressed and1,000,000 events. Its compact summary omits the huge
logical-request array; raw logical and HTTP-attempt records remain in
`background/events.jsonl.gz`. Quiet observer output remains plain JSONL.
Independent900-ms metrics/readiness/liveness requests prevent a completed large
scrape from consuming the health request's entire shared budget; strict missing
capture and acceptance flags remain unchanged.

`hosted-constrained.yml` provisions an owned disposable cluster, verifies inputs,
calibrates, then calls `run_pair.py` for a new20/−1 pair. A failed observation
remains failed. Advancing to the next arm requires a complete durable timeline,
reconciled terminal runs, completed post-auth probes, passed container health,
settled background process and no prolonged missing/failing health. The coordinator
owns its45-minute SIGALRM and bounded process-group cleanup; the wrapper does not
abruptly kill that parent with a second timeout. Every dispatch and outcome is
written before advancing. No already-completed stock arm is rerun.

This first constrained pair cannot establish three-pair reproducibility, minimum
Kubernetes-version compatibility, APF-disabled fallback, outgoing retry cessation,
owner artifact-byte semantics or universal default safety. All remain explicit
unmet gates even if the exploratory workflow succeeds.
