#!/bin/bash
#
# Define functions to customize the Kubeflow app for GCP.
#
set -xe

gcpCreateSecretsDir() {
  # Create a directory to contain GCP secrets.
  mkdir -p ${KUBEFLOW_SECRETS_DIR}

  # We want to prevent secrets from being checked into source control.
  # We have two different checks.
  # 1. We put the secrets in a directory with a .gitignore file
  # 2. We will delete the secrets immediately.
  if [ ! -f ${KUBEFLOW_SECRETS_DIR}/.gitignore ]; then
  cat > ${KUBEFLOW_SECRETS_DIR}/.gitignore << EOF
**
EOF
  fi
}

gcpInitProject() {
  # Enable GCloud APIs
  gcloud services enable deploymentmanager.googleapis.com \
                         servicemanagement.googleapis.com \
                         container.googleapis.com \
                         cloudresourcemanager.googleapis.com \
                         endpoints.googleapis.com \
                         file.googleapis.com \
                         iam.googleapis.com --project=${PROJECT}

  # Set IAM Admin Policy
  gcloud projects add-iam-policy-binding ${PROJECT} \
     --member serviceAccount:${PROJECT_NUMBER}@cloudservices.gserviceaccount.com \
     --role roles/resourcemanager.projectIamAdmin
}

generateDMConfigs() {
  # Create the DM configs if they don't exist.
  if [ ! -d "${KUBEFLOW_DM_DIR}" ]; then
    echo creating Deployment Manager configs in directory "${KUBEFLOW_DM_DIR}"
    mkdir -p "${KUBEFLOW_DM_DIR}"
    cp -r "${KUBEFLOW_REPO}"/scripts/gke/deployment_manager_configs/cluster* "${KUBEFLOW_DM_DIR}"

    if [[ "$EMAIL" =~ .*iam\.gserviceaccount\.com ]]; then
      IAP_IAM_ENTRY="serviceAccount:${EMAIL}"
    else
      IAP_IAM_ENTRY="user:${EMAIL}"
    fi

    cp "${KUBEFLOW_REPO}"/scripts/gke/deployment_manager_configs/iam_bindings_template.yaml "${KUBEFLOW_DM_DIR}"/iam_bindings.yaml

    # Set the various service accounts in iam_bindings.yaml
    sed -i.bak "s/set-kubeflow-admin-service-account/serviceAccount:${DEPLOYMENT_NAME}-admin@${PROJECT}.iam.gserviceaccount.com/" "${KUBEFLOW_DM_DIR}"/iam_bindings.yaml
    sed -i.bak "s/set-kubeflow-user-service-account/serviceAccount:${DEPLOYMENT_NAME}-user@${PROJECT}.iam.gserviceaccount.com/" "${KUBEFLOW_DM_DIR}"/iam_bindings.yaml
    sed -i.bak "s/set-kubeflow-vm-service-account/serviceAccount:${DEPLOYMENT_NAME}-vm@${PROJECT}.iam.gserviceaccount.com/" "${KUBEFLOW_DM_DIR}"/iam_bindings.yaml
    sed -i.bak "s/set-kubeflow-iap-account/${IAP_IAM_ENTRY}/" "${KUBEFLOW_DM_DIR}"/iam_bindings.yaml
    rm "${KUBEFLOW_DM_DIR}/iam_bindings.yaml.bak"

    # Set values in DM config file
    sed -i.bak "s/zone: SET_THE_ZONE/zone: ${ZONE}/" "${KUBEFLOW_DM_DIR}/${CONFIG_FILE}"
    sed -i.bak "s/cluster-version: SET_CLUSTER_VERSION/cluster-version: ${CLUSTER_VERSION}/" "${KUBEFLOW_DM_DIR}/${CONFIG_FILE}"
    sed -i.bak "s/users:/users: [\"${IAP_IAM_ENTRY}\"]/" "${KUBEFLOW_DM_DIR}/${CONFIG_FILE}"
    sed -i.bak "s/ipName: kubeflow-ip/ipName: ${KUBEFLOW_IP_NAME}/" "${KUBEFLOW_DM_DIR}/${CONFIG_FILE}"
    rm "${KUBEFLOW_DM_DIR}/${CONFIG_FILE}.bak"
  else
    echo Deployment Manager configs already exist in directory "${KUBEFLOW_DM_DIR}"
  fi
}

