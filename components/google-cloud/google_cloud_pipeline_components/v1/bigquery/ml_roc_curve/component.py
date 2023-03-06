# Copyright 2022 The Kubeflow Authors. All Rights Reserved.
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

from typing import Dict, List

from google_cloud_pipeline_components.types.artifact_types import BQMLModel
from google_cloud_pipeline_components.types.artifact_types import BQTable
from kfp.dsl import ConcatPlaceholder
from kfp.dsl import container_component
from kfp.dsl import ContainerSpec
from kfp.dsl import Input
from kfp.dsl import Output
from kfp.dsl import OutputPath


@container_component
def bigquery_ml_roc_curve_job(
    project: str,
    model: Input[BQMLModel],
    # TODO(b/243411151): misalignment of arguments in documentation vs function
    # signature.
    roc_curve: Output[BQTable],
    gcp_resources: OutputPath(str),
    location: str = 'us-central1',
    table_name: str = '',
    query_statement: str = '',
    thresholds: str = '',
    query_parameters: List[str] = [],
    job_configuration_query: Dict[str, str] = {},
    labels: Dict[str, str] = {},
):
  """Launch a BigQuery roc curve job and waits for it to finish.

    Args:
        project (str): Required. Project to run BigQuery roc curve job.
        location (Optional[str]): Location of the job to run BigQuery roc curve
          job. If not set, default to `US` multi-region.  For more details, see
          https://cloud.google.com/bigquery/docs/locations#specifying_your_location
        model (google.BQMLModel): Required. BigQuery ML model for BigQuery roc
          curv job. For more details, see
          https://cloud.google.com/bigquery-ml/docs/reference/standard-sql/bigqueryml-syntax-roc#roc_model_name
        table_name (Optional[str]): BigQuery table id of the input table that
          contains the evaluation data. For more details, see
          https://cloud.google.com/bigquery-ml/docs/reference/standard-sql/bigqueryml-syntax-roc#roc_table_name
        query_statement (Optional[str]): Query statement string used to generate
          the evaluation data. For more details, see
          https://cloud.google.com/bigquery-ml/docs/reference/standard-sql/bigqueryml-syntax-roc#roc_query_statement
        thresholds (Optional[str]): Percentile values of the prediction output.
          For more details, see
          https://cloud.google.com/bigquery-ml/docs/reference/standard-sql/bigqueryml-syntax-roc#roc_thresholds
        query_parameters (Optional[Sequence]): jobs.query parameters for
          standard SQL queries. If query_parameters are both specified in here
          and in job_configuration_query, the value in here will override the
          other one.
        job_configuration_query (Optional[dict]): A json formatted string
          describing the rest of the job configuration. For more details, see
          https://cloud.google.com/bigquery/docs/reference/rest/v2/Job#JobConfigurationQuery
        labels (Optional[dict]): The labels associated with this job. You can
          use these to organize and group your jobs. Label keys and values can
          be no longer than 63 characters, can only containlowercase letters,
          numeric characters, underscores and dashes. International characters
          are allowed. Label values are optional. Label keys must start with a
          letter and each label in the list must have a different key.
            Example: { "name": "wrench", "mass": "1.3kg", "count": "3" }.

    Returns:
        roc_curve (system.Artifact):
          Describes common metrics applicable to the type of model supplied.
          For more details, see
          https://cloud.google.com/bigquery-ml/docs/reference/standard-sql/bigqueryml-syntax-roc#mlroc_curve_output
        gcp_resources (str):
            Serialized gcp_resources proto tracking the BigQuery job.
            For more details, see
            https://github.com/kubeflow/pipelines/blob/master/components/google-cloud/google_cloud_pipeline_components/proto/README.md.
  """
  return ContainerSpec(
      image='gcr.io/ml-pipeline/google-cloud-pipeline-components:2.0.0b1',
      command=[
          'python3', '-u', '-m',
          'google_cloud_pipeline_components.container.v1.bigquery.ml_roc_curve.launcher'
      ],
      args=[
          '--type',
          'BigqueryMLRocCurveJob',
          '--project',
          project,
          '--location',
          location,
          '--model_name',
          ConcatPlaceholder([
              "{{$.inputs.artifacts['model'].metadata['projectId']}}", '.',
              "{{$.inputs.artifacts['model'].metadata['datasetId']}}", '.',
              "{{$.inputs.artifacts['model'].metadata['modelId']}}"
          ]),
          '--table_name',
          table_name,
          '--query_statement',
          query_statement,
          '--thresholds',
          thresholds,
          '--payload',
          ConcatPlaceholder([
              '{', '"configuration": {', '"query": ', job_configuration_query,
              ', "labels": ', labels, '}', '}'
          ]),
          '--job_configuration_query_override',
          ConcatPlaceholder(
              ['{', '"query_parameters": ', query_parameters, '}']),
          '--gcp_resources',
          gcp_resources,
          '--executor_input',
          '{{$}}',
      ])
