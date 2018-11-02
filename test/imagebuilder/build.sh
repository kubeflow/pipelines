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
    echo "usage: deploy.sh
    [--commit_sha   commit SHA to pull code from]
    [--docker_path  path to the Dockerfile]
    [--docker_file  name of the Docker file. Dockerfile by default]
    [--image_name   project of the GCR to upload image to]
    [--build_script path to the build script that builds the image. If specified, --docker_path and --docker_file are not required.]
    [-h help]"
}

while [ "$1" != "" ]; do
    case $1 in
             --commit_sha )     shift
                                COMMIT_SHA=$1
                                ;;
             --docker_path )    shift
                                DOCKER_PATH=$1
                                ;;
             --docker_file )    shift
                                DOCKER_FILE=$1
                                ;;
             --image_name )     shift
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

BASE_DIR=/ml

ssh-keygen -F github.com || ssh-keyscan github.com >>~/.ssh/known_hosts
cp ~/.ssh/github/* ~/.ssh

echo "Clone ML pipeline code in COMMIT SHA ${COMMIT_SHA}..."
git clone git@github.com:googleprivate/ml.git ${BASE_DIR}
cd ${BASE_DIR}
git checkout ${COMMIT_SHA}

echo "Waiting for dind to start..."
until docker ps; do sleep 3; done;

if [ "$BUILD_SCRIPT" == "" ]; then
  echo "Build image ${IMAGE_NAME} using ${BASE_DIR}/${DOCKER_PATH}/${DOCKER_FILE}..."
  docker build -t ${IMAGE_NAME} -f ${BASE_DIR}/${DOCKER_PATH}/${DOCKER_FILE} ${BASE_DIR}/${DOCKER_PATH}
else
  echo "Build image ${IMAGE_NAME} using ${BUILD_SCRIPT}..."
  cd $(dirname ${BUILD_SCRIPT})
  gcloud auth configure-docker
  bash $(basename ${BUILD_SCRIPT}) -i ${IMAGE_NAME}
fi

echo "Push image ${IMAGE_NAME} to gcr..."
gcloud auth configure-docker
docker push ${IMAGE_NAME}
