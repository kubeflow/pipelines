# KFP 2.18.0 source fixture

This fixture was captured from a disposable installation of the released KFP
`2.18.0` tag (`555bfa60a70da88c594960bdacfd78fa7c061043`). It uses that
tag's single-user MySQL manifests, MLMD `1.14.0`, Argo Workflows `v4.1.6`,
and the `2.18.0` SDK, API server, driver, launcher, persistence agent, and
metadata writer. The Argo manifests were fetched from the tag pinned by KFP's
Kustomize files; their archive SHA-256 was
`e793bc7ac65f63fc90af373d75a91fd560a3183b4e0a81c581ab952e8d4b2645`.
The existing migration fixtures were not inputs.

## Source and expected results

`source.sql` is the captured upgrade input. The
[fixture generator](../../fixturetools/build_migration_fixture.py) reads it
and writes `migration_fixture.json`. Its `expected` section records the
settled native fields, and `unresolved` records open mappings. To change a
mapping, update the generator and regenerate the contract. Edit `cases.json`
when changing a supplemental scenario.

## Contents

| File | Source |
| --- | --- |
| `source.sql` | One transaction's `mysqldump` of the released `mlpipeline` and `metadb` databases, including **every** `run_details` and old `tasks` row and all MLMD contexts, executions, artifacts, properties, and events. |
| `migration_fixture.json` | Generated contract with normalized source rows, expected native rows, field dispositions, and unresolved mappings. |
| `produce_native_fingerprint.txt` | Expected native cache key for MLMD execution 2. The migration and native driver tests independently check this key using the captured producer and log-publishing configuration. |
| `provenance.json` | Released source revision, source databases, file checksums, and record counts. |
| `cases.json` | Named synthetic source and expected cases for states and cache gating, all eight task types, orphaned contexts and missing parents, missing optional properties, unknown execution types, and metric splitting. |
| `workflows.json` | Submitted Argo Workflow objects and their terminal states. |
| `api_runs.json`, `runs.json` | Released API responses and run IDs/states. |
| `deployment-images.json`, `install-config.json` | Installed image tags and release database configuration. |
| `table-counts.tsv`, `coverage.tsv` | Compact checks of source counts, task rows, fingerprints, metrics, and ID ranges. |

[Fixture tools](../../fixturetools/) contains the capture scripts, pipeline
source, exact YAML packages submitted to the API, and fixture generator tests.
The pipeline source was formatted after capture, so recompiling it changes
the whitespace in the embedded Python.

The runs cover success, an identical cached submission, a 512-character input,
a taken and an untaken condition, an intentional failure, a loop, nested DAG,
exit handler, importer, and sparse MLMD IDs. Before the last run, only the
**next** auto-increment values on MLMD `Context`, `Execution`, `Artifact`, and
`Event` were advanced to `1000`; existing source rows were not changed. The
last run was then written by the released KFP components. `coverage.tsv`
records the resulting ID ranges.

The final snapshot has 7 `run_details` rows, all 191 old `tasks` rows,
10 MLMD contexts, 34 executions, 22 artifacts, and 26 events. Six Workflows
were successful and the intentional failure Workflow failed. There are 12
task rows carrying an MLMD execution ID and fingerprint, 6 skipped task rows,
and 4 MLMD metrics artifacts. The maximum execution, artifact, and event IDs
are `1003`, confirming the intended gaps.

