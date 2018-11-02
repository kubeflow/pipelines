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

#set -o nounset

#This script runs the presubmit tests on a Minikube cluster.
#The script creates a new GCE VM, sets up Minikube, copies the SSH keys and repo and and then runs the tests.
#This script is usually run from the PROW cluster, but it can be run locally:
#ssh_key_file=~/.ssh/id_rsa WORKSPACE=/var/tmp PULL_PULL_SHA=master test/presubmit-tests-gce-minikube.sh --workflow_file integration_test_gke.yaml --test_result_folder api_integration_test

WORKSPACE=$WORKSPACE
PULL_PULL_SHA=$PULL_PULL_SHA
ARTIFACT_DIR=$WORKSPACE/_artifacts
PROJECT_ID=${PROJECT_ID:-ml-pipeline-test}
ZONE=${ZONE:-us-west1-a}

repo_test_dir=$(dirname $0)

instance_name=${instance_name:-test-minikube-${PULL_PULL_SHA:0:6}-$(date +%s)-$(echo "$@" | md5sum | cut -c 1-6)}

firewall_rule_name=allow-prow-ssh-$instance_name

#Function to delete VM
function delete_vm {
    if [ "$keep_created_vm" != true ]; then
        echo "Deleting VM $instance_name"
        gcloud compute instances delete $instance_name --zone=$ZONE --quiet
    fi
    echo "Deleting the SSH firewall rule $firewall_rule_name"
    gcloud compute firewall-rules delete $firewall_rule_name
}

#Setting the exit handler to delete VM. The VM will be deleted when the script exists (either completes or fails)
#TODO: Find a more resilent way to clean up VMs. Right now the VM is not deleted if the machine running this script fails. (See https://github.com/googleprivate/ml/issues/1064)
trap delete_vm EXIT

#Creating the VM
gcloud config set project $PROJECT_ID
gcloud config set compute/zone $ZONE

machine_type=n1-standard-16
boot_disk_size=200GB

gcloud compute instances create $instance_name --zone=$ZONE --machine-type=$machine_type --boot-disk-size=$boot_disk_size --scopes=storage-rw --tags=presubmit-test-vm

#Adding firewall entry that allows the current instance to access the newly created VM using SSH.
#This is needed for cases when the current instance is in different project (e.g. in the Prow cluster project)
self_external_ip=$(curl -sSL "http://metadata/computeMetadata/v1/instance/network-interfaces/0/access-configs/0/external-ip" -H "Metadata-Flavor: Google")
gcloud compute firewall-rules create $firewall_rule_name --allow tcp:22 --source-ranges=${self_external_ip}/32 --target-tags=presubmit-test-vm

#Workaround the problems with prow cluster and GCE SSH access.
#Prow tests run as root. GCE instances do not allow SSH access for root.
if [ "$(whoami)" == root ]; then
  export USER=not-root
fi

#Copy the ssh keys
ssh_key_file=${ssh_key_file:-/etc/ssh-knative/ssh-knative}
gcloud compute ssh --zone=$ZONE $instance_name -- 'sudo mkdir -m 777 -p /etc/ssh-knative' #Making the directory writable by a non-root user so that we can copy the file
gcloud compute scp --zone=$ZONE "$ssh_key_file" $instance_name:/etc/ssh-knative/ssh-knative

#Copy repo
git_root=$(git rev-parse --show-toplevel)
git_root_parent=$(dirname "$git_root")
gcloud compute scp --zone=$ZONE --verbosity=error --recurse "$git_root" $instance_name:'~' >/dev/null || true #Do not fail on error here because of broken symlinks until this is fixed: https://github.com/googleprivate/ml/issues/1084

#Installing software on VM
gcloud compute ssh --zone=$ZONE $instance_name -- "~/ml/test/minikube/install_docker_minikube_argo.sh"

#Running the presubmit tests
gcloud compute ssh --zone=$ZONE $instance_name -- PULL_PULL_SHA="$PULL_PULL_SHA" WORKSPACE="~/${WORKSPACE}" "~/ml/test/presubmit-tests.sh" --cluster-type none "$@"

#Copy back the artifacts
mkdir -p "${ARTIFACT_DIR}"
gcloud compute scp --zone=$ZONE --verbosity=error --recurse $instance_name:"~/${ARTIFACT_DIR}/*" "${ARTIFACT_DIR}/"
