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

## Upgrade Go

The effective compiler in the root `go.mod` is the repository-wide Go version.
Update all managed module and builder-image pins with:

```bash
make update-go-version GO_VERSION=1.X.Y
```

Then review the diff and run `make check-go-version`. The deliberately narrow
managed forms and explicit non-goals are documented in
[`go-version-policy.md`](go-version-policy.md); register new locations instead
of expanding the updater into a language interpreter.

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
| `RUNS_GC_INDEX_BUILD_TIMEOUT` | Positive Go duration per retention-index build; defaults to `12h`, independent of the five-second metadata-lock wait |
| `RUNS_GC_INDEX_PREPARE_ONLY` | Boolean, default `false`; prepare indexes without archiving/deleting runs, even if retention windows are configured |

## Run garbage collection database migration

The API server automatically prepares `idx_run_gc_lifecycle` and
`idx_run_gc_archived` in the background when either retention window is enabled.
With retention disabled and `RUNS_GC_INDEX_PREPARE_ONLY=false`, it does not build
these indexes. API startup and readiness do not wait for retention-index creation. If retention
was already configured but paused due to missing indexes, this change can resume
archival/deletion automatically after preparation and the collection interval.
Use prepare-only mode to inspect the existing backlog before allowing catch-up.

The elected GC leader prepares indexes sequentially, also taking a database/schema
advisory lock so other installations sharing that database cannot race the build.
Existing valid, optimizer-visible indexes are reused. MySQL explicitly requires
`ALGORITHM=INPLACE, LOCK=NONE`; PostgreSQL uses `CREATE INDEX CONCURRENTLY` outside
a transaction. There is no column narrowing, table-copy fallback or automatic
replacement of an incompatible index. Metadata-lock waits are bounded to five
seconds. Online builds still consume I/O, CPU and disk, and pending metadata locks
can briefly delay application writes.

`RUNS_GC_INDEX_BUILD_TIMEOUT` defaults to `12h` **per index**; use a positive Go
duration such as `24h` for a rehearsed large-table build. It does not increase the
metadata-lock wait. Invalid values fail configuration validation. Cancellation or
leadership loss cancels the client request. Database cleanup is not instantaneous:
MySQL can retain the operation and session advisory lock after client cancellation;
other preparers defer until that session releases ownership. The configured timeout
bounds the client attempt, not server-side rollback/cleanup duration. Failures leave GC paused and retry with
exponential backoff and jitter (about one minute initially, capped at fifteen
minutes). API traffic continues; no archive/delete operation runs in preparation.

After successful preparation, a newly elected leader waits a full
`RUNS_GC_INTERVAL` before its first collection. Creating a missing index or recovering
from a failed preparation also restarts that wait. Thus a long build cannot cause
an overdue ticker to launch catch-up immediately. Both indexes are checked before
collection; unusable indexes pause retention.

### Prepare first, enable retention later

For a large table, rehearse against a populated snapshot on comparable hardware
with concurrent traffic. Size disk headroom for the new indexes, temporary files
and concurrent-change logs; measure latency and replica lag. A build timeout is
not an I/O throttle or an estimate of build duration.

1. Set `RUNS_GC_INDEX_PREPARE_ONLY=true` on **all API replicas**. Retention windows
   can stay disabled while indexes are prepared. A true value prevents this
   version from archiving or deleting even when windows are configured. Invalid
   boolean values fail validation instead of enabling collection.
2. Wait for both `Prepared retention index ...` logs (or verify existing valid
   definitions), and the preparation-completed message. Inspect database load and
   any retry diagnostics. Preparation uses the existing GC leader-election RBAC.
3. When ready for retention catch-up, configure the desired windows and set
   `RUNS_GC_INDEX_PREPARE_ONLY=false` consistently. The new leader still waits one
   full collection interval. Catch-up is a separate workload: it drains eligible
   runs in bounded batches and can take substantial time on old history.

A rolling change to prepare-only cannot stop an old leader that has not received
that configuration or does not implement it. Start with retention disabled for
first adoption, and finish the configuration rollout before relying on the mode.
Keep the mode consistent across installations sharing the database. Disabling
retention does not remove indexes; existing valid indexes permit rollback to the
previous collector. This mode and timeout apply only to retention indexes, not
other startup migrations or the recurring-run lookup index.

### Inspect and recover failed preparation

Ordinary supported upgrades no longer require manual index SQL. Investigate
permissions, disk capacity, lock contention and engine online-DDL support when
preparation fails. Increasing the build timeout cannot resolve the separate
five-second lock wait. A canceled PostgreSQL concurrent build can leave an invalid
index. The preparer automatically drops and rebuilds an abandoned index only when
it matches the exact managed retention-index definition. Both operations use
`CONCURRENTLY`, share the per-index build deadline, and keep GC paused. Active DDL
causes preparation to defer; conflicting definitions, custom storage options and
nondefault tablespaces remain intact for inspection. A configured nondefault
`default_tablespace` can therefore require coordinated manual recovery.

The names `idx_run_gc_lifecycle` and `idx_run_gc_archived` are managed by the
preparer. API preparers coordinate with a schema-scoped advisory lock. Manual DDL
does not automatically honor that lock, and an activity check cannot exclude an
administrator starting DDL immediately afterward. Before manually changing these
indexes, disable retention and prepare-only across all preparers and let active
work stop. Do not run uncoordinated manual index builds alongside automation.

The SQL below is for inspection and coordinated recovery, not a prerequisite for
normal enablement. Never retry failed online DDL with a blocking/copying algorithm.

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
maintenance operation before applying the command above. Ordinary interrupted
PostgreSQL builds with matching definitions recover automatically; manual repair
is reserved for conflicting definitions or persistent database failures. For
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

`TENSORBOARD_PROXY_SIGNING_SECRET` controls the HMAC key for scoped TensorBoard
proxy paths. The default Kustomize installation uses an initialization Job to
populate the persistent `ml-pipeline-ui-tensorboard-proxy` Secret only when its
key is absent, then injects the shared key into every UI replica. The UI uses
`Recreate` by default to prevent old and shared-key UI pods from overlapping
during the first migration. Preserve the Secret across upgrades so proxy URLs
remain valid. After the initial rollout completes, operators can
[enable rolling UI updates](../operator-guides/server-config.md#enabling-rolling-ui-updates)
in a separate apply or GitOps sync. See the
[operator guide](../operator-guides/server-config.md#tensorboard-proxy-signing-secret)
for initialization, recovery, GitOps, and rotation guidance.

Outside the default manifests, leaving this variable unset generates a
cryptographically random, process-local key at startup for local development.
Custom deployments need a shared key before enabling multiple replicas or
rolling updates. Use a dedicated random secret of at least 32 UTF-8 bytes;
the frontend rejects shorter values and values matching `MINIO_SECRET_KEY`.
