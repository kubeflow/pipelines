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
    echo "usage: deploy.sh
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

BASE_DIR=/go/src/github.com/googleprivate/ml

ssh-keygen -F github.com || ssh-keyscan github.com >>~/.ssh/known_hosts

echo "Clone ML pipeline code in COMMIT SHA ${COMMIT_SHA}..."
git clone git@github.com:googleprivate/ml.git ${BASE_DIR}
cd ${BASE_DIR}/backend/test
git checkout ${COMMIT_SHA}

go get -t -v ./...
# TODO(IronPan): use go dep https://github.com/googleprivate/ml/issues/561
rm -r /go/src/k8s.io/kubernetes/vendor/github.com/golang/glog

echo "Run test..."
go test -v ./... -namespace ${NAMESPACE} 2>&1 | go-junit-report > api_integration_test_output.xml

echo "Copy test result to GCS gs://ml-pipeline-test/${COMMIT_SHA}"
gsutil cp api_integration_test_output.xml gs://ml-pipeline-test/${COMMIT_SHA}/api_integration_test_output.xml
