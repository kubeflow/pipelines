# Create, use, pass, and track ML artifacts

Most machine learning pipelines aim to create one or more machine learning artifacts, such as a model, dataset, evaluation metrics, etc.

KFP provides first-class support for creating machine learning artifacts via the [`dsl.Artifact`][dsl-artifact] class and other artifact subclasses. KFP maps these artifacts to their underlying [ML Metadata][ml-metadata] schema title, the canonical name for the artifact type.

In general, artifacts and their associated annotations serve several purposes:
* To provide logical groupings of component/pipeline input/output types
* To provide a convenient mechanism for writing to object storage via the task's local filesystem
* To enable [type checking][type-checking] of pipelines that create ML artifacts
* To make the contents of some artifact types easily observable via special UI rendering

The following `training_component` demonstrates usage of both input and output artifacts using the [traditional artifact syntax][traditional-artifact-syntax]:

```python
from kfp.dsl import Input, Output, Dataset, Model

@dsl.component
def training_component(dataset: Input[Dataset], model: Output[Model]):
    """Trains an output Model on an input Dataset."""
    with open(dataset.path) as f:
        contents = f.read()

    # ... train tf_model model on contents of dataset ...

    tf_model.save(model.path)
    model.metadata['framework'] = 'tensorflow'
```

This `training_component` does the following:
1. Accepts an input dataset and declares an output model
2. Reads the input dataset's content from the local filesystem
3. Trains a model (omitted)
4. Saves the model as a component output
5. Sets some metadata about the saved model

As illustrated by `training_component`, artifacts are simply a thin wrapper around some artifact properties, including the `.path` from which the artifact can be read/written and the artifact's `.metadata`. The following sections describe these properties and other aspects of artifacts in detail.

## Artifact properties

To use create and consume artifacts from components, you'll use the available properties on [artifact instances](#artifact-types). Artifacts feature four properties:
* `name`, the name of the artifact (cannot be overwritten on Vertex Pipelines).
* `.uri`, the location of your artifact object. For input artifacts, this is where the object resides currently. For output artifacts, this is where you will write the artifact from within your component.
* `.metadata`, additional key-value pairs about the artifact.
* `.path`, a local path that corresponds to the artifact's `.uri`.

The artifact `.path` attribute is particularly helpful. When you write the contents of your artifact to the location provided by the artifact's `.path` attribute, the pipelines backend will handle copying the file at `.path` to the URI at `.uri` automatically, allowing you to create artifact files within a component by only interacting with the task's local filesystem.

As you will see more in the other examples in this section, each of these properties are accessible on artifacts inside components:

```python
from kfp import dsl
from kfp.dsl import Dataset
from kfp.dsl import Input

@dsl.component
def print_artifact_properties(dataset: Input[Dataset]):
    with open(dataset.path) as f:
        lines = f.readlines()

    print('Information about the artifact')
    print('Name:', dataset.name)
    print('URI:', dataset.uri)
    print('Path:', dataset.path)
    print('Metadata:', dataset.metadata)

    return len(lines)
```

Note that input artifacts should be treated as immutable. You should not try to modify the contents of the file at `.path` and any changes to the artifact's properties will not affect the artifact's metadata in [ML Metadata][ml-metadata].

### Artifacts in components

The KFP SDK supports two forms of artifact authoring syntax for components: traditional and Pythonic.

The **traditional artifact** authoring syntax is the original artifact authoring style provided by the KFP SDK. The traditional artifact authoring syntax is supported for both [Python Components][python-components] and [Container Components][container-components]. It is supported at runtime by the open source KFP backend and the Google Cloud Vertex Pipelines backend.

The **Pythonic artifact** authoring syntax provides an alterative artifact I/O syntax that is familiar to Python developers. The Pythonic artifact authoring syntax is supported for [Python Components][python-components] only. This syntax is not supported for [Container Components][container-components]. It is currently only supported at runtime by the Google Cloud Vertex Pipelines backend.

#### Traditional artifact syntax

When using the traditional artifact authoring syntax, all artifacts are provided to the component function as an input wrapped in an `Input` or `Output` type marker.

