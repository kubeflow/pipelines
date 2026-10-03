# Completed run history transfer for release 2.18

This administrative tool copies one **completed** run from a KFP 2.18 installation
to another installation with its own populated MySQL database and ML Metadata
(MLMD) store. The installations can continue running independent workloads. It
does not move running workflows, require a shared database, or create Kubernetes
resources. Repeat export/import for additional finished runs, then remove the
old cluster when its required history and durable objects have been retained.

The archive format is `kfp-history-mlmd-2.18/v1`. It deliberately rejects the
post-MLMD generation on master. Source and destination must use matching 2.18 SQL
schemas, the SQL pipeline catalog, and compatible MLMD types. This release tool
supports MySQL deployments; it does not enable PostgreSQL production support on
the 2.18 release line. Upgrade the destination to a build containing the imported
history guards and run its normal database migrations before importing.

## Contents and behavior

An archive contains the selected run, its experiments, referenced pipeline
definitions/versions and tags, task records, metrics, resource references, and
MLMD contexts, executions, artifacts, events, associations, attributions, and
parent relationships. The MLMD graph includes cached execution references and
artifact-producing ancestors, but does not follow unrelated consumers. Export
rejects unfinished referenced executions, missing dependencies, and graphs over
100,000 nodes. The archive is limited to 256 MiB on import.

SQL run, task, experiment, pipeline, and version UUIDs are retained. MLMD integer
IDs are remapped, including task execution/artifact references and execution
parent/cache references. The `system.PipelineRun` context keeps the original run
UUID as its name because the 2.18 UI uses that lookup. Other imported node names
are isolated by the stable source installation ID. Original MLMD node values are
retained in the `kfp_history_original` custom property. MLMD controls native node
creation/update timestamps, so those timestamps show import time; original
timestamps remain in that property and in the archive. SQL run/task times and
event times retain the source values. Runtime manifests are retained for archived
logs and provenance; they are not rewritten into runnable destination manifests.

Imported runs have internal `ImportedFrom` and `ImportDigest` markers. The API
server rejects retry and cancellation, reads only archived logs, and deletes only
history records without touching local workflows. Submit a new run to execute the
pipeline on the destination. Source recurring-run metadata remains in the archive,
but **no job rows or schedules are imported** and the destination run's recurring
run link is cleared. Even disabled historical job rows could otherwise reconcile
against an unrelated destination schedule with the same Kubernetes name.

Imported SQL task fingerprints are cleared, including their hydrated payloads,
and MLMD cache fingerprints are isolated. Imported history is not an execution
cache. Source cache databases are not copied. Artifact URIs, parameters, model
metadata, and lineage remain available as history.

## Connections and commands

Use a Python 3.11 environment on Linux with the optional operator dependencies:

```sh
python3 -m venv /tmp/kfp-history-env
/tmp/kfp-history-env/bin/pip install -r tools/run-history/requirements.txt
```

Keep connection files private. MySQL connection files use
`mysql.connector.connect` options, for example:

```json
{
  "driver": "mysql",
  "host": "mysql.example.internal",
  "port": 3306,
  "database": "mlpipeline",
  "user": "history_operator",
  "password": "SET_IN_PRIVATE_FILE",
  "ssl_ca": "/path/to/mysql-ca.pem",
  "ssl_verify_cert": true,
  "ssl_verify_identity": true
}
```

An MLMD connection file uses gRPC TLS by default:

```json
{
  "target": "metadata.example.internal:443",
  "ca_file": "/path/to/mlmd-ca.pem",
  "timeout_seconds": 60
}
```

Optional `cert_file` and `key_file` enable mutual TLS. For a local port-forward to
the standard plaintext metadata gRPC service, use
`{"target":"localhost:8080","insecure":true}`. The source needs SELECT access
to the relevant SQL tables and read access to MLMD; import requires SQL inserts
and updates plus MLMD write access. The CLI never connects to Kubernetes.

Keep the selected completed run quiescent while exporting: do not retry it or
modify its artifact metadata. SQL snapshot reads and MLMD RPC reads are separate
transactions and do not provide a cross-service snapshot. Other runs can continue.

