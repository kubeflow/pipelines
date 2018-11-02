#!/bin/bash -e
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

set -e
set -o nounset
set -o pipefail

PROJECT_ID=${PROJECT_ID:-}
ZONE=${ZONE:-us-central1-a}
CLUSTER_NAME=${CLUSTER_NAME:-ml-pipelines-$(date +%s)}
NUM_NODES=${NUM_NODES:-4}
MACHINE_TYPE=${MACHINE_TYPE:-n1-standard-2}
PIPELINES_VERSION=${PIPELINES_VERSION:-latest}
CLUSTER_TYPE=create-gke
NAMESPACE=kubeflow
PORT=8080
LAUNCH_BROWSER=true
INSTALL_TOOLS=false

usage()
{
    echo "Usage: bootstrap.sh
    [--cluster-type   The type of cluster to use for the tests. One of: create-gke, gke, install_minikube, none. Defaults to create-gke ]
    [--project-id     The id of the Google cloud project for GKE.]
    [--zone           Compute zone (e.g. us-central1-a) for the GKE cluster. Defaults to $ZONE]
    [--cluster-name   The GKE cluster name to create or use. Defaults to $CLUSTER_NAME.]
    [--num-nodes      The number of nodes to be created in each of the new cluster's zones. Defaults to $NUM_NODES.]
    [--machine-type   The type of machine to use for nodes. Defaults to $MACHINE_TYPE.]
    [--pipelines-version   Specify the explicit version of the pipelines system. Defaults to $PIPELINES_VERSION.]
    [--bootstrapper-url   Specify the explicit pipelines system bootstrapper.yaml URL.]
    [--port           Local TCP port for the pipeline UI port forwarding. Defaults to $PORT.]
    [--no-browser     Do not launch the web browser with the Pipelines dashboard]
    [--install-tools   Installs gcloud, kubectl, docker, socat, minikube if needed missing. Default is $INSTALL_TOOLS.]
    [-h help]

    Examples:
    bootstrap.sh    #Create a GKE cluster, install ML Pipelines, start port forwarding and launch the dashboard in a web browser
    bootstrap.sh --cluster-type gke --cluster-name <cluster-name> #Install ML pipelines on existing GKE cluster
    "
}

while (($#)); do
    arg=$1; shift
    case "$arg" in
        --project-id )
                                PROJECT_ID=$1; shift
                                ;;
        --zone )
                                ZONE=$1; shift
                                ;;
        --cluster-type )
                                CLUSTER_TYPE=$1; shift
                                ;;
        --cluster-name )
                                CLUSTER_NAME=$1; shift
                                ;;
        --num-nodes )
                                NUM_NODES=$1; shift
                                ;;
        --machine-type )
                                MACHINE_TYPE=$1; shift
                                ;;
        --pipelines-version )
                                PIPELINES_VERSION=$1; shift
                                ;;
        --bootstrapper-url )
                                BOOTSTRAPPER_URL=$1; shift
                                ;;
        --port )
                                PORT=$1; shift
                                ;;
        --install-tools )
                                INSTALL_TOOLS=$1; shift
                                ;;
        --no-browser )
                                LAUNCH_BROWSER=false
                                ;;
        -h | --help )           usage
                                exit
                                ;;
        * )                     usage
                                exit 1
    esac
done

namespace_option="--namespace $NAMESPACE"
if [ -z "${BOOTSTRAPPER_URL:+1}" ]; then
    BOOTSTRAPPER_URL=https://storage.googleapis.com/ml-pipeline/release/$PIPELINES_VERSION/bootstrapper.yaml
fi
UNINSTALLER_URL=${BOOTSTRAPPER_URL/%bootstrapper.yaml/uninstaller.yaml}


kubectl_cmd=$(command -v kubectl)
if [ -z "$kubectl_cmd" ]; then
    if [ "$INSTALL_TOOLS" == true ]; then
        echo "Installing kubectl..."
        #gcloud components install kubectl #gcloud
        KUBECTL_VERSION=${KUBECTL_VERSION:-$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)}
        curl -Lo kubectl https://storage.googleapis.com/kubernetes-release/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl && chmod +x kubectl && sudo cp kubectl /usr/local/bin/ && rm kubectl
    else
        echo "Error: kubectl command not found"
        echo "Run the bootstrapper --install-tools true option or install kubectl by running gcloud components install kubectl"
        exit 1
    fi
fi

