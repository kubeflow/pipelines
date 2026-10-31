# Pinned MLMD migration fixtures

These fixtures are evidence for [PR 1 of the migration plan](../../../../../../proposals/12147-mlmd-removal/database-migration-implementation-plan.md), not a migrator or an enabled upgrade.

- `legacy_source.json` was written by the unmodified KFP `2.17.3` metadata client at commit `a0733cb51b0484a6050cafee989ffe3312f808ad` against a disposable MLMD `1.14.0` server. The generator calls the writer directly; it does not run a pipeline, driver, or launcher. Execution 12 belongs to an unowned run; retained execution 13 proves that eligible IDs need not be consecutive.
- `runtime_source.json` was produced by the pinned `driver.RootDAG`, `driver.DAG`, and `component.ImportLauncher` methods against a separate disposable MLMD server. The driver used a fake Kubernetes ConfigMap client and the launcher did not submit an Argo Workflow. It establishes runtime-written root naming, skipped DAG properties, importer output, and event shape; it is still short of the pipeline-executed fixture required for PR 1.
- `expected_native.json` is an independently authored projection. It tests deterministic identities, all metric links, a metadata-only empty metrics artifact, native loop attributes, and an inactive legacy cache fingerprint stored as provenance. The source `CANCELED` record has no persisted reason, so its native state remains `FAILED` with its raw state retained. This is a partial projection, not a full field-merge oracle.
- `merge_contract.json` is a separate synthetic case that pairs the pinned consumer execution with a populated legacy task row. It specifies preservation of pod and lifecycle timestamps, inactive cache provenance, and the difference between a workflow-proven skipped branch and an ambiguous `CANCELED` record. It is not a row produced by the writer scenario; conflicting source values still need a separate fixture.
- `legacy_schema_mysql.json` and `legacy_schema_pgx.json` capture the old initializer's statements, populated seed data, and catalogs on disposable MySQL 8.0 and PostgreSQL 16. Each has an ordinary task row and a separate row with 240-character name/pod values and 1,024-character large-text values. They are startup reproducer inputs, not the source of the native projection.
- `legacy_partial_mysql.json` and `legacy_partial_pgx.json` capture the actual catalogs after the current `InitDBClient` fails on those populated legacy databases. MySQL has already added `pods` and timestamp columns; both engines have narrowed `Namespace` before failing. They are recognized *source shapes for future classification*, not evidence that the current startup can resume them safely.
- `legacy_workflow.yaml` is copied from the pinned commit. Its in-memory retry proves the old workflow retains its MLMD-dependent runtime; it was not submitted.
- `identity_vectors.json` pins UUID v5 names and results. `manifest.json` pins file hashes and source provenance. Review source and expected-output changes separately.

The initial support target is KFP `2.17.3` with MLMD `1.14.0` into the current native schema on MySQL 8.0 or PostgreSQL 16. The two captured partial catalogs are inputs for the later classifier; other partial or older schemas require separate qualification. PR 1 still requires an actual pinned pipeline execution, matching KFP rows, and a disposable writer-handoff test before the projection can be an acceptance oracle.

Run the service-free checks:

```bash
go test ./backend/src/apiserver/migration/mlmd ./backend/src/apiserver/server \
  ./backend/src/apiserver/client_manager \
  -run 'Test(Fixture|Legacy|ExpectedNative|MigrationContract|MigrationIdentity|MLMDLegacy)' -count=1
```

The client-manager startup reproducer skips its MySQL/PostgreSQL cases unless `KFP_MLMD_MYSQL_TEST_DSN` and `KFP_MLMD_POSTGRES_TEST_DSN` point to disposable local databases. Its baseline assertion is that current startup fails on populated legacy schemas; it does not prove an upgrade.

To regenerate `legacy_source.json`, export the pinned source into an isolated local directory and run the generator against a disposable loopback MLMD service. The generator checks pinned source hashes and refuses a nonempty store:

