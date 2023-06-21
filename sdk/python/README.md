
Kubeflow Pipelines is a platform for building and deploying portable, scalable machine learning workflows based on Docker containers within the [Kubeflow](https://www.kubeflow.org/) project.

Use Kubeflow Pipelines to compose a multi-step workflow ([pipeline](https://www.kubeflow.org/docs/components/pipelines/concepts/pipeline/)) as a [graph](https://www.kubeflow.org/docs/components/pipelines/concepts/graph/) of containerized [tasks](https://www.kubeflow.org/docs/components/pipelines/concepts/step/) using Python code and/or YAML. Then, [run](https://www.kubeflow.org/docs/components/pipelines/concepts/run/) your pipeline with specified pipeline arguments, rerun your pipeline with new arguments or data, [schedule](https://www.kubeflow.org/docs/components/pipelines/concepts/run-trigger/) your pipeline to run on a recurring basis, organize your runs into [experiments](https://www.kubeflow.org/docs/components/pipelines/concepts/experiment/), save machine learning artifacts to compliant [artifact registries](https://www.kubeflow.org/docs/components/pipelines/concepts/metadata/), and visualize it all through the [Kubeflow Dashboard](https://www.kubeflow.org/docs/components/central-dash/overview/).

## Installation

To install `kfp`, run:

```sh
pip install kfp
```

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
