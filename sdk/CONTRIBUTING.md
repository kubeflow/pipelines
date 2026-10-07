## Contributing to the `kfp` SDK

For developing the KFP SDK, use the `master` branch.

For general contribution guidelines including pull request conventions, see [pipelines/CONTRIBUTING.md](https://github.com/kubeflow/pipelines/blob/master/CONTRIBUTING.md).

### Pre-requisites

Clone the repo:

```bash
git clone https://github.com/kubeflow/pipelines.git && cd pipelines
```

Install [uv](https://docs.astral.sh/uv/), a fast Python package manager:

```bash
curl -LsSf https://astral.sh/uv/install.sh | sh
```

For supported python versions, see the [pyproject.toml](https://github.com/kubeflow/pipelines/blob/master/sdk/python/pyproject.toml).

### Development Setup

Install the unified SDK in editable mode with development dependencies:

```bash
uv sync --extra dev
```

This command will:
- Install `kfp`, including pipeline-spec, Kubernetes configuration, and the server API client
- Install development dependencies (pytest, linters, type checkers, etc.)
- Create or update the `uv.lock` lockfile for reproducible builds

Before running tests, generate the proto files:

```bash
make -C sdk generate-python
```

CI retains explicit generation before building and testing. Generated bindings
are committed so installing directly from Git also works; regenerate rather than
editing them. `make -C sdk python` runs all Python generators and builds the single
wheel and sdist. Both artifacts contain their generated modules; installing a
release or rebuilding its sdist does not require Docker, Java, or protoc.

### Testing
We suggest running unit tests using [`pytest`](https://docs.pytest.org/en/7.1.x/). From the project root, the following runs all KFP SDK unit tests:
```sh
uv run pytest
```

To run tests in parallel for faster execution, you can run the tests using the `pytest-xdist` plugin:

```sh
uv run pytest -n auto
```

#### Runtime command regressions

`test/presubmit-tests-sdk.sh` builds a fresh SDK wheel and runs the runtime
regressions against it in isolated uv environments. Other regression cases keep
their Git source override so Docker containers can install the SDK.

For a direct runtime-only run, build with `uv build --package kfp --wheel`,
set `KFP_PACKAGE_PATH` to the absolute path of the resulting `kfp-*.whl`, and run
`uv run pytest sdk/python/test/runtime -m regression`. These tests require the
wheel; they never fall back to the published SDK. The generated `--no-deps`
installation and `_KFP_RUNTIME=true` executor invocation remain unchanged.
The package inventory is compared before and after bootstrap: only `kfp` may
be added, independently of the installer dependencies uv seeds for each Python
version.

### Code Style
Dependencies for code style checks/changes are managed in [pyproject.toml](https://github.com/kubeflow/pipelines/blob/master/pyproject.toml) via the `dev`, `lint`, and `test` extras.


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

### Code Style

#### Formatting Imports [Required]
Please organize your imports using [isort](https://pycqa.github.io/isort/index.html) according to the [`.isort.cfg`](https://github.com/kubeflow/pipelines/blob/master/.isort.cfg) file.

From the project root, run the following code to format your code:
```sh
uv run isort sdk/python
```

#### Pylint [Encouraged]
We encourage you to lint your code using [pylint](https://pylint.org/) according to the project [`.pylintrc`](https://github.com/kubeflow/pipelines/blob/master/.pylintrc) file.

From the project root, run the following code to lint your code:
```sh
uv run pylint ./sdk/python/kfp
```

Note: `kfp` is not currently fully pylint-compliant. Consider substituting the path argument with the files touched by your development.

#### Static Type Checking [Encouraged]
Please use [mypy](https://mypy.readthedocs.io/en/stable/) to check your type annotations.

From the project root, run the following code to lint your docstrings:
```sh
uv run mypy ./sdk/python/kfp/
```
Note: `kfp` is not currently fully mypy-compliant. Consider substituting the path argument with the files touched by your development.


#### Pre-commit [Recommended]
Consider using [`pre-commit`](https://github.com/pre-commit/pre-commit) with the provided [`.pre-commit-config.yaml`](https://github.com/kubeflow/pipelines/blob/master/.pre-commit-config.yaml) to implement the above changes:
```sh
uv run pre-commit install
```
