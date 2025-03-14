# Copyright 2024 The Kubeflow Authors. All Rights Reserved.
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
"""StarryNet get training artifacts component."""

from typing import NamedTuple

from kfp import dsl


@dsl.component(packages_to_install=['tensorflow==2.11.0'])
def get_training_artifacts(
    docker_region: str,
    trainer_dir: dsl.InputPath(),
) -> NamedTuple(
    'TrainingArtifacts',
    image_uri=str,
    artifact_uri=str,
    prediction_schema_uri=str,
    instance_schema_uri=str,
):
  # fmt: off
  """Gets the artifact URIs from the training job.

  Args:
    docker_region: The region from which the training docker image is pulled.
    trainer_dir: The directory where training artifacts where stored.

  Returns:
    A NamedTuple containing the image_uri for the prediction server,
    the artifact_uri with model artifacts, the prediction_schema_uri,
    and the instance_schema_uri.
  """
  import os  # pylint: disable=g-import-not-at-top
  import tensorflow as tf  # pylint: disable=g-import-not-at-top

  with tf.io.gfile.GFile(os.path.join(trainer_dir, 'trainer.txt')) as f:
    private_dir = f.read().strip()

  outputs = NamedTuple(
      'TrainingArtifacts',
      image_uri=str,
      artifact_uri=str,
      prediction_schema_uri=bool,
      instance_schema_uri=str,
  )
  return outputs(
      f'{docker_region}-docker.pkg.dev/vertex-ai/starryn/predictor:20240723_0542_RC00',  # pylint: disable=too-many-function-args
      private_dir,  # pylint: disable=too-many-function-args
      os.path.join(private_dir, 'predict_schema.yaml'),  # pylint: disable=too-many-function-args
      os.path.join(private_dir, 'instance_schema.yaml'),  # pylint: disable=too-many-function-args
  )
