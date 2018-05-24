#!/bin/bash -e
# Copyright 2018 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


if [ -z "$1" ]
  then
    PROJECT_ID=$(gcloud config config-helper --format "value(configuration.properties.core.project)")
else
  PROJECT_ID=$1
fi

# build base image
pushd ../base
./build.sh
popd

docker build -t ml-pipeline-dataflow-tfma .
docker tag ml-pipeline-dataflow-tfma gcr.io/${PROJECT_ID}/ml-pipeline-dataflow-tfma
gcloud docker -- push gcr.io/${PROJECT_ID}/ml-pipeline-dataflow-tfma

