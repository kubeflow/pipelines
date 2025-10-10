#!/bin/bash
# Super stable port-forward using kubectl in a loop

NAMESPACE="kubeflow"
SERVICE="ml-pipeline-debug"
LOCAL_PORT="8888"
REMOTE_PORT="8888"

echo "🚀 Starting stable port-forward..."
echo "   Service: $NAMESPACE/$SERVICE"
echo "   Port: localhost:$LOCAL_PORT -> $REMOTE_PORT"
echo ""
echo "Press Ctrl+C to stop"
echo ""

# Cleanup function
cleanup() {
    echo ""
    echo "🛑 Stopping port-forward..."
    exit 0
}

trap cleanup SIGINT SIGTERM

# Counter for reconnections
RECONNECT_COUNT=0

while true; do
    if [ $RECONNECT_COUNT -gt 0 ]; then
        echo "[$(date '+%H:%M:%S')] 🔄 Reconnecting (attempt #$RECONNECT_COUNT)..."
    else
        echo "[$(date '+%H:%M:%S')] ✅ Connected"
    fi

    kubectl port-forward -n $NAMESPACE svc/$SERVICE $LOCAL_PORT:$REMOTE_PORT 2>&1 | \
        grep -v "Forwarding from" | \
        grep -v "Handling connection"

    EXIT_CODE=$?

    # If exit code is 0, user pressed Ctrl+C
    if [ $EXIT_CODE -eq 0 ]; then
        break
    fi

    # Otherwise, connection was lost
    RECONNECT_COUNT=$((RECONNECT_COUNT + 1))
    echo "[$(date '+%H:%M:%S')] ⚠️  Connection lost (exit code: $EXIT_CODE)"
    echo "[$(date '+%H:%M:%S')] ⏳ Waiting 1 second before reconnecting..."
    sleep 1
done

echo "👋 Goodbye!"
