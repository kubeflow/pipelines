# Upgrade a KFP 2.18.0 installation from MLMD to native metadata

## Production release hold

Keep an MLMD-backed production installation on KFP 2.18 until the database
cutover meets the [nonblocking database qualification](../../../../../proposals/12147-mlmd-removal/pr-3-review-fix-design.md).
The shipped API and CLI refuse migration writes. The shipped CLI still supports
read-only validation, drain checks, and capacity estimation. Leave production
routing, API replicas, and recurring-run controllers in service.

## Disposable upgrade procedure

The commands below are for the pinned test workflow and disposable databases.
The workflow builds the API and CLI with the `mlmd_migration_test` build tag
and verifies refusal with the normal release binaries before testing cutover.
Use a fresh test environment for each run.

Use a maintenance window for this MySQL upgrade. Legacy PostgreSQL installations
must stay on the old release. Fresh and already-native PostgreSQL installations
start without MLMD.

The release API image contains the read-only production CLI at `/bin/mlmd-migrate`.
For a preflight workstation with access to the source, database, and Kubernetes API:

```sh
container=$(docker create "$NEW_API_IMAGE")
docker cp "$container:/bin/mlmd-migrate" ./mlmd-migrate-read-only
docker rm "$container"
```

The standalone CLI and API startup use the same [migration engine](../../../../../proposals/12147-mlmd-removal/database-migration-design.md). The API-server image also contains `/bin/mlmd-migrate`.

Build from the repository root:

```sh
go build -tags=mlmd_migration_test -o mlmd-migrate ./backend/src/apiserver/migration/mlmd/cmd/mlmd-migrate
```

Back up the KFP MySQL database and MLMD store before cutover. Block user API
traffic at the installation's ingress while preserving runtime reporting, and
scale the recurring-run controller to zero. Wait for all KFP runs and Argo
Workflows to finish and for terminal reports to be stored. Then scale the old
`ml-pipeline` API Deployment to zero and wait until its pods and database
connections have closed. Keep the ingress gate in place while applying the new
API image. Its startup gate refuses migration if an old API pod, active run, or
unreported Workflow remains. Readiness stays false while migration runs;
liveness remains available on port 8889. Do not route normal traffic back to
the API until startup completes and the completion marker exists. Resume the
recurring-run controller only after that point.

The CLI requires the old Deployment scaled to zero with no remaining pods. It
checks the same run and Workflow drain conditions and requires Kubernetes
credentials with access to the API Deployment and cluster-wide KFP Workflows.
Use `--namespace` and `--deployment` when the installation does not use
`kubeflow/ml-pipeline`. The advisory lock coordinates CLI and API migration;
it does not fence old writers. Legacy PostgreSQL upgrades are not qualified.

`cutover.sh` performs the drain and writer-fence sequence after the operator
has installed a public ingress gate that returns HTTP 503 with a maintenance
message while keeping internal runtime reporting reachable. The optional
`manifests/kustomize/components/mlmd-maintenance/maintenance.yaml` supplies
that HTTP response: apply it separately before upgrading the old API, set
its `maintenance` container to the new API image, and scale it to one replica.
Route public HTTP API traffic to `ml-pipeline-maintenance:8888` and block
public gRPC access while leaving the old internal `ml-pipeline` Service
reachable by the persistence agent, driver, and launcher. The maintenance
Deployment is deliberately separate from the normal Kustomize base so the
upgrade does not scale it down during cutover. Prepare it from the repository
root, replacing `NEW_API_IMAGE` with the image built from this revision:

```sh
kubectl -n kubeflow apply -f manifests/kustomize/components/mlmd-maintenance/maintenance.yaml
kubectl -n kubeflow set image deployment/ml-pipeline-maintenance maintenance="$NEW_API_IMAGE"
kubectl -n kubeflow scale deployment/ml-pipeline-maintenance --replicas=1
kubectl -n kubeflow rollout status deployment/ml-pipeline-maintenance
```

After routing public traffic, set
`KFP_MLMD_MAINTENANCE_URL`, `KFP_MLMD_MYSQL_DSN`, and
`KFP_MLMD_ADDRESS`, then run the script from an operator workstation
with `kubectl`, `curl`, and `mlmd-migrate` on `PATH`. Pass optional
`--mlmd-tls` and `--mlmd-header` arguments to the script for preflight.
It pauses recurring triggers, waits for terminal reports, runs `validate`,
scales the old API Deployment to zero, waits for its pods to disappear,
and rechecks the fence. Prevent deployment controllers and autoscalers from
restoring old API replicas or scheduled-run triggers during migration.
It does not reopen traffic or resume recurring triggers; do that after the new
API completes migration. If the script refuses to proceed, leave the public
gate in place and resume the old Deployment only after investigating.
Restore public routing and scale the maintenance Deployment to zero only after
the new API readiness and migration completion checks pass.

