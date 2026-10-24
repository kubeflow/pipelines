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

The current worker layout uses an OS-held installation lock under
`KFP_V2_RUNTIME_WORKSPACE_ROOT`. Run one worker per installation. API-only operation
uses `KFP_V2_RUNTIME_AUTO_EXECUTE=false`. A Kubernetes replica's private filesystem
is not a distributed lock: do not scale embedded workers across replicas without
the proposal's database-lease coordination. Kubernetes deployments must use a
single replica and the `Recreate` deployment strategy so upgrades do not overlap
embedded workers. The coordinator CI deployment action applies that strategy only
when opting into the non-default runtime.

## Validation

```shell
go test -race ./backend/src/v2/runtime/poc
go test ./backend/src/v2/cmd/launcher-v2 ./backend/src/v2/config

TEST_LABEL='E2ECritical || E2EEssential || E2EParallelNested || E2EFailure' \
  NUM_NODES=3 ./backend/test/local_docker_e2e/run_local_docker_e2e.sh
TEST_LABEL=E2EProxy RUN_PROXY_TESTS=true \
  ./backend/test/local_docker_e2e/run_local_docker_e2e.sh
```

The Docker harness uses its own SQLite database, WAL/busy handling, artifact
directory, and installation lock. Kubernetes E2E runs use the existing
`backend/test/end2end` suites against a coordinator-enabled deployment.
