#!/bin/bash
#
# Copyright 2018 The Kubeflow Authors
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

usage()
{
    echo "usage: run_test.sh
    [-h help]"
}

while [[ "$1" != "" ]]; do
    case $1 in
             -h | --help )      usage
                                exit
                                ;;
             * )                usage
                                exit 1
    esac
    shift
done

npm install

KFP_BASE_URL="${KFP_BASE_URL:-http://localhost:3000}"
export KFP_BASE_URL
KFP_BASE_HOST=$(node -e "const u = new URL(process.env.KFP_BASE_URL); console.log(u.hostname);")
KFP_BASE_PORT=$(node -e "const u = new URL(process.env.KFP_BASE_URL); console.log(u.port || (u.protocol === 'https:' ? 443 : 80));")

# Run Selenium server
SELENIUM_PORT="${SELENIUM_PORT:-4444}"
/opt/bin/entry_point.sh &
./node_modules/.bin/wait-port 127.0.0.1:${SELENIUM_PORT} -t 20000
./node_modules/.bin/wait-port ${KFP_BASE_HOST}:${KFP_BASE_PORT} -t 20000

# Don't exit early if 'npm test' fails
set +e
npm test
TEST_EXIT_CODE=$?
set -e

exit $TEST_EXIT_CODE
