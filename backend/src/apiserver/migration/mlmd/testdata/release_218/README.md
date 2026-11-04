# KFP 2.18.0 pipeline-executed source

This snapshot comes from the released KFP `2.18.0` source at commit
`555bfa60a70da88c594960bdacfd78fa7c061043`, deployed with its
`manifests/kustomize/cluster-scoped-resources` and
`manifests/kustomize/env/platform-agnostic` manifests on a disposable Kind
v1.32.2 cluster. MySQL was 8.4 and MLMD was 1.14.0. `image_provenance.json`
records the image digests observed on the API, MLMD, Argo, driver, launcher,
and component pods. The checked-in `.py` and `.yaml` files are copied from
that release; the KFP Python client `2.15.2` submitted the compiled packages
through the released API. No Workflow was submitted directly to Argo.

| Run | ID | Workflow | Result |
| --- | --- | --- | --- |
| Parameters and loop | `2ec7ad3b-d821-4b1d-8355-190393a06ddc` | `math-pipeline-k948h` | Succeeded; five container executions |
| Same package, cache enabled | `5dedb489-f0c5-496e-b2e3-eaf46a5fcc74` | `math-pipeline-9z8t8` | Succeeded; five cached container executions refer to the first run |
| Metrics | `be2ae7cc-da75-4cbb-bdb6-25145b26ae52` | `output-metrics-59hc2` | Succeeded; a metrics artifact has an output event |

`source.sql` is a single-transaction dump of the `mlpipeline` and `metadb`
databases after those runs reached terminal state. `source_graph.json` is a
readable projection of all KFP run/task rows, including each run's saved
pipeline spec and submission fields, and their MLMD types, contexts,
executions, properties, artifacts, events, and event paths. It includes KFP
task rows without an MLMD execution ID, including the cache run's `SKIPPED`
rows. The SQL also contains an earlier failed
exploratory run (`1dd61cdf-fd5f-471c-81a7-a14532a31fdd`) of the release's
`conditional_producer_and_consumers.yaml`; it failed during conditional output
resolution and is excluded from the successful-run assertions. The three
Workflow files retain the submitted spec, run label, and terminal status.
They omit Kubernetes managed fields and detailed Argo node status. The SQL
snapshot's `run_details.WorkflowRuntimeManifest` retains the node status.
The cache run's submitted Workflow contains an `executor` condition and its
KFP rows record `SKIPPED` without MLMD execution IDs. The failed conditional
run in the same SQL snapshot has MLMD execution 7 in
`CANCELED`, a KFP `SKIPPED` row without an execution ID, and an Argo
`condition-3` node marked `Skipped` because its guard evaluated false.
`TestMLMDReleasedStartup` checks all three against the same run. The original
design's `CANCELED` → `FAILED` rule applies to the unjoined MLMD execution;
the KFP row cannot be joined by execution ID.

`expected_native.json` is a hand-authored, deliberately partial native
projection of ten executions, a metrics artifact, an iteration-produced
artifact, and their output events from
the same three run IDs. `TestReleasedNativeProjectionMatchesSource` checks its
source IDs, joins, parent links, loop attributes, cache producer, timestamps,
metric, and event path against `source_graph.json`.
`TestReleasedMigrationProjectionAPI` loads those native rows into the current
store and checks the API conversion. `TestMLMDReleasedStartup` replays the same
KFP and MLMD SQL in disposable MySQL databases, joins task row
`bf1ec48e-f30d-4c09-92d6-0765639d2a56`
to execution 19, checks the independently recorded native fields, and verifies
that current API startup fails without changing the KFP row. CI runs it with
`KFP_MLMD_MYSQL_TEST_DSN`. Neither test runs a migration. In this release
snapshot, some MLMD DAG executions remain `RUNNING` after their API
runs succeed; the KFP task rows joined to container executions have empty pod
names and `RUNTIME_STATE_UNSPECIFIED`. The projection takes the MLMD execution
state and pod when the KFP fields are empty or unspecified. It takes the KFP
row's run identity and lifecycle timestamps, including creation time. The
iteration-produced artifact's link retains the source parent DAG's iteration
index, including a present zero. The projection preserves eligible source
fingerprints in status metadata. The PR 2 engine separately verifies and
backfills native producer keys within its fixed 90-day window.
`TestReleasedWriterRoutesAndDrainTimestamps` runs in backend PR CI and checks
that the submitted Workflows route legacy driver work to MLMD and the KFP API,
and that recorded MLMD/KFP writes precede each Workflow's terminal timestamp.
It is writer-route evidence, not a finalization-barrier test: the maintenance
gate and old-writer coordination are not implemented in PR 1.

