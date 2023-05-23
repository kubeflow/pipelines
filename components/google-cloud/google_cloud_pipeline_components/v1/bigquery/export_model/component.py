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

from typing import Dict

from google_cloud_pipeline_components import _image
from google_cloud_pipeline_components.types.artifact_types import BQMLModel
from kfp.dsl import ConcatPlaceholder
from kfp.dsl import container_component
from kfp.dsl import ContainerSpec
from kfp.dsl import Input
from kfp.dsl import OutputPath


@container_component
def bigquery_export_model_job(
    project: str,
    model: Input[BQMLModel],
    model_destination_path: str,
    exported_model_path: OutputPath(str),
    gcp_resources: OutputPath(str),
    location: str = 'us-central1',
    job_configuration_extract: Dict[str, str] = {},
    labels: Dict[str, str] = {},
):
  # fmt: off
  """Launch a BigQuery export model job and waits for it to finish.

  Args:
      project: Project to run BigQuery model export job.
      location: Location of the job to export the BigQuery
        model. If not set, default to `US` multi-region.  For more details,
        see
        https://cloud.google.com/bigquery/docs/locations#specifying_your_location
      model: BigQuery ML model to export.
        model_destination_path:
        The gcs bucket to export the
        model to.
      job_configuration_extract: A json formatted string
        describing the rest of the job configuration.  For more details, see
        https://cloud.google.com/bigquery/docs/reference/rest/v2/Job#JobConfigurationQuery
      labels: The labels associated with this job. You can
        use these to organize and group your jobs. Label keys and values can
        be no longer than 63 characters, can only containlowercase letters,
        numeric characters, underscores and dashes. International characters
        are allowed. Label values are optional. Label keys must start with a
        letter and each label in the list must have a different key.
          Example: { "name": "wrench", "mass": "1.3kg", "count": "3" }.

  Returns:
      exported_model_path: The gcs bucket path where you export the model to.
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
          'google_cloud_pipeline_components.container.v1.bigquery.export_model.launcher',
      ],
      args=[
          '--type',
          'BigqueryExportModelJob',
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
          '--model_destination_path',
          model_destination_path,
          '--payload',
          ConcatPlaceholder([
              '{',
              '"configuration": {',
              '"query": ',
              job_configuration_extract,
              ', "labels": ',
              labels,
              '}',
              '}',
          ]),
          '--exported_model_path',
          exported_model_path,
          '--gcp_resources',
          gcp_resources,
          '--executor_input',
          '{{$}}',
      ],
  )
