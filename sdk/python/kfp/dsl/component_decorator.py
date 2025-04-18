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
from typing import Callable, List, Optional
import warnings

from kfp.dsl import component_factory


def component(func: Optional[Callable] = None,
              *,
              base_image: Optional[str] = None,
              target_image: Optional[str] = None,
              packages_to_install: List[str] = None,
              pip_index_urls: Optional[List[str]] = None,
              output_component_file: Optional[str] = None,
              install_kfp_package: bool = True,
              kfp_package_path: Optional[str] = None,
              pip_trusted_hosts: Optional[List[str]] = None,
              use_venv: bool = False):
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

    Returns:
        A component task factory that can be used in pipeline definitions.

    Example:
      ::

        from kfp import dsl

        @dsl.component
        def my_function_one(input: str, output: Output[Model]):
            ...

        @dsl.component(
        base_image='python:3.9',
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

    if func is None:
        return functools.partial(
            component,
            base_image=base_image,
            target_image=target_image,
            packages_to_install=packages_to_install,
            pip_index_urls=pip_index_urls,
            output_component_file=output_component_file,
            install_kfp_package=install_kfp_package,
            kfp_package_path=kfp_package_path,
            pip_trusted_hosts=pip_trusted_hosts,
            use_venv=use_venv)

    return component_factory.create_component_from_func(
        func,
        base_image=base_image,
        target_image=target_image,
        packages_to_install=packages_to_install,
        pip_index_urls=pip_index_urls,
        output_component_file=output_component_file,
        install_kfp_package=install_kfp_package,
        kfp_package_path=kfp_package_path,
        pip_trusted_hosts=pip_trusted_hosts,
        use_venv=use_venv)
