# Pipeline

A *pipeline* declares the logical structure for executing [components][component] together as a machine learning (ML) workflow in a Kubernetes cluster; this includes defining execution order and conditions, as well as configuring parameter passing and data flow. Pipelines are organized as directed [graphs][graph] that progress through individual [steps][step] (defined by [components][component]).

When a pipeline is executed through a [run][run], the Kubeflow Pipelines backend converts the pipeline into instructions to launch Kubernetes Pods (and other resources)
to carry out the [steps][step] ([components][component]) in the workflow ([pipeline][pipeline]). The Pods start containers, which in turn run their respective component code. The Kubeflow Pipelines backend manages the technical details of coordinating data passing and control flow at the container level.


## The Why Behind Kubeflow Pipelines

The work of an AI/ML engineer requires structured experimentation and iteration of workflows with complex data processing and model preparation. These workflows carry the expectation of on-demand specialized resources that are not only available, but that can coordinate closely towards optimized ML outcomes. Iteration needs are near-term (for example, during exploratory analysis and model development), as well as long-term (for example, to correct data drift, or to add a new ML feature).

Kubeflow Pipelines enable AI/ML engineers to define the structure of their workflows using Python, for [pipelines][pipeline] that are executed on a Kubernetes cluster. Therefore, KFP combines the power of Python for experimentation and development, with the power of Kubernetes for resources and execution. This can translate to accelerated production-level AI/ML development, and ultimately to better AI/ML product outcomes. Some benefits include:

- **Structured workflow management**: organize and structure ML workflows, ensuring clarity and maintainability
- **ML experimentation and iteration**: enable modification and quick iterations while ensuring repeatable and consistent structure
- **Reproducibility and versioning**: support for versioning and run history
- **Simplified execution**: abstract away Kubernetes commands and YAML specifications, accessing lower levels when needed
- **Optimization and efficiency**: support caching, parallel execution, retries, and exit handling
- **Enterprise features**: support on-demand scaling, metadata and artifact tracking, and pipeline sharing
- **ML-first SDK and platform design** support ML experimentation, development, and production needs

## What Does a Pipeline Declaration Consist Of?
A KFP [pipeline][pipeline] consists of the following key elements:
### 1. Input Parameters
- **name, description, display_name** - pipeline-level metadata parameters
- **pipeline_root** - object storage root path for metadata and artifact persistence (see [Pipeline Root][Pipeline Root])
- **functional parameters** - parameters to be passed as inputs to component steps at runtime

### 2. Component Tasks
- **[component][component] tasks** - define the individual pipeline steps that will run in containers

### 3. Control Logic
- **flow logic** - sequential, parallel (see [Control Flow][control-flow])
- **conditional logic** - if, elif, else (see [Control Flow/Conditions][Conditions])
- **parameter passing** - support for small values of types like strings, numbers, lists, dicts, bool (see [Pass small amounts of data between components][Param Passing])
- **artifact inputs and outputs** - datasets, models, markdown, HTML, metrics (see [Create, use, pass, and track ML artifacts][Artifact Handling])
- **exit handling** - handles exit tasks, even if upstream tasks fail (see [Control Flow/Exit Handling][Exit Handling])

### 4. Runtime Logic
- **caching** - cached outputs can be used per task (see [Caching][Caching])
- **retries** - set retries for tasks
- **resource requests and limits** - request memory, cpu, GPU
- **node selectors** - request task containers to run on specific nodes
- **environment variables** - pass environment variables to task containers

## Basic Pipeline Construction

The recommended way to define a [pipeline][pipeline] is using the [KFP Python SDK][KFP SDK].

Steps:
1. Define individual components:
    ```python
    # my_components.py
    from kfp.dsl import component

    @component()
    def add(a: float, b: float) -> float:
        '''add two numbers'''
        return a + b

    @component()
    def multiply(a: float, b: float) -> float:
        '''multiply two numbers'''
        return a * b
    ```

2. Compose components in a pipeline:
    ```python
    # my_pipeline.py
    import my_components
    from kfp.dsl import pipeline

    @pipeline()
    def arithmetic_pipeline(a: float, b: float):
        step1 = my_components.add(a, b)
        step2 = my_component.multiply(step1.output, 2.0)


    if __name__ == "__main__":
        from kfp.compiler import Compiler

        Compiler().compile(
            pipeline_func=arithmetic_pipeline,
            package_path="my_pipeline.yaml"
        )
    ```

Step 1 defines two basic components, which are saved in `my_components.py`. Step 2 defines a simple pipeline that takes `float` inputs `a` and `b`, adds them in the `add` component, and then gets the product (`multiply` component) of the output from the `add` step with the number 2.0.

A few things to note from the above construction are:
- Since `step2` depends on the output of `step1`, the KFP backend will know to run the steps sequentially (when no dependency exists, steps run in parallel by default).
- Steps `step1` and `step2` will run in independent containers, and KFP will take care of transferring the needed parameter.
- The pipeline function, `arithmetic_pipeline`, is not run directly by the user, but rather compiled to [IR YAML][IR YAML]. Running the pipeline function will not run the steps. The compiler translates the pipeline and component Python instructions into instructions the KFP backend can understand.

The KFP Python SDK client (`kfp.client.Client`) supports submitting pipeline runs to the KFP backend either directly (`create_run_from_pipeline_func`), or from the compiled pipeline YAML file (`create_run_from_pipeline_package`). The KFP UI also supports running pipelines once their YAML files have been uploaded (either from the GUI or Python client interface).

The KFP documentation includes more detailed examples exploring pipeline implementation and runs. See the [Getting Started Guide][Getting Started] to quickly test out running a pipeline on a KFP cluster. For a more in-depth treatment of pipeline implementation, visit the [User Guide][User Guide] section of the documentation, including sections on [Core Functions][Core Functions] and [Data Handling][Data Handling].

## Next steps
* Read an [overview of Kubeflow Pipelines][overview].
* Follow the [pipelines quickstart guide][Getting Started]
  to deploy Kubeflow and run a sample pipeline directly from the Kubeflow
  Pipelines UI.


[overview]: ../overview.md
[pipeline]: pipeline.md
[component]: component.md
[graph]: graph.md
[step]: step.md
[run]: run.md
[KFP SDK]: https://kubeflow-pipelines.readthedocs.io
[IR YAML]: ir-yaml.md
[Getting Started]: ../getting-started.md
[User Guide]: ../user-guides/index.md
[Data Handling]: ../user-guides/data-handling/parameters.md
[Core Functions]: ../user-guides/core-functions/index.md
[Run a Pipeline]: ../user-guides/core-functions/run-a-pipeline.md
[Compile a Pipeline]: ../user-guides/core-functions/compile-a-pipeline.md
[Param Passing]: ../user-guides/data-handling/parameters.md
[Artifact Handling]: ../user-guides/data-handling/artifacts.md
[Exit Handling]: ../user-guides/core-functions/control-flow.md#exit-handling
[Conditions]: ../user-guides/core-functions/control-flow.md#conditions
[Caching]: ../user-guides/core-functions/caching.md
[Pipeline Root]: pipeline-root.md
[control-flow]: ../user-guides/core-functions/control-flow.md
