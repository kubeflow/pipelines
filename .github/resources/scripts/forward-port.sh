#!/usr/bin/env bash

usage() {
    echo "Usage: $0 [-q] <KUBEFLOW_NS> <APP_NAME> <LOCAL_PORT> <REMOTE_PORT>"
    exit 1
}

QUIET=0
while getopts "q" opt; do
    case $opt in
        q) QUIET=1 ;;
        *) usage ;;
    esac
done
shift $((OPTIND -1))

if [ $# -ne 4 ]; then
    usage
fi

KUBEFLOW_NS=$1
APP_NAME=$2
LOCAL_PORT=$3
REMOTE_PORT=$4

POD_NAME=$(kubectl get pods -n "$KUBEFLOW_NS" -l "app=$APP_NAME" -o jsonpath='{.items[0].metadata.name}')
echo "POD_NAME=$POD_NAME"

if [ $QUIET -eq 1 ]; then
    kubectl port-forward -n "$KUBEFLOW_NS" "$POD_NAME" "$LOCAL_PORT:$REMOTE_PORT" > /dev/null 2>&1 &
else
    kubectl port-forward -n "$KUBEFLOW_NS" "$POD_NAME" "$LOCAL_PORT:$REMOTE_PORT" &
fi

# wait for the port-forward
sleep 5
