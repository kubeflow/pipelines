#!/bin/bash

# Copyright 2018 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

usage()
{
    echo "usage: run_test.sh
    [--commit_sha     commit SHA to pull code from]
    [--namespace      k8s namespace where ml-pipelines is deployed. The tests run against the instance in this namespace]
    [-h help]"
}

while [ "$1" != "" ]; do
    case $1 in
             --commit_sha )     shift
                                COMMIT_SHA=$1
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

GITHUB_REPO=googleprivate/ml
BASE_DIR=/go/src/github.com/${GITHUB_REPO}
TEST_RESULT_BASE_DIR=gs://ml-pipeline-test
JUNIT_TEST_RESULT=junit_ApiIntegrationTestOutput.xml
TEST_DIR=backend/test

# Add github to SSH known host.
ssh-keygen -F github.com || ssh-keyscan github.com >>~/.ssh/known_hosts

echo "Clone ML pipeline code in COMMIT SHA ${COMMIT_SHA}..."
git clone git@github.com:${GITHUB_REPO}.git ${BASE_DIR}
cd ${BASE_DIR}/${TEST_DIR}
git checkout ${COMMIT_SHA}

go get -t -v ./...
# TODO(IronPan): use go dep https://github.com/googleprivate/ml/issues/561
rm -r /go/src/k8s.io/kubernetes/vendor/github.com/golang/glog

echo "Run integration test..."
TEST_RESULT=`go test -v ./... -namespace ${NAMESPACE} 2>&1`
TEST_EXIT_CODE=$?

# Log the test result
printf '%s\n' "$TEST_RESULT"
# Convert test result to junit.xml
printf '%s\n' "$TEST_RESULT" | go-junit-report > ${JUNIT_TEST_RESULT}

echo "Copy test result to GCS ${TEST_RESULT_BASE_DIR}/${COMMIT_SHA}"
gsutil cp ${JUNIT_TEST_RESULT} ${TEST_RESULT_BASE_DIR}/${COMMIT_SHA}/api_integration_test/${JUNIT_TEST_RESULT}

exit $TEST_EXIT_CODE