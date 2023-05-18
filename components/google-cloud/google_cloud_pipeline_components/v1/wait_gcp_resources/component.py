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
from google_cloud_pipeline_components import _image
from google_cloud_pipeline_components import utils
from kfp import dsl
from kfp.dsl import OutputPath


@utils.gcpc_output_name_converter('gcp_resources')
@dsl.container_component
def wait_gcp_resources(
    gcp_resources: str,
    output__gcp_resources: OutputPath(str),
):
  # fmt: off
  """Receives a GCP Resource, polling the resource status and waits for it to
  finish. Currently this component only support waiting on a DataflowJob
  resource.

  To use this component, first create a component that outputs a JSON formatted gcp_resources proto, then pass it to the wait component.

  dataflow_python_op = gcpc.v1.dataflow.LaunchPythonOp(
      python_file_path = ...
  )

  dataflow_wait_op = gcpc.v1.wait_gcp_resources.WaitGcp_ResourcesOp(
      gcp_resources = dataflow_python_op.outputs["gcp_resources"]
  )

  For details on how to create a Json serialized gcp_resources proto as output, see
  https://github.com/kubeflow/pipelines/tree/master/components/google-cloud/google_cloud_pipeline_components/proto


  Args:
    gcp_resources: Serialized JSON of gcp_resources proto, indicating the resource to wait on by this component
        For details, see https://github.com/kubeflow/pipelines/tree/master/components/google-cloud/google_cloud_pipeline_components/proto

  Returns:
    gcp_resources: The final result of the gcp resource, including the error information, if exists.
  """
  # fmt: on
  return dsl.ContainerSpec(
      image=_image.GCPC_IMAGE_TAG,
      command=[
          'python3',
          '-u',
          '-m',
          'google_cloud_pipeline_components.container.v1.wait_gcp_resources.launcher',
      ],
      args=[
          '--type',
          'Wait',
          '--project',
          '',
          '--location',
          '',
          '--payload',
          gcp_resources,
          '--gcp_resources',
          output__gcp_resources,
      ],
  )
