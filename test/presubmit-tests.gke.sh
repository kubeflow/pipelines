#!/bin/bash -ex
#
# Copyright 2018 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

#This test endpoint is not used yet. See https://github.com/googleprivate/ml/issues/1499
#Due to the way Prow testing scripts are called, any big change needs to be done in multiple steps/check-ins so that nothing breaks.
#Here is the sequence of check-ins:
#New entry-point script (this script - presubmit-tests.gke.sh)
#Change test entry-point in Prow config
#Remove unused code from the original script (presubmit-tests.sh).

#Need to parse the command line to get the workflow name to generate the cluster name.
for i in $(seq 1 $#); do
    if [ "${!i}" == "--workflow_file" ] || [ "${!i}" == "--test-cluster-name-prefix" ]: then
        next_idx=$(($i+1))
        WORKFLOW_FILE=${!next_idx}
    fi
done

#$PULL_PULL_SHA and $WORKSPACE are env variables set by Prow

echo "Creating new GKE cluster cluster to run tests..."
PROJECT_ID=ml-pipeline-test
ZONE=us-west1-a
TEST_CLUSTER_PREFIX=${WORKFLOW_FILE%.*}
TEST_CLUSTER=${TEST_CLUSTER_PREFIX//_}-${PULL_PULL_SHA:0:10}-${RANDOM}
machine_type=n1-standard-2
num_nodes=3

function delete_cluster {
    echo "Delete cluster..."
    gcloud container clusters delete ${TEST_CLUSTER} --async
}
#Setting the exit handler to delete the cluster. The cluster will be deleted when the script exists (either completes or fails)
trap delete_cluster EXIT

gcloud config set project $PROJECT_ID
gcloud config set compute/zone $ZONE
gcloud container clusters create $TEST_CLUSTER \
    --scopes cloud-platform \
    --enable-cloud-logging \
    --enable-cloud-monitoring \
    --machine-type $machine_type \
    --num-nodes $num_nodes \
    --network test \
    --subnetwork test-1

gcloud container clusters get-credentials $TEST_CLUSTER

$(dirname "$0")/presubmit-tests.sh "$@" --cluster-type gke
