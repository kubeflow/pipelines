# KEP-13884: Native TriggerPipeline (Airflow-like TriggerDagRun)

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User stories](#user-stories)
  - [DSL / IR contract](#dsl--ir-contract)
  - [Version selection](#version-selection)
  - [Parent linkage](#parent-linkage)
  - [Auth](#auth)
- [Design Details](#design-details)
  - [Proto](#proto)
  - [SDK](#sdk)
  - [Backend](#backend)
  - [Frontend](#frontend)
  - [Data flow](#data-flow)
- [Frontend Considerations](#frontend-considerations)
- [KFP Local Considerations](#kfp-local-considerations)
- [Test Plan](#test-plan)
- [Migration Strategy](#migration-strategy)
- [Alternatives Considered](#alternatives-considered)
- [Open Questions](#open-questions)
- [Implementation History](#implementation-history)
- [References](#references)
<!-- /toc -->

- **Authors**: @kliushenkov
- **Status**: implementable (MVP in PR)
- **Tracking issue**: https://github.com/kubeflow/pipelines/issues/13884
- **Implementation PR**: https://github.com/kubeflow/pipelines/pull/14100
- **Created**: 2026-08-20
- **Last updated**: 2026-08-21

## Summary

Add a first-class executor `TriggerPipeline` that starts an **independent**
Kubeflow Pipelines run of a registered pipeline from a parent pipeline task,
similar to Airflow `TriggerDagRunOperator`. Unlike pipelines-as-components
(nested DAG), the child pipeline IR is **not** inlined into the parent Argo
Workflow.

The MVP ships end-to-end: proto + SDK (`dsl.trigger_pipeline`) + Argo compiler
template + launcher CreateRun/wait + MLMD I/O metadata + UI Open Child Run.

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
4. Publish task outputs `run_id`, `state`, and resolved `pipeline_version_id`.
5. Persist MLMD custom properties `child_run_id` and `child_pipeline_version_id`,
   and record resolved child-launch parameters as execution `inputs`.
6. UI: **Open Child Run** + **Triggered Child Run** details block; Input/Output
   tab shows launch parameters and outputs; `run_id` is linkable; navigating to
   the child run remounts run details so Sub-DAG layers do not leak from the
   parent.
7. Version resolution when `pipeline_version_id` is empty: prefer a child
   PipelineVersion whose `display_name`/`name` matches the parent run’s version;
   otherwise fall back to latest (`created_at desc`).

### Non-Goals

1. Fire-and-forget UX polish (`failed_states` / `allowed_states` like Airflow).
2. Idempotency / `skip_when_already_exists`.
3. Passing child **artifacts** / pipeline parameter outputs back through the
   CreateRun API into the parent graph (use object storage + `dsl.importer`, or
   consume trigger outputs `run_id` / `state` / `pipeline_version_id` only).
4. Dedicated React Flow canvas node type `TRIGGER` (MVP reuses `EXECUTION`).
5. KFP local (Subprocess/Docker) execution of trigger nodes in the first
   iteration.
6. Cross-run **Lineage Explorer** edges between parent task and child run
   (child is a separate PipelineRun MLMD context). Follow-up if/when Run labels
   or cross-context lineage APIs land.

## Proposal

Mirror the **importer** pattern (system executor + single launcher pod), not a
user Python component calling `kfp.Client`.

```
dsl.trigger_pipeline
  → IR ExecutorSpec.trigger_pipeline
  → Argo template system-trigger-pipeline
  → launcher-v2 --executor_type trigger_pipeline
  → resolve PipelineVersion
  → CreateRun (+ optional wait)
  → MLMD inputs + outputs + child_run_id
  → UI Open Child Run / Triggered Child Run
```

### User stories

1. **Orchestrator author**: Register heavy subpipelines once; parent only
   references them by name so the parent Workflow stays small.
2. **Operator**: From the parent run graph, open the trigger task, inspect
   launch arguments and child `run_id` / `state` / `pipeline_version_id`, and
   jump to the child run without losing context incorrectly.
3. **Release engineer**: Upload parent and child PipelineVersions with the same
   version label; empty `pipeline_version_id` keeps them aligned, with latest as
   fallback.

### DSL / IR contract

```python
task = dsl.trigger_pipeline(
    pipeline_name="get-sasrec-recommendations",
    arguments={"model_name": model_name},  # constants or PipelineParameterChannel
    pipeline_version_id="",                # empty → match parent version name, else latest
    wait_for_completion=True,
    poke_interval_seconds=30,
)
# task.outputs["run_id"]
# task.outputs["state"]
# task.outputs["pipeline_version_id"]

# Downstream parent tasks may consume trigger outputs only:
summarize(run_id=task.outputs["run_id"], state=task.outputs["state"])
```

`TriggerPipelineSpec` fields: `pipeline_name`, `pipeline_version_id`,
`wait_for_completion`, `poke_interval_seconds`. Child run parameters flow through
normal `TaskInputsSpec` / resolved runtime parameters — the child pipeline spec
is never copied into the parent IR.

### Version selection

Resolution order in the launcher:

1. Explicit `pipeline_version_id` on the trigger spec (validated via Get).
2. Else list child versions (`created_at desc`) and pick one whose
   `display_name` or `name` matches the parent run’s PipelineVersion
   `display_name`/`name`.
3. Else the newest child version.

The resolved ID is passed to CreateRun and published as output
`pipeline_version_id` and MLMD `child_pipeline_version_id`.

### Parent linkage

v2beta1 `Run` has no Labels field. MVP encodes parent linkage in the child run
**Description** using:

- `pipelines.kubeflow.org/parent-run-id`
- `pipelines.kubeflow.org/parent-task-name`

### Auth

In-cluster call from launcher-v2 to `ml-pipeline`. Prefer projected SA token
(`/var/run/secrets/kubeflow/pipelines/token`); on standalone clusters without
that mount, fall back to `PassThroughAuth` (compatible with multi_user=false
Kind/dev installs).

## Design Details

### Proto

`PipelineDeploymentConfig.TriggerPipelineSpec` and
`ExecutorSpec.trigger_pipeline = 5` in `api/v2alpha1/pipeline_spec.proto`.

### SDK

| Path | Role |
|------|------|
| `sdk/python/kfp/dsl/trigger_pipeline_node.py` | Public `dsl.trigger_pipeline` |
| `sdk/python/kfp/dsl/trigger_pipeline_component.py` | Component wrapper |
| `structures.py` / `pipeline_task.py` / `__init__.py` / `pipeline_spec_builder.py` | IR wiring |

Outputs declared: `run_id`, `state`, `pipeline_version_id` (all `String`).

### Backend

| Path | Role |
|------|------|
| `compiler/visitor.go` | `TriggerPipeline(...)` visitor |
| `compiler/argocompiler/trigger_pipeline.go` | `system-trigger-pipeline` template (flags aligned with importer: `cache_disabled`, TLS, `ca_cert_path`, logs) |
| `compiler/argocompiler/dag.go` | Single task, no separate driver (like importer) |
| `cmd/launcher-v2/main.go` | `--executor_type trigger_pipeline` switch + flag validation |
| `component/trigger_pipeline_launcher.go` | Resolve inputs → resolve version → CreateRun → wait → PublishExecution |

Launcher behavior:

1. Resolve task input parameters (constants, component inputs, upstream task
   outputs) and store them on MLMD execution `inputs`.
2. Resolve child PipelineVersion (see [Version selection](#version-selection)).
3. `CreateRun` in the parent’s experiment with runtime parameters and parent
   linkage Description.
4. If `wait_for_completion`, poll until terminal; fail parent task unless
   `SUCCEEDED`.
5. Publish outputs + custom properties `child_run_id` /
   `child_pipeline_version_id`.

### Frontend

| Path | Role |
|------|------|
| `RuntimeNodeDetailsV2.tsx` | Banner **Open Child Run**; **Triggered Child Run** details table with linked Child Run |
| `InputOutputTab.tsx` | Input Parameters from MLMD; Output Parameters; link `run_id` when `child_run_id` present |
| `RunDetailsRouter.tsx` | `key={runId}` remount so Open Child Run does not leak parent Sub-DAG layers |
| `RunDetailsV2.tsx` | Toolbar actions use live `runId` from route params |

Canvas: MVP reuses `EXECUTION` node type (no dedicated `TRIGGER` glyph).

### Data flow

```
Parent IR (small)
  └─ trigger task (system-trigger-pipeline pod)
        ├─ MLMD execution.inputs  = { child launch args }
        ├─ MLMD execution.outputs = { run_id, state, pipeline_version_id }
        ├─ MLMD custom props      = { child_run_id, child_pipeline_version_id }
        └─ Child Run (independent WF + MLMD context)
              └─ child tasks / child pipeline outputs (not pulled into parent)
```

Parent downstream tasks may only depend on trigger outputs (`run_id`, `state`,
`pipeline_version_id`). Child-internal outputs remain visible on the child run’s
Input/Output panel.

## Frontend Considerations

- Side panel only (no new graph node type).
- Open Child Run must remount run details (`key={runId}`) so `layers` /
  selection from the parent do not appear on the child graph.
- Lineage Explorer is unchanged (no cross-run edges).

## KFP Local Considerations

Local Subprocess/Docker runners do not implement system executors like importer
today. Trigger nodes are unsupported in local mode for MVP; remote API-server
execution (cluster) is required.

## Test Plan

| Layer | Coverage |
|-------|----------|
| SDK | `trigger_pipeline_node_test.py` — IR emission, validation, outputs |
| Backend unit | Parent-label helpers, input resolve, version identity match, terminal states |
| Frontend | Vitest: Open Child Run / Triggered Child Run / linked `run_id` in I/O |
| E2E (manual / Kind) | `backend/test/trigger_pipeline_e2e/toy_trigger_test.py`, `upload_manual_trigger.py` |
| Compile smoke | Parent IR contains `triggerPipeline`; child templates not inlined |

## Migration Strategy

No breaking change. Opt-in DSL. Existing nested pipelines-as-components continue
to work. Users with large orchestrators can migrate heavy subpipelines to
registered pipelines + `dsl.trigger_pipeline` after rolling out a **launcher**
(and matching driver/apiserver where flags require it) image that includes the
new executor type.

## Alternatives Considered

1. **Pipelines-as-components only** — good UX drill-down, but WF size grows with
   child IR.
2. **User `@dsl.component` + Client** — flexible but no IR contract, harder to
   secure/observe, no first-class UI.
3. **Argo WorkflowTemplate / sensor** — engine-specific; breaks KFP portability.

## Open Questions

1. Should parent labels become first-class `Run.labels` if/when the API adds them?
2. Should fire-and-forget (no wait) be the default for some orchestrator patterns?
3. Should a later iteration pull selected child pipeline outputs into parent
   parameters (beyond `run_id` / `state` / `pipeline_version_id`)?

## Implementation History

| Date | Change |
|------|--------|
| 2026-08-20 | Initial KEP + MVP stack (proto, SDK, Argo, launcher, minimal Open Run) |
| 2026-08-21 | Kind E2E hardening (flag validation, importer-aligned flags, PassThroughAuth) |
| 2026-08-21 | MLMD inputs, version match/latest, `pipeline_version_id` output, richer UI, Open Child Run remount |

## References

- Issue: https://github.com/kubeflow/pipelines/issues/13884
- PR: https://github.com/kubeflow/pipelines/pull/14100
- Related pattern: importer system executor (`system-importer`)
