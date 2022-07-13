## Contributing to the `kfp` SDK
KFP v2 for SDK is under development on the `master` branch.

To contribute to v1, please go to [sdk/release-1.8 branch](https://github.com/kubeflow/pipelines/tree/sdk/release-1.8) to submit your change.

For general contribution guidelines including pull request conventions, see [pipelines/CONTRIBUTING.md](https://github.com/kubeflow/pipelines/blob/master/CONTRIBUTING.md).

### Requirements
All development requirement versions are pinned in [requirements-dev.txt](https://github.com/kubeflow/pipelines/blob/master/sdk/python/requirements-dev.txt).

### Testing
We suggest running unit tests using [`pytest`](https://docs.pytest.org/en/7.1.x/). From the project root, the following runs all KFP SDK unit tests:
```sh
pytest sdk/python
```

To run tests in parallel for faster execution, you can run the tests using the `pytest-xdist` plugin:

```sh
pytest sdk/python -n auto
```

### Code Style
Dependencies for code style checks/changes can be found in the kfp SDK [requirements-dev.txt](https://github.com/kubeflow/pipelines/blob/master/sdk/python/requirements-dev.txt).


#### Style Guide [Required]
The KFP SDK follows the [Google Python Style Guide](https://google.github.io/styleguide/pyguide.html).

#### Formatting [Required]
Please format your code using [yapf](https://github.com/google/yapf) according to the [`.style.yapf`](https://github.com/kubeflow/pipelines/blob/master/.style.yapf) file.

From the project root, run the following code to format your code:
```sh
yapf --in-place --recursive ./sdk/python
```

#### Docformatter [Required]
We encourage you to lint your docstrings using [docformatter](https://github.com/PyCQA/docformatter).

From the project root, run the following code to lint your docstrings:
```sh
docformatter --in-place --recursive ./sdk/python
```

#### Formatting Imports [Required]
Please organize your imports using [isort](https://pycqa.github.io/isort/index.html) according to the [`.isort.cfg`](https://github.com/kubeflow/pipelines/blob/master/.isort.cfg) file.

From the project root, run the following code to format your code:
```sh
isort sdk/python --sg sdk/python/kfp/deprecated
```

#### Pylint [Encouraged]
We encourage you to lint your code using [pylint](https://pylint.org/) according to the project [`.pylintrc`](https://github.com/kubeflow/pipelines/blob/master/.pylintrc) file.

From the project root, run the following code to lint your code:
```sh
pylint ./sdk/python/kfp
```

Note: `kfp` is not currently fully pylint-compliant. Consider substituting the path argument with the files touched by your development.

#### Static Type Checking [Encouraged]
Please use [mypy](https://mypy.readthedocs.io/en/stable/) to check your type annotations.

From the project root, run the following code to lint your docstrings:
```sh
mypy ./sdk/python/kfp/
```
Note: `kfp` is not currently fully mypy-compliant. Consider substituting the path argument with the files touched by your development.


#### Pre-commit [Recommended]
Consider using [`pre-commit`](https://github.com/pre-commit/pre-commit) with the provided [`.pre-commit-config.yaml`](https://github.com/kubeflow/pipelines/blob/master/.pre-commit-config.yaml) to implement the above changes:
```sh
pre-commit install
```
