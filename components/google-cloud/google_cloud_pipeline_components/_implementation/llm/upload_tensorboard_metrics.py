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
"""KFP Component that uploads tensorboard metrics."""

from google_cloud_pipeline_components import _image
from kfp import dsl

_TB_UPLOADER_SHELL_SCRIPT = """
set -e -x
TENSORBOARD_RESOURCE_ID="$0"
METRICS_DIRECTORY_URI="$1"
EXPERIMENT_NAME="$2"
TENSORBOARD_URI="$3"

mkdir -p "$(dirname ${TENSORBOARD_URI})"
if [ -z "${TENSORBOARD_RESOURCE_ID}" ];
then
  echo "TensorBoard ID is not set. Skip uploading the TensorBoard."
  echo -n "" > "${TENSORBOARD_URI}"
  exit 0
fi

if [ -z "${METRICS_DIRECTORY_URI}" ]; then
  echo "Metrics directory uri is not set."
  exit 1
elif [ -z "${EXPERIMENT_NAME}" ]; then
  echo "Experiment name is not set."
  exit 1
elif [ -z "${TENSORBOARD_URI}" ]; then
  echo "TensorBoard URI is not set."
  exit 1
fi

case "${METRICS_DIRECTORY_URI}" in
  "gs://"*) ;;
  "/gcs/"*)
    METRICS_DIRECTORY_URI=${METRICS_DIRECTORY_URI/"/gcs/"/"gs://"}
    echo "Replaced /gcs/ path with ${METRICS_DIRECTORY_URI}"
    ;;
  *)
    echo "Invalid metrics directory uri. Metrics directory uri must start with gs:// or /gcs/."
    exit 1
    ;;
esac

if [[ "${TENSORBOARD_RESOURCE_ID}" =~ ^projects/[^/]+/locations/[^/]+/tensorboards/[0-9]+$ ]]; then
  echo "Split tensorboard resource id"
  TENSORBOARD_RESOURCE_ARR=(${TENSORBOARD_RESOURCE_ID//\\// })
  PROJECT=${TENSORBOARD_RESOURCE_ARR[1]}
  LOCATION=${TENSORBOARD_RESOURCE_ARR[3]}
  TENSORBOARD_ID=${TENSORBOARD_RESOURCE_ARR[5]}
else
  echo '[ERROR]: Invalid format of tensorboard_resource_id. It must be a string with format projects/${PROJECT_NUMBER}/locations/${LOCATION}/tensorboards/${TENSORBOARD_ID}'
  exit 1
fi

set +e

/opt/conda/bin/tb-gcp-uploader --tensorboard_resource_name \\
  "${TENSORBOARD_RESOURCE_ID}" \\
  --logdir="${METRICS_DIRECTORY_URI}" \\
  --experiment_name="${EXPERIMENT_NAME}" \\
  --one_shot=True

if [ $? -ne 0 ]; then
  exit 13
fi

set -e

web_server_uri="tensorboard.googleusercontent.com"
tensorboard_resource_name_uri="projects+${PROJECT}+locations+${LOCATION}+tensorboards+${TENSORBOARD_ID}+experiments+${EXPERIMENT_NAME}"
echo -n "https://${LOCATION}.${web_server_uri}/experiment/${tensorboard_resource_name_uri}" > "${TENSORBOARD_URI}"
"""


@dsl.container_component
def upload_tensorboard_metrics(
    tensorboard_resource_id: str,
    experiment_name: str,
    metrics_directory: dsl.Input[dsl.Artifact],  # pytype: disable=unsupported-operands
    tensorboard_uri: dsl.OutputPath(str),  # pytype: disable=invalid-annotation
) -> dsl.ContainerSpec:
  # fmt: off
  # pylint: disable=g-doc-args
  """Uploads tensorboard metrics.

  Args:
      tensorboard_resource_id: TensorBoard resource ID in the form `projects/{project_number}/locations/{location}/tensorboards/{tensorboard_id}`.
      experiment_name: Name of this tensorboard experiment. Must be unique to a given `projects/{project_number}/locations/{location}/tensorboards/{tensorboard_id}`.
      metrics_directory_uri: Cloud storage location of the TensorBoard logs.

  Returns:
      tensorboard_uri: URI of the uploaded tensorboard experiment.
  """
  # pylint: enable=g-doc-args
  # fmt: on
  return dsl.ContainerSpec(
      image='us-docker.pkg.dev/vertex-ai/training/tf-cpu.2-11:latest',
      command=[
          'bash',
          '-c',
          _TB_UPLOADER_SHELL_SCRIPT,
      ],
      args=[
          tensorboard_resource_id,
          metrics_directory.path,
          experiment_name,
          tensorboard_uri,
      ],
  )
