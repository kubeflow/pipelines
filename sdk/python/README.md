
Kubeflow Pipelines is a platform for building and deploying portable, scalable machine learning workflows based on Docker containers within the [Kubeflow](https://www.kubeflow.org/) project.

Use Kubeflow Pipelines to compose a multi-step workflow ([pipeline](https://www.kubeflow.org/docs/components/pipelines/concepts/pipeline/)) as a [graph](https://www.kubeflow.org/docs/components/pipelines/concepts/graph/) of containerized [tasks](https://www.kubeflow.org/docs/components/pipelines/concepts/step/) using Python code and/or YAML. Then, [run](https://www.kubeflow.org/docs/components/pipelines/concepts/run/) your pipeline with specified pipeline arguments, rerun your pipeline with new arguments or data, [schedule](https://www.kubeflow.org/docs/components/pipelines/concepts/run-trigger/) your pipeline to run on a recurring basis, organize your runs into [experiments](https://www.kubeflow.org/docs/components/pipelines/concepts/experiment/), save machine learning artifacts to compliant [artifact registries](https://www.kubeflow.org/docs/components/pipelines/concepts/metadata/), and visualize it all through the [Kubeflow Dashboard](https://www.kubeflow.org/docs/components/central-dash/overview/).

## Installation

The KFP SDK requires Python 3.11 or later, including in component runtime images
that install the SDK. Generic container components that do not install the SDK
can use other runtimes.

To install `kfp`, run:

```sh
pip install kfp
```

The unified SDK includes `kfp.pipeline_spec`, `kfp.kubernetes`, and the generated
REST client at `kfp.server_api`. `kfp[kubernetes]` remains a compatibility extra;
Kubernetes support is included by default.

### Migrating from the split packages

For the first release containing consolidation, set `KFP_VERSION` to that
release's version and run this one-time cleanup **before** installing the SDK:

```sh
python -m pip uninstall -y kfp-pipeline-spec kfp-server-api kfp-kubernetes &&
python -m pip install --upgrade --force-reinstall "kfp==$KFP_VERSION"
```

Do not uninstall those packages after installing unified `kfp`: their old
installation records contain shared files and uninstalling them can remove
parts of the new SDK. If that has already happened, repeat the installation
command with `--force-reinstall`. Restart running notebooks/interpreters after
upgrading. Update direct imports of `kfp_server_api` to `kfp.server_api`;
`kfp.pipeline_spec` and `kfp.kubernetes` imports are unchanged. Dependencies that
explicitly require the retired distributions must also be migrated; use a fresh
environment if other applications still need the split SDK.

## Preview pipelines locally

`kfp visualize` reads Python (`.py`) or compiled KFP v2 specifications
(`.yaml`, `.yml`, `.json`), without a cluster or server credentials:

```sh
# Offline, interactive React Flow HTML (defaults to pipeline.html).
kfp visualize pipeline.py

# Select a function when a Python file defines multiple pipelines.
kfp visualize pipeline.py --function training -o training.html

# Mermaid diagrams in Markdown; the output extension selects the format.
kfp visualize pipeline.yaml -o pipeline.md
# Equivalent: kfp visualize pipeline.yaml --format markdown

# Regenerate HTML and print a local live-viewer URL.
kfp visualize pipeline.py --watch

# Regenerate Markdown for an editor or third-party Mermaid viewer.
kfp visualize pipeline.py --format markdown --watch
```

HTML exports include JavaScript, styles, data, and third-party licenses: they
work offline, including in HTML previewers that permit JavaScript. They contain
the same graph canvas, task/artifact cards, inline DAG expansion, controls, and
node inspector as the KFP pipeline UI. Markdown exports contain one Mermaid
task-dependency diagram per DAG and need a Mermaid-aware renderer. Both show
static dependencies, including parameter/artifact inputs and explicit ordering;
loops are not unrolled, and no runtime execution status is shown. Legacy Argo Workflow YAML and standalone component specs are
not supported. Compiler multi-document YAML and Kubernetes `PipelineVersion`
manifests are supported.

`-w` is a short alias for `--watch`. Watch mode observes **saved files**, not
unsaved editor buffers. Python inputs
are recompiled in a fresh process after changes to `.py` files beneath the
input's directory (excluding hidden directories, `venv`, `node_modules`, and
`__pycache__`). Imports outside that directory and non-Python configuration files
are not watched. YAML/JSON inputs watch only the input file. Compilation has a
60-second timeout. Failed updates preserve the last successful export and print
the error; the live HTML viewer also displays it and recovers on the next valid
change. Markdown viewers control their own reload behavior.

For live HTML updates, open the URL printed by `--watch`. It binds only to
`127.0.0.1` on an automatically selected port and serves only the viewer and its
graph data, not the workspace. Keep it private: the graph includes component
specifications. A separately opened `.html` file is an offline snapshot, not a
live connection. Remote environments require port forwarding to reach the
printed URL; retain its `127.0.0.1:<port>` Host and token path. Stop with Ctrl+C.

**Only compile trusted Python:** importing a pipeline executes its module-level
code, including arbitrary side effects. The visualizer does not itself submit
runs. Exported task/component specifications can contain source code or literal
credentials; review them before sharing.

## Getting started

The following is an example of a simple pipeline that uses the `kfp` v2 syntax:

```python
from kfp import dsl
import kfp


@dsl.component
def add(a: float, b: float) -> float:
    '''Calculates sum of two arguments'''
    return a + b


@dsl.pipeline(
    name='Addition pipeline',
    description='An example pipeline that performs addition calculations.')
def add_pipeline(
    a: float = 1.0,
    b: float = 7.0,
):
    first_add_task = add(a=a, b=4.0)
    second_add_task = add(a=first_add_task.output, b=b)


client = kfp.Client(host='<my-host-url>')
client.create_run_from_pipeline_func(
    add_pipeline, arguments={
        'a': 7.0,
        'b': 8.0
    })

```
