# Additional Functionality

## Component docstring format
KFP allows you to document your components and pipelines using Python docstrings. The KFP SDK automatically parses your docstrings and include certain fields in [IR YAML][ir-yaml] when you compile components and pipelines.

For components, KFP can extract your component **input descriptions** and **output descriptions**.

For pipelines, KFP can extract your pipeline **input descriptions** and **output descriptions**, as well as a **description of your full pipeline**.

For the KFP SDK to correctly parse your docstrings, you should write your docstrings in the KFP docstring style. The KFP docstring style is a particular variant on the [Google docstring style][google-docstring-style], with the following changes:
* The `Returns:` section takes the same structure as the `Args:` section, where each return value in the `Returns:` section should take the form `<name>: <description>`. This is distinct from the typical Google docstring `Returns:` section which takes the form `<type>: <description>`, with no names for return values.
* Component outputs should be included in the `Returns:` section, even though they are declared via component function input parameters. This applies to function parameters annotated with [`dsl.OutputPath`][dsl-outputpath] and the [`Output[<Artifact>]`][output-type-marker] type marker for declaring [output artifacts][output-artifacts].
* *Suggested:* Type information, including which inputs are optional/required, should be omitted from the input/output descriptions. This information is duplicative of the annotations.

For example, the KFP SDK can extract input and output descriptions from the following component docstring which uses the KFP docstring style:


```python
@dsl.component
def join_datasets(
    dataset_a: Input[Dataset],
    dataset_b: Input[Dataset],
    out_dataset: Output[Dataset],
) -> str:
    """Concatenates two datasets.

    Args:
        dataset_a: First dataset.
        dataset_b: Second dataset.

    Returns:
        out_dataset: The concatenated dataset.
        Output: The concatenated string.
    """
    ...
```

Similarly, KFP can extract the component input descriptions, the component output descriptions, and the pipeline description from the following pipeline docstring:

```python
@dsl.pipeline(display_name='Concatenation pipeline')
def dataset_concatenator(
    string: str,
    in_dataset: Input[Dataset],
) -> Dataset:
    """Pipeline to convert string to a Dataset, then concatenate with
    in_dataset.

    Args:
        string: String to concatenate to in_artifact.
        in_dataset: Dataset to which to concatenate string.

    Returns:
        Output: The final concatenated dataset.
    """
    ...
```

Note that if you provide a `description` argument to the [`@dsl.pipeline`][dsl-pipeline] decorator, KFP will use this description instead of the docstring description.

[ir-yaml]: ../../concepts/ir-yaml.md
[google-docstring-style]: https://sphinxcontrib-napoleon.readthedocs.io/en/latest/example_google.html
[dsl-pipeline]: https://kubeflow-pipelines.readthedocs.io/en/stable/source/dsl.html#kfp.dsl.pipeline
[output-artifacts]: ../data-handling/artifacts.md
[dsl-outputpath]: https://kubeflow-pipelines.readthedocs.io/en/latest/source/dsl.html#kfp.dsl.OutputPath
[output-type-marker]: https://kubeflow-pipelines.readthedocs.io/en/latest/source/dsl.html#kfp.dsl.Output
