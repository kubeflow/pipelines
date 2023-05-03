# Copyright 2021 The Kubeflow Authors. All Rights Reserved.
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
"""Google Cloud Pipeline wait GCP resource component."""

import os
from kfp.components import load_component_from_file

__all__ = [
    'WaitGcpResourcesOp',
]

WaitGcpResourcesOp = load_component_from_file(
    os.path.join(os.path.dirname(__file__), 'component.yaml')
)
"""
Receives a GCP Resource, polling the resource status and waits for it to finish.
Currently this component only support waiting on a DataflowJob resource.

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
  gcp_resources (str):
      Serialized JSON of gcp_resources proto, indicating the resource to wait on by this component
      For details, see https://github.com/kubeflow/pipelines/tree/master/components/google-cloud/google_cloud_pipeline_components/proto

Returns:
  gcp_resources (str):
      The final result of the gcp resource, including the error information, if exists.
"""
