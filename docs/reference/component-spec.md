# Component Specification

Components and pipelines share the [PipelineSpec IR](../concepts/ir-yaml.md).
Author components with the Python SDK and compile them to IR YAML for sharing.

For containers with an existing entrypoint, use
[`@dsl.container_component`](../user-guides/components/container-components.md):

```python
from kfp import compiler, dsl

@dsl.container_component
def echo(message: str):
    return dsl.ContainerSpec(image='alpine', command=['echo', message])

compiler.Compiler().compile(echo, 'echo.yaml')
```

The compiler records the component's typed interface in `root` and its container
implementation in `deploymentSpec.executors`. The authoritative schema is
[`api/v2alpha1/pipeline_spec.proto`](https://github.com/kubeflow/pipelines/blob/master/api/v2alpha1/pipeline_spec.proto).

## Loading component definitions

Load compiled components with
[`kfp.components.load_component_from_file`](../user-guides/components/load-and-share-components.md).
The file, text, and URL loaders accept only PipelineSpec IR YAML, optionally
followed by a PlatformSpec document.

Legacy v1 component YAML with top-level `implementation: container:` or
`implementation: graph:` is not supported, even when used by a v2 pipeline.
Raw Argo Workflow YAML is also unsupported. Rewrite components with the v2 SDK,
or convert legacy container files to IR using an older compatible KFP v2 SDK
**before upgrading**. See [migrating component files](../user-guides/migration.md).
