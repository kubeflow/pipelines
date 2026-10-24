# Coordinator runtime

This package ports the existing direct-DAG runtime to the workload and workspace
boundaries described in the [local-runtime proposal](../../../../../proposals/kfp-local/README.md).
Argo remains the default. Enable the coordinator on the API server with:

```shell
KFP_V2_RUNTIME_MODE=coordinator-poc
KFP_V2_RUNTIME_EXECUTOR=kubernetes
```

The Kubernetes execution port and Docker backend share these contracts. The
[optional native packaging proof](../../../../../local/README.md) bundles the
API/embedded Docker worker and UI. Subprocess execution and published multi-platform
wheels remain outside this implementation; existing Argo and Docker coverage is retained.

## Ownership

The coordinator interprets the pipeline, resolves inputs, materializes loop
iterations, applies conditions and caching, and owns retries and public task
completion. Independent tasks run concurrently. Iterator parallelism and each
executor's active-task capacity are independent limits; waiting for retry backoff
does not occupy an execution slot.

Infrastructure retries receive bounded priority over fresh attempts, capped
exponential backoff with jitter, and never shorten the pipeline's configured
backoff. User-computation retries keep normal priority. Fatal failures bypass
every automatic retry policy, including `Always`.

Executors receive an immutable, resolved, task-scoped request. They submit, observe,
recover, and cancel workloads without receiving runs, pipeline jobs, upstream task
collections, or database clients. Kubernetes receives a prepared Pod. Docker
receives the container workload, staged artifacts, and required mounts. Unsupported
requirements in any reachable component, including unselected conditional branches
and loop bodies, fail centralized capability validation before provisioning or
execution. Workspace compatibility is an explicit executor capability.

`Wait` returns a terminal task result, not a computation/publication handoff.
Executors own shared staging, bounded transfer retries, and output publication;
Docker runs the component command directly without an injected Go launcher.
Cancellation confirms, defers, or cannot establish termination. Only confirmation
releases execution capacity. Logs and cleanup are required plugin methods;
cleanup is separate from cancellation and cannot discard unretained log sources.

Workspaces are provisioned only when requested by the pipeline. PVC requirements
use the same validation and field-wise overrides as Argo. Tasks that do not use
the workspace receive no attachment. Workspace cleanup never deletes published
artifact storage.

Docker staging is released only after workload termination and publication.
Root-created workspace or staging directories may need a restricted cleanup
container; offline Docker workers must preload `alpine:3.23`. That helper mounts
only the validated run-owned directory, does not follow symlinks, and has bounded
resource and execution limits.

## Persistence and recovery

The selected executor and workspace provisioner are persisted on the run.
Every non-terminal materialized task has a `runtime_tasks` scheduling row, with a
NULL attempt until execution is admitted. Cached and skipped tasks remove that row
atomically with their public result and output links. Public task
parameters, diagnostics, artifact metadata, and log references remain in their
existing tables.

An attempt identity and protobuf-safe execution request are durable before
submission. The execution reference is recorded before observation. Recovery
distinguishes live work, completed work, confirmed termination, and an unknown
outcome. Unknown outcomes remain unresolved; they do not authorize resubmission.
Canceling observation during worker shutdown does not cancel the workload.

Executors checkpoint internal completion and transfer progress as opaque,
backend-owned execution-reference data. The coordinator does not track publication
phases or an upload retry counter. Transfer retries have a bounded budget and never
spend computation retries. Exhaustion after successful computation is a fatal
infrastructure failure; only an explicit user-requested run retry may repeat it. Task success
and removal of the runtime row occur together, after required publication and
executor cleanup. A stale completion cannot recreate a deleted attempt or
overwrite a replacement attempt.

Each loop iteration has its own task identity and retains the pipeline's logical
scope path. Retries preserve that task identity but receive a new attempt identity.
Recovery resumes an existing attempt before checking the cache.

Startup reserves all recoverable execution capacity before admitting queued work.
Changing the configured executor cannot reroute unfinished work: incompatible
persisted selections fail explicitly, retaining execution references for cleanup.
The task-log endpoint selects the current attempt; earlier attempts remain linked
artifacts rather than being presented as the current attempt's live output.

### User-requested run retry

The normal run-retry API reauthorizes execution and takes Kubernetes ownership
before reopening a failed or canceled run. Retry requires prior execution cleanup
to be complete: unknown attempts and retained resources are not discarded to make
the request succeed. Run state, retry generation, selected task resets, and fresh
WAITING rows change in one transaction. Each dispatched task receives a new
attempt identity.

Independent successful or cached tasks are reused. Failed/skipped work, affected
downstream tasks, and composite ancestors are reevaluated. A run that used a
workspace instead reconstructs the workspace and recomputes all tasks without
cache reuse; old ephemeral workspace files cannot be assumed to survive cleanup.
This explicit action can repeat computation and external side effects. It is not
an automatic retry of a fatal publication failure.

### Kubernetes ownership and takeover