if [ "$CLUSTER_TYPE" == "create-gke" ] || [ "$CLUSTER_TYPE" == "gke" ]; then
    if ! command -v gcloud; then
        if [ "$INSTALL_TOOLS" == true ]; then
            echo "Installing Google Cloud client..."
            export CLOUD_SDK_REPO="cloud-sdk-$(lsb_release -c -s)"
            echo "deb http://packages.cloud.google.com/apt $CLOUD_SDK_REPO main" | sudo tee -a /etc/apt/sources.list.d/google-cloud-sdk.list
            curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key add -
            sudo apt-get update && sudo apt-get install google-cloud-sdk
            gcloud init
        else
            echo "Error: gcloud command not found"
            echo "Run the bootstrapper --install-tools true option or follow the instructions here: https://cloud.google.com/sdk/install"
            exit 1
        fi
    fi

    rbac_user=$(gcloud config get-value account)
    if [ -z "$rbac_user" ]; then
        echo "Logging in to Google Cloud..."
        gcloud auth login
        rbac_user=$(gcloud config get-value account)
    fi

    if [ -z "$PROJECT_ID" ]; then
        PROJECT_ID=$(gcloud config get-value project)
        if [ -z "$PROJECT_ID" ]; then
            echo "There is no default Google Cloud project"
            echo "You can configure a default project or pass an existing project name."
            echo "You can create a new project by following the instructions here: https://cloud.google.com/resource-manager/docs/creating-managing-projects"
            exit 1
        else
            echo "Using Google Cloud project $PROJECT_ID"
        fi
    else
        echo "Setting default Google cloud project to $PROJECT_ID..."
        gcloud config set project $PROJECT_ID
    fi

    echo Setting default GKE zone to $ZONE...
    gcloud config set compute/zone $ZONE

    if [ "$CLUSTER_TYPE" == "create-gke" ]; then
        echo "Creating new GKE cluster $CLUSTER_NAME. This can take several minutes..."
        echo "The cluster can be deleted by running the following command: \"gcloud container clusters delete $CLUSTER_NAME --zone $ZONE\""
        gcloud container clusters create $CLUSTER_NAME \
            --zone $ZONE \
            --scopes cloud-platform \
            --enable-cloud-logging \
            --enable-cloud-monitoring \
            --machine-type "$MACHINE_TYPE" \
            --num-nodes "$NUM_NODES"
    fi

    echo "Setting $CLUSTER_NAME as default k8s cluster..."
    gcloud config set container/cluster $CLUSTER_NAME

    if ! kubectl get clusterrolebinding ml-pipeline-admin-binding --output=name 2>/dev/null; then
        echo "Adding cluster role binding..."
        kubectl create clusterrolebinding ml-pipeline-admin-binding --clusterrole=cluster-admin --user=$rbac_user
    fi
elif [ "$CLUSTER_TYPE" == "install_minikube" ]; then
    if ! command -v docker; then
        echo "Installing Docker (needed for VM-less Minikube)"
        $(dirname "$0")/../../test/minikube/install_docker.sh
    fi
    if ! command -v socat; then
        echo "Installing socat (needed for port forwarding)"
        sudo apt-get install socat
    fi
    if ! command -v minikube; then
        echo "Installing Minikube and starting it in VM-less mode"
        $(dirname "$0")/../../test/minikube/install_and_start_minikube_without_vm.sh
    fi
fi

ui_pod_name=$(kubectl get pods --all-namespaces --selector app=ml-pipeline-ui --output name)
if [ -z "$ui_pod_name" ]; then
    echo "Launching bootstrapper from $BOOTSTRAPPER_URL"
    bootstrapper_job_full_name=$(kubectl create -f "$BOOTSTRAPPER_URL" --output=name | grep 'deploy-ml-pipeline-') #=job.batch/deploy-ml-pipeline-xxxxx
    #clusterrole.rbac.authorization.k8s.io/mlpipeline-deploy-admin
    #clusterrolebinding.rbac.authorization.k8s.io/mlpipeline-admin-role-binding
    #job.batch/deploy-ml-pipeline-xxxxx

    bootstrapper_job_name=${bootstrapper_job_full_name##*/} #=deploy-ml-pipeline-xxxxx

    echo "Waiting for the bootstrapper job to complete ($bootstrapper_job_name). This can take several minutes..."
    kubectl wait --for=condition=complete --timeout=5m job/$bootstrapper_job_name

    bootstrapper_pod_full_name=$(kubectl get pods --selector job-name=$bootstrapper_job_name --output name) #=pod/deploy-ml-pipeline-xxxxx-yyyyy

    pod_phase=$(kubectl get $bootstrapper_pod_full_name --output jsonpath={.status.phase}) #=Succeeded
    #kubectl get job/$bootstrapper_job_name --output=jsonpath={.status.succeeded} #=1
    if [ "$pod_phase" != "Succeeded" ]; then
        echo "Bootstrapper did not succeed: $pod_phase"
        echo "Bootstrapper logs:"
        kubectl logs $bootstrapper_pod_full_name
        exit 1
    fi
fi

echo "------------------------------------------------"
echo "The Pipelines system is installed on the cluster"
echo "It can be uninstalled by running the following commands:"
echo "kubectl create -f $UNINSTALLER_URL"
echo "kubectl delete clusterroles mlpipeline-deploy-admin"
echo "kubectl delete clusterrolebindings mlpipeline-admin-role-binding"

echo "Starting port-forwarding..."
dashboard_url="http://localhost:$PORT/pipeline"
echo "To see the Pipelines UI, open $dashboard_url in web browser"
port_forward_command='kubectl port-forward '"$namespace_option"' $(kubectl get pods '"$namespace_option"' --selector=service=ambassador --output name | head -n 1)'" $PORT:80"
echo "To connect to the UI in future run \"$port_forward_command\""

if [ "$LAUNCH_BROWSER" == true ]; then
    #Waiting a bit and then opening the browser after we start port-forwarding (which is a blocking call).
    #Launching the browser before starting port forwarding doesn not work (error page)
    #Starting the port forwarding in background and then re-joining to the process does not work (re-attach via fg does not work).
    sh -c "
        sleep 1
        echo Opening the Pipelines dashboard: \"$dashboard_url\"
        sensible-browser \"$dashboard_url\"
    " &
fi

sh -c "$port_forward_command"
