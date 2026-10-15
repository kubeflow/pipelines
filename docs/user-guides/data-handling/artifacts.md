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

### Artifact properties

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

The **Pythonic artifact** authoring syntax provides an alternative artifact I/O syntax that is familiar to Python developers. The Pythonic artifact authoring syntax is supported for [Python Components][python-components] only. This syntax is not supported for [Container Components][container-components]. It is currently only supported at runtime by the Google Cloud Vertex Pipelines backend.

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

:::{warning} Not yet supported
The Pythonic artifact authoring syntax is not yet supported by the KFP orchestration backend, but may be supported by other orchestration backends.
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
| [`HTML`][dsl-html]                        | system.HTML                        |
| [`Markdown`][dsl-markdown]                    | system.Markdown                    |


`Artifact`, `Dataset`, `Model`, and `Metrics` are the most generic and commonly used artifact types. `Artifact` is the default artifact base type and should be used in cases where the artifact type does not fit neatly into another artifact category. `Artifact` is also compatible with all other artifact types. In this sense, the `Artifact` type is also an artifact "any" type.

On the [KFP open source][oss-be] UI, `ClassificationMetrics`, `SlicedClassificationMetrics`, `HTML`, and `Markdown` provide special UI rendering to make the contents of the artifact easily observable.

### Dataset and Model previews

When deployment administrators enable automatic previews, supported Python
components include a small, versioned
`preview.json` **file** in Dataset and Model output directories. Select that same artifact
in the run graph and open **Visualization** to view it. On the artifact's own
details page, select **Show artifact visualization** in Overview.

The preview is a static snapshot, but the UI loads it lazily, only when opened.
It does not download the dataset or model, inspect Parquet files, or deserialize
models. Preview generation runs inside the existing producer component; it
needs no additional container, service, or preview-generating pipeline task.

Previews currently support directory artifacts on S3/MinIO-compatible and GCS
stores. They use the existing artifact access and namespace isolation policies.
HTTP URLs, PVC-only artifacts, OCI images, archives, and existing single-file
outputs do not support this preview layout. Existing artifacts without a
preview continue to work normally. Do not silently turn a single-file output
into a directory: its consumers must know which data/model file to read.

#### Automatic previews: deployment opt-in

Automatic generation is **off by default**. Once your deployment enables it,
keep writing the native data/model files normally: the SDK generates
`preview.json` and its metadata
descriptor inside that component. No adapter call or hand-written JSON is
required, and no component or pipeline opt-in is needed.

```python
from kfp import dsl


@dsl.component(
    packages_to_install=['pyarrow==21.0.0'],
)
def create_dataset(dataset: dsl.Output[dsl.Dataset]):
    from pathlib import Path

    import pyarrow as pa
    import pyarrow.parquet as pq

    directory = Path(dataset.path)
    directory.mkdir(parents=True, exist_ok=True)
    pq.write_table(pa.table({'distance': [1.25, 2.5]}), directory / 'data.parquet')
```

Dataset previews include up to ten rows and twenty columns by default.
The optional `artifact_preview_rows=False` setting
omits dataset values while retaining available schema and dimensions; it does not
disable model previews. For sensitive outputs, a component can disable automatic
generation with `artifact_previews=False`.

