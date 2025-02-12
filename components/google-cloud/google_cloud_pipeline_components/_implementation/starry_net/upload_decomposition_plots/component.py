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
"""Starry Net upload decomposition plots component."""

from kfp import dsl


@dsl.component(
    packages_to_install=[
        'google-cloud-aiplatform[tensorboard]',
        'protobuf==3.20.*',
    ]
)
def upload_decomposition_plots(
    project: str,
    location: str,
    tensorboard_id: str,
    display_name: str,
    trainer_dir: dsl.InputPath(),
) -> dsl.Artifact:
  # fmt: off
  """Uploads decomposition plots to Tensorboard.

  Args:
    project: The project where the pipeline is run. Defaults to current project.
    location: The location where the pipeline components are run.
    tensorboard_id: The tensorboard instance ID.
    display_name: The diplay name of the job.
    trainer_dir: The directory where training artifacts where stored.

  Returns:
    A dsl.Artifact where the URI is the URI where decomposition plots can be
    viewed.
  """
  import os  # pylint: disable=g-import-not-at-top
  from google.cloud import aiplatform  # pylint: disable=g-import-not-at-top

  log_dir = os.path.join(trainer_dir, 'tensorboard', 'r=1:gc=0')
  project_number = os.environ['CLOUD_ML_PROJECT_ID']
  aiplatform.init(project=project, location=location)
  aiplatform.upload_tb_log(
      tensorboard_id=tensorboard_id,
      tensorboard_experiment_name=display_name,
      logdir=log_dir,
      experiment_display_name=display_name,
      description=f'Tensorboard for {display_name}',
  )
  uri = (
      f'https://{location}.tensorboard.googleusercontent.com/experiment/'
      f'projects+{project_number}+locations+{location}+tensorboards+'
      f'{tensorboard_id}+experiments+{display_name}/#images'
  )
  return dsl.Artifact(uri=uri)
