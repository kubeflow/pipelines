# KEP-12238: Jupyter Notebook Components

<!-- toc -->

- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Baseline Feature For Embedded Assets](#baseline-feature-for-embedded-assets)
  - [SDK User Experience](#sdk-user-experience)
    - [Example](#example)
    - [notebook_component Decorator Arguments](#notebook_component-decorator-arguments)
    - [Behavior Notes](#behavior-notes)
  - [User Stories](#user-stories)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Security Considerations](#security-considerations)
  - [Test Plan](#test-plan)
    - [Unit Tests](#unit-tests)
    - [Integration tests](#integration-tests)
  - [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
<!-- /toc -->

## Summary

This proposal introduces a first-class `@dsl.notebook_component` decorator that lets users build Kubeflow Pipelines
(KFP) components directly from Jupyter notebooks. The decorator embeds a `.ipynb` file into the component and executes
it at runtime via `nbclient`, with parameters injected as a prepended cell. This provides a simple path for
notebook-centric workflows to run in KFP without requiring separate packaging or bespoke wrappers.

## Motivation

Many users begin experimentation and development in Jupyter notebooks. Turning those notebooks into pipeline components
currently requires boilerplate: exporting to Python, writing a wrapper function, or managing custom container images. A
native notebook component:

- Reduces friction to productionize notebook code in pipelines
- Preserves the notebook as the source of truth while allowing parameterization
- Avoids extra build steps by embedding notebook content into the component

### Goals

1. Enable defining a component from a `.ipynb` notebook with a single decorator.
2. Support parameter injection into the notebook at execution time.
3. Use the existing Python executor to keep compatibility with existing inputs/outputs concepts.

### Non-Goals

1. Notebook validation or linting beyond JSON and structural checks.
2. Adding a new pipeline IR or backend executor. This reuses the Python executor.

## Proposal

### Baseline Feature For Embedded Assets

This KEP establishes a baseline, generic capability for Python-function components to embed arbitrary files or
directories directly into a lightweight component. The goal is to support cases where a component needs a small amount
of read-only assets (configs, scripts, models, notebooks, etc.) without requiring a custom image.

- Add a new decorator argument to `@dsl.component`:
  - `embedded_artifact_path: Optional[str]` — path to a file or directory on the authoring machine to embed within the
    component.
- Add a new SDK type: `dsl.EmbeddedInput[T]` — a runtime-only input annotation that resolves to an artifact instance of
  type `T` (e.g. `dsl.EmbeddedInput[dsl.Dataset]`) whose `path` points to the extracted embedded artifact root.
  - If a directory is embedded, `.path` points to the extracted directory.
  - If a single file is embedded, `.path` points to that file.

Example:

```python
@dsl.component(
    embedded_artifact_path="assets/config_dir",
)
def my_component(cfg: dsl.EmbeddedInput[dsl.Dataset], param: int):
    # cfg.path is a directory when a directory is embedded; a file path if a file is embedded
    print(cfg.path)
```

Execution model (lightweight components):

- At compile time, the file/dir is archived (tar + gzip) and base64-embedded into the ephemeral module.
- At runtime, the embedded artifact is made available by extracting to a temporary location and the `EmbeddedInput[...]`
  parameter is injected as an artifact with `.path` pointing to the extracted file/dir in a temporary directory. The
  extracted root is also added to `sys.path` for Python module resolution if it's a directory.
- `sys.path` precedence: the embedded path/zip is prepended to `sys.path` (before existing entries) to ensure
  deterministic use of embedded modules when names overlap.

Relationship to `@dsl.notebook_component`:

- The notebook component leverages the same embedding pattern, specializing the runtime helper to execute `.ipynb`
  content via `nbclient`. If `notebook_path` is a directory, the `.ipynb` file in the directory is executed to allow for
  utility functions to be embedded. Pipeline compilation fails if there is more than one `.ipynb` file.

### SDK User Experience

#### Example

```python
from kfp import dsl


@dsl.notebook_component(
    notebook_path="train.ipynb",
    packages_to_install=["pandas", "scikit-learn", "nbclient"],
)
def train_from_notebook(dataset_uri: str, model: dsl.Output[dsl.Model]):
    dsl.run_notebook(
        dataset_uri=dataset_uri,
        output_model_path=model.path,
    )


@dsl.pipeline(name="nb-pipeline")

def pipeline(dataset_uri: str = "s3://bucket/dataset.csv"):
    train_from_notebook(dataset_uri=dataset_uri)
```

#### Complex Example

A mixed pipeline with a Python preprocessor, a notebook training step, and a notebook evaluation step:

```python
from kfp import dsl


@dsl.component(
    base_image="python:3.11-slim",
    packages_to_install=["pandas==2.2.2"],
)
def preprocess(text: str, cleaned_text: dsl.Output[dsl.Dataset]):
    """Cleans whitespace from input text and writes to cleaned_text."""
    import re

    cleaned = re.sub(r"\s+", " ", text).strip()
    with open(cleaned_text.path, "w", encoding="utf-8") as f:
        f.write(cleaned)


@dsl.notebook_component(
    notebook_path="dev-files/nb_train.ipynb",
    base_image="registry.access.redhat.com/ubi9/python-311:latest",
)
def train_model(
    cleaned_text: dsl.Input[dsl.Dataset],
    model: dsl.Output[dsl.Model],
):
    """Trains a model from cleaned text and writes model."""
    import shutil

    # Read the dataset for the notebook since it expects a string
    with open(cleaned_text.path, "r", encoding="utf-8") as f:
        cleaned_text_str = f.read()

    # Execute the embedded notebook with kwargs injected as variables
    dsl.run_notebook(cleaned_text=cleaned_text_str)

    # Translate notebook outputs into KFP outputs
    nb_model_dir = "/tmp/kfp_nb_outputs/model_dir"
    shutil.copytree(nb_model_dir, model.path, dirs_exist_ok=True)


@dsl.notebook_component(
    notebook_path="dev-files/nb_eval.ipynb",
    base_image="registry.access.redhat.com/ubi9/python-311:latest",
)
def evaluate_model(
    model: dsl.Input[dsl.Model],
    metrics_json: dsl.Output[dsl.Metrics],
):
    """Evaluates a model and writes metrics JSON output."""
    import json
    # Execute the notebook with the model artifact path
    dsl.run_notebook(model=model.path)

    # Copy notebook-generated metrics into output parameter
    with open("/tmp/kfp_nb_outputs/metrics.json", "r", encoding="utf-8") as f:
        metrics_dict = json.load(f)

    for metric_name, metric_value in metrics_dict.items():
        if isinstance(metric_value, (int, float)):
            metrics_json.log_metric(metric_name, metric_value)


@dsl.pipeline(name="three-step-nb-mix")
def pipeline(text: str = "Hello   world"):
    p = preprocess(text=text).set_caching_options(False)
    t = train_model(cleaned_text=p.output).set_caching_options(False)
    evaluate_model(model=t.output).set_caching_options(False)
```

#### notebook_component Decorator Arguments

Only differences from the standard Python executor component:

- `notebook_path: str` – New parameter specifying the `.ipynb` file to embed and execute (required).
- `packages_to_install: Optional[List[str]]` – Same as Python executor except when `None` (default here) it installs a
  slimmer default runtime of `nbclient>=0.10,<1`; `[]` installs nothing; a non-empty list installs exactly the provided
  packages.

All other decorator arguments and behaviors are identical to the Python executor.

#### Behavior Notes

- The notebook JSON is compressed using gzip and base64-encoded before embedding into the ephemeral Python module used
  by the Python executor. This reduces command line length and allows for larger notebooks.
- At runtime, `dsl.run_notebook(**kwargs)` is bound to a helper that:
  1. Decompresses and parses the embedded notebook into memory
  2. Injects parameters following Papermill semantics:
     - If the notebook contains a code cell tagged with `parameters`, a new code cell tagged `injected-parameters` is
       inserted immediately after it to override defaults.
     - If no `parameters` cell exists, the `injected-parameters` cell is inserted at the top of the notebook.
  3. Executes via `nbclient.NotebookClient`
  4. Streams cell outputs (stdout/stderr and `text/plain` displays)

For the baseline bundling feature:

- Files/directories are archived (tar+gz) and base64-embedded similarly; by default they are extracted at import time to
  satisfy the `EmbeddedInput[...]` contract of providing a real filesystem `.path` and to support non-Python assets.
- For Python import resolution, the embedded path (extracted root or zip archive) is prepended into `sys.path` if it's a
  directory.
- `dsl.EmbeddedInput[T]` is not part of the component interface; it is injected at runtime and provides an artifact with
  `.path` set to the extracted file/dir.

### User Stories

1. As a data scientist, I can take an existing exploratory notebook and run it as a KFP component with parameters,
   without rewriting it into a Python script.
2. As a platform user, I can standardize execution images and dependency sources while still allowing teams to embed
   notebooks into components.

### Notes/Constraints/Caveats

- Embedded content increases the size of the generated command; extremely large notebooks may hit container argument
  length limits, though gzip compression typically reduces notebook size significantly.
- Notebooks must be valid JSON and include a `cells` array; otherwise creation fails with a clear error.
- The SDK warns when embedded artifacts or notebooks exceed 1MB to flag potential issues. The backend has a configurable
  maximum pipeline spec size; if exceeded, the error recommends moving content to a container image or object store.

### Risks and Mitigations

- **Dependency drift/conflicts**: Installing packages at runtime can introduce variability.
  - Mitigation: Encourage providing a `base_image` with pinned deps or using `packages_to_install` with exact versions.
- **Command length/performance**: Large embedded notebooks may slow compilation or exceed limits.
  - Mitigation: Automatic gzip compression reduces notebook size; warn on large files (>1MB original); recommend
    refactoring or pre-building images for very large notebooks.

## Design Details

### Security Considerations

This feature does not introduce additional security risks beyond those inherent to executing notebooks. It relies on the
`nbclient` package within the execution environment (installed automatically unless overridden).

### Test Plan

#### Unit Tests

- Verify `packages_to_install` behavior for `None`, `[]`, and non-empty lists.
- Ensure helper source is generated and injected correctly; binding of `dsl.run_notebook`.
- Test notebook compression and decompression round-trip correctness.
- Large-notebook warning logic.
- KFP local executing a pipeline with embedded artifacts.
- KFP local executing a pipeline with notebooks.

#### Integration tests

- Execute a pipele with two parameterized notebooks that writes to an output artifact.
  - One notebook should have no `parameters` cell.
  - The other notebook should have a `parameters` cell with some overrides.
- Failure path: invalid notebook JSON; notebook cell raises execution error.
- Large notebook over 1 MB.

### Graduation Criteria

N/A

## Implementation History

- Initial proposal: 2025-09-10

## Drawbacks

- Embedded notebooks can bloat the command payload and slow compilation/execution for large files, though gzip
  compression typically helps.
- Notebooks are less modular than Python modules for code reuse and testing.

## Alternatives

1. Use `@dsl.component` with manual `nbconvert` calls inside the function. This requires boilerplate and manual
   packaging of the notebook.
2. Pre-build a container image containing the notebook and its dependencies, then use `@dsl.container_component`. This
   improves reproducibility but increases operational overhead.