Before starting, take a consistent backup of the entire KFP MySQL database
and retain the paired MLMD database and old API image. Allow temporary disk
space for a second `tasks` table and new artifact tables. If draining times
out, leave the old API in place and investigate the active run or Workflow.
The task-table swap and cleanup use a short MySQL metadata-lock timeout.
A transaction on a table related by a foreign key, such as `run_details`,
can delay the swap and a queued swap can briefly delay writes to that table.
If the lock timeout expires, close the blocking transaction before retrying.
If migration fails after the fence, keep public maintenance routing active;
fix the cause and rerun the idempotent CLI or restart the new API. To roll
back, stop the new API, restore the pre-upgrade KFP database backup, deploy
the old API image, verify its readiness and runtime reports, then restore
public routing and the recurring-run controller. Do not reopen traffic on a
partially migrated database.

Measure the bytes available on the destination MySQL data volume, for example
with `df -B1` on that volume or the database provider's free-storage metric.
Measure the allocated bytes of the source MLMD database, including indexes. For
an MLMD MySQL database, use the sum of `DATA_LENGTH + INDEX_LENGTH` in
`information_schema.TABLES` restricted to the MLMD schema. The CLI estimates
required space as twice the MLMD database size plus the estimated size of the
legacy KFP tasks table and its indexes. Refresh the measurements before retrying.

Immediately before scaling down the old API, run the read-only preflight.
It checks both endpoints and the target schema, reports active KFP runs and
owned Argo Workflows, and exits nonzero if it cannot establish the prerequisites.
Keep submissions and retries blocked after it succeeds. The checks reflect a
point-in-time view of the installation.

```sh
./mlmd-migrate validate --mlmd-address='metadata-grpc-service:8080'
```

For historical single-user databases whose run and experiment namespaces are
both unset, explicitly declare a single-user source with
`--source-single-user` (CLI) or `MLMD_MIGRATION_SOURCE_SINGLE_USER=true` (API).
The fallback requires a verified server-persisted Workflow identity. The
checker reads the API Deployment and any ConfigMaps used to supply its
`MULTIUSER` setting. Give its credentials `get` access to those referenced
ConfigMaps when the installation uses them. An API Deployment configured for
multiple users cannot use the fallback.

Both MLMD clients default to a 100 MiB receive limit. Set
`METADATA_GRPC_MESSAGE_SIZE` if the released installation used a different
limit. The read-only preflight checks connectivity and drain status. A full
history read occurs during migration.

Use the same TLS and header flags you plan to use for migration. Check capacity
separately using the measured byte counts:

```sh
./mlmd-migrate --check-capacity \
  --mysql-dsn='user:password@tcp(mysql-host:3306)/kfp?parseTime=true' \
  --available-disk-bytes="$KFP_MYSQL_FREE_BYTES" \
  --mlmd-database-bytes="$MLMD_DATABASE_BYTES"
```

After fencing the old Deployment, run the migration. It repeats the capacity check before writing:

```sh
./mlmd-migrate \
  --mysql-dsn='user:password@tcp(mysql-host:3306)/kfp?parseTime=true' \
  --available-disk-bytes="$KFP_MYSQL_FREE_BYTES" \
  --mlmd-database-bytes="$MLMD_DATABASE_BYTES" \
  --mlmd-address='metadata-grpc-service:8080'
```

The CLI preserves eligible MLMD fingerprints in task status metadata and puts
only verified native keys in `tasks.Fingerprint`. It fixes the 90-day cutoff
in `mlmd_migration_state` on the first migration attempt. A producer whose
inputs cannot reproduce its source fingerprint gets an error log and a
`cache_key` entry in the report's skipped list; that skip does not block
migration completion. Custom keys are also skipped because the native lookup
does not retain their legacy pipeline scope. Cached reuse rows are never native
lookup targets.

The report counts verified cache candidates, newly written keys, and candidates
whose existing keys were preserved. Copied and skipped counts group source
entities by kind. The separate table totals include each row from a metric split.

Use `--mlmd-tls` for a TLS endpoint and repeat `--mlmd-header=key:value` for required gRPC metadata. The CLI prints a JSON report with `status: complete`, row counts, warnings, and skipped source entities; it exits nonzero on an incomplete migration. Keep the source MLMD store and database backup until the completed installation has been validated. Restore the pre-upgrade database backup with the old server image to roll back. A failed or interrupted CLI run can be repeated while the source remains stable.

For API-only migration, set `METADATA_GRPC_SERVICE_HOST`,
`METADATA_GRPC_SERVICE_PORT`, `MLMD_MIGRATION_AVAILABLE_DISK_BYTES`, and
`MLMD_MIGRATION_DATABASE_BYTES` on the upgrade boot. Use the same measured
byte values as the CLI disk check. Optional
`MLMD_MIGRATION_TLS=true` enables TLS, and `MLMD_MIGRATION_HEADERS_JSON`
contains a JSON object of gRPC request headers. The Deployment must expose
`POD_NAME` through the downward API. A completed installation starts without
MLMD settings. If validation finished and startup was interrupted during
cleanup, the next attempt verifies the writer fence and finishes cleanup
without MLMD. Reuse the original MLMD endpoint for every retry that has not
finished validation. Review skip diagnostics before removing the source. Startup
also preserves a released 2.18 `transfer_identities` installation UUID while
reconciling that table for the current model. A failed
upgrade stays out of service; restore the pre-upgrade database backup and old
server image before reopening traffic.