```python
def my_component(in_artifact: Input[Artifact], out_artifact: Output[Artifact]):
    ...
```

For _input artifacts_, you can read the artifact using its `.uri` or `.path` attribute.

For _output artifacts_, a pre-constructed output artifact will be passed into the component. You can update the output artifact's [properties](#artifact-properties) in place and write the artifact's contents to the artifact's `.path` or `.uri` attribute. You should not return the artifact instance from your component. For example:

```python
from kfp import dsl
from kfp.dsl import Dataset, Input, Model, Output

@dsl.component
def train_model(dataset: Input[Dataset], model: Output[Model]):
    with open(dataset.path) as f:
        dataset_lines = f.readlines()

    # train a model
    trained_model = ...

    trained_model.save(model.path)
    model.metadata['samples'] = len(dataset_lines)
```

#### **New** Pythonic artifact syntax

To use the Pythonic artifact authoring syntax, simply annotate your components with the artifact class as you would when writing normal Python.

```python
def my_component(in_artifact: Artifact) -> Artifact:
    ...
```

Inside the body of your component, you can read artifacts passed in as input (no change from the traditional artifact authoring syntax). For artifact outputs, you'll construct the artifact in your component code, then return the artifact as an output. For example:

```python
from kfp import dsl
from kfp.dsl import Dataset, Model

@dsl.component
def train_model(dataset: Dataset) -> Model:
    with open(dataset.path) as f:
        dataset_lines = f.readlines()

    # train a model
    trained_model = ...

    model_artifact = Model(uri=dsl.get_uri(), metadata={'samples': len(dataset_lines)})
    trained_model.save(model_artifact.path)

    return model_artifact
```

For a typical output artifact which is written to one or more files, the `dsl.get_uri` function can be used at runtime to obtain a unique object storage URI that corresponds to the current task. The optional `suffix` parameter is useful for avoiding path collisions when your component has multiple artifact outputs.

Multiple output artifacts should be specified similarly to [multiple output parameters][multiple-outputs]:

```python
from kfp import dsl
from kfp.dsl import Dataset, Model
from typing import NamedTuple

@dsl.component
def train_multiple_models(
    dataset: Dataset,
) -> NamedTuple('outputs', model1=Model, model2=Model):
    with open(dataset.path) as f:
        dataset_lines = f.readlines()

    # train a model
    trained_model1 = ...
    trained_model2 = ...

    model_artifact1 = Model(uri=dsl.get_uri(suffix='model1'), metadata={'samples': len(dataset_lines)})
    trained_model1.save(model_artifact1.path)

    model_artifact2 = Model(uri=dsl.get_uri(suffix='model2'), metadata={'samples': len(dataset_lines)})
    trained_model2.save(model_artifact2.path)

    outputs = NamedTuple('outputs', model1=Model, model2=Model)
    return outputs(model1=model_artifact1, model2=model_artifact2)
```

:::{warning}
**Not yet supported:** The Pythonic artifact authoring syntax is not yet supported by the KFP orchestration backend, but may be supported by other orchestration backends.
:::

### Embedded artifacts
Embedded artifacts allow Python-function components to embed files or directories directly into a lightweight component.
At compile time, the file/dir is archived (tar + gzip) and base64-embedded into the ephemeral module.
At runtime, the embedded artifact is made available by extracting to a temporary location, and the EmbeddedInput[...] parameter is injected as an artifact with .path pointing to the extracted file/dir in a temporary directory. The extracted root is also added to sys.path for Python module resolution if it's a directory.
This feature is similar to notebook components in that it allows users to embed files into a lightweight component.

```python
@dsl.component(embedded_artifact_path=tmpdir_path)
def read_embedded_artifact_dir(artifact: dsl.EmbeddedInput[dsl.Dataset]):
    import os

    with open(os.path.join(artifact.path, "log.txt"), "r", encoding="utf-8") as f:
        log = f.read()

    return log
```

### Artifacts in pipelines

Irrespective of whether your components use the Pythonic or traditional artifact authoring syntax, pipelines that use artifacts should be annotated with the [Pythonic artifact syntax][pythonic-artifact-syntax]:

