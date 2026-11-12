# MLMD migration CLI

The standalone CLI and API startup use the same [migration engine](../../../../../proposals/12147-mlmd-removal/database-migration-design.md). The API-server image also contains `/bin/mlmd-migrate`.

Build from the repository root:

```sh
go build -o mlmd-migrate ./backend/src/apiserver/migration/mlmd/cmd/mlmd-migrate
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
`KFP_MLMD_MAINTENANCE_URL` and `KFP_MLMD_MYSQL_DSN`, then run the script from an
operator workstation with `kubectl`, `curl`, and `mlmd-migrate` on `PATH`.
It pauses recurring triggers, waits for terminal reports, scales the old API
Deployment to zero, waits for its pods to disappear, and rechecks the fence.
It does not reopen traffic or resume recurring triggers; do that after the new
API completes migration. If the script refuses to proceed, leave the public
gate in place and resume the old Deployment only after investigating.
Restore public routing and scale the maintenance Deployment to zero only after
the new API readiness and migration completion checks pass.

Before starting, take a consistent backup of the entire KFP MySQL database
and retain the paired MLMD database and old API image. Allow temporary disk
space for a second `tasks` table and new artifact tables. If draining times
out, leave the old API in place and investigate the active run or Workflow.
If migration fails after the fence, keep public maintenance routing active;
fix the cause and rerun the idempotent CLI or restart the new API. To roll
back, stop the new API, restore the pre-upgrade KFP database backup, deploy
the old API image, verify its readiness and runtime reports, then restore
public routing and the recurring-run controller. Do not reopen traffic on a
partially migrated database.

Supply the KFP MySQL driver DSN and MLMD gRPC address:

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

For API-only migration, set `METADATA_GRPC_SERVICE_HOST` and
`METADATA_GRPC_SERVICE_PORT` on the upgrade boot. Optional
`MLMD_MIGRATION_TLS=true` enables TLS, and `MLMD_MIGRATION_HEADERS_JSON`
contains a JSON object of gRPC request headers. The Deployment must expose
`POD_NAME` through the downward API. A completed installation starts without
MLMD settings. If the CLI already completed migration, startup reads its
durable completion marker and proceeds without connecting to MLMD. Startup
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

`TestMLMD218ReleasedStartupMigration` exercises both the API-only and
CLI-ahead paths from the pinned 2.18 SQL, MLMD graph, and terminal Workflows.
It checks the completion marker, preserved installation UUID, and an API
restart without MLMD settings in the backend CI MySQL lane.
