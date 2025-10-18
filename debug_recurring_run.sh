#!/bin/bash

echo "=== Checking Recurring Runs API ==="
curl -s http://localhost:8888/apis/v2beta1/recurringruns | jq '.'

echo -e "\n=== Checking ScheduledWorkflows in Kubernetes ==="
kubectl get scheduledworkflows -n kubeflow

echo -e "\n=== Checking Recent Workflows ==="
kubectl get workflows -n kubeflow --sort-by=.metadata.creationTimestamp | tail -5

echo -e "\n=== Checking ml-pipeline logs for recurring run creation ==="
kubectl logs -n kubeflow deployment/ml-pipeline --tail=100 | grep -i "recurring\|schedule" | tail -20

echo -e "\n=== Checking ml-pipeline-scheduledworkflow logs ==="
kubectl logs -n kubeflow deployment/ml-pipeline-scheduledworkflow --tail=30