All 191 old task UUIDs appear in the released `GetRun` responses. Of those,
179 are persistence-agent Workflow reports without MLMD execution IDs and 12
are cache rows with MLMD execution IDs. Each of the 12 cache rows resolves to
one of the 34 captured MLMD executions and therefore creates no separate
native task. The [migration design](../../../../../../../proposals/12147-mlmd-removal/database-migration-design.md#migration-phases)
skips the 179 Workflow report rows in the native task backfill. Their Argo
snapshots remain in `run_details.WorkflowRuntimeManifest`.

The missing IDs are a property of the released write paths, not a result of
calculating the fixture after MLMD removal. The persistence agent reported
Argo Workflow nodes through `reportTasksFromExecution` and `toModelTasks`;
that conversion populated run, namespace, and task timestamps but did not
populate `MLMDExecutionID`. The launcher and driver separately created cache
entries with both `MLMDExecutionID` and fingerprint, and the corresponding
MLMD execution stored the fingerprint as a custom property. The captured SQL
contains both the old API `mlpipeline` database and the MLMD `metadb` database.

The fixture expects 34 native tasks, 26 artifacts after metric splits, and
26 artifact-task links. The native task UUIDs derive from MLMD execution IDs.
The pre-upgrade API still shows 191 legacy task rows. Validate the upgraded
task API against the MLMD-derived inventory and compare saved Workflow
manifests with their captured values. These are source-side expectations.

The text contract keeps all captured MLMD source fields, including typed
properties and millisecond timestamps. Original timestamp precision, several
execution properties, and event time still need final native mappings. The
target TaskType values follow the current `PipelineTask` proto (1 to 8).
The remaining schema and field questions keep the full target contract
provisional.

The released source rows are unchanged.
The generator verifies that each old task `Payload` repeats its captured task
fields before classifying that payload as redundant.
The supplemental cases specify source-side transform expectations; they are
not SQL replay snapshots. PR 2 must seed their source records through MLMD
before using them as migration output assertions.

## Refresh the capture

Install the KFP `2.18.0` tag's
`manifests/kustomize/cluster-scoped-resources` and
`manifests/kustomize/env/platform-agnostic` in an isolated cluster. The latter
references Argo `v4.1.6`. If Kustomize times out fetching Argo, use a local
copy of that exact tag for its two remote manifest references. Scale down the
UI, viewer, visualization server, and scheduled workflow controller if they
are not needed for these runs. Wait for the API, MySQL, MLMD, persistence
agent, object store, Argo controller, and cache server to be ready.

Run the capture commands from `backend/src/apiserver/migration/mlmd/fixturetools`.
Set `KFP_HOST` to the disposable installation's API URL and
`KFP218_OUTPUT` to an empty directory for the new capture. With KFP SDK
`2.18.0`, run `python3 run_fixture.py` to submit the checked-in packages.
Use the comma-separated `FIXTURE_CASES` filter to submit
`success-first,success-cache,success-long,success-optional,intentional-failure,structure`
first. The runner records each run ID before waiting and resumes by that ID
on restart. `success-cache` can only start after `success-first` succeeds.
Then advance the four next MLMD IDs to `1000` with:

```sql
ALTER TABLE metadb.Context AUTO_INCREMENT=1000;
ALTER TABLE metadb.Execution AUTO_INCREMENT=1000;
ALTER TABLE metadb.Artifact AUTO_INCREMENT=1000;
ALTER TABLE metadb.Event AUTO_INCREMENT=1000;
```

Submit `success-sparse` with `FIXTURE_CASES=success-sparse`. Wait until all API
runs and Argo Workflows are terminal and final reports are stored. Run
`python3 capture_api.py`, then set `KFP218_KUBECONFIG` and `KFP218_CONTEXT`
for the disposable installation and run `bash capture_fixture.sh`. Both
scripts save their results to `KFP218_OUTPUT`.

After reviewing and copying the captured files into `testdata/release_218`,
run the generator from `fixturetools/` to refresh the text contract. Install
`requirements.txt` in the active Python environment to decode protobuf
parameters and read the saved pipeline specs.

```bash
python3 -m pip install -r requirements.txt
python3 build_migration_fixture.py
```

The fixture preserves typed input and output values. It records I/O types and
producer links when the captured run, execution, and pipeline spec establish
them. For example, the loop aggregate keeps its captured list while its native
I/O type remains pending.

Run the [fixture validation suite](../../README.md#validate-the-fixture)
after regenerating the contract.
