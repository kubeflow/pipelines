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
from typing import List

from google_cloud_pipeline_components import _image
from google_cloud_pipeline_components import _placeholders
from kfp.dsl import container_component
from kfp.dsl import ContainerSpec
from kfp.dsl import OutputPath


@container_component
def dataflow_python(
    python_module_path: str,
    temp_location: str,
    gcp_resources: OutputPath(str),
    location: str = 'us-central1',
    requirements_file_path: str = '',
    args: List[str] = [],
    project: str = _placeholders.PROJECT_ID_PLACEHOLDER,
):
  # fmt: off
  """Launch a self-executing Beam Python file on Google Cloud using the
  Dataflow Runner.

  Args:
      location: Location of the Dataflow job. If not set, defaults to `'us-central1'`.
      python_module_path: The GCS path to the Python file to run.
      temp_location: A GCS path for Dataflow to stage temporary job files created during the execution of the pipeline.
      requirements_file_path: The GCS path to the pip requirements file.
      args: The list of args to pass to the Python file. Can include additional parameters for the Dataflow Runner.
      project: Project to create the Dataflow job. Defaults to the project in which the PipelineJob is run.

  Returns:
      gcp_resources: Serialized gcp_resources proto tracking the Dataflow job. For more details, see https://github.com/kubeflow/pipelines/blob/master/components/google-cloud/google_cloud_pipeline_components/proto/README.md.
  """
  # fmt: on
  return ContainerSpec(
      image=_image.GCPC_IMAGE_TAG,
      command=[
          'python3',
          '-u',
          '-m',
          'google_cloud_pipeline_components.container.v1.dataflow.dataflow_launcher',
      ],
      args=[
          '--project',
          project,
          '--location',
          location,
          '--python_module_path',
          python_module_path,
          '--temp_location',
          temp_location,
          '--requirements_file_path',
          requirements_file_path,
          '--args',
          args,
          '--gcp_resources',
          gcp_resources,
      ],
  )
