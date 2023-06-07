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
"""Artifact types corresponding to Google Cloud Resources produced and consumed by GCPC components.

These artifact types can be used in your custom KFP SDK components similarly to
other `KFP SDK artifacts
<https://www.kubeflow.org/docs/components/pipelines/v2/data-types/artifacts/>`_.
If you wish to produce Google artifacts from your own components, it is
recommended that you use `Containerized Python Components
<https://www.kubeflow.org/docs/components/pipelines/v2/components/containerized-python-components/>`_.
You should assign metadata to the Google artifacts according to the artifact's
schema (provided by each artifact's ``.schema`` attribute).
"""


__all__ = [
    'VertexModel',
    'VertexEndpoint',
    'VertexBatchPredictionJob',
    'VertexDataset',
    'BQMLModel',
    'BQTable',
    'UnmanagedContainerModel',
    'ClassificationMetrics',
    'RegressionMetrics',
    'ForecastingMetrics',
]

import textwrap
from typing import Dict, Optional
from kfp import dsl

_RESOURCE_NAME_KEY = 'resourceName'


class VertexModel(dsl.Artifact):
  """An artifact representing a Vertex AI `Model resource <https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.models>`_."""
  schema_title = 'google.VertexModel'
  schema_version = '0.0.1'
  schema = textwrap.dedent("""\
title: google.VertexModel
type: object
properties:
  resourceName:
    type: string""")

  @classmethod
  def create(
      cls,
      name: str,
      uri: str,
      model_resource_name: str,
  ) -> 'VertexModel':
    """Create a VertexModel artifact instance.

    Args:
      name: The artifact name.
      uri: the Vertex Model resource uri, in a form of
      https://{service-endpoint}/v1/projects/{project}/locations/{location}/models/{model},
        where {service-endpoint} is one of the supported service endpoints at
      https://cloud.google.com/vertex-ai/docs/reference/rest#rest_endpoints
      model_resource_name: The name of the Model resource, in a form of
        projects/{project}/locations/{location}/models/{model}. For more
        details, see
      https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.models/get
    """
    return cls(
        name=name,
        uri=uri,
        metadata={_RESOURCE_NAME_KEY: model_resource_name},
    )


class VertexEndpoint(dsl.Artifact):
  """An artifact representing a Vertex AI `Endpoint resource <https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.endpoints>`_."""
  schema_title = 'google.VertexEndpoint'
  schema_version = '0.0.1'
  schema = textwrap.dedent("""\
title: google.VertexEndpoint
type: object
properties:
  resourceName:
    type: string""")

  @classmethod
  def create(
      cls,
      name: str,
      uri: str,
      endpoint_resource_name: str,
  ) -> 'VertexEndpoint':
    """Create a VertexEndpoint artifact instance.

    Args:
      name: The artifact name.
      uri: the Vertex Endpoint resource uri, in a form of
      https://{service-endpoint}/v1/projects/{project}/locations/{location}/endpoints/{endpoint},
        where {service-endpoint} is one of the supported service endpoints at
      https://cloud.google.com/vertex-ai/docs/reference/rest#rest_endpoints
      endpoint_resource_name: The name of the Endpoint resource, in a form of
        projects/{project}/locations/{location}/endpoints/{endpoint}. For more
        details, see
      https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.endpoints/get
    """
    return cls(
        name=name,
        uri=uri,
        metadata={_RESOURCE_NAME_KEY: endpoint_resource_name},
    )


