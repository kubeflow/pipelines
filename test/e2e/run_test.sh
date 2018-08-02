#!/bin/bash

set -xe

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

npm test
TEST_EXIT_CODE=$?

TEST_RESULT_BASE_DIR=gs://ml-pipeline-test
JUNIT_TEST_RESULT=junit_E2eTestOutput.xml

echo "Copy test result to GCS ${TEST_RESULT_BASE_DIR}/${COMMIT_SHA}/api_integration_test/${JUNIT_TEST_RESULT}"
tools/google-cloud-sdk/bin/gsutil cp ${JUNIT_TEST_RESULT} ${TEST_RESULT_BASE_DIR}/${COMMIT_SHA}/api_integration_test/${JUNIT_TEST_RESULT}

exit $TEST_EXIT_CODE
