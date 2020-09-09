#!/bin/bash
#
# Copyright 2019 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# This script lets you build the API server image. It is mostly useful in
# build/test infrastructure that uses Remote Build Execution. Since access to
# RBE requires authentication credentials, this script lets you pass those
# credentials to the docker builder to be used by Bazel.
set -e

usage() {
  cat <<EOM
  Usage:
    $0 [ --use_remote_build ] [ --gcp_credentials_file <FILE> ] -i IMAGE_TAG

      --use_remote_build       Use Remote Build Execution (RBE). Defaults to true.
      --gcp_credentials_file   Path to JSON file with credentials for RBE.
EOM
}

OPTS=i:
LONGOPTS=use_remote_build,gcp_credentials_file:

PARSED=$(getopt --longoptions=$LONGOPTS --options=$OPTS --name "$0" -- "$@")
eval set -- "$PARSED"

USE_REMOTE_BUILD=false
GCP_CREDENTIALS_FILE="gs://ml-pipeline-test-bazel/ml-pipeline-test-bazel-builder-credentials.json"
MACHINE_ARCH=`uname -m`

while true; do
  case $1 in
    --use_remote_build)
      USE_REMOTE_BUILD=true;
      shift;;
    --gcp_credentials_file)
      GCP_CREDENTIALS_FILE=$2;
      shift 2;;
    -i)
      IMAGE_TAG=$2;
      shift 2;;
    --)
      shift;
      break;;
    *)
      usage;
      exit 1;;
  esac
done

if [[ -z "${IMAGE_TAG}" ]]; then
    echo "Error: no image tag specified"
    usage
    exit 1
fi

WORKING_DIRECTORY="$(cd $(dirname "${BASH_SOURCE[0]}")/../. && pwd )"
cd ${WORKING_DIRECTORY}

if [[ ${USE_REMOTE_BUILD} == true ]]; then
  echo "Building API Server Using Remote Build Execution..."
  if [[ -z "${GCP_CREDENTIALS_FILE}" ]]; then
    echo "Error: GCP_CREDENTIALS_FILE not specified"
    exit 1
  fi

  GCP_CREDENTIALS="$(gsutil cat ${GCP_CREDENTIALS_FILE})"
  if [ $MACHINE_ARCH == "aarch64" ]; then
    docker build \
      -t bazel:0.24.0 \
      -f backend/Dockerfile.bazel .

    docker build \
      -t "${IMAGE_TAG}" \
      -f backend/Dockerfile \
      . \
      --build-arg use_remote_build=true \
      --build-arg google_application_credentials="${GCP_CREDENTIALS}" \
      --build-arg BAZEL_IMAGE=bazel:0.24.0
  else
    docker build \
      -t "${IMAGE_TAG}" \
      -f backend/Dockerfile \
      . \
      --build-arg use_remote_build=true \
      --build-arg google_application_credentials="${GCP_CREDENTIALS}"
  fi

else
  echo "Building API Server with local execution..."
  if [ $MACHINE_ARCH == "aarch64" ]; then
    docker build \
      -t bazel:0.24.0 \
      -f backend/Dockerfile.bazel .

    docker build \
      -t "${IMAGE_TAG}" \
      -f backend/Dockerfile \
      . \
      --build-arg BAZEL_IMAGE=bazel:0.24.0
  else
    docker build \
      -t "${IMAGE_TAG}" \
      -f backend/Dockerfile \
      .
  fi
fi
