# Execute KFP pipelines locally

## Overview

KFP supports executing components and pipelines locally, enabling a tight development loop before running your code remotely.

Executing components and pipelines locally is easy. Simply initialize a local session using `local.init()`, then call the component or pipeline like a normal Python function.
KFP will log information about the execution.
Once execution completes, you can access the task outputs just as you would when composing a pipeline; the only difference is that the outputs are now materialized values, not references to future outputs.

## Limitations

Local execution is designed to help quickly *test* components and pipelines locally before testing in a remote environment.

Local execution comes with several limitations:

- Local execution does not feature optimizations and additional features such as caching, retry, etc. While these feature are important for production pipelines, they are less critical for a local testing environment. You will find that task methods like `.set_retry`, `.set_caching_options`, etc. have no effect locally.
- Local execution makes simple assumptions about the resources available on your machine. Local execution does not support specifying resource requests/limits/affinities related to memory, cores, accelerators, etc. You will find that task methods like `.set_memory_limit`, `.set_memory_request`, `.set_accelerator_type` etc. have no effect locally.
- Local execution doesn't support authentication mechanisms. If your component interacts with cloud resources or requires other privileged actions, you must test your pipeline in the cloud.
- While local pipeline execution has full support for sequential and nested pipelines, it does not yet support `dsl.Condition`, `dsl.ParallelFor`, or `dsl.ExitHandler`.

## Basic Example

In the following example, we use the `DockerRunner` type, the [runner types](#runner-types) are covered in more detail below.

```python
from kfp import local
from kfp import dsl

local.init(runner=local.DockerRunner())

@dsl.component
def add(a: int, b: int) -> int:
    return a + b

# run a single component
task = add(a=1, b=2)
assert task.output == 3

# or run it in a pipeline
@dsl.pipeline
def math_pipeline(x: int, y: int, z: int) -> int:
    t1 = add(a=x, b=y)
    t2 = add(a=t1.output, b=z)
    return t2.output

pipeline_task = math_pipeline(x=1, y=2, z=3)
assert pipeline_task.output == 6
```

Similarly, you can create artifacts and read the contents:
```python
from kfp import local
from kfp import dsl
from kfp.dsl import Output, Artifact
import json

local.init(runner=local.SubprocessRunner())

@dsl.component
def add(a: int, b: int, out_artifact: Output[Artifact]):
    import json

    result = json.dumps(a + b)

    with open(out_artifact.path, 'w') as f:
        f.write(result)

    out_artifact.metadata['operation'] = 'addition'


task = add(a=1, b=2)

# can read artifact contents
with open(task.outputs['out_artifact'].path) as f:
    contents = f.read()

assert json.loads(contents) == 3
assert task.outputs['out_artifact'].metadata['operation'] == 'addition'
```

By default, KFP will raise an exception if your component exits with a failure status. You can toggle this behavior using `raise_on_error`. You can also specify a new local "pipeline root" using `pipeline_root`. This is the local directory to which component outputs, including artifacts, are written.

```python
local.init(runner=...,
           raise_on_error=False,
           pipeline_root='~/my/component/outputs')
```

## Runner Types

Kubeflow pipelines has two local runners that you can use to execute your components and pipelines locally: `DockerRunner` and `SubprocessRunner`.

__We strongly recommended using `DockerRunner` whenever possible.__

### **Runner:** DockerRunner

The `DockerRunner` requires [Docker to be installed](https://docs.docker.com/engine/install/), but requires essentially no knowledge of Docker to use.

For example, to use the `DockerRunner`:

```python
from kfp import local

local.init(runner=local.DockerRunner())
```

Since the local `DockerRunner` executes each task in a separate container, the `DockerRunner`:

- Offers the strongest form of local runtime environment isolation
- Is most faithful to the remote runtime environment
- Allows execution of all component types: [Lightweight Python Component][lightweight-python-component], [Containerized Python Components][containerized-python-components], and [Container Components][container-components]

When you use the `DockerRunner`, KFP mounts your local pipeline root to the container to write outputs outside of the container.
This means that your component outputs will still be available for inspection even after the container exits.

### **Runner:** SubprocessRunner

The `SubprocessRunner` is only recommended where Docker cannot be installed, such as in some notebook environments.

For example, to use the `SubprocessRunner`:

```python
from kfp import local

local.init(runner=local.SubprocessRunner())
```

Since `SubprocessRunner` runs your code in a subprocess, the `SubprocessRunner`:
- Offers less local runtime environment isolation than the `DockerRunner`
- Does not support custom images or easily support tasks with complex environment dependencies
- Only allows execution of [Lightweight Python Component][lightweight-python-component]

:::{tip}
By default, the `SubprocessRunner` will install your dependencies into a virtual environment.

This is recommended, but can be disabled by setting `use_venv=False`:

```python
from kfp import local

local.init(runner=local.SubprocessRunner(use_venv=False))
```
:::

[lightweight-python-component]: ../components/lightweight-python-components.md
[containerized-python-components]: ../components/containerized-python-components.md
[container-components]: ../components/container-components.md
