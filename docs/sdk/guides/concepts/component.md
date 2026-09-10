# Component

A **pipeline component** is the fundamental building block for an ML engineer to construct a Kubeflow Pipelines [pipeline][pipeline]. The component structure serves the purpose of packaging a functional unit of code along with its dependencies, so that it can be run as part of a workflow in a Kubernetes environement. Components can be combined in a [pipeline][pipeline] that creates a repeatable workflow, with individual components coordinating on inputs and outputs like parameters and [artifacts][artifacts].

A component is similar to a programming function. Indeed, it is most often implemented as a wrapper to a Python function using the [KFP Python SDK][KFP SDK]. However, a KFP component goes further than a simple function, with support for code dependencies, runtime environments, and distributed execution requirements.

KFP components are designed to simplify constructing and running ML workflows in a Kubernetes environment. Using KFP components, engineers can iterate faster, reduce maintenance overhead, and focus more attention on ML work.

## The Why Behind KFP Components

Running ML code in a Kubernetes cluster presents many challenges. Some of the main challenges are:
- Managing **code dependencies** (Python libraries and versions)
- Handling **system dependencies** (OS-level packages, GPU drivers, runtime environments)
- **Building and maintaining container images** and everything around this from container registry support to CVE (Common Vulnerabilities and Exposures) fixes
- **Deploying supporting resources** like PersistentVolumeClaims and ConfigMaps
- Handling **inputs and outputs** including metadata, parameters, logs, and artifacts
- **Ensuring compatibility** across clusters, images, and dependencies

KFP components simplify these challenges by enabling ML engineers to:
- **Stay at the Python level** - where most modern ML work occurs
- **Iterate quickly** – modify code without creating or rebuilding containers at each step
- **Focus on ML tasks** - rather than on platform and infrastructure concerns
- **Work seamlessly with Python IDE tools** – enable debugging, syntax highlighting, type checking, and docstring usage
- **Move between environments** – transition from local development to distributed execution with minimal changes

## What Does a Component Consist Of?

A KFP component consist of the following key elements:

### 1. Code
- Typically a Python function, but can be other code such as a Bash command.

### 2. Dependency Support
- **Python libraries** - to be installed at runtime
- **Environment variables** - to be available in the runtime environment
- **Python package indices** - (for example, private PyPi servers) if needed to support installations
- **Cluster resources** - to support use of ConfigMaps, Secrets, PersistentVolumeClaims, and more
- **Runtime dependencies** - to support CPU, memory, and GPU requests and limits

### 3. Base Image
- Defines the base container runtime environment (defaults to a generic Python base image)
- May include system dependencies and pre-installed Python libraries

### 4. Input/Output (I/O) Specification
- Individual components cannot share in-memory data with each other, so they use the following concepts to support exchanging information and publishing results:
  - **Parameters** – for small values
  - **[Artifacts][artifacts]** - for larger data like model files, processed datasets, and metadata

## Constructing a Component

### 1. Python-Based Components
The recommended way to define a component is using the `@dsl.component` decorator from the [KFP Python SDK][KFP SDK]. Below are two basic component definition examples:
```python
from kfp.dsl import component, Output, Dataset

# hello world component
@component()
def hello_world(name: str = "World") -> str:
    print(f"Hello {name}!")
    return name

# process data component
@component(
    base_image="python:3.12-slim-bookworm",
    packages_to_install=["pandas>=2.2.3"],
)
def process_data(output_data: Output[Dataset]):
    '''Get max from an array'''
    import pandas as pd
   # create dataset to write to output
    data = pd.DataFrame(data=[[1,2,3],[4,5,6]], columns=["a","b","c"])
    data.to_csv(output_data.path)
```

Observe that these are wrapped Python functions. The `@component` wrapper helps the KFP Python SDK supply the needed context for running these functions in containers as part of a KFP [pipeline][pipeline].

The `hello_world` component just uses the default behavior, which is to run the Python function on the default base image (`kfp.dsl.component_factory._DEFAULT_BASE_IMAGE`).

