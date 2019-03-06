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
CLEANUP=true

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
             --results-file-name )shift
                                RESULT_FILE_NAME=$1
                                ;;
             --test-dir         )shift
                                TEST_DIR=$1
                                ;;
             --cleanup          )shift
                                CLEANUP=$1
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

GITHUB_REPO=kubeflow/pipelines
BASE_DIR=/go/src/github.com/${GITHUB_REPO}

cd "${BASE_DIR}/${TEST_DIR}"

# turn on go module
export GO111MODULE=on

echo "Run integration test..."
TEST_RESULT=`go test -v ./... -namespace ${NAMESPACE} -args -runIntegrationTests=true -cleanup=${CLEANUP} 2>&1`
TEST_EXIT_CODE=$?

# Log the test result
printf '%s\n' "$TEST_RESULT"
# Convert test result to junit.xml
printf '%s\n' "$TEST_RESULT" | go-junit-report > ${RESULT_FILE_NAME}

echo "Copy test result to GCS ${RESULTS_GCS_DIR}/${RESULT_FILE_NAME}"
gsutil cp ${RESULT_FILE_NAME} ${RESULTS_GCS_DIR}/${RESULT_FILE_NAME}

exit $TEST_EXIT_CODE