class VertexBatchPredictionJob(dsl.Artifact):
  """An artifact representing a Vertex AI `BatchPredictionJob resource <https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.batchPredictionJobs#resource:-batchpredictionjob>`_."""
  schema_title = 'google.VertexBatchPredictionJob'
  schema_version = '0.0.1'
  schema = textwrap.dedent("""\
title: google.VertexBatchPredictionJob
type: object
properties:
  resourceName:
    type: string
  bigqueryOutputTable:
    type: string
  gcsOutputDirectory:
    type: string
  bigqueryOutputDataset:
    type: string""")

  @classmethod
  def create(
      cls,
      name: str,
      uri: str,
      job_resource_name: str,
      bigquery_output_table: Optional[str] = None,
      bigquery_output_dataset: Optional[str] = None,
      gcs_output_directory: Optional[str] = None,
  ) -> 'VertexBatchPredictionJob':
    """Create a VertexBatchPredictionJob artifact instance.

    Args:
      name: The artifact name.
      uri: the Vertex Batch Prediction resource uri, in a form of
      https://{service-endpoint}/v1/projects/{project}/locations/{location}/batchPredictionJobs/{batchPredictionJob},
        where {service-endpoint} is one of the supported service endpoints at
      https://cloud.google.com/vertex-ai/docs/reference/rest#rest_endpoints
      job_resource_name: The name of the batch prediction job resource, in a
        form of
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
    return cls(
        name=name,
        uri=uri,
        metadata={
            _RESOURCE_NAME_KEY: job_resource_name,
            'bigqueryOutputTable': bigquery_output_table,
            'bigqueryOutputDataset': bigquery_output_dataset,
            'gcsOutputDirectory': gcs_output_directory,
        },
    )


class VertexDataset(dsl.Artifact):
  """An artifact representing a Vertex AI `Dataset resource <https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.datasets>`_."""
  schema_title = 'google.VertexDataset'
  schema_version = '0.0.1'
  schema = textwrap.dedent("""\
title: google.VertexDataset
type: object
properties:
  resourceName:
    type: string""")

  @classmethod
  def create(
      cls,
      name: str,
      uri: str,
      dataset_resource_name: str,
  ) -> 'VertexDataset':
    """Create a VertexDataset artifact instance.

    Args:
      name: The artifact name.
      uri: the Vertex Dataset resource uri, in a form of
      https://{service-endpoint}/v1/projects/{project}/locations/{location}/datasets/{datasets_name},
        where {service-endpoint} is one of the supported service endpoints at
      https://cloud.google.com/vertex-ai/docs/reference/rest#rest_endpoints
      dataset_resource_name: The name of the Dataset resource, in a form of
        projects/{project}/locations/{location}/datasets/{datasets_name}. For
        more details, see
      https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.datasets/get
    """
    return cls(
        uri=uri,
        name=name,
        metadata={_RESOURCE_NAME_KEY: dataset_resource_name},
    )


class BQMLModel(dsl.Artifact):
  """An artifact representing a Google Cloud `BQML Model resource <https://cloud.google.com/bigquery/docs/reference/rest/v2/models>`_."""
  schema_title = 'google.BQMLModel'
  schema_version = '0.0.1'
  schema = textwrap.dedent("""\
title: google.BQMLModel
type: object
properties:
  projectId:
    type: string
  datasetId:
    type: string
  modelId:
    type: string""")

  @classmethod
  def create(
      cls,
      name: str,
      project_id: str,
      dataset_id: str,
      model_id: str,
  ) -> 'BQMLModel':
    """Create a BQMLModel artifact instance.

    Args:
      name: The artifact name.
      project_id: The ID of the project containing this model.
      dataset_id: The ID of the dataset containing this model.
      model_id: The ID of the model.  For more details, see
      https://cloud.google.com/bigquery/docs/reference/rest/v2/models#ModelReference
    """
    return cls(
        name=name,
        uri=f'https://www.googleapis.com/bigquery/v2/projects/{project_id}/datasets/{dataset_id}/models/{model_id}',
        metadata={
            'projectId': project_id,
            'datasetId': dataset_id,
            'modelId': model_id,
        },
    )


class BQTable(dsl.Artifact):
  """An artifact representing a Google Cloud `BQ Table resource <https://cloud.google.com/bigquery/docs/reference/rest/v2/tables>`_."""
  schema_title = 'google.BQTable'
  schema_version = '0.0.1'
  schema = textwrap.dedent("""\
title: google.BQTable
type: object
properties:
  projectId:
    type: string
  datasetId:
    type: string
  tableId:
    type: string
  expirationTime:
    type: string""")

  @classmethod
  def create(
      cls,
      name: str,
      project_id: str,
      dataset_id: str,
      table_id: str,
  ) -> 'BQTable':
    """Create a BQTable artifact instance.

    Args:
      name: The artifact name.
      project_id: The ID of the project containing this table.
      dataset_id: The ID of the dataset containing this table.
      table_id: The ID of the table.  For more details, see
      https://cloud.google.com/bigquery/docs/reference/rest/v2/TableReference
    """
    return cls(
        name=name,
        uri=f'https://www.googleapis.com/bigquery/v2/projects/{project_id}/datasets/{dataset_id}/tables/{table_id}',
        metadata={
            'projectId': project_id,
            'datasetId': dataset_id,
            'tableId': table_id,
        },
    )


class UnmanagedContainerModel(dsl.Artifact):
  """An artifact representing a Vertex AI `unmanaged container model <https://cloud.google.com/vertex-ai/docs/reference/rest/v1/ModelContainerSpec>`_."""
  schema_title = 'google.UnmanagedContainerModel'
  schema_version = '0.0.1'
  schema = textwrap.dedent("""\
