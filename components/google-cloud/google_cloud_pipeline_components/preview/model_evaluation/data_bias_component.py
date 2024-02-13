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

from typing import Any, List

from google_cloud_pipeline_components import _placeholders
from google_cloud_pipeline_components._implementation.model_evaluation import version
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
    target_field_name: str,
    bias_configs: List[Any],
    location: str = 'us-central1',
    dataset_format: str = 'jsonl',
    dataset_storage_source_uris: List[str] = [],
    dataset: Input[VertexDataset] = None,
    columns: List[str] = [],
    encryption_spec_key_name: str = '',
    project: str = _placeholders.PROJECT_ID_PLACEHOLDER,
):
  # fmt: off
  """Detects data bias metrics in a dataset.

  Creates a Dataflow job with Apache Beam to category each data point in the
  dataset to the corresponding bucket based on bias configs, then compute data
  bias metrics for the dataset.

  Args:
      location: Location for running data bias detection.
      target_field_name: The full name path of the features target field in the predictions file. Formatted to be able to find nested columns, delimited by `.`. Alternatively referred to as the ground truth (or ground_truth_column) field.
      bias_configs: A list of `google.cloud.aiplatform_v1beta1.types.ModelEvaluation.BiasConfig`. When provided, compute data bias metrics for each defined slice. Below is an example of how to format this input.

        1: First, create a BiasConfig. `from google.cloud.aiplatform_v1beta1.types.ModelEvaluation import BiasConfig` `from google.cloud.aiplatform_v1.types.ModelEvaluationSlice.Slice import SliceSpec` `from google.cloud.aiplatform_v1.types.ModelEvaluationSlice.Slice.SliceSpec import SliceConfig` `bias_config = BiasConfig(bias_slices=SliceSpec(configs={ 'feature_a': SliceConfig(SliceSpec.Value(string_value= 'label_a') ) }))`
        2: Create a list to store the bias configs into. `bias_configs = []`
        3: Format each BiasConfig into a JSON or Dict. `bias_config_json = json_format.MessageToJson(bias_config` or `bias_config_dict = json_format.MessageToDict(bias_config).`
        4: Combine each bias_config JSON into a list. `bias_configs.append(bias_config_json)`
        5: Finally, pass bias_configs as an parameter for this component. `DetectDataBiasOp(bias_configs=bias_configs)`

      dataset_format: The file format for the dataset. `jsonl` and `csv` are the currently allowed formats.
      dataset_storage_source_uris: Google Cloud Storage URI(-s) to unmanaged test datasets.`jsonl` and `csv` is currently allowed format. If `dataset` is also provided, this field will be overriden by the provided Vertex Dataset.
      dataset: A `google.VertexDataset` artifact of the dataset. If `dataset_gcs_source` is also provided, this Vertex Dataset argument will override the GCS source.
      encryption_spec_key_name: Customer-managed encryption key options for the Dataflow. If this is set, then all resources created by the Dataflow will be encrypted with the provided encryption key. Has the form: `projects/my-project/locations/my-location/keyRings/my-kr/cryptoKeys/my-key`. The key needs to be in the same region as where the compute resource is created.
      project: Project to run data bias detection. Defaults to the project in which the PipelineJob is run.

  Returns:
      data_bias_metrics: Artifact tracking the data bias detection output.
      gcp_resources: Serialized gcp_resources proto tracking the Dataflow job. For more details, see https://github.com/kubeflow/pipelines/blob/master/components/google-cloud/google_cloud_pipeline_components/proto/README.md.
  """
  # fmt: on
  return ContainerSpec(
      image=version.EVAL_IMAGE_TAG,
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
          dataset.metadata['resourceName'],
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
