#!/bin/bash

set -e

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

# Prow env reference: https://github.com/kubernetes/test-infra/blob/master/prow/jobs.md
export COVERALLS_REPO_TOKEN=$(node $DIR/get-coveralls-repo-token.js)
REPO_BASE="https://github.com/kubeflow/pipelines"
export CI_PULL_REQUEST="$REPO_BASE/pull/$PULL_NUMBER"

export COVERALLS_SERVICE_JOB_ID="${PROW_JOB_ID}-frontend"
export COVERALLS_SERVICE_NAME="prow-frontend"
cat ./coverage/lcov.info | npx coveralls

export COVERALLS_SERVICE_JOB_ID="${PROW_JOB_ID}-frontend-server"
export COVERALLS_SERVICE_NAME="prow-frontend-server"
cat ./server/coverage/lcov.info | npx coveralls
