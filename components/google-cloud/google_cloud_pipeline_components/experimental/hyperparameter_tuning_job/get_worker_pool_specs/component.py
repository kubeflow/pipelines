# Copyright 2021 The Kubeflow Authors
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
"""Component supporting Google Vertex AI Hyperparameter Tuning Job Op."""

import os
import re
from kfp import components


def get_worker_pool_specs(best_hyperparameters: list,
                         worker_pool_specs: list) -> list:
  """Constructs worker_pool_specs based on the best hyperparameters.

  Args:
      best_hyperparameters (list): Required. List representing the intermediate
        JSON representation of the best hyperparameters from the
        hyperparameter tuning job.
      worker_pool_specs (list): Required. The spec of the worker pools
        including machine type and Docker image. All worker pools except the
        first one are optional and can be skipped by providing an empty value.

  Returns:
      List containing an intermediate JSON representation of the
      worker_pool_specs updated with the best hyperparameters as arguments
      in the container_spec.

  """
  from google.cloud.aiplatform_v1.types import study

  for worker_pool_spec in worker_pool_specs:
    if 'args' not in worker_pool_spec['container_spec']:
      worker_pool_spec['container_spec']['args'] = []
    for param in best_hyperparameters:
      p = study.Trial.Parameter.from_json(param)
      worker_pool_spec['container_spec']['args'].append(
          f'--{p.parameter_id}={p.value}')

  return worker_pool_specs


if __name__ == "__main__":
  if "BUILD_WORKSPACE_DIRECTORY" in os.environ:
    rel_path = re.sub(".*/(?=third_party/)", "", os.path.dirname(__file__))
    os.chdir(os.path.join(os.environ["BUILD_WORKSPACE_DIRECTORY"], rel_path))

  get_worker_pool_specs_op = components.create_component_from_func(
      get_worker_pool_specs,
      base_image="python:3.10",
      packages_to_install=["google-cloud-aiplatform==1.18.3"],
      output_component_file="component.yaml",
  )
