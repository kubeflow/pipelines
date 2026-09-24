# Upgrading Pipeline Definitions

Current Kubeflow Pipelines supports v2 SDK pipelines compiled to
[IR YAML](../concepts/ir-yaml.md) and the `/apis/v2beta1` REST API. Before upgrading,
export the source of your pipelines and recompile them with the current SDK.
Historical Argo-format pipelines and legacy v1 component YAML are unsupported.
This is a breaking change for existing v2 pipeline sources that load
`implementation: container:` component files, as well as for v1 pipelines.

## Author and compile components

Use [`@dsl.component`](components/lightweight-python-components.md) for Python
functions and [`@dsl.container_component`](components/container-components.md)
for existing container entrypoints. Use `dsl.Input`, `dsl.Output`,
`dsl.InputPath`, and `dsl.OutputPath` to declare the typed interface; use
`dsl.ContainerSpec` to configure image, command, and arguments.

Compile both components and pipelines with `kfp.compiler.Compiler().compile()`.
The `load_component_from_file`, `load_component_from_text`, and
`load_component_from_url` helpers accept only the resulting PipelineSpec IR,
optionally followed by a PlatformSpec document.

## Migrate legacy container component files before upgrading

Choose one of these migration paths:

- Rewrite the component using `@dsl.container_component` and compile it with the
  current SDK. Preserve its image, command, arguments, typed inputs/outputs,
  defaults, and optional-input behavior.
- Before upgrading, use an older KFP v2 SDK that still supports legacy container
  YAML, in a separate environment, to convert the file to IR:

  ```python
  # Run with an older compatible KFP v2 SDK, not the current SDK.
  from kfp import compiler, components

  component = components.load_component_from_file('legacy-component.yaml')
  compiler.Compiler().compile(component, 'component-ir.yaml')
  ```

Then change the pipeline source or component package to load `component-ir.yaml`
instead of the legacy file. Recompile and test the pipeline with the current SDK
before deploying it. If a third-party package loads legacy YAML during import,
use a migrated package release or coordinate migration with its publisher.

The current SDK cannot perform the legacy-to-IR conversion. Legacy graph
implementations and Argo Workflow YAML must be rewritten as v2 pipelines; they
are not covered by the container conversion path above.

## Existing runs and recurring runs

Historical database records are retained, but the v1 run-details, graph, output,
and comparison UI views are removed. Retaining a record does not preserve
read-only access to its old UI. Export any required historical information before
upgrading. The v2 API continues to expose stored parameters, including historical
name/value arrays.

New runs and recurring runs store ownership and pipeline references in their
native database columns only; they no longer populate `resource_references`.
Existing reference rows remain available for historical ownership fallback,
migrations, and deletion cleanup. Integrations that query the database directly
must use the native columns for new records.

Existing recurring runs with unsupported embedded workflow templates stop firing
after upgrade, regardless of the former `BLOCK_V1_PIPELINES` setting. An embedded
template must contain `spec.podMetadata.labels` or `spec.podMetadata.annotations`
with `pipelines.kubeflow.org/v2_component: "true"`, as emitted by the IR compiler.
The old workflow-level `pipelines.kubeflow.org/v2_pipeline` marker alone is not
sufficient, and malformed templates are rejected too. The controller currently
reports an error on each attempted submission; it does not automatically disable
the schedule or record a dedicated unsupported-template status. Disable affected
recurring runs before upgrading, then recreate them from recompiled IR pipelines.
Do not add a marker to an old workflow as a substitute for recompilation.

Retry uses the persisted workflow manifest and requires the same pod-metadata
marker. Controllers and admission webhooks must preserve `spec.podMetadata`;
removing it makes even an originally IR-compiled run non-retriable. Create a new
run from pipeline IR if the stored manifest no longer carries the marker.

The `kubeflow.org/v1beta1` ScheduledWorkflow Kubernetes CRD is still used by native
v2 recurring runs. Its version is independent of the removed KFP v1beta1 REST and
gRPC APIs; do not delete this CRD when upgrading.

## Submit and inspect runs

Use `kfp.Client` to upload the compiled pipeline, create an experiment in the
execution namespace, and submit runs. Direct REST integrations use
`/apis/v2beta1` and its typed fields (`display_name`, `experiment_id`, `namespace`,
and `pipeline_version_reference`) rather than resource-reference lists.

See [connecting to the API](core-functions/connect-api.md),
[compiling pipelines](core-functions/compile-a-pipeline.md), and the
[REST API reference](../reference/api/kubeflow-pipeline-api-spec.md).
