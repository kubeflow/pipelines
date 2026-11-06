# MLMD migration CLI

This is the standalone entry point for PR 2 of the [MLMD migration plan](../../../../../proposals/12147-mlmd-removal/database-migration-implementation-plan.md). API-server startup migration and deployment packaging are PR 3 work.

Build from the repository root:

```sh
go build -o mlmd-migrate ./backend/src/apiserver/migration/mlmd/cmd/mlmd-migrate
```

PR 2 qualifies this CLI on disposable databases. Production use awaits the PR 3
maintenance and writer fence. That upgrade must block submissions, retries, and
recurring triggers; wait for zero active KFP runs and Argo Workflows with final
reports stored; then terminate old API replicas and their database connections
before Phase 2/3. Back up the KFP MySQL database and allow space for the
temporary `tasks` copy and new artifact tables. The CLI requires a stable
source and does not enforce those deployment gates itself. Legacy PostgreSQL
upgrades are not qualified. Supply the KFP MySQL driver DSN and MLMD gRPC
address:

```sh
./mlmd-migrate \
  --mysql-dsn='user:password@tcp(mysql-host:3306)/kfp?parseTime=true' \
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

Use `--mlmd-tls` for a TLS endpoint and repeat `--mlmd-header=key:value` for required gRPC metadata. The CLI prints a JSON report with `status: complete`, row counts, warnings, and skipped source entities; it exits nonzero on an incomplete migration. Keep the source MLMD store and database backup until the completed installation has been validated. Restore the pre-upgrade database backup with the old server image to roll back. A failed or interrupted CLI run can be repeated while the source remains stable.

The MySQL regression suite uses `KFP_MLMD_MYSQL_TEST_DSN` pointing to a
disposable loopback server. It creates and drops uniquely named test databases,
replays the pinned 2.17 and 2.18 schemas, runs the built CLI against a fixture
gRPC server, checks interruption and repeated runs, and compares manual
artifact-table SQL with the native GORM schema:

```sh
KFP_MLMD_MYSQL_TEST_DSN='user:password@tcp(127.0.0.1:3306)/mysql?parseTime=true' \
  go test ./backend/src/apiserver/migration/mlmd -run 'TestMySQL|TestBuiltCLI' -count=1
```
