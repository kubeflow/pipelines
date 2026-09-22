# Upgrading Pipeline Definitions

Current Kubeflow Pipelines supports v2 SDK pipelines compiled to
[IR YAML](../concepts/ir-yaml.md) and the `/apis/v2beta1` REST API. Before upgrading,
export the source of your pipelines and recompile them with the current SDK.
Historical Argo-format pipelines are unsupported. Existing v2 pipelines may
continue loading legacy `implementation: container:` component YAML: the SDK
converts those components to native v2 IR without v1 backend support.

## Author and compile components

Use [`@dsl.component`](components/lightweight-python-components.md) for Python
functions and [`@dsl.container_component`](components/container-components.md)
for existing container entrypoints. Use `dsl.Input`, `dsl.Output`,
`dsl.InputPath`, and `dsl.OutputPath` to declare the typed interface; use
`dsl.ContainerSpec` to configure image, command, and arguments.

Compile both components and pipelines with `kfp.compiler.Compiler().compile()`.
The `load_component_from_file`, `load_component_from_text`, and
`load_component_from_url` helpers accept both the resulting IR and legacy
container component YAML. To migrate a shared component file without rewriting
its implementation, load it and compile it to IR:

```python
from kfp import compiler, components

component = components.load_component_from_file('legacy-component.yaml')
compiler.Compiler().compile(component, 'component-ir.yaml')
```

This compatibility adapter does not accept Argo Workflow YAML or restore v1
pipeline compilation. New components should use the v2 decorators above.

## Submit and inspect runs

Use `kfp.Client` to upload the compiled pipeline, create an experiment in the
execution namespace, and submit runs. Direct REST integrations use
`/apis/v2beta1` and its typed fields (`display_name`, `experiment_id`, `namespace`,
and `pipeline_version_reference`) rather than resource-reference lists.

See [connecting to the API](core-functions/connect-api.md),
[compiling pipelines](core-functions/compile-a-pipeline.md), and the
[REST API reference](../reference/api/kubeflow-pipeline-api-spec.md).
