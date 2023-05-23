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

from typing import Dict, List

from google_cloud_pipeline_components import _image
from google_cloud_pipeline_components.types.artifact_types import BQMLModel
from google_cloud_pipeline_components.types.artifact_types import BQTable
from kfp.dsl import ConcatPlaceholder
from kfp.dsl import container_component
from kfp.dsl import ContainerSpec
from kfp.dsl import Input
from kfp.dsl import Output
from kfp.dsl import OutputPath


@container_component
def bigquery_ml_global_explain_job(
    project: str,
    model: Input[BQMLModel],
    destination_table: Output[BQTable],
    gcp_resources: OutputPath(str),
    location: str = 'us-central1',
    class_level_explain: bool = False,
    query_parameters: List[str] = [],
    job_configuration_query: Dict[str, str] = {},
    labels: Dict[str, str] = {},
    encryption_spec_key_name: str = '',
):
  # fmt: off
  """Launch a BigQuery global explain fetching job and waits for it to finish.

  Args:
      project: Project to run BigQuery model creation job.
      location: Location of the job to create the BigQuery
        model. If not set, default to `US` multi-region.  For more details,
        see
        https://cloud.google.com/bigquery/docs/locations#specifying_your_location
      model: BigQuery ML model for global
        explain. For more details, see
        https://cloud.google.com/bigquery-ml/docs/reference/standard-sql/bigqueryml-syntax-predict#predict_model_name
      class_level_explain: For classification
          models, if class_level_explain is set to TRUE then global feature
          importances are returned for each class. Otherwise, the global
          feature importance of the entire model is returned rather than that
          of each class. By default, class_level_explain is set to FALSE. This
          option only applies to classification models. Regression models only
          have model-level global feature importance.

  Returns:
      destination_table: Describes the table where the global explain results should be stored.
      gcp_resources: Serialized gcp_resources proto tracking the BigQuery job. For more details, see
        https://github.com/kubeflow/pipelines/blob/master/components/google-cloud/google_cloud_pipeline_components/proto/README.md.
  """
  # fmt: on
  return ContainerSpec(
      image=_image.GCPC_IMAGE_TAG,
      command=[
          'python3',
          '-u',
          '-m',
          'google_cloud_pipeline_components.container.v1.bigquery.global_explain.launcher',
      ],
      args=[
          '--type',
          'BigqueryMLGlobalExplainJob',
          '--project',
          project,
          '--location',
          location,
          '--model_name',
          ConcatPlaceholder([
              model.metadata['projectId'],
              '.',
              model.metadata['datasetId'],
              '.',
              model.metadata['modelId'],
          ]),
          '--class_level_explain',
          class_level_explain,
          '--payload',
          ConcatPlaceholder([
              '{',
              '"configuration": {',
              '"query": ',
              job_configuration_query,
              ', "labels": ',
              labels,
              '}',
              '}',
          ]),
          '--job_configuration_query_override',
          ConcatPlaceholder([
              '{',
              '"query_parameters": ',
              query_parameters,
              ', "destination_encryption_configuration": {',
              '"kmsKeyName": "',
              encryption_spec_key_name,
              '"}',
              '}',
          ]),
          '--gcp_resources',
          gcp_resources,
          '--executor_input',
          '{{$}}',
      ],
  )
