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
"""Classes for ML Metadata input/output Artifacts for tracking Google resources."""

from typing import Dict, Optional
from kfp.v2 import dsl
import json

# The artifact property key for the resource name
ARTIFACT_PROPERTY_KEY_RESOURCE_NAME = 'resourceName'


class GoogleArtifact(dsl.Artifact):
  # Converts the artifact into executor output artifact
  # https://github.com/kubeflow/pipelines/blob/master/api/v2alpha1/pipeline_spec.proto#L878
  def to_executor_output_artifact(self, executor_input: str):
    executor_input_json = json.loads(executor_input)
    for name, artifacts in executor_input_json.get('outputs',
                                                   {}).get('artifacts',
                                                           {}).items():
      artifacts_list = artifacts.get('artifacts')
      if name == self.name and artifacts_list:
        updated_runtime_artifact = artifacts_list[0]
        updated_runtime_artifact['uri'] = self.uri
        updated_runtime_artifact['metadata'] = self.metadata
        artifacts_list = {'artifacts': [updated_runtime_artifact]}
    return {self.name: artifacts_list}


class VertexModel(GoogleArtifact):
  """An artifact representing a Vertex Model."""
  TYPE_NAME = 'google.VertexModel'

  def __init__(self, name: str, uri: str, model_resource_name: str):
    super().__init__(
        uri=uri,
        name=name,
        metadata={ARTIFACT_PROPERTY_KEY_RESOURCE_NAME: model_resource_name})


class VertexEndpoint(GoogleArtifact):
  """An artifact representing a Vertex Endpoint."""
  TYPE_NAME = 'google.VertexEndpoint'

  def __init__(self, name: str, uri: str, endpoint_resource_name: str):
    super().__init__(
        uri=uri,
        name=name,
        metadata={ARTIFACT_PROPERTY_KEY_RESOURCE_NAME: endpoint_resource_name})


class VertexBatchPredictionJob(GoogleArtifact):
  """An artifact representing a Vertex BatchPredictionJob."""
  TYPE_NAME = 'google.VertexBatchPredictionJob'

  def __init__(self,
               name: str,
               uri: str,
               job_name: str,
               bigquery_output_table: Optional[str] = None,
               bigquery_output_dataset: Optional[str] = None,
               gcs_output_directory: Optional[str] = None):
    super().__init__(
        uri=uri,
        name=name,
        metadata={
            ARTIFACT_PROPERTY_KEY_RESOURCE_NAME: job_name,
            'bigqueryOutputTable': bigquery_output_table,
            'bigqueryOutputDataset': bigquery_output_dataset,
            'gcsOutputDirectory': gcs_output_directory
        })


class VertexDataset(GoogleArtifact):
  """An artifact representing a Vertex Dataset."""
  TYPE_NAME = 'google.VertexDataset'

  def __init__(self,
               name: Optional[str] = None,
               uri: Optional[str] = None,
               metadata: Optional[Dict] = None):
    super().__init__(uri=uri, name=name, metadata=metadata)


class BQMLModel(GoogleArtifact):
  """An artifact representing a BQML Model."""
  TYPE_NAME = 'google.BQMLModel'

  def __init__(self, name: str, uri: str, project_id: str, dataset_id: str,
               model_id: str):
    super().__init__(
        uri=uri,
        name=name,
        metadata={
            'projectId': project_id,
            'datasetId': dataset_id,
            'modelId': model_id
        })


class BQTable(GoogleArtifact):
  """An artifact representing a BQ Table."""
  TYPE_NAME = 'google.BQTable'

  def __init__(self, name: str, uri: str, project_id: str, dataset_id: str,
               table_id: str):
    super().__init__(
        uri=uri,
        name=name,
        metadata={
            'projectId': project_id,
            'datasetId': dataset_id,
            'tableId': table_id
        })


class UnmanagedContainerModel(GoogleArtifact):
  """An artifact representing an unmanaged container model."""
  TYPE_NAME = 'google.UnmanagedContainerModel'

  def __init__(self, metadata: Optional[Dict] = None):
    super().__init__(metadata=metadata)
