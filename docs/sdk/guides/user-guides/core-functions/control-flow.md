# Control Flow

## Overview

Although a KFP pipeline decorated with the `@dsl.pipeline` decorator looks like a normal Python function, it is actually an expression of pipeline topology and control flow semantics, constructed using the KFP domain-specific language (DSL).

The [components guide][pipeline-basics] shows how pipeline topology is expressed by [data passing and task dependencies][data-passing].
This section describes how to introduce control flow in your pipelines to create more complex workflows.

The core types of control flow in KFP pipelines are:

1. [__Conditions__](#conditions)
2. [__Loops__](#loops)
3. [__Exit handling__](#exit-handling)

## Conditions

Kubeflow Pipelines supports common conditional control flow constructs.
You can use these constructs to conditionally execute tasks based on the output of an upstream task or pipeline input parameter.

:::{admonition} Deprecated
:class: warning

The `dsl.Condition` is deprecated in favor of the functionally identical `dsl.If`, which is concise, Pythonic, and consistent with the `dsl.Elif` and `dsl.Else` objects.
:::

### **dsl.If** / **dsl.Elif** / **dsl.Else**

The [`dsl.If`][dsl-if] context manager enables conditional execution of tasks within its scope based on the output of an upstream task or pipeline input parameter.

The context manager takes two arguments: a required `condition` and an optional `name`.
The `condition` is a comparative expression where at least one of the two operands is an output from an upstream task or a pipeline input parameter.

In the following pipeline, `conditional_task` only executes if `coin_flip_task` has the output `'heads'`.

```python
from kfp import dsl

@dsl.component
def flip_coin() -> str:
    import random
    return random.choice(['heads', 'tails'])

#@dsl.component
#def my_comp():
#    print('Conditional task executed!')

@dsl.pipeline
def my_pipeline():
    coin_flip_task = flip_coin()
    with dsl.If(coin_flip_task.output == 'heads'):
        conditional_task = my_comp()
```

You may also use [`dsl.Elif`][dsl-elif] and [`dsl.Else`][dsl-else] context managers **immediately downstream** of `dsl.If` for additional conditional control flow functionality:

```python
from kfp import dsl

@dsl.component
def flip_three_sided_coin() -> str:
    import random
    return random.choice(['heads', 'tails', 'draw'])

@dsl.component
def print_comp(text: str):
    print(text)

@dsl.pipeline
def my_pipeline():
    coin_flip_task = flip_three_sided_coin()
    with dsl.If(coin_flip_task.output == 'heads'):
        print_comp(text='Got heads!')
    with dsl.Elif(coin_flip_task.output == 'tails'):
        print_comp(text='Got tails!')
    with dsl.Else():
        print_comp(text='Draw!')
```

### **dsl.OneOf**

[`dsl.OneOf`][dsl-oneof] can be used to gather outputs from mutually exclusive branches into a single task output which can be consumed by a downstream task or outputted from a pipeline.
Branches are mutually exclusive if exactly one will be executed.
To enforce this, the KFP SDK compiler requires `dsl.OneOf` consume from tasks within a logically associated group of conditional branches and that one of the branches is a `dsl.Else` branch.

For example, the following pipeline uses `dsl.OneOf` to gather outputs from mutually exclusive branches:

```python
from kfp import dsl

@dsl.component
def flip_three_sided_coin() -> str:
    import random
    return random.choice(['heads', 'tails', 'draw'])

@dsl.component
def print_and_return(text: str) -> str:
    print(text)
    return text

@dsl.component
def announce_result(result: str):
    print(f'The result is: {result}')

@dsl.pipeline
def my_pipeline() -> str:
    coin_flip_task = flip_three_sided_coin()
    with dsl.If(coin_flip_task.output == 'heads'):
        t1 = print_and_return(text='Got heads!')
    with dsl.Elif(coin_flip_task.output == 'tails'):
        t2 = print_and_return(text='Got tails!')
    with dsl.Else():
        t3 = print_and_return(text='Draw!')

    oneof = dsl.OneOf(t1.output, t2.output, t3.output)
    announce_result(oneof)
    return oneof
```

You should provide task outputs to the `dsl.OneOf` using `.output` or `.outputs[<key>]`, just as you would pass an output to a downstream task.
The outputs provided to `dsl.OneOf` must be of the same type and cannot be other instances of `dsl.OneOf` or [`dsl.Collected`][dsl-collected].

## Loops

Kubeflow Pipelines supports loops which cause fan-out and fan-in of tasks.

### **dsl.ParallelFor**

The [`dsl.ParallelFor`][dsl-parallelfor] context manager allows parallel execution of tasks over a static set of items.

The context manager takes three arguments:

- `items`: static set of items to loop over
- `name` (optional): is the name of the loop context
- `parallelism` (optional): is the maximum number of concurrent iterations while executing the `dsl.ParallelFor` group
     - note, `parallelism=0` indicates unconstrained parallelism

In the following pipeline, `train_model` will train a model for 1, 5, 10, and 25 epochs, with no more than two training tasks running at one time:

```python
from kfp import dsl

#@dsl.component
#def train_model(epochs: int) -> Model:
#    ...

@dsl.pipeline
def my_pipeline():
    with dsl.ParallelFor(
        items=[1, 5, 10, 25],
        parallelism=2
    ) as epochs:
        train_model(epochs=epochs)
```

### **dsl.Collected**

Use [`dsl.Collected`][dsl-collected] with `dsl.ParallelFor` to gather outputs from a parallel loop of tasks.

#### **Example:** Using `dsl.Collected` as an input to a downstream task

Downstream tasks might consume `dsl.Collected` outputs via an input annotated with a `List` of parameters or a `List` of artifacts.

For example, in the following pipeline, `max_accuracy` has the input `models` with type `Input[List[Model]]`, and will find the model with the highest accuracy from the models trained in the parallel loop:

```python
from kfp import dsl
from kfp.dsl import Model, Input

#def score_model(model: Model) -> float:
#    return ...

#@dsl.component
#def train_model(epochs: int) -> Model:
#    ...

@dsl.component
def max_accuracy(models: Input[List[Model]]) -> float:
    return max(score_model(model) for model in models)

@dsl.pipeline
def my_pipeline():

    # Train a model for 1, 5, 10, and 25 epochs
    with dsl.ParallelFor(
        items=[1, 5, 10, 25],
    ) as epochs:
        train_model_task = train_model(epochs=epochs)

    # Find the model with the highest accuracy
    max_accuracy(
        models=dsl.Collected(train_model_task.outputs['model'])
    )
```

#### **Example:** Nested lists of parameters

You can use `dsl.Collected` to collect outputs from nested loops in a *nested list* of parameters.

For example, output parameters from two nested `dsl.ParallelFor` groups are collected in a multilevel nested list of parameters, where each nested list contains the output parameters from one of the `dsl.ParallelFor` groups.
The number of nested levels is based on the number of nested `dsl.ParallelFor` contexts.

By comparison, *artifacts* created in nested loops are collected in a *flat* list.

#### **Example:** Returning `dsl.Collected` from a pipeline

You can also return a `dsl.Collected` from a pipeline.

Use a `List` of parameters or a `List` of artifacts in the return annotation, as shown in the following example:

```python
from typing import List

from kfp import dsl
from kfp.dsl import Model

#@dsl.component
#def train_model(epochs: int) -> Model:
#    ...

@dsl.pipeline
def my_pipeline() -> List[Model]:
    with dsl.ParallelFor(
        items=[1, 5, 10, 25],
    ) as epochs:
        train_model_task = train_model(epochs=epochs)
    return dsl.Collected(train_model_task.outputs['model'])
```

## Exit handling

Kubeflow Pipelines supports exit handlers for implementing cleanup and error handling tasks that run after the main pipeline tasks finish execution.

### **dsl.ExitHandler**

The [`dsl.ExitHandler`][dsl-exithandler] context manager allows pipeline authors to specify an exit task which will run after the tasks within the context manager's scope finish execution, even if one of those tasks fails.

This construct is analogous to using a `try:` block followed by a `finally:` block in normal Python, where the exit task is in the `finally:` block.
The context manager takes two arguments: a required `exit_task` and an optional `name`. `exit_task` accepts an instantiated [`PipelineTask`][dsl-pipelinetask].

#### **Example:** Basic cleanup task

The most common use case for `dsl.ExitHandler` is to run a cleanup task after the main pipeline tasks finish execution.

In the following pipeline, `clean_up_task` will execute after both `create_dataset` and `train_and_save_models` finish (regardless of whether they succeed or fail):

```python
from kfp import dsl
from kfp.dsl import Dataset

#@dsl.component
#def clean_up_resources():
#    ...

#@dsl.component
#def create_datasets():
#    ...

#@dsl.component
#def train_and_save_models(dataset: Dataset):
#    ...

@dsl.pipeline
def my_pipeline():
    clean_up_task = clean_up_resources()
    with dsl.ExitHandler(exit_task=clean_up_task):
        dataset_task = create_datasets()
        train_task = train_and_save_models(dataset=dataset_task.output)
```

### **Example:** Accessing pipeline and task status metadata

The task you use as an exit task may use a special input that provides access to pipeline and task status metadata, including pipeline failure or success status.

You can use this special input by annotating your exit task with the [`dsl.PipelineTaskFinalStatus`][dsl-pipelinetaskfinalstatus] annotation.
The argument for this parameter will be provided by the backend automatically at runtime.
You should not provide any input to this annotation when you instantiate your exit task.

The following pipeline uses `dsl.PipelineTaskFinalStatus` to obtain information about the pipeline and task failure, even after `fail_op` fails:

```python
from kfp import dsl
from kfp.dsl import PipelineTaskFinalStatus

@dsl.component
def print_op(message: str):
    print(message)

@dsl.component
def exit_op(user_input: str, status: PipelineTaskFinalStatus):
    """Prints pipeline run status."""
    print(user_input)
    print('Pipeline status: ', status.state)
    print('Job resource name: ', status.pipeline_job_resource_name)
    print('Pipeline task name: ', status.pipeline_task_name)
    print('Error code: ', status.error_code)
    print('Error message: ', status.error_message)

@dsl.component
def fail_op():
    import sys
    sys.exit(1)

@dsl.pipeline
def my_pipeline():
    print_op(message='Starting pipeline...')
    print_status_task = exit_op(user_input='Task execution status:')
    with dsl.ExitHandler(exit_task=print_status_task):
        fail_op()
```

#### **Example:** Ignoring upstream task failures

The [`.ignore_upstream_failure()`][ignore-upstream-failure] task method on [`PipelineTask`][dsl-pipelinetask] enables another approach to author pipelines with exit handling behavior.
Calling this method on a task causes the task to ignore failures of any specified upstream tasks (as established by data exchange or by use of [`.after()`][dsl-pipelinetask-after]).
If the task has no upstream tasks, this method has no effect.

In the following pipeline definition, `clean_up_task` is executed after `fail_task`, regardless of whether `fail_op` succeeds:

```python
from kfp import dsl

@dsl.component
def cleanup_op(message: str = 'Cleaning up...'):
    print(message)

@dsl.component
def fail_op(message: str):
    print(message)
    raise ValueError('Task failed!')

@dsl.pipeline()
def my_pipeline(text: str = 'message'):
    fail_task = fail_op(message=text)
    clean_up_task = cleanup_op(
        message=fail_task.output
    ).ignore_upstream_failure()
```

Note that the component used for the caller task (`cleanup_op` in the example above) requires a default value for all inputs it consumes from an upstream task.
The default value is applied if the upstream task fails to produce the outputs that are passed to the caller task.
Specifying default values ensures that the caller task always succeeds, regardless of the status of the upstream task.

[data-passing]: ../components/compose-components-into-pipelines.md#data-passing-and-task-dependencies
[pipeline-basics]: ../components/compose-components-into-pipelines.md
[dsl-condition]: https://kubeflow-pipelines.readthedocs.io/en/latest/source/dsl.html#kfp.dsl.Condition
[dsl-exithandler]: https://kubeflow-pipelines.readthedocs.io/en/latest/source/dsl.html#kfp.dsl.ExitHandler
[dsl-parallelfor]: https://kubeflow-pipelines.readthedocs.io/en/latest/source/dsl.html#kfp.dsl.ParallelFor
[dsl-pipelinetaskfinalstatus]: https://kubeflow-pipelines.readthedocs.io/en/latest/source/dsl.html#kfp.dsl.PipelineTaskFinalStatus
[ignore-upstream-failure]: https://kubeflow-pipelines.readthedocs.io/en/latest/source/dsl.html#kfp.dsl.PipelineTask.ignore_upstream_failure
[dsl-pipelinetask]: https://kubeflow-pipelines.readthedocs.io/en/latest/source/dsl.html#kfp.dsl.PipelineTask
[dsl-pipelinetask-after]: https://kubeflow-pipelines.readthedocs.io/en/latest/source/dsl.html#kfp.dsl.PipelineTask.after
[dsl-if]: https://kubeflow-pipelines.readthedocs.io/en/latest/source/dsl.html#kfp.dsl.If
[dsl-elif]: https://kubeflow-pipelines.readthedocs.io/en/latest/source/dsl.html#kfp.dsl.Elif
[dsl-else]: https://kubeflow-pipelines.readthedocs.io/en/latest/source/dsl.html#kfp.dsl.Else
[dsl-oneof]: https://kubeflow-pipelines.readthedocs.io/en/latest/source/dsl.html#kfp.dsl.OneOf
[dsl-collected]: https://kubeflow-pipelines.readthedocs.io/en/latest/source/dsl.html#kfp.dsl.Collected
