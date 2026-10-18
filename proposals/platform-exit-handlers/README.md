# KEP-NNNN: Platform Exit Handlers and Resource Usage Reports

Status: Draft; not yet approved for implementation.

Tracking issue and KEP number: Not yet assigned.

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
  - [Notes, Constraints, and Caveats](#notes-constraints-and-caveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Relationship to Existing Plugins and Exit Handlers](#relationship-to-existing-plugins-and-exit-handlers)
  - [Platform Configuration](#platform-configuration)
  - [Execution Lifecycle and Failure Isolation](#execution-lifecycle-and-failure-isolation)
  - [Handler Interface and Status](#handler-interface-and-status)
  - [Resource Usage Collection](#resource-usage-collection)
  - [Report Artifact and Publication](#report-artifact-and-publication)
  - [Authorization and Isolation](#authorization-and-isolation)
  - [Implementation Areas](#implementation-areas)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
- [Migration Strategy](#migration-strategy)
- [Frontend Considerations](#frontend-considerations)
- [KFP Local Considerations](#kfp-local-considerations)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
- [Open Questions](#open-questions)
- [References](#references)
<!-- /toc -->

## Summary

Add optional, platform-controlled exit handlers to Kubeflow Pipelines (KFP).
Administrators configure handlers that KFP injects into a run's execution workflow,
after the user pipeline and its user-defined exit handlers have completed. Pipeline
and component authors do not need to modify their code. Platform handlers execute
on normal pipeline success or failure, have bounded execution budgets, and report
their own outcomes without changing the user pipeline's result.

The first handler produces a resource usage report. It queries an existing
Prometheus-compatible monitoring backend for executor-container CPU and memory
measurements, summarizes average and observed maximum usage, and publishes a
versioned JSON artifact associated with the run. The UI reads that artifact to
show usage in each task's details and makes the full report available from the
run. Reports remain available after pod deletion and monitoring retention expiry,
subject to the normal artifact storage retention policy.

This proposal reuses KFP's existing run-plugin metadata and artifact APIs. It does
not introduce a second plugin registry, a time-series database, or per-component
instrumentation.

## Motivation

Pipeline authors need resource usage data to choose CPU and memory requests and
limits. Resource declarations describe allocations, not actual consumption.
Inspecting cluster monitoring manually requires knowing how tasks map to pods,
which containers belong to the workload, and which time intervals to query.
These details are particularly difficult after failures, retries, or pod deletion.

A durable report should be produced even if nobody opens the UI. Computing it only
on demand makes availability depend on when the user first views a run and on the
monitoring backend's retention window. Publishing a report artifact preserves the
result using infrastructure KFP already has.

Resource reporting is also an instance of a broader platform need: operators may
want standardized reporting or notifications after every eligible run without
asking every pipeline author to implement the same exit task. User-defined
`dsl.ExitHandler` groups are useful for pipeline-owned cleanup, but are not an
administrator-controlled installation policy.

### Goals

1. Allow administrators to enable named, platform-controlled exit handlers without
   changing pipeline source, SDK compilation, or component images.
2. Execute enabled handlers after the user pipeline, including its user-defined
   exit handlers, on normal success and failure.
3. Bound scheduling, execution, retries, and finalization latency. Keep platform
   handler failures separate from the user pipeline's outcome.
4. Publish CPU and memory average and observed maximum usage as a durable,
   discoverable run artifact, with per-task and per-executor identity.
5. Display the measurements and their limitations in task details, including
   partial coverage, missing samples, cache hits, and retries.
6. Preserve namespace isolation, run authorization, and existing behavior when the
   feature is disabled.

### Non-Goals

- Guaranteed handler execution after workflow deletion, forceful termination, or
  loss of the execution infrastructure; an external recovery worker is deferred.
- Live resource charts, automatic resource tuning, GPU/network/disk metrics, or
  measurement of external workloads launched by components.
- Exact instantaneous CPU peaks or kernel memory high-water marks.
- A new monitoring installation managed by KFP or polling metrics-server.
- User-supplied handler images, arbitrary code in the API server, or a general
  workflow for dependencies between platform handlers.
- Platform handlers whose failure intentionally fails the pipeline. All handlers
  introduced here are best-effort; required/compliance hooks need a separate policy.
- Local-runner telemetry or a new Python SDK exit-handler API.

## Proposal

An administrator enables a platform exit-handler facility and configures a bounded
list of handlers. KFP resolves and snapshots the effective policy when creating a
run, then injects a platform finalization stage through the execution compiler.
The user pipeline remains unchanged. Independent handlers may run concurrently
within a platform-configured limit.

The resource reporter resolves the run's task executions, obtains resource
summaries through a run-scoped backend operation, and writes one canonical JSON
report per run execution generation. It registers the report as a normal KFP
artifact produced by a system-owned reporting task. Run-plugin output identifies
the authoritative report artifact and reporting status.

The UI offers a run-level report link and a small resource usage table in task
details. It does not query Prometheus and does not require the reporting pod to
still exist.

### User Stories

#### Right-size a component

A pipeline author opens a completed task and sees average and observed maximum CPU
in cores and working-set memory in MiB/GiB. They can inspect the measurement window
and coverage before adjusting resource requests or limits.

#### Diagnose a failed run

A pipeline fails after executing several tasks. The platform reporter still runs,
includes the available measurements from failed and successful executions, and
marks missing data explicitly. The original failure remains the run's result.
An OOM-related spike missed by scraping is not presented as a proven memory peak.

#### Enable reporting without changing pipelines

An administrator configures an existing monitoring backend and enables the
resource reporter for an installation or allowed namespaces. Existing pipeline
versions gain reports on newly created runs without recompilation. A deployment
without the feature continues to run those same pipeline versions unchanged.

#### Add another platform-owned finalization action

An administrator enables another approved handler alongside resource reporting.
It receives the same bounded execution context and independent status. Failure of
one handler does not prevent the other from running.

### Notes, Constraints, and Caveats

- These are **in-workflow exit handlers**, not independent background jobs. They
  execute without a browser session but delay workflow completion by a bounded
  finalization interval.
- Normal failure and cancellation are different cases. Argo documents that
  `argo terminate` skips exit handlers. KFP currently terminates workflows using
  `activeDeadlineSeconds: 0`; its behavior must be tested directly rather than
  inferred from the CLI command.
- Monitoring retention must exceed the interval from the earliest task execution
  to report generation. A final report cannot recover already expired samples.
- Short tasks may finish between scrapes. Missing data is not zero usage.
- The report measures the workload container, including launcher/setup work in
  that container, not exclusively the user's Python function.

### Risks and Mitigations

| Risk | Mitigation |
| --- | --- |
| A reporting failure turns a successful run into a failure | Capture the user-body outcome separately; isolate handler failures in compilation and execution-status mapping; require scheduling-failure and timeout tests before enablement. |
| A handler never starts or hangs | Apply a finalization deadline covering pending time, execution, and all retries, not just a timeout inside its process. |
| Handler and user exit tasks race | Place platform finalization outside the complete user execution, including user exit hooks. |
| Cross-namespace metrics or privileged credential exposure | Server-resolve execution identity, use fixed bounded metric queries, and do not inject monitoring credentials into user pods. |
| Misleading measurements | Publish metric definitions, sampling/rate windows, per-metric coverage, and explicit unavailable reasons. |
| Monitoring load grows with run size | Batch bounded queries, limit concurrency and response sizes, and publish partial results on budget exhaustion. |
| Duplicate publication or stale retry overwrites | Use generation-scoped invocation identity and fenced, idempotent artifact publication. |
| Optional hooks become an implicit security guarantee | Document best-effort execution; do not use this facility for mandatory cleanup, compliance, or revocation. |

Backend and multi-user security maintainers should review the authorization and
credential model. Frontend maintainers should review failure/coverage states and
accessibility. Execution-engine maintainers should review lifecycle semantics and
the supported-version test matrix before the feature is enabled.

## Design Details

### Relationship to Existing Plugins and Exit Handlers

This checkout already contains:

- `RunPluginHandler` lifecycle callbacks, including `OnBeforeRunCreation`,
  `HandleRetry`, and `OnRunEnd`, in `backend/src/apiserver/plugins/`.
- Server-owned `Run.plugins_output`, with `PluginOutput.state`, `state_message`,
  and typed metadata entries; client creation requests cannot set these outputs.
- User exit-handler compilation in
  `backend/src/v2/compiler/argocompiler/dag.go`, using exit tasks' trigger policies
  to build lifecycle hooks.
- Task-to-pod mappings and run/task artifact associations in the v2beta1 APIs.

Reuse the run-plugin configuration and output conventions. Add the ability to
resolve a platform exit-handler plan and attach it through the execution compiler;
do not register another independent lifecycle dispatcher. Existing in-process
plugin callbacks retain their behavior. `OnRunEnd` can reconcile final handler
status but must not execute a second copy of the reporter or synchronously query
all resource samples.

A platform handler is distinct from a user `dsl.ExitHandler`. It runs outside the
user graph, is selected by platform policy, and cannot be replaced by a matching
user task name. Reserve an internal identity for the injected stage and preserve
all existing user exit hooks.

The broader plugin work described in
[KEP-12862](../12862-mlflow-integration/README.md) remains complementary. This
proposal adds a bounded container execution mode for finalization, not a competing
plugin system or a rewrite of the MLflow lifecycle.

### Platform Configuration

The feature is disabled by default. Administrators configure an approved handler
list through the existing platform/namespace plugin configuration mechanism.
Namespace overrides may only select capabilities explicitly delegated by the
installation administrator; a pipeline cannot choose an image, command, service
account, credential, or monitoring endpoint.

Each handler definition includes:

- Stable handler name and implementation image pinned by digest.
- Enablement and permitted namespace scope.
- Resource requests/limits and a constrained runtime identity.
- Overall deadline, maximum attempts, and bounded retry backoff.
- Handler-specific validated configuration and secret references.

The resource reporter additionally needs the approved monitoring connection,
expected scrape interval, CPU rate window, query step, ingestion grace period, and
query/report size limits. An initial deployment may use 15-second scrapes and a
60-second CPU rate window; these are operator settings, not guarantees about every
monitoring backend.

Resolve configuration at run creation, including each recurring-run occurrence.
Persist the non-secret effective configuration, implementation digest, and config
hash in server-owned plugin output and the compiled execution plan. Record secret
references, never secret values. Credential rotation remains possible; an image
or measurement-policy change does not silently alter a run already in progress.
An explicit retry of the same run retains its resolved handler plan. Emergency
policy revocation may prevent execution and must be surfaced as unavailable, not
silently substituted with different code.

Initial enablement is administrator-controlled. A per-run user opt-out or selection
UI is not required for this proposal. Validate definitions before activation;
invalid handler configuration must not produce a malformed user workflow.

### Execution Lifecycle and Failure Isolation

The engine-neutral lifecycle is:

```text
Resolve and persist platform handler plan
                |
Run user pipeline, including all user exit handlers
                |
Capture user-body outcome and completion time
                |
Run bounded, independent platform exit handlers
                |
Finalize workflow using the user-body outcome
```

The Argo compiler implements this with an outer finalization stage and appropriate
failure-tolerant dependencies/hooks. Do not append reporter work as an ordinary
success-dependent DAG task or overwrite an existing user hook. The stage must be
reachable for failed and errored user execution as well as success.

Execution-status reporting must distinguish the user-body outcome from handler
outcomes. Put engine-specific node traversal and hook interpretation behind the
existing execution abstraction, not in shared resource-management code. Preserve
original user failures, including failures of user-defined exit handlers. A
handler's image-pull failure, eviction, nonzero exit, or timeout is not a new user
pipeline failure. Infrastructure failures that prevent determining the body
outcome must not be converted to success.

Failure isolation must work when the handler process never starts; catching
exceptions inside the reporter is insufficient. Compiler and status-conversion
tests must prove the contract for each supported engine version. The raw engine
status and KFP status must not disagree silently: any unavoidable engine-specific
diagnostic is retained separately from the preserved pipeline outcome.

No new public run state is required. While platform finalization is executing,
the run remains non-terminal and plugin output explains that reporting is in
progress. Existing `finished_at` denotes final run completion, including this
bounded stage. Capture user-body completion separately for measurement and
interpretation; task execution windows exclude finalization latency.

The finalization deadline includes scheduling delay and retries. It is enforced by
the engine integration even when no reporting pod is running. A run's existing
overall deadline and an explicit cancellation take precedence; handlers do not
extend those limits or prevent cancellation.

Do not wait for the run's terminal API state from inside a handler: the handler
itself may be preventing that state. Pass the captured body outcome and execution
generation directly. Wait only a bounded interval for task metadata and monitoring
ingestion to catch up, then publish partial results if possible.

If the workflow survives but a handler is interrupted, persist its failure through
normal execution reporting. If the workflow is forcibly removed or reporting
infrastructure is lost, a report is not guaranteed. The UI must not infer successful
report generation from pipeline completion alone. Automatic recovery after such
cases requires an external reconciler and is deferred.

### Handler Interface and Status

The execution context contains:

- Run ID, namespace, execution identity, and retry generation.
- Handler name, version/config hash, and stable invocation ID.
- User-body outcome and completion time.
- Approved artifact destination and run-scoped API credentials.

Handlers have no dependencies on each other. They run with caching disabled and
must tolerate repeated delivery. Stable invocation identity is derived from
`(run ID, execution generation, handler name)`. Retrying the handler preserves the
identity; retrying the user run creates a new generation. The existing retry claim
generation provides a starting point, but must be exposed and enforced consistently
across execution, publication, and plugin status updates.

Reuse a reserved `plugins_output["platform_exit_handlers"]` entry for the plan and
per-handler status records. Each record has `pending`, `running`, `complete`,
`partial`, `failed`, or `unavailable` status, timestamps, diagnostic reason, and
output artifact IDs. These are values in typed metadata, not additions to the
pipeline's runtime-state enum. The existing plugin state summarizes the group:
`PLUGIN_RUNNING` while unfinished, `PLUGIN_SUCCEEDED` if every handler completed
(including an explicitly partial report), otherwise `PLUGIN_FAILED`.

The backend owns these records. A narrow handler-result operation accepts only
updates for the caller's assigned invocation; it does not permit arbitrary writes
to `plugins_output`, other handlers, or the pipeline outcome. Merge updates with
generation fencing and concurrency control so simultaneous completion and existing
plugin callbacks cannot overwrite unrelated plugin entries. A stale execution
must not replace a newer generation's report pointer.

System-owned reporting tasks reuse the task/artifact model, but are marked by
server-controlled provenance and displayed separately from user graph nodes. Their
failed state must not participate in user-body outcome calculation. Do not rely on
user-settable names or artifact metadata alone to establish system ownership.

### Resource Usage Collection

Prometheus (or a compatible backend) scrapes kubelet/cAdvisor independently of KFP.
The reporter does not install a sidecar, read process-local resource counters, or
poll metrics-server. Metrics-server is intended for autoscaling and does not
provide the historical monitoring store needed here.

The reporting pod calls a fixed, run-scoped backend summary operation. The backend:

1. Verifies the assigned invocation and resolves tasks/executions from the run.
2. Selects executor workload containers, excluding drivers, init containers,
   sidecars, and platform handler tasks. User-defined exit tasks are included when
   they have workload executors.
3. Resolves cluster, namespace, pod UID, container identity, and execution windows.
4. Queries only those series and windows through the configured monitoring client.
5. Returns per-execution summaries and coverage diagnostics in bounded batches.

The operation accepts neither arbitrary PromQL nor caller-selected namespaces,
pods, metric names, or endpoints. The reporter assembles the results into the
artifact. Keep query construction and summarization in a focused module rather
than growing `ResourceManager` into a telemetry implementation.

`PipelineTask.pods` already contains pod name, UID, and driver/executor type, but
not per-container start/end timestamps or explicit attempt numbers. Extend the
execution metadata path to retain the identity and actual container windows needed
by reporting before pods are garbage-collected. Task creation and completion
timestamps are not a substitute for container runtime. If the data cannot be
resolved safely, return unavailable rather than query a guessed interval.

Prometheus label sets vary. Validate and document the supported mapping from pod
UID/container identity to metric series. Do not assume a `uid` label exists. Use a
verified historical identity mapping where needed; pod name alone is insufficient
when names can be reused. Deduplicate replicated scrape series before aggregation.

| Field | Definition |
| --- | --- |
| CPU average | Arithmetic mean of available CPU-rate evaluations on the configured regular evaluation grid, in cores. |
| CPU observed max | Maximum of those CPU-rate evaluations; not instantaneous CPU usage. |
| Memory average | Arithmetic mean of available working-set observations on the configured regular grid, in bytes. |
| Memory observed max | Maximum observed working-set value, in bytes; not a kernel high-water mark. |

Use `rate(container_cpu_usage_seconds_total[<window>])` for CPU and
`container_memory_working_set_bytes` for memory. Apply counter-reset handling before
aggregating; never average the cumulative CPU counter. Clip evaluations to the
container execution interval and do not carry stale samples into unobserved gaps.
Document the source freshness threshold and report the scrape interval, evaluation
step, CPU rate window, expected evaluations, observed evaluations, and covered
interval for each metric. Rate windows need sufficient counter samples; short
executions may have memory results but no usable CPU estimate.

Averages are **observed-window averages**, not guaranteed whole-lifetime values.
Missing intervals are not filled with zero. Completeness describes usable sampled
coverage, not proof that every spike was observed. Validate finite, nonnegative
values; represent missing metrics as `null` with an explanation.

Keep separate records for each task/pod/container execution, including failed
attempts and loop iterations. Do not infer attempt numbers from pod-list position.
Do not sum maxima or average per-attempt averages to create a misleading task
aggregate. Cached/skipped tasks are recorded as not executed; they do not inherit
resource measurements from the cached producer. A failed task that never started
has no usage estimate.

Bound query concurrency, points, tasks per batch, memory, and total elapsed time.
Wait a bounded ingestion grace period after body completion, then query the
historical task windows, not the reporter's current time. Publish partial results
when some queries fail or a budget is exhausted. Expose expiration or missing
identity distinctly from monitoring connection failure.

### Report Artifact and Publication

Publish a generic KFP `Artifact` containing `application/json` with a versioned
resource-report schema. No new artifact enum is required. The following is an
illustrative complete entry; real reports also include entries for unavailable or
not-executed tasks:

```json
{
  "schema_version": 1,
  "run_id": "run-123",
  "execution_generation": 0,
  "handler": "resource-usage",
  "config_hash": "sha256:example",
  "status": "complete",
  "generated_at": "2026-10-16T10:06:00Z",
  "measurement": {
    "scope": "executor_workload_container",
    "memory_metric": "working_set_bytes",
    "scrape_interval_seconds": 15,
    "evaluation_step_seconds": 15,
    "cpu_rate_window_seconds": 60
  },
  "executions": [
    {
      "task_id": "task-456",
      "pod_name": "train-executor",
      "pod_uid": "pod-789",
      "container": "main",
      "started_at": "2026-10-16T10:00:00Z",
      "finished_at": "2026-10-16T10:05:00Z",
      "status": "complete",
      "cpu_cores": {
        "avg": 0.42,
        "max": 1.25,
        "coverage": {
          "expected_evaluations": 20,
          "observed_evaluations": 20,
          "start": "2026-10-16T10:00:15Z",
          "end": "2026-10-16T10:05:00Z"
        }
      },
      "memory_bytes": {
        "avg": 671088640,
        "max": 964689920,
        "coverage": {
          "expected_evaluations": 20,
          "observed_evaluations": 20,
          "start": "2026-10-16T10:00:15Z",
          "end": "2026-10-16T10:05:00Z"
        }
      }
    }
  ]
}
```

Use this publication sequence:

1. Write a complete immutable object under the approved run/generation/handler
   prefix. Never expose an in-progress object as the final report.
2. Register it through KFP's artifact API with the reporting task as producer and
   a reserved output key. Existing `ArtifactTask` associations carry run and task
   IDs; an Argo-only output declaration does not create the KFP association.
3. Atomically publish the authoritative artifact ID and terminal reporting status
   in server-owned plugin output, fenced to the invocation generation.

Retries reuse a completed publication or finish an interrupted one. Use the
existing artifact reuse facilities where they provide the required guarantee;
prove behavior under concurrent publication rather than assuming a lookup followed
by create is atomic. An interruption after upload but before registration may
leave an unreferenced object, handled by normal storage lifecycle cleanup. Readers
only follow the committed artifact pointer. If storage is unavailable, mark
reporting failed rather than claiming a report exists.

A user-run retry publishes a new generation's report without deleting older
reports. Preserve links to prior reports and prevent late callbacks from selecting
an older generation. The current generation's report covers its own executor
executions; unchanged/cached work is not attributed new resource consumption.

Store detailed measurements in object storage, not as four new database columns
per task. Plugin output contains compact status and artifact references. Report
retention and deletion follow existing run/artifact policies; this proposal does
not promise that keeping run metadata alone preserves an object deleted by an
external bucket lifecycle rule.

### Authorization and Isolation

Monitoring credentials remain in the backend's configured metrics client. User
pods and reporting artifacts never receive them. Reporter pods use short-lived,
run/invocation-scoped credentials for summary reads, assigned-task updates, and
artifact publication. Pipeline code cannot obtain a handler identity simply by
supplying a task name, plugin input, annotation, or artifact metadata marker.

The server resolves ownership and namespace from persisted run/task data. Fixed
metric selectors and time/size budgets prevent a caller from exploring other
namespaces or issuing unbounded queries. Use authenticated TLS for the monitoring
connection and prevent user-controlled URLs or redirects from changing its target.

Runtime isolation must also prevent user tasks from mounting reporter credentials
or modifying its configuration. Namespace administrators with Kubernetes-level
control remain within the platform's existing trust model; the feature does not
claim to isolate mutually hostile administrators in a shared namespace.

Artifact reads use the existing authorized artifact-serving path and namespace
checks. The UI treats JSON as data, validates schema and size, and never executes
report HTML or scripts. Only a server-owned artifact pointer establishes that a
report is the platform's resource report.

### Implementation Areas

| Area | Expected work |
| --- | --- |
| `backend/src/apiserver/plugins/` | Resolve handler plans; reuse plugin status; fence concurrent result updates and retries. |
| Execution compiler and `backend/src/v2/compiler/argocompiler/` | Inject an outer platform stage; preserve user hooks; enforce deadlines and failure isolation. |
| `backend/src/common/util/` and execution reporting | Separate user outcome from platform status; retain execution identity/windows through pod cleanup. |
| `backend/api/v2beta1/` and backend API handlers | Add narrowly scoped handler operations and any additive execution metadata needed; regenerate clients rather than editing generated code. |
| Reporter runtime and focused backend telemetry module | Batch metric queries, summarize samples, and publish the report artifact. |
| Existing artifact storage and APIs | Reuse producer associations and artifact serving; validate publication idempotency. |
| `frontend/src/components/tabs/RuntimeNodeDetailsV2.tsx` and run details | Display report status, link, and per-task usage using shared run-level fetching. |
| Deployment manifests and operator documentation | Optional configuration, scoped identities, monitoring prerequisites, retention and cancellation caveats. |

### Test Plan

- [x] This proposal acknowledges that owners of the involved components may require
  updates to existing tests before accepting the implementation.

#### Prerequisite testing updates

Add characterization tests for user exit handlers, pipeline-body outcome mapping,
run retry generations, and KFP's actual termination path. Establish failure
isolation for non-starting hook pods before implementing resource reporting.
Record tested execution-engine versions; do not infer behavior from another Argo
release or from successful reporter executions alone.

Current unit coverage has not been measured for this documentation-only proposal.
Implementation PRs should record coverage for the compiler, plugin dispatcher,
execution conversion, telemetry module, and affected frontend components.

#### Unit Tests

- Policy resolution, namespace delegation, immutable plan snapshots, disabled
  behavior, reserved-name collisions, and deterministic compilation.
- Composition with user exit handlers, independent platform handlers, body-outcome
  preservation, bounded retries/deadlines, and reporting-task exclusion.
- Counter resets, duplicate series, missing/stale samples, uneven source scrapes,
  very short executions, failed executions, and exact unit conversions.
- Pod identity mapping, name reuse, loop iterations, cache hits, and explicit
  separation of executor pods without invented attempt numbers.
- Partial reports, non-finite values, response/report limits, schema validation,
  and artifact-pointer generation fencing.
- Concurrent plugin updates, retry replay, and no overwriting of MLflow or other
  plugin state.
- Frontend loading/partial/unavailable states, separate retries, null versus zero,
  unknown schema versions, authorized report discovery, and accessibility.

#### E2E tests

Run on a cluster with a known scrape configuration and a synthetic CPU/memory
workload. Assert bounded ranges and units, not exact timing-sensitive usage values.

| Scenario | Required result |
| --- | --- |
| Feature disabled | No handler pod, report fetch, or changed pipeline outcome. |
| Successful pipeline | Report artifact is associated with the run; UI shows task summaries. |
| Failed task or user exit handler | Reporter runs; original failure is preserved. |
| Reporter process failure, image-pull failure, unschedulable pod, or eviction | Bounded finalization; independent handler status; no new user-body failure. |
| Multiple handlers, one failed | Other handlers still execute. |
| Task retries and parallel loop iterations | Separate, correctly attributed executor records. |
| Cache hit, skipped task, or no started container | No fabricated zero or inherited usage. |
| Prometheus unavailable or samples expired | Partial/failed report state with actionable reason. |
| Pods garbage-collected before reporting | Persisted identity/windows permit querying retained samples. |
| KFP cancellation, workflow deadline, and forceful deletion | No cancellation blockage; documented report availability, without a false completion claim. |
| User-run retry and delayed old reporter | New generation selected; old callback cannot overwrite it. |
| Report viewed after monitoring retention expiry | Artifact remains readable if its storage retention permits. |

#### Integration tests

- Mock monitoring responses through the real backend summary operation and verify
  selectors, query bounds, tenant isolation, and no arbitrary PromQL access.
- Exercise real artifact registration plus object-store publication, including
  crashes between upload, registration, and plugin-pointer commit.
- Test concurrent/stale callbacks and plugin-map updates against supported database
  engines; validate artifact reuse under duplicate requests.
- Reject forged reporter identities, cross-run artifact associations, and attempts
  to replace platform policy or another plugin's output.
- Verify late task metadata and final scrapes are handled within the ingestion
  budget, without waiting for the handler's own workflow to become terminal.
- Test large task counts and multiple finalizing runs against query, memory,
  artifact-size, and latency budgets. Establish supported limits before graduation.
- Test frontend polling during finalization, terminal report fetch, and retained
  reports after the deployment feature is subsequently disabled.

### Graduation Criteria

**Initial opt-in availability:** failure isolation and cancellation behavior are
verified on supported engine versions; the report schema, scoped credentials,
artifact publication, and UI states have integration coverage. Disabled behavior
is unchanged. Operator documentation states sampling and retention limitations.

**Stable availability:** realistic large-run and concurrent-run tests demonstrate
bounded overhead; upgrade/rollback and retry-generation fencing are verified;
security and frontend review are complete; operator diagnostics identify handler
failures and partial coverage. Published report schema versions remain readable.
No release number or commitment to default enablement is assigned by this draft.

## Migration Strategy

Existing pipelines, components, and compiled pipeline specs require no migration.
Enablement applies to newly created execution workflows; do not patch an already
running workflow to add handlers. Historical runs without reports remain valid and
show reporting as not enabled or unavailable, not zero usage. No historical
backfill is proposed.

Reuse existing plugin-output storage, tasks, artifacts, and artifact-task
associations. No dedicated telemetry table or per-task aggregate columns are
proposed. Any additive execution metadata should use compatible existing payloads
where appropriate. If implementation requires a schema/index change for identity
or publication guarantees, it must follow the repository's
[nonblocking database change policy](../../docs/agents/database-migrations.md), with
supported-engine evidence; this draft does not authorize blocking startup DDL or
assume `AutoMigrate` makes an upgrade safe.

Roll out backend readers and handler APIs before enabling compiler injection, then
deploy the reporter and UI. New API fields are additive, and unknown report schema
versions produce an explanatory UI state rather than a rendering failure. Keep the
schema version separate from the handler image version.

Disabling the feature stops injection for new runs, but does not prevent authorized
reads of existing reports. Before rolling back to a backend that lacks the new
handler operations, drain or explicitly cancel active finalization work. Do not
delete report artifacts, plugin metadata, or task associations during rollback.

## Frontend Considerations

Add a run-level **Resource usage report** entry showing pending, running, complete,
partial, failed, or unavailable status and a downloadable artifact link. Platform
handler tasks are visible in a separate system/finalization section rather than
appearing as authored nodes in the pipeline graph. Make clear that a reporting
failure is not the pipeline's failure.

Under the existing **Task Details** tab, display:

```text
Resource usage — workload container

                 Average       Observed max
CPU              0.42 cores    1.25 cores
Memory           640 MiB       920 MiB

15s sampling · CPU rate window: 60s
```

For multiple executor pods, show separate pod/container records and their windows.
Do not invent retry numbers from pod order or silently merge attempts. Show
coverage and a concise reason for missing metrics. Cached/skipped tasks read
**Not executed**, not `0`. Use absolute units; percentages of requests/limits,
charts, and graph badges are deferred.

Fetch the report once at run scope, index by task ID, and share the data across task
panels. While finalizing, poll lightweight plugin status with bounded backoff; fetch
the immutable report when its artifact ID becomes available or changes. Do not
refetch the full report separately for every node. A report is generated even if
no UI is open. Enforce payload limits and schema validation before rendering.

The feature is displayed from the run's recorded plan/status or existing report,
not merely today's deployment toggle. Runs created without the feature do not
cause speculative artifact or monitoring requests.

## KFP Local Considerations

There are no SDK or local-runner changes. `SubprocessRunner` and `DockerRunner` do
not pass through the server-side execution compiler or platform policy, so they do
not inject these handlers or collect Kubernetes metrics. Existing local behavior
of user-defined exit handlers is unchanged. Document that platform reports are a
remote-deployment capability, not a portable pipeline output guaranteed locally.

## Implementation History

- 2026-10-16: Initial draft covering platform-controlled exit handlers, resource
  report artifacts, and task-detail visualization. No implementation or release
  has been completed. Assign the KEP number when a tracking issue is created.

## Drawbacks

- Each enabled handler adds execution overhead and bounded completion latency,
  even though it cannot change the user pipeline's result.
- Outcome isolation, generation fencing, and handler authentication add complexity
  beyond a single user-authored exit task.
- Reporting depends on cluster monitoring labels, retention, scrape cadence, and
  artifact storage; KFP cannot guarantee complete measurements.
- One JSON artifact is simple and durable, but cannot provide efficient arbitrary
  cross-run analytics. Large reports need enforced limits and partial-result UX.
- Best-effort exit handling is weaker than an independent post-run reconciler for
  cancellation and infrastructure-loss cases.

## Alternatives

### Query monitoring only when the UI opens

This avoids reporter execution and artifact publication, but results disappear when
monitoring retention expires and availability depends on someone viewing the run.
It does not meet the durable, unattended reporting goal.

### Require a user-defined exit handler in every pipeline

This can prototype reporting, but requires authors to modify pipelines, propagates
monitoring configuration into user code, and does not establish platform policy.
It remains useful for pipeline-specific cleanup rather than installation-wide
reporting.

### Use the existing in-process `OnRunEnd` callback for all reporting

This reuses existing lifecycle delivery without a workflow stage and is a useful
option for small external notifications. Performing large monitoring queries and
artifact writes inline would consume backend/plugin callback budgets and would
not provide the requested isolated container execution mode. Keep existing
callbacks, but use them for planning/status integration rather than executing this
reporter inside the API server.

### Launch independent post-run Jobs from a reconciler

This separates reporting from workflow completion and can provide better recovery
after cancellation or workflow loss, provided execution metadata survives. It is
a stronger choice if guaranteed eventual reporting becomes a requirement. It also
needs durable scheduling, ownership, retry, and cleanup outside the workflow.
Defer it while keeping reporter invocation and publication idempotent so the same
reporter can later be called by such a reconciler.

### Summarize each task at completion

This reduces dependence on whole-run monitoring retention, but adds per-task
coordination and persistent intermediate results. Revisit for pipelines longer
than the monitoring retention window; it is not needed for the initial end-of-run
reporting contract.

### Add sidecars or component instrumentation

These can capture finer-grained observations, but change every workload, introduce
runtime overhead, and complicate arbitrary-container support. Reuse existing
cluster telemetry and state its sampling limitations instead.

### Store every aggregate directly in task database columns

This supports database-side comparisons, but adds schema and write-path changes
without being necessary for four values in task details. A report artifact plus
compact discovery metadata is sufficient for this proposal.

## Open Questions

These require implementation/reviewer decisions before the KEP is implementable:

1. Which supported Argo versions can satisfy failure isolation and a deadline
   covering unscheduled handlers with the selected outer-stage structure? If the
   contract cannot be met, use the external-job alternative rather than weaken the
   promise that optional reporting does not fail user work.
2. Which pod/container identity labels and historical mapping are supported by the
   first monitoring integration, and what execution metadata must be retained
   before pod cleanup to make that mapping reliable?
3. What default finalization, query concurrency, report-size, and ingestion-grace
   budgets are supported by representative deployment measurements?

## References

- [Kubeflow KEP process](https://github.com/kubeflow/community/blob/master/proposals/README.md)
- [Kubeflow KEP template](https://github.com/kubeflow/community/blob/master/proposals/NNNN-template/README.md)
- [KFP proposal requirements](../README.md)
- [KFP MLflow integration proposal](../12862-mlflow-integration/README.md)
- [KFP run and task API](../../backend/api/v2beta1/run.proto)
- [KFP artifact API](../../backend/api/v2beta1/artifact.proto)
- [Argo exit handlers](https://argo-workflows.readthedocs.io/en/latest/walk-through/exit-handlers/)
- [Argo termination](https://argo-workflows.readthedocs.io/en/latest/cli/argo_terminate/)
- [cAdvisor Prometheus metrics](https://github.com/google/cadvisor/blob/master/docs/storage/prometheus.md)
- [Prometheus query functions](https://prometheus.io/docs/prometheus/latest/querying/functions/)
- [Metrics-server scope and limitations](https://github.com/kubernetes-sigs/metrics-server#readme)
