# How to Contribute

Thank you for your interest in contributing to our project. Whether it's a bug report, new feature, correction, or additional documentation, we greatly value feedback and contributions from our community.

Please read through this document before submitting any issues or pull requests to ensure we have all the necessary information to effectively respond to your bug report or contribution.

## Coding Style

We follow the [`black`](https://black.readthedocs.io/en/stable/) format for all of our Python code, and require that all contributions are formatted using the `black` formatter before submission. Also, it is encouraged to lint Python docstrings by [`docformatter`](https://github.com/myint/docformatter).

### Automatic Code Formatting

Both `black` and `docformatter` provide CLIs for automatically formatting all relevant files. To use these tools, first follow the necessary setup as defined in the [Developer Setup](#developer-setup) section. Then, in the root of the component run the following commands:

```
black .
docformatter -r . --in-place
```

## Developer Setup

To set up your development environment, we have included all necessary Python pip requirements in the `requirements.txt` and `dev_requirements.txt` files. We recommend that you install these in either a `conda` or `pyenv` environment. The current version of Python that the project uses is `Python 3.7`. The `PYTHONPATH` for this project should be set at the root of the component (`components/aws/sagemaker`).

### Code Generators

We rely on a code generator to build our `component.yaml` files. After making modifications to any component, navigate to the root of the component (`components/aws/sagemaker`) and run the generator by using the following command:

```
./common/generate_components.py --tag {DOCKER TAG} [--image {DOCKER IMAGE URI}]
```

## Testing

For information regarding testing, please see the appropriate `README` files within the `tests` directory.