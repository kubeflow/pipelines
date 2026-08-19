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
| `ARCHIVED_RUNS_RETENTION_TIME` | Auto-delete archived runs after this Go duration (e.g., `2160h`); empty disables |
| `RUNS_GC_INTERVAL` | Poll interval for the run garbage collector; defaults to `6h` |
| `RUNS_GC_BATCH_SIZE` | Maximum rows per GC batch; defaults to `100` |

## Run garbage collection database migration

Before enabling either retention setting, create the required lifecycle index
once using the same database and schema as the API server. The API server only
validates the index at startup; it does not run heavyweight DDL from every
replica.

For PostgreSQL, inspect the existing index first. Replace `public` if the API
server's current schema is different:

```sql
SELECT index_metadata.indisvalid, index_metadata.indisready,
       pg_get_indexdef(index_metadata.indexrelid)
FROM pg_index AS index_metadata
JOIN pg_class AS index_class ON index_class.oid = index_metadata.indexrelid
JOIN pg_namespace AS index_namespace ON index_namespace.oid = index_class.relnamespace
WHERE index_namespace.nspname = 'public'
  AND index_class.relname = 'idx_run_gc_lifecycle';
```

If no row is returned, create the index outside a transaction:

```sql
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_run_gc_lifecycle ON public.run_details ("StorageState", "FinishedAtInSec");
```

For MySQL, explicitly require online DDL so the command fails instead of falling
back to a table-copying or write-blocking operation:

```sql
ALTER TABLE run_details ADD INDEX idx_run_gc_lifecycle (StorageState, FinishedAtInSec), ALGORITHM=INPLACE, LOCK=NONE;
```

If a same-named index has different columns, remove it in a coordinated
maintenance operation before applying the command above. A failed PostgreSQL
concurrent build can leave an invalid index; remove only that invalid index with
`DROP INDEX CONCURRENTLY public.idx_run_gc_lifecycle` before retrying. For
MySQL, use `SHOW INDEX FROM run_details` first and run the `ALTER TABLE` only
when the exact index is absent.

**Note on archive query performance:** The `idx_run_gc_lifecycle` index on
`(StorageState, FinishedAtInSec)` efficiently serves the delete pass
(`StorageState = 'ARCHIVED'`). The archive pass uses `StorageState NOT IN (...)`
which may not drive a range scan on all engines. For large tables (>1M rows),
verify with `EXPLAIN` and consider an additional index on
`(FinishedAtInSec, StorageState)` if the archive pass shows a full table scan.

### Argo Workflow cleanup

The run garbage collector only deletes database rows. Argo Workflow custom
resources are left behind. To avoid misleading persistence-agent log entries
("workflow does not have a valid RunID") after GC deletes a run's DB record,
configure Argo's native workflow TTL or set `ttlSecondsAfterFinished` on your
workflow templates. The TTL should be shorter than `ARCHIVED_RUNS_RETENTION_TIME`
so Argo cleans up the CR before GC deletes the database row.

`TENSORBOARD_PROXY_SIGNING_SECRET` is optional; it defaults to `MINIO_SECRET_KEY`.
