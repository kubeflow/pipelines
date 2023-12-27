# Copyright 2023 The Kubeflow Authors
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
"""Implementation of the subprocess runner."""

import contextlib
import os
import subprocess
import sys
import tempfile
from typing import List
import venv
import warnings

from kfp.dsl import component_factory
from kfp.local import config
from kfp.local import status
from kfp.local import task_handler_interface


class SubprocessTaskHandler(task_handler_interface.ITaskHandler):
    """The task handler corresponding to kfp.local.SubprocessRunner."""

    def __init__(
        self,
        image: str,
        full_command: List[str],
        pipeline_root: str,
        runner: config.SubprocessRunner,
    ) -> None:
        self.validate_image(image)
        self.validate_not_container_component(full_command)
        self.validate_not_containerized_python_component(full_command)

        self.image = image
        self.full_command = full_command
        self.pipeline_root = pipeline_root
        self.runner = runner

    def run(self) -> status.Status:
        """Runs the local subprocess and returns the status.

        Returns:
            Status.
        """
        with environment(use_venv=self.runner.use_venv) as py_executable:
            full_command = replace_python_executable(
                self.full_command,
                py_executable,
            )
            return_code = run_local_subprocess(full_command=full_command)
            return status.Status.SUCCESS if return_code == 0 else status.Status.FAILURE

    def validate_image(self, image: str) -> None:
        if 'python' not in image:
            warnings.warn(
                f"You may be attemping to run a task that uses custom or non-Python base image '{image}' in a Python environment. This may result in incorrect dependencies and/or incorrect behavior. Consider using the 'DockerRunner' to run this task in a container.",
                RuntimeWarning,
            )

    def validate_not_container_component(
        self,
        full_command: List[str],
    ) -> None:
        if not any(component_factory.EXECUTOR_MODULE in part
                   for part in full_command):
            raise RuntimeError(
                f'The {config.SubprocessRunner.__name__} only supports running Lightweight Python Components. You are attempting to run a Container Component.'
            )

    def validate_not_containerized_python_component(
        self,
        full_command: List[str],
    ) -> None:
        if full_command[:len(
                component_factory.CONTAINERIZED_PYTHON_COMPONENT_COMMAND
        )] == component_factory.CONTAINERIZED_PYTHON_COMPONENT_COMMAND:
            raise RuntimeError(
                f'The {config.SubprocessRunner.__name__} only supports running Lightweight Python Components. You are attempting to run a Containerized Python Component.'
            )


def run_local_subprocess(full_command: List[str]) -> int:
    with subprocess.Popen(
            full_command,
            stdout=subprocess.PIPE,
            # No change to behavior in terminal for user,
            # but inner process logs redirected to stdout. This separates from
            # the outer process logs which, per logging module default, go to
            # stderr.
            stderr=subprocess.STDOUT,
            text=True,
            # buffer line-by-line
            bufsize=1,
    ) as process:
        if process.stdout:
            for line in iter(process.stdout.readline, ''):
                print(line, end='')

            # help with visual separation to show termination of subprocess logs
            print('\n')

        return process.wait()


def replace_python_executable(full_command: List[str],
                              new_executable: str) -> List[str]:
    """Replaces the 'python3' string in each element of the full_command with
    the new_executable.

    Args:
        full_command: Commands and args.
        new_executable: The Python executable to use for local execution.

    Returns:
        The updated commands and args.
    """
    return [el.replace('python3', f'{new_executable}') for el in full_command]


@contextlib.contextmanager
def environment(use_venv: bool) -> str:
    """Context manager that handles the environment used for the subprocess.

    Args:
        use_venv: Whether to use the virtual environment instead of current environment.

    Returns:
        The Python executable path to use.
    """
    if use_venv:
        with tempfile.TemporaryDirectory() as tempdir:
            # Create the virtual environment inside the temporary directory
            venv.create(tempdir, with_pip=True)

            yield os.path.join(tempdir, 'bin', 'python')
    else:
        yield sys.executable