The MySQL regression suite uses `KFP_MLMD_MYSQL_TEST_DSN` pointing to a
disposable loopback server. It creates and drops uniquely named test databases,
replays the pinned 2.17 and 2.18 schemas, runs the built CLI against a fixture
gRPC server, checks interruption and repeated runs, and compares manual
artifact-table SQL with the native GORM schema:

```sh
KFP_MLMD_MYSQL_TEST_DSN='user:password@tcp(127.0.0.1:3306)/mysql?parseTime=true' \
  go test ./backend/src/apiserver/migration/mlmd -run 'TestMySQL|TestBuiltCLI' -count=1
```

The pinned upgrade workflow runs CLI-first and startup-only upgrades with
both databases restored from the same release dump. It checks the serving APIs,
restarts without MLMD, then restores the KFP backup and verifies the old API.
The backend presubmit retains the smaller SQL, fixture-RPC, and startup tests.

## KFP 2.18.0 migration fixtures

### Validate the fixture

| Check | What it verifies | Requirements |
|---|---|---|
| [Release capture](release_fixture_test.go) | Legacy API task IDs, saved reports, and captured Argo Workflows agree. | Saved fixture |
| [Pinned upgrade](pinned_upgrade_test.go) | Clean migration, interrupted metric copy, cache backfill, stable manifests, and run, task, artifact, and lineage reads through the native API handlers. | Disposable MySQL 8.0 in CI |
| [Migration mappings](migration_text_fixture_test.go) | The generator reproduces the saved contract, and the expected rows satisfy the mapping and edge-case assertions. | Saved fixture and Python dependencies |
| [API projection](../../server/migration_contract_test.go) | Native stores and API conversion expose the expected task fields and artifact lineage. | Temporary SQLite database |
| [Startup reproducer](../../client_manager/mlmd_released_startup_test.go) | The old schema triggers the expected startup failures and all source task rows survive. | Disposable MySQL 8.0 in CI |
| [Workflow drain checker](check_workflow_drain_test.py) | The operator command identifies active Workflows that belong to the installation. | Python standard library |

From the repo root, install the fixture's Python dependencies and run the
Go suite in an active virtual environment. Backend CI uses the same Go tests.

```bash
python3 -m pip install -r backend/src/apiserver/migration/mlmd/fixturetools/requirements.txt
go test ./backend/src/apiserver/migration/mlmd ./backend/src/apiserver/server \
  ./backend/src/apiserver/client_manager \
  -run 'Test(Released|MigrationText|MLMDReleasedStartup|MySQLPinned218UpgradeContract)' -count=1
```

To enable the startup reproducer locally, set `KFP_MLMD_MYSQL_TEST_DSN` to a
disposable MySQL 8.0 server over loopback TCP without TLS. For example, use
`root@tcp(127.0.0.1:3306)/?parseTime=true` for a local instance with an empty
test password. The reproducer covers the original snapshot and a variant with
a 240-character task name. The [migration TLS setting](../../../../../proposals/12147-mlmd-removal/database-migration-design.md#mlmd-connection-configuration)
applies to the migrator's MLMD gRPC endpoint.

To run the standalone drain checker tests, use
`python3 backend/src/apiserver/migration/mlmd/check_workflow_drain_test.py`.

### Historical fields and event identity

The migrator preserves typed parameter values. It fills an I/O type and
producer only when the saved binding and captured values establish them.
It omits cross-run parent relationships and keeps independently owned tasks.
Unmapped details remain in the legacy source or pre-upgrade backup.

MLMD gRPC events omit the numeric SQL event ID. A migrated link's UUID v5
is derived from its native artifact ID, task ID, I/O type, iteration, and
artifact key. Equivalent links have the same identity. The report identifies
unsupported or duplicate wire events by their execution, artifact, type,
timestamp, and path. Source SQL IDs remain available in the capture.

### Scale qualification

Backend presubmit runs the pinned replay with exact data and resume assertions.
Keep that deterministic correctness check in required CI. Run large-history
qualification as an opt-in or scheduled benchmark on fixed-capacity runners.
Shared-runner wall-clock thresholds would vary with CPU, memory, and storage
contention and would provide a poor estimate for a maintenance window.

For release qualification, record the database engine version, runner capacity,
source and target row counts and byte sizes, peak process memory and temporary
database disk use, elapsed time by phase, and restart time after an injected
interruption. Include a clean CLI-first run. A default presubmit pass does not
establish this capacity or qualify production DDL.

### Refresh the capture

The [release fixture README](testdata/release_218/README.md#refresh-the-capture)
describes how to collect a new snapshot from a disposable 2.18.0 installation.
The scripts and submitted pipeline packages live in [`fixturetools/`](fixturetools/).
