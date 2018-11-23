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

# K8s Namespace that all resources deployed to
NAMESPACE=kubeflow

usage()
{
    echo "usage: run_test.sh
    --results-gcs-dir GCS directory for the test results. Usually gs://<project-id>/<commit-sha>/api_integration_test
    [--namespace      k8s namespace where ml-pipelines is deployed. The tests run against the instance in this namespace]
    [-h help]"
}

while [ "$1" != "" ]; do
    case $1 in
             --results-gcs-dir )shift
                                RESULTS_GCS_DIR=$1
                                ;;
             --namespace )      shift
                                NAMESPACE=$1
                                ;;
             -h | --help )      usage
                                exit
                                ;;
             * )                usage
                                exit 1
    esac
    shift
done

if [ -z "$RESULTS_GCS_DIR" ]; then
    usage
    exit 1
fi

# Setup Google Cloud SDK, and use it to install kubectl so it gets the right context
wget -nv https://dl.google.com/dl/cloudsdk/release/google-cloud-sdk.zip
unzip -qq google-cloud-sdk.zip -d tools
rm google-cloud-sdk.zip
tools/google-cloud-sdk/install.sh --usage-reporting=false \
  --path-update=false --bash-completion=false \
  --disable-installation-options
tools/google-cloud-sdk/bin/gcloud -q components install kubectl

npm install

# Port forward the UI so tests can work against localhost
POD=`/src/tools/google-cloud-sdk/bin/kubectl get pods -n ${NAMESPACE} -l app=ml-pipeline-ui -o jsonpath='{.items[0].metadata.name}'`
/src/tools/google-cloud-sdk/bin/kubectl port-forward -n ${NAMESPACE} ${POD} 3000:3000 &

# Run Selenium server
/opt/bin/entry_point.sh &
./node_modules/.bin/wait-port 127.0.0.1:4444 -t 20000
./node_modules/.bin/wait-port 127.0.0.1:3000 -t 20000

export PIPELINE_OUTPUT=${RESULTS_GCS_DIR}/pipeline_output
npm test
TEST_EXIT_CODE=$?

JUNIT_TEST_RESULT=junit_FrontendIntegrationTestOutput.xml

echo "Copy test result to GCS ${RESULTS_GCS_DIR}/${JUNIT_TEST_RESULT}"
tools/google-cloud-sdk/bin/gsutil cp ${JUNIT_TEST_RESULT} ${RESULTS_GCS_DIR}/${JUNIT_TEST_RESULT}

exit $TEST_EXIT_CODE
