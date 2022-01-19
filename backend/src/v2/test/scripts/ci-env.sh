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

COMMIT_SHA="$(git rev-parse HEAD)"
PROJECT="${PROJECT:-kfp-ci}"
GCS_ROOT="gs://${PROJECT}/${COMMIT_SHA}/v2-sample-test"
GCR_ROOT="gcr.io/${PROJECT}/${COMMIT_SHA}/v2-sample-test"
# This is kfp-ci endpoint.
HOST=${HOST:-"https://$(curl https://raw.githubusercontent.com/kubeflow/testing/master/test-infra/kfp/endpoint)"}

if [[ -z "${PULL_NUMBER}" ]]; then
  KFP_PACKAGE_PATH='git+https://github.com/kubeflow/pipelines\#egg=kfp&subdirectory=sdk/python'
else
  KFP_PACKAGE_PATH='git+https://github.com/kubeflow/pipelines@refs/pull/${PULL_NUMBER}/merge\#egg=kfp&subdirectory=sdk/python'
fi

cat <<EOF >kfp-ci.env
PROJECT=${PROJECT}
GCS_ROOT=${GCS_ROOT}
GCR_ROOT=${GCR_ROOT}
HOST=${HOST}
KFP_PACKAGE_PATH='${KFP_PACKAGE_PATH}'
EOF
