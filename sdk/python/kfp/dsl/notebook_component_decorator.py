"""Decorator for creating notebook-based components.

Uses the existing Python executor. The embedded notebook is executed by
a helper bound to `dsl.run_notebook(**kwargs)` that users can call
inside their component function.
"""

from __future__ import annotations

import functools
from typing import Any, Callable, List, Optional

from kfp.dsl import component_factory
from kfp.dsl.component_task_config import TaskConfigField
from kfp.dsl.component_task_config import TaskConfigPassthrough


def notebook_component(
    func: Optional[Callable[..., Any]] = None,
    *,
    notebook_path: str,
    base_image: Optional[str] = None,
    packages_to_install: Optional[List[str]] = None,
    output_component_file: Optional[str] = None,
    pip_index_urls: Optional[List[str]] = None,
    pip_trusted_hosts: Optional[List[str]] = None,
    use_venv: bool = False,
    kfp_package_path: Optional[str] = None,
    install_kfp_package: bool = True,
    task_config_passthroughs: Optional[List[TaskConfigPassthrough]] = None,
    use_local_pip_config: bool = False,
):
    """Decorator to define a Notebook-based KFP component.

    Args:
        notebook_path: Path to the .ipynb file or a directory containing one to embed and execute.
        base_image: Base container image for the component.
        packages_to_install: Runtime-only packages to install inside the
            component container. When None, defaults to
            ["nbclient>=0.10,<1", "ipykernel>=6,<7", "jupyter_client>=7,<9"]
            to ensure the notebook can execute with a Python kernel and client.
            When [], installs nothing. When non-empty, installs the exact list.
        output_component_file: Optional path to write the component YAML.
        pip_index_urls: Optional pip index URLs for installation.
        pip_trusted_hosts: Optional pip trusted hosts.
        use_venv: Whether to create and use a venv inside the container.
        kfp_package_path: Optional KFP package path to install.
        install_kfp_package: Whether to auto-install KFP when appropriate.
        task_config_passthroughs: Optional task config passthroughs.
        use_local_pip_config: Whether to inherit safe pip configuration settings from the user's local pip.conf/pip.ini.
            When True, safe options (index-url, extra-index-url, trusted-host, timeout, retries, no-cache-dir,
            disable-pip-version-check) are extracted and applied. Explicit parameters take precedence over local config.
            Defaults to False.

    Parameter injection and execution:
        - The notebook bytes are embedded in the component. At runtime, the
          helper binds `dsl.run_notebook(**kwargs)`.
        - Inside the decorated function body, call `dsl.run_notebook(...)` with
          keyword arguments for parameters you want available in the notebook
          (e.g., `dsl.run_notebook(text=my_text)`).
        - Parameters are injected following Papermill semantics: if the
          notebook contains a code cell tagged `parameters`, an overriding
          `injected-parameters` cell is inserted immediately after it; otherwise
          the injected cell is placed at the top of the notebook before
          execution.
        - Notebook outputs you want to expose as KFP outputs should be written
          by the notebook to known paths, then copied or logged by the function
          after `dsl.run_notebook(...)` returns.
    """

    formatted_passthroughs = None
    if task_config_passthroughs is not None:
        formatted_passthroughs = [
            TaskConfigPassthrough(field=p, apply_to_task=False) if isinstance(
                p, TaskConfigField) else p for p in task_config_passthroughs
        ]

    if func is None:
        return functools.partial(
            component_factory.create_notebook_component_from_func,
            notebook_path=notebook_path,
            base_image=base_image,
            packages_to_install=packages_to_install,
            output_component_file=output_component_file,
            pip_index_urls=pip_index_urls,
            pip_trusted_hosts=pip_trusted_hosts,
            use_venv=use_venv,
            kfp_package_path=kfp_package_path,
            install_kfp_package=install_kfp_package,
            task_config_passthroughs=formatted_passthroughs,
            use_local_pip_config=use_local_pip_config,
        )

    return component_factory.create_notebook_component_from_func(
        func=func,
        notebook_path=notebook_path,
        base_image=base_image,
        packages_to_install=packages_to_install,
        output_component_file=output_component_file,
        pip_index_urls=pip_index_urls,
        pip_trusted_hosts=pip_trusted_hosts,
        use_venv=use_venv,
        kfp_package_path=kfp_package_path,
        install_kfp_package=install_kfp_package,
        task_config_passthroughs=formatted_passthroughs,
        use_local_pip_config=use_local_pip_config,
    )
