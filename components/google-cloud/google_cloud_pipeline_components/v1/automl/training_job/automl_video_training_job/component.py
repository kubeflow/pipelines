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
def automl_video_training_job(
    project: str,
    display_name: str,
    dataset: Input[VertexDataset],
    model: Output[VertexModel],
    location: Optional[str] = 'us-central1',
    prediction_type: Optional[str] = 'classification',
    model_type: Optional[str] = 'CLOUD',
    labels: Optional[dict] = {},
    training_encryption_spec_key_name: Optional[str] = None,
    model_encryption_spec_key_name: Optional[str] = None,
    training_fraction_split: Optional[float] = None,
    test_fraction_split: Optional[float] = None,
    model_display_name: Optional[str] = None,
    model_labels: Optional[dict] = None,
):
  # fmt: off
  """Runs the AutoML Video training job and returns a model.

  If training on a Vertex AI dataset, you can use one of the following split configurations:

  Data fraction splits:
  ``training_fraction_split``, and ``test_fraction_split`` may optionally
  be provided, they must sum to up to 1. If none of the fractions are set,
  by default roughly 80% of data will be used for training, and 20% for test.
  Data filter splits:
  Assigns input data to training, validation, and test sets
  based on the given filters, data pieces not matched by any
  filter are ignored. Currently only supported for Datasets
  containing DataItems.
  If any of the filters in this message are to match nothing, then
  they can be set as '-' (the minus sign).

  Supported only for unstructured Datasets.

  Args:
      dataset: The dataset within the same Project from which data will be used to train the Model. The
          Dataset must use schema compatible with Model being trained,
          and what is compatible should be described in the used
          TrainingPipeline's [training_task_definition]
          [google.cloud.aiplatform.v1beta1.TrainingPipeline.training_task_definition].
          For tabular Datasets, all their data is exported to
          training, to pick and choose from.
      training_fraction_split: The fraction of the input data that is to be used to train
          the Model. This is ignored if Dataset is not provided.
      test_fraction_split: The fraction of the input data that is to be used to evaluate
          the Model. This is ignored if Dataset is not provided.
      model_display_name: The display name of the managed Vertex AI Model. The name
          can be up to 128 characters long and can be consist of any UTF-8
          characters. If not provided upon creation, the job's display_name is used.
      model_labels: The labels with user-defined metadata to
          organize your Models.
          Label keys and values can be no longer than 64
          characters (Unicode codepoints), can only
          contain lowercase letters, numeric characters,
          underscores and dashes. International characters
          are allowed.
          See https://goo.gl/xmQnxf for more information
          and examples of labels.
      display_name: The user-defined name of this TrainingPipeline.
      prediction_type: The type of prediction the Model is to produce, one of:
              "classification" - A video classification model classifies shots and segments in your videos according to your own defined labels.
              "object_tracking" - A video object tracking model detects and tracks multiple objects in shots and segments. You can use these models to track objects in your videos according to your own pre-defined, custom labels.
              "action_recognition" - A video action reconition model pinpoints the location of actions with short temporal durations (~1 second).
      model_type: str = "CLOUD"
          One of the following:
              "CLOUD" - available for "classification", "object_tracking" and "action_recognition"
                  A Model best tailored to be used within Google Cloud,
                  and which cannot be exported.
              "MOBILE_VERSATILE_1" - available for "classification", "object_tracking" and "action_recognition"
                  A model that, in addition to being available within Google
                  Cloud, can also be exported (see ModelService.ExportModel)
                  as a TensorFlow or TensorFlow Lite model and used on a
                  mobile or edge device with afterwards.
              "MOBILE_CORAL_VERSATILE_1" - available only for "object_tracking"
                  A versatile model that is meant to be exported (see
                  ModelService.ExportModel) and used on a Google Coral device.
              "MOBILE_CORAL_LOW_LATENCY_1" - available only for "object_tracking"
                  A model that trades off quality for low latency, to be
                  exported (see ModelService.ExportModel) and used on a
                  Google Coral device.
              "MOBILE_JETSON_VERSATILE_1" - available only for "object_tracking"
                  A versatile model that is meant to be exported (see
                  ModelService.ExportModel) and used on an NVIDIA Jetson device.
              "MOBILE_JETSON_LOW_LATENCY_1" - available only for "object_tracking"
                  A model that trades off quality for low latency, to be
                  exported (see ModelService.ExportModel) and used on an
                  NVIDIA Jetson device.
      project: Project to retrieve dataset from.
      location: Optional location to retrieve dataset from.
      labels: The labels with user-defined metadata to
          organize TrainingPipelines.
          Label keys and values can be no longer than 64
          characters (Unicode codepoints), can only
          contain lowercase letters, numeric characters,
          underscores and dashes. International characters
          are allowed.
          See https://goo.gl/xmQnxf for more information
          and examples of labels.
      training_encryption_spec_key_name: The Cloud KMS resource identifier of the customer
          managed encryption key used to protect the training pipeline. Has the
          form:
          ``projects/my-project/locations/my-region/keyRings/my-kr/cryptoKeys/my-key``.
          The key needs to be in the same region as where the compute
          resource is created.
          If set, this TrainingPipeline will be secured by this key.
          Note: Model trained by this TrainingPipeline is also secured
          by this key if ``model_to_upload`` is not set separately.
          Overrides encryption_spec_key_name set in aiplatform.init.
      model_encryption_spec_key_name: The Cloud KMS resource identifier of the customer
          managed encryption key used to protect the model. Has the
          form:
          ``projects/my-project/locations/my-region/keyRings/my-kr/cryptoKeys/my-key``.
          The key needs to be in the same region as where the compute
          resource is created.
          If set, the trained Model will be secured by this key.
          Overrides encryption_spec_key_name set in aiplatform.init.

  Returns:
      model: The trained Vertex AI Model resource or None if training did not
          produce a Vertex AI Model.
  """
  # fmt`:` on

  return dsl.ContainerSpec(
      image=_image.GCPC_IMAGE_TAG,
      command=[
          'python3',
          '-m',
          'google_cloud_pipeline_components.container.v1.aiplatform.remote_runner',
          '--cls_name',
          'AutoMLVideoTrainingJob',
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
          '--init.labels',
          labels,
          '--init.model_type',
          model_type,
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
