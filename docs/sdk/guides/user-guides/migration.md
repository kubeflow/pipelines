# Migrate to Kubeflow Pipelines v2

## Overview

Kubeflow Pipelines V2 is a significant update to the Kubeflow Pipelines (KFP) platform.

The key features introduced by KFP V2 are:

- A more pythonic SDK - use decorators like ([`@dsl.pipeline`][dsl-pipeline], [`@dsl.component`][dsl-component], [`@dsl.container_component`][dsl-container-component])
- Decouple from Argo Workflows - compile pipelines to a generic [IR YAML][ir-yaml] rather than Argo `Workflow` YAML
- Enhanced Workflow GUI - visualize pipelines, sub-DAGs (nested pipelines), loops, and artifacts (datasets, models, and metrics) to understand and debug your pipelines

## Version Matrix

The first version of [Kubeflow Community Distribution](https://www.kubeflow.org/docs/started/introduction/#kubeflow-distribution)
to include the Kubeflow Pipelines V2 backend was [Kubeflow Community Distribution 1.8](https://www.kubeflow.org/docs/kubeflow-distribution/releases/kubeflow-1.8/).

The following table shows which versions of KFP backend are included in each version of Kubeflow Community Distribution:

Release Date | Kubeflow Community Distribution Version | KFP Backend Version | SDK Mode: [`v1`](https://www.kubeflow.org/docs/components/pipelines/legacy-v1/sdk) | SDK Mode: [`v2`](core-functions/compile-a-pipeline.md) | SDK Mode: [`v2-compatible`](https://v1-7-branch.kubeflow.org/docs/components/pipelines/v1/sdk-v2/)
--- | --- | --- | --- | --- | ---
2024-07-22 | [Kubeflow 1.9](https://www.kubeflow.org/docs/kubeflow-distribution/releases/kubeflow-1.9/) | [2.2.0](https://github.com/kubeflow/pipelines/releases/tag/2.2.0) | <i class="fa-solid fa-check"></i> | <i class="fa-solid fa-check"></i> | <i class="fa-solid fa-xmark"></i>
2023-11-01 | [Kubeflow 1.8](https://www.kubeflow.org/docs/kubeflow-distribution/releases/kubeflow-1.8/) | [2.0.3](https://github.com/kubeflow/pipelines/releases/tag/2.0.3) | <i class="fa-solid fa-check"></i> | <i class="fa-solid fa-check"></i> | <i class="fa-solid fa-xmark"></i>
2023-03-29 | [Kubeflow 1.7](https://www.kubeflow.org/docs/kubeflow-distribution/releases/kubeflow-1.7/) | [2.0.0-alpha.7](https://github.com/kubeflow/pipelines/releases/tag/2.0.0-alpha.7) | <i class="fa-solid fa-xmark"></i> | <i class="fa-solid fa-xmark"></i> | <i class="fa-solid fa-check"></i>
2022-10-10 | [Kubeflow 1.6](https://www.kubeflow.org/docs/kubeflow-distribution/releases/kubeflow-1.6/) | [2.0.0-alpha.5](https://github.com/kubeflow/pipelines/releases/tag/2.0.0-alpha.5) | <i class="fa-solid fa-xmark"></i> | <i class="fa-solid fa-xmark"></i> | <i class="fa-solid fa-check"></i>
2022-06-15 | [Kubeflow 1.5](https://www.kubeflow.org/docs/kubeflow-distribution/releases/kubeflow-1.5/) | [1.8.2](https://github.com/kubeflow/pipelines/releases/tag/1.8.2) | <i class="fa-solid fa-check"></i> | <i class="fa-solid fa-xmark"></i> | <i class="fa-solid fa-xmark"></i>

## Backward Compatibility

If you have existing KFP Pipelines that you compiled with the V1 SDK, you can run them on the new KFP V2 backend without any changes.
If you wish to author new pipelines, there are some recommended and required steps to migrate which are detailed below.

:::{warning}
Running V1 pipelines on KFP V2 requires that you compile and submit them using the V1 SDK.
The last version of the V1 SDK was [`kfp==1.8.22`](https://pypi.org/project/kfp/1.8.22/), there will be no further releases.
:::

## Terminology

Term | Definition
--- | ---
SDK v1 | The `1.x.x` versions of the [`kfp`](https://pypi.org/project/kfp/1.8.22/) Python SDK.
SDK v2 | The `2.x.x` versions of the [`kfp`](https://pypi.org/project/kfp/) Python SDK.
SDK v1 (v2-namespace) | The preview V2 module that was available in the V1 SDK (e.g. `from kfp.v2 import *`).<br>_Only ever used by [Google Cloud Vertex AI Pipelines][vertex-pipelines] users._

## Migration Paths

How you migrate to KFP V2 will depend on your current SDK version and usage.

There are two common migration paths:

1. [__SDK v1__ → __SDK v2__](#migrate-from-sdk-v1-to-sdk-v2)
2. [__SDK v1 (v2-namespace) → SDK v2__](#migrate-from-sdk-v1-v2-namespace-to-sdk-v2)

<br>

### **Migrate from 'SDK v1' to 'SDK v2'**

KFP SDK v2 is generally not backward compatible with user code that uses the KFP SDK v1 main namespace. This section describes some of the important breaking changes and migration steps to upgrade to KFP SDK v2.

We indicate whether each breaking change affects [KFP OSS backend][oss-be-v1] users or [Google Cloud Vertex AI Pipelines][vertex-pipelines] users.

#### **Breaking changes**

<details>
<summary>Click to expand</summary>
<hr>

##### **create_component_from_func and func_to_container_op support**

**Affects:** KFP OSS users and Vertex AI Pipelines users

`create_component_from_func` and `func_to_container_op` are both used in KFP SDK v1 to create lightweight Python function-based components.

Both functions are removed in KFP SDK v2.

**Change:** Use the [`@dsl.component`][dsl-component] decorator, as described in [Lightweight Python Components][lightweight-python-components] and [Containerized Python Components][containerized-python-components].

<table>
<tr>
<th>Previous usage</th>
<th>New usage</th>
</tr>
<tr>
<td>

```python
from kfp.components import create_component_from_func
from kfp.components import func_to_container_op

@create_component_from_func
def component1(...):
    ...

def component2(...):
    ...

component2 = create_component_from_func(component2)

@func_to_container_op
def component3(...):
    ...

@dsl.pipeline(name='my-pipeline')
def pipeline():
    component1(...)
    component2(...)
    component3(...)
```

</td>
<td>

```python
from kfp import dsl

@dsl.component
def component1(...):
    ...

@dsl.component
def component2(...):
    ...

@dsl.component
def component3(...):
    ...

@dsl.pipeline(name='my-pipeline')
def pipeline():
    component1(...)
    component2(...)
    component3(...)
```

</td>
</tr>
</table>

---

##### **Keyword arguments required**

**Affects:** KFP OSS users and Vertex AI Pipelines users

Keyword arguments are required when instantiating components as tasks within a pipeline definition.

**Change:** Use keyword arguments.

<table>
<tr>
<th>Previous usage</th>
<th>New usage</th>
</tr>
<tr>
<td>

```python
def my_pipeline():
    trainer_component(100, 0.1)
```

</td>
<td>

```python
def my_pipeline():
    trainer_component(epochs=100, learning_rate=0.1)
```

</td>
</tr>
</table>

---

##### **ContainerOp support**

**Affects:** KFP OSS users

`ContainerOp` has been deprecated since mid-2020. `ContainerOp` instances do not carry a description of inputs and outputs and therefore cannot be compiled to [IR YAML][ir-yaml].

`ContainerOp` is removed from v2.

**Change:** Use the [`@dsl.container_component`][dsl-container-component] decorator as described in [Container Components][container-components].

<table>
<tr>
<th>Previous usage</th>
<th>New usage</th>
</tr>
<tr>
<td>

```python
from kfp import dsl

# v1 ContainerOp will not be supported.
component_op = dsl.ContainerOp(...)

# v1 ContainerOp from class will not be supported.
class FlipCoinOp(dsl.ContainerOp):
```

</td>
<td>

```python
from kfp import dsl

@dsl.container_component
def flip_coin(rand: int, result: dsl.OutputPath(str)):
  return ContainerSpec(
    image='gcr.io/flip-image'
    command=['flip'],
    arguments=['--seed', rand, '--result-file', result])
```

</td>
</tr>
</table>

---

##### **VolumeOp and ResourceOp support**

**Affects:** KFP OSS users

`VolumeOp` and `ResourceOp` expose direct access to Kubernetes resources within a pipeline definition. There is no support for these features on a non-Kubernetes platforms.

KFP v2 enables support for [platform-specific features](core-functions/platform-specific-features.md) via KFP SDK extension libraries. Kubernetes-specific features are supported in KFP v2 via the [`kfp-kubernetes`](https://kfp-kubernetes.readthedocs.io/) extension library.

---

##### **v1 component YAML support**

**Affects:** KFP OSS users and Vertex AI Pipelines users

KFP v1 supported authoring components directly in YAML via the v1 component YAML format ([example][v1-component-yaml-example]). This authoring style enabled component authors to set their component's image, command, and args directly.

In KFP v2, both components and pipelines are compiled to the same [IR YAML][ir-yaml] format, which is different than the v1 component YAML format.

KFP v2 will continue to support loading existing v1 component YAML using the [`components.load_component_from_file`][components-load-component-from-file] function and [similar functions][load] for backward compatibility.

**Change:** To author components via custom image, command, and args, use the [`@dsl.container_component`][dsl-container-component] decorator as described in [Container Components][container-components]. Note that unlike when authoring v1 component YAML, Container Components do not support setting environment variables on the component itself. Environment variables should be set on the task instantiated from the component within a pipeline definition using the [`.set_env_variable`][dsl-pipelinetask-set-env-variable] task [configuration method][task-configuration-methods].

---

##### **v1 lightweight component types InputTextFile, InputBinaryFile, OutputTextFile and OutputBinaryFile support**

**Affects:** KFP OSS users and Vertex AI Pipelines users

These types ensure files are written either in text mode or binary mode in components authored using the KFP SDK v1.

KFP SDK v2 does not support authoring with these types since users can easily do this themselves.

**Change:** Component authors should inputs and outputs using KFP's [artifact][artifacts] and [parameter][parameters] types.

---

##### **AIPlatformClient support**

**Affects:** Vertex AI Pipelines users

KFP SDK v1 included an `AIPlatformClient` for submitting pipelines to [Vertex AI Pipelines][vertex-pipelines].

KFP SDK v2 does not include this client.

**Change:** Use the official [Python Vertex SDK][vertex-sdk]'s `PipelineJob` class.

<table>
<tr>
<th>Previous usage</th>
<th>New usage</th>
</tr>
<tr>
<td>

```python
from kfp.v2.google.client import AIPlatformClient

api_client = AIPlatformClient(
    project_id=PROJECT_ID,
    region=REGION,
)

response = api_client.create_run_from_job_spec(
    job_spec_path=PACKAGE_PATH, pipeline_root=PIPELINE_ROOT,
)
```

</td>
<td>

```python
# pip install google-cloud-aiplatform
from google.cloud import aiplatform

aiplatform.init(
    project=PROJECT_ID,
    location=REGION,
)

job = aiplatform.PipelineJob(
    display_name=DISPLAY_NAME,
    template_path=PACKAGE_PATH,
    pipeline_root=PIPELINE_ROOT,
)

job.submit()
```

</td>
</tr>
</table>

---

##### **run_as_aiplatform_custom_job support**

**Affects:** Vertex AI Pipelines users

KFP v1's `run_as_aiplatform_custom_job` was an experimental feature that allowed converting any component into a [Vertex AI CustomJob][vertex-customjob].

KFP v2 does not include this feature.

**Change:** Use [Google Cloud Pipeline Component][gcpc]'s [create_custom_training_job_from_component][create-custom-training-job-from-component] function.

<table>
<tr>
<th>Previous usage</th>
<th>New usage</th>
</tr>
<tr>
<td>

```python
from kfp import components
from kfp.v2 import dsl
from kfp.v2.google.experimental import run_as_aiplatform_custom_job

training_op = components.load_component_from_url(...)

@dsl.pipeline(name='my-pipeline')
def pipeline():
  training_task = training_op(...)
  run_as_aiplatform_custom_job(
      training_task, ...)
```

</td>
<td>

```python
# pip install google-cloud-pipeline-components
from kfp import components
from kfp import dsl
from google_cloud_pipeline_components.v1.custom_job import utils

training_op = components.load_component_from_url(...)

@dsl.pipeline(name='my-pipeline')
def pipeline():
    utils.create_custom_training_job_from_component(training_op, ...)
```

</td>
</tr>
</table>

---

##### **Typecasting behavior change**

**Affects:** KFP OSS users and Vertex AI Pipelines users

KFP SDK v1 had more lenient pipeline typechecking than does KFP SDK v2. Some pipelines that utilized this leniency may not be compilable using KFP SDK v2. For example, parameters typed with `float` would accept the string `"0.1"`:

```python
from kfp.v2 import compiler
from kfp.v2 import dsl
from kfp import components


@dsl.component
def train(
    number_of_epochs: int,
    learning_rate: float,
):
    print(f"number_of_epochs={number_of_epochs}")
    print(f"learning_rate={learning_rate}")


def training_pipeline(number_of_epochs: int = 1):
    train(
        number_of_epochs=number_of_epochs,
        learning_rate="0.1",  # string cannot be passed to float parameter using KFP SDK v2
    )
```

**Change:** We recommend updating your components and pipelines to use types strictly.

---

</details>

<br>
<br>

### **Migrate from 'SDK v1 (v2-namespace)' to 'SDK v2'**

With few exceptions, KFP SDK v2 is backward compatible with user code that uses the KFP SDK v1 v2-namespace.

:::{note}
This migration path ONLY affects v1 SDK users that were running pipelines on Google Cloud's Vertex AI Pipelines.
:::

#### **Non-breaking changes**

This section documents non-breaking changes in SDK v2 relative to the SDK v1 v2-namespace.
We suggest you migrate your code to the "New usage", even though the "Previous usage" will still work with warnings.

<details>
<summary>Click to expand</summary>
<hr>

##### **Import namespace**

KFP SDK v1 v2-namespace imports (`from kfp.v2 import *`) should be converted to imports from the primary namespace (`from kfp import *`).

**Change:** Remove the `.v2` module from any KFP SDK v1 v2-namespace imports.

<style>
    th {
        text-align: center;
    }
</style>

<table>
<tr>
<th>Previous usage</th>
<th>New usage</th>
</tr>
<tr>
<td>

```python
from kfp.v2 import dsl
from kfp.v2 import compiler

@dsl.pipeline(name='my-pipeline')
def pipeline():
  ...

compiler.Compiler().compile(...)
```

</td>
<td>

```python
from kfp import dsl
from kfp import compiler

@dsl.pipeline(name='my-pipeline')
def pipeline():
  ...

compiler.Compiler().compile(...)
```

</td>
</tr>
</table>

---

##### **output_component_file parameter**

In KFP SDK v2, components can be [compiled][compile] to and [loaded][load] from [IR YAML][ir-yaml] in the same way as pipelines.

KFP SDK v1 v2-namespace supported compiling components via the [`@dsl.component`][dsl-component] decorator's `output_component_file` parameter. This is deprecated in KFP SDK v2. If you choose to still use this parameter, your pipeline will be compiled to [IR YAML][ir-yaml] instead of v1 component YAML.

**Change:** Remove uses of `output_component_file`. Replace with a call to [`Compiler().compile()`][compiler-compile].

<table>
<tr>
<th>Previous usage</th>
<th>New usage</th>
</tr>
<tr>
<td>

```python
from kfp.v2.dsl import component

@component(output_component_file='my_component.yaml')
def my_component(input: str):
   ...
```

</td>
<td>

```python
from kfp.dsl import component
from kfp import compiler

@component()
def my_component(input: str):
   ...

compiler.Compiler().compile(my_component, 'my_component.yaml')
```

</td>
</tr>
</table>

---

##### **Pipeline package file extension**

The KFP compiler will compile your pipeline according to the extension provided to the compiler (`.yaml` or `.json`).

In KFP SDK v2, YAML is the preferred serialization format.

**Change:** Convert `package_path` arguments that use a `.json` extension to use a `.yaml` extension.

<table>
<tr>
<th>Previous usage</th>
<th>New usage</th>
</tr>
<tr>
<td>

```python
from kfp.v2 import compiler
# .json extension, deprecated format
compiler.Compiler().compile(pipeline, package_path='my_pipeline.json')
```

</td>
<td>

```python
from kfp import compiler
# .yaml extension, preferred format
compiler.Compiler().compile(pipeline, package_path='my_pipeline.yaml')
```

</td>
</tr>
</table>

---

</details>

#### **Breaking changes**

There are only a few subtle breaking changes in SDK v2 relative to the SDK v1 v2-namespace.

<details>
<summary>Click to expand</summary>
<hr>

##### **Drop support for Python 3.6**

KFP SDK v1 supported Python 3.6. KFP SDK v2 supports Python >=3.7.0,\<3.12.0.

---

##### **CLI output change**

The v2 [KFP CLI][cli] is more consistent, readable, and parsable. Code that parsed the v1 CLI output may fail to parse the v2 CLI output.

---

##### **.after referencing upstream task in a dsl.ParallelFor loop**

The following pipeline cannot be compiled in KFP SDK v2:

```python
with dsl.ParallelFor(...):
    t1 = comp()
t2 = comp().after(t1)
```

This usage was primarily used by KFP SDK v1 users who implemented a custom `dsl.ParallelFor` fan-in. KFP SDK v2 natively supports fan-in from [`dsl.ParallelFor`][dsl-parallelfor] using [`dsl.Collected`][dsl-collected]. See [Control Flow][parallelfor-control-flow] user docs for instructions.

---

##### **Importer component import statement**

The location of the `importer_node` object has changed.

**Change:** Import from `kfp.dsl`.

<table>
<tr>
<th>Previous usage</th>
<th>New usage</th>
</tr>
<tr>
<td>

```python
from kfp.components import importer_node
```

</td>
<td>

```python
from kfp.dsl import importer_node
```

</td>
</tr>
</table>

---

##### **Adding node selector constraint/accelerator**

The task method `.add_node_selector_constraint` is deprecated in favor of `.set_accelerator_type`. Compared to the previous implementation of `.add_node_selector_constraint`, both methods have the `label_name` parameter removed and the `value` parameter is replaced by the parameter `accelerator`.

**Change:** Use `task.set_accelerator_type(accelerator=...)`. Provide the previous `value` argument to the `accelerator` parameter. Omit the `label_name`.

<table>
<tr>
<th>Previous usage</th>
<th>New usage</th>
</tr>
<tr>
<td>

```python
@dsl.pipeline
def my_pipeline():
    task.add_node_selector_constraint(
        label_name='cloud.google.com/gke-accelerator',
        value='NVIDIA_TESLA_A100',
    )
```

</td>
<td>

```python
@dsl.pipeline
def my_pipeline():
    task.set_accelerator_type(accelerator="NVIDIA_TESLA_K80")
```

</td>
</tr>
</table>

---

</details>

<br>

## Did we miss something?

If you believe we missed a breaking change or an important migration step, please [create an issue][new-issue] describing the change in the [`kubeflow/pipelines` repository][pipelines-repo].

[artifacts]: data-handling/artifacts.md
[cli]: core-functions/cli.md
[compile]: core-functions/compile-a-pipeline.md
[compiler-compile]: https://kubeflow-pipelines.readthedocs.io/en/stable/source/compiler.html#kfp.compiler.Compiler.compile
[components-load-component-from-file]: https://kubeflow-pipelines.readthedocs.io/en/stable/source/components.html#kfp.components.load_component_from_file
[container-components]: components/container-components.md
[containerized-python-components]: components/containerized-python-components.md
[create-custom-training-job-from-component]: https://cloud.google.com/vertex-ai/docs/pipelines/customjob-component
[dsl-collected]: https://kubeflow-pipelines.readthedocs.io/en/stable/source/dsl.html#kfp.dsl.Collected
[dsl-component]: https://kubeflow-pipelines.readthedocs.io/en/stable/source/dsl.html#kfp.dsl.component
[dsl-container-component]: https://kubeflow-pipelines.readthedocs.io/en/stable/source/dsl.html#kfp.dsl.container_component
[dsl-pipeline]: https://kubeflow-pipelines.readthedocs.io/en/stable/source/dsl.html#kfp.dsl.pipeline
[dsl-parallelfor]: https://kubeflow-pipelines.readthedocs.io/en/stable/source/dsl.html#kfp.dsl.ParallelFor
[gcpc]: https://cloud.google.com/vertex-ai/docs/pipelines/components-introduction
[ir-yaml]: ../concepts/ir-yaml.md
[lightweight-python-components]: components/lightweight-python-components.md
[load]: components/load-and-share-components.md
[new-issue]: https://github.com/kubeflow/pipelines/issues/new
[oss-be-v1]: https://www.kubeflow.org/docs/components/pipelines/legacy-v1
[oss-be-v2]: ../operator-guides/installation/index.md
[parallelfor-control-flow]: core-functions/control-flow.md#dslparallelfor
[parameters]: data-handling/parameters.md
[pipelines-repo]: https://github.com/kubeflow/pipelines
[semver-minor-version]: https://semver.org/#:~:text=MINOR%20version%20when%20you%20add%20functionality%20in%20a%20backwards%20compatible%20manner
[v1-component-yaml-example]: https://github.com/kubeflow/pipelines/blob/01c87f8a032e70a6ca92cdbefa974a7da387f204/sdk/python/test_data/v1_component_yaml/add_component.yaml
[vertex-customjob]: https://cloud.google.com/vertex-ai/docs/training/create-custom-job
[vertex-pipelines]: https://cloud.google.com/vertex-ai/docs/pipelines/introduction
[vertex-sdk]: https://cloud.google.com/vertex-ai/docs/pipelines/run-pipeline#vertex-ai-sdk-for-python
[dsl-pipelinetask-set-env-variable]: https://kubeflow-pipelines.readthedocs.io/en/2.0.0b13/source/dsl.html#kfp.dsl.PipelineTask.set_env_variable
[task-configuration-methods]: components/compose-components-into-pipelines.md#task-configurations