title: google.UnmanagedContainerModel
type: object
properties:
  predictSchemata:
    type: object
    properties:
      instanceSchemaUri:
        type: string
      parametersSchemaUri:
        type: string
      predictionSchemaUri:
        type: string
  containerSpec:
    type: object
    properties:
      imageUri:
        type: string
      command:
        type: array
        items:
          type: string
      args:
        type: array
        items:
          type: string
      env:
        type: array
        items:
          type: object
          properties:
            name:
              type: string
            value:
              type: string
      ports:
        type: array
        items:
          type: object
          properties:
            containerPort:
              type: integer
      predictRoute:
        type: string
      healthRoute:
        type: string""")

  @classmethod
  def create(
      cls,
      predict_schemata: Dict,
      container_spec: Dict,
  ) -> 'UnmanagedContainerModel':
    """Create a UnmanagedContainerModel artifact instance.

    Args:
      predict_schemata: Contains the schemata used in Model's predictions and
        explanations via PredictionService.Predict, PredictionService.Explain
        and BatchPredictionJob. For more details, see
      https://cloud.google.com/vertex-ai/docs/reference/rest/v1/PredictSchemata
      container_spec: Specification of a container for serving predictions. Some
        fields in this message correspond to fields in the Kubernetes Container
        v1 core specification. For more details, see
      https://cloud.google.com/vertex-ai/docs/reference/rest/v1/ModelContainerSpec
    """
    return cls(
        metadata={
            'predictSchemata': predict_schemata,
            'containerSpec': container_spec,
        }
    )


class ClassificationMetrics(dsl.Artifact):
  """An artifact representing evaluation `classification metrics <https://cloud.google.com/vertex-ai/docs/tabular-data/classification-regression/evaluate-model#classification_1>`_."""

  schema_title = 'google.ClassificationMetrics'
  schema_version = '0.0.1'
  schema = textwrap.dedent("""\
title: google.ClassificationMetrics
type: object
properties:
  aggregationType:
    type: string
    enum:
      - AGGREGATION_TYPE_UNSPECIFIED
      - MACRO_AVERAGE
      - MICRO_AVERAGE
  aggregationThreshold:
    type: number
    format: float
  recall:
    type: number
    format: float
  precision:
    type: number
    format: float
  f1_score:
    type: number
    format: float
  accuracy:
    type: number
    format: float
  auPrc:
    type: number
    format: float
  auRoc:
    type: number
    format: float
  logLoss:
    type: number
    format: float
  confusionMatrix:
    type: object
    properties:
      rows:
        type: array
        items:
          type: array
          items:
            type: integer
            format: int64
      annotationSpecs:
        type: array
        items:
          type: object
          properties:
            id:
              type: string
            displayName:
              type: string
  confidenceMetrics:
    type: array
    items:
      type: object
      properties:
        confidenceThreshold:
          type: number
          format: float
        recall:
          type: number
          format: float
        precision:
          type: number
          format: float
        f1Score:
          type: number
          format: float
        maxPredictions:
          type: integer
          format: int32
        falsePositiveRate:
          type: number
          format: float
        accuracy:
          type: number
          format: float
        truePositiveCount:
          type: integer
          format: int64
        falsePositiveCount:
          type: integer
          format: int64
        falseNegativeCount:
          type: integer
          format: int64
        trueNegativeCount:
          type: integer
          format: int64
        recallAt1:
          type: number
          format: float
        precisionAt1:
          type: number
          format: float
        falsePositiveRateAt1:
          type: number
          format: float
        f1ScoreAt1:
          type: number
          format: float
        confusionMatrix:
          type: object
          properties:
            rows:
              type: array
              items:
                type: array
                items:
                  type: integer
                  format: int64
            annotationSpecs:
              type: array
              items:
                type: object
                properties:
                  id:
                    type: string
                  displayName:
                    type: string""")

  @classmethod
  def create(
      cls,
      name: str = 'evaluation_metrics',
      recall: Optional[float] = None,
      precision: Optional[float] = None,
      f1_score: Optional[float] = None,
      accuracy: Optional[float] = None,
      au_prc: Optional[float] = None,
      au_roc: Optional[float] = None,
      log_loss: Optional[float] = None,
  ) -> 'ClassificationMetrics':
    """Create a ClassificationMetrics artifact instance.

    Args:
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
    return cls(
        name=name,
        metadata=metadata,
    )


