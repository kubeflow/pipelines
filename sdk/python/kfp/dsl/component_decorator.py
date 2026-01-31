# Copyright 2021-2022 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import functools
from typing import Callable, List, Optional, Union
import warnings

from kfp.dsl import component_factory
from kfp.dsl.component_task_config import TaskConfigField
from kfp.dsl.component_task_config import TaskConfigPassthrough


def component(
    func: Optional[Callable] = None,
    *,
    base_image: Optional[str] = None,
    target_image: Optional[str] = None,
    packages_to_install: List[str] = None,
    pip_index_urls: Optional[List[str]] = None,
    output_component_file: Optional[str] = None,
    install_kfp_package: bool = True,
    kfp_package_path: Optional[str] = None,
    pip_trusted_hosts: Optional[List[str]] = None,
    use_venv: bool = False,
    additional_funcs: Optional[List[Callable]] = None,
    embedded_artifact_path: Optional[str] = None,
    task_config_passthroughs: Optional[List[Union[TaskConfigPassthrough,
                                                  TaskConfigField]]] = None,
    use_local_pip_config: bool = False):
    """Decorator for Python-function based components.

    A KFP component can either be a lightweight component or a containerized
    component.

    If ``target_image`` is not specified, this function creates a lightweight
    component. A lightweight component is a self-contained Python function that
    includes all necessary imports and dependencies. In lightweight components,
    ``packages_to_install`` will be used to install dependencies at runtime. The
    parameters ``install_kfp_package`` and ``kfp_package_path`` can be used to control
    how and from where KFP should be installed when the lightweight component is executed.

    If ``target_image`` is specified, this function creates a component definition
    based around the ``target_image``. The assumption is that the function in ``func``
    will be packaged by KFP into this ``target_image``. You can use the KFP CLI's ``build``
    command to package the function into ``target_image``.

    Args:
        func: Python function from which to create a component. The function
            should have type annotations for all its arguments, indicating how
            each argument is intended to be used (e.g. as an input/output artifact,
            a plain parameter, or a path to a file).
        base_image: Image to use when executing the Python function. It should
            contain a default Python interpreter that is compatible with KFP.
        target_image: Image to use when creating containerized components.
        packages_to_install: List of packages to install before
            executing the Python function. These will always be installed at component runtime.
        pip_index_urls: Python Package Index base URLs from which to
            install ``packages_to_install``. Defaults to installing from only PyPI
            (``'https://pypi.org/simple'``). For more information, see `pip install docs <https://pip.pypa.io/en/stable/cli/pip_install/#cmdoption-0>`_.
        output_component_file: If specified, this function will write a
            shareable/loadable version of the component spec into this file.

            **Warning:** This compilation approach is deprecated.
        embedded_artifact_path: Optional path to a local file or directory to
            embed into the component. At runtime the embedded content is
            extracted into a temporary directory and made available via
            parameters annotated as ``dsl.EmbeddedInput[T]`` (e.g.,
            ``dsl.EmbeddedInput[dsl.Dataset]``). For a directory, the injected
            artifact's ``path`` points to the extraction root (the temporary
            directory containing the directory's contents). For a single file,
            the injected artifact's ``path`` points to the extracted file. The
            extraction root is also prepended to ``sys.path`` to enable
            importing embedded Python modules.
        install_kfp_package: Specifies if the KFP SDK should add the ``kfp`` Python package to
            ``packages_to_install``. Lightweight Python functions always require
            an installation of KFP in ``base_image`` to work. If you specify
            a ``base_image`` that already contains KFP, you can set this to ``False``.
            This flag is ignored when ``target_image`` is specified, which implies
            a choice to build a containerized component. Containerized components
            will always install KFP as part of the build process.
        kfp_package_path: Specifies the location from which to install KFP. By
            default, this will try to install from PyPI using the same version
            as that used when this component was created. Component authors can
            choose to override this to point to a GitHub pull request or
            other pip-compatible package server.
        use_venv: Specifies if the component should be executed in a virtual environment.
            The environment will be created in a temporary directory and will inherit the system site packages.
            This is useful in restricted environments where most of the system is read-only.
        additional_funcs: List of additional functions to include in the component.
            These functions will be available to the main function. This is useful for adding util functions that
            are shared across multiple components but are not packaged as an importable Python package.
        task_config_passthroughs: List of task configurations (e.g. resources, env, volumes etc.) to pass through
            to the component. This is useful when the component launches another Kubernetes resource (for example,
            a Kubeflow Trainer job). Use this in conjunction with dsl.TaskConfig.
        use_local_pip_config: Whether to inherit safe pip configuration settings from the user's local pip.conf/pip.ini.
            When True, safe options (index-url, extra-index-url, trusted-host, timeout, retries, no-cache-dir,
            disable-pip-version-check) are extracted and applied. Explicit parameters take precedence over local config.
            Defaults to False.

    Returns:
        A component task factory that can be used in pipeline definitions.

    Example:
      ::

        from kfp import dsl

        @dsl.component
        def my_function_one(input: str, output: Output[Model]):
            ...

        @dsl.component(
        base_image='python:3.11',
        output_component_file='my_function.yaml'
        )
        def my_function_two(input: Input[Mode])):
            ...

        @dsl.pipeline(name='my-pipeline', pipeline_root='...')
        def pipeline():
            my_function_one_task = my_function_one(input=...)
            my_function_two_task = my_function_two(input=my_function_one_task.outputs)
    """
    if output_component_file is not None:
        warnings.warn(
            'output_component_file parameter is deprecated and will eventually be removed. Please use `Compiler().compile()` to compile a component instead.',
            DeprecationWarning,
            stacklevel=2)

    task_config_passthroughs_formatted: Optional[
        List[TaskConfigPassthrough]] = None
    if task_config_passthroughs is not None:
        task_config_passthroughs_formatted = []

        for passthrough in task_config_passthroughs:
            if isinstance(passthrough, TaskConfigField):
                task_config_passthroughs_formatted.append(
                    TaskConfigPassthrough(
                        field=passthrough, apply_to_task=False))
            else:
                task_config_passthroughs_formatted.append(passthrough)

    if func is None:
        return functools.partial(
            component,
            base_image=base_image,
            target_image=target_image,
            packages_to_install=packages_to_install,
            embedded_artifact_path=embedded_artifact_path,
            pip_index_urls=pip_index_urls,
            output_component_file=output_component_file,
            install_kfp_package=install_kfp_package,
            kfp_package_path=kfp_package_path,
            pip_trusted_hosts=pip_trusted_hosts,
            use_venv=use_venv,
            additional_funcs=additional_funcs,
            task_config_passthroughs=task_config_passthroughs_formatted,
            use_local_pip_config=use_local_pip_config)

    return component_factory.create_component_from_func(
        func,
        base_image=base_image,
        target_image=target_image,
        embedded_artifact_path=embedded_artifact_path,
        packages_to_install=packages_to_install,
        pip_index_urls=pip_index_urls,
        output_component_file=output_component_file,
        install_kfp_package=install_kfp_package,
        kfp_package_path=kfp_package_path,
        pip_trusted_hosts=pip_trusted_hosts,
        use_venv=use_venv,
        additional_funcs=additional_funcs,
        task_config_passthroughs=task_config_passthroughs_formatted,
        use_local_pip_config=use_local_pip_config)
