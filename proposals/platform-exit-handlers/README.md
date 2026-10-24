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
  - [SDK Pipeline Input and Run-Time Opt-In](#sdk-pipeline-input-and-run-time-opt-in)
  - [Administrator-Customizable Resource Finalizer](#administrator-customizable-resource-finalizer)
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
after the user pipeline and its user-defined exit handlers have completed. The
platform owns handler implementations; component authors do not need to instrument
their code. Platform handlers execute on normal pipeline success or failure, have
bounded execution budgets, and report their own outcomes without changing the user
pipeline's result.

The first handler produces a resource usage report and is **off by default**.
Pipeline authors explicitly bind a Boolean pipeline input defaulting to `False` to
report generation. Users opt in per run by passing `True` through the SDK's existing
pipeline `arguments` dictionary, including when calling
`create_run_from_pipeline_func` or `create_run_from_pipeline_package`. Reporting
requires both this user opt-in and platform permission/configuration; platform
availability alone never turns it on. The built-in handler queries an existing
Prometheus-compatible monitoring backend. Administrators may replace its logic
with an approved finalizer image that reads another metrics source and implements
the same invocation and report contracts; neither internal Prometheus nor a KFP
backend code change is required to install such an implementation. The handler
summarizes average and observed maximum executor-container CPU and memory usage
and publishes a
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

1. Allow administrators to configure approved platform-controlled exit handlers
   without component instrumentation or user-authored exit tasks.
2. Make resource reporting off by default, with a Boolean pipeline input that users
   override through ordinary SDK run arguments, subject to platform permission.
3. Execute enabled handlers after the user pipeline, including its user-defined
   exit handlers, on normal success and failure.
4. Bound scheduling, execution, retries, and finalization latency. Keep platform
   handler failures separate from the user pipeline's outcome.
5. Publish CPU and memory average and observed maximum usage as a durable,
   discoverable run artifact, with per-task and per-executor identity.
6. Display the measurements and their limitations in task details, including
   partial coverage, missing samples, cache hits, and retries.
7. Allow administrators to replace the resource finalizer's collection and reporting
   logic, including querying non-Prometheus sources, without changing the SDK opt-in
   or task-detail UI contract.
8. Preserve namespace isolation, run authorization, and existing behavior when the
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
list of handlers. Resource report generation additionally requires a user-enabled
Boolean pipeline input; an omitted or false input does not generate a report. KFP
resolves and snapshots the effective policy and input value when creating a run,
then injects the selected platform handlers through the execution compiler.
Workload tasks and user-defined exit handlers remain unchanged. Independent
platform handlers may run concurrently within a platform-configured limit.

The resource reporter resolves the run's task executions and writes one canonical
JSON report per run execution generation. The built-in implementation obtains
summaries through a run-scoped backend operation; an administrator-supplied
implementation may query a different approved source using the same execution
inventory and normalize results to the same report schema. It registers the report as a normal KFP
artifact produced by a system-owned reporting task. Run-plugin output identifies
the authoritative report artifact and reporting status.

The UI offers a run-level report link and a small resource usage table in task
details. It does not query a metrics provider and does not require the reporting
pod to still exist.

### User Stories

#### Right-size a component

A pipeline author declares a reporting input with a `False` default. For a profiling
run, they set it to `True` in `create_run_from_pipeline_func(..., arguments=...)` or
`create_run_from_pipeline_package(..., arguments=...)`. They open a completed task
and see average and observed maximum CPU in cores and working-set memory in MiB/GiB.
They can inspect measurement windows and coverage before adjusting resource
requests or limits. Subsequent runs that omit the override do not generate reports.

#### Diagnose a failed run

A run with reporting requested fails after executing several tasks. The reporter
still runs, includes the available measurements from failed and successful executions, and
marks missing data explicitly. The original failure remains the run's result.
An OOM-related spike missed by scraping is not presented as a proven memory peak.

#### Make reporting available without forcing it on users

An administrator configures an existing monitoring backend and permits the
resource reporter for an installation or allowed namespaces. Pipeline authors add
the Boolean input and explicit binding once; users can then choose reporting for
each run without recompiling an already published package. Old pipelines without
a binding and runs using the false default do not gain reporting overhead. On a
supported backend where reporting is unavailable, the workload still runs and an
explicit reporting request is recorded as unavailable.

#### Use a different metrics source

An administrator whose metrics live in a managed monitoring service installs an
approved resource-finalizer image and its endpoint/credential configuration. It
queries that service, maps observations to the supplied task execution identities,
and emits the standard resource report. Users still enable it with the same
Boolean pipeline input; they do not select providers or supply credentials. The
same UI renders the report without Prometheus-specific assumptions.

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
| Cross-namespace metrics or privileged credential exposure | Server-resolve execution identity; use fixed queries for the built-in implementation and scoped provider permissions for custom implementations; never inject provider credentials into user pods. |
| A custom finalizer produces misleading data or leaks secrets | Treat images as trusted operator code, restrict their identity/egress, validate reports, and require explicit metric semantics and source provenance. |
| Misleading measurements | Publish metric definitions, sampling/rate windows, per-metric coverage, and explicit unavailable reasons. |
| Unexpected reporting cost on ordinary runs | Require an explicitly bound Boolean input defaulting to `False`; platform configuration alone cannot enable the reporter. |
| SDK/package disagreement or accidental truthiness | Persist the binding in the compiled package and validate Boolean types/defaults in both SDK and backend. |
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

The platform facility is disabled by default. Administrators configure an approved
handler list through the existing platform/namespace plugin configuration mechanism.
For resource reporting, this configuration grants availability, not run-level
consent: even on a configured platform, users must explicitly opt in through the
bound Boolean pipeline input.
Namespace overrides may only select capabilities explicitly delegated by the
installation administrator; a pipeline cannot choose an image, command, service
account, credential, or monitoring endpoint.

Each handler definition includes:

- Stable handler name and implementation image pinned by digest.
- Enablement and permitted namespace scope.
- Resource requests/limits and a constrained runtime identity.
- Overall deadline, maximum attempts, and bounded retry backoff.
- Handler-specific validated configuration and secret references.

The resource reporter additionally needs an approved source connection and
collection configuration. The built-in implementation uses an expected scrape
interval, CPU rate window, query step, ingestion grace period, and query/report size
limits. Custom implementations receive their administrator-defined provider
configuration and must honor the shared deadline and output-size limits. An initial deployment may use 15-second scrapes and a
60-second CPU rate window; these are operator settings, not guarantees about every
monitoring backend.

Resolve configuration at run creation, including each recurring-run occurrence.
Persist the bound input name, resolved Boolean request, non-secret effective
configuration, implementation digest, and config hash in server-owned plugin
output and the compiled execution plan. Record secret references, never secret
values. Credential rotation remains possible; an image
or measurement-policy change does not silently alter a run already in progress.
An explicit retry of the same run retains its resolved input value and handler plan;
a new run may choose a different Boolean value. Emergency policy revocation may
prevent execution and must be surfaced as unavailable, not
silently substituted with different code.

Validate definitions before activation; invalid handler configuration must not
produce a malformed user workflow. Other platform handlers keep their own approved
selection policy; the utilization input controls only resource-report generation.

### SDK Pipeline Input and Run-Time Opt-In

Add an optional `resource_usage_report_parameter: str` binding to the existing
`dsl.PipelineConfig`. It identifies an ordinary root pipeline input rather than
introducing a reserved input name or a new client submission flag. Pipeline authors
choose the input name. The SDK validates that the referenced input exists, has
Boolean type, and declares a literal `False` default.

Proposed SDK usage (the new `PipelineConfig` field is not implemented yet):

```python
from kfp import client, compiler, dsl


@dsl.component
def train(dataset: str):
    print(f'Training on {dataset}')


@dsl.pipeline(
    name='training-with-optional-report',
    pipeline_config=dsl.PipelineConfig(
        resource_usage_report_parameter='report_utilization',
    ),
)
def training_pipeline(dataset: str, report_utilization: bool = False):
    train(dataset=dataset)


kfp_client = client.Client()

# Off by default: no reporter task or resource-report queries.
kfp_client.create_run_from_pipeline_func(
    pipeline_func=training_pipeline,
    arguments={'dataset': 'gs://example/data'},
)

# Opt in for this run through the normal pipeline arguments.
kfp_client.create_run_from_pipeline_func(
    pipeline_func=training_pipeline,
    arguments={'dataset': 'gs://example/data', 'report_utilization': True},
)

# The binding survives compilation; package submission supports the same override.
compiler.Compiler().compile(
    pipeline_func=training_pipeline,
    package_path='training.yaml',
)
kfp_client.create_run_from_pipeline_package(
    pipeline_file='training.yaml',
    arguments={'dataset': 'gs://example/data', 'report_utilization': True},
)
```

Omitting `report_utilization` or explicitly setting it to `False` disables report
generation. Do not add an `enable_resource_usage_report` keyword to client methods;
the existing `arguments` path is the user interface. The pipeline does not need a
Python `if`, a conditional reporter task, or a user-authored exit handler. Declaring
a similarly named input without the explicit binding has no reporting side effect.

Serialize the binding as an additive optional `resource_usage_report_parameter`
field in the pipeline-spec `PipelineConfig`, carried in the Kubernetes platform's
`pipelineConfig` alongside existing pipeline-level settings. The Boolean itself
remains a normal root input and is supplied through existing runtime parameter
values. This preserves the binding in compiled packages and uploaded versions.
Choose an unused proto field number during implementation and regenerate bindings;
do not encode the binding in a description, task name, or inferred naming convention.

Only the submitted root pipeline's binding controls the run-level report. Nested
pipeline bindings cannot independently enable additional reporters. The binding
does not modify workload inputs, dependencies, or task cache keys; the reporter
itself is never cached. Authors may explicitly pass the Boolean to workload tasks,
but doing so is not needed to control reporting.

Resolve reporting before injecting execution handlers:

1. Validate the binding and the referenced root input on the server as well as in
   the SDK. The bound input must be Boolean with a `False` default, including in
   uploaded or hand-authored packages; missing defaults and true defaults are
   rejected. A compile-time parameter-default override must not bypass this rule.
2. Resolve a provided runtime argument over that default, using the normal pipeline
   input path. Require a Boolean value; do not interpret strings such as `"false"`
   or integers through language truthiness.
3. Inject the reporter only when the resolved value is `True` and the platform
   allows/configures it. A false request produces no reporter pod, telemetry query,
   or report artifact and adds no reporting finalization delay.
4. Snapshot the resolved request and policy. A true request on an unavailable or
   disallowed platform does not fail the workload: publish a lightweight
   `unavailable` reporting status with the reason, without injecting the reporter.

| Root binding / resolved value | Platform permits and configures reporting | Result |
| --- | --- | --- |
| No binding | Either | Off; no report generation. |
| Bound input omitted (default `False`) | Either | Off; no report generation. |
| Explicit `False` | Either | Off; no report generation. |
| Explicit `True` | Yes | Inject resource reporter for this run. |
| Explicit `True` | No | No reporter; expose requested-but-unavailable status. |
| Invalid binding or non-Boolean value | Either | Reject invalid pipeline/submission before workload execution. |

The same resolution applies to uploaded pipeline versions and recurring runs using
their normal parameter values. Recurring runs preserve the selected input and
resolve current platform availability for each new run. Platform configuration and
`plugins_input` cannot silently override a false or omitted reporting input.

### Administrator-Customizable Resource Finalizer

The `resource-usage` handler is a logical capability, not a hard-coded image or an
assumption that metrics are stored in internal Prometheus. The administrator may
choose the shipped implementation or supply a custom OCI image, pinned by digest,
with an approved command, configuration, secret references or workload identity,
and network policy. This replacement may change source queries, authentication,
collection, and aggregation logic. Only one selected implementation produces the
canonical resource report for an invocation.

Changing the endpoint of the built-in implementation is enough for another
compatible Prometheus service. A custom image supports a managed monitoring API,
an external time-series store, or other collection logic without adding provider
implementations to KFP's backend. Do not require a provider registry, arbitrary
code loading in the API server, or a new SDK option for each source.

Both implementations use the same finalizer contract:

1. Receive the resolved run/invocation context and administrator-selected config.
2. Retrieve a paginated, read-only execution inventory from KFP using the scoped
   invocation identity. It includes task IDs, execution generations, pod/container
   identities and windows, and explicit unavailable/not-executed entries. It does
   not expose arbitrary run inputs, secrets, or another run's executions.
3. Obtain historical measurements from the approved source. The built-in image can
   use the backend's fixed summary operation; a custom image can query its provider
   directly. It is not forced through a Prometheus-only endpoint or parser.
4. Normalize observations into the versioned report schema, preserving the required
   units, coverage and metric semantics. Upload/register the artifact and complete
   the invocation through the same fenced publication path.

Selecting an implementation is an administrator action, never a pipeline argument.
The resolved image digest, non-secret configuration hash and source identifier are
snapshotted with the handler plan. The user's Boolean remains off by default and
only requests this capability; it cannot replace its implementation or credentials.

Custom code must preserve working-set memory and observed CPU-rate semantics.
Memory RSS, allocated memory, a provider's undocumented percentage, or a differently
windowed maximum must not be silently relabeled as comparable values. A missing or
unsupported metric is `null` with a reason. Record actual source resolution,
aggregation method, CPU window and coverage; the UI derives its explanation from
the report rather than hard-coding Prometheus terminology or a 60-second window.
Custom sources may already have downsampled data, which must be disclosed.

Provide a report schema validator and contract fixtures for administrator-authored
images. Malformed reports fail publication as resource reports, and a custom image
has the same timeout, retry, partial-result, failure-isolation, and idempotency
requirements as the built-in image. Validation can check shape, units and ownership;
it cannot prove an operator-supplied algorithm measures the right thing. Operators
are responsible for validating source mappings and computations.

The finalizer consumes historical telemetry. Merely scraping a live metrics endpoint
at run exit cannot reconstruct earlier usage; an alternative source must retain
samples or trustworthy historical aggregates for the task execution windows.

### Execution Lifecycle and Failure Isolation

The engine-neutral lifecycle is:

```text
Resolve root input defaults + runtime overrides + platform policy
                |
Persist selected platform handler plan
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

The following describes the built-in Prometheus implementation. A Prometheus or
compatible backend scrapes kubelet/cAdvisor independently of KFP; it may be internal
or external to the cluster. The reporter does not install a sidecar, read
process-local resource counters, or poll metrics-server. Metrics-server is intended
for autoscaling and does not provide the historical monitoring store needed here.
Custom implementations use their own provider queries under the shared contract
above, not the PromQL-specific steps below.

The built-in reporting pod calls a fixed, run-scoped backend summary operation.
The backend:

1. Verifies the assigned invocation and resolves tasks/executions from the run.
2. Selects executor workload containers, excluding drivers, init containers,
   sidecars, and platform handler tasks. User-defined exit tasks are included when
   they have workload executors.
3. Resolves cluster, namespace, pod UID, container identity, and execution windows.
4. Queries only those series and windows through the configured monitoring client.
5. Returns per-execution summaries and coverage diagnostics in bounded batches.

The built-in summary operation accepts neither arbitrary PromQL nor caller-selected
namespaces, pods, metric names, or endpoints. The reporter assembles the results
into the artifact. The read-only execution inventory is shared with custom
implementations; it must be available without a configured Prometheus connection.
Keep query construction and summarization in a focused module rather than growing
`ResourceManager` into a telemetry implementation.

`PipelineTask.pods` already contains pod name, UID, and driver/executor type, but
not per-container start/end timestamps or explicit attempt numbers. Extend the
execution metadata path to retain the identity and actual container windows needed
by reporting before pods are garbage-collected. Task creation and completion
timestamps are not a substitute for container runtime. If the data cannot be
resolved safely, return unavailable rather than query a guessed interval.

Metric-source identity conventions vary. Validate and document the supported mapping
from pod UID/container identity to provider series for every implementation.
For Prometheus specifically, label sets vary. Do not assume a `uid` label exists.
Use a verified historical identity mapping where needed; pod name alone is insufficient
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
resource-report schema shared by built-in and custom finalizers. Include a
non-secret source identifier, implementation identity and aggregation method in
addition to the measurement windows and units. No new artifact enum is required.
The following is an illustrative complete entry; real reports also include unavailable or
not-executed tasks:

```json
{
  "schema_version": 1,
  "run_id": "run-123",
  "execution_generation": 0,
  "handler": "resource-usage",
  "implementation_id": "kfp-prometheus-reporter/v1",
  "config_hash": "sha256:example",
  "status": "complete",
  "generated_at": "2026-10-16T10:06:00Z",
  "measurement": {
    "source": {"kind": "prometheus", "id": "platform-monitoring"},
    "aggregation_method": "regular_grid_observed_samples",
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

For the built-in implementation, monitoring credentials remain in the backend's
configured metrics client. For a custom implementation that queries its provider
directly, administrators may supply restricted secret mounts or workload identity
to that finalizer pod only. The user pipeline's service account must not acquire
these permissions. Never distribute credentials through shared user-pod environment
injection, pipeline parameters, logs, report artifacts or public plugin metadata.

All reporter pods separately use short-lived, run/invocation-scoped KFP credentials
for execution inventory/summary reads, assigned-task updates, and artifact
publication. Pipeline code cannot obtain a handler identity simply by
supplying a task name, plugin input, annotation, or artifact metadata marker.

The server resolves ownership and namespace from persisted run/task data. Fixed
metric selectors and time/size budgets constrain the built-in summary operation.
For direct custom queries, enforce least-privilege access at the provider or a
scoped proxy and restrict finalizer egress; handing out cluster-wide credentials
and relying on Python code to filter namespaces is not equivalent isolation. If a
provider cannot scope access, the administrator must explicitly treat that finalizer
as privileged trusted platform code; never make its execution context available
to pipeline authors. Use authenticated TLS and administrator-approved endpoints,
and prevent user-controlled URLs or redirects from changing their targets.

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
| `sdk/python/kfp/dsl/pipeline_config.py` and SDK compiler | Add and validate the root Boolean-input binding, require its false default, and preserve it in compiled packages. |
| `api/v2alpha1/pipeline_spec.proto` | Add the optional `PipelineConfig` binding field and regenerate Python/Go bindings and relevant frontend types. |
| SDK client tests and SDK documentation | Verify normal `arguments` overrides for function and package submission; no new client flag. |
| `backend/src/apiserver/plugins/` and run-input resolution | Validate and resolve the bound Boolean before handler injection; intersect with platform permission; fence concurrent result updates and retries. |
| Execution compiler and `backend/src/v2/compiler/argocompiler/` | Inject an outer platform stage; preserve user hooks; enforce deadlines and failure isolation. |
| `backend/src/common/util/` and execution reporting | Separate user outcome from platform status; retain execution identity/windows through pod cleanup. |
| `backend/api/v2beta1/` and backend API handlers | Add narrowly scoped handler operations and any additive execution metadata needed; regenerate clients rather than editing generated code. |
| Reporter runtime and focused backend telemetry module | Ship the default Prometheus implementation; expose provider-independent execution inventory and fixed built-in summaries; support approved replacement images and a shared report validator. |
| Existing artifact storage and APIs | Reuse producer associations and artifact serving; validate publication idempotency. |
| `frontend/src/components/tabs/RuntimeNodeDetailsV2.tsx` and run details | Display report status, link, and per-task usage using shared run-level fetching. |
| Deployment manifests and operator documentation | Optional implementation/image/config selection, scoped KFP and provider identities, custom-image contract fixtures, retention and cancellation caveats. |

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
Implementation PRs should record coverage for SDK configuration/compiler/client
paths, the execution compiler, plugin dispatcher, execution conversion, telemetry
module, and affected frontend components.

#### Unit Tests

- SDK binding validation: missing input, wrong type, missing/true default, arbitrary
  input names, and compile-time default overrides. An unbound similarly named input
  must have no special behavior; nested pipeline bindings must not inject reporters.
- Function/package round trips: omitted argument stays false, explicit false stays
  off, and explicit true opts in through existing `arguments`. Check strict Boolean
  validation and preservation in uploaded pipeline versions and recurring runs.
- Server validation of hand-authored packages, input/default resolution before
  injection, false inputs despite administrator availability, and true requests on
  unavailable/disallowed platforms. A plugin input cannot bypass the Boolean gate.
- Policy resolution, namespace delegation, immutable request/plan snapshots,
  disabled behavior, reserved-name collisions, and deterministic compilation.
- Composition with user exit handlers, independent platform handlers, body-outcome
  preservation, bounded retries/deadlines, and reporting-task exclusion.
- Counter resets, duplicate series, missing/stale samples, uneven source scrapes,
  very short executions, failed executions, and exact unit conversions.
- Pod identity mapping, name reuse, loop iterations, cache hits, and explicit
  separation of executor pods without invented attempt numbers.
- Custom implementation selection, image/config snapshots, secret redaction,
  provider-independent execution inventory, and no dependency on built-in Prometheus
  configuration when a replacement image is selected.
- Partial reports, non-finite values, response/report limits, shared schema
  validation, metric provenance, and artifact-pointer generation fencing.
- Concurrent plugin updates, retry replay, and no overwriting of MLflow or other
  plugin state.
- Frontend loading/partial/unavailable states, separate retries, null versus zero,
  unknown schema versions, authorized report discovery, and accessibility.

#### E2E tests

Run on a cluster with a known scrape configuration and a synthetic CPU/memory
workload. Assert bounded ranges and units, not exact timing-sensitive usage values.
Unless a scenario explicitly varies enablement, reporting is requested and allowed.

| Scenario | Required result |
| --- | --- |
| Platform configured, no binding or default/explicit false input | No resource reporter pod, telemetry query, report artifact, or reporting delay. |
| Function submission with Boolean argument `True` | Report artifact is associated with the run; UI shows task summaries. |
| Compiled package submission with Boolean argument `True` | Same behavior without recompilation; binding survives package/upload round trips. |
| Explicit request but platform disabled or not configured | Workload runs; requested-but-unavailable status, no reporter. |
| Recurring run and user-run retry | Request value is preserved; platform permission and generation rules are enforced. |
| Failed task or user exit handler | Reporter runs; original failure is preserved. |
| Reporter process failure, image-pull failure, unschedulable pod, or eviction | Bounded finalization; independent handler status; no new user-body failure. |
| Multiple handlers, one failed | Other handlers still execute. |
| Task retries and parallel loop iterations | Separate, correctly attributed executor records. |
| Cache hit, skipped task, or no started container | No fabricated zero or inherited usage. |
| Built-in internal or external Prometheus endpoint | Same canonical artifact and UI contract with correct source provenance. |
| Custom image using a non-Prometheus test provider, with Prometheus disabled | Boolean opt-in runs the approved replacement; inventory, report publication and UI work without the built-in summary service. |
| Provider unavailable, metrics unsupported, or samples expired | Partial/failed report state with actionable reason, not mislabeled or fabricated values. |
| Pods garbage-collected before reporting | Persisted identity/windows permit querying retained samples. |
| KFP cancellation, workflow deadline, and forceful deletion | No cancellation blockage; documented report availability, without a false completion claim. |
| User-run retry and delayed old reporter | New generation selected; old callback cannot overwrite it. |
| Report viewed after monitoring retention expiry | Artifact remains readable if its storage retention permits. |

#### Integration tests

- Submit the same compiled package with omitted, false, and true arguments and
  verify handler injection and plugin status. Test both SDK submission methods,
  uploaded versions, and the ordinary UI Boolean input. Platform enablement must
  never change a false request to true.
- Mock monitoring responses through the real backend summary operation and verify
  selectors, query bounds, tenant isolation, and no arbitrary PromQL access.
- Run shared contract fixtures against the built-in reporter and a custom image
  backed by a non-Prometheus test API. Verify metric semantics, source resolution,
  unsupported-metric handling, and identical UI rendering without provider-specific
  frontend code.
- Verify provider credentials are mounted only into the approved finalizer, never
  workload pods, and cannot be selected or overridden through pipeline arguments.
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
At least one non-Prometheus contract-test implementation demonstrates that the
finalizer interface is not coupled to the built-in metrics backend. Resource
reporting remains off by default at every maturity level. No release
number or commitment to default platform-facility enablement is assigned by this
draft.

## Migration Strategy

Existing pipelines and components continue to work without migration and do not
automatically generate reports. To make a pipeline eligible, its author adds the
Boolean input with a `False` default and the `PipelineConfig` binding, then compiles
and publishes a new package/version once. Subsequent runs override the Boolean
through normal SDK `arguments` without recompiling that package. A runtime argument
alone cannot opt an older, unbound package into reporting.

Selection applies to newly created execution workflows; do not patch an already
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

Roll out backend readers, Boolean binding resolution, and handler APIs before
publishing bound pipeline packages and enabling compiler injection; deploy the
reporter, SDK support, and UI with the documented compatibility floor. Old packages
without the binding retain their behavior. The new compiled-spec field requires a
backend that understands it, even when the input defaults to false: older strict
proto readers may reject the package. Do not claim that the false default makes
new packages compatible with every older backend.

New API fields are additive, and unknown report schema versions produce an
explanatory UI state rather than a rendering failure. Keep the schema version
separate from the handler image version.

Disabling platform availability stops reporter injection for new runs, including
explicitly requested runs, which receive an unavailable status. It does not prevent
authorized reads of existing reports. Before rolling back to a backend that lacks
the new handler operations, drain or explicitly cancel active finalization work. Do not
delete report artifacts, plugin metadata, or task associations during rollback.

## Frontend Considerations

The bound Boolean is an ordinary pipeline input, so the existing run-creation input
form can display it with a false default and let users set it to true. No independent
UI toggle or client-only flag is needed. Preserve the input value when submitting
from the UI; backend validation and platform permission apply equally to SDK and UI
requests. A missing or false request does not show a pending report.

Add a run-level **Resource usage report** entry for explicitly requested runs,
showing pending, running, complete, partial, failed, or unavailable status and a
downloadable artifact link when one exists. Platform handler tasks are visible in
a separate system/finalization section rather than
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

These sampling/window labels come from the report's actual measurement metadata.
Display source provenance and any downsampling caveats for custom finalizers;
never assume all reports came from internal Prometheus.

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

The SDK gains the `PipelineConfig` binding and compiler validation, but local runners
do not generate utilization reports. `SubprocessRunner` and `DockerRunner` resolve
the input normally and do not pass through server-side policy or inject platform
handlers. The false default has no local effect. If the submitted root pipeline's
bound value is true during local execution, emit an explicit unsupported-reporting
warning and continue the workload without a reporter; do not claim a report will
appear. Add local regression tests for both values and for valid configuration
handling without cloud credentials.

Existing local behavior of user-defined exit handlers is unchanged. Document that
platform reports are a remote-deployment capability, not a portable pipeline output
guaranteed locally.

## Implementation History

- 2026-10-16: Initial draft covering platform-controlled exit handlers, resource
  report artifacts, and task-detail visualization. No implementation or release
  has been completed. Assign the KEP number when a tracking issue is created.
- 2026-10-18: Revise resource reporting to require an explicit Boolean pipeline input
  defaulting to `False`, overridden through SDK run `arguments`. Add binding,
  validation, compatibility, and SDK/UI/local test requirements. Specify
  administrator-replaceable finalizer images and provider-independent inventory,
  report, and publication contracts for alternate metrics sources.

## Drawbacks

- Each enabled handler adds execution overhead and bounded completion latency,
  even though it cannot change the user pipeline's result. Resource reporting pays
  this cost only for explicitly opted-in runs.
- Authors must add and bind a Boolean input once before users can request reports;
  older packages cannot opt in solely through a client-side argument.
- Outcome isolation, generation fencing, and handler authentication add complexity
  beyond a single user-authored exit task.
- Reporting depends on provider identity mappings, retention, sampling resolution,
  and artifact storage; KFP cannot guarantee complete measurements.
- Administrator-authored finalizer images are trusted code. Schema validation
  cannot establish the correctness of their measurements or protect a provider
  from code to which the administrator deliberately grants broad access.
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

This can prototype reporting, but requires authors to own the reporting task and
propagates monitoring configuration into user code. The proposed Boolean input and
binding express consent only; the platform retains implementation, configuration,
credentials, and execution policy. User exit handlers remain useful for
pipeline-specific cleanup.

### Use a client-only flag or an administrator-enabled default

A new `enable_resource_usage_report` client keyword would not be a declared pipeline
input and would create a second configuration path beside ordinary run arguments.
An administrator-enabled default would generate reports without per-run user
consent. Instead, keep the flag in the pipeline's typed input contract, default it
to false, and preserve its explicit binding in compiled packages. Platform
availability is necessary but never sufficient to enable resource reporting.

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

### Hard-code Prometheus or add a backend plugin for every metrics provider

Requiring all reporting to pass through a Prometheus parser excludes deployments
with other monitoring systems. Adding a backend provider interface and shipping a
new KFP backend for every source is unnecessary when administrators already control
the finalizer image. Keep a useful built-in implementation, expose execution
inventory independently, and standardize the report/publication contract instead.

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