```python
def my_pipeline(in_artifact: Artifact) -> Artifact:
    ...
```

See the following pipeline which accepts a `Dataset` as input and outputs a `Model`, surfaced from the inner component `train_model`:

```python
from kfp import dsl
from kfp.dsl import Dataset, Model

@dsl.pipeline
def augment_and_train(dataset: Dataset) -> Model:
    augment_task = augment_dataset(dataset=dataset)
    return train_model(dataset=augment_task.output).output
```

The [KFP SDK compiler][compiler] will type check artifact usage according to the rules described in [Type Checking][type-checking].

Please see [Pipeline Basics][pipelines] for comprehensive documentation on how to author a pipeline.


### Lists of artifacts

KFP supports input lists of artifacts, annotated as `List[Artifact]` or `Input[List[Artifact]]`. This is useful for collecting output artifacts from a loop of tasks using the [`dsl.ParallelFor`][dsl-parallelfor] and [`dsl.Collected`][dsl-collected] control flow objects.

Pipelines can also return an output list of artifacts by using a `-> List[Artifact]` return annotation and returning a [`dsl.Collected`][dsl-collected] instance.

Both consuming an input list of artifacts and returning an output list of artifacts from a pipeline are described in [Pipeline Control Flow: Parallel looping][parallel-looping]. Creating output lists of artifacts from a single-step component is not currently supported.

### Pipeline Run Workspaces

**⚠️ Version Requirement**: The Pipeline Run Workspace feature is available starting from Kubeflow Pipelines version 2.15.0.

While the traditional artifact authoring syntax is effective for passing smaller amounts of data between components at runtime,
many pipelines require passing larger amounts of data between components. Doing so with artifact inputs and outputs introduces overhead and requires additional S3 storage.
Pipeline Run Workspaces provide a way to pass data between components without this overhead - artifacts are stored in a PVC mounted on the component.

A Pipeline Run Workspace is configured with `dsl.WorkspaceConfig` through the `pipeline_config` parameter shown in the example below. `dsl.WorkspaceConfig` accepts the following parameters:
- `size` (required): String representation of workspace size, including units.
- `kubernetes` (optional): [`dsl.KubernetesWorkspaceConfig`][kubernetes-workspace]

#### Using the workspace mount path
When a pipeline is configured with a workspace, a PVC is automatically created and mounted into every task pod at a platform-defined mount path.
To pass the workspace mount path into a component, use `dsl.WORKSPACE_PATH_PLACEHOLDER`. At runtime, the backend replaces this placeholder with the actual mount path of the workspace PVC (e.g., `/kfp-workspace`). This allows components to read and write shared data on the workspace volume without hard-coding mount paths.


#### Kubernetes-specific Workspace Settings
Specific settings are configured as a dictionary through the `pvcSpecPatch` parameter shown in the example below.

```python
from kfp import dsl
from kfp import compiler


@dsl.component()
def clone_repo(workspacePath: str, repo: str) -> str:
    import os
    import subprocess

    clone_path = os.path.join(workspacePath, "repo")
    subprocess.call(["git", "clone", repo, clone_path])

    return clone_path


@dsl.component()
def process_data(repo_path: str):
    print("Processing the data at " + repo_path)


@dsl.component()
def train(model: dsl.Model, trained_model: dsl.Output[dsl.Model]):
    with open(model.path, "r") as model_file:
        print("Training the model")

    # Upload the model to S3 and register it in MLMD
    trained_model.set_path(model.path) # or trained_model.custom_path = model.path

@dsl.pipeline(
    name="my-pipeline",
    pipeline_config=dsl.PipelineConfig(
        workspace=dsl.WorkspaceConfig(
            size="250GB",
            kubernetes=dsl.KubernetesWorkspaceConfig(
                pvcSpecPatch={
                    "storageClassName": "super-fast-storage",
                    "accessModes": ["ReadWriteMany"],
                }
            ),
        ),
    )
def pipeline(repo: str, model_uri: str):
    clone_repo_task = clone_repo(
        workspacePath=dsl.WORKSPACE_PATH_PLACEHOLDER, repo=repo,
    )
    process_data_task = process_data(repo_path=clone_repo_task.output)
    import_base_model_task = dsl.importer(
        artifact_class=dsl.Model, artifact_uri=model_uri, download_to_workspace=True,
    )
    train(model=import_base_model_task.output).after(process_data_task)
```

