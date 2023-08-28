#!/bin/bash -e

set -x

KUBERNETES_NAMESPACE="${KUBERNETES_NAMESPACE:-kubeflow}"
SERVER_NAME="${SERVER_NAME:-model-server}"
SERVER_ENDPOINT_OUTPUT_FILE="${SERVER_ENDPOINT_OUTPUT_FILE:-/tmp/server_endpoint/data}"

while (($#)); do
   case $1 in
     "--model-export-path")
       shift
       export MODEL_EXPORT_PATH="$1"
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
     "--replicas")
       shift
       export REPLICAS="$1"
       shift
       ;;
     "--batch-size")
       shift
       export BATCH_SIZE="$1"
       shift
       ;;
     "--model-version-policy")
       shift
       export MODEL_VERSION_POLICY="$1"
       shift
       ;;
     "--log-level")
       shift
       export LOG_LEVEL="$1"
       shift
       ;;
     "--server-endpoint-output-file")
       shift
       SERVER_ENDPOINT_OUTPUT_FILE = "$1"
       shift
       ;;
     *)
       echo "Unknown argument: '$1'"
       exit 1
       ;;
   esac
done

if [ -z "${MODEL_EXPORT_PATH}" ]; then
  echo "You must specify a path to the saved model"
  exit 1
fi

echo "Deploying the model '${MODEL_EXPORT_PATH}'"

if [ -z "${CLUSTER_NAME}" ]; then
  CLUSTER_NAME=$(wget -q -O- --header="Metadata-Flavor: Google" http://metadata.google.internal/computeMetadata/v1/instance/attributes/cluster-name)
fi

# Ensure the server name is not more than 63 characters.
export SERVER_NAME="${SERVER_NAME:0:63}"
# Trim any trailing hyphens from the server name.
while [[ "${SERVER_NAME:(-1)}" == "-" ]]; do SERVER_NAME="${SERVER_NAME::-1}"; done

echo "Deploying ${SERVER_NAME} to the cluster ${CLUSTER_NAME}"

# Connect kubectl to the local cluster
kubectl config set-cluster "${CLUSTER_NAME}" --server=https://kubernetes.default --certificate-authority=/var/run/secrets/kubernetes.io/serviceaccount/ca.crt
kubectl config set-credentials pipeline --token "$(cat /var/run/secrets/kubernetes.io/serviceaccount/token)"
kubectl config set-context kubeflow --cluster "${CLUSTER_NAME}" --user pipeline
kubectl config use-context kubeflow

echo "Generating service and deployment yaml files"
python apply_template.py

kubectl apply -f ovms.yaml

sleep 10
echo "Waiting for the TF Serving deployment to have at least one available replica..."
timeout="1000"
start_time=`date +%s`
while [[ $(kubectl get deploy --namespace "${KUBERNETES_NAMESPACE}" --selector=app="ovms-${SERVER_NAME}" --output=jsonpath='{.items[0].status.availableReplicas}') < "1" ]]; do
  current_time=`date +%s`
  elapsed_time=$(expr $current_time + 1 - $start_time)
  if [[ $elapsed_time -gt $timeout ]];then
    echo "timeout"
    exit 1
  fi
  sleep 5
done

echo "Obtaining the pod name..."
start_time=`date +%s`
pod_name=""
while [[ $pod_name == "" ]];do
  pod_name=$(kubectl get pods --namespace "${KUBERNETES_NAMESPACE}" --selector=app="ovms-${SERVER_NAME}" --template '{{range .items}}{{.metadata.name}}{{"\n"}}{{end}}')
  current_time=`date +%s`
  elapsed_time=$(expr $current_time + 1 - $start_time)
  if [[ $elapsed_time -gt $timeout ]];then
    echo "timeout"
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
  kubectl get po ${pod_name} --namespace "${KUBERNETES_NAMESPACE}" -o jsonpath='{.status.containerStatuses[0].state.running}'
  exit_code=$?
  current_time=`date +%s`
  elapsed_time=$(expr $current_time + 1 - $start_time)
  if [[ $elapsed_time -gt $timeout ]];then
    echo "timeout"
    exit 1
  fi
  sleep 2
done

start_time=`date +%s`
while [ -z "$(kubectl get po ${pod_name} --namespace "${KUBERNETES_NAMESPACE}" -o jsonpath='{.status.containerStatuses[0].state.running}')" ]; do
  current_time=`date +%s`
  elapsed_time=$(expr $current_time + 1 - $start_time)
  if [[ $elapsed_time -gt $timeout ]];then
    echo "timeout"
    exit 1
  fi
  sleep 5
done

# Wait a little while and then grab the logs of the running server
sleep 10
echo "Logs from the TF Serving pod:"
kubectl logs ${pod_name} --namespace "${KUBERNETES_NAMESPACE}"

mkdir -p "$(dirname "$SERVER_ENDPOINT_OUTPUT_FILE")"
echo "ovms-${SERVER_NAME}:80" > "$SERVER_ENDPOINT_OUTPUT_FILE"
