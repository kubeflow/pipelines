# Notebook Python Components

Create a component directly from a Jupyter notebook file. The decorator embeds a .ipynb file into the component and executes it at runtime via nbclient.

```python
@dsl.notebook_component(
    notebook_path="train.ipynb",
    packages_to_install=["pandas", "scikit-learn", "nbclient"],
)
def evaluate_model(dataset_uri: str, model: dsl.Output[dsl.Model]):
    dsl.run_notebook(
        dataset_uri=dataset_uri,
        output_model_path=model.path,
    )
```
## Python function requirements
Python notebook components have the same requirements of [lightweight Python components][lightweight-python-components] , but they allow a user to embed a Jupyter notebook into the component to be executed at runtime.

### dsl.notebook_component decorator arguments
The `@dsl.notebook_component` decorator takes the following arguments, some of which it shares in common with the [`dsl.component`][lightweight-python-components] decorator:

#### notebook_path

`notebook_path` contains the path to the .ipynb file or a directory containing an .ipynb file to embed and execute. This field is required.

```python
@dsl.notebook_component(
    notebook_path="train.ipynb",)
def evaluate_model(dataset_uri: str, model: dsl.Output[dsl.Model]):
    dsl.run_notebook(
        dataset_uri=dataset_uri,
        output_model_path=model.path,
    )
```

#### base_image

When you create a Python Notebook component, your Python function code is extracted by the KFP SDK to be executed inside a container at pipeline runtime. By default, the container image used is [`python:3.11`](https://hub.docker.com/_/python). You can override this image by providing an argument to `base_image`. This can be useful if your code requires a specific Python version or other dependencies not included in the default image.

#### packages_to_install

Most components will depend on other Python libraries. You can pass a list of requirements to `packages_to_install` and the component will install these packages at runtime before executing the component function.

This is similar to including requirements in a [`requirements.txt`][requirements-txt] file.

#### output_component_file

If specified, this function will write a shareable/loadable version of the component spec YAML into this file.

#### pip_index_urls

`pip_index_urls` exposes the ability to pip install `packages_to_install` from package indices other than the default [PyPI.org][pypi-org].

When you set `pip_index_urls`, KFP passes these indices to [`pip install`][pip-install]'s [`--index-url`][pip-index-url] and [`--extra-index-url`][pip-extra-index-url] options. It also sets each index as a `--trusted-host`.

Take the following component:

```python
@dsl.notebook_component(
    notebook_path="train.ipynb",
    packages_to_install=['custom-ml-package==0.0.1'],
    pip_index_urls=['http://myprivaterepo.com/simple'])
def evaluate_model(dataset_uri: str):
    from custom_ml_package import model_trainer
    dsl.run_notebook(
        dataset_uri=dataset_uri,
        output_model_path=model_trainer.path,
    )
```
#### pip_trusted_hosts

`pip_trusted_hosts` is set with optional pip trusted hosts.

#### kfp_package_path

`kfp_package_path` specifies the location from which to install KFP. By default, this will try to install from PyPI using the same version as that used when this component was created. Component authors can choose to override this to point to a GitHub pull request or other pip-compatible package server.

#### install_kfp_package

`install_kfp_package` can be used together with `pip_index_urls` to provide granular control over installation of the `kfp` package at component runtime.

By default, Python Components install `kfp` at runtime. This is required to define symbols used by your component (such as [artifact annotations][artifacts]) and to access additional KFP library code required to execute your component remotely. If `install_kfp_package` is `False`, `kfp` will not be installed via the normal automatic mechanism. Instead, you can use `packages_to_install` and `pip_index_urls` to install a different version of `kfp`, possibly from a non-default pip index URL.

Note that setting `install_kfp_package` to `False` is rarely necessary and is discouraged for the majority of use cases.

#### use_venv

`use_venv` is used to specify if the component should be executed in a virtual environment. If so, the environment will be created in a temporary directory and will inherit the system site packages.

This is useful in restricted environments where most of the system is read-only.

```python
@dsl.notebook_component(
    notebook_path="train.ipynb",
    pip_index_urls=["https://pypi.org/simple"],
    pip_trusted_hosts=['pypi.org', 'pypi.org:8888'],
    use_venv=True,
)
def evaluate_model(input: int):
    import math
    dsl.run_notebook(input=math.sqrt(input))

```

#### task_config_passthroughs

`task_config_passthroughs` can be used to pass a list of one or more task configurations (e.g. resources, env, volumes etc.) through to the component.

This is useful when the component launches another Kubernetes resource (e.g. a Kubeflow Trainer job). Use `task_config_passthroughs` in conjunction with `dsl.TaskConfig`.

```python
@dsl.notebook_component(
    task_config_passthroughs=[
        dsl.TaskConfigPassthrough(field=dsl.TaskConfigField.ENV, apply_to_task=True),
    ],
)
def train(task_config: dsl.TaskConfig):
    dsl.run_notebook(task_config.env)

```

[lightweight-python-components]: lightweight-python-components.md
[artifacts]: ../data-handling/artifacts.md
[requirements-txt]: https://pip.pypa.io/en/stable/reference/requirements-file-format/
[pypi-org]: https://pypi.org/
[pip-install]: https://pip.pypa.io/en/stable/cli/pip_install/
[pip-index-url]: https://pip.pypa.io/en/stable/cli/pip_install/#cmdoption-0
[pip-extra-index-url]: https://pip.pypa.io/en/stable/cli/pip_install/#cmdoption-extra-index-url
