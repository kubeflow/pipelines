# KEP-13884: Native TriggerPipeline (Airflow-like TriggerDagRun)

- **Authors**: @kliushenkov
- **Status**: provisional
- **Tracking issue**: https://github.com/kubeflow/pipelines/issues/13884
- **Created**: 2026-08-20

## Summary

Add a first-class executor `TriggerPipeline` that starts an independent Kubeflow
Pipelines run of a registered pipeline from a parent pipeline task, similar to
Airflow `TriggerDagRunOperator`. Unlike pipelines-as-components (nested DAG),
the child pipeline IR is **not** inlined into the parent Argo Workflow.

## Motivation

Nested pipelines-as-components provide UI drill-down but inflate the parent Argo
Workflow (many nodes, large `pod-spec-patch` parameters). Large orchestrators can
exceed Argo’s compressed Workflow size limit (~1 MiB). Users currently work
around this with ad-hoc `@dsl.component` + `kfp.Client` calls, which lack IR
typing, launcher integration, and UI linkage.

### Goals

1. Provide `dsl.trigger_pipeline(...)` that emits IR
   `ExecutorSpec.trigger_pipeline` (importer-style system executor).
2. Compile to a single Argo template `system-trigger-pipeline` that runs
   launcher-v2 with `--executor_type trigger_pipeline`.
3. Create a child run via the ml-pipeline API; optionally wait for completion.
4. Publish task outputs `run_id` and `state`; store MLMD custom property
   `child_run_id` for UI.
5. Minimal UI: **Open Run** button on the parent task when `child_run_id` is set.

### Non-Goals

1. Fire-and-forget UX polish (`failed_states` / `allowed_states` like Airflow).
2. Idempotency / `skip_when_already_exists`.
3. Passing child artifacts through the KFP API (use object storage +
   `dsl.importer`).
4. Dedicated React Flow canvas node type `TRIGGER` (MVP reuses `EXECUTION`).
5. KFP local (Subprocess/Docker) execution of trigger nodes in the first
   iteration.

## Proposal

Mirror the **importer** pattern (system executor + single launcher pod), not a
user Python component calling `kfp.Client`.

```
dsl.trigger_pipeline
  → IR ExecutorSpec.trigger_pipeline
  → Argo template system-trigger-pipeline
  → launcher-v2 --executor_type trigger_pipeline
  → CreateRun (+ optional wait)
  → MLMD custom property child_run_id
  → UI “Open Run”
```

### DSL / IR contract

```python
task = dsl.trigger_pipeline(
    pipeline_name="get-sasrec-recommendations",
    arguments={"model_name": model_name},  # constants or PipelineParameterChannel
    pipeline_version_id="",                # empty → default / latest version
    wait_for_completion=True,
    poke_interval_seconds=30,
)
# task.outputs["run_id"], task.outputs["state"]
```

`TriggerPipelineSpec` fields: `pipeline_name`, `pipeline_version_id`,
`wait_for_completion`, `poke_interval_seconds`. Child run parameters flow through
normal `TaskInputsSpec` / resolved runtime parameters — the child pipeline spec
is never copied into the parent IR.

### Parent linkage

v2beta1 `Run` has no Labels field. MVP encodes parent linkage in the child run
**Description** using:

- `pipelines.kubeflow.org/parent-run-id`
- `pipelines.kubeflow.org/parent-task-name`

### Auth

In-cluster call from launcher-v2 to `ml-pipeline` (same SA token projection path
as other in-cluster API clients).

## Design Details

### Proto

`PipelineDeploymentConfig.TriggerPipelineSpec` and
`ExecutorSpec.trigger_pipeline = 5` in `api/v2alpha1/pipeline_spec.proto`.

### SDK

- `sdk/python/kfp/dsl/trigger_pipeline_node.py`
- `sdk/python/kfp/dsl/trigger_pipeline_component.py`
- Wire through `structures`, `pipeline_task`, `__init__`,
  `pipeline_spec_builder`

### Backend

- Visitor: `TriggerPipeline(...)`
- Argo: `trigger_pipeline.go` → template `system-trigger-pipeline`
- `dag.go`: single task, no separate driver (like importer)
- `component/trigger_pipeline_launcher.go`: resolve pipeline by name → CreateRun
  → optional poll → publish outputs + MLMD `child_run_id`

### Frontend

`RuntimeNodeDetailsV2`: if execution has custom property `child_run_id`, show
**Child Run ID** in Task Details and an **Open Run** button linking to
`/runs/details/{id}`.

## Frontend Considerations

MVP only adds Open Run / Child Run ID in the existing execution side panel.
No new canvas node type in this iteration.

## KFP Local Considerations

Local Subprocess/Docker runners do not implement system executors like importer
today. Trigger nodes are unsupported in local mode for MVP; document that
remote API-server execution is required.

## Test Plan

- SDK unit tests for `dsl.trigger_pipeline` IR emission and validation
- Go unit tests for parent-label helpers and input-parameter resolution
- Frontend Vitest for Open Run when `child_run_id` is present
- Manual: compile a toy pipeline; confirm IR contains `triggerPipeline` and
  parent WF does not inline child templates; Argo template name is
  `system-trigger-pipeline`

## Migration Strategy

No breaking change. Opt-in DSL. Existing nested pipelines-as-components continue
to work. Users with large orchestrators can migrate heavy subpipelines to
registered pipelines + `dsl.trigger_pipeline` after rolling out a launcher image
that includes the new executor type.

## Alternatives Considered

1. **Pipelines-as-components only** — good UX drill-down, but WF size grows with
   child IR.
2. **User `@dsl.component` + Client** — flexible but no IR contract, harder to
   secure/observe, no first-class UI.
3. **Argo WorkflowTemplate / sensor** — engine-specific; breaks KFP portability.

## Open Questions

1. Should parent labels become first-class `Run.labels` if/when the API adds them?
2. Should fire-and-forget (no wait) be the default for some orchestrator patterns?
