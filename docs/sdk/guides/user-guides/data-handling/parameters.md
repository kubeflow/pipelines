# Pass small amounts of data between components

Parameters are useful for passing small amounts of data between components and when the data created by a component does not represent a machine learning artifact such as a model, dataset, or more complex data type.

Specify parameter inputs and outputs using built-in Python type annotations:

```python
from kfp import dsl

@dsl.component
def join_words(word: str, count: int = 10) -> str:
    return ' '.join(word for _ in range(count))
```


KFP maps Python type annotations to the types stored in [ML Metadata][ml-metadata] according to the following table:

| Python object          | KFP type |
| ---------------------- | -------- |
| `str`                  | string   |
| `int`                  | number   |
| `float`                | number   |
| `bool`                 | boolean  |
| `typing.List` / `list` | object   |
| `typing.Dict` / `dict` | object   |

As with normal Python function, input parameters can have default values, indicated in the standard way: `def func(my_string: str = 'default'):`

Under the hood KFP passes all parameters to and from components by serializing them as JSON.

For all Python Components ([Lightweight Python Components][lightweight-python-components] and [Containerized Python Components][containerized-python-components]), parameter serialization and deserialization is invisible to the user; KFP handles this automatically.

For [Container Components][container-component], input parameter deserialization is invisible to the user; KFP passes inputs to the component automatically. For Container Component *outputs*, the user code in the Container Component must handle serializing the output parameters as described in [Container Components: Create component outputs][container-component-outputs].

## Input parameters
Using input parameters is very easy. Simply annotate your component function with the types and, optionally, defaults. This is demonstrated by the following pipeline, which uses a Python Component, a Container Component, and a pipeline with all parameter types as inputs:

<!-- TODO: document None default -->

```python
from typing import Dict, List
from kfp import dsl

@dsl.component
def python_comp(
    string: str = 'hello',
    integer: int = 1,
    floating_pt: float = 0.1,
    boolean: bool = True,
    dictionary: Dict = {'key': 'value'},
    array: List = [1, 2, 3],
):
    print(string)
    print(integer)
    print(floating_pt)
    print(boolean)
    print(dictionary)
    print(array)


@dsl.container_component
def container_comp(
    string: str = 'hello',
    integer: int = 1,
    floating_pt: float = 0.1,
    boolean: bool = True,
    dictionary: Dict = {'key': 'value'},
    array: List = [1, 2, 3],
):
    return dsl.ContainerSpec(
        image='alpine',
        command=['sh', '-c', """echo $0 $1 $2 $3 $4 $5 $6"""],
        args=[
            string,
            integer,
            floating_pt,
            boolean,
            dictionary,
            array,
        ])

@dsl.pipeline
def my_pipeline(
    string: str = 'Hey!',
    integer: int = 100,
    floating_pt: float = 0.1,
    boolean: bool = False,
    dictionary: Dict = {'key': 'value'},
    array: List = [1, 2, 3],
):
    python_comp(
        string='howdy',
        integer=integer,
        array=[4, 5, 6],
    )
    container_comp(
        string=string,
        integer=20,
        dictionary={'other key': 'other val'},
        boolean=boolean,
    )
```

### Output parameters

For Python Components and pipelines, output parameters are indicated via return annotations:

```python
from kfp import dsl

@dsl.component
def my_comp() -> int:
    return 1

@dsl.pipeline
def my_pipeline() -> int:
    task = my_comp()
    return task.output
```

For Container Components, output parameters are indicated using a [`dsl.OutputPath`][dsl-outputpath] annotation:

```python
from kfp import dsl

@dsl.container_component
def my_comp(int_path: dsl.OutputPath(int)):
    return dsl.ContainerSpec(
        image='alpine',
        command=[
            'sh', '-c', f"""mkdir -p $(dirname {int_path})\
                            && echo 1 > {int_path}"""
        ])

@dsl.pipeline
def my_pipeline() -> int:
    task = my_comp()
    return task.outputs['int_path']
```

See [Container Components: Create component outputs][container-component-outputs] for more information on how to use `dsl.OutputPath`

### Multiple output parameters
You can specify multiple named output parameters using a [`typing.NamedTuple`][typing-namedtuple]. You can access a named output using `.outputs['<output-key>']` on [`PipelineTask`][pipelinetask]:

```python
from kfp import dsl
from typing import NamedTuple

@dsl.component
def my_comp() -> NamedTuple('outputs', a=int, b=str):
    outputs = NamedTuple('outputs', a=int, b=str)
    return outputs(1, 'hello')

@dsl.pipeline
def my_pipeline() -> NamedTuple('pipeline_outputs', c=int, d=str):
    task = my_comp()
    pipeline_outputs = NamedTuple('pipeline_outputs', c=int, d=str)
    return pipeline_outputs(task.outputs['a'], task.outputs['b'])
```


[ml-metadata]: https://github.com/google/ml-metadata
[lightweight-python-components]: ../components/lightweight-python-components.md
[containerized-python-components]: ../components/containerized-python-components.md
[container-component]: ../components/container-components.md
[container-component-outputs]: ../components/container-components.md#create-component-outputs
[pipelinetask]: https://kubeflow-pipelines.readthedocs.io/en/stable/source/dsl.html#kfp.dsl.PipelineTask
[dsl-outputpath]: https://kubeflow-pipelines.readthedocs.io/en/latest/source/dsl.html#kfp.dsl.OutputPath
[typing-namedtuple]: https://docs.python.org/3/library/typing.html#typing.NamedTuple
