# Use Caching

Kubeflow Pipelines support caching to eliminate redundant executions and improve
the efficiency of your pipeline runs. This page provides an overview of caching
in KFP and how to use it in your pipelines.

## Overview

Caching in KFP is a feature that allows you to cache the results of a component
execution and reuse them in subsequent runs. When caching is enabled for a
component, KFP will reuse the component's outputs if the component
is executed again with the same inputs and parameters (and the output is still
available).

Caching is particularly useful when you have components that take a long time to
execute or when you have components that are executed multiple times with the
same inputs and parameters.

If a task's results are retrieved from cache, its representation in the UI will
be marked with a green "arrow from cloud" icon.

## How to use caching

Caching is enabled by default for all components in KFP. You can disable caching
for a component by calling [`.set_caching_options(enable_caching=False)`](https://kubeflow-pipelines.readthedocs.io/en/latest/source/dsl.html#kfp.dsl.PipelineTask.set_caching_options) on a task object.

```python
from kfp import dsl

@dsl.component
def say_hello(name: str) -> str:
    hello_text = f'Hello, {name}!'
    print(hello_text)
    return hello_text

@dsl.pipeline
def hello_pipeline(recipient: str = 'World!') -> str:
    hello_task = say_hello(name=recipient)
    hello_task.set_caching_options(False)
    return hello_task.output
```

You can also enable or disable caching for all components in a pipeline by
setting the argument `caching` when submitting a pipeline for execution.
This will override the caching settings for all components in the pipeline.

```python
from kfp.client import Client

client = Client()
client.create_run_from_pipeline_func(
    hello_pipeline,
    enable_caching=True,  # overrides the above disabling of caching
)
```

The `--disable-execution-caching-by-default` flag disables caching for all pipeline tasks by default.

Example:

```
kfp dsl compile --py my_pipeline.py --output my_pipeline.yaml --disable-execution-caching-by-default
```

You can also set the default caching behavior using the `KFP_DISABLE_EXECUTION_CACHING_BY_DEFAULT` environment variable. When set to `true`, `1`, or other truthy values, it will disable execution caching by default for all pipelines. When set to `false` or when absent, the default of caching enabled remains.

Example:

```
KFP_DISABLE_EXECUTION_CACHING_BY_DEFAULT=true \
kfp dsl compile --py my_pipeline.py --output my_pipeline.yaml
```
This environment variable also works for `Compiler().compile()`.

Given the following pipeline file:
```
@dsl.pipeline(name='my-pipeline')
def my_pipeline():
    task_1 = create_dataset()
    task_2 = create_dataset()
    task_1.set_caching_options(False)

Compiler().compile(
    pipeline_func=my_pipeline,
    package_path='my_pipeline.yaml',

)
```
Executing the following:
```
KFP_DISABLE_EXECUTION_CACHING_BY_DEFAULT=true \
python my_pipeline.py
```
will result in `task_2` having caching disabled.

**NOTE**: Since Python initializes configurations during the import process, setting the `KFP_DISABLE_EXECUTION_CACHING_BY_DEFAULT` environment variable after importing pipeline components will not affect the caching behavior. Therefore, always set it before importing any Kubeflow Pipelines components.
