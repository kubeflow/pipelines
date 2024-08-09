#!/bin/bash
#
# Copyright 2020 The Kubeflow Authors
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

set -ex

# It's called in ".cloudbuild.yaml" for each commits to master
# It's used for Hosted deployment testing but also can be reused for other usages.
#
# Later we need to unify the image names and path, after that, we may don't need this file.
# All will be handled in Cloud Build yaml file.
PROJECT_ID=$1
COMMIT_SHA=$2
SEM_VER=$3
MM_VER=$4

docker tag gcr.io/$PROJECT_ID/frontend:$COMMIT_SHA gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/frontend:$SEM_VER
docker tag gcr.io/$PROJECT_ID/frontend:$COMMIT_SHA gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/frontend:$MM_VER
docker push gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/frontend:$SEM_VER
docker push gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/frontend:$MM_VER

docker tag gcr.io/$PROJECT_ID/api-server:$COMMIT_SHA gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/apiserver:$SEM_VER
docker tag gcr.io/$PROJECT_ID/api-server:$COMMIT_SHA gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/apiserver:$MM_VER
docker push gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/apiserver:$SEM_VER
docker push gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/apiserver:$MM_VER

docker tag gcr.io/$PROJECT_ID/scheduledworkflow:$COMMIT_SHA gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/scheduledworkflow:$SEM_VER
docker tag gcr.io/$PROJECT_ID/scheduledworkflow:$COMMIT_SHA gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/scheduledworkflow:$MM_VER
docker push gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/scheduledworkflow:$SEM_VER
docker push gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/scheduledworkflow:$MM_VER

docker tag gcr.io/$PROJECT_ID/viewer-crd-controller:$COMMIT_SHA gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/viewercrd:$SEM_VER
docker tag gcr.io/$PROJECT_ID/viewer-crd-controller:$COMMIT_SHA gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/viewercrd:$MM_VER
docker push gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/viewercrd:$SEM_VER
docker push gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/viewercrd:$MM_VER

docker tag gcr.io/$PROJECT_ID/persistenceagent:$COMMIT_SHA gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/persistenceagent:$SEM_VER
docker tag gcr.io/$PROJECT_ID/persistenceagent:$COMMIT_SHA gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/persistenceagent:$MM_VER
docker push gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/persistenceagent:$SEM_VER
docker push gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/persistenceagent:$MM_VER

docker tag gcr.io/$PROJECT_ID/inverse-proxy-agent:$COMMIT_SHA gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/proxyagent:$SEM_VER
docker tag gcr.io/$PROJECT_ID/inverse-proxy-agent:$COMMIT_SHA gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/proxyagent:$MM_VER
docker push gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/proxyagent:$SEM_VER
docker push gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/proxyagent:$MM_VER

docker tag gcr.io/$PROJECT_ID/visualization-server:$COMMIT_SHA gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/visualizationserver:$SEM_VER
docker tag gcr.io/$PROJECT_ID/visualization-server:$COMMIT_SHA gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/visualizationserver:$MM_VER
docker push gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/visualizationserver:$SEM_VER
docker push gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/visualizationserver:$MM_VER

docker tag gcr.io/$PROJECT_ID/metadata-writer:$COMMIT_SHA gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/metadatawriter:$SEM_VER
docker tag gcr.io/$PROJECT_ID/metadata-writer:$COMMIT_SHA gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/metadatawriter:$MM_VER
docker push gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/metadatawriter:$SEM_VER
docker push gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/metadatawriter:$MM_VER

docker tag gcr.io/$PROJECT_ID/cache-server:$COMMIT_SHA gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/cacheserver:$SEM_VER
docker tag gcr.io/$PROJECT_ID/cache-server:$COMMIT_SHA gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/cacheserver:$MM_VER
docker push gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/cacheserver:$SEM_VER
docker push gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/cacheserver:$MM_VER

