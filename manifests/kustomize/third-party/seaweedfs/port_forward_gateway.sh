#!/bin/bash
set -euxo pipefail

GATEWAY_SERVICE=$(kubectl get svc -n istio-system -l app=istio-ingressgateway -o jsonpath='{.items[0].metadata.name}')
nohup kubectl port-forward -n istio-system svc/$GATEWAY_SERVICE 8080:80 &
timeout 60s bash -c 'until curl -s localhost:8080 > /dev/null || curl -s -I localhost:8080 | grep -q "HTTP/"; do sleep 5; done' 