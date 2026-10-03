# Copyright 2019 The Kubeflow Authors. All Rights Reserved.
#
# Licensed under the Apache License,Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing,software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Supporting tools and classes for diagnose_me."""

import dataclasses
import json
import subprocess
from typing import Any, List, Optional


@dataclasses.dataclass
class ExecutorResponse:
    """Data model for the captured output of an executed command.

    This class is a pure data container, it does not execute anything. Use
    execute_command to run a command and obtain a populated instance.

    Attributes:
      stdout: Standard output captured from the executed command.
      stderr: Standard error captured from the executed command.
      return_code: Exit code of the executed command. This is None when the
        command could not be started at all and the underlying OSError did not
        carry an errno.
    """
    stdout: str = ''
    stderr: str = ''
    return_code: Optional[int] = None

    @property
    def parsed_output(self) -> Any:
        """Json load results of stdout or raw results if stdout was not
        Json."""
        try:
            return json.loads(self.stdout)
        except json.JSONDecodeError:
            return self.stdout

    @property
    def json_output(self) -> Any:
        """Alias for parsed_output, kept for backward compatibility."""
        return self.parsed_output

    @property
    def has_error(self) -> bool:
        """Returns true if execution error code was not 0."""
        return self.return_code != 0


def execute_command(command_list: List[str]) -> ExecutorResponse:
    """Executes the command in command_list.

    Args:
      command_list: A List of strings that represts the command and parameters
        to be executed.

    Returns:
      An ExecutorResponse populated with stdout, stderr and the return code.
    """
    try:
        process = subprocess.run(command_list, capture_output=True, check=False)
        return ExecutorResponse(
            stdout=process.stdout.decode('utf-8'),
            stderr=process.stderr.decode('utf-8'),
            return_code=process.returncode)
    except OSError as e:
        return ExecutorResponse(stdout='', stderr=str(e), return_code=e.errno)
