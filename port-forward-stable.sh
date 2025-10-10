#!/bin/bash
# Stable port-forward script with auto-reconnect

NAMESPACE="kubeflow"
SERVICE="ml-pipeline-debug"
PORT="8888"

echo "Starting stable port-forward to $SERVICE in namespace $NAMESPACE on port $PORT"
echo "Press Ctrl+C to stop"

while true; do
    echo "[$(date)] Forwarding port $PORT..."
    kubectl port-forward -n $NAMESPACE svc/$SERVICE $PORT:$PORT

    # If port-forward exits, wait a bit and retry
    EXIT_CODE=$?
    if [ $EXIT_CODE -ne 0 ]; then
        echo "[$(date)] Port-forward exited with code $EXIT_CODE. Retrying in 2 seconds..."
        sleep 2
    else
        echo "[$(date)] Port-forward stopped normally."
        break
    fi
done
