# Copyright 2026 The Kubeflow Authors
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
"""Kubeflow PipelinesClient — simplified, name-first KFP interface.

Can be used directly via ``from kfp.kubeflow_client import
PipelinesClient`` or through the Kubeflow SDK re-export at
``kubeflow.pipelines``.
"""

from kfp.kubeflow_client import constants
from kfp.kubeflow_client.pipelines_client import PipelinesBackendConfig
from kfp.kubeflow_client.pipelines_client import PipelinesClient
from kfp.kubeflow_client.types import Experiment
from kfp.kubeflow_client.types import ListExperimentsResponse
from kfp.kubeflow_client.types import ListPipelinesResponse
from kfp.kubeflow_client.types import ListPipelineVersionsResponse
from kfp.kubeflow_client.types import ListRunsResponse
from kfp.kubeflow_client.types import Pipeline
from kfp.kubeflow_client.types import PipelineVersion
from kfp.kubeflow_client.types import Run

__all__ = [
    'constants',
    'Experiment',
    'ListExperimentsResponse',
    'ListPipelinesResponse',
    'ListPipelineVersionsResponse',
    'ListRunsResponse',
    'Pipeline',
    'PipelinesBackendConfig',
    'PipelinesClient',
    'PipelineVersion',
    'Run',
]
