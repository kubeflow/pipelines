# Development and Deployment

## Python setup

Use a `.venv`:

```bash
python3 -m venv .venv
source .venv/bin/activate
python -m pip install -U pip setuptools wheel
make -C api python-dev
make -C kubernetes_platform python-dev
pip install -e api/v2alpha1/python --config-settings editable_mode=strict
pip install -e sdk/python --config-settings editable_mode=strict
pip install -e kubernetes_platform/python --config-settings editable_mode=strict
```

Install Ginkgo into the repository when running its suites:

```bash
make ginkgo
export PATH="$PWD/bin:$PATH"
```

## Local clusters

| Need | Command |
| --- | --- |
| Standalone development cluster | `make -C backend kind-cluster-agnostic` |
| API-server development cluster | `make -C backend dev-kind-cluster` |

Standalone mode is single-user and unauthenticated. Multi-user deployments require an identity provider, namespace isolation, and Istio; see [`manifests/kustomize/README.md`](../../manifests/kustomize/README.md).

Both targets wait for cluster components to become ready with configurable timeouts (defaults unchanged; override for slow connections or resource-constrained CI): `MYSQL_WAIT_TIMEOUT` (default `10m`), `CERT_MANAGER_WAIT_TIMEOUT` (default `300s`), `METADATA_GRPC_WAIT_TIMEOUT` (default `3m`), `ML_PIPELINE_WAIT_TIMEOUT` (default `3m`), e.g. `make -C backend kind-cluster-agnostic MYSQL_WAIT_TIMEOUT=20m`.

## Environment variables

| Variable | Purpose |
| --- | --- |
| `_KFP_RUNTIME=true` | Runtime mode that disables most SDK imports |
| `VITE_NAMESPACE` | Frontend namespace for multi-user development |
| `LOCAL_API_SERVER=true` | Local API-server integration-test mode |
| `FRONTEND_SERVER_NAMESPACE` | Namespace used by a local frontend server |
| `MINIO_ENDPOINT_REWRITE` | Rewrites object-store endpoints in local proxy mode |
| `MAX_METRICS_FILE_BYTES` | Maximum uncompressed metrics JSON size; defaults to 1 MiB |
| `RUNS_RETENTION_TIME` | Auto-archive terminal runs after this Go duration (e.g., `720h`); empty disables |
| `ARCHIVED_RUNS_RETENTION_TIME` | Auto-delete archived runs this Go duration after they were archived (e.g., `2160h`); pre-upgrade archived rows fall back to completion time; empty disables |
| `RUNS_GC_INTERVAL` | Poll interval for the run garbage collector; defaults to `6h` |
| `RUNS_GC_BATCH_SIZE` | Maximum rows per GC batch; defaults to `100` |

## Run garbage collection database migration

Before enabling either retention setting, create the two required indexes
once using the same database and schema as the API server. The API server only
validates the indexes (at startup and on every collection tick); it does not
run heavyweight DDL from every replica.

For PostgreSQL, inspect the existing index first. Replace `public` if the API
server's current schema is different:

```sql
SELECT index_metadata.indisvalid, index_metadata.indisready,
       pg_get_indexdef(index_metadata.indexrelid)
FROM pg_index AS index_metadata
JOIN pg_class AS index_class ON index_class.oid = index_metadata.indexrelid
JOIN pg_namespace AS index_namespace ON index_namespace.oid = index_class.relnamespace
WHERE index_namespace.nspname = 'public'
  AND index_class.relname IN ('idx_run_gc_lifecycle', 'idx_run_gc_archived');
```

If a row is missing, create that index outside a transaction:

```sql
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_run_gc_lifecycle ON public.run_details ("StorageState", "FinishedAtInSec");
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_run_gc_archived ON public.run_details ("StorageState", "ArchivedAtInSec");
```

For MySQL, explicitly require online DDL so the command fails instead of falling
back to a table-copying or write-blocking operation:

