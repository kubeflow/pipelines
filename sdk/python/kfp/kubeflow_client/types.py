# Copyright The Kubeflow Authors
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
"""Type aliases for PipelinesClient.

These aliases provide clean names over the auto-generated kfp.server_api
model classes used by the KFP backend API.
"""

import kfp.server_api

__all__ = [
    'Pipeline',
    'PipelineVersion',
    'Run',
    'Experiment',
    'ListPipelinesResponse',
    'ListPipelineVersionsResponse',
    'ListRunsResponse',
    'ListExperimentsResponse',
]

Pipeline = kfp.server_api.V2beta1Pipeline
PipelineVersion = kfp.server_api.V2beta1PipelineVersion
Run = kfp.server_api.V2beta1Run
Experiment = kfp.server_api.V2beta1Experiment

ListPipelinesResponse = kfp.server_api.V2beta1ListPipelinesResponse
ListPipelineVersionsResponse = (
    kfp.server_api.V2beta1ListPipelineVersionsResponse)
ListRunsResponse = kfp.server_api.V2beta1ListRunsResponse
ListExperimentsResponse = kfp.server_api.V2beta1ListExperimentsResponse
