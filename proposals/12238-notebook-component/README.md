# KEP-12238: Jupyter Notebook Components

<!-- toc -->

- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Baseline Feature For Bundled Assets](#baseline-feature-for-bundled-assets)
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
it at runtime via `nbconvert`, with parameters injected as a prepended cell. This provides a simple path for
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

### Baseline Feature For Bundled Assets

This KEP establishes a baseline, generic capability for Python-function components to embed arbitrary files or
directories directly into a lightweight component. The goal is to support cases where a component needs a small amount
of read-only assets (configs, scripts, models, notebooks, etc.) without requiring a custom image.

- Add a new decorator argument to `@dsl.component`:
  - `bundled_artifact_path: Optional[str]` — path to a file or directory on the authoring machine to bundle with the
    component.
- Add a new SDK type: `dsl.BundledInput[T]` — a runtime-only input annotation that resolves to an artifact instance of
  type `T` (e.g. `dsl.Dataset`) whose `path` points to the extracted bundle root.
  - If a directory is bundled, `.path` points to the extracted directory.
  - If a single file is bundled, `.path` points to that file.

Example:

```python
@dsl.component(
    bundled_artifact_path="assets/config_dir",
)
def my_component(cfg: dsl.BundledInput[dsl.Dataset], param: int):
    # cfg.path is a directory when a directory is bundled; a file path if a file is bundled
    print(cfg.path)
```

Execution model (lightweight components):

- At compile time, the file/dir is archived (tar + gzip) and base64-embedded into the ephemeral module.
- At runtime, the bundle is safely extracted to a temporary location and the `BundledInput[...]` parameter is injected
  as an artifact with `.path` pointing to the extracted file/dir in a temporary directory.

Relationship to `@dsl.notebook_component`:

- The notebook component leverages the same embedding pattern, specializing the runtime helper to execute `.ipynb`
  content via `nbconvert`.

### SDK User Experience

#### Example

```python
from kfp import dsl


@dsl.notebook_component(
    notebook_path="train.ipynb",
    packages_to_install=["pandas", "scikit-learn", "nbconvert", "jupyter"],
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
from kfp.dsl import Output, OutputPath, Input, Model


@dsl.component(
    base_image="python:3.11-slim",
    packages_to_install=["pandas==2.2.2"],
)
def preprocess(text: str, cleaned_text: OutputPath(str)):
    """Cleans whitespace from input text and writes to cleaned_text."""
    import re

    cleaned = re.sub(r"\s+", " ", text).strip()
    with open(cleaned_text, "w") as f:
        f.write(cleaned)


@dsl.notebook_component(
    notebook_path="dev-files/nb_train.ipynb",
    base_image="registry.access.redhat.com/ubi9/python-311:latest",
)
def train_model(
    cleaned_text: str,
    model: Output[Model],
    summary: OutputPath(str),
):
    """Trains a model from cleaned text and writes model + summary outputs."""
    import os, shutil
    # Execute the embedded notebook with kwargs injected as variables
    dsl.run_notebook(cleaned_text=cleaned_text)

    # Translate notebook outputs into KFP outputs
    nb_model_dir = "/tmp/kfp_nb_outputs/model_dir"
    nb_summary_path = "/tmp/kfp_nb_outputs/train_summary.txt"

    os.makedirs(model.path, exist_ok=True)
    for root, _, files in os.walk(nb_model_dir):
        rel = os.path.relpath(root, nb_model_dir)
        dst_dir = os.path.join(model.path, rel) if rel != "." else model.path
        os.makedirs(dst_dir, exist_ok=True)
        for fn in files:
            shutil.copy2(os.path.join(root, fn), os.path.join(dst_dir, fn))

    with open(nb_summary_path, "r", encoding="utf-8") as f_in, open(summary, "w") as f_out:
        f_out.write(f_in.read())


@dsl.notebook_component(
    notebook_path="dev-files/nb_eval.ipynb",
    base_image="registry.access.redhat.com/ubi9/python-311:latest",
)
def evaluate_model(
    model: Input[Model],
    metrics_json: OutputPath(dict),
):
    """Evaluates a model and writes metrics JSON output."""
    import json
    # Execute the notebook with the model artifact path
    dsl.run_notebook(model=model.path)

    # Copy notebook-generated metrics into output parameter
    with open("/tmp/kfp_nb_outputs/metrics.json", "r") as f_in, open(metrics_json, "w") as f_out:
        import json
        json.dump(json.load(f_in), f_out)


@dsl.pipeline(name="three-step-nb-mix")
def pipeline(text: str = "Hello   world"):
    p = preprocess(text=text).set_caching_options(False)
    t = train_model(cleaned_text=p.outputs["cleaned_text"]).set_caching_options(False)
    evaluate_model(model=t.outputs["model"]).set_caching_options(False)
```

#### notebook_component Decorator Arguments

Only differences from the standard Python executor component:

- `notebook_path: str` – New parameter specifying the `.ipynb` file to embed and execute (required).
- `packages_to_install: Optional[List[str]]` – Same as Python executor except when `None` (default here) it additionally
  installs `jupyter>=1,<2` and `nbconvert>=7,<8` to avoid major-version bumps; `[]` installs nothing; a non-empty list
  installs exactly the provided packages.

All other decorator arguments and behaviors are identical to the Python executor.

#### Behavior Notes

- The notebook JSON is compressed using gzip and base64-encoded before embedding into the ephemeral Python module used
  by the Python executor. This reduces command line length and allows for larger notebooks.
- At runtime, `dsl.run_notebook(**kwargs)` is bound to a helper that:
  1. Decompresses and parses the embedded notebook into memory
  2. Injects a cell at the start of the notebook with the input parameters passed to `dsl.run_notebook`
  3. Executes via `nbconvert.ExecutePreprocessor`
  4. Streams cell outputs (stdout/stderr and `text/plain` displays)

For the baseline bundling feature:

- Files/directories are archived (tar+gz) and base64-embedded similarly; they are extracted at import time.
- `dsl.BundledInput[T]` is not part of the component interface; it is injected at runtime and provides an artifact with
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
- The implementation warns users when notebooks exceed 1MB in original size to help identify potential performance
  impacts.

### Risks and Mitigations

- **Dependency drift/conflicts**: Installing packages at runtime can introduce variability.
  - Mitigation: Encourage providing a `base_image` with pinned deps or using `packages_to_install` with exact versions.
- **Command length/performance**: Large embedded notebooks may slow compilation or exceed limits.
  - Mitigation: Automatic gzip compression reduces notebook size; warn on large files (>1MB original); recommend
    refactoring or pre-building images for very large notebooks.

## Design Details

### Security Considerations

This feature does not introduce additional security risks beyond those inherent to executing notebooks. It relies on the
additional Python packages of `jupyter` and `nbconvert` within the execution environment.

### Test Plan

#### Unit Tests

- Validate notebook JSON checks (well-formed, has `cells`).
- Verify `packages_to_install` behavior for `None`, `[]`, and non-empty lists.
- Ensure helper source is generated and injected correctly; binding of `dsl.run_notebook`.
- Test notebook compression and decompression round-trip correctness.
- Large-notebook warning logic (now at 1MB threshold).
- KFP local executing a pipeline with notebooks.

#### Integration tests

- Execute a minimal parameterized notebook that writes to an output artifact.
- Failure path: invalid notebook JSON; notebook cell raises execution error.
- Large notebook over 5 MB.

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
