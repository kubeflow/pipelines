# Container Components

In KFP, each task execution corresponds to a container execution. This means that all components, even Python Components, are defined by an `image`, `command`, and `args`.

Python Components are unique because they abstract most aspects of the container definition away from the user, making it convenient to construct components that use pure Python. Under the hood, the KFP SDK sets the `image`, `commands`, and `args` to the values needed to execute the Python component for the user.

**Container Components, unlike Python Components, enable component authors to set the `image`, `command`, and `args` directly.** This makes it possible to author components that execute shell scripts, use other languages and binaries, etc., all from within the KFP Python SDK.

## A simple Container Component

The following starts with a simple `say_hello` Container Component and gradually modifies it until it is equivalent to our `say_hello` component from the [Hello World Pipeline example][hello-world-pipeline]. Here is a simple Container Component:

```python
from kfp import dsl

@dsl.container_component
def say_hello():
    return dsl.ContainerSpec(image='alpine', command=['echo'], args=['Hello'])
```

To create a Container Components, use the [`dsl.container_component`][dsl-container-component] decorator and create a function that returns a [`dsl.ContainerSpec`][dsl-containerspec] object. `dsl.ContainerSpec` accepts three arguments: `image`, `command`, and `args`. The component above runs the command `echo` with the argument `Hello` in a container running the image [`alpine`][alpine].

Container Components can be used in pipelines just like Python Components:

```python
from kfp import dsl
from kfp import compiler

@dsl.pipeline
def hello_pipeline():
    say_hello()

compiler.Compiler().compile(hello_pipeline, 'pipeline.yaml')
```

If you run this pipeline, you'll see the string `Hello` in `say_hello`'s logs.

### Use component inputs

To be more useful, `say_hello` should be able to accept arguments. You can modify `say_hello` so that it accepts an input argument `name`:

```python
from kfp import dsl

@dsl.container_component
def say_hello(name: str):
    return dsl.ContainerSpec(image='alpine', command=['echo'], args=[f'Hello, {name}!'])
```

The parameters and annotations in the Container Component function declare the component's interface. In this case, the component has one input parameter `name` and no output parameters.

When you compile this component, `name` will be replaced with a placeholder. At runtime, this placeholder is replaced with the actual value for `name` provided to the `say_hello` component.

Another way to implement this component is to use `sh -c` to read the commands from a single string and pass the name as an argument. This approach tends to be more flexible, as it readily allows chaining multiple commands together.

```python
from kfp import dsl

@dsl.container_component
def say_hello(name: str):
    return dsl.ContainerSpec(image='alpine', command=['sh', '-c', 'echo Hello, $0!'], args=[name])
```

When you run the component with the argument `name='World'`, you’ll see the string `'Hello, World!'` in `say_hello`’s logs.

### Create component outputs

Unlike Python functions, containers do not have a standard mechanism for returning values. To enable Container Components to have outputs, KFP requires you to write outputs to a file inside the container. KFP will read this file and persist the output.

To return an output string from the say `say_hello` component, you can add an output parameter to the function using a `dsl.OutputPath(str)` annotation:

```python
@dsl.container_component
def say_hello(name: str, greeting: dsl.OutputPath(str)):
    ...
```

This component now has one input parameter named `name` typed `str` and one output parameter named `greeting` also typed `str`. At runtime, parameters annotated with [`dsl.OutputPath`][dsl-outputpath] will be provided a system-generated path as an argument. Your component logic should write the output value to this path as JSON. The argument `str` in `greeting: dsl.OutputPath(str)` describes the type of the output `greeting` (e.g., the JSON written to the path `greeting` will be a string). You can fill in the `command` and `args` to write the output:

```python
@dsl.container_component
def say_hello(name: str, greeting: dsl.OutputPath(str)):
    """Log a greeting and return it as an output."""

    return dsl.ContainerSpec(
        image='alpine',
        command=[
            'sh', '-c', '''RESPONSE="Hello, $0!"\
                            && echo $RESPONSE\
                            && mkdir -p $(dirname $1)\
                            && echo $RESPONSE > $1
                            '''
        ],
        args=[name, greeting])
```

### Use in a pipeline

Finally, you can use the updated `say_hello` component in a pipeline:

```python
from kfp import dsl
from kfp import compiler

@dsl.pipeline
def hello_pipeline(person_to_greet: str) -> str:
    # greeting argument is provided automatically at runtime!
    hello_task = say_hello(name=person_to_greet)
    return hello_task.outputs['greeting']

compiler.Compiler().compile(hello_pipeline, 'pipeline.yaml')
```

