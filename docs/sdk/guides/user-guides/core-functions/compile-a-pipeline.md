# Compile a Pipeline

## Overview

To [submit a pipeline for execution](run-a-pipeline.md), you must compile it to YAML with the KFP SDK compiler.

In the following example, the compiler creates a file called `pipeline.yaml`, which contains a hermetic representation of your pipeline.
The output is called an [Intermediate Representation (IR) YAML][ir-yaml], which is a serialized [`PipelineSpec`][pipeline-spec] protocol buffer message.

```python
from kfp import compiler, dsl

@dsl.component
def comp(message: str) -> str:
    print(message)
    return message

@dsl.pipeline
def my_pipeline(message: str) -> str:
    """My ML pipeline."""
    return comp(message=message).output

compiler.Compiler().compile(my_pipeline, package_path='pipeline.yaml')
```

Because components are actually pipelines, you may also compile them to IR YAML:

```python
@dsl.component
def comp(message: str) -> str:
    print(message)
    return message

compiler.Compiler().compile(comp, package_path='component.yaml')
```

You can view an [example of IR YAML][compiled-output-example] on GitHub.
The contents of the file are not intended to be human-readable, however the comments at the top of the file provide a summary of the pipeline:

```yaml
# PIPELINE DEFINITION
# Name: my-pipeline
# Description: My ML pipeline.
# Inputs:
#    message: str
# Outputs:
#    Output: str
...
```

## Type checking

By default, the DSL compiler statically type checks your pipeline to ensure type consistency between components that pass data between one another.
Static type checking helps identify component I/O inconsistencies without having to run the pipeline, shortening development iterations.

Specifically, the type checker checks for type equality between the type of data a component input expects and the type of the data provided.
See [Data Types][data-types] for more information about KFP data types.

For example, for parameters, a list input may only be passed to parameters with a `typing.List` annotation.
Similarly, a float may only be passed to parameters with a `float` annotation.

Input data types and annotations must also match for artifacts, with one exception: the `Artifact` type is compatible with all other artifact types.
In this sense, the `Artifact` type is both the default artifact type and an artifact "any" type.

As described in the following section, you can disable type checking.

## Compiler arguments

The [`Compiler.compile`][compiler-compile] method accepts the following arguments:

| Name | Type | Description |
|------|------|-------------|
| `pipeline_func` | `function` | _Required_<br/>Pipeline function constructed with the `@dsl.pipeline` or component constructed with the @dsl.component decorator. |
| `package_path` | `string` | _Required_<br/>Output YAML file path. For example, `~/my_pipeline.yaml` or `~/my_component.yaml`. |
| `pipeline_name` | `string` | _Optional_<br/>If specified, sets the name of the pipeline template in the `pipelineInfo.name` field in the compiled IR YAML output. Overrides the name of the pipeline or component specified by the `name` parameter in the `@dsl.pipeline` decorator. |
| `pipeline_parameters` | `Dict[str, Any]` | _Optional_<br/>Map of parameter names to argument values. This lets you provide default values for pipeline or component parameters. You can override these default values during pipeline submission. |
| `type_check` | `bool` | _Optional_<br/>Indicates whether static type checking is enabled during compilation. |
| `kubernetes_manifest_format` | `bool` | _Optional_<br/>Output the compiled pipeline as a Kubernetes manifest instead of PipelineSpec IR. |
| `kubernetes_manifest_options` | `KubernetesManifestOptions` | _Optional_<br/>Options for Kubernetes manifest output during pipeline compilation. Only relevant when `kubernetes_manifest_format=True`. |

## Compiling for Kubernetes Native API Mode

When using Kubeflow Pipelines deployed in Kubernetes native API mode, you can compile pipelines directly to Kubernetes manifests using the KFP SDK. This mode stores pipeline definitions as Kubernetes Custom Resources and provides better integration with Kubernetes native tooling.

### KubernetesManifestOptions Parameters

When using `kubernetes_manifest_format=True`, you can configure the generated Kubernetes resources using `KubernetesManifestOptions`:

