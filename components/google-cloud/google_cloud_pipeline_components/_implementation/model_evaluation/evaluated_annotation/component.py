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


from typing import Optional

from google_cloud_pipeline_components._implementation.model_evaluation import version
from kfp import dsl
from kfp.dsl import Artifact
from kfp.dsl import Input


@dsl.container_component
def evaluated_annotation(
    project: str,
    predictions_storage_source: Input[Artifact],
    evaluated_annotation_output_uri: dsl.OutputPath(str),
    gcp_resources: dsl.OutputPath(str),
    location: Optional[str] = 'us-central1',
    ground_truth_storage_source: Optional[str] = '',
    dataflow_service_account: Optional[str] = '',
    dataflow_disk_size_gb: Optional[int] = 50,
    dataflow_machine_type: Optional[str] = 'n1-standard-4',
    dataflow_workers_num: Optional[int] = 1,
    dataflow_max_workers_num: Optional[int] = 5,
    dataflow_subnetwork: Optional[str] = '',
    dataflow_use_public_ips: Optional[bool] = True,
    encryption_spec_key_name: Optional[str] = '',
):
  # fmt: off
  """Computes the Evaluated Annotations of a dataset for a model's prediction
  results.

  Predicted labels are categorized into TruePositive(TP) or FalsePositive(FP) or
  FalseNegative(FN) based on their ground truth annotations.

  Args:
      project: GCP Project ID.
      location: GCP Region. If not set, defaulted to `us-central1`.
      predictions_storage_source: An artifact with its URI pointing toward a GCS
        directory with prediction files to be used for this evaluation. `jsonl`
        is currently the only allowed format. For prediction results, the files
        should be named "prediction.results-*" or "predictions_*".
        ground_truth_storage_source(str): Required. The GCS URI representing
        where the preprocessed test dataset is located. The dataset must contain
        the ground truth annotations.This field must be the output from
        DatasetPreprocessorErrorAnalysis component. `jsonl` is currently the
        only allowed format.
      dataflow_service_account: Service account to run the dataflow job. If not
        set, dataflow will use the default woker service account. For more
        details, see
        https://cloud.google.com/dataflow/docs/concepts/security-and-permissions#default_worker_service_account
      dataflow_disk_size_gb: The disk size (in GB) of the machine executing the
        evaluation run. If not set, defaulted to `50`.
      dataflow_machine_type: The machine type executing the evaluation run. If
        not set, defaulted to `n1-standard-4`.
      dataflow_workers_num: The number of workers executing the evaluation run.
        If not set, defaulted to `10`.
      dataflow_max_workers_num: The max number of workers executing the
        evaluation run. If not set, defaulted to `25`.
      dataflow_subnetwork: Dataflow's fully qualified subnetwork name, when
        empty the default subnetwork will be used. More details:
          https://cloud.google.com/dataflow/docs/guides/specifying-networks#example_network_and_subnetwork_specifications
      dataflow_use_public_ips: Specifies whether Dataflow workers use public IP
        addresses.
      encryption_spec_key_name: Customer-managed encryption key for the Dataflow
        job. If this is set, then all resources created by the Dataflow job will
        be encrypted with the provided encryption key.

  Returns:
      evaluated_annotation_output_uri:
          String representing the GCS URI of the computed evaluated annotations.
  """
  # fmt: on
  return dsl.ContainerSpec(
      image=version.EVAL_IMAGE_TAG,
      command=['python', '/main.py'],
      args=[
          '--task',
          'evaluated_annotation',
          '--display_name',
          'evaluated-annotation-run',
          '--project_id',
          project,
          '--location',
          location,
          '--root_dir',
          f'{dsl.PIPELINE_ROOT_PLACEHOLDER}/{dsl.PIPELINE_JOB_ID_PLACEHOLDER}-{dsl.PIPELINE_TASK_ID_PLACEHOLDER}',
          '--batch_prediction_format',
          'jsonl',
          dsl.IfPresentPlaceholder(
              input_name='predictions_storage_source',
              then=[
                  '--batch_prediction_gcs_source',
                  predictions_storage_source.uri,
              ],
          ),
          '--ground_truth_format',
          'jsonl',
          '--ground_truth_storage_source',
          ground_truth_storage_source,
          '--dataflow_job_prefix',
          f'evaluated-annotation-{dsl.PIPELINE_JOB_ID_PLACEHOLDER}-{dsl.PIPELINE_TASK_ID_PLACEHOLDER}',
          '--dataflow_service_account',
          dataflow_service_account,
          '--dataflow_disk_size',
          dataflow_disk_size_gb,
          '--dataflow_machine_type',
          dataflow_machine_type,
          '--dataflow_workers_num',
          dataflow_workers_num,
          '--dataflow_max_workers_num',
          dataflow_max_workers_num,
          '--dataflow_subnetwork',
          dataflow_subnetwork,
          '--dataflow_use_public_ips',
          dataflow_use_public_ips,
          '--kms_key_name',
          encryption_spec_key_name,
          '--evaluated_annotation_output_uri',
          evaluated_annotation_output_uri,
          '--gcp_resources',
          gcp_resources,
          '--executor_input',
          '{{$}}',
      ],
  )
