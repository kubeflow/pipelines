#!/bin/bash -e
# Copyright 2021 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


cd "$(dirname "$0")"

project_id="${1}-ml-pipeline"

if [ -z "$project_id" ];  then
  echo "Usage: ./build_image.sh <YOUR_PROJECT_ID>"
  exit 1
fi

image_name="gcr.io/${project_id}/notebook-executor"
image_tag="latest"
base_python_tag="3.9-slim"
docker_image=${image_name}:${image_tag}

cat <<EOT > /tmp/cloudbuild.yaml
steps:
- name: "gcr.io/cloud-builders/docker"
  args:
  - build
  - "--build-arg=BASE_PYTHON_TAG=${base_python_tag}"
  - "--tag=${docker_image}"
  - "--file=./Dockerfile"
  - .
images:
- ${docker_image}
EOT

gcloud --project "${project_id}" builds submit --config=/tmp/cloudbuild.yaml