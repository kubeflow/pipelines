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
"""Google Cloud Pipeline Endpoint components."""

import os
from kfp import components
from .create_endpoint import component as create_endpoint_component
from .deploy_model import component as deploy_model_component

__all__ = [
    'EndpointCreateOp',
    'EndpointDeleteOp',
    'ModelDeployOp',
    'ModelUndeployOp',
]

EndpointCreateOp = create_endpoint_component.endpoint_create
EndpointDeleteOp = components.load_component_from_file(
    os.path.join(os.path.dirname(__file__), 'delete_endpoint/component.yaml')
)
ModelDeployOp = deploy_model_component.model_deploy
ModelUndeployOp = components.load_component_from_file(
    os.path.join(os.path.dirname(__file__), 'undeploy_model/component.yaml')
)
