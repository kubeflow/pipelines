# Representative APF observer and independent competitor candidate

**Offline candidate only. No cluster, Docker, deployment, GitHub, or source-tree
mutation has been performed. No live result or upstream-default acceptance is
claimed.** Files live exclusively in this temporary directory and use the
existing KFP Go module dependencies.

## Scope

The executable combines a bounded one-second Kubernetes API observer with an
independent service-account ConfigMap workload. It does not provision resources,
patch APF policy, adopt another cluster, or call the KFP API. It verifies:

- Explicit `kind-kfp-apf-representative-*` kubeconfig context and recorded
  `kube-system` namespace UID, matching the provisioning candidate's identity.
- Externally supplied KFP and competitor FlowSchema UIDs both map to the exact
  same existing **Limited** PriorityLevel UID; an exempt priority is rejected.
- Every actual competitor HTTP attempt must return the expected FlowSchema and
  PriorityLevel response-header UIDs, including retries. Missing/wrong headers
  and transport failures mark classification incomplete. A shared policy name
  alone is insufficient.
- The matched priority must expose a positive current seat limit in each sample.
  The observer never assumes separate protected-priority progress means shared
  fairness. It does not create a new FlowSchema to force a desired result.

KFP's supplied UID is **not independently observed by this process**. The summary
always reports `kfp_live_classification_verified:false` and requires integration
with actual KFP outbound-header evidence. Its policy snapshot only proves that
the supplied UID references the same priority. A false or stale supplied identity
cannot be upgraded into proven shared-priority behavior by this slice.

## Competitor behavior and bounds

Provide an independently provisioned service account, short-lived Kubernetes
API-audience token file (0600), a dedicated namespace and one precreated ConfigMap.
The account needs only get/list/update ConfigMaps in that namespace; it must not
be cluster-admin or share KFP's own service account. This slice never creates
credentials or RBAC and never gives credentials in CLI arguments/artifacts.

Offers use an absolute schedule, round-robin get/list/update, fixed configured
rate, and a hard inflight cap. List uses a name field-selector and limit1. Update
first gets the current object and changes only the experiment sequence annotation;
The entire update GET/PUT sequence has a single-writer admission slot. Overlapping
update offers are recorded as `update_busy_drop` instead of waiting or issuing a
self-conflicting write; reads retain concurrency. External resourceVersion409s
remain distinctly recorded as `conflict_409`, never retried by the generator. Keep the
ConfigMap's initial content/size and rate frozen across paired arms. Client-go's
HTTP retries remain unchanged. Logical request timeout is10s. Generator scheduler
and inflight drops are explicit, so slower completion cannot silently reduce
offered load. Each timestamped logical record has operation, schedule lag,
latency and outcome. Separate raw-attempt records preserve HTTP status, header
latency and classification; raw429 count is not a count of terminal failures.

Limits:2–2700s observation, max1000 offers/s, max64 inflight, 10s call deadline,
12s final call allowance, 4MiB metrics response, 4KiB health response,100,000
events and128MiB event output. Configuration caps offered calls at90,000; raw
HTTP retries may exhaust the event budget earlier, which fails capture. Full
metrics snapshots stream to disk; only the previous sample and compact per-second
status are retained, not every metrics series across the entire experiment. Write/short-write, periodic1-second fsync, final fsync or event-budget failure
immediately cancels both activities and marks capture incomplete. JSON artifacts
are also fsynced before success. Calibrate bytes/second and retry volume before
choosing a duration; capture exhaustion is never treated as successful evidence. The output directory must be new and is0700; files are0600. Tokens,
request bodies, raw error messages and authorization headers are never saved.

## Observer behavior

At the shared `start_at` and once per second through the configured end, record:

- APF queue/seat/limit gauges, dispatched/rejected counters by reason, queue wait
  and execution histograms exposed by kube-apiserver.
- Kubernetes API request latency/status/inflight, exposed etcd request/WAL/backend
  histograms, API-server process CPU/RSS.
- readyz and livez status; health bodies are discarded.
- Counter/histogram interval deltas, with resets flagged rather than treated as
  negative improvement. Allowlisted metric labels are retained; omitted labels
  are aggregated consistently when computing deltas.

Each sample has a900ms overall fetch budget and250ms schedule-lag threshold.
A slow metrics request can leave no health budget; those are missing health
observations, not proof the server was unhealthy. Metrics families required for
APF/seat/request/resource analysis are explicitly flagged if absent, while rejection/wait counters not yet instantiated on an idle server are
recorded separately as `unknown_counters`. Their absence does not fail a healthy
calibration merely to force a rejection, and is never assumed zero. The summary
reports `counter_baselines_complete:false`; newly appearing series have no
fabricated delta against zero.
The final count must be seconds+1; gaps, fetch failure, unavailable seat capacity,
wrong classification, output failure, or counter resets make
`observation_complete:false` and exit nonzero. This strict flag concerns this
slice, **not** representative acceptance: `representative_acceptance_complete`
is always false.

## Build/offline tests

From the existing source module without editing it:

```sh
go build -o /private/tmp/kfp-apf-overload-observer/observer \
  /private/tmp/kfp-apf-overload-observer/main.go
go test -race /private/tmp/kfp-apf-overload-observer/main.go \
  /private/tmp/kfp-apf-overload-observer/main_test.go
```

Tests use local httptest plus real Kubernetes wire clients, not a real cluster.
They cover absolute offer counts/drop accounting and actual get/list/update wire
requests, expected bearer credential transmission, exact APF header matching and
mismatch flags, metric deltas/resets/missing series, aggregation after filtering,
one-second incomplete snapshots, configuration/cancellation and output bounds.

## Configuration and invocation — not executed

