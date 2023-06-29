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

from google_cloud_pipeline_components._implementation.model_evaluation import version
from google_cloud_pipeline_components.types.artifact_types import VertexDataset
from kfp.dsl import container_component
from kfp.dsl import ContainerSpec
from kfp.dsl import IfPresentPlaceholder
from kfp.dsl import Input
from kfp.dsl import OutputPath
from kfp.dsl import PIPELINE_JOB_ID_PLACEHOLDER
from kfp.dsl import PIPELINE_ROOT_PLACEHOLDER
from kfp.dsl import PIPELINE_TASK_ID_PLACEHOLDER


@container_component
def dataset_preprocessor_error_analysis(
    gcp_resources: OutputPath(str),
    batch_prediction_storage_source: OutputPath(list),
    model_evaluation_storage_source: OutputPath(list),
    test_data_items_storage_source: OutputPath(str),
    training_data_items_storage_source: OutputPath(str),
    project: str,
    location: str = 'us-central1',
    test_dataset: Input[VertexDataset] = None,
    test_dataset_annotation_set_name: str = '',
    training_dataset: Input[VertexDataset] = None,
    training_dataset_annotation_set_name: str = '',
    test_dataset_storage_source_uris: list = [],
    training_dataset_storage_source_uris: list = [],
):
  # fmt: off
  """Preprocesses datasets for Vision Error Analysis pipelines.

  Args:
      project: GCP Project ID.
      location: GCP Region. If not set, defaulted to
        `us-central1`.
      test_dataset: A
        google.VertexDataset artifact of the test dataset. If
        `test_dataset_storage_source_uris` is also provided, this Vertex Dataset
        argument will override the GCS source.
      test_dataset_annotation_set_name: A string of
        the annotation_set resource name containing the ground truth of the test
        datset used for evaluation.
      training_dataset: A
        google.VertexDataset artifact of the training dataset. If
        `training_dataset_storage_source_uris` is also provided, this Vertex
        Dataset argument will override the GCS source.
      training_dataset_annotation_set_name: A string
        of the annotation_set resource name containing the ground truth of the
        test datset used for evaluation.
      test_dataset_storage_source_uris: Google
        Cloud Storage URI(-s) to unmanaged test datasets.`jsonl` is currently
        the only allowed format. If `test_dataset` is also provided, this field
        will be overriden by the provided Vertex Dataset.
      training_dataset_storage_source_uris:
        Google Cloud Storage URI(-s) to unmanaged test datasets.`jsonl` is
        currently the only allowed format. If `training_dataset` is also
        provided, this field will be overriden by the provided Vertex Dataset.

  Returns:
      batch_prediction_storage_source: Google Cloud Storage URI to preprocessed dataset instances for Vertex
        Batch Prediction
        component. Target field name is removed.
        For example, this is an input instance:
          {"content":"gs://flowers/roses/001.jpg","mimeType":"image/jpeg"}
      model_evaluation_storage_source: Google Cloud Storage URI to preprocessed dataset instances for Vertex
        Model Evaluation
        component. Target field name is defaulted to "ground_truth".
        For example, this is an input instance:
          {"content":"gs://flowers/roses/001.jpg","mimeType":"image/jpeg",
          "ground_truth": "roses"}
      test_data_items_storage_source: Google Cloud Storage URI to preprocessed test dataset instances for
        Evaluated
        Annotation or Vision Error Analysis pipelines. It is the input source
        for Evaluated
        Annotation component and Feature Extraction component.
        For example, this is an input instance:
          {"dataItemID":"projects/*/locations/*/datasets/*/dataItems/000",
          "imageGcsUri":"gs://flowers/roses/001.jpg",
          "mimeType":  "image/jpeg",
          "data_split":"test",
          "annotation_set_name":"333",
          "ground_truth_annotations":["roses"],
          "annotation_spec_ids": ["123"],
          "annotation_resource_names":
            ["projects/*/locations/*/datasets/*/dataItems/000/annotations/111"]}
      training_data_items_storage_source: Google Cloud Storage URI to preprocessed training dataset instances for
        Vision Error
        Analysis pipelines. It is the input source for Feature Extraction
        component. It uses the
        same output format as test_data_items_storage_source.
      gcp_resources: Serialized gcp_resources proto tracking the custom job. For more details, see
        https://github.com/kubeflow/pipelines/blob/master/components/google-cloud/google_cloud_pipeline_components/proto/README.md.
  """
  # fmt: on
  return ContainerSpec(
      image=version.EVAL_IMAGE_TAG,
      command=['python3', '/main.py'],
      args=[
          '--task',
          'dataset_preprocessor',
          '--project_id',
          project,
          '--location',
          location,
          '--root_dir',
          f'{PIPELINE_ROOT_PLACEHOLDER}/{PIPELINE_JOB_ID_PLACEHOLDER}-{PIPELINE_TASK_ID_PLACEHOLDER}',
          IfPresentPlaceholder(
              input_name='test_dataset',
              then=[
                  '--test_dataset',
                  test_dataset.metadata['resourceName'],
                  '--test_dataset_annotation_set_name',
                  test_dataset_annotation_set_name,
              ],
          ),
          IfPresentPlaceholder(
              input_name='training_dataset',
              then=[
                  '--training_dataset',
                  training_dataset.metadata['resourceName'],
                  '--training_dataset_annotation_set_name',
                  training_dataset_annotation_set_name,
              ],
          ),
          '--test_dataset_storage_source_uris',
          test_dataset_storage_source_uris,
          '--training_dataset_storage_source_uris',
          training_dataset_storage_source_uris,
          '--batch_prediction_storage_source',
          batch_prediction_storage_source,
          '--model_evaluation_storage_source',
          model_evaluation_storage_source,
          '--test_data_items_storage_source',
          test_data_items_storage_source,
          '--training_data_items_storage_source',
          training_data_items_storage_source,
          '--gcp_resources',
          gcp_resources,
          '--executor_input',
          '{{$}}',
      ],
  )
