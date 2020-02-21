#!/bin/bash

set -e

echo "The tests run against the cluster your kubectl communicates to.";
echo "It's currently '$(kubectl config current-context)'."

if [ -z "${NAMESPACE}" ]; then
    echo "NAMESPACE env var is not provided, please set it to your KFP namespace"
    exit
fi

echo "Starting integration tests..."
go test -v ./... -namespace ${NAMESPACE} -args -runIntegrationTests=true -skipWaitForCluster=true
