# Copyright 2024 The Kubeflow Authors. All Rights Reserved.
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

"""Utilities for surface user defined error messages."""

import json
import os
from google.protobuf import json_format
from google_cloud_pipeline_components.proto import task_error_pb2


def write_customized_error(
    executor_input: str, error: task_error_pb2.TaskError
):
  """Writes a TaskError customized by the author of the pipelines to a JSON file ('executor_error.json') in the output directory specified in the executor input.

  Args:
    executor_input: JSON string containing executor input data.
    error: TaskError protocol buffer message.
  """
  executor_input_json = json.loads(executor_input)
  os.makedirs(
      os.path.dirname(executor_input_json['outputs']['outputFile']),
      exist_ok=True,
  )
  executor_out_path = executor_input_json['outputs']['outputFile']
  directory_path = os.path.dirname(executor_out_path)
  executor_error_path = os.path.join(directory_path, 'executor_error.json')
  error_dict = json_format.MessageToDict(error)
  with open(
      executor_error_path,
      'w',
  ) as f:
    json.dump(error_dict, f)
