# MLMD migration CLI

The standalone migrator implements PR 2 of the [KFP 2.18 to 3.0 upgrade](../../../../../proposals/12147-mlmd-removal/database-migration-implementation-plan.md). API-server startup migration and deployment packaging are PR 3 work.

Build from the repository root:

```sh
go build -o mlmd-migrate ./backend/src/apiserver/migration/mlmd/cmd/mlmd-migrate
```

PR 2 qualifies this CLI on disposable databases. Production use awaits the PR 3
maintenance and writer fence. That upgrade must block submissions, retries, and
recurring triggers; wait for zero active KFP runs and Argo Workflows with final
reports stored; then terminate old API replicas and their database connections
before Phase 2/3. Back up the KFP MySQL database. The CLI requires a stable
source and does not enforce those deployment gates itself. Legacy PostgreSQL
upgrades are not qualified.

Measure the bytes available on the destination MySQL data volume, for example
with `df -B1` on that volume or the database provider's free-storage metric.
Measure the allocated bytes of the source MLMD database, including indexes. For
an MLMD MySQL database, use the sum of `DATA_LENGTH + INDEX_LENGTH` in
`information_schema.TABLES` restricted to the MLMD schema. The CLI estimates
required space as twice the MLMD database size plus the estimated size of the
legacy KFP tasks table and its indexes. Refresh the measurements before retrying.

Run the read-only check using those measurements:

```sh
./mlmd-migrate --check-capacity \
  --mysql-dsn='user:password@tcp(mysql-host:3306)/kfp?parseTime=true' \
  --available-disk-bytes="$KFP_MYSQL_FREE_BYTES" \
  --mlmd-database-bytes="$MLMD_DATABASE_BYTES"
```

The migration repeats the capacity check before writing:

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

The migrator drops its internal `tasks_pre_migration` table after successful
cutover. Take and retain the operator backup before running the CLI.

The MySQL regression suite uses `KFP_MLMD_MYSQL_TEST_DSN` pointing to a
disposable loopback server and requires MySQL's `mysqldump` on `PATH` for the
restore check. It creates and drops uniquely named test databases, compares
persisted rows with the direct-writer golden, checks task-copy gap repair,
and restores a dump taken immediately before migration. It also runs the
built CLI against a fixture gRPC server and compares manual artifact-table
SQL with the native GORM schema.

```sh
KFP_MLMD_MYSQL_TEST_DSN='user:password@tcp(127.0.0.1:3306)/mysql?parseTime=true' \
  go test ./backend/src/apiserver/migration/mlmd -run 'TestMySQL|TestBuiltCLI' -count=1
```

The files under [`testdata/engine/`](testdata/engine/) pin the direct-writer
and earlier 2.18 pipeline captures used by these engine tests.

The pinned 2.18 engine test restores KFP SQL rows and serves fixture MLMD
responses from the same release capture. The production qualification plan
adds replay through the pinned MLMD server and old API server.

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
