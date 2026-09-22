# Component

A **pipeline component** is the fundamental building block for an ML engineer to construct a Kubeflow Pipelines [pipeline][pipeline]. The component structure serves the purpose of packaging a functional unit of code along with its dependencies, so that it can be run as part of a workflow in a Kubernetes environment. Components can be combined in a [pipeline][pipeline] that creates a repeatable workflow, with individual components coordinating on inputs and outputs like parameters and [artifacts][artifacts].

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

Component usage can get much more complex, as AI/ML use-cases often have demanding code and environment dependencies. For more on creating Python-based components, see the {py:func}`component <kfp.dsl.component>` SDK documentation.


### 2. Container Components and IR YAML

Use [`@dsl.container_component`](../user-guides/components/container-components.md)
to configure an existing container's image, command, and arguments. Compile the
component with `kfp.compiler.Compiler().compile()` to produce a portable
[IR YAML definition](ir-yaml.md). The file, URL, and text component loaders accept
this compiled format and retain compatibility with legacy container component
YAML by converting it to native v2 components.

See [loading and sharing components](../user-guides/components/load-and-share-components.md).

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
  [component and pipeline](../user-guides/components/index.md).
* Build a [reusable component](../user-guides/components/index.md) for
  sharing in multiple pipelines.


[pipeline]: pipeline.md
[KFP SDK]: ../python-sdk.md
[artifacts]: output-artifact.md
[docker-cli]: https://github.com/docker/cli
[podman-cli]: https://github.com/containers/podman
[Creating Components]: ../user-guides/components/index.md
