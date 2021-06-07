# Using the command line to deploy the application

## Only For Development

This CLI installation guide is **ONLY** for AI Platform Pipelines development. It's not suitable for users of Kubeflow Pipelines.

For users, please refer to [Kubeflow Pipelines Standalone](https://www.kubeflow.org/docs/pipelines/installation/standalone-deployment/) on how to install via commandline using kustomize manifests. The installation is almost the same as AI Platform Pipelines.

Refer to the [Installation Options for Kubeflow Pipelines](https://www.kubeflow.org/docs/pipelines/installation/overview/) doc for all installation options and their feature comparison.

## Prerequisites

You can use [Google Cloud Shell](https://cloud.google.com/shell/) or a local
workstation to complete these steps.

[![Open in Cloud Shell](http://gstatic.com/cloudssh/images/open-btn.svg)](https://console.cloud.google.com/cloudshell/editor?cloudshell_git_repo=https://github.com/kubeflow/pipelines&cloudshell_open_in_editor=README.md&cloudshell_working_dir=manifests/gcp_marketplace)

### Set up command-line tools

You'll need the following tools in your development environment. If you are
using Cloud Shell, these tools are installed in your environment by default.

-   [gcloud](https://cloud.google.com/sdk/gcloud/)
-   [kubectl](https://kubernetes.io/docs/reference/kubectl/overview/)
-   [docker](https://docs.docker.com/install/)
-   [git](https://git-scm.com/book/en/v2/Getting-Started-Installing-Git)

Configure `gcloud` as a Docker credential helper:

```shell
gcloud auth configure-docker
```

You can install Kubeflow Pipelines in an existing GKE cluster or create a new GKE cluster.

* If you want to **create** a new Google GKE cluster, follow the instructions from the section [Create a GKE cluster](#create-gke-cluster) onwards.

* If you have an **existing** GKE cluster, ensure that the cluster nodes have a minimum 3 node cluster with each node having a minimum of 2 vCPU and running k8s version 1.9 and follow the instructions from section [Install the application resource definition](#install-application-resource-definition) onwards.

### <a name="create-gke-cluster"></a>Create a GKE cluster

Kubeflow Pipelines requires a minimum 3 node cluster with each node having a minimum of 2 vCPU and k8s version 1.9. Available machine types can be seen [here](https://cloud.google.com/compute/docs/machine-types).

Create a new cluster from the command line:

```shell
# The following parameters can be customized based on your needs.

export CLUSTER=kubeflow-pipelines-cluster
export ZONE=us-west1-a
export MACHINE_TYPE=n1-standard-2
export SCOPES="cloud-platform" # This scope is needed for running some pipeline samples. Read the warning below for its security implications.


gcloud config set project <your_project>
gcloud container clusters create "$CLUSTER" --zone "$ZONE" --machine-type "$MACHINE_TYPE" \
  --scopes $SCOPES
```

Configure `kubectl` to connect to the new cluster:

```shell
gcloud container clusters get-credentials "$CLUSTER" --zone "$ZONE"
```

> **Warning**: Using `SCOPES="cloud-platform"` grants all GCP permissions to the cluster. For a more secure cluster setup, refer to [Authenticating Pipelines to GCP](https://www.kubeflow.org/docs/gke/authentication/#authentication-from-kubeflow-pipelines).

### <a name="install-application-resource-definition"></a>Install the application resource definition

An application resource is a collection of individual Kubernetes components,
such as services, stateful sets, deployments, and so on, that you can manage as a group.

To set up your cluster to understand application resources, run the following command:

```shell
kubectl apply -f "https://raw.githubusercontent.com/GoogleCloudPlatform/marketplace-k8s-app-tools/master/crd/app-crd.yaml"
```

You need to run this command once.

The application resource is defined by the Kubernetes
[SIG-apps](https://github.com/kubernetes/community/tree/master/sig-apps)
community. The source code can be found on
[github.com/kubernetes-sigs/application](https://github.com/kubernetes-sigs/application).

### Prerequisites for using Role-Based Access Control
You must grant your user the ability to create roles in Kubernetes by running the following command.

```shell
kubectl create clusterrolebinding cluster-admin-binding \
  --clusterrole cluster-admin \
  --user $(gcloud config get-value account)
```

You need to run this command once.


## Install the Application

### Run installer script
Set your application instance name and the Kubernetes namespace to deploy:

```shell
# set the application instance name
export APP_INSTANCE_NAME=kubeflow-pipelines-app

# set the Kubernetes namespace the application was originally installed
export NAMESPACE=<namespace>
```

Creat the namespace
```shell
kubectl create namespace $NAMESPACE
```

Follow the [instruction](https://github.com/GoogleCloudPlatform/marketplace-k8s-app-tools/blob/master/docs/tool-prerequisites.md#tool-prerequisites) and install mpdev
TODO: The official mpdev won't work because it doesn't have permission to deploy CRD. The latest unofficial build will have right permission. Remove following instruction when change is in prod.
```
BIN_FILE="$HOME/bin/mpdev"
docker run gcr.io/cloud-marketplace-staging/marketplace-k8s-app-tools/k8s/dev:remove-ui-ownerrefs cat /scripts/dev > "$BIN_FILE"
chmod +x "$BIN_FILE"
export MARKETPLACE_TOOLS_TAG=remove-ui-ownerrefs
export MARKETPLACE_TOOLS_IMAGE=gcr.io/cloud-marketplace-staging/marketplace-k8s-app-tools/k8s/dev
```

Run the install script

```shell
mpdev scripts/install  --deployer=gcr.io/ml-pipeline/google/pipelines/deployer:0.1   --parameters='{"name": "'$APP_INSTANCE_NAME'", "namespace": "'$NAMESPACE'"}'

```

Watch the deployment come up with

```shell
kubectl get pods -n $NAMESPACE --watch
```

Get public endpoint
```shell
kubectl describe configmap inverse-proxy-config -n $NAMESPACE | grep googleusercontent.com

```

## Delete the Application

```shell
kubectl delete applications -n $NAMESPACE $APP_INSTANCE_NAME
```
