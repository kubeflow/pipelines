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


def google_artifact(type_name):
  "Decorator for Google Artifact types for handling KFP v1/v2 artifact types"
  def add_type_name(cls):
    if hasattr(dsl.Artifact, 'schema_title'):
      cls.schema_title = type_name
      cls.schema_version = '0.0.1'
    else:
      cls.TYPE_NAME = type_name
    return cls
  return add_type_name

@google_artifact('google.VertexModel')
class VertexModel(dsl.Artifact):
  """An artifact representing a Vertex Model."""

  def __init__(self, name: str, uri: str, model_resource_name: str):
    """Args:

         name: The artifact name.
         uri: the Vertex Model resource uri, in a form of
         https://{service-endpoint}/v1/projects/{project}/locations/{location}/models/{model},
         where
         {service-endpoint} is one of the supported service endpoints at
         https://cloud.google.com/vertex-ai/docs/reference/rest#rest_endpoints
         model_resource_name: The name of the Model resource, in a form of
         projects/{project}/locations/{location}/models/{model}. For
         more details, see
         https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.models/get
    """
    super().__init__(
        uri=uri,
        name=name,
        metadata={ARTIFACT_PROPERTY_KEY_RESOURCE_NAME: model_resource_name})


@google_artifact('google.VertexEndpoint')
class VertexEndpoint(dsl.Artifact):
  """An artifact representing a Vertex Endpoint."""

  def __init__(self, name: str, uri: str, endpoint_resource_name: str):
    """Args:

         name: The artifact name.
         uri: the Vertex Endpoint resource uri, in a form of
         https://{service-endpoint}/v1/projects/{project}/locations/{location}/endpoints/{endpoint},
         where
         {service-endpoint} is one of the supported service endpoints at
         https://cloud.google.com/vertex-ai/docs/reference/rest#rest_endpoints
         endpoint_resource_name: The name of the Endpoint resource, in a form of
         projects/{project}/locations/{location}/endpoints/{endpoint}. For
         more details, see
         https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.endpoints/get
    """
    super().__init__(
        uri=uri,
        name=name,
        metadata={ARTIFACT_PROPERTY_KEY_RESOURCE_NAME: endpoint_resource_name})


@google_artifact('google.VertexBatchPredictionJob')
class VertexBatchPredictionJob(dsl.Artifact):
  """An artifact representing a Vertex BatchPredictionJob."""

  def __init__(self,
               name: str,
               uri: str,
               job_resource_name: str,
               bigquery_output_table: Optional[str] = None,
               bigquery_output_dataset: Optional[str] = None,
               gcs_output_directory: Optional[str] = None):
    """Args:

         name: The artifact name.
         uri: the Vertex Batch Prediction resource uri, in a form of
         https://{service-endpoint}/v1/projects/{project}/locations/{location}/batchPredictionJobs/{batchPredictionJob},
         where {service-endpoint} is one of the supported service endpoints at
         https://cloud.google.com/vertex-ai/docs/reference/rest#rest_endpoints
         job_resource_name: The name of the batch prediction job resource,
         in a form of
         projects/{project}/locations/{location}/batchPredictionJobs/{batchPredictionJob}.
         For more details, see
         https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.batchPredictionJobs/get
         bigquery_output_table: The name of the BigQuery table created, in
         predictions_<timestamp> format, into which the prediction output is
         written. For more details, see
         https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.batchPredictionJobs#outputinfo
         bigquery_output_dataset: The path of the BigQuery dataset created, in
         bq://projectId.bqDatasetId format, into which the prediction output is
         written. For more details, see
         https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.batchPredictionJobs#outputinfo
         gcs_output_directory: The full path of the Cloud Storage directory
         created, into which the prediction output is written. For more details,
         see
         https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.batchPredictionJobs#outputinfo
    """
    super().__init__(
        uri=uri,
        name=name,
        metadata={
            ARTIFACT_PROPERTY_KEY_RESOURCE_NAME: job_resource_name,
            'bigqueryOutputTable': bigquery_output_table,
            'bigqueryOutputDataset': bigquery_output_dataset,
            'gcsOutputDirectory': gcs_output_directory
        })


@google_artifact('google.VertexDataset')
class VertexDataset(dsl.Artifact):
  """An artifact representing a Vertex Dataset."""

  def __init__(self, name: str, uri: str, dataset_resource_name: str):
    """Args:

         name: The artifact name.
         uri: the Vertex Dataset resource uri, in a form of
         https://{service-endpoint}/v1/projects/{project}/locations/{location}/datasets/{datasets_name},
         where
         {service-endpoint} is one of the supported service endpoints at
         https://cloud.google.com/vertex-ai/docs/reference/rest#rest_endpoints
         dataset_resource_name: The name of the Dataset resource, in a form of
         projects/{project}/locations/{location}/datasets/{datasets_name}. For
         more details, see
         https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.datasets/get
    """
    super().__init__(
        uri=uri,
        name=name,
        metadata={ARTIFACT_PROPERTY_KEY_RESOURCE_NAME: dataset_resource_name})


@google_artifact('google.BQMLModel')
class BQMLModel(dsl.Artifact):
  """An artifact representing a BQML Model."""

  def __init__(self, name: str, project_id: str, dataset_id: str,
               model_id: str):
    """Args:

         name: The artifact name.
         project_id: The ID of the project containing this model.
         dataset_id: The ID of the dataset containing this model.
         model_id: The ID of the model.

         For more details, see
         https://cloud.google.com/bigquery/docs/reference/rest/v2/models#ModelReference
    """
    super().__init__(
        uri=f'https://www.googleapis.com/bigquery/v2/projects/{project_id}/datasets/{dataset_id}/models/{model_id}',
        name=name,
        metadata={
            'projectId': project_id,
            'datasetId': dataset_id,
            'modelId': model_id
        })


@google_artifact('google.BQTable')
class BQTable(dsl.Artifact):
  """An artifact representing a BQ Table."""

  def __init__(self, name: str, project_id: str, dataset_id: str,
               table_id: str):
    """Args:

         name: The artifact name.
         project_id: The ID of the project containing this table.
         dataset_id: The ID of the dataset containing this table.
         table_id: The ID of the table.

         For more details, see
         https://cloud.google.com/bigquery/docs/reference/rest/v2/TableReference
    """
    super().__init__(
        uri=f'https://www.googleapis.com/bigquery/v2/projects/{project_id}/datasets/{dataset_id}/tables/{table_id}',
        name=name,
        metadata={
            'projectId': project_id,
            'datasetId': dataset_id,
            'tableId': table_id
        })


@google_artifact('google.UnmanagedContainerModel')
class UnmanagedContainerModel(dsl.Artifact):
  """An artifact representing an unmanaged container model."""

  def __init__(self, predict_schemata: Dict, container_spec: Dict):
    """Args:

         predict_schemata: Contains the schemata used in Model's predictions and
         explanations via PredictionService.Predict, PredictionService.Explain
         and BatchPredictionJob. For more details, see
         https://cloud.google.com/vertex-ai/docs/reference/rest/v1/PredictSchemata
         container_spec: Specification of a container for serving predictions.
         Some fields in this message correspond to fields in the Kubernetes
         Container v1 core specification. For more details, see
         https://cloud.google.com/vertex-ai/docs/reference/rest/v1/ModelContainerSpec
    """
    super().__init__(metadata={
        'predictSchemata': predict_schemata,
        'containerSpec': container_spec
    })
