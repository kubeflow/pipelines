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
        permitted_runners = (SubprocessRunner,)
        if not isinstance(runner, permitted_runners):
            raise ValueError(
                f'Got unknown runner {runner} of type {runner.__class__.__name__}. Runner should be one of the following types: {". ".join(prunner.__name__ for prunner in permitted_runners)}.'
            )
        self.runner = runner
        self.pipeline_root = pipeline_root
        self.raise_on_error = raise_on_error


def init(
    # more runner types will eventually be supported
    runner: SubprocessRunner,
    pipeline_root: str = './local_outputs',
    raise_on_error: bool = True,
) -> None:
    """Initializes a local execution session.

    Once called, components can be invoked locally outside of a pipeline definition.

    Args:
        runner: The runner to use. Currently only SubprocessRunner is supported.
        pipeline_root: Destination for task outputs.
        raise_on_error: If True, raises an exception when a local task execution fails. If Falls, fails gracefully and does not terminal the current program.
    """
    # updates a global config
    LocalExecutionConfig(
        runner=runner,
        pipeline_root=pipeline_root,
        raise_on_error=raise_on_error,
    )
