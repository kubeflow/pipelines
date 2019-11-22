#!/bin/bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

export COVERALLS_REPO_TOKEN=$(node $DIR/get-coveralls-repo-token.js)
export COVERALLS_SERVICE_NAME="prow"
# Prow env reference: https://github.com/kubernetes/test-infra/blob/master/prow/jobs.md
export COVERALLS_SERVICE_JOB_ID=$PROW_JOB_ID
REPO_BASE="https://github.com/kubeflow/pipelines"
export CI_PULL_REQUEST="$REPO_BASE/pull/$PULL_NUMBER"
cat ./coverage/lcov.info | npx coveralls
