#!/bin/bash
# Copyright 2018-2023 The Kubeflow Authors
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

set -ex

if [ -z "${NAMESPACE}" ]; then
    echo "NAMESPACE env var is not provided, please set it to your KFP namespace"
    exit
fi

# Define the IP address for port-forwarding the database for local testing.
# This value should be kept in sync with the DB_FORWARD_IP in CI workflows.
export DB_FORWARD_IP=${DB_FORWARD_IP:-"127.0.0.3"}

echo "The api integration tests run against the cluster your kubectl communicates to.";
echo "It's currently '$(kubectl config current-context)'."
echo "WARNING: this will clear up all existing KFP data in this cluster."
read -r -p "Are you sure? [y/N] " response
case "$response" in
    [yY][eE][sS]|[yY])
        ;;
    *)
        exit
        ;;
esac

function cleanup() {
  echo "killing kubectl port forward before exit"
  kill "$PORT_FORWARD_PID"
}
trap cleanup EXIT

echo "Starting integration tests..."

if [ "$1" == "postgres" ]; then
    echo "Starting PostgreSQL DB port forwarding..."
    kubectl -n "$NAMESPACE" port-forward svc/postgres-service 5432:5432 --address="${DB_FORWARD_IP}" & PORT_FORWARD_PID=$!
    # wait for kubectl port forward
    sleep 10
    command="DBCONFIG_POSTGRESQLCONFIG_HOST=${DB_FORWARD_IP} go test -v ./... -namespace ${NAMESPACE} -args -runIntegrationTests=true -isDevMode=true -runPostgreSQLTests=true -localTest=true"
else
    echo "Starting MySQL DB port forwarding..."
    kubectl -n "$NAMESPACE" port-forward svc/mysql 3306:3306 --address=localhost & PORT_FORWARD_PID=$!
    # wait for kubectl port forward
    sleep 10
    command="go test -v ./... -namespace ${NAMESPACE} -args -runIntegrationTests=true -isDevMode=true -localTest=true"
fi

$command "$@"