### Artifact types

The artifact annotation indicates the type of the artifact. KFP provides several artifact types within the DSL:

| DSL object                    | Artifact schema title              |
| ----------------------------- | ---------------------------------- |
| [`Artifact`][dsl-artifact]                    | system.Artifact                    |
| [`Dataset`][dsl-dataset]                     | system.Dataset                     |
| [`Model`][dsl-model]                       | system.Model                       |
| [`Metrics`][dsl-metrics]                     | system.Metrics                     |
| [`ClassificationMetrics`][dsl-classificationmetrics]       | system.ClassificationMetrics       |
| [`SlicedClassificationMetrics`][dsl-slicedclassificationmetrics] | system.SlicedClassificationMetrics |
| [`HTML`][dsl-html]                        | system.HTML                        |****
| [`Markdown`][dsl-markdown]                    | system.Markdown                    |


`Artifact`, `Dataset`, `Model`, and `Metrics` are the most generic and commonly used artifact types. `Artifact` is the default artifact base type and should be used in cases where the artifact type does not fit neatly into another artifact category. `Artifact` is also compatible with all other artifact types. In this sense, the `Artifact` type is also an artifact "any" type.

On the [KFP open source][oss-be] UI, `ClassificationMetrics`, `SlicedClassificationMetrics`, `HTML`, and `Markdown` provide special UI rendering to make the contents of the artifact easily observable.


[ml-metadata]: https://github.com/google/ml-metadata
[compiler]: https://kubeflow-pipelines.readthedocs.io/en/latest/source/compiler.html#kfp.compiler.Compiler
[dsl-artifact]: https://kubeflow-pipelines.readthedocs.io/en/latest/source/dsl.html#kfp.dsl.Artifact
[dsl-dataset]: https://kubeflow-pipelines.readthedocs.io/en/latest/source/dsl.html#kfp.dsl.Dataset
[dsl-model]: https://kubeflow-pipelines.readthedocs.io/en/latest/source/dsl.html#kfp.dsl.Model
[dsl-metrics]: https://kubeflow-pipelines.readthedocs.io/en/latest/source/dsl.html#kfp.dsl.Metrics
[dsl-classificationmetrics]: https://kubeflow-pipelines.readthedocs.io/en/latest/source/dsl.html#kfp.dsl.ClassificationMetrics
[dsl-slicedclassificationmetrics]: https://kubeflow-pipelines.readthedocs.io/en/latest/source/dsl.html#kfp.dsl.SlicedClassificationMetrics
[dsl-html]: https://kubeflow-pipelines.readthedocs.io/en/latest/source/dsl.html#kfp.dsl.HTML
[dsl-markdown]: https://kubeflow-pipelines.readthedocs.io/en/latest/source/dsl.html#kfp.dsl.Markdown
[type-checking]: ../core-functions/compile-a-pipeline.md#type-checking
[oss-be]: ../../operator-guides/installation/index.md
[pipelines]: ../components/compose-components-into-pipelines.md
[container-components]: ../components/container-components.md
[python-components]: ../components/lightweight-python-components.md
[dsl-parallelfor]: https://kubeflow-pipelines.readthedocs.io/en/latest/source/dsl.html#kfp.dsl.ParallelFor
[dsl-collected]: https://kubeflow-pipelines.readthedocs.io/en/latest/source/dsl.html#kfp.dsl.Collected
[parallel-looping]: ../core-functions/control-flow.md#dslparallelfor
[kubernetes-workspace]: artifacts.md#kubernetes-specific-workspace-settings
[traditional-artifact-syntax]: artifacts.md#traditional-artifact-syntax
[multiple-outputs]: parameters.md#multiple-output-parameters
[pythonic-artifact-syntax]: artifacts.md#new-pythonic-artifact-syntax
