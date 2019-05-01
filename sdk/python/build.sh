#!/bin/bash -ex
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


# The scripts creates the Kubeflow Pipelines python SDK package.
#
# Usage:
#   ./build.sh [output_file]


target_archive_file=${1:-kfp.tar.gz}

pushd "$(dirname "$0")"
dist_dir=$(mktemp -d)
python setup.py sdist --format=gztar --dist-dir "$dist_dir"
cp "$dist_dir"/*.tar.gz "$target_archive_file"
popd

echo "{\"packageName\": \"argo\"}" > /tmp/config.json
java -jar /tmp/swagger-codegen-cli.jar generate -l python \
  -i https://raw.githubusercontent.com/argoproj/argo/master/api/openapi-spec/swagger.json \
  -o $DIR \
  -c /tmp/config.json \
  -DmodelTests=false \
  -DmodelDocs=false \
  -Dmodels=io.argoproj.workflow.v1alpha1.S3Artifact,io.k8s.api.core.v1.SecretKeySelector \
  --import-mappings "IoK8sApiCoreV1SecretKeySelector=from kubernetes.client.models.v1_secret_key_selector import V1SecretKeySelector as IoK8sApiCoreV1SecretKeySelector"
rm /tmp/config.json