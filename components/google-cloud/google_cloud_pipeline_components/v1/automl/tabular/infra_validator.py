# Copyright 2023 The Kubeflow Authors. All Rights Reserved.
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

"""AutoML Infra Validator component spec."""

from google_cloud_pipeline_components.types.artifact_types import UnmanagedContainerModel
from kfp import dsl
from kfp.dsl import Input


@dsl.container_component
def automl_tabular_infra_validator(
    unmanaged_container_model: Input[UnmanagedContainerModel],  # pylint: disable=unused-argument
):
  # fmt: off
  """Validates the trained AutoML Tabular model is a valid model.

  Args:
      unmanaged_container_model: google.UnmanagedContainerModel for model to be validated.
  """
  # fmt: on

  return dsl.ContainerSpec(
      image='us-docker.pkg.dev/vertex-ai/automl-tabular/prediction-server:20240710_0625',
      command=[],
      args=['--executor_input', '{{$}}'],
  )
