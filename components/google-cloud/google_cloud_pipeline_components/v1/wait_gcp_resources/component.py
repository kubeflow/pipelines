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
  """Waits for the completion of one or more GCP resources by polling for completion statuses.

  Currently this component only supports waiting on a `DataflowJob <https://cloud.google.com/config-connector/docs/reference/resource-docs/dataflow/dataflowjob>`_ resource.

  To use this component, first create a component that outputs a ``gcp_resources`` proto as JSON, then pass it to this component's ``gcp_resources`` parameter.

  See `details <https://github.com/kubeflow/pipelines/tree/master/components/google-cloud/google_cloud_pipeline_components/proto>`_ on how to create a ``gcp_resources`` proto as a component output.

  Examples:
    ::

      dataflow_python_op = gcpc.v1.dataflow.LaunchPythonOp(
          python_file_path=...
      )

      dataflow_wait_op = WaitGcpResourcesOp(
          gcp_resources=dataflow_python_op.outputs["gcp_resources"]
      )

  Args:
    gcp_resources: Serialized JSON of ``gcp_resources`` proto, indicating the resource(s) this component should wait on.

  Returns:
    gcp_resources: The ``gcp_resource``, including any relevant error information.

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
