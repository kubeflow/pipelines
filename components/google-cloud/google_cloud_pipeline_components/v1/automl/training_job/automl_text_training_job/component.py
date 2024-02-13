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

from google_cloud_pipeline_components import _image
from google_cloud_pipeline_components.types.artifact_types import VertexDataset
from google_cloud_pipeline_components.types.artifact_types import VertexModel
from kfp import dsl
from kfp.dsl import Input
from kfp.dsl import Output


@dsl.container_component
def automl_text_training_job(
    project: str,
    display_name: str,
    dataset: Input[VertexDataset],
    model: Output[VertexModel],
    location: Optional[str] = 'us-central1',
    prediction_type: Optional[str] = 'classification',
    multi_label: Optional[bool] = False,
    labels: Optional[dict] = {},
    training_encryption_spec_key_name: Optional[str] = None,
    model_encryption_spec_key_name: Optional[str] = None,
    training_fraction_split: Optional[float] = None,
    validation_fraction_split: Optional[float] = None,
    test_fraction_split: Optional[float] = None,
    sentiment_max: Optional[int] = 10,
    model_display_name: Optional[str] = None,
    model_labels: Optional[dict] = None,
):
  # fmt: off
  """Runs the training job and returns a model.

  If training on a Vertex AI dataset, you can use one of the following split configurations: Data fraction splits: Any of `training_fraction_split`, `validation_fraction_split` and `test_fraction_split` may optionally be provided, they must sum to up to 1. If the provided ones sum to less than 1, the remainder is assigned to sets as decided by Vertex AI. If none of the fractions are set, by default roughly 80% of data will be used for training, 10% for validation, and 10% for test. Data filter splits: Assigns input data to training, validation, and test sets based on the given filters, data pieces not matched by any filter are ignored. Currently only supported for Datasets containing DataItems. If any of the filters in this message are to match nothing, then they can be set as '-' (the minus sign). Supported only for unstructured Datasets.

  Args:
      dataset: The dataset within the same Project from which data will be used to train the Model. The Dataset must use schema compatible with Model being trained, and what is compatible should be described in the used TrainingPipeline's [training_task_definition] [google.cloud.aiplatform.v1beta1.TrainingPipeline.training_task_definition].
      training_fraction_split: The fraction of the input data that is to be used to train the Model. This is ignored if Dataset is not provided.
      validation_fraction_split: The fraction of the input data that is to be used to validate the Model. This is ignored if Dataset is not provided.
      test_fraction_split: The fraction of the input data that is to be used to evaluate the Model. This is ignored if Dataset is not provided.
      model_display_name: The display name of the managed Vertex AI Model. The name can be up to 128 characters long and can consist of any UTF-8 characters. If not provided upon creation, the job's display_name is used.
      model_labels: The labels with user-defined metadata to organize your Models. Label keys and values can be no longer than 64 characters (Unicode codepoints), can only contain lowercase letters, numeric characters, underscores and dashes. International characters are allowed. See https://goo.gl/xmQnxf for more information and examples of labels.
      display_name: The user-defined name of this TrainingPipeline.
      prediction_type: The type of prediction the Model is to produce, one of: "classification" - A classification model analyzes text data and returns a list of categories that apply to the text found in the data. Vertex AI offers both single-label and multi-label text classification models. "extraction" - An entity extraction model inspects text data known entities referenced in the data and labels those entities in the text. "sentiment" - A sentiment analysis model inspects text data and identifies the prevailing emotional opinion within it, especially to determine a writer's attitude as positive, negative, or neutral.
      multi_label: Required and only applicable for text classification task. If false, a single-label (multi-class) Model will be trained (i.e. assuming that for each text snippet just up to one annotation may be applicable). If true, a multi-label Model will be trained (i.e. assuming that for each text snippet multiple annotations may be applicable).
      sentiment_max: Required and only applicable for sentiment task. A sentiment is expressed as an integer ordinal, where higher value means a more positive sentiment. The range of sentiments that will be used is between 0 and sentimentMax (inclusive on both ends), and all the values in the range must be represented in the dataset before a model can be created. Only the Annotations with this sentimentMax will be used for training. sentimentMax value must be between 1 and 10 (inclusive).
      project: Project to retrieve dataset from.
      location: Optional location to retrieve dataset from.
      labels: The labels with user-defined metadata to organize TrainingPipelines. Label keys and values can be no longer than 64 characters (Unicode codepoints), can only contain lowercase letters, numeric characters, underscores and dashes. International characters are allowed. See https://goo.gl/xmQnxf for more information and examples of labels.
      training_encryption_spec_key_name: The Cloud KMS resource identifier of the customer managed encryption key used to protect the training pipeline. Has the form: `projects/my-project/locations/my-region/keyRings/my-kr/cryptoKeys/my-key`. The key needs to be in the same region as where the compute resource is created. If set, this TrainingPipeline will be secured by this key. Note: Model trained by this TrainingPipeline is also secured by this key if `model_to_upload` is not set separately. Overrides encryption_spec_key_name set in aiplatform.init.
      model_encryption_spec_key_name: The Cloud KMS resource identifier of the customer managed encryption key used to protect the model. Has the form: `projects/my-project/locations/my-region/keyRings/my-kr/cryptoKeys/my-key`. The key needs to be in the same region as where the compute resource is created. If set, the trained Model will be secured by this key. Overrides encryption_spec_key_name set in aiplatform.init.

  Returns:
      model: The trained Vertex AI Model resource.
  """
  # fmt: on

  return dsl.ContainerSpec(
      image=_image.GCPC_IMAGE_TAG,
      command=[
          'python3',
          '-m',
          'google_cloud_pipeline_components.container.v1.aiplatform.remote_runner',
          '--cls_name',
          'AutoMLTextTrainingJob',
          '--method_name',
          'run',
      ],
      args=[
          '--init.project',
          project,
          '--init.location',
          location,
          '--init.display_name',
          display_name,
          '--init.prediction_type',
          prediction_type,
          '--init.multi_label',
          multi_label,
          '--init.labels',
          labels,
          '--init.sentiment_max',
          sentiment_max,
          '--method.dataset',
          dataset.metadata['resourceName'],
          dsl.IfPresentPlaceholder(
              input_name='training_encryption_spec_key_name',
              then=[
                  '--init.training_encryption_spec_key_name',
                  training_encryption_spec_key_name,
              ],
          ),
          dsl.IfPresentPlaceholder(
              input_name='model_encryption_spec_key_name',
              then=[
                  '--init.model_encryption_spec_key_name',
                  model_encryption_spec_key_name,
              ],
          ),
          dsl.IfPresentPlaceholder(
              input_name='model_display_name',
              then=['--method.model_display_name', model_display_name],
          ),
          dsl.IfPresentPlaceholder(
              input_name='training_fraction_split',
              then=[
                  '--method.training_fraction_split',
                  training_fraction_split,
              ],
          ),
          dsl.IfPresentPlaceholder(
              input_name='validation_fraction_split',
              then=[
                  '--method.validation_fraction_split',
                  validation_fraction_split,
              ],
          ),
          dsl.IfPresentPlaceholder(
              input_name='test_fraction_split',
              then=['--method.test_fraction_split', test_fraction_split],
          ),
          dsl.IfPresentPlaceholder(
              input_name='model_labels',
              then=['--method.model_labels', model_labels],
          ),
          '--executor_input',
          '{{$}}',
          '--resource_name_output_artifact_uri',
          model.uri,
      ],
  )
