# Component Specification

This specification describes the container component data model for Kubeflow
Pipelines. The data model is serialized to a file in YAML format for sharing.

Below are the main parts of the component definition:

* **Metadata:** Name, description, and other metadata.
* **Interface (inputs and outputs):** Name, type, default value.
* **Implementation:** How to run the component, given the input arguments.

## Example of a component specification

A component specification takes the form of a YAML file, `component.yaml`. Below
is an example:

```yaml
name: xgboost4j - Train classifier
description: Trains a boosted tree ensemble classifier using xgboost4j

inputs:
- {name: Training data}
- {name: Rounds, type: Integer, default: '30', description: 'Number of training rounds'}

outputs:
- {name: Trained model, type: XGBoost model, description: 'Trained XGBoost model'}

implementation:
  container:
    image: gcr.io/ml-pipeline/xgboost-classifier-train@sha256:b3a64d57
    command: [
      /ml/train.py,
      --train-set, {inputPath: Training data},
      --rounds,    {inputValue: Rounds},
      --out-model, {outputPath: Trained model},
    ]
```

See some examples of real-world
[component specifications](https://github.com/search?q=repo%3Akubeflow%2Fpipelines+path%3A**%2Fcomponent.yaml&type=code).

## Detailed specification (ComponentSpec)

This section describes the
[ComponentSpec](https://github.com/kubeflow/pipelines/blob/sdk/release-1.8/sdk/python/kfp/components/_structures.py).

### Metadata

* `name`: Human-readable name of the component.
* `description`: Description of the component.
* `metadata`: Standard object's metadata:

    * `annotations`: A string key-value map used to add information about the component.
        Currently, the annotations get translated to Kubernetes annotations when the component task is executed on Kubernetes. Current limitation: the key cannot contain more that one slash ("/"). See more information in the
        [Kubernetes user guide](https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/).
    * `labels`: Deprecated. Use `annotations`.

### Interface

* `inputs` and `outputs`:
    Specifies the list of inputs/outputs and their properties. Each input or
    output has the following properties:

    * `name`: Human-readable name of the input/output. Name must be
        unique inside the inputs or outputs section, but an output may have the
        same name as an input.
    * `description`: Human-readable description of the input/output.
    * `default`: Specifies the default value for an input. **Only
        valid for inputs.**
    * `type`: Specifies the type of input/output. The types are used
        as hints for pipeline authors and can be used by the pipeline system/UI
        to validate arguments and connections between components. Basic types
        are **String**, **Integer**, **Float**, and **Bool**. See the full list
        of [types](https://github.com/kubeflow/pipelines/blob/sdk/release-1.8/sdk/python/kfp/dsl/types.py)
        defined by the Kubeflow Pipelines SDK.
    * `optional`: Specifies if input is optional or not. This is of type
        **Bool**, and defaults to **False**. **Only valid for inputs.**

### Implementation

* `implementation`: Specifies how to execute the component instance.
    There are two implementation types,  `container` and `graph`. (The latter is
    not in scope for this document.) In future we may introduce more
    implementation types like `daemon` or `K8sResource`.

    * `container`:
        Describes the Docker container that implements the component. A portable
        subset of the Kubernetes
        [Container v1 spec](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#container-v1-core).

        * `image`: Name of the Docker image.
        * `command`: Entrypoint array. The Docker image's
            ENTRYPOINT is used if this is not provided. Each item is either a
            string or a placeholder. The most common placeholders are
            `{inputValue: Input name}`, `{inputPath: Input name}` and `{outputPath: Output name}`.
        * `args`: Arguments to the entrypoint. The Docker
            image's CMD is used if this is not provided. Each item is either a
            string or a placeholder. The most common placeholders are
            `{inputValue: Input name}`, `{inputPath: Input name}` and `{outputPath: Output name}`.
        * `env`: Map of environment variables to set in the container.
        * `fileOutputs`: Legacy property that is only needed in
            cases where the container always stores the output data in some
            hard-coded non-configurable local location. This property specifies
            a map between some outputs and local file paths where the program
            writes the output data files. Only needed for components that have
            hard-coded output paths. Such containers need to be fixed by
            modifying the program or adding a wrapper script that copies the
            output to a configurable location. Otherwise the component may be
            incompatible with future storage systems.

You can set all other Kubernetes container properties when you
use the component inside a pipeline.

## Using placeholders for command-line arguments

### Consuming input by value

The `{inputValue: <Input name>}` placeholder is replaced by the value of the input argument:

* In `component.yaml`:

  ```yaml
  command: [program.py, --rounds, {inputValue: Rounds}]
  ```

* In the pipeline code:

  ```python
  task1 = component1(rounds=150)
  ```

* Resulting command-line code (showing the value of the input argument that
  has replaced the placeholder):

  ```shell
  program.py --rounds 150
  ```

### Consuming input by file

The `{inputPath: <Input name>}` placeholder is replaced by the (auto-generated) local file path where the system has put the argument data passed for the "Input name" input.

* In `component.yaml`:

  ```yaml
  command: [program.py, --train-set, {inputPath: training_data}]
  ```

* In the pipeline code:

  ```python
  task2 = component1(training_data=some_task1.outputs['some_data'])
  ```

* Resulting command-line code (the placeholder is replaced by the
  generated path):

  ```shell
  program.py --train-set /inputs/train_data/data
  ```


### Producing outputs

The `{outputPath: <Output name>}` placeholder is replaced by a (generated) local file path where the component program is supposed to write the output data.
The parent directories of the path may or may not exist. Your
program must handle both cases without error.

* In `component.yaml`:

  ```yaml
  command: [program.py, --out-model, {outputPath: trained_model}]
  ```

* In the pipeline code:

  ```python
  task1 = component1()
  # You can now pass `task1.outputs['trained_model']` to other components as argument.
  ```

* Resulting command-line code (the placeholder is replaced by the
  generated path):

  ```shell
  program.py --out-model /outputs/trained_model/data
  ```
