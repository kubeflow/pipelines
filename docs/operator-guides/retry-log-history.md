# Retry log history

Manual **Retry run** keeps the run and logical task identities. Before resetting
an unsuccessful task, the API server saves its troubleshooting metadata and log
artifact references in `task_log_history`. Successful, cached, and skipped tasks
are retained without a reset. The current task continues to describe only its
current execution; historical output is available through its log history.

Each record identifies the manual retry generation that caused the snapshot.
Automatic retry attempts remain distinguishable by their executor pods and log
artifacts within that generation. A generation is an identity, not a guarantee
that the workflow operation completed: a retry claim may fail after the snapshot.
Repeated delivery must preserve the original snapshot rather than replace it
with a partially reset task.

## Upgrade and rollback

The schema addition is a standalone table with a composite task/generation key
and run lookup index. It has no foreign keys and does not alter existing tables
or backfill historical runs. Startup creates it under database advisory ownership
on one physical connection. Ownership acquisition fails immediately when another
replica holds it; metadata lock waits are limited to one second and the migration
has a ten-second context deadline. Failure aborts startup with an actionable error;
resolve contention and retry startup. Session settings and locks are restored,
and connections whose cleanup fails are discarded.

PostgreSQL creates the table and index in a transaction. MySQL creates the table
with its index in the same DDL statement. Existing history tables are not
reconciled by broad `AutoMigrate`. These bounds apply to the new history-table
operation, not to unrelated schema reconciliation in the release. Database
operations still acquire brief catalog/metadata locks; this is not a promise of
zero locking or zero downtime.

Roll out the API server and new launcher/runtime together. New launchers write
executor logs to `.../attempts/<pod UID>/executor-logs-<automatic retry index>`.
This prevents a manual retry that reuses a successful driver's output prefix
from overwriting an earlier executor's object. Existing archived URIs remain
valid. Older launchers can still overwrite their reused paths, so snapshot
preservation alone cannot guarantee immutable log contents during a mixed-runtime
rollout.

Rolling back the API server leaves the additive table intact but stops recording
and exposing new history. Old-server run deletion does not clean this new table;
operators who delete runs while rolled back should account for orphan history
rows before eventual data-retention cleanup. After all old servers are retired,
identify rows whose `RunUUID` has no matching `run_details.UUID` and remove them
in bounded batches under the deployment's normal database maintenance process.
Review counts first; this metadata cleanup must not delete object-store blobs. Do not drop the table as part of
routine rollback. Forward rollout can resume recording without reconstructing
already-cleared references.

## Retention and access

History preserves references, not a second copy of log bytes. Kubernetes pod
collection, object-store lifecycle policies, missing objects, and disabled log
publication still affect availability. Run deletion on the new server removes
its history in the same database transaction as task deletion. Artifact/object
retention is otherwise unchanged. History is exposed through the existing
run/task authorization boundary; it does not create a public log endpoint.

Existing references or blobs already removed or overwritten before this upgrade
cannot be recovered by this migration. The guarantee applies going forward once
the updated server and runtime are active.

Namespace export/import does not currently include the separate retry history
table. Moving a run through that mechanism preserves its supported current-task
metadata, but does not transfer historical retry snapshots. Keep the source
installation and its archive retention available when that history is needed;
a future archive-format change is separate from this migration.

## Verification

The migration regression test uses `KFP_RECURRING_MYSQL_TEST_DSN` and
`KFP_RECURRING_POSTGRES_TEST_DSN`, creates isolated databases/schemas, and exercises
`autoMigrate` against a populated immediately preceding schema. It checks live
read/write completion during startup, repeated startup, retained values,
competing migration ownership, cancellation, and recovery. Run it with:

```sh
go test ./backend/src/apiserver/client_manager -run TestRetryLogHistoryMigrationProductionDatabases -count=1 -v
```

The output reports completed concurrent request pairs and maximum observed
latency. Storage lifecycle tests cover snapshot/reset atomicity, repeated manual
retries, successful siblings, artifact references, and deletion. Launcher tests
cover immutable paths across automatic and manual retries without changing
non-log outputs. These tests do not establish availability for every historical
schema variant or unrelated release migration.