Kubernetes workers coordinate through `run_leases` in the shared metadata
database. Each process has a distinct owner ID. Acquisition increments a
monotonic generation; releasing a lease clears ownership without resetting its
generation. Lease expiry and validation use database time, not a worker's clock.
The row remains until the run is deleted.

Each run coordinator receives immutable lease-scoped store views. Its task,
runtime, run, artifact, and artifact-link mutations check the current unexpired
owner and generation in the same transaction as the write. Launcher reports
remain fenced by attempt identity: a workload may legitimately keep reporting
while a replacement worker recovers that same attempt.

Workers claim bounded batches only when the configured executor has capacity.
Unlimited task capacity does not allow an unbounded claim batch. Renewal runs
independently of workload observation and retry waiting. Losing ownership cancels
observation and releases only local capacity accounting; it does not cancel or
delete the workloads that another worker may recover.

API-only replicas use `KFP_V2_RUNTIME_AUTO_EXECUTE=false`. Submission persists
initialization metadata in the same insertion that makes the run claimable; it
does not perform a later unleased update over a worker's progress. A lease holder
provisions the workspace and materializes tasks. Cancellation is persisted through
the API and handled by the owner or a subsequent lease holder, even when no new
task-execution capacity is available. Archiving changes visibility, not execution
or takeover eligibility. Explicit deletion waits for owned cancellation and
cleanup before deleting recovery metadata.

Kubernetes replicas may use normal rolling updates. Multi-host installations
require shared MySQL or PostgreSQL, access to the same Kubernetes backend, and
shared artifact storage. SQLite lease tests cover workers sharing one local
database, not database files shared between hosts. Leases do not provide
exactly-once external side effects.

When upgrading the unreleased single-worker proof of concept, stop its embedded
workers before starting lease-enabled workers: the older binary does not
participate in lease fencing. Normal rolling updates apply between lease-enabled
versions.

Kubernetes cancellation stops scheduled Pods with an active deadline, retaining
their log source across worker loss until logs can be published. Unscheduled
Pods can be deleted because no workload has started. Platform PVC creation uses
the materialized task identity, so recovery does not allocate a second claim.

Docker continues to use an OS-held installation lock under
`KFP_V2_RUNTIME_WORKSPACE_ROOT`; run only one Docker worker per installation.

### Offline cleanup

The API binary has a cleanup-only control command. Use the existing installation's
database, artifact-store, executor, and workspace-root configuration:

```shell
KFP_V2_RUNTIME_MODE=coordinator-poc \
KFP_V2_RUNTIME_EXECUTOR=docker \
KFP_V2_RUNTIME_AUTO_EXECUTE=false \
apiserver --config /path/to/config --runtimeCleanup
```

This path never starts HTTP/RPC servers, webhooks, GC, scheduled-workflow
reconciliation, or an execution worker. Docker cleanup must acquire the existing
worker lock. Kubernetes cleanup acquires each terminal run's maintenance lease
without waiting for a competing owner, uses lease-scoped stores, renews ownership,
and fences backend operations. It does not rely on a shared filesystem lock or
make SQLite safe across hosts.

Only terminal managed runs for the configured executor are considered.
Finalization requires a matching attempt, a durable terminal result and logs,
and the executor's identity-checked, terminal-only `Cleanup`. Cancellation
finalization requires its retained log checkpoint. Unknown outcomes, active
workloads, and incomplete publication are retained with a nonzero diagnostic;
cleanup never calls `Cancel`, `Submit`, or `Wait`. The normal worker must resolve
non-finalizing attempts before cleanup can release their workspace or staging.
No published artifact, log, public task, or run is deleted, and the run's outcome
and finish time remain unchanged.

`--runtimeCleanupKeepWorkspaces` preserves requested workspaces;
otherwise existing workspace lifecycle cleanup applies. Workspace handles remain
as historical/recovery identity, and PVC deletion must be observed before staging
cleanup is considered complete. JSON counts distinguish skipped/reconciled runs,
finalized attempts, retained workspaces, and failures. Repeating cleanup is safe;
already-clean runs count as reconciled. The default total timeout is two minutes,
configurable with `--runtimeCleanupTimeout`.

## Validation

```shell
go test -race ./backend/src/v2/runtime/poc
go test ./backend/src/v2/cmd/launcher-v2 ./backend/src/v2/config

KFP_USE_COORDINATOR_RUNTIME=true \
  go run github.com/onsi/ginkgo/v2/ginkgo -v \
  --label-filter=CoordinatorRuntimeMultiReplica ./backend/test/end2end -- \
  -namespace=kubeflow -apiUrl=http://127.0.0.1:8888 -apiScheme=http

TEST_LABEL='E2ECritical || E2EEssential || E2EParallelNested || E2EFailure' \
  NUM_NODES=3 ./backend/test/local_docker_e2e/run_local_docker_e2e.sh
TEST_LABEL=E2EProxy RUN_PROXY_TESTS=true \
  ./backend/test/local_docker_e2e/run_local_docker_e2e.sh
```

The Docker harness uses its own SQLite database, WAL/busy handling, artifact
directory, and installation lock. Kubernetes E2E runs use the existing
`backend/test/end2end` suites against a coordinator-enabled deployment.
