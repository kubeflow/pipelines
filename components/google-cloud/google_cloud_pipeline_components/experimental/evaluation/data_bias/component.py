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

from google_cloud_pipeline_components.types.artifact_types import VertexDataset
from kfp.dsl import Artifact
from kfp.dsl import container_component
from kfp.dsl import ContainerSpec
from kfp.dsl import Input
from kfp.dsl import Output
from kfp.dsl import OutputPath
from kfp.dsl import PIPELINE_JOB_ID_PLACEHOLDER
from kfp.dsl import PIPELINE_ROOT_PLACEHOLDER
from kfp.dsl import PIPELINE_TASK_ID_PLACEHOLDER


@container_component
def detect_data_bias(
    gcp_resources: OutputPath(str),
    data_bias_metrics: Output[Artifact],
    project: str,
    target_field_name: str,
    bias_configs: list,
    location: str = 'us-central1',
    dataset_format: str = 'jsonl',
    dataset_storage_source_uris: list = [],
    dataset: Input[VertexDataset] = None,
    columns: list = [],
    encryption_spec_key_name: str = '',
):
  """Detects data bias metrics in a dataset.

  Args:
      project (str): Project to run data bias detection.
      location (Optional[str]): Location for running data bias detection. If not
        set, defaulted to `us-central1`.
      target_field_name (str): The full name path of the features target field
        in the predictions file. Formatted to be able to find nested columns,
        delimited by `.`. Alternatively referred to as the ground truth (or
        ground_truth_column) field.
      bias_configs (Sequence[BiasConfig]): A list of BiasConfig protos, with
        each BiasConfig in the list formatted with json_format.MessageToJson or
        json_format.MessageToDict.
      dataset_format (Optional[str]): The file format for the dataset. `jsonl`,
        `csv`, and `bigquery` are the currently allowed formats. If not set,
        defaulted to `jsonl`.
      dataset_storage_source_uris (Optional[list[str]]): Optional. Google Cloud
        Storage URI(-s) to unmanaged test datasets.`jsonl` and `csv`is currently
        allowed format. If `dataset` is also provided, this field will be
        overriden by the provided Vertex Dataset.
      dataset (Optional[google.VertexDataset]): Optional. A google.VertexDataset
        artifact of the dataset. If `dataset_gcs_source` is also provided, this
        Vertex Dataset argument will override the GCS source.
      encryption_spec_key_name (Optional[str]): Customer-managed encryption key
        options for the CustomJob. If this is set, then all resources created by
        the CustomJob will be encrypted with the provided encryption key.

  Returns:
      data_bias_metrics (system.Artifact):
          Artifact tracking the model bias detection output.
      gcp_resources (str):
          Serialized gcp_resources proto tracking the batch prediction job.

          For more details, see
          https://github.com/kubeflow/pipelines/blob/master/components/google-cloud/google_cloud_pipeline_components/proto/README.md.
  """
  return ContainerSpec(
      image='gcr.io/ml-pipeline/model-evaluation:v0.9.1',
      command=[
          'python3',
          '/main.py',
      ],
      args=[
          '--task',
          'data_bias',
          '--bias_configs',
          bias_configs,
          '--target_field_name',
          target_field_name,
          '--columns',
          columns,
          '--dataset_format',
          dataset_format,
          '--dataset_storage_source_uris',
          dataset_storage_source_uris,
          '--dataset',
          "{{$.inputs.artifacts['dataset'].metadata['resourceName']}}",
          '--project',
          project,
          '--location',
          location,
          '--root_dir',
          f'{PIPELINE_ROOT_PLACEHOLDER}/{PIPELINE_JOB_ID_PLACEHOLDER}-{PIPELINE_TASK_ID_PLACEHOLDER}',
          '--dataflow_job_prefix',
          f'evaluation-detect-data-bias-{PIPELINE_JOB_ID_PLACEHOLDER}-{PIPELINE_TASK_ID_PLACEHOLDER}',
          '--output_metrics_gcs_path',
          data_bias_metrics.path,
          '--gcp_resources',
          gcp_resources,
          '--executor_input',
          '{{$}}',
          '--kms_key_name',
          encryption_spec_key_name,
      ],
  )