Note that you will never provide output parameters to components when constructing your pipeline; output parameters are always provided automatically by the backend at runtime.

This should look very similar to the [Hello World pipeline][hello-world-pipeline] with one key difference: since `greeting` is a named output parameter, we access it and return it from the pipeline using `hello_task.outputs['greeting']`, instead of `hello_task.output`. Data passing is discussed in more detail in [Pipelines Basics][pipeline-basics].

### Special placeholders
Each of the three component authoring styles automatically handle data passing into your component via placeholders in the container `command` and `args`. In general, you do not need to know how this works. Container Components also enable you to directly utilize two special placeholders if you wish: [`dsl.ConcatPlaceholder`][dsl-concatplaceholder] and [`dsl.IfPresentPlaceholder`][dsl-ifpresentplaceholder].

You may only use these placeholders in the [`dsl.ContainerSpec`][dsl-containerspec] returned from a Container Component authored via the [`@dsl.container_component`][dsl-container-component] decorator.

#### dsl.ConcatPlaceholder

When you provide a container `command` or container `args` as a list of strings, each element in the list is concatenated using a space separator, then issued to the container at runtime. Concatenating one input to another string without a space separator requires special handling provided by the [`dsl.ConcatPlaceholder`][dsl-concatplaceholder].

`dsl.ConcatPlaceholder` takes one argument, `items`, which may be a list of any combination of static strings, upstream outputs, pipeline parameters, or other instances of `dsl.ConcatPlaceholder` or `dsl.IfPresentPlaceholder`. At runtime, these strings will be concatenated together without a separator.

For example, you can use `dsl.ConcatPlaceholder` to concatenate a file path prefix, suffix, and extension:

```python
from kfp import dsl

@dsl.container_component
def concatenator(prefix: str, suffix: str):
    return dsl.ContainerSpec(
        image='alpine',
        command=[
            'my_program.sh'
        ],
        args=['--input', dsl.ConcatPlaceholder([prefix, suffix, '.txt'])]
    )
```

#### dsl.IfPresentPlaceholder
[`dsl.IfPresentPlaceholder`][dsl-ifpresentplaceholder] is used to conditionally provide command line arguments. The `dsl.IfPresentPlaceholder` takes three arguments: `input_name`, `then`, and optionally `else_`. This placeholder is easiest to understand through an example:

```python
@dsl.container_component
def hello_someone(optional_name: str = None):
    return dsl.ContainerSpec(
        image='python:3.7',
        command=[
            'say_hello',
            dsl.IfPresentPlaceholder(
                input_name='optional_name', then=['--name', optional_name])
        ])
```

If the `hello_someone` component is passed `'world'` as an argument for `optional_name`, the component will pass `--name world` to the executable `say_hello`. If `optional_name` is not provided, `--name world` is omitted.

The third parameter `else_` can be used to provide a default value to fall back to if `input_name` is not provided. For example:

```python
@dsl.container_component
def hello_someone(optional_name: str = None):
    return dsl.ContainerSpec(
        image='python:3.7',
        command=[
            'say_hello',
            dsl.IfPresentPlaceholder(
                input_name='optional_name',
                then=['--name', optional_name],
                else_=['--name', 'friend'])
        ])
```

Arguments to `then` and `else_` may be a list of any combination of static strings, upstream outputs, pipeline parameters, or other instances of `dsl.ConcatPlaceholder` or `dsl.IfPresentPlaceholder`


[hello-world-pipeline]: ../../getting-started.md
[pipeline-basics]: compose-components-into-pipelines.md
[alpine]: https://hub.docker.com/_/alpine
[dsl-outputpath]: https://kubeflow-pipelines.readthedocs.io/en/latest/source/dsl.html#kfp.dsl.OutputPath
[dsl-container-component]: https://kubeflow-pipelines.readthedocs.io/en/latest/source/dsl.html#kfp.dsl.container_component
[dsl-containerspec]: https://kubeflow-pipelines.readthedocs.io/en/latest/source/dsl.html#kfp.dsl.ContainerSpec
[dsl-concatplaceholder]: https://kubeflow-pipelines.readthedocs.io/en/latest/source/dsl.html#kfp.dsl.ConcatPlaceholder
[dsl-ifpresentplaceholder]: https://kubeflow-pipelines.readthedocs.io/en/latest/source/dsl.html#kfp.dsl.IfPresentPlaceholder