createGcpSecret() {
  # Get a private key for a GCP service account and store it as a K8s secret.
  local EMAIL=$1
  local SECRET=$2

  set +e
  O=`kubectl get secret --namespace=${K8S_NAMESPACE} ${SECRET} 2>&1`
  local RESULT=$?
  set -e

  if [ "${RESULT}" -eq 0 ]; then
    echo secret ${SECRET} already exists
    return
  fi

  local FILE=${KUBEFLOW_SECRETS_DIR}/${EMAIL}.json
  gcloud --project=${PROJECT} iam service-accounts keys create ${FILE} --iam-account ${EMAIL}

  kubectl create secret generic --namespace=${K8S_NAMESPACE} ${SECRET} --from-file=${SECRET}.json=${FILE}

  # Delete the local copy of the secret to reduce the risk of compromise.
  rm ${FILE}
}

downloadK8sManifests() {
  # Download K8s manifests for resources to deploy on the cluster.
  mkdir -p ${KUBEFLOW_K8S_MANIFESTS_DIR}
  # Install the GPU driver. It has no effect on non-GPU nodes.
  curl -o ${KUBEFLOW_K8S_MANIFESTS_DIR}/daemonset-preloaded.yaml \
    https://raw.githubusercontent.com/GoogleCloudPlatform/container-engine-accelerators/stable/nvidia-driver-installer/cos/daemonset-preloaded.yaml

  curl -o ${KUBEFLOW_K8S_MANIFESTS_DIR}/rbac-setup.yaml \
  https://storage.googleapis.com/stackdriver-kubernetes/stable/rbac-setup.yaml

  curl -o ${KUBEFLOW_K8S_MANIFESTS_DIR}/agents.yaml \
        https://storage.googleapis.com/stackdriver-kubernetes/stable/agents.yaml

}

updateDeployment() {
  # Update or create a GCP deployment.
  local NAME=$1
  local CONFIG_FILE=$2
  pushd .
  cd ${KUBEFLOW_DM_DIR}
  # Check if it already exists
  set +e
  O=`gcloud deployment-manager --project=${PROJECT} deployments describe ${DEPLOYMENT_NAME} 2>&1`
  exists=$?
  set -e

  if [ ${exists} -eq 0 ]; then
    echo ${DEPLOYMENT_NAME} exists
    gcloud deployment-manager --project=${PROJECT} deployments update ${NAME} --config=${CONFIG_FILE}
  else
    # Run Deployment Manager
    gcloud deployment-manager --project=${PROJECT} deployments create ${NAME} --config=${CONFIG_FILE}
  fi

  popd
}

