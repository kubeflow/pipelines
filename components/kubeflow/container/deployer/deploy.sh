#!/bin/bash -e

# Copyright 2018 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

KUBERNETES_NAMESPACE="${KUBERNETES_NAMESPACE:-default}"
SERVER_NAME="${SERVER_NAME:-model-server}"

while (($#)); do
   case $1 in
     "--model-path")
       shift
       MODEL_PATH="$1"
       shift
       ;;
     "--project")
       shift
       PROJECT="$1"
       shift
       ;;
     "--zone")
       shift
       ZONE="$1"
       shift
       ;;
     "--cluster-name")
       shift
       CLUSTER_NAME="$1"
       shift
       ;;
     "--namespace")
       shift
       KUBERNETES_NAMESPACE="$1"
       shift
       ;;
     "--server-name")
       shift
       SERVER_NAME="$1"
       shift
       ;;
     *)
       echo "Unknown argument: '$1'"
       exit 1
       ;;
   esac
done

if [ -z "${MODEL_PATH}" ]; then
  echo "You must specify a path to the saved model"
  exit 1
fi

echo "Deploying the model '${MODEL_PATH}'"

if [ -z "${PROJECT}" ]; then
  PROJECT=$(wget -q -O- --header="Metadata-Flavor: Google" http://metadata.google.internal/computeMetadata/v1/project/project-id)
fi
 
if [ -z "${CLUSTER_NAME}" ]; then
  CLUSTER_NAME=$(wget -q -O- --header="Metadata-Flavor: Google" http://metadata.google.internal/computeMetadata/v1/instance/attributes/cluster-name)
fi
 
if [ -z "${ZONE}" ]; then
  ZONE=$(wget -q -O- --header="Metadata-Flavor: Google" http://metadata.google.internal/computeMetadata/v1/instance/zone | cut -d '/' -f 4)
fi

# Ensure the server name is not more than 63 characters.
SERVER_NAME="${SERVER_NAME:0:63}"
# Trim any trailing hyphens from the server name.
while [[ "${SERVER_NAME:(-1)}" == "-" ]]; do SERVER_NAME="${SERVER_NAME::-1}"; done

echo "Deploying ${SERVER_NAME} to the cluster ${CLUSTER_NAME} in the project ${PROJECT} and the zone ${ZONE}..."

# Connect kubectl to the cluster
gcloud --project "${PROJECT}" container clusters get-credentials "${CLUSTER_NAME}" --zone "${ZONE}"

# Configure and deploy the TF serving app
cd /src/github.com/kubeflow/kubeflow
git checkout ${KUBEFLOW_VERSION}

cd /opt
echo "Initializing KSonnet app..."
ks init tf-serving-app
cd tf-serving-app/

if [ -n "${KUBERNETES_NAMESPACE}" ]; then
  echo "Setting Kubernetes namespace: ${KUBERNETES_NAMESPACE} ..."
  ks env set default --namespace "${KUBERNETES_NAMESPACE}"
fi

echo "Installing Kubeflow packages..."
ks registry add kubeflow /src/github.com/kubeflow/kubeflow/kubeflow
ks pkg install kubeflow/core@${KUBEFLOW_VERSION}
ks pkg install kubeflow/tf-serving@${KUBEFLOW_VERSION}

echo "Generating the TF Serving config..."
ks generate tf-serving server --name="${SERVER_NAME}"
ks param set server modelPath "${MODEL_PATH}/export/export"

echo "Deploying the TF Serving service..."
ks apply default -c server

# Wait for the deployment to have at least one available replica
echo "Waiting for the TF Serving deployment to show up..."
timeout="1000"
start_time=`date +%s`
while [[ $(kubectl get deploy --selector=app="${SERVER_NAME}" 2>&1|wc -l) != "2" ]];do
  current_time=`date +%s`
  elapsed_time=$(expr $current_time - $start_time)
  if [[ $elapsed_time > $timeout ]];then
    exit 1
  fi
  sleep 2
done

echo "Waiting for the valid workflow json..."
start_time=`date +%s`
exit_code="1"
while [[ $exit_code != "0" ]];do
  kubectl get deploy --selector=app="${SERVER_NAME}" --output=jsonpath='{.items[0].status.availableReplicas}'
  exit_code = $?
  current_time=`date +%s`
  elapsed_time=$(expr $current_time - $start_time)
  if [[ $elapsed_time > $timeout ]];then
    exit 1
  fi
  sleep 2
done

echo "Waiting for the TF Serving deployment to have at least one available replica..."
start_time=`date +%s`
while [[ $(kubectl get deploy --selector=app="${SERVER_NAME}" --output=jsonpath='{.items[0].status.availableReplicas}') < "1" ]]; do
  current_time=`date +%s`
  elapsed_time=$(expr $current_time - $start_time)
  if [[ $elapsed_time > $timeout ]];then
    exit 1
  fi
  sleep 5
done

echo "Obtaining the pod name..."
start_time=`date +%s`
pod_name=""
while [[ $pod_name == "" ]];do
  pod_name=$(kubectl get pods --selector=app=${SERVER_NAME} --template '{{range .items}}{{.metadata.name}}{{"\n"}}{{end}}')
  current_time=`date +%s`
  elapsed_time=$(expr $current_time - start_time)
  if [[ $elapsed_time > $timeout ]];then
    exit 1
  fi
  sleep 2
done
echo "Pod name is: " $pod_name

# Wait for the pod container to start running
echo "Waiting for the TF Serving pod to start running..."
start_time=`date +%s`
exit_code="1"
while [[ $exit_code != "0" ]];do
  kubectl get po ${pod_name} -o jsonpath='{.status.containerStatuses[0].state.running}'
  exit_code = $?
  current_time=`date +%s`
  elapsed_time=$(expr $current_time - $start_time)
  if [[ $elapsed_time > $timeout ]];then
    exit 1
  fi
  sleep 2
done

start_time=`date +%s`
while [ -z "$(kubectl get po ${pod_name} -o jsonpath='{.status.containerStatuses[0].state.running}')" ]; do
  current_time=`date +%s`
  elapsed_time=$(expr $current_time - $start_time)
  if [[ $elapsed_time > $timeout ]];then
    exit 1
  fi
  sleep 5
done

# Wait a little while and then grab the logs of the running server
sleep 10
echo "Logs from the TF Serving pod:"
kubectl logs ${pod_name}
