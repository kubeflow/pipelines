# Copyright 2022 The Kubeflow Authors. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Utility function for processing gcp resource labels."""
import json
import logging
import os

SYSTEM_LABEL_ENV_VAR = "VERTEX_AI_PIPELINES_RUN_LABELS"


def attach_system_labels(existing_labels=None):
  """Attaches system labels to the given existing_labels map.

  Will combine
    - any labels from the system environmental variable
      `VERTEX_AI_PIPELINES_RUN_LABELS`, with
    - any labels from the given existing_labels map.

  For key conflict, will respect existing_labels (since they are user provided).

  Args:
    existing_labels: Optional[dict[str,str]]. If provided, will combine with the
    system labels read from the environmental variable.

  Returns:
    Optional[dict[str,str]] The combined labels, or None if existing_labels is
    None and system environmental variable `VERTEX_AI_PIPELINES_RUN_LABELS` is
    not set.
  """
  # Reads the system labels.
  system_labels = os.getenv(SYSTEM_LABEL_ENV_VAR)

  # If there is no system label presented, return existing_labels.
  if system_labels is None:
    return existing_labels

  # If the system label is malformated, warn and return existing_labels.
  try:
    labels = json.loads(system_labels)
  except json.decoder.JSONDecodeError:
    logging.warning("Failed to parse system labels: %s", system_labels)
    return existing_labels

  # If there is no existing_labels, return the system labels.
  if existing_labels is None:
    return labels

  # Combines system labels with existing labels. During key conflict, respect
  # the value in existing_labels.
  labels.update(existing_labels)
  return labels
