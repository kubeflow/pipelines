#!/bin/bash
#
# Copyright 2019 Google LLC
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

set -e

# Usage: run this from root of KFP source repo and specify PROJECT env
# $PROJECT: gcp project

RELEASE_PROJECT=ml-pipeline
TAG=v2.7.5-license-compliance

gcloud builds submit --config third_party/argo/cloudbuild.yaml . \
  --substitutions=TAG_NAME="$TAG" --project "$PROJECT"

gcloud container images add-tag --quiet \
  gcr.io/$PROJECT/argoexec:$TAG \
  gcr.io/$RELEASE_PROJECT/argoexec:$TAG
gcloud container images add-tag --quiet \
  gcr.io/$PROJECT/workflow-controller:$TAG \
  gcr.io/$RELEASE_PROJECT/workflow-controller:$TAG
