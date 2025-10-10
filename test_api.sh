#!/bin/bash

# Get an experiment ID
EXPERIMENT_ID=$(kubectl exec -n kubeflow mysql-6c56959ffd-qfkg7 -- mysql -u root mlpipeline -N -e "SELECT DISTINCT ExperimentUUID FROM run_details LIMIT 1;" 2>/dev/null)
echo "Experiment ID: $EXPERIMENT_ID"

# Try filtering by experiment
echo -e "\n=== Filtering by experiment_id ==="
curl -s "http://localhost:8888/apis/v2beta1/runs?experiment_id=$EXPERIMENT_ID&page_size=2" | jq '.'
