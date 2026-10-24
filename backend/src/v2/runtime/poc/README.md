# Coordinator runtime

This package ports the existing direct-DAG runtime to the workload and workspace
boundaries described in the [local-runtime proposal](../../../../../proposals/kfp-local/README.md).
Argo remains the default. Enable the coordinator on the API server with:

```shell
KFP_V2_RUNTIME_MODE=coordinator-poc
KFP_V2_RUNTIME_EXECUTOR=kubernetes
```

This is the Kubernetes execution port, not the local Python distribution. It does
not add subprocess execution, CLI packaging, or new multi-tenant authorization.
The existing Docker development backend and its E2E coverage are retained.

## Ownership

The coordinator interprets the pipeline, resolves inputs, materializes loop
iterations, applies conditions and caching, and owns retries and public task
completion. Independent tasks run concurrently. Iterator parallelism and each
executor's active-task capacity are independent limits; waiting for retry backoff
does not occupy an execution slot.

Executors receive an immutable, resolved, task-scoped request. They submit, observe,
recover, and cancel workloads without receiving runs, pipeline jobs, upstream task
collections, or database clients. Kubernetes receives a prepared Pod. Docker
receives the container workload, staged artifacts, and required mounts. Unsupported
Docker requirements fail explicitly instead of being ignored.

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
`runtime_tasks` contains non-terminal scheduling and attempt state. Public task
parameters, diagnostics, artifact metadata, and log references remain in their
existing tables.

An attempt identity and protobuf-safe execution request are durable before
submission. The execution reference is recorded before observation. Recovery
distinguishes live work, completed work, confirmed termination, and an unknown
outcome. Unknown outcomes remain unresolved; they do not authorize resubmission.
Canceling observation during worker shutdown does not cancel the workload.

Completed computation is checkpointed before publication. Publication retries
have a separate, bounded budget and never spend computation retries. Task success
and removal of the runtime row occur together, after required publication and
executor cleanup. A stale completion cannot recreate a deleted attempt or
overwrite a replacement attempt.

Each loop iteration has its own task identity and retains the pipeline's logical
scope path. Retries preserve that task identity but receive a new attempt identity.
Recovery resumes an existing attempt before checking the cache.

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
