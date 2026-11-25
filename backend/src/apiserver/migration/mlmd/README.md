# KFP 2.18.0 migration tests

PR 1 prepares and verifies the fixtures for the
[KFP 2.18 to 3.0 upgrade](../../../../../proposals/12147-mlmd-removal/database-migration-implementation-plan.md).
PRs 2 and 3 will use them to verify the migration engine and both upgrade paths.

## Validate the fixture

| Check | What it verifies | Requirements |
|---|---|---|
| [Release capture](release_fixture_test.go) | Legacy API task IDs, saved reports, and captured Argo Workflows agree. | Saved fixture |
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
  -run 'Test(Released|MigrationText|MLMDReleasedStartup)' -count=1
```

To enable the startup reproducer locally, set `KFP_MLMD_MYSQL_TEST_DSN` to a
disposable MySQL 8.0 server over loopback TCP without TLS. For example, use
`root@tcp(127.0.0.1:3306)/?parseTime=true` for a local instance with an empty
test password. The reproducer covers the original snapshot and a variant with
a 240-character task name. The [migration TLS setting](../../../../../proposals/12147-mlmd-removal/database-migration-design.md#mlmd-connection-configuration)
applies to the migrator's MLMD gRPC endpoint.

To run the standalone drain checker tests, use
`python3 backend/src/apiserver/migration/mlmd/check_workflow_drain_test.py`.

## Refresh the capture

The [release fixture README](testdata/release_218/README.md#refresh-the-capture)
describes how to collect a new snapshot from a disposable 2.18.0 installation.
The scripts and submitted pipeline packages live in [`fixturetools/`](fixturetools/).
