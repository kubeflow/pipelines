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
    --results-gcs-dir GCS directory for the test results. Usually gs://<project-id>/<commit-sha>/e2e_test
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

if [[ ! -z "${GOOGLE_APPLICATION_CREDENTIALS}" ]]; then
  gcloud auth activate-service-account --key-file="${GOOGLE_APPLICATION_CREDENTIALS}"
fi

npm install

# Port forward the UI so tests can work against localhost
POD=`kubectl get pods -n ${NAMESPACE} -l app=ml-pipeline-ui -o jsonpath='{.items[0].metadata.name}'`
kubectl port-forward -n ${NAMESPACE} ${POD} 3000:3000 &

# Run Selenium server
/opt/bin/entry_point.sh &
./node_modules/.bin/wait-port 127.0.0.1:4444 -t 20000
./node_modules/.bin/wait-port 127.0.0.1:3000 -t 20000

export PIPELINE_OUTPUT=${RESULTS_GCS_DIR}/pipeline_output
# Don't exit early if 'npm test' fails
set +e
npm test
TEST_EXIT_CODE=$?
set -e

JUNIT_TEST_RESULT=junit_FrontendIntegrationTestOutput.xml

echo "Copy test result to GCS ${RESULTS_GCS_DIR}/${JUNIT_TEST_RESULT}"
gsutil cp ${JUNIT_TEST_RESULT} ${RESULTS_GCS_DIR}/${JUNIT_TEST_RESULT}

exit $TEST_EXIT_CODE
