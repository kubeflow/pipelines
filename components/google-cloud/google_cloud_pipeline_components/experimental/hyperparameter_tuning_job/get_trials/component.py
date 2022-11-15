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


def get_trials(gcp_resources: str) -> list:
  """Retrieves the best trial from the trials.

  Args:
      gcp_resources (str): Proto tracking the hyperparameter tuning job.

  Returns:
      List of strings representing the intermediate JSON representation of the
      trials from the hyperparameter tuning job.
  """
  import json
  from google.cloud import aiplatform
  from google.cloud.aiplatform_v1.types import study

  api_endpoint_suffix = '-aiplatform.googleapis.com'
  gcp_resources_obj = json.loads(gcp_resources)
  gcp_resources_split = gcp_resources_obj["resources"][0]["resourceUri"].partition(
      'projects')
  resource_name = gcp_resources_split[1] + gcp_resources_split[2]
  prefix_str = gcp_resources_split[0]
  prefix_str = prefix_str[:prefix_str.find(api_endpoint_suffix)]
  api_endpoint = prefix_str[(prefix_str.rfind('//') + 2):] + api_endpoint_suffix

  client_options = {'api_endpoint': api_endpoint}
  job_client = aiplatform.gapic.JobServiceClient(client_options=client_options)
  response = job_client.get_hyperparameter_tuning_job(name=resource_name)

  return [study.Trial.to_json(trial) for trial in response.trials]


if __name__ == "__main__":
  if "BUILD_WORKSPACE_DIRECTORY" in os.environ:
    rel_path = re.sub(".*/(?=third_party/)", "", os.path.dirname(__file__))
    os.chdir(os.path.join(os.environ["BUILD_WORKSPACE_DIRECTORY"], rel_path))

  get_trials_op = components.create_component_from_func(
      get_trials,
      base_image="python:3.10",
      packages_to_install=[
          "google-cloud-aiplatform==1.18.3",
      ],
      output_component_file="component.yaml",
  )
