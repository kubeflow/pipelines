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


def get_hyperparameters(trial: str) -> list:
  """Retrieves the hyperparameters from the given trial.

  Args:
      trial (str): Required. The intermediate JSON representation of a
        hyperparameter tuning job trial.

  Returns:
      List representing the intermediate JSON representation of the
      hyperparameters from the trial.
  """
  from google.cloud.aiplatform_v1.types import study

  trial_proto = study.Trial.from_json(trial)

  return [
      study.Trial.Parameter.to_json(param) for param in trial_proto.parameters
  ]

if __name__ == "__main__":
  if "BUILD_WORKSPACE_DIRECTORY" in os.environ:
    rel_path = re.sub(".*/(?=third_party/)", "", os.path.dirname(__file__))
    os.chdir(os.path.join(os.environ["BUILD_WORKSPACE_DIRECTORY"], rel_path))

  get_hyperparameters_op = components.create_component_from_func(
      get_hyperparameters,
      base_image="python:3.10",
      packages_to_install=["google-cloud-aiplatform==1.18.3"],
      output_component_file="component.yaml",
  )
