# Copyright 2021 The Kubeflow Authors
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

export COMMIT_SHA="$(git rev-parse HEAD)"
export PROJECT="${PROJECT:-kfp-ci}"
export GCS_ROOT="gs://${PROJECT}/${COMMIT_SHA}/v2-sample-test"
export GCR_ROOT="gcr.io/${PROJECT}/${COMMIT_SHA}/v2-sample-test"
export HOST='https://1c0b05d19abcd1c1-dot-us-central1.pipelines.googleusercontent.com/'
export KFP_PACKAGE_PATH="git+https://github.com/kubeflow/pipelines@${COMMIT_SHA}&subdirectory=sdk/python"

cat <<EOF >kfp-ci.env
PROJECT=${PROJECT}
GCS_ROOT=${GCS_ROOT}
GCR_ROOT=${GCR_ROOT}
HOST=${HOST}
KFP_PACKAGE_PATH='${KFP_PACKAGE_PATH}'
EOF

