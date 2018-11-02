#!/bin/bash -e
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


while getopts ":hp:t:i:" opt; do
  case "${opt}" in
    h) echo "-p: project name"
        echo "-t: tag name"
        echo "-i: image name. If provided, project name and tag name are not necessary"
        exit
      ;;
    p) PROJECT_ID=${OPTARG}
      ;;
    t) TAG_NAME=${OPTARG}
      ;;
    i) LAUNCHER_IMAGE_NAME=${OPTARG}
      ;;
    \? ) echo "Usage: cmd [-p] project [-t] tag [-i] image"
      exit
      ;;
  esac
done

LOCAL_LAUNCHER_IMAGE_NAME=ml-pipeline-kubeflow-tf
LOCAL_TRAINER_IMAGE_NAME=ml-pipeline-kubeflow-tf-trainer

if [ -z "${PROJECT_ID}" ]; then
  PROJECT_ID=$(gcloud config config-helper --format "value(configuration.properties.core.project)")
fi

if [ -z "${TAG_NAME}" ]; then
  TAG_NAME="latest"
fi

mkdir -p ./build
rsync -arvp "../../launcher"/ ./build/

cp ../../../license.sh ./build
cp ../../../third_party_licenses.csv ./build

# Build the trainer image
if [ -z "${LAUNCHER_IMAGE_NAME}" ]; then
  TRAINER_IMAGE_NAME=gcr.io/${PROJECT_ID}/${LOCAL_TRAINER_IMAGE_NAME}:${TAG_NAME}
else
  # construct the trainer image name as "laucher_image_name"-trainer:"launcher_image_tag"
  colon_index=`expr index "${LAUNCHER_IMAGE_NAME}" :`
  if [ $colon_index == '0' ]; then
    TRAINER_IMAGE_NAME=${LAUNCHER_IMAGE_NAME}-trainer
  else
    tag=${LAUNCHER_IMAGE_NAME:$colon_index}
    TRAINER_IMAGE_NAME=${LAUNCHER_IMAGE_NAME:0:$colon_index-1}-trainer:${tag}
  fi
fi

bash_dir=`dirname $0`
bash_dir_abs=`realpath $bash_dir`
parent_dir=`dirname ${bash_dir_abs}`
trainer_dir=${parent_dir}/trainer
cd ${trainer_dir}
if [ -z "${LAUNCHER_IMAGE_NAME}" ]; then
  ./build.sh -p ${PROJECT_ID} -t ${TAG_NAME}
else
  ./build.sh -i ${TRAINER_IMAGE_NAME}
fi
cd -

docker build -t ${LOCAL_LAUNCHER_IMAGE_NAME} . --build-arg TRAINER_IMAGE_NAME=${TRAINER_IMAGE_NAME}
if [ -z "${LAUNCHER_IMAGE_NAME}" ]; then
  docker tag ${LOCAL_LAUNCHER_IMAGE_NAME} gcr.io/${PROJECT_ID}/${LOCAL_LAUNCHER_IMAGE_NAME}:${TAG_NAME}
  docker push gcr.io/${PROJECT_ID}/${LOCAL_LAUNCHER_IMAGE_NAME}:${TAG_NAME}
else
  docker tag ${LOCAL_LAUNCHER_IMAGE_NAME} "${LAUNCHER_IMAGE_NAME}"
  docker push "${LAUNCHER_IMAGE_NAME}"
fi

rm -rf ./build
