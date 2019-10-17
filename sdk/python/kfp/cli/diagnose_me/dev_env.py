# Lint as: python3
# Copyright 2019 Google LLC. All Rights Reserved.
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
"""Functions for diagnostic data collection from development development."""

import enum
from typing import Optional
from . import utility


class Commands(enum.Enum):
  """Enum for gcloud and gsutil commands."""
  PIP3LIST = 1
  PYTHON3PIPLIST = 2
  PIP3VERSION = 3
  PYHYON3PIPVERSION = 4


_command_string = {
    Commands.PIP3LIST: 'pip3 list',
    Commands.PYTHON3PIPLIST: 'python3 -m pip list',
    Commands.PIP3VERSION: 'pip3 -V',
    Commands.PYHYON3PIPVERSION: 'python3 -m pip -V',
}


def get_dev_env_configuration(
    configuration: Commands,
    human_readable: Optional[bool] = False) -> utility.ExecutorResponse:
  """Captures the specified environment configuration.

  Captures the developement environment configuration including PIP version and
  Phython version as specifeid by configuration

  Args:
    configuration: Commands for specific information to be retrieved
      - PIP3LIST: captures pip3 freeze results
      - PYTHON3PIPLIST: captuers python3 -m pip freeze results
      - PIP3VERSION: captuers pip3 -V results
      - PYHYON3PIPVERSION: captuers python3 -m pip -V results
    human_readable: If true all output will be in human readable form insted of
      Json.

  Returns:
    A utility.ExecutorResponse with the output results for the specified
    command.
  """
  command_list = _command_string[configuration].split(' ')
  if not human_readable and configuration not in (Commands.PIP3VERSION,
                                                  Commands.PYHYON3PIPVERSION):
    command_list.extend(['--format', 'json'])

  return utility.ExecutorResponse().execute_command(command_list)
