# Copyright 2021 Google LLC
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
"""Utilities functions KFP DSL."""

import re


def sanitize_component_name(name: str):
  """Sanitizes component name."""
  return 'comp-{}'.format(_sanitize_name(name))


def sanitize_task_name(name: str):
  """Sanitizes task name."""
  return 'task-{}'.format(_sanitize_name(name))


def sanitize_executor_label(label: str):
  """Sanitizes executor label."""
  return 'exec-{}'.format(_sanitize_name(label))


def _sanitize_name(name: str):
  """Sanitizes name to comply with IR naming convention.

  The sanitized name contains only lower case alphanumeric characters and
  dashes.
  """
  return re.sub('-+', '-', re.sub('[^-0-9a-z]+', '-',
                                  name.lower())).lstrip('-').rstrip('-')