updateDM() {
  # TODO(jlewi): We should create deployments for all .yaml files in
  # the gcp_config directory.
  updateDeployment ${DEPLOYMENT_NAME} ${CONFIG_FILE}

  # Network needs to be created before GCFS.
  if [ -f ${KUBEFLOW_DM_DIR}/network.yaml ]; then
    updateDeployment ${DEPLOYMENT_NAME}-network network.yaml
  fi

  if [ -f ${KUBEFLOW_DM_DIR}/gcfs.yaml ]; then
    updateDeployment ${DEPLOYMENT_NAME}-gcfs gcfs.yaml
  fi

  python "${KUBEFLOW_REPO}/scripts/gke/iam_patch.py" --action=add \
    --project=${PROJECT} \
    --iam_bindings_file="${KUBEFLOW_DM_DIR}/iam_bindings.yaml"

  # Set credentials for kubectl context
  gcloud --project=${PROJECT} container clusters get-credentials --zone=${ZONE} ${DEPLOYMENT_NAME}

  # Create a conveniently named context
  CURRENT_CONTEXT=$(kubectl config current-context)
  CURRENT_CLUSTER=$(kubectl config get-contexts $CURRENT_CONTEXT | tail -1 | awk '{print $3}')
  CURRENT_USER=$(kubectl config get-contexts $CURRENT_CONTEXT | tail -1 | awk '{print $4}')

  kubectl config set-context ${KUBEFLOW_K8S_CONTEXT} \
    --namespace ${K8S_NAMESPACE} \
  --cluster $CURRENT_CLUSTER \
  --user $CURRENT_USER

  echo created context named: ${KUBEFLOW_K8S_CONTEXT}
  kubectl config use-context ${KUBEFLOW_K8S_CONTEXT}
  # Make yourself cluster admin
  set +e
  O=`kubectl get clusterrolebinding default-admin 2>&1`
  RESULT=$?
  set -e

  if [ "${RESULT}" -eq 0 ]; then
    echo clusterrolebinding default-admin already exists
  else
    kubectl create clusterrolebinding default-admin --clusterrole=cluster-admin --user=${EMAIL}
  fi

  set +e
  O=`kubectl get namespace ${K8S_NAMESPACE} 2>&1`
  RESULT=$?
  set -e

  if [ "${RESULT}" -eq 0 ]; then
    echo namespace ${K8S_NAMESPACE} already exists
  else
    kubectl create namespace ${K8S_NAMESPACE}
  fi

  # Install the GPU driver. It has no effect on non-GPU nodes.
  kubectl apply -f ${KUBEFLOW_K8S_MANIFESTS_DIR}/daemonset-preloaded.yaml

  # Install Stackdriver Kubernetes agents.
  kubectl apply -f ${KUBEFLOW_K8S_MANIFESTS_DIR}/rbac-setup.yaml --as=admin --as-group=system:masters
  kubectl apply -f ${KUBEFLOW_K8S_MANIFESTS_DIR}/agents.yaml
}

createSecrets() {
  # Whether to setup the project. Set to false to skip setting up the project.
  local ADMIN_EMAIL=${DEPLOYMENT_NAME}-admin@${PROJECT}.iam.gserviceaccount.com
  local USER_EMAIL=${DEPLOYMENT_NAME}-user@${PROJECT}.iam.gserviceaccount.com

  check_variable "${CLIENT_ID}" "CLIENT_ID"
  check_variable "${CLIENT_SECRET}" "CLIENT_SECRET"

  # We want the secret name to be the same by default for all clusters so that users don't have to set it manually.
  createGcpSecret ${ADMIN_EMAIL} admin-gcp-sa
  createGcpSecret ${USER_EMAIL} user-gcp-sa

  set +e
  O=`kubectl get secret --namespace=${K8S_NAMESPACE} kubeflow-oauth 2>&1`
  RESULT=$?
  set -e

  if [ "${RESULT}" -eq 0 ]; then
    echo Secret kubeflow-oauth already exists
  else
    kubectl create secret generic --namespace=${K8S_NAMESPACE} kubeflow-oauth --from-literal=client_id=${CLIENT_ID} --from-literal=client_secret=${CLIENT_SECRET}
  fi
}

gcpGenerateKsApp() {
  pushd .
  cd "${KUBEFLOW_KS_DIR}"

  ks generate cloud-endpoints cloud-endpoints
  ks generate cert-manager cert-manager --acmeEmail=${EMAIL}
  ks generate iap-ingress iap-ingress --ipName=${KUBEFLOW_IP_NAME} --hostname=${KUBEFLOW_HOSTNAME}
  ks param set jupyterhub jupyterHubAuthenticator iap
  popd
}

gcpKsApply() {
  # Apply the components generated
  pushd .
  cd "${KUBEFLOW_KS_DIR}"

  set +e
  O=`ks env describe default 2>&1`
  RESULT=$?
  set -e

  if [ "${RESULT}" -eq 0 ]; then
    echo environment default already exists
  else
    ks env add default --namespace "${K8S_NAMESPACE}"
  fi

  ks apply default -c cloud-endpoints
  ks apply default -c cert-manager
  ks apply default -c iap-ingress
  ks apply default -c pytorch-operator

  popd
}
