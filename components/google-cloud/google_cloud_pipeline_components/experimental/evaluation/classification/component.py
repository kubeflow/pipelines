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

from google_cloud_pipeline_components.types.artifact_types import BQTable
from google_cloud_pipeline_components.types.artifact_types import ClassificationMetrics
from google_cloud_pipeline_components.types.artifact_types import VertexModel
from kfp.dsl import Artifact
from kfp.dsl import ConcatPlaceholder
from kfp.dsl import container_component
from kfp.dsl import ContainerSpec
from kfp.dsl import IfPresentPlaceholder
from kfp.dsl import Input
from kfp.dsl import Output
from kfp.dsl import OutputPath
from kfp.dsl import PIPELINE_JOB_ID_PLACEHOLDER
from kfp.dsl import PIPELINE_TASK_ID_PLACEHOLDER


@container_component
def model_evaluation_classification(
    gcp_resources: OutputPath(str),
    evaluation_metrics: Output[ClassificationMetrics],
    project: str,
    root_dir: str,
    target_field_name: str,
    model: Input[VertexModel] = None,
    location: str = 'us-central1',
    predictions_format: str = 'jsonl',
    predictions_gcs_source: Input[Artifact] = None,
    predictions_bigquery_source: Input[BQTable] = None,
    ground_truth_format: str = 'jsonl',
    ground_truth_gcs_source: list = [],
    ground_truth_bigquery_source: str = '',
    classification_type: str = 'multiclass',
    class_labels: list = [],
    prediction_score_column: str = '',
    prediction_label_column: str = '',
    slicing_specs: list = [],
    positive_classes: list = [],
    dataflow_service_account: str = '',
    dataflow_disk_size: int = 50,
    dataflow_machine_type: str = 'n1-standard-4',
    dataflow_workers_num: int = 1,
    dataflow_max_workers_num: int = 5,
    dataflow_subnetwork: str = '',
    dataflow_use_public_ips: bool = True,
    encryption_spec_key_name: str = '',
):
  """Computes a google.ClassificationMetrics Artifact, containing evaluation metrics given a model's prediction results.

    Creates a dataflow job with Apache Beam and TFMA to compute evaluation
    metrics.
    Supports mutliclass classification evaluation for tabular, image, video, and
    text data.

  Args:
    project (str): Project to run evaluation container.
    location (Optional[str]): Location for running the evaluation. If not set,
      defaulted to `us-central1`.
    root_dir (str): The GCS directory for keeping staging files. A random
      subdirectory will be created under the directory to keep job info for
      resuming the job in case of failure.
    predictions_format (Optional[str]): The file format for the batch prediction
      results. `jsonl` is currently the only allowed format. If not set,
      defaulted to `jsonl`.
    predictions_gcs_source (Optional[system.Artifact]): An artifact with its URI
      pointing toward a GCS directory with prediction or explanation files to be
      used for this evaluation. For prediction results, the files should be
      named "prediction.results-*". For explanation results, the files should be
      named "explanation.results-*".
    predictions_bigquery_source (Optional[google.BQTable]): BigQuery table with
      prediction or explanation data to be used for this evaluation. For
      prediction results, the table column should be named "predicted_*".
      ground_truth_format(Optional[str]): Required for custom tabular and non
      tabular data. The file format for the ground truth files. `jsonl` is
      currently the only allowed format. If not set, defaulted to `jsonl`.
      ground_truth_gcs_source(Optional[Sequence[str]]): Required for custom
      tabular and non tabular data. The GCS uris representing where the ground
      truth is located. Used to provide ground truth for each prediction
      instance when they are not part of the batch prediction jobs prediction
      instance. ground_truth_bigquery_source(Optional[str]): Required for custom
      tabular. The BigQuery table uri representing where the ground truth is
      located. Used to provide ground truth for each prediction instance when
      they are not part of the batch prediction jobs prediction instance.
    classification_type (Optional[str]): The type of classification problem,
      either `multiclass` or `multilabel`. If not set, defaulted to
      `multiclass`.
    class_labels (Optional[Sequence[str]]): The list of class names for the
      target_field_name, in the same order they appear in the batch predictions
      jobs predictions output file. For instance, if the values of
      target_field_name could be either `1` or `0`, and the predictions output
      contains ["1", "0"] for the prediction_label_column, then the class_labels
      input will be ["1", "0"]. If not set, defaulted to the classes found in
      the prediction_label_column in the batch prediction jobs predictions file.
    target_field_name (str): The full name path of the features target field in
      the predictions file. Formatted to be able to find nested columns,
      delimited by `.`. Alternatively referred to as the ground truth (or
      ground_truth_column) field.
    model (Optional[google.VertexModel]): The Model used for predictions job.
      Must share the same ancestor Location.
    prediction_score_column (Optional[str]): Optional. The column name of the
      field containing batch prediction scores. Formatted to be able to find
      nested columns, delimited by `.`. If not set, defaulted to
      `prediction.scores` for classification.
    prediction_label_column (Optional[str]): Optional. The column name of the
      field containing classes the model is scoring. Formatted to be able to
      find nested columns, delimited by `.`. If not set, defaulted to
      `prediction.classes` for classification.
    prediction_id_column (Optional[str]): Optional. The column name of the field
      containing ids for classes the model is scoring. Formatted to be able to
      find nested columns, delimited by `.`.
    slicing_specs (Optional[Sequence[SlicingSpec]]): Optional. List of
      google.cloud.aiplatform_v1.types.ModelEvaluationSlice.SlicingSpec. When
      provided, compute metrics for each defined slice. Below is an example of
      how to format this input.
      1: First, create a SlicingSpec. ```from
        google.cloud.aiplatform_v1.types.ModelEvaluationSlice.Slice import
        SliceSpec from
        google.cloud.aiplatform_v1.types.ModelEvaluationSlice.Slice.SliceSpec
        import SliceConfig  slicing_spec = SliceSpec(configs={ 'feature_a':
        SliceConfig(SliceSpec.Value(string_value='label_a') ) })```
      2: Create a list to store the slicing specs into. `slicing_specs = []`.
      3: Format each SlicingSpec into a JSON or Dict. `slicing_spec_json =
        json_format.MessageToJson(slicing_spec)` or `slicing_spec_dict =
        json_format.MessageToDict(slicing_spec).
      4: Combine each slicing_spec JSON into a list.
        `slicing_specs.append(slicing_spec_json)`.
      5: Finally, pass slicing_specs as an parameter for this component.
        `ModelEvaluationClassificationOp(slicing_specs=slicing_specs)` For more
        details on configuring slices, see
      https://cloud.google.com/python/docs/reference/aiplatform/latest/google.cloud.aiplatform_v1.types.ModelEvaluationSlice
    positive_classes (Optional[Sequence[str]]): Optional. The list of class
      names to create binary classification metrics based on one-vs-rest for
      each value of positive_classes provided.
    dataflow_service_account (Optional[str]): Optional. Service account to run
      the dataflow job. If not set, dataflow will use the default woker service
      account.  For more details, see
      https://cloud.google.com/dataflow/docs/concepts/security-and-permissions#default_worker_service_account
    dataflow_disk_size (Optional[int]): Optional. The disk size (in GB) of the
      machine executing the evaluation run. If not set, defaulted to `50`.
    dataflow_machine_type (Optional[str]): Optional. The machine type executing
      the evaluation run. If not set, defaulted to `n1-standard-4`.
    dataflow_workers_num (Optional[int]): Optional. The number of workers
      executing the evaluation run. If not set, defaulted to `10`.
    dataflow_max_workers_num (Optional[int]): Optional. The max number of
      workers executing the evaluation run. If not set, defaulted to `25`.
    dataflow_subnetwork (Optional[str]): Dataflow's fully qualified subnetwork
      name, when empty the default subnetwork will be used. More
      details:
        https://cloud.google.com/dataflow/docs/guides/specifying-networks#example_network_and_subnetwork_specifications
    dataflow_use_public_ips (Optional[bool]): Specifies whether Dataflow workers
      use public IP addresses.
    encryption_spec_key_name (Optional[str]): Customer-managed encryption key.

  Returns:
    evaluation_metrics (google.ClassificationMetrics): Artifact
      google.ClassificationMetrics representing the classification
      evaluation metrics in GCS.
    gcp_resources (str): Serialized gcp_resources proto tracking the dataflow
      job.

      For more details, see
      https://github.com/kubeflow/pipelines/blob/master/components/google-cloud/google_cloud_pipeline_components/proto/README.md.
  """
  return ContainerSpec(
      image='gcr.io/ml-pipeline/model-evaluation:v0.9',
      command=[
          'python3',
          '/main.py',
      ],
      args=[
          '--setup_file',
          '/setup.py',
          '--json_mode',
          'true',
          '--project_id',
          project,
          '--location',
          location,
          '--problem_type',
          'classification',
          '--target_field_name',
          ConcatPlaceholder(['instance.', target_field_name]),
          '--batch_prediction_format',
          predictions_format,
          IfPresentPlaceholder(
              input_name='predictions_gcs_source',
              then=[
                  '--batch_prediction_gcs_source',
                  "{{$.inputs.artifacts['predictions_gcs_source'].uri}}",
              ],
          ),
          IfPresentPlaceholder(
              input_name='predictions_bigquery_source',
              then=[
                  '--batch_prediction_bigquery_source',
                  ConcatPlaceholder([
                      'bq://',
                      "{{$.inputs.artifacts['predictions_bigquery_source'].metadata['projectId']}}",
                      '.',
                      "{{$.inputs.artifacts['predictions_bigquery_source'].metadata['datasetId']}}",
                      '.',
                      "{{$.inputs.artifacts['predictions_bigquery_source'].metadata['tableId']}}",
                  ]),
              ],
          ),
          IfPresentPlaceholder(
              input_name='model',
              then=[
                  '--model_name',
                  "{{$.inputs.artifacts['model'].metadata['resourceName']}}",
              ],
          ),
          '--ground_truth_format',
          ground_truth_format,
          '--ground_truth_gcs_source',
          ground_truth_gcs_source,
          '--ground_truth_bigquery_source',
          ground_truth_bigquery_source,
          '--root_dir',
          f'{root_dir}/{PIPELINE_JOB_ID_PLACEHOLDER}-{PIPELINE_TASK_ID_PLACEHOLDER}',
          '--classification_type',
          classification_type,
          '--class_labels',
          class_labels,
          '--prediction_score_column',
          prediction_score_column,
          '--prediction_label_column',
          prediction_label_column,
          IfPresentPlaceholder(
              input_name='slicing_specs',
              then=[
                  '--slicing_specs',
                  slicing_specs,
              ],
          ),
          '--positive_classes',
          positive_classes,
          '--dataflow_job_prefix',
          f'evaluation-{PIPELINE_JOB_ID_PLACEHOLDER}-{PIPELINE_TASK_ID_PLACEHOLDER}',
          '--dataflow_service_account',
          dataflow_service_account,
          '--dataflow_disk_size',
          dataflow_disk_size,
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
          '--output_metrics_gcs_path',
          evaluation_metrics.path,
          '--gcp_resources',
          gcp_resources,
          '--executor_input',
          '{{$}}',
      ],
  )
