#!/bin/bash

echo "=== Checking Workflows ==="
kubectl get workflows -n kubeflow | grep helloworld | tail -5

echo -e "\n=== Checking Runs via API ==="
curl -s "http://localhost:8888/apis/v2beta1/runs?page_size=100" | jq '{total: .total_size, runs: (.runs // [] | length)}'

echo -e "\n=== Checking Recurring Runs ==="
curl -s "http://localhost:8888/apis/v2beta1/recurringruns" | jq '.recurringRuns[] | {id: .recurring_run_id, name: .display_name, status: .status}'
