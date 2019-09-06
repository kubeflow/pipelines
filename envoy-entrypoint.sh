#!/bin/sh
set -e

# DEBUG
cat /etc/envoy.yaml

echo "Starting Envoy..."
/usr/local/bin/envoy -c /etc/envoy.yaml