Deployment administrators enable automatic generation for supported components
by setting `artifactPreviewsEnabled: "true"` in the deployment's `kfp-launcher`
ConfigMap. A missing or false deployment setting overrides `artifact_previews=True`
and prevents framework hooks, dataset sampling, and automatic preview publication.
In multi-user installations,
the profile controller propagates the setting to execution namespaces; see the
[operator rollout instructions](../../operator-guides/server-config.md#automatic-artifact-preview-generation).
Invalid or unreadable configuration also disables automatic previews with a
warning, without failing the component. The component flag cannot enable
generation in a deployment that has not opted in.

Automatic Dataset adapters require exactly one supported data file directly
inside the artifact directory; partitioned datasets, archives, and symlinks are
not sampled. Pandas and Polars native writers are supported without
preview-specific calls:

* **Parquet Dataset directories:** exactly one directly contained `*.parquet`
  file. PyArrow reads its footer and one projected batch, not the full table.
  Schema-only mode does not read row data. Partitioned directories, duplicate
  column names, symlinked files, and ambiguous projections are not sampled.
  Binary and nested cells are explicitly marked as omitted.
* **CSV:** UTF-8, comma-delimited files with a header. Quoting and multiline
  cells are preserved. Cells are displayed as strings rather than inventing
  original dataframe dtypes.
* **JSON/JSONL:** UTF-8 JSON records arrays, newline-delimited records
  (`.jsonl` or `.ndjson`), and Pandas column/split orientations (`.json`).
  JSON types are inferred from the bounded sample and labeled accordingly;
  nested values are omitted. Text parsing reads at most a 1 MiB source prefix.
  Columns/split JSON must fit within that source limit; use records or JSONL
  for larger outputs. Incomplete records are not displayed.
* **Arrow/Feather:** Arrow IPC files/streams (`.arrow` or `.ipc`) and Feather v2
  (`.feather`), using PyArrow. Only the selected columns and enough record
  batches for the sample are decoded; schema-only mode decodes no batches.
  IPC readers may decompress whole projected batches, so the ten-row
  **display** limit is not a bound on batch-decompression memory. Use
  appropriately sized producer record batches.

The adapters report total dimensions only when known without scanning the
whole dataset. JSON records can introduce later columns, so the preview does
not claim the sample's width is the total width. Schema-only JSON still
examines a bounded record prefix to discover columns/types, but publishes no
row values.

Automatic Model adapters support:

* **XGBoost Model directories:** save one native model under the declared
  output directory using the normal `Booster.save_model` or sklearn estimator
  `save_model` method. During execution, scoped SDK hooks capture
  the live model's configuration after a successful save; the original methods
  are restored afterwards. No model reload, pickle loading, or tree/weight dump
  is used. The snapshot reflects configuration at save time, not subsequent
  changes to the estimator. Saving several model files in one artifact, or
  replacing the saved file afterwards, leaves the artifact without an automatic
  preview rather than attaching an ambiguous or stale summary.
* **PyTorch Model directories:** use `torch.save` with a path to save an
  `nn.Module`, a state dictionary, or a checkpoint containing `state_dict` or
  `model_state_dict`. Native `torch.jit.save` and ScriptModule `save` are also
  supported. Scoped hooks record tensor names, shapes, dtypes, and counts,
  **never tensor values**. Module previews can count trainable parameters;
  state-dictionary element counts include buffers and are not labeled as
  parameter counts. A state dictionary does not retain the original class,
  architecture settings, or training hyperparameters. Existing checkpoints
  are not deserialized to infer these details.
* **Hugging Face Transformers Model directories:** native `save_pretrained`
  bundles, including sharded safetensors, use their saved `config.json`.
  The SDK reads at most 64 KiB of this configuration and displays its recorded
  framework version, architecture, and configuration. Weights are never
  opened or validated. This configuration is not a complete training record.

Scientific libraries remain optional dependencies of the **component image**,
not the core SDK or UI. Native save errors still fail normally; optional preview
failures are logged without failing successful artifact production. Existing
preview descriptors and files are not overwritten. Other output formats remain
usable without a preview.

Use an SDK supporting automatic previews inside the component image. Existing
compiled Python components also get automatic previews when they execute with
this SDK in an enabled deployment; images pinned to older SDKs do not.
Use a supporting SDK when compiling
component opt-out or schema-only settings, which are preserved in the compiled
executor arguments, including when the component is saved and reloaded.

The deployment switch is read from the existing read-only launcher configuration
mount before installing hooks and again before publishing previews. It does not
require recompiling pipelines. Kubernetes configuration updates and multi-user
reconciliation take time to reach component pods. Already-produced, cached, and
manually written previews are not removed or hidden by disabling automatic
generation, and older artifacts are not backfilled. Automatic generation does
not change payload serialization or convert single-file artifacts into directories.

#### Artifact lists

For `dsl.ParallelFor` producers followed by `dsl.Collected`, each producing task
generates a preview for its own artifact. Collection groups the artifact
references; it does not combine their data or create a new preview task.
The Visualization tab lets you select a member and fetches only that member's
preview. Its identity, limits, and access checks remain independent of the
other members.

The [example](../../../samples/core/artifact_previews/artifact_previews.py)
includes a pipeline returning collected Dataset and Model lists. This does not
add support for a single primitive Python component emitting artifact lists,
which the SDK currently rejects.

#### Interoperability contract

Other producers, container components, and imported artifacts can implement the
same file contract themselves. This is not needed for the automatic adapters.
Write the data/model files and a UTF-8 JSON preview under the artifact directory,
then advertise it in the artifact's metadata:

```python
artifact.metadata['_kfp_preview'] = {
    'version': 1,
    'path': 'preview.json',
}
```

The path must be the literal `preview.json`; external URLs and other paths are
not accepted. The preview is part of the existing artifact, not a separate HTML
or metadata artifact. The normal launcher upload copies the directory's files.
Manually produced and imported directory artifacts can use the same contract.
Keep published bundles immutable so the snapshot stays associated with the
same data/model version.

Before setting the metadata, serialize with `json.dumps(..., allow_nan=False)`,
verify the encoded content fits **64 KiB**, and atomically rename a temporary
file to `preview.json`. Only the producer decides what to include: the UI does
not create or repair previews. Missing files, unsupported versions, invalid
payloads, and access failures produce a preview error without breaking artifact
information or downloads.

#### Dataset format

Dataset previews display at most **10 rows and 20 columns**, preserving the
producer's row/column order. There is no full-dataset pagination or implicit
sorting. For example:

```json
{
  "version": 1,
  "kind": "dataset",
  "format": "parquet",
  "columns": [
    {"name": "distance", "dtype": "float64"},
    {"name": "category", "dtype": "string"}
  ],
  "rows": [[1.25, "A"], [null, "B"]],
  "total_rows": 100,
  "total_columns": 2,
  "sample": {
    "method": "head",
    "rows_truncated": true,
    "columns_truncated": false,
    "values_truncated": false,
    "redacted": false,
    "rows_omitted": false
  }
}
```

Each column requires a string `name` and `dtype`; duplicate names are allowed
because rows are positional. Each row must have exactly one cell per preview
column. `format` is descriptive, not an instruction to load another file.
`total_rows` and `total_columns` are optional: omit them or use `null` when
unknown, rather than treating the sample size as a total. Known counts must be
nonnegative safe integers and cannot be smaller than the displayed sample.

The `sample` object requires `method`, `rows_truncated`, `columns_truncated`,
and `values_truncated`. Mark incomplete row/column samples explicitly.
`redacted` and `rows_omitted` default to `false` when absent. To deliberately
publish only the schema, use `"rows": []` and `"rows_omitted": true`; an empty
dataset instead has `"total_rows": 0`. The SDK sets these fields automatically
when `artifact_preview_rows=False`.

Cells accept `null`, booleans, finite numbers, and strings. Strings and labels
are limited to **256 Unicode code points**. Convert timestamps, decimals,
integers outside JavaScript's safe integer range, and non-finite numbers to
explicit strings rather than losing precision or emitting nonstandard JSON.
Format complex cells as bounded strings. Shorten or omit values before
publication and set the corresponding indicators; oversized payloads are
rejected, not silently clipped into partial JSON.

For Parquet, the producer can reuse its in-memory data, or read the schema and
row count from Parquet metadata and project the chosen columns into the first
bounded batch. Avoid loading the full file just to build its preview. Other
formats, including Pandas/Polars/Arrow outputs, can emit the same JSON; no
format-specific library is installed in the UI.

#### Model format

Model previews separate recorded **training hyperparameters** from structural
facts. They are not a learned-weight or individual-tree viewer:

```json
{
  "version": 1,
  "kind": "model",
  "framework": {"name": "xgboost", "version": "3.0.5"},
  "estimator_class": "xgboost.XGBClassifier",
  "hyperparameters": {
    "objective": "binary:logistic",
    "learning_rate": 0.1,
    "max_depth": 2,
    "n_estimators": 5
  },
  "structure": {
    "feature_count": 24,
    "boosting_rounds": 5,
    "tree_count": 5
  },
  "notes": ["Training configuration recorded by the producer."]
}
```

Framework/version, estimator class, sections, and notes are optional. Omitted
information is shown as not recorded; native model IO does not necessarily
retain every training setting. The automatic XGBoost adapter captures current
estimator parameters and structural facts while the live model is available.
Unset estimator parameters are omitted rather than filled with inferred
defaults. Fit-time arguments that the estimator does not retain are not recoverable.
If an estimator was loaded from a saved model, its current parameters are not
necessarily the original training configuration. Display values may be
shortened or converted to strings, so the preview is not an executable set of
parameters for reconstructing the model.

The two sections may contain at most **100 combined entries**. Values are
JSON scalars or bounded objects/arrays, with at most four nested containers,
100 entries/items per nested collection, and 256 code points per string/key.
There may be at most ten notes. The entire file still has the 64 KiB limit.
Only escaped text is rendered; the server does not run Python or model code.

#### Producer example and privacy

The [examples](../../../samples/core/artifact_previews/artifact_previews.py)
produce real Parquet/XGBoost files, Pandas/Polars dataframes in multiple formats,
PyTorch modules/state dictionaries/TorchScript, and a tiny locally initialized
Transformer. They download no pretrained models. In an enabled deployment, the SDK
creates previews without component-written preview JSON or adapter calls.
The format-matrix pipeline also collects dataframe and PyTorch artifact lists.
Add `artifact_preview_rows=False` for a schema-only dataset preview.
Its [requirements](../../../samples/core/artifact_previews/requirements.txt)
are producer/example dependencies, not additions to the UI or core SDK.

From the repository root, compile or exercise the example locally:

```bash
uv run python samples/core/artifact_previews/artifact_previews.py
HF_HUB_OFFLINE=1 uv run --with pytest \
  --extra-index-url https://download.pytorch.org/whl/cpu \
  --with-requirements samples/core/artifact_previews/requirements.txt \
  pytest -v samples/core/artifact_previews/artifact_previews_unit_test.py
```

Tests explicitly simulate an enabled deployment; local execution without the
deployment policy does not generate previews. The example pins CPU-only PyTorch
on Linux and includes its official package index.

Row samples and model settings may contain sensitive data. Review this before
opting in deployment-wide, and set `artifact_previews=False` on individual
components whose outputs are not safe to expose. Use
`artifact_preview_rows=False` when dataset row values must not be published.
Automatic sampling uses the first columns; it does not detect or redact
sensitive fields. For custom previews, select permitted columns and redact
values **before writing the preview**. UI hiding and lazy loading are not
access controls. Users who can
download a directory artifact can generally obtain its preview file too.
Preview responses use `Cache-Control: no-store`; ordinary metadata/list
responses contain only the descriptor, not row samples.

The existing UI service resolves the stored parent artifact, applies its viewer
and namespace ownership policies, and reads only the fixed child file with a
byte limit and cancellation. Existing namespace artifact proxies must also
support this endpoint; upgrade their existing images with the UI rather than
adding containers or bypassing isolation. Strict artifact-only ownership mode
does not require registering the preview file as another artifact.


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
[oss-be]: ../../operator-guides/installation.md
[pipelines]: ../components/compose-components-into-pipelines.md
[container-components]: ../components/container-components.md
[python-components]: ../components/lightweight-python-components.md
[dsl-parallelfor]: https://kubeflow-pipelines.readthedocs.io/en/latest/source/dsl.html#kfp.dsl.ParallelFor
[dsl-collected]: https://kubeflow-pipelines.readthedocs.io/en/latest/source/dsl.html#kfp.dsl.Collected
[parallel-looping]: ../core-functions/control-flow.md#dslparallelfor
[kubernetes-workspace]: #kubernetes-specific-workspace-settings
[traditional-artifact-syntax]: #traditional-artifact-syntax
[multiple-outputs]: parameters.md#multiple-output-parameters
[pythonic-artifact-syntax]: #new-pythonic-artifact-syntax