```json
{
  "kubeconfig": "/private/tmp/apf-representative-r1/kubeconfig",
  "context": "kind-kfp-apf-representative-r1",
  "cluster_uid": "REPLACE_FROM_PROVISIONING",
  "namespace": "apf-independent-probe",
  "configmap": "fixed-probe",
  "token_file": "/private/tmp/apf-competitor.token",
  "kfp_flow_uid": "REPLACE_FROM_ACTUAL_KFP_RESPONSE_HEADERS",
  "competitor_flow_uid": "REPLACE_FROM_CLASSIFICATION_CALIBRATION",
  "priority_uid": "REPLACE_SHARED_LIMITED_PRIORITY_UID",
  "timeline_file": "/private/tmp/apf-workload-20-r1/timeline.json",
  "seconds": 960,
  "rps": 5,
  "max_inflight": 4,
  "start_at": "REPLACE_SHARED_FUTURE_UTC_TIMESTAMP"
}
```

```sh
/private/tmp/kfp-apf-overload-observer/observer \
  --config /private/tmp/apf-observer-config.json \
  --out /private/tmp/apf-observer-20-r1
```

The required timeline is the workload candidate's `timeline.json`, containing
contiguous absolute load windows followed by fixed `after_load` probe, `drain`,
and `after_drain` probe windows. Observer rejects any gap, duplicate window name,
wrong shared start, missing reserved stage, or duration that differs from the
FULL schedule (`sum(load seconds)+2*probe_seconds+drain_seconds`). It saves the
validated timeline and original-file SHA256. Snapshot and competitor records
carry the corresponding phase; final end-boundary samples are outside the last
half-open window. `seconds:960` above is illustrative and must exactly match the
actual timeline, not a separately guessed duration.

Both processes require the same future UTC timestamp; observer accepts2–60s
lead time. The workload writes its timeline before waiting. A coordinator must
wait for that file, launch observer with enough setup lead, and cancel workload
if `baseline-ready.json` is absent before the start. Observer captures/fsyncs a
baseline at start−1s before sending competitor offers, then writes that marker.
A missed baseline/start deadline sends no competitor traffic. The marker binds
the common start and timeline hash. This candidate cannot itself cancel the
separate workload process if baseline capture fails: peer-failure coordination
remains mandatory. Run calibration first to resolve actual identities without
assuming bootstrap policy names.

## Remaining integration gaps and no-claim boundaries

1. Provision least-privilege competitor account/ConfigMap, token refresh and real
   KFP outgoing APF-header evidence from core/auth/execution/SWF clients. A stock
   policy classification mismatch is a failed stock-policy variant, not permission
   to silently patch policy. Direct metrics use admin credentials only as observer;
   they are never counted as competitor traffic.
2. Kubernetes API-server metrics do not reliably expose direct etcd WAL/fsync or
   container restart/OOM evidence. Those remain explicit `integration_missing`
   entries; integrate a separately bounded etcd/resource/status collector. Process
   metrics alone do not prove no restarts. No pod logs are read here.
3. Cross-process failure/cancellation coordination, noisy/quiet tenant progress, actual KFP authorization,
   workflow/output/SWF assertions, no-deadline disconnect watchdog, APF-disabled
   variant and minimum Kubernetes version are supplied by other required slices.
4. The predeclared paired acceptance evaluator (10 consecutive pressure samples,
   fairness p95/throughput windows, recovery and errors) is not implemented here.
   Timestamped evidence enables it but does not itself pass those thresholds.
5. Inspect dropped offers, conflict errors and missing counters during calibration.
   Choose and freeze input rate/object size/concurrency before comparison. No
   parameter is auto-increased until it produces a desired outcome.

## Review-fix verification

Additional offline race regressions cover full timeline duration including probe
and drain, gap rejection, phase mapping, capture short-write/fsync/event limits,
periodic fsync cancellation, compact retained snapshots, and single-writer update
admission with unchanged absolute offered-count/drop accounting. All14 offline test functions pass under `go test -race -timeout30s`
(10.572s), and the candidate builds. No new live cluster execution has been performed. Missing live KFP classification and broader
health/acceptance gates remain explicit; these fixes do not establish them.

### Metrics capture bounds and projection

Scrapes stream through a 32 MiB network limit and 64 KiB line limit within the
existing 900 ms snapshot deadline. Unrelated families are discarded before
retention. Capture records expose only HTTP status, consumed byte count, retained
byte count, a fixed reason code, and the configured deadline; failed, truncated,
malformed selected metrics and exceeded limits fail closed as missing evidence.
The previous 4 MiB full-response limit could reject otherwise usable scrapes;
the prior run did not retain enough diagnostics to prove which fetch error occurred.

For the selected shared priority, retain exact labels for current queue requests,
executing seats, limit seats, rejected/dispatched request totals, and wait-duration
histograms. APF names were checked against Kubernetes apiserver v0.36.1
`pkg/util/flowcontrol/metrics/metrics.go`. Required series and positive selected
capacity checks remain mandatory; absent idle counters remain unknown.

Other retained families are explicitly projected: API request totals sum across
labels except verb and code; API/etcd latency histograms sum across labels except
`le`; process health is scalar. Stable keys preserve aggregate cumulative deltas,
histogram bucket boundaries and aggregate reset detection. This projection cannot
attribute latency to an individual resource or detect a reset in an individual
aggregated series that other series' growth masks. It does not claim those details.
At most 256 projected series and 24 KiB encoded metrics are retained per scrape;
combined metrics plus deltas are capped at 48 KiB. The 1,232 maximum snapshots in
this fixed experiment use at most 58 MiB for metric payload, leaving over 69 MiB
inside the unchanged 128 MiB event limit for competitor records and metadata.
Any retention bound failure remains explicit and invalidates complete capture.
