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
set -o pipefail

# K8s Namespace that all resources deployed to
NAMESPACE=kubeflow

usage()
{
    echo "usage: run_test.sh
    [--results-gcs-dir GCS directory for the test results. Usually gs://<project-id>/<commit-sha>/backend_unit_test]
    [--namespace      k8s namespace where ml-pipelines is deployed. The tests run against the instance in this namespace]
    [--run_upgrade_tests_preparation run preparation step of upgrade tests instead]
    [--run_upgrade_tests_verification run verification step of upgrade tests instead]
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
             --run_upgrade_tests_preparation )
                                UPGRADE_TESTS_PREPARATION=true
                                ;;
             --run_upgrade_tests_verification )
                                UPGRADE_TESTS_VERIFICATION=true
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
JUNIT_TEST_RESULT=junit_ApiIntegrationTestOutput.xml
TEST_DIR=backend/test/integration

cd "${BASE_DIR}/${TEST_DIR}"

# turn on go module
export GO111MODULE=on

echo "Run integration test..."
LOG_FILE=$(mktemp)
# Note, "set -o pipefail" at top of file is required to catch exit code of the pipe.
TEST_EXIT_CODE=0 # reference for how to save exit code: https://stackoverflow.com/a/18622662
if [ -n "$UPGRADE_TESTS_PREPARATION" ]; then
  go test -v ./... -namespace ${NAMESPACE} -args -runUpgradeTests=true -testify.m=Prepare |& tee $LOG_FILE || TEST_EXIT_CODE=$?
elif [ -n "$UPGRADE_TESTS_VERIFICATION" ]; then
  go test -v ./... -namespace ${NAMESPACE} -args -runUpgradeTests=true -testify.m=Verify |& tee $LOG_FILE || TEST_EXIT_CODE=$?
else
  go test -v ./... -namespace ${NAMESPACE} -args -runIntegrationTests=true |& tee $LOG_FILE || TEST_EXIT_CODE=$?
fi

# Convert test result to junit.xml
< "$LOG_FILE" go-junit-report > "${JUNIT_TEST_RESULT}"

echo "Copy test result to GCS ${RESULTS_GCS_DIR}/${JUNIT_TEST_RESULT}"
gsutil cp ${JUNIT_TEST_RESULT} ${RESULTS_GCS_DIR}/${JUNIT_TEST_RESULT}

exit $TEST_EXIT_CODE
