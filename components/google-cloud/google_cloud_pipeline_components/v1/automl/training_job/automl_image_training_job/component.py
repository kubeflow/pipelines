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


from typing import Dict, List, Optional

from google_cloud_pipeline_components import _image
from google_cloud_pipeline_components.types.artifact_types import VertexDataset
from google_cloud_pipeline_components.types.artifact_types import VertexModel
from kfp import dsl
from kfp.dsl import Input
from kfp.dsl import Output
from kfp.dsl import OutputPath


@dsl.container_component
def automl_image_training_job(
    project: str,
    display_name: str,
    dataset: Input[VertexDataset],
    model: Output[VertexModel],
    gcp_resources: OutputPath(str),
    location: Optional[str] = 'us-central1',
    prediction_type: Optional[str] = 'classification',
    multi_label: Optional[bool] = False,
    model_type: Optional[str] = 'CLOUD',
    base_model: Optional[Input[VertexModel]] = None,
    incremental_train_base_model: Optional[Input[VertexModel]] = None,
    parent_model: Optional[Input[VertexModel]] = None,
    is_default_version: Optional[bool] = True,
    model_version_aliases: Optional[List[str]] = None,
    model_version_description: Optional[str] = None,
    labels: Optional[Dict[str, str]] = {},
    training_encryption_spec_key_name: Optional[str] = None,
    model_encryption_spec_key_name: Optional[str] = None,
    training_fraction_split: Optional[float] = None,
    validation_fraction_split: Optional[float] = None,
    test_fraction_split: Optional[float] = None,
    training_filter_split: Optional[str] = None,
    validation_filter_split: Optional[str] = None,
    test_filter_split: Optional[str] = None,
    budget_milli_node_hours: Optional[int] = None,
    model_display_name: Optional[str] = None,
    model_labels: Optional[Dict[str, str]] = None,
    disable_early_stopping: Optional[bool] = False,
):
  # fmt: off
  """Runs the AutoML Image training job and returns a model.

  If training on a Vertex AI dataset, you can use one of the following split configurations: Data fraction splits: Any of `training_fraction_split`, `validation_fraction_split` and `test_fraction_split` may optionally be provided, they must sum to up to 1. If the provided ones sum to less than 1, the remainder is assigned to sets as decided by Vertex AI. If none of the fractions are set, by default roughly 80% of data will be used for training, 10% for validation, and 10% for test. Data filter splits: Assigns input data to training, validation, and test sets based on the given filters, data pieces not matched by any filter are ignored. Currently only supported for Datasets containing DataItems. If any of the filters in this message are to match nothing, then they can be set as '-' (the minus sign). If using filter splits, all of `training_filter_split`, `validation_filter_split` and `test_filter_split` must be provided. Supported only for unstructured Datasets.

  Args:
      dataset: The dataset within the same Project from which data will be used to train the Model. The Dataset must use schema compatible with Model being trained, and what is compatible should be described in the used TrainingPipeline's [training_task_definition] [google.cloud.aiplatform.v1beta1.TrainingPipeline.training_task_definition]. For tabular Datasets, all their data is exported to training, to pick and choose from.
      training_fraction_split: The fraction of the input data that is to be used to train the Model. This is ignored if Dataset is not provided.
      validation_fraction_split: The fraction of the input data that is to be used to validate the Model. This is ignored if Dataset is not provided.
      test_fraction_split: The fraction of the input data that is to be used to evaluate the Model. This is ignored if Dataset is not provided.
      training_filter_split: A filter on DataItems of the Dataset. DataItems that match this filter are used to train the Model. A filter with same syntax as the one used in DatasetService.ListDataItems may be used. If a single DataItem is matched by more than one of the FilterSplit filters, then it is assigned to the first set that applies to it in the training, validation, test order. This is ignored if Dataset is not provided. Example usage: training_filter_split="labels.aiplatform.googleapis.com/ml_use=training".
      validation_filter_split: A filter on DataItems of the Dataset. DataItems that match this filter are used to validate the Model. A filter with same syntax as the one used in DatasetService.ListDataItems may be used. If a single DataItem is matched by more than one of the FilterSplit filters, then it is assigned to the first set that applies to it in the training, validation, test order. This is ignored if Dataset is not provided. Example usage: validation_filter_split= "labels.aiplatform.googleapis.com/ml_use=validation".
      test_filter_split: A filter on DataItems of the Dataset. DataItems that match this filter are used to test the Model. A filter with same syntax as the one used in DatasetService.ListDataItems may be used. If a single DataItem is matched by more than one of the FilterSplit filters, then it is assigned to the first set that applies to it in the training, validation, test order. This is ignored if Dataset is not provided. Example usage: test_filter_split= "labels.aiplatform.googleapis.com/ml_use=test".
      budget_milli_node_hours: The train budget of creating this Model, expressed in milli node hours i.e. 1,000 value in this field means 1 node hour. Defaults by `prediction_type`: `classification` - For Cloud models the budget must be: 8,000 - 800,000 milli node hours (inclusive). The default value is 192,000 which represents one day in wall time, assuming 8 nodes are used. `object_detection` - For Cloud models the budget must be: 20,000 - 900,000 milli node hours (inclusive). The default value is 216,000 which represents one day in wall time, assuming 9 nodes are used. The training cost of the model will not exceed this budget. The final cost will be attempted to be close to the budget, though may end up being (even) noticeably smaller - at the backend's discretion. This especially may happen when further model training ceases to provide any improvements. If the budget is set to a value known to be insufficient to train a Model for the given training set, the training won't be attempted and will error.
      model_display_name: The display name of the managed Vertex AI Model. The name can be up to 128 characters long and can be consist of any UTF-8 characters. If not provided upon creation, the job's display_name is used.
      model_labels: The labels with user-defined metadata to organize your Models. Label keys and values can be no longer than 64 characters (Unicode codepoints), can only contain lowercase letters, numeric characters, underscores and dashes. International characters are allowed. See https://goo.gl/xmQnxf for more information and examples of labels.
      disable_early_stopping: If true, the entire budget is used. This disables the early stopping feature. By default, the early stopping feature is enabled, which means that training might stop before the entire training budget has been used, if further training does no longer brings significant improvement to the model.
      display_name: The user-defined name of this TrainingPipeline.
      prediction_type: The type of prediction the Model is to produce, one of: "classification" - Predict one out of multiple target values is picked for each row. "object_detection" - Predict a value based on its relation to other values. This type is available only to columns that contain semantically numeric values, i.e. integers or floating point number, even if stored as e.g. strings.
      multi_label: Default is False. If false, a single-label (multi-class) Model will be trained (i.e. assuming that for each image just up to one annotation may be applicable). If true, a multi-label Model will be trained (i.e. assuming that for each image multiple annotations may be applicable). This is only applicable for the "classification" prediction_type and will be ignored otherwise.
      model_type: One of the following: "CLOUD" - Default for Image Classification. A Model best tailored to be used within Google Cloud, and which cannot be exported. "CLOUD_HIGH_ACCURACY_1" - Default for Image Object Detection. A model best tailored to be used within Google Cloud, and which cannot be exported. Expected to have a higher latency, but should also have a higher prediction quality than other cloud models. "CLOUD_LOW_LATENCY_1" - A model best tailored to be used within Google Cloud, and which cannot be exported. Expected to have a low latency, but may have lower prediction quality than other cloud models. "MOBILE_TF_LOW_LATENCY_1" - A model that, in addition to being available within Google Cloud, can also be exported as TensorFlow or Core ML model and used on a mobile or edge device afterwards. Expected to have low latency, but may have lower prediction quality than other mobile models. "MOBILE_TF_VERSATILE_1" - A model that, in addition to being available within Google Cloud, can also be exported as TensorFlow or Core ML model and used on a mobile or edge device with afterwards. "MOBILE_TF_HIGH_ACCURACY_1" - A model that, in addition to being available within Google Cloud, can also be exported as TensorFlow or Core ML model and used on a mobile or edge device afterwards. Expected to have a higher latency, but should also have a higher prediction quality than other mobile models.
      base_model: Only permitted for Image Classification models. If it is specified, the new model will be trained based on the `base` model. Otherwise, the new model will be trained from scratch. The `base` model must be in the same Project and Location as the new Model to train, and have the same model_type.
      incremental_train_base_model: Optional for both Image Classification and Object detection models, to incrementally train a new model using an existing model as the starting point, with a reduced training time. If not specified, the new model will be trained from scratch. The `base` model must be in the same Project and Location as the new Model to train, and have the same prediction_type and model_type.
      parent_model: The resource name or model ID of an existing model. The new model uploaded by this job will be a version of `parent_model`.  Only set this field when training a new version of an existing model.
      is_default_version: When set to True, the newly uploaded model version will automatically have alias "default" included. Subsequent uses of the model produced by this job without a version specified will use this "default" version.  When set to False, the "default" alias will not be moved. Actions targeting the model version produced by this job will need to specifically reference this version by ID or alias.  New model uploads, i.e. version 1, will always be "default" aliased.
      model_version_aliases: User provided version aliases so that the model version uploaded by this job can be referenced via alias instead of auto-generated version ID. A default version alias will be created for the first version of the model. The format is [a-z][a-zA-Z0-9-]{0,126}[a-z0-9]
      model_version_description: The description of the model version being uploaded by this job.
      project: Project to retrieve dataset from.
      location: Optional location to retrieve dataset from.
      labels: The labels with user-defined metadata to organize TrainingPipelines. Label keys and values can be no longer than 64 characters (Unicode codepoints), can only contain lowercase letters, numeric characters, underscores and dashes. International characters are allowed. See https://goo.gl/xmQnxf for more information and examples of labels.
      training_encryption_spec_key_name: The Cloud KMS resource identifier of the customer managed encryption key used to protect the training pipeline. Has the form: `projects/my-project/locations/my-region/keyRings/my-kr/cryptoKeys/my-key`. The key needs to be in the same region as where the compute resource is created. If set, this TrainingPipeline will be secured by this key. Note: Model trained by this TrainingPipeline is also secured by this key if `model_to_upload` is not set separately. Overrides encryption_spec_key_name set in aiplatform.init.
      model_encryption_spec_key_name: The Cloud KMS resource identifier of the customer managed encryption key used to protect the model. Has the form: `projects/my-project/locations/my-region/keyRings/my-kr/cryptoKeys/my-key`. The key needs to be in the same region as where the compute resource is created. If set, the trained Model will be secured by this key. Overrides encryption_spec_key_name set in aiplatform.init.

  Returns:
      model: The trained Vertex AI Model resource or None if training did not produce a Vertex AI Model.
      gcp_resources: Serialized gcp_resources proto tracking the batch prediction job. For more details, see https://github.com/kubeflow/pipelines/blob/master/components/google-cloud/google_cloud_pipeline_components/proto/README.md.
  """
  # fmt: on

  return dsl.ContainerSpec(
      image=_image.GCPC_IMAGE_TAG,
      command=[
          'python3',
          '-m',
          'google_cloud_pipeline_components.container.v1.automl_training_job.image.launcher',
      ],
      args=[
          '--type',
          'AutoMLImageTrainingJob',
          '--project',
          project,
          '--location',
          location,
          '--display_name',
          display_name,
          '--prediction_type',
          prediction_type,
          '--multi_label',
          multi_label,
          '--model_type',
          model_type,
          '--labels',
          labels,
          '--dataset',
          dataset.metadata['resourceName'],
          '--disable_early_stopping',
          disable_early_stopping,
          dsl.IfPresentPlaceholder(
              input_name='training_encryption_spec_key_name',
              then=[
                  '--training_encryption_spec_key_name',
                  training_encryption_spec_key_name,
              ],
          ),
          dsl.IfPresentPlaceholder(
              input_name='model_encryption_spec_key_name',
              then=[
                  '--model_encryption_spec_key_name',
                  model_encryption_spec_key_name,
              ],
          ),
          dsl.IfPresentPlaceholder(
              input_name='model_display_name',
              then=['--model_display_name', model_display_name],
          ),
          dsl.IfPresentPlaceholder(
              input_name='training_fraction_split',
              then=[
                  '--training_fraction_split',
                  training_fraction_split,
              ],
          ),
          dsl.IfPresentPlaceholder(
              input_name='validation_fraction_split',
              then=[
                  '--validation_fraction_split',
                  validation_fraction_split,
              ],
          ),
          dsl.IfPresentPlaceholder(
              input_name='test_fraction_split',
              then=['--test_fraction_split', test_fraction_split],
          ),
          dsl.IfPresentPlaceholder(
              input_name='budget_milli_node_hours',
              then=[
                  '--budget_milli_node_hours',
                  budget_milli_node_hours,
              ],
          ),
          dsl.IfPresentPlaceholder(
              input_name='training_filter_split',
              then=['--training_filter_split', training_filter_split],
          ),
          dsl.IfPresentPlaceholder(
              input_name='validation_filter_split',
              then=[
                  '--validation_filter_split',
                  validation_filter_split,
              ],
          ),
          dsl.IfPresentPlaceholder(
              input_name='test_filter_split',
              then=['--test_filter_split', test_filter_split],
          ),
          dsl.IfPresentPlaceholder(
              input_name='base_model',
              then=[
                  '--base_model',
                  base_model.metadata['resourceName'],
                  '--model_labels',
                  base_model.metadata['labels'],
              ],
              else_=[
                  dsl.IfPresentPlaceholder(
                      input_name='model_labels',
                      then=['--model_labels', model_labels],
                  )
              ],
          ),
          dsl.IfPresentPlaceholder(
              input_name='incremental_train_base_model',
              then=[
                  '--incremental_train_base_model',
                  incremental_train_base_model.metadata['resourceName'],
              ],
          ),
          dsl.IfPresentPlaceholder(
              input_name='parent_model',
              then=[
                  '--parent_model',
                  parent_model.metadata['resourceName'],
              ],
          ),
          dsl.IfPresentPlaceholder(
              input_name='is_default_version',
              then=[
                  '--is_default_version',
                  is_default_version,
              ],
          ),
          dsl.IfPresentPlaceholder(
              input_name='model_version_aliases',
              then=[
                  '--model_version_aliases',
                  model_version_aliases,
              ],
          ),
          dsl.IfPresentPlaceholder(
              input_name='model_version_description',
              then=[
                  '--model_version_description',
                  model_version_description,
              ],
          ),
          '--gcp_resources',
          gcp_resources,
          '--executor_input',
          '{{$}}',
          '--resource_name_output_artifact_uri',
          model.uri,
      ],
  )
