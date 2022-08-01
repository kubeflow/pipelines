#!/bin/bash

set -e

if [ -z "${NAMESPACE}" ]; then
    echo "NAMESPACE env var is not provided, please set it to your KFP namespace"
    exit
fi

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

echo "Starting integration tests..."
# turn on go module
# export GO111MODULE=on
go env -w GO111MODULE=off
export PATH=$PATH:$(dirname $(go list -f '{{.Target}}' .))
command="go test -v ./... -namespace ${NAMESPACE} -args -runIntegrationTests=true -isDevMode=true"
echo $command "$@"
$command "$@"
