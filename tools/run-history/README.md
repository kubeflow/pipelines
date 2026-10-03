# Transfer completed run history (native runtime)

This administrator tool merges completed run history into a different, already
populated KFP database. The two installations continue to operate independently.
It does not move active workflows or require a shared live database.

This implementation is for the native task/artifact storage on `master`. The
`release-2.18` implementation uses a different archive format for ML Metadata.
**The formats are not interchangeable.** Use the tool from the same KFP revision
as the source and destination. Both API servers must include the imported-history
schema and runtime protections before using this tool. The tool never migrates
schemas. It checks column layouts and actual SQL column types, lengths, and
nullability, and rejects mismatches. Source and destination must use the same
database engine and compatible schema; this is not a MySQL/PostgreSQL converter.
Use the database backend supported by your KFP release.

## What is transferred

- Completed runs, original timestamps/statuses, parameters, embedded pipeline
  specifications, stored workflow manifests, and plugin data.
- Experiments, referenced SQL pipeline/version definitions, and their tags.
- Tasks and parent/child relationships, artifact metadata and task links,
  metrics, and artifact URIs. Lineage includes the selected runs; it does not
  recursively copy every other run that ever used the same artifact.
- The original source installation ID and a content digest for repeat detection.

Only finished runs are accepted. Export in batches of at most 100 explicitly
selected IDs and 128 MiB per file. A repeatable-read snapshot keeps each batch
consistent. Ensure the source's final reports have been persisted before its
last export; do not remove the source until the required records and external
files have been checked at the destination.

Recurring schedules are **not** imported or activated. Their original IDs remain
in the export, but imported run records have no destination recurring-run link.
When a catalog definition has already been deleted, an embedded run specification
is retained and missing catalog links are cleared. Kubernetes-native pipeline CRs
are not copied; migrate definitions separately if new executions need them.
Runs containing legacy MLMD context IDs are rejected rather than silently losing
legacy runtime details.

## Build and configure

From the repository root:

```sh
go build -o run-history ./tools/run-history
```

Set `KFP_HISTORY_DSN` through your secret-management process. The CLI reads it from
the environment so database credentials do not appear in command arguments.
`--dsn-env` selects a different environment variable; `--driver` is `mysql`
(default) or `postgres`. DSN syntax and TLS settings follow the respective Go
SQL drivers. Use source credentials with read-only access and destination
credentials with the required table read/insert permissions. This is an
administrator operation; it does not use the KFP API's tenant authorization.
Preserve namespace names and provision equivalent access in the destination.

## Export from the old installation

With `KFP_HISTORY_DSN` pointing to the source:

```sh
./run-history export \
  --source-id retired-cluster-a \
  --run-id RUN_UUID_1 --run-id RUN_UUID_2 \
  --file history-001.json
```

Choose a stable, unique installation ID (1–63 letters, digits, `.`, `_`, or `-`).
Export files are created with mode `0600`; an existing file is never overwritten.
They may contain sensitive parameters and metadata. Keep the source ID unchanged
for later batches. Export directly from the original installation; forwarding
previously imported records through another installation is not supported.

## Validate and import into the replacement

Switch `KFP_HISTORY_DSN` to the destination, then:

```sh
./run-history import --file history-001.json --name-prefix retired-a-
./run-history import --file history-001.json --name-prefix retired-a- --apply
```

Without `--apply`, the tool performs the merge in a transaction and rolls it back.
Dry runs execute insert statements and acquire database locks, but leave no
history changes. `--apply` commits the whole batch atomically. A conflict or
invalid relationship rolls back every insert in that batch. No Kubernetes or
object-store API is called by the tool.

`--name-prefix` prefixes experiment and pipeline names, avoiding common conflicts
such as two independent `Default` experiments. IDs, display names where separate,
and pipeline-version names remain unchanged. Alternatively, use
`--experiment-id DESTINATION_EXPERIMENT_UUID` to explicitly place all runs in an
existing experiment in the **same namespace**. Pipeline name collisions can still
be resolved with `--name-prefix`. There is no automatic namespace/ID remapping or
silent merge of unrelated same-name definitions.

Importing the identical history again skips existing runs. Source installation,
content digest, and experiment mapping must agree. Existing destination runs are
never replaced. Shared experiment/pipeline definitions can be reused by ID when
their identity matches; their destination descriptions and activity are not
overwritten. Changed run payloads, changed version specifications, conflicting
artifact metadata, and conflicting tag values fail explicitly. Do not bypass a
conflict with SQL `REPLACE` or by changing the source installation ID.

Both clusters can keep accepting work while you export and import completed-run
batches. Repeat the process for later completed runs, and perform a final catch-up
export when retiring the old installation. This is a history snapshot transfer,
not ongoing replication of edits or active runs.

## Imported-history behavior

The destination API server allows reading, archiving, unarchiving, and deleting
imported history. It rejects retry, termination, workflow reports, and runtime
task/link writes. Deletion removes database records without contacting Kubernetes.
Existing UI execution controls may display this server rejection; the initial
implementation does not add a separate UI badge. To execute the pipeline again,
create a **new** run with the required destination resources.

Imported tasks cannot be execution-cache candidates. Imported artifacts have
no reusable artifact identity key, so they neither collide with independently
registered destination datasets nor become find-or-create artifact identities.
The original fingerprints/identities remain in the archive for provenance.
Destination history-retention policies still apply to the original timestamps
and archive state; account for them before importing old history.

## Artifacts, logs, and recovery

An export contains **references and metadata**, not artifact files, pipeline
package objects, or pod logs. Keep referenced object storage accessible to the
replacement, including its credentials and endpoint configuration. If the old
cluster hosts the object store or uses local PVCs, preserve/copy those contents
before removing it. This version does not rewrite storage URIs.

Imported log reads use only the configured log archive, never a same-name pod in
the new cluster. Archive required logs while the source pods still exist, and
configure the replacement to read the same archive. Unarchived pod logs cannot
be recovered from this database export.

A failed import can be retried after its reported conflict is resolved; successful
batches can be retried without duplication. Keep the original export until run
graphs, task parameters, metrics, artifacts, and archived logs have been checked
in the replacement UI. Take a destination database backup before administrative
migration. Do not roll back the destination API server to a build lacking the
history protections while imported runs remain in its database.

## Verification

```sh
go test ./backend/src/apiserver/history ./tools/run-history
go test ./backend/src/apiserver/storage ./backend/src/apiserver/resource
```

SQLite fixtures cover populated destinations, repeat imports, conflicts,
transaction rollback/dry runs, graph validation, schema drift, JSON integer
precision, iteration zero, deleted definitions, and artifact identity collisions.
The backend CI supplies `KFP_HISTORY_MYSQL_TEST_DSN` and
`KFP_HISTORY_POSTGRES_TEST_DSN` for the same transfer against disposable services.
Those tests create/drop only randomly named databases or schemas; they are skipped
locally when the corresponding variable is unset.
