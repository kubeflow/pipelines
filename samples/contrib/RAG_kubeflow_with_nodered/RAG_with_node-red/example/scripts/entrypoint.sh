#!/bin/bash

trap stop SIGINT SIGTERM

function stop() {
	kill $CHILD_PID
	wait $CHILD_PID
}

/usr/local/bin/node $NODE_OPTIONS node_modules/node-red/red.js --userDir $USERDIR &

CHILD_PID="$!"

wait "${CHILD_PID}"