```bash
fixture_repo="$PWD"
fixture_source=$(mktemp -d /tmp/kfp-mlmd-source.XXXXXX)
fixture_config=$(mktemp -d /tmp/kfp-mlmd-config.XXXXXX)
git archive a0733cb51b0484a6050cafee989ffe3312f808ad \
  go.mod go.sum api backend/src backend/api backend/test kubernetes_platform \
  third_party/ml-metadata manifests | tar -xf - -C "$fixture_source"
cat > "$fixture_config/server.pbtxt" <<'EOF'
connection_config {
  sqlite {
    filename_uri: "/tmp/fixture.db"
    connection_mode: READWRITE_OPENCREATE
  }
}
EOF
podman run --rm -d --name kfp-mlmd-contract-fixture \
  -p 127.0.0.1:18080:8080 \
  -e METADATA_STORE_SERVER_CONFIG_FILE=/fixture/server.pbtxt \
  -v "$fixture_config/server.pbtxt:/fixture/server.pbtxt:ro,Z" \
  gcr.io/tfx-oss-public/ml_metadata_store_server@sha256:51404ef4419c1426da80f3e14a83dcb55f65bb5a971371d36d14cec1e50af830
(
  cd "$fixture_source"
  go run "$fixture_repo/backend/src/apiserver/migration/mlmd/testdata/generate_legacy.go"
) > "$fixture_config/legacy_source.json"
podman stop kfp-mlmd-contract-fixture
```

Compare the generated output to the checked-in file before replacing it. Regeneration normalizes timestamps and record order; it does not replace execution names or properties. To regenerate both full and interrupted database catalogs with fresh disposable MySQL/PostgreSQL containers, run:

```bash
catalog_output=$(mktemp -d /tmp/kfp-mlmd-catalogs.XXXXXX)
bash "$fixture_repo/backend/src/apiserver/migration/mlmd/testdata/regenerate_catalogs.sh" \
  "$fixture_source" "$catalog_output"
diff -u "$fixture_repo/backend/src/apiserver/migration/mlmd/testdata/legacy_schema_mysql.json" \
  "$catalog_output/legacy_schema_mysql.json"
```

Compare the other three generated catalogs the same way. The script binds only local database ports, invokes the pinned old initializer, runs the current startup reproducer, and stops its own containers. No command in this README accesses a Kubernetes cluster.

To reproduce the runtime-method fixture, use the same pinned checkout and a **new** disposable MLMD container on loopback port `18081`. Run the pinned driver and launcher methods in order, then export the source records:

```bash
cp "$fixture_repo/backend/src/apiserver/migration/mlmd/testdata/runtime_fixture_driver_test.go" \
  "$fixture_source/backend/src/v2/driver/runtime_fixture_test.go"
cp "$fixture_repo/backend/src/apiserver/migration/mlmd/testdata/runtime_fixture_importer_test.go" \
  "$fixture_source/backend/src/v2/component/runtime_fixture_test.go"
podman run --rm -d --name kfp-mlmd-runtime-fixture \
  -p 127.0.0.1:18081:8080 \
  -e METADATA_STORE_SERVER_CONFIG_FILE=/fixture/server.pbtxt \
  -v "$fixture_config/server.pbtxt:/fixture/server.pbtxt:ro,Z" \
  gcr.io/tfx-oss-public/ml_metadata_store_server@sha256:51404ef4419c1426da80f3e14a83dcb55f65bb5a971371d36d14cec1e50af830
(
  cd "$fixture_source"
  KFP_RUNTIME_FIXTURE=1 go test ./backend/src/v2/driver -run '^TestPinnedRuntimeFixture$' -count=1 -v
  KFP_RUNTIME_FIXTURE=1 go test ./backend/src/v2/component -run '^TestPinnedRuntimeImporterFixture$' -count=1 -v
  go run "$fixture_repo/backend/src/apiserver/migration/mlmd/testdata/export_runtime.go" \
    > "$fixture_config/runtime_source.json"
)
podman stop kfp-mlmd-runtime-fixture
```

The exporter refuses stores with other execution or artifact counts and normalizes only timestamps and ordering. Compare its output with `runtime_source.json` before replacing it. This harness invokes real pinned runtime methods locally; it does not prove execution through Argo or the API server. The remaining pipeline-executed fixture must include the submitted Workflow, matching KFP rows, runtime image digests and MLMD export. Stop disposable containers even if a generation step fails.