```sql
ALTER TABLE run_details ADD INDEX idx_run_gc_lifecycle (StorageState, FinishedAtInSec), ALGORITHM=INPLACE, LOCK=NONE;
ALTER TABLE run_details ADD INDEX idx_run_gc_archived (StorageState, ArchivedAtInSec), ALGORITHM=INPLACE, LOCK=NONE;
```

If a same-named index has different columns, remove it in a coordinated
maintenance operation before applying the command above. A failed PostgreSQL
concurrent build can leave an invalid index; remove only that invalid index with
`DROP INDEX CONCURRENTLY public.idx_run_gc_lifecycle` before retrying. For
MySQL, use `SHOW INDEX FROM run_details` first and run the `ALTER TABLE` only
when the exact index is absent.

**Note on query/index alignment:** `idx_run_gc_lifecycle`
`(StorageState, FinishedAtInSec)` drives the archive pass (positive `IN` list
of non-archived storage states plus `IS NULL`) and the delete pass's legacy
predicate (rows archived before `ArchivedAtInSec` existed, which only
shrinks). `idx_run_gc_archived` `(StorageState, ArchivedAtInSec)` drives the
delete pass's archival-time predicate, so a mass archival does not cause the
delete pass to re-scan the old finish-time range every tick while it waits
out the observation window. The collector re-validates both indexes against
the database catalog on every tick: applying this migration takes effect
without an API-server restart, and GC pauses automatically if either index
is dropped.

### Argo Workflow cleanup

The run garbage collector only deletes database rows. Argo Workflow custom
resources are left behind. To avoid misleading persistence-agent log entries
("workflow does not have a valid RunID") after GC deletes a run's DB record,
configure Argo's native workflow TTL or set `ttlSecondsAfterFinished` on your
workflow templates. The TTL should be shorter than `ARCHIVED_RUNS_RETENTION_TIME`
so Argo cleans up the CR before GC deletes the database row.

The API server only changes a run or deletes a Workflow after fetching the live
Workflow and matching its immutable UID, resource version, run label, namespace,
and recurring-run owner. Workflow deletion uses UID and resource-version
preconditions so a same-name replacement or concurrently retried Workflow is
never removed. If a one-time Workflow has no database run, the API server waits
out the run-creation grace period and then safely deletes the UID-verified live
orphan. A missing or inconsistent recurring-run owner remains fail-closed
because its tenant cannot be established. If orphan propagation strips the
owner while the recurring-run row still exists, the report is rejected
permanently and counted as an identity mismatch instead of being retried
indefinitely. Configure Argo's workflow TTL (one hour in the shipped
configuration) and explicitly inspect these orphans.

Rejected reports are logged and counted by
`resource_manager_workflow_reports_rejected_total`, with
`reason="ownership_unresolved"`, `reason="namespace_mismatch"`, or
`reason="identity_mismatch"`. Alert on these reasons and repair legacy database
ownership before retrying a multi-user report. Standalone single-user mode may
recover the execution namespace from the live Workflow for old rows that have
no persisted namespace; multi-user namespace mismatches always fail closed.
Transient Kubernetes lookup failures and terminal reports accepted through a
stored-identity fallback are not counted as rejections. Workflow reports do
perform synchronous identity reads, so Kubernetes API outages delay report
persistence until those reads succeed.

New runs persist their actual Kubernetes execution namespace even in
single-user mode. API resource references therefore expose that namespace
instead of an empty namespace; legacy empty-namespace rows remain supported by
resolving their namespace from the reporting Workflow.

The garbage collector also does not remove rows from the `artifacts` table,
MLMD records, or object-store artifacts; those lifecycles are managed
separately (see `ARTIFACT_RETENTION_DAYS` for object-store artifacts).

`TENSORBOARD_PROXY_SIGNING_SECRET` is optional; it defaults to `MINIO_SECRET_KEY`.