The `process_data` component adds layers of customization, by supplying the name of a specific `base_image`, and `packages_to_install`. Note the inclusion of the `import pandas as pd` statement inside the function; since the function will run inside a container (and won't have the script context), all Python library dependencies need to be imported within the component function. This component also uses KFP's `Output[Dataset]` class, which takes care of creating a KFP [artifact][artifacts] type output.

Note that inputs and outputs are defined as Python function parameters. Also, dependencies can often be installed at runtime, avoiding the need for custom base containers. Python-based components give close access to the Python tools that ML experimenters rely on, like modules and imports, usage information, type hints, and debugging tools.


#### Run a Component's Python Function
Provided that dependencies are satisfied in your environment, it is also easy to run Python-based components as simple Python functions, which can be useful for local work. For example, to run `process_data` as a Python function try:
``` python
# Provide path as dataset type (as the function expects)
dataset = Dataset(uri="data.csv")
# execute the function
# (writes data to data.csv locally)
process_data.execute(output_data=dataset)
# access the underlying function docstring
print(process_data.python_func.__doc__)
```

Component usage can get much more complex, as AI/ML use-cases often have demanding code and environment dependencies. For more on creating Python-based components, see the [component][python-sdk-component] SDK documentation.


### 2. YAML-Based Components

The KFP backend uses YAML-based definitions to specify components. While the [KFP Python SDK][KFP SDK] can do this conversion automatically when a Python-based [pipeline][pipeline] is submitted, some use-cases can benefit from the direct YAML-based component approach.

A YAML-based component definition has the following parts:

* **Metadata:** name, description, etc.
* **Interface:** input/output specifications (name, type, description, default value, etc).
* **Implementation:** A specification of how to run the component given a set of argument values for the component’s inputs. The implementation section also describes how to get the output values from the component once the component has finished running.

YAML-based components support system commands directly. In fact, any command (or binary) that exists on the base image can be run. Here is simple YAML-based component example:
```yaml
# my_component.yaml file
name: my-component
description: "Component that outputs \"<string prefix>...<num>\""

inputs:
- {name: string prefix, type: String}
- {name: num, type: Integer}

outputs: []

implementation:
  container:
    image: python:3.12-slim-bookworm
    args:
    - echo
    - {inputValue: string prefix}
    - ...
    - {inputValue: num}
```

For the complete definition of a YAML-based component, see the [component specification][yaml-component].

YAML-based components can be loaded for use in the Python SDK alongside Python-based components:
```python
from kfp.components import load_component_from_file

my_comp = load_component_from_file("my_component.yaml")
```

Note that a component loaded from a YAML-based component will not have the same level of Python support that Python-based components do (like executing the function locally).

<!-- TODO: Briefly discuss graph components, container components, and importer components (see sdk dsl scripts) -->

## "Containerize" a Component

The KFP command-line tool contains a build command to help users "containerize" a component. This can be used to create the `Dockerfile`, `runtime-dependencies.txt`, and other supporting files, or even to build the custom image and push it to a registry. In order to use this utility, the `target_image` parameter must be set in the Python-based component definition, which itself is saved in a file.
```bash
# build Dockerfile and runtime-dependencies.txt
kfp component build --component-filepattern the_component.py --no-build-image --platform linux/amd64 .
```
Note that creating and maintaining custom containers can carry a significant maintenance burden. In general, a 1-to-1 relationship between components and containers is not needed or recommended, as AI/ML work is often highly iterative. A best practice is to work with a small set of base images that can support many components. If you need more control over the container build than the `kfp` CLI provides, consider using a container CLI like [docker][docker-cli] or [podman][podman-cli].


## Next steps

* Read the user guides for [Creating Components][Creating Components]
* Read an [overview of Kubeflow Pipelines](../overview.md).
* Follow the [pipelines quickstart guide](../getting-started.md)
  to deploy Kubeflow and run a sample pipeline directly from the Kubeflow
  Pipelines UI.
* Build your own
  [component and pipeline](https://www.kubeflow.org/docs/components/pipelines/legacy-v1/sdk/component-development).
* Build a [reusable component](https://www.kubeflow.org/docs/components/pipelines/legacy-v1/sdk/component-development) for
  sharing in multiple pipelines.


[pipeline]: pipeline.md
[KFP SDK]: https://kubeflow-pipelines.readthedocs.io
[artifacts]: output-artifact.md
[python-sdk-component]: https://kubeflow-pipelines.readthedocs.io/en/stable/source/dsl.html#kfp.dsl.component
[yaml-component]: ../reference/component-spec.md
[docker-cli]: https://github.com/docker/cli
[podman-cli]: https://github.com/containers/podman
[Creating Components]: ../user-guides/components/index.md
