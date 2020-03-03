#!/bin/bash
#
# Copyright 2020 Google LLC
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

PROJECT_ID=$1
COMMIT_SHA=$2
SEM_VER=$3
MM_VER=$4

docker tag gcr.io/$PROJECT_ID/frontend:$COMMIT_SHA gcr.io/$PROJECT_ID/hosted/frontend:$SEM_VER
docker tag gcr.io/$PROJECT_ID/frontend:$COMMIT_SHA gcr.io/$PROJECT_ID/hosted/frontend:$MM_VER
docker push gcr.io/$PROJECT_ID/hosted/frontend:$SEM_VER
docker push gcr.io/$PROJECT_ID/hosted/frontend:$MM_VER

docker tag gcr.io/$PROJECT_ID/api-server:$COMMIT_SHA gcr.io/$PROJECT_ID/hosted/apiserver:$SEM_VER
docker tag gcr.io/$PROJECT_ID/api-server:$COMMIT_SHA gcr.io/$PROJECT_ID/hosted/apiserver:$MM_VER
docker push gcr.io/$PROJECT_ID/hosted/apiserver:$SEM_VER
docker push gcr.io/$PROJECT_ID/hosted/apiserver:$MM_VER

docker tag gcr.io/$PROJECT_ID/viewer-crd-controller:$COMMIT_SHA gcr.io/$PROJECT_ID/hosted/viewercrd:$SEM_VER
docker tag gcr.io/$PROJECT_ID/viewer-crd-controller:$COMMIT_SHA gcr.io/$PROJECT_ID/hosted/viewercrd:$MM_VER
docker push gcr.io/$PROJECT_ID/hosted/viewercrd:$SEM_VER
docker push gcr.io/$PROJECT_ID/hosted/viewercrd:$MM_VER

