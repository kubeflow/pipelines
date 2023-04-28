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
from kfp import dsl

# The artifact property key for the resource name
ARTIFACT_PROPERTY_KEY_RESOURCE_NAME = 'resourceName'


def google_artifact(type_name):
  'Set v2 artifact schema_title and schema_version attributes.'

  def add_type_name(cls):
    cls.schema_title = type_name
    cls.schema_version = '0.0.1'
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
        metadata={ARTIFACT_PROPERTY_KEY_RESOURCE_NAME: model_resource_name},
    )


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
        metadata={ARTIFACT_PROPERTY_KEY_RESOURCE_NAME: endpoint_resource_name},
    )


@google_artifact('google.VertexBatchPredictionJob')
class VertexBatchPredictionJob(dsl.Artifact):
  """An artifact representing a Vertex BatchPredictionJob."""

  def __init__(
      self,
      name: str,
      uri: str,
      job_resource_name: str,
      bigquery_output_table: Optional[str] = None,
      bigquery_output_dataset: Optional[str] = None,
      gcs_output_directory: Optional[str] = None,
  ):
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
            'gcsOutputDirectory': gcs_output_directory,
        },
    )


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
        metadata={ARTIFACT_PROPERTY_KEY_RESOURCE_NAME: dataset_resource_name},
    )


@google_artifact('google.BQMLModel')
class BQMLModel(dsl.Artifact):
  """An artifact representing a BQML Model."""

  def __init__(
      self, name: str, project_id: str, dataset_id: str, model_id: str
  ):
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
            'modelId': model_id,
        },
    )


@google_artifact('google.BQTable')
class BQTable(dsl.Artifact):
  """An artifact representing a BQ Table."""

  def __init__(
      self, name: str, project_id: str, dataset_id: str, table_id: str
  ):
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
            'tableId': table_id,
        },
    )


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
    super().__init__(
        metadata={
            'predictSchemata': predict_schemata,
            'containerSpec': container_spec,
        }
    )


@google_artifact('google.ClassificationMetrics')
class _ClassificationMetrics(dsl.Metrics):
  """An artifact representing evaluation classification metrics."""

  def __init__(
      self,
      name: str = 'evaluation_metrics',
      recall: Optional[float] = None,
      precision: Optional[float] = None,
      f1_score: Optional[float] = None,
      accuracy: Optional[float] = None,
      au_prc: Optional[float] = None,
      au_roc: Optional[float] = None,
      log_loss: Optional[float] = None,
  ):
    """Args:

    recall: Recall (True Positive Rate) for the given confidence threshold.
    precision: Precision for the given confidence threshold.
    f1_score: The harmonic mean of recall and precision.
    accuracy: Accuracy is the fraction of predictions given the correct label.
    au_prc: The Area Under Precision-Recall Curve metric.
    au_roc: The Area Under Receiver Operating Characteristic curve metric.
    log_loss: The Log Loss metric.
    """
    metadata = {}
    if recall is not None:
      metadata['recall'] = recall
    if precision is not None:
      metadata['precision'] = precision
    if f1_score is not None:
      metadata['f1Score'] = f1_score
    if accuracy is not None:
      metadata['accuracy'] = accuracy
    if au_prc is not None:
      metadata['auPrc'] = au_prc
    if au_roc is not None:
      metadata['auRoc'] = au_roc
    if log_loss is not None:
      metadata['logLoss'] = log_loss
    super().__init__(
        name=name,
        metadata=metadata,
    )


@google_artifact('google.RegressionMetrics')
class _RegressionMetrics(dsl.Metrics):
  """An artifact representing evaluation regression metrics."""

  def __init__(
      self,
      name: str = 'evaluation_metrics',
      root_mean_squared_error: Optional[float] = None,
      mean_absolute_error: Optional[float] = None,
      mean_absolute_percentage_error: Optional[float] = None,
      r_squared: Optional[float] = None,
      root_mean_squared_log_error: Optional[float] = None,
  ):
    """Args:

    root_mean_squared_error: Root Mean Squared Error (RMSE).
    mean_absolute_error: Mean Absolute Error (MAE).
    mean_absolute_percentage_error: Mean absolute percentage error.
    r_squared: Coefficient of determination as Pearson correlation coefficient.
    root_mean_squared_log_error: Root mean squared log error.
    """
    metadata = {}
    if root_mean_squared_error is not None:
      metadata['rootMeanSquaredError'] = root_mean_squared_error
    if mean_absolute_error is not None:
      metadata['meanAbsoluteError'] = mean_absolute_error
    if mean_absolute_percentage_error is not None:
      metadata['meanAbsolutePercentageError'] = mean_absolute_percentage_error
    if r_squared is not None:
      metadata['rSquared'] = r_squared
    if root_mean_squared_log_error is not None:
      metadata['rootMeanSquaredLogError'] = root_mean_squared_log_error
    super().__init__(
        name=name,
        metadata=metadata,
    )


@google_artifact('google.ForecastingMetrics')
class _ForecastingMetrics(dsl.Metrics):
  """An artifact representing evaluation forecasting metrics."""

  def __init__(
      self,
      name: str = 'evaluation_metrics',
      root_mean_squared_error: Optional[float] = None,
      mean_absolute_error: Optional[float] = None,
      mean_absolute_percentage_error: Optional[float] = None,
      r_squared: Optional[float] = None,
      root_mean_squared_log_error: Optional[float] = None,
      weighted_absolute_percentage_error: Optional[float] = None,
      root_mean_squared_percentage_error: Optional[float] = None,
      symmetric_mean_absolute_percentage_error: Optional[float] = None,
  ):
    """Args:

    root_mean_squared_error: Root Mean Squared Error (RMSE).
    mean_absolute_error: Mean Absolute Error (MAE).
    mean_absolute_percentage_error: Mean absolute percentage error.
    r_squared: Coefficient of determination as Pearson correlation coefficient.
    root_mean_squared_log_error: Root mean squared log error.
    weighted_absolute_percentage_error: Weighted Absolute Percentage Error.
      Does not use weights, this is just what the metric is called.
      Undefined if actual values sum to zero.
      Will be very large if actual values sum to a very small number.
    root_mean_squared_percentage_error: Root Mean Square Percentage Error.
      Square root of MSPE.
      Undefined/imaginary when MSPE is negative.
    symmetric_mean_absolute_percentage_error: Symmetric Mean Absolute Percentage
      Error.
    """
    metadata = {}
    if root_mean_squared_error is not None:
      metadata['rootMeanSquaredError'] = root_mean_squared_error
    if mean_absolute_error is not None:
      metadata['meanAbsoluteError'] = mean_absolute_error
    if mean_absolute_percentage_error is not None:
      metadata['meanAbsolutePercentageError'] = mean_absolute_percentage_error
    if r_squared is not None:
      metadata['rSquared'] = r_squared
    if root_mean_squared_log_error is not None:
      metadata['rootMeanSquaredLogError'] = root_mean_squared_log_error
    if weighted_absolute_percentage_error is not None:
      metadata['weightedAbsolutePercentageError'] = (
          weighted_absolute_percentage_error
      )
    if root_mean_squared_percentage_error is not None:
      metadata['rootMeanSquaredPercentageError'] = (
          root_mean_squared_percentage_error
      )
    if symmetric_mean_absolute_percentage_error is not None:
      metadata['symmetricMeanAbsolutePercentageError'] = (
          symmetric_mean_absolute_percentage_error
      )
    super().__init__(
        name=name,
        metadata=metadata,
    )