```sh
python tools/run-history/history.py \
  --db-config /private/source-db.json --mlmd-config /private/source-mlmd.json \
  export --source-id installation-a --run-id RUN_UUID --output run-history.json

python tools/run-history/history.py \
  --db-config /private/destination-db.json --mlmd-config /private/destination-mlmd.json \
  import --archive run-history.json --dry-run

python tools/run-history/history.py \
  --db-config /private/destination-db.json --mlmd-config /private/destination-mlmd.json \
  import --archive run-history.json
```

Use the same stable `--source-id` for every export from an installation, and a
different ID for independent installations. Archives are created with mode 0600,
are never overwritten, and can contain sensitive parameters and storage URIs.
The SHA-256 digest detects accidental edits; it does not authenticate untrusted
archives. Import only archives from a trusted administrator.

If both installations already contain a `Default` experiment in the same
namespace, explicitly map the historical run to the destination experiment:

```sh
python tools/run-history/history.py \
  --db-config /private/destination-db.json --mlmd-config /private/destination-mlmd.json \
  import --archive run-history.json --experiment-id DESTINATION_EXPERIMENT_UUID
```

The namespace must match. Existing experiments are not overwritten. Other name,
UUID, catalog content, type-schema, or MLMD provenance conflicts fail rather than
silently merge different resources. Resolve catalog conflicts explicitly before
retrying. A run already imported with the same source/digest is a no-op; a changed
archive for that run is rejected. Changing experiment mapping after import is
also rejected. Mutable source artifacts reused across exports must still match
their previously imported provenance.

## Transactions, failure recovery, and retained storage

Import validates the graph and reserves SQL rows and unique names in one SQL
transaction before writing MLMD. Ordinary SQL conflicts therefore leave MLMD
unchanged. Dry-run rolls back those SQL reservations and makes no MLMD writes.
Concurrent transfers or schema changes can still make a real import fail after
a successful dry-run. Serialize administrative imports into one destination.

MLMD and KFP SQL cannot commit together. After preflight, MLMD nodes and edges are
staged through idempotent, source-identified writes; only after successful staging
does the SQL transaction publish the imported run. If staging fails,
SQL is rolled back but **staged MLMD metadata may remain and may be visible through
direct MLMD queries**. It is not a runnable KFP run and is not added to execution
cache lookups. Retry the *same archive and source ID*: deterministic node names,
provenance digests, and existing-edge checks reuse the staged data. Do not delete
the archive or manually delete staged graph nodes, since ancestors can be shared
with successful imports. The archive and node provenance support operator audit;
there is no automatic MLMD garbage collector or distributed rollback.

A connection failure during SQL commit has an uncertain outcome: the server may
have committed before the response was lost. Retry the same archive; the committed
source/digest marker makes that retry a no-op. Do not change the run's UUID or
source ID to work around an ambiguous failure.

The archive does **not** contain artifact bytes, pipeline-spec objects referenced
by URI, or log files. Keep source buckets accessible to the destination under the
original paths, or arrange a separate verified storage migration first. Namespace
RBAC and artifact-storage credentials must permit the intended destination users
to read the imported history. Source pod logs are unavailable after deletion
unless already archived; destination archived-log configuration must be compatible.
Do not remove source object storage with the old cluster. Existing destination
retention policies apply to imported records, including old finish/archive times;
configure retention before import if that history must be kept.

## Verification

Dependency-free transaction and graph tests:

```sh
python3 -m unittest discover -s tools/run-history -p 'test_*.py' -v
```

MySQL tests run when `KFP_HISTORY_MYSQL_CONFIG` contains a JSON connection object
for an isolated test server. That user must be able to create/drop temporary
`history_test_*` databases. Real MLMD tests use the pinned MLMD package's embedded
SQLite store and generated protobufs. Set `KFP_HISTORY_REQUIRE_MLMD=1` to fail if
the package is missing. The release CI job requires both integrations. Runtime
guard coverage lives in the backend storage and resource test packages.
