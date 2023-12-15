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
"""An abstract base class that specifies the interface of a task handler."""
import abc
from typing import List

from kfp.local import config
from kfp.local import status


class ITaskHandler(abc.ABC):
    """Interface for a TaskHandler."""

    def __init__(
        self,
        image: str,
        full_command: List[str],
        pipeline_root: str,
        runner: config.LocalRunnerType,
    ) -> None:
        pass

    @abc.abstractmethod
    def run(self) -> status.Status:
        """Runs the task and returns the status.

        Returns:
            Status.
        """
        pass
