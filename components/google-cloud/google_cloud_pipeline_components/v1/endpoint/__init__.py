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
# fmt: off
"""Manage model serving endpoints via [Vertex AI Endpoints](https://cloud.google.com/vertex-ai/docs/predictions/overview?_ga=2.161419069.-1686833729.1684288907#model_deployment)."""
# fmt: on

from google_cloud_pipeline_components.v1.endpoint.create_endpoint.component import endpoint_create as EndpointCreateOp
from google_cloud_pipeline_components.v1.endpoint.delete_endpoint.component import endpoint_delete as EndpointDeleteOp
from google_cloud_pipeline_components.v1.endpoint.deploy_model.component import model_deploy as ModelDeployOp
from google_cloud_pipeline_components.v1.endpoint.undeploy_model.component import model_undeploy as ModelUndeployOp

__all__ = [
    'EndpointCreateOp',
    'EndpointDeleteOp',
    'ModelDeployOp',
    'ModelUndeployOp',
]