| Parameter | Type | Description |
|-----------|------|-------------|
| `pipeline_name` | `str` | _Optional_<br/>Name for the Pipeline resource. |
| `pipeline_display_name` | `str` | _Optional_<br/>Display name for the Pipeline resource. |
| `pipeline_version_name` | `str` | _Optional_<br/>Name for the PipelineVersion resource. |
| `pipeline_version_display_name` | `str` | _Optional_<br/>Display name for the PipelineVersion resource. |
| `namespace` | `str` | _Optional_<br/>Kubernetes namespace for the resources. |
| `include_pipeline_manifest` | `bool` | _Optional_<br/>Whether to include the Pipeline manifest. |

### Using KFP SDK with Kubernetes Native API Mode

The KFP SDK can compile pipelines directly to Kubernetes manifests for native deployment:

```python
import kfp
from kfp import dsl
from kfp.compiler.compiler_utils import KubernetesManifestOptions

@dsl.component
def hello_world(name: str = "World") -> str:
    print(f"Hello, {name}!")
    return f"Hello, {name}!"

@dsl.pipeline(
    name="hello-world-pipeline",
    description="A simple hello world pipeline"
)
def hello_world_pipeline(name: str = "Kubernetes Native") -> str:
    hello_task = hello_world(name=name)
    return hello_task.output

# Compile directly to Kubernetes manifests
kfp.compiler.Compiler().compile(
    pipeline_func=hello_world_pipeline,
    package_path="hello-world-pipeline-k8s.yaml",
    kubernetes_manifest_format=True,
    kubernetes_manifest_options=KubernetesManifestOptions(
        pipeline_name="hello-world-pipeline",
        pipeline_display_name="Hello World Pipeline",
        pipeline_version_name="hello-world-pipeline-v1",
        pipeline_version_display_name="Hello World Pipeline v1",
        namespace="kubeflow-user-example-com",
        include_pipeline_manifest=True
    )
)
```
### Output Format

The compiled output will contain native Kubernetes manifests that can be directly applied to your cluster using `kubectl apply -f hello-world-pipeline-k8s.yaml`.

**Generated Resources:**
- **Pipeline**: Kubernetes Custom Resource (CRD) with API version `pipelines.kubeflow.org/v2beta1`
- **PipelineVersion**: Kubernetes Custom Resource (CRD) containing the actual pipeline specification

**Example Output Structure:**
```yaml
apiVersion: pipelines.kubeflow.org/v2beta1
kind: Pipeline
metadata:
  name: hello-world-pipeline
  namespace: kubeflow-user-example-com
spec:
  displayName: Hello World Pipeline
---
apiVersion: pipelines.kubeflow.org/v2beta1
kind: PipelineVersion
metadata:
  name: hello-world-pipeline-v1
  namespace: kubeflow-user-example-com
spec:
  displayName: Hello World Pipeline v1
  pipelineName: hello-world-pipeline
  pipelineSpec:
    # Pipeline specification details
  platformSpec:
    # Platform specification details
```

### Important Notes

- **Namespace**: Ensure the namespace in `KubernetesManifestOptions` matches where Kubeflow Pipelines is deployed
- **API Version**: The generated manifests use `pipelines.kubeflow.org/v2beta1` API version
- **Kubernetes Native Mode**: This feature requires Kubeflow Pipelines to be deployed in Kubernetes Native API mode (version 2.14.0+)

[pipeline-spec]: https://github.com/kubeflow/pipelines/blob/master/api/v2alpha1/pipeline_spec.proto#L50
[ir-yaml]: ../../concepts/ir-yaml.md
[compiled-output-example]: https://github.com/kubeflow/pipelines/blob/984d8a039d2ff105ca6b21ab26be057b9552b51d/sdk/python/test_data/pipelines/two_step_pipeline.yaml
[components-example]: https://github.com/kubeflow/pipelines/blob/984d8a039d2ff105ca6b21ab26be057b9552b51d/sdk/python/test_data/pipelines/two_step_pipeline.yaml#L1-L21
[data-types]: ../data-handling/data-types.md
[compiler-compile]: https://kubeflow-pipelines.readthedocs.io/en/latest/source/compiler.html#kfp.compiler.Compiler.compile
