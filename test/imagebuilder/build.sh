#!/bin/bash
#
# Copyright 2018 Google LLC
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

set -xe

DOCKER_FILE=Dockerfile

usage()
{
    echo "usage: build.sh
    [--image-build-context-gcs-uri   GCS URI pointing to a .tar.gz archive of Docker build context]
    [--docker_path  path to the Dockerfile]
    [--docker_file  name of the Docker file. Dockerfile by default]
    [--image_name   project of the GCR to upload image to]
    [--build_script path to the build script that builds the image. If specified, --docker_path and --docker_file are not required.]
    [-h help]"
}

while [ "$1" != "" ]; do
    case $1 in
             --image-build-context-gcs-uri ) shift
                                CONTEXT_GCS_URI=$1
                                ;;
             --docker_path )    shift
                                DOCKER_PATH=$1
                                ;;
             --docker_file )    shift
                                DOCKER_FILE=$1
                                ;;
             --image_name )   shift
                                IMAGE_NAME=$1
                                ;;
             --build_script )   shift
                                BUILD_SCRIPT=$1
                                ;;
             -h | --help )      usage
                                exit
                                ;;
             * )                usage
                                exit 1
    esac
    shift
done

BASE_DIR=/image_builder
mkdir $BASE_DIR
cd $BASE_DIR

if [[ ! -z "${GOOGLE_APPLICATION_CREDENTIALS}" ]]; then
  gcloud auth activate-service-account --key-file="${GOOGLE_APPLICATION_CREDENTIALS}"
fi

echo "Downloading Docker build context from $CONTEXT_GCS_URI..."
downloaded_code_archive_file=$(mktemp)
gsutil cp "$CONTEXT_GCS_URI" "$downloaded_code_archive_file"
tar -xzf "$downloaded_code_archive_file" --directory .

echo "Waiting for Docker-in-Docker daemon to start..."
until docker ps; do sleep 3; done;

gcloud auth configure-docker

if [ "$BUILD_SCRIPT" == "" ]; then
  echo "Build image ${IMAGE_NAME} using ${BASE_DIR}/${DOCKER_PATH}/${DOCKER_FILE}..."
  docker build -t ${IMAGE_NAME} -f ${BASE_DIR}/${DOCKER_PATH}/${DOCKER_FILE} ${DOCKER_PATH}
else
  echo "Build image ${IMAGE_NAME} using ${BUILD_SCRIPT}..."
  cd $(dirname ${BUILD_SCRIPT})
  bash $(basename ${BUILD_SCRIPT}) -i ${IMAGE_NAME}
fi

echo "Pushing image ${IMAGE_NAME}..."
docker push ${IMAGE_NAME}

#Output the strict image name (which contains the sha256 image digest)
#This name can be used by the subsequent steps to refer to the exact image that was built even if another image with the same name was pushed.
image_name_with_digest=$(docker inspect --format="{{index .RepoDigests 0}}" "$IMAGE_NAME")
strict_image_name_output_file=/outputs/strict-image-name/file
mkdir -p "$(dirname "$strict_image_name_output_file")"
echo $image_name_with_digest > "$strict_image_name_output_file"
