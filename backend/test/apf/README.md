# Disposable Kubernetes APF experiment

This opt-in harness compares the exact client-go dependency used by KFP with
5 QPS / 10 burst, 20 QPS / 40 burst, and QPS -1 (no local token bucket). It runs
real TokenReview and SubjectAccessReview requests, retaining authentication and
RBAC. It is a Kubernetes-client experiment, not a KFP end-to-end benchmark.
Production defaults and the normal CI matrices are unchanged.

Use a dedicated Kind cluster named `kfp-apf-experiment` and a separate kubeconfig.
The harness refuses any other current context. The cluster must have APF enabled;
for controlled pressure, a disposable kube-apiserver can use
`max-requests-inflight: "10"` and `max-mutating-requests-inflight: "10"`.

```sh
go run ./backend/test/apf \
  --kubeconfig /private/tmp/kfp-apf-kubeconfig \
  --out /private/tmp/kfp-apf-sample-1 --seconds 10
```

Run three repetitions with separate output directories. Wait until namespace
`kfp-apf-experiment` has finished deleting before starting the next repetition.
The harness creates only its named namespace, RBAC, FlowSchemas and
PriorityLevelConfigurations, and deletes those resources on exit. It never
patches the default APF policies. Do not run against an existing shared cluster.

Each setting runs one-worker and 32-worker loads while a separate service-account
client makes progress in a separate priority level. These are real HTTP calls
with independent TokenReview and SubjectAccessReview client token buckets, as in
KFP. A scoped denied cross-namespace Secret permission probe must remain denied.
The harness records successful paired calls, latency percentiles, local limiter
wait, raw HTTP status counts, and APF response-header UIDs. Window cancellation,
request deadlines and terminal throttling are reported separately from unexpected
errors. Raw HTTP attempts include client-go retries; they are not logical call
counts. After terminal errors the generator pauses for a context-aware 10 ms
before offering another logical call, preventing a tight loop on local limiter
deadline errors; client-go HTTP retries are unchanged. Time spent waiting is summed across concurrent calls, not wall time.

Metrics are saved before/after each sample. A final **synthetic zero-seat** priority
level verifies actual queuing, 429 responses, bounded cancellation/retry and
successful recovery after capacity is restored. This is an intentional overload
fixture, not a proposed production APF configuration. Classification UIDs must
match the exact created policies, and the zero-seat phase must show increases in queue-timeout and queue-wait metrics plus actual
429s. Cumulative deltas avoid missing short-lived queues between observations.
Results are saved after each completed phase, before validation assertions. The ordinary pressure phases need separate interpretation:
no observed queuing means APF overload was not demonstrated in that phase.

Output includes no tokens, request bodies or HTTP authorization headers. Metrics,
policy snapshots and results are written locally with restrictive permissions.
One-node Kind throughput and fairness evidence cannot establish production
control-plane sizing or justify disabling client limiting universally. Evaluate
real KFP workload performance and upstream deployment policy separately.

## Reference experiment

Three repetitions on Kubernetes v1.36.1 with client-go v0.37.1 and the constrained
API-server configuration above produced these one-worker paired TokenReview/SAR
medians (range across repetitions):

| Client setting | Median pair latency | Completed pairs per 10-second window |
| --- | --- | --- |
| 5 QPS / 10 burst | 196–200 ms | 50–59 |
| 20 QPS / 40 burst | 27–49 ms | 121–239 |
| QPS -1 | 1.7–2.4 ms | 2,130–4,493 |

At 32 workers with QPS -1, each window completed 5,102–7,240 pairs while receiving
116 raw HTTP 429 responses, including attempts retried by client-go. The separate
control priority completed 2,016–2,758 pairs with p95 latency 7.6–12.1 ms and no
429 responses. Exact policy UIDs matched. Allowed/denied authorization probes
remained correct. The zero-seat phase completed no pairs, observed queue timeouts
and 32 HTTP 429s per repetition, and recovered after capacity restoration.

These results support testing the existing operator opt-in with representative
KFP deployments. They do not establish a universal default, full-pipeline wall
speedup, or identical performance across runs. In particular, fixed-window
cancellation and request deadlines still terminate work under pressure.
