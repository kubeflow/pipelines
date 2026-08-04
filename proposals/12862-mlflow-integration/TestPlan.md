# MLflow Integration — Test Plan

**Feature**: MLflow Integration for Kubeflow Pipelines  
**Design Doc**: [KEP-12862](https://github.com/kubeflow/pipelines/blob/master/proposals/12862-mlflow-integration/README.md)

---

## 1\. Test Plan

### 1.1 CreateRun with MLflow

| Priority | Scenario | Steps | Expected |
| :---- | :---- | :---- | :---- |
| **P0** | Default experiment | 1\. Configure global `plugins.mlflow` 2\. Create a run without `plugins_input` | Experiment **`KFP-Default`** (unless the admin default name overrides it); MLflow parent run exists; `plugins_output.mlflow` includes `experiment_id`, `root_run_id`, `run_url`; plugin `state` \= `SUCCEEDED` |
| **P0** | Custom experiment name | 1\. Create a run with `plugins_input.mlflow.experiment_name = "my-exp"` | Experiment `my-exp` exists; parent run lives under it |
| **P0** | Experiment id wins over name | 1\. Send both `experiment_name` and `experiment_id` | The id is used; no lookup by name |
| **P0** | Runtime config on driver and launcher | 1\. Create a run with MLflow enabled 2\. Inspect the compiled workflow | A JSON MLflow config env var appears on **driver** and **launcher** templates only, with endpoint, workspace (when workspaces are enabled), parent run id, experiment id, auth type, timeout, and TLS skip flag |
| **P0** | Failed save after MLflow parent exists | 1\. MLflow parent is created, then workflow create or DB write fails | KFP run creation fails; orphan MLflow parent run remains in `RUNNING` state in MLflow indefinitely. No compensating update or delete is performed on the MLflow run, which is acceptable per design. |
| **P0** | Clients cannot set plugin output | 1\. Create a run with client-supplied `plugins_output` | Request is rejected |
| **P1** | KFP metadata in MLflow tags | 1\. Run linked to `pipeline_id` / `pipeline_version_id` | MLflow tags include pipeline run id, pipeline run URL, and pipeline identifiers when present |
| **P1** | Deep links are usable | 1\. Complete a successful run with MLflow | Tag and `plugins_output` URLs open the correct **KFP run** page and **MLflow run** page (including workspace query when workspaces are on) |
| **P1** | Experiment description variants | 1\. Omit description, send empty string, or send a custom description | Behavior matches configured defaults (including optional empty description) |
| **P0** | Same pipeline, different experiments | 1\. Two runs with different `experiment_name` values | Two MLflow experiments and two separate parent runs |
| **P2** | Concurrent experiment creation | 1\. Two creates race; MLflow reports the experiment already exists | Integration recovers (e.g. lookup by name) and both runs succeed |

### 1.2 Run finishes (terminal pipeline state)

| Priority | Scenario | Steps | Expected |
| :---- | :---- | :---- | :---- |
| **P0** | Success | 1\. Pipeline ends in **Succeeded** | MLflow parent run ends `FINISHED`; plugin output reflects success |
| **P0** | Failure / cancel | 1\. Pipeline ends in **Failed** or **Canceled** | KFP **Canceled** (or canceling) maps to MLflow `KILLED`; KFP **Failed** and other non-success terminals (per mapping) map to `FAILED` |
| **P1** | Straggler nested runs | 1\. Some nested runs still active when the pipeline completes | All relevant nested runs are found and updated (pagination respected for large fan-out) |
| **P1** | Persisted output | 1\. After terminal handling | Latest plugin output is stored on the KFP run record |
| **P1** | Sync errors | 1\. MLflow returns errors while syncing terminal state | Errors are logged; the KFP run’s terminal state is still recorded as usual |

### 1.3 RetryRun

| Priority | Scenario | Steps | Expected |
| :---- | :---- | :---- | :---- |
| **P0** | Retry after failure | 1\. Failed run that had MLflow output 2\. Retry | Parent MLflow run and nested runs in `FAILED`/`KILLED` go back to `RUNNING`; nested runs that were `FINISHED` stay `FINISHED`; when retried tasks re-execute, existing reopened nested runs are reused (matched by task name tag) rather than duplicated, so nested run count under the parent remains unchanged and reused runs contain updated retry metrics/parameters |
| **P1** | Retry without MLflow output | 1\. Failed run with no MLflow plugin output | No MLflow side effects; no errors |

### 1.4 CloneRun

| Priority | Scenario | Steps | Expected |
| :---- | :---- | :---- | :---- |
| **P0** | Clone keeps plugin input | 1\. Clone a run that had `plugins_input.mlflow` | New run inherits input; a **new** MLflow parent is created for the clone |
| **P1** | Clone vs original | 1\. Both runs complete | May share an experiment; must have distinct MLflow run ids |

### 1.5 Recurring runs

| Priority | Scenario | Steps | Expected |
| :---- | :---- | :---- | :---- |
| **P0** | Job remembers plugin input | 1\. Create a recurring job with `plugins_input` | Value is stored and appears on the scheduled workflow |
| **P0** | Each trigger inherits input | 1\. Schedule fires → new run | Triggered runs carry the same `plugins_input` |
| **P0** | Recurring job tied to a pipeline version | 1\. Job references a pipeline version and uses plugins | Job is accepted; controller path creates runs via the API without requiring an inline workflow spec |
| **P1** | One MLflow parent per trigger | 1\. Several scheduled executions | Each trigger gets its own MLflow parent under the configured experiment |

### 1.6 Negative & edge cases

| Priority | Scenario | Steps | Expected |
| :---- | :---- | :---- | :---- |
| **P0** | MLflow unreachable at create | 1\. Bad or unreachable endpoint | KFP run is created; plugin output shows failure; runtime env is absent since setup never completed |
| **P0** | Plugin explicitly disabled | 1\. `CreateRun` with `plugins_input.mlflow.disabled: true` | No MLflow side effects; KFP run is created and proceeds normally |
| **P0** | Invalid `plugins_input.mlflow` | 1\. Invalid JSON, unknown fields, or structurally invalid values under `plugins_input.mlflow` | Request rejected with a clear validation error |
| **P1** | Terminal handling without stored MLflow ids | 1\. Corrupt or incomplete stored plugin output | Failure path updates plugin state appropriately |
| **P1** | MLflow HTTP behavior | 1\. Client vs server errors from MLflow | Retries only where appropriate; no infinite retry on permanent errors |

### 1.7 Compiled workflow injection

| Priority | Scenario | Steps | Expected |
| :---- | :---- | :---- | :---- |
| **P0** | Env set only on driver and launcher | 1\. Inspect templates after compile | MLflow runtime env appears only on driver/launcher (not on arbitrary DAG or executor-only templates unless specified by design) |
| **P0** | User container gets runtime MLflow env vars | 1\. Run a pipeline task with MLflow enabled 2\. Inspect the launched user container env | User container receives `MLFLOW_TRACKING_URI`, `MLFLOW_EXPERIMENT_ID`, `MLFLOW_RUN_ID`, and `MLFLOW_TRACKING_AUTH` from the driver (plus `MLFLOW_WORKSPACE` when workspaces are enabled); `MLFLOW_RUN_ID` points to the task nested run (not the parent run) |
| **P1** | Duplicate env keys | 1\. Template already defines the same env name | Injected value replaces consistently; no duplicate keys |
| **P2** | MLflow globally disabled | 1\. No `plugins.mlflow` configuration | No MLflow env on workflows |
| **P1** | `KFP_MLFLOW_CONFIG` not exposed to user code | 1\. Run a pipeline task with MLflow enabled 2\. Inspect the user container environment | `KFP_MLFLOW_CONFIG` is not present in the user process environment; only standard `MLFLOW_*` env vars are available |

### 1.8 Cluster & E2E

| Priority | Scenario | Steps | Expected |
| :---- | :---- | :---- | :---- |
| **P0** | End-to-end success | 1\. Cluster with KFP and MLflow 2\. Run a pipeline | Parent and nested MLflow runs; `plugins_output` populated |
| **P0** | User `mlflow.log_metric()` without explicit setup | 1\. Run a pipeline component that calls `mlflow.log_metric()` directly without `mlflow.start_run()` or explicit run id 2\. Complete the run | Metric is recorded on the task's nested MLflow run identified by `MLFLOW_RUN_ID`; no duplicate nested run is created for this logging path |
| **P0** | Retry and schedules | 1\. Failed retry; recurring job | Behavior matches sections 1.3 and 1.5 |
| **P0** | Upgrade without MLflow | 1\. Deploy without `plugins.mlflow` | Existing behavior unchanged; no MLflow env |
| **P1** | Enable MLflow on an existing deployment | 1\. Start from a deployment with existing runs created before MLflow enablement 2\. Enable `plugins.mlflow` and roll out 3\. Verify `GetRun`/`ListRuns` on pre-existing runs and observe an in-flight run started before enablement 4\. Create a new run after enablement | Pre-existing runs remain readable with plugin fields handled safely (empty when absent); in-flight runs complete normally; new runs created after enablement receive MLflow tracking (`plugins_output` and parent/nested MLflow runs) |
| **P1** | Multi-tenant namespace config merge | 1\. Per-namespace `kfp-launcher` overrides selected fields 2\. Run in multiple namespaces | Namespace overrides apply only to that namespace; unset fields inherit global config; workspace behavior matches effective settings |
| **P0** | Delete KFP run | 1\. After a successful run, delete the KFP run | MLflow runs remain |
| **P2** | TLS networks | Deploy MLflow behind **HTTPS** | MLflow run creation must be successful. |

### 1.9 Driver-level MLflow behavior

| Priority | Scenario | Steps | Expected |
| :---- | :---- | :---- | :---- |
| **P0** | Parallel-for hierarchy without extra iteration DAG run | 1\. Run a pipeline with a parallel-for loop 2\. Inspect MLflow run tree and task tags | Hierarchy is parent run -> loop nested run -> iteration nested runs; no extra intermediate nested run is created by the iteration DAG driver |
| **P0** | Cache hit still appears in MLflow | 1\. Run a cacheable pipeline task once 2\. Re-run to trigger cache hit 3\. Inspect MLflow nested runs for second run | Cached task has a nested MLflow run on the second run; cached outputs are logged and that nested run is marked complete |
| **P0** | Driver-only task creates nested run | 1\. Execute a workflow that includes a driver-only task (for example PVC create/delete) 2\. Inspect MLflow runs | Driver-only task gets its own nested MLflow run and it is marked complete/failed according to task outcome, even without launcher execution |

### 1.10 Launcher MLflow logging

| Priority | Scenario | Steps | Expected |
| :---- | :---- | :---- | :---- |
| **P0** | Logs parameters and scalar metrics to nested run | 1\. Run a task with known input parameters and scalar outputs 2\. Inspect the task nested MLflow run | Nested run contains the expected parameters and scalar metrics logged by launcher `log-batch` calls |
| **P1** | Chunks large parameter sets correctly | 1\. Run a task with more than 100 input parameters 2\. Inspect MLflow nested run contents | All parameters are present on the nested run even when launcher must split them across multiple `log-batch` requests |
| **P0** | Nested run terminal status mirrors task result | 1\. Run one successful task and one failing task 2\. Inspect nested MLflow run statuses | Successful task nested run is marked `FINISHED`; failing task nested run is marked `FAILED` (or `KILLED` for canceled task, per mapping) |
| **P0** | Launcher MLflow failure does not block task outcome | 1\. Make MLflow unreachable during launcher post-execution logging 2\. Run a task that otherwise succeeds | Task completion result in KFP is unchanged; MLflow logging/status-update errors are logged and do not fail the user task |

### 1.11 API readback compatibility

| Priority | Scenario | Steps | Expected |
| :---- | :---- | :---- | :---- |
| **P1** | `ListRuns` returns plugin fields for all returned runs | 1\. Create multiple runs (MLflow-enabled and non-MLflow) 2\. Call `ListRuns` for the namespace | Each returned run includes `plugins_input` / `plugins_output` as stored for that run (not only the currently inspected run) |
| **P1** | Pre-upgrade runs with `NULL` plugin columns are handled safely | 1\. Query runs created before MLflow columns existed (or seed rows with `NULL` plugin columns) 2\. Call `GetRun` and `ListRuns` | API returns empty maps for plugin fields and does not error on `NULL` database values |

### 1.12 Authentication modes and credential paths

| Priority | Scenario | Steps | Expected |
| :---- | :---- | :---- | :---- |
| **P0** | Kubernetes auth mode | 1\. Configure `authType: "kubernetes"` 2\. Create and run a pipeline | API server uses Kubernetes SA token for MLflow calls; user container gets `MLFLOW_TRACKING_AUTH=kubernetes`; no secret credential env vars are required |
| **P0** | Bearer auth mode | 1\. Configure `authType: "bearer"` with namespace `credentialSecretRef` 2\. Create and run a pipeline | Secret is mounted with `valueFrom.secretKeyRef`; `MLFLOW_TRACKING_TOKEN` is present on driver/launcher/user containers; MLflow operations succeed with token auth |
| **P0** | Basic-auth mode | 1\. Configure `authType: "basic-auth"` with namespace `credentialSecretRef` 2\. Create and run a pipeline | Secret is mounted with `valueFrom.secretKeyRef`; `MLFLOW_TRACKING_USERNAME` and `MLFLOW_TRACKING_PASSWORD` are present on driver/launcher/user containers; MLflow operations succeed with basic auth |
| **P0** | Missing namespace `credentialSecretRef` disables secret-based auth | 1\. In multi-user mode set `authType: "bearer"` or `"basic-auth"` without namespace `credentialSecretRef` 2\. Create a run in that namespace | MLflow integration is disabled for that namespace/run (no MLflow side effects); run creation and execution still proceed |
| **P1** | Invalid credentials at API-server MLflow call path | 1\. Provide invalid/expired credentials affecting CreateRun or terminal sync calls 2\. Create/complete a run | Run proceeds; `plugins_output.mlflow.state` is set to failed with an auth-related `state_message` for API-server-side failures |
| **P1** | Invalid credentials at driver/launcher MLflow call path | 1\. Provide credentials that fail only during driver/launcher MLflow calls (nested run/logging) 2\. Run a task that otherwise succeeds | Task outcome is unchanged (not blocked by MLflow auth failure); errors are logged; do not require per-task auth failures to be reflected in `plugins_output.mlflow.state` |