class RegressionMetrics(dsl.Artifact):
  """An artifact representing evaluation `regression metrics <https://cloud.google.com/vertex-ai/docs/tabular-data/classification-regression/evaluate-model#regression_1>`_."""

  schema_title = 'google.RegressionMetrics'
  schema_version = '0.0.1'
  schema = textwrap.dedent("""\
title: google.RegressionMetrics
type: object
properties:
  rootMeanSquaredError:
    type: number
    format: float
  meanAbsoluteError:
    type: number
    format: float
  meanAbsolutePercentageError:
    type: number
    format: float
  rSquared:
    type: number
    format: float
  rootMeanSquaredLogError:
    type: number
    format: float""")

  @classmethod
  def create(
      cls,
      name: str = 'evaluation_metrics',
      root_mean_squared_error: Optional[float] = None,
      mean_absolute_error: Optional[float] = None,
      mean_absolute_percentage_error: Optional[float] = None,
      r_squared: Optional[float] = None,
      root_mean_squared_log_error: Optional[float] = None,
  ) -> 'RegressionMetrics':
    """Create a RegressionMetrics artifact instance.

    Args:
      root_mean_squared_error: Root Mean Squared Error (RMSE).
      mean_absolute_error: Mean Absolute Error (MAE).
      mean_absolute_percentage_error: Mean absolute percentage error.
      r_squared: Coefficient of determination as Pearson correlation
        coefficient.
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
    return cls(
        name=name,
        metadata=metadata,
    )


class ForecastingMetrics(dsl.Artifact):
  """An artifact representing evaluation `forecasting metrics <https://cloud.google.com/vertex-ai/docs/tabular-data/forecasting/evaluate-model#metrics>`_."""

  schema_title = 'google.ForecastingMetrics'
  schema_version = '0.0.1'
  schema = textwrap.dedent("""\
title: google.ForecastingMetrics
type: object
properties:
  rootMeanSquaredError:
    type: number
    format: float
  meanAbsoluteError:
    type: number
    format: float
  meanAbsolutePercentageError:
    type: number
    format: float
  rSquared:
    type: number
    format: float
  rootMeanSquaredLogError:
    type: number
    format: float
  weightedAbsolutePercentageError:
    type: number
    format: float
  rootMeanSquaredPercentageError:
    type: number
    format: float
  symmetricMeanAbsolutePercentageError:
    type: number
    format: float
  quantileMetrics:
    type: array
    items:
      type: object
      properties:
        quantile:
          type: number
          format: double
        scaledPinballLoss:
          type: number
          format: float
        observedQuantile:
          type: number
          format: double""")

  @classmethod
  def create(
      cls,
      name: str = 'evaluation_metrics',
      root_mean_squared_error: Optional[float] = None,
      mean_absolute_error: Optional[float] = None,
      mean_absolute_percentage_error: Optional[float] = None,
      r_squared: Optional[float] = None,
      root_mean_squared_log_error: Optional[float] = None,
      weighted_absolute_percentage_error: Optional[float] = None,
      root_mean_squared_percentage_error: Optional[float] = None,
      symmetric_mean_absolute_percentage_error: Optional[float] = None,
  ) -> 'ForecastingMetrics':
    """Create a ForecastingMetrics artifact instance.

    Args:
      root_mean_squared_error: Root Mean Squared Error (RMSE).
      mean_absolute_error: Mean Absolute Error (MAE).
      mean_absolute_percentage_error: Mean absolute percentage error.
      r_squared: Coefficient of determination as Pearson correlation
        coefficient.
      root_mean_squared_log_error: Root mean squared log error.
      weighted_absolute_percentage_error: Weighted Absolute Percentage Error.
        Does not use weights, this is just what the metric is called. Undefined
        if actual values sum to zero. Will be very large if actual values sum to
        a very small number.
      root_mean_squared_percentage_error: Root Mean Square Percentage Error.
        Square root of MSPE. Undefined/imaginary when MSPE is negative.
      symmetric_mean_absolute_percentage_error: Symmetric Mean Absolute
        Percentage Error.
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
    return cls(
        name=name,
        metadata=metadata,
    )