docker tag gcr.io/$PROJECT_ID/cache-deployer:$COMMIT_SHA gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/cachedeployer:$SEM_VER
docker tag gcr.io/$PROJECT_ID/cache-deployer:$COMMIT_SHA gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/cachedeployer:$MM_VER
docker push gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/cachedeployer:$SEM_VER
docker push gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/cachedeployer:$MM_VER

docker tag gcr.io/$PROJECT_ID/metadata-envoy:$COMMIT_SHA gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/metadataenvoy:$SEM_VER
docker tag gcr.io/$PROJECT_ID/metadata-envoy:$COMMIT_SHA gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/metadataenvoy:$MM_VER
docker push gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/metadataenvoy:$SEM_VER
docker push gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/metadataenvoy:$MM_VER

# Deployer required by MKP
docker tag gcr.io/$PROJECT_ID/deployer:$COMMIT_SHA gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/deployer:$SEM_VER
docker tag gcr.io/$PROJECT_ID/deployer:$COMMIT_SHA gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/deployer:$MM_VER
docker push gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/deployer:$SEM_VER
docker push gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/deployer:$MM_VER

# Primary binary required by MKP
docker tag gcr.io/$PROJECT_ID/deployer:$COMMIT_SHA gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA:$SEM_VER
docker tag gcr.io/$PROJECT_ID/deployer:$COMMIT_SHA gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA:$MM_VER
docker push gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA:$SEM_VER
docker push gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA:$MM_VER

# 3rd-party images
# ! Sync to the same MLMD version:
# * backend/metadata_writer/requirements.in and requirements.txt
# * frontend/src/mlmd/generated
# * .cloudbuild.yaml and .release.cloudbuild.yaml
# * manifests/kustomize/base/metadata/base/metadata-grpc-deployment.yaml
# * test/tag_for_hosted.sh
docker tag gcr.io/tfx-oss-public/ml_metadata_store_server:1.14.0 gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/metadataserver:$SEM_VER
docker tag gcr.io/tfx-oss-public/ml_metadata_store_server:1.14.0 gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/metadataserver:$MM_VER
docker push gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/metadataserver:$SEM_VER
docker push gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/metadataserver:$MM_VER

docker tag gcr.io/ml-pipeline/minio:RELEASE.2019-08-14T20-37-41Z-license-compliance gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/minio:$SEM_VER
docker tag gcr.io/ml-pipeline/minio:RELEASE.2019-08-14T20-37-41Z-license-compliance gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/minio:$MM_VER
docker push gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/minio:$SEM_VER
docker push gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/minio:$MM_VER

docker tag gcr.io/ml-pipeline/mysql:8.0.26 gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/mysql:$SEM_VER
docker tag gcr.io/ml-pipeline/mysql:8.0.26 gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/mysql:$MM_VER
docker push gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/mysql:$SEM_VER
docker push gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/mysql:$MM_VER

docker tag gcr.io/cloudsql-docker/gce-proxy:1.25.0 gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/cloudsqlproxy:$SEM_VER
docker tag gcr.io/cloudsql-docker/gce-proxy:1.25.0 gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/cloudsqlproxy:$MM_VER
docker push gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/cloudsqlproxy:$SEM_VER
docker push gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/cloudsqlproxy:$MM_VER

docker tag gcr.io/ml-pipeline/argoexec:v3.4.17-license-compliance gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/argoexecutor:$SEM_VER
docker tag gcr.io/ml-pipeline/argoexec:v3.4.17-license-compliance gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/argoexecutor:$MM_VER
docker push gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/argoexecutor:$SEM_VER
docker push gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/argoexecutor:$MM_VER

docker tag gcr.io/ml-pipeline/workflow-controller:v3.4.17-license-compliance gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/argoworkflowcontroller:$SEM_VER
docker tag gcr.io/ml-pipeline/workflow-controller:v3.4.17-license-compliance gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/argoworkflowcontroller:$MM_VER
docker push gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/argoworkflowcontroller:$SEM_VER
docker push gcr.io/$PROJECT_ID/hosted/$COMMIT_SHA/argoworkflowcontroller:$MM_VER
