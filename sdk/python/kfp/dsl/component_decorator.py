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
from typing import Callable, List, Optional, TypeVar, Union
import warnings

from kfp.dsl import component_factory
from kfp.dsl.component_task_config import TaskConfigField
from kfp.dsl.component_task_config import TaskConfigPassthrough
from typing_extensions import ParamSpec

P = ParamSpec("P")
R = TypeVar("R")


def component(
    func: Optional[Callable[P, R]] = None,
    *,
    base_image: Optional[str] = None,
    target_image: Optional[str] = None,
    packages_to_install: Optional[List[str]] = None,
    pip_index_urls: Optional[List[str]] = None,
    output_component_file: Optional[str] = None,
    install_kfp_package: bool = True,
    kfp_package_path: Optional[str] = None,
    pip_trusted_hosts: Optional[List[str]] = None,
    use_venv: bool = False,
    additional_funcs: Optional[List[Callable]] = None,
    embedded_artifact_path: Optional[str] = None,
    task_config_passthroughs: Optional[List[Union[TaskConfigPassthrough,
                                                  TaskConfigField]]] = None
) -> Callable[P, R]:
    """Decorator for Python-function based components with improved typing."""

    if output_component_file is not None:
        warnings.warn(
            'output_component_file parameter is deprecated. Use Compiler().compile() instead.',
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
            task_config_passthroughs=task_config_passthroughs_formatted)

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
        task_config_passthroughs=task_config_passthroughs_formatted)