To reproduce the capture, deploy the two release manifest directories to a
**new disposable cluster**, submit `parameters_simple.yaml` twice with caching
enabled and `output_metrics.yaml` once through `kfp.Client` against that
cluster's API. From this directory, with the API port forwarded to
`127.0.0.1:18888`, the submissions were:

```python
from kfp import Client

client = Client(host='http://127.0.0.1:18888')
for name in ('mlmd-migration-218-parameters',
             'mlmd-migration-218-parameters-cached'):
    run = client.create_run_from_pipeline_package(
        'parameters_simple.yaml', run_name=name,
        experiment_name='mlmd-migration-218-fixture',
        enable_caching=True,
    )
    client.wait_for_run_completion(run.run_id, timeout=900)
run = client.create_run_from_pipeline_package(
    'output_metrics.yaml', run_name='mlmd-migration-218-metrics',
    experiment_name='mlmd-migration-218-fixture', enable_caching=True,
)
client.wait_for_run_completion(run.run_id, timeout=900)
```

Wait for the three Workflows and API runs to succeed. Then capture the source
with explicit fixture-cluster selection:

```bash
fixture_kubeconfig=/path/to/fixture.kubeconfig
fixture_context=kind-your-disposable-fixture
fixture_dir=backend/src/apiserver/migration/mlmd/testdata/release_218
python3 "$fixture_dir/capture_graph.py" \
  --kubeconfig "$fixture_kubeconfig" --context "$fixture_context" \
  --output "$fixture_dir/source_graph.json"
kubectl --kubeconfig "$fixture_kubeconfig" --context "$fixture_context" \
  exec -n kubeflow deployment/mysql -- sh -c \
  'MYSQL_PWD="$MYSQL_ROOT_PASSWORD" mysqldump -uroot --databases mlpipeline metadb --single-transaction --set-gtid-purged=OFF --no-tablespaces --skip-comments' \
  > "$fixture_dir/source.sql"
```

Capture each Workflow with `kubectl get workflow NAME -n kubeflow -o json`,
retaining `apiVersion`, `kind`, `metadata` name/namespace/labels/annotations,
`spec`, and the terminal `status` phase/timestamps/progress/nodes. The node
status must link the conditional `executor` to the same run before using this
fixture to distinguish `NotTriggered` from cancellation. Capture image
digests from pod `status.containerStatuses` and `status.initContainerStatuses`.
New runs will have different UUIDs and timestamps; review the source and
expectations before replacing this pinned snapshot. The CI coherence check is:

```bash
go test ./backend/src/apiserver/migration/mlmd ./backend/src/apiserver/server \
  -run 'Test(ReleasedPipelineFixtureJoins|ReleasedNativeProjectionMatchesSource|ReleasedMigrationProjectionAPI|ReleasedWriterRoutesAndDrainTimestamps)$' -count=1
```

The graph can also be regenerated without a cluster from the checked-in SQL
dump in a disposable local MySQL container:

```bash
fixture_dir=backend/src/apiserver/migration/mlmd/testdata/release_218
podman run --rm -d --name kfp-mlmd-218-sql-fixture \
  -e MYSQL_ALLOW_EMPTY_PASSWORD=yes mysql:8.4
until podman exec kfp-mlmd-218-sql-fixture mysqladmin -uroot ping --silent; do sleep 1; done
podman exec -i kfp-mlmd-218-sql-fixture mysql -uroot < "$fixture_dir/source.sql"
python3 "$fixture_dir/capture_graph.py" \
  --container kfp-mlmd-218-sql-fixture --output /tmp/release_218_source_graph.json
diff -u "$fixture_dir/source_graph.json" /tmp/release_218_source_graph.json
podman stop kfp-mlmd-218-sql-fixture
```
