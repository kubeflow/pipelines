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
"""Objects for configuring local execution."""
import abc
import dataclasses
import os
from typing import Union

from kfp import local


class LocalRunnerType(abc.ABC):
    """The ABC for user-facing Runner configurations.

    Subclasses should be a dataclass.

    They should implement a .validate() method.
    """

    @abc.abstractmethod
    def validate(self) -> None:
        """Validates that the configuration arguments provided by the user are
        valid."""
        raise NotImplementedError


@dataclasses.dataclass
class SubprocessRunner:
    """Runner that indicates that local tasks should be run in a subprocess.

    Args:
        use_venv: Whether to run the subprocess in a virtual environment. If True, dependencies will be installed in the virtual environment. If False, dependencies will be installed in the current environment. Using a virtual environment is recommended.
    """
    use_venv: bool = True


@dataclasses.dataclass
class DockerRunner:
    """Runner that indicates that local tasks should be run as a Docker
    container."""

    def __post_init__(self):
        try:
            import docker  # noqa
        except ImportError as e:
            raise ImportError(
                f"Package 'docker' must be installed to use {DockerRunner.__name__!r}. Install it using 'pip install docker'."
            ) from e


class LocalExecutionConfig:
    instance = None

    def __new__(
        cls,
        runner: SubprocessRunner,
        pipeline_root: str,
        raise_on_error: bool,
    ) -> 'LocalExecutionConfig':
        # singleton pattern
        cls.instance = super(LocalExecutionConfig, cls).__new__(cls)
        return cls.instance

    def __init__(
        self,
        runner: SubprocessRunner,
        pipeline_root: str,
        raise_on_error: bool,
    ) -> None:
        permitted_runners = (SubprocessRunner, DockerRunner)
        if not isinstance(runner, permitted_runners):
            raise ValueError(
                f'Got unknown runner {runner} of type {runner.__class__.__name__}. Runner should be one of the following types: {". ".join(prunner.__name__ for prunner in permitted_runners)}.'
            )
        self.runner = runner
        self.pipeline_root = pipeline_root
        self.raise_on_error = raise_on_error

    @classmethod
    def validate(cls):
        if cls.instance is None:
            raise RuntimeError(
                f"Local environment not initialized. Please run '{local.__name__}.{init.__name__}()' before executing tasks locally."
            )


def init(
    # annotate with subclasses, not parent class, for more helpful ref docs
    runner: Union[SubprocessRunner, DockerRunner],
    pipeline_root: str = './local_outputs',
    raise_on_error: bool = True,
) -> None:
    """Initializes a local execution session.

    Once called, components can be invoked locally outside of a pipeline definition.

    Args:
        runner: The runner to use. Supported runners: kfp.local.SubprocessRunner and kfp.local.DockerRunner.
        pipeline_root: Destination for task outputs.
        raise_on_error: If True, raises an exception when a local task execution fails. If False, fails gracefully and does not terminate the current program.
    """
    # updates a global config
    pipeline_root = os.path.abspath(pipeline_root)
    LocalExecutionConfig(
        runner=runner,
        pipeline_root=pipeline_root,
        raise_on_error=raise_on_error,
    )
