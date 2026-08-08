# Run a Pipeline

## Overview

Kubeflow Pipelines (KFP) provides several ways to trigger a pipeline run:

1. [__KFP Dashboard__](#run-pipeline---kfp-dashboard)
2. [__KFP SDK Client__](#run-pipeline---kfp-sdk-client)
3. [__KFP CLI__](#run-pipeline---kfp-cli)

:::{tip}
This guide only covers how to trigger an immediate pipeline run.
For more advanced scheduling options please see the "Recurring Runs" section of the dashboard and API.
:::

## Run Pipeline - KFP Dashboard

The first and easiest way to run a pipeline is by submitting it via the KFP dashboard.

To submit a pipeline for an immediate run:

1. [Compile a pipeline][compile-a-pipeline] to IR YAML.

2. From the "Pipelines" tab in the dashboard, select `+ Upload pipeline`:

   :::{image} ../../images/pipelines/submit-a-pipeline-on-dashboard.png
   :alt: Upload pipeline button
   :class: mt-3 mb-3 border rounded
   :::

3. Upload the pipeline `.yaml`, `.zip` or `.tar.gz` file, populate the upload form, then click `Create`.

   :::{image} ../../images/pipelines/upload-a-pipeline.png
   :alt: Upload pipeline screen
   :class: mt-3 mb-3 border rounded
   :::

4. From the "Runs" tab, select `+ Create run`:

   :::{image} ../../images/pipelines/create-run.png
   :alt: Create run button
   :class: mt-3 mb-3 border rounded
   :::

5. Select the pipeline you want to run, populate the run form, then click `Start`:

   :::{image} ../../images/pipelines/start-a-run.png
   :alt: Start a run screen
   :class: mt-3 mb-3 border rounded
   :::

## Run Pipeline - KFP SDK Client

You may also programmatically submit pipeline runs from the KFP SDK client.
The client supports two ways of submitting runs: _from IR YAML_ or _from a Python pipeline function_.

:::{note}
See the [Connect the SDK to the API](connect-api.md) guide for more information about creating a KFP client.
:::

For either approach, start by instantiating a [`kfp.Client()`][kfp-client]:

```python
import kfp

# TIP: you may need to authenticate with the KFP instance
kfp_client = kfp.Client()
```

To submit __IR YAML__ for execution use the [`.create_run_from_pipeline_package()`][kfp-client-create_run_from_pipeline_package] method:

```python
#from kfp import compiler, dsl
#
#@dsl.component
#def add(a: float, b: float) -> float:
#   return a + b
#
#@dsl.pipeline(name="Add two Numbers")
#def add_pipeline(a: float, b: float):
#   add_task = add(a=a, b=b)
#
#compiler.Compiler().compile(
#    add_pipeline,
#    package_path="./add-pipeline.yaml"
#)

kfp_client.create_run_from_pipeline_package(
    "./add-pipeline.yaml",
    arguments={
        "a": 1,
        "b": 2,
    }
)
```

To submit a python __pipeline function__ for execution use the [`.create_run_from_pipeline_func()`][kfp-client-create_run_from_pipeline_func] convenience method, which wraps compilation and run submission into one method:

```python
#from kfp import dsl
#
#@dsl.component
#def add(a: float, b: float) -> float:
#    return a + b
#
#@dsl.pipeline(name="Add two Numbers")
#def add_pipeline(a: float, b: float):
#    add_task = add(a=a, b=b)

kfp_client.create_run_from_pipeline_func(
    add_pipeline,
    arguments={
        "a": 1,
        "b": 2,
    }
)
```

## Run Pipeline - KFP CLI

The [`kfp run create`][kfp-cli-run-create] command allows you to submit a pipeline from the command line.

Here is the output of `kfp run create --help`:

```shell
kfp run create [OPTIONS] [ARGS]...
```

For example, the following command submits the `./path/to/pipeline.yaml` IR YAML to the KFP backend:

```shell
kfp run create \
  --experiment-name "my-experiment" \
  --package-file "./path/to/pipeline.yaml"
```

For more information about the `kfp` CLI, please see:

- [Use the CLI][use-the-cli]
- [CLI Reference][kfp-cli]

[compile-a-pipeline]: compile-a-pipeline.md
[use-the-cli]: cli.md
[kfp-cli]: https://kubeflow-pipelines.readthedocs.io/en/latest/source/cli.html
[kfp-cli-run-create]: https://kubeflow-pipelines.readthedocs.io/en/latest/source/cli.html#kfp-run-create
[kfp-client]: https://kubeflow-pipelines.readthedocs.io/en/latest/source/client.html#kfp.client.Client
[kfp-client-create_run_from_pipeline_func]: https://kubeflow-pipelines.readthedocs.io/en/latest/source/client.html#kfp.client.Client.create_run_from_pipeline_func
[kfp-client-create_run_from_pipeline_package]: https://kubeflow-pipelines.readthedocs.io/en/latest/source/client.html#kfp.client.Client.create_run_from_pipeline_package
