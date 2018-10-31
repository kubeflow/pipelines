#!/bin/bash
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

# K8s Namespace that all resources deployed to
NAMESPACE=kubeflow

# ML pipeline Ksonnet app name
APP_DIR=ml-pipeline-app

# Kubeflow directory
KF_DIR=/kf-app

# Kubeflow Ksonnet app name
KFAPP=/kubeflow-ks-app

# Version number of this release.
RELEASE_VERSION="${RELEASE_VERSION:-0.0.20}"

# Default ml pipeline api server image
API_SERVER_IMAGE="${API_SERVER_IMAGE:-gcr.io/ml-pipeline/api-server:${RELEASE_VERSION}}"

# Default ml pipeline scheduledworkflow CRD controller image
SCHEDULED_WORKFLOW_IMAGE="${SCHEDULED_WORKFLOW_IMAGE:-gcr.io/ml-pipeline/scheduledworkflow:${RELEASE_VERSION}}"

# Default ml pipeline persistence agent image
PERSISTENCE_AGENT_IMAGE="${PERSISTENCE_AGENT_IMAGE:-gcr.io/ml-pipeline/persistenceagent:${RELEASE_VERSION}}"

# Default ml pipeline ui image
UI_IMAGE="${UI_IMAGE:-gcr.io/ml-pipeline/frontend:${RELEASE_VERSION}}"

# Whether report usage or not. Default yes.
REPORT_USAGE="true"

# Whether to deploy argo or not. Argo might already exist in the cluster,
# installed by Kubeflow or exist in a test cluster.
DEPLOY_ARGO="true"

# Whether to include kubeflow or not.
WITH_KUBEFLOW="true"

# Whether this is an install or uninstall.
UNINSTALL=false

usage()
{
    echo "usage: deploy.sh
    [-n | --namespace namespace ]
    [-a | --api_image ml-pipeline apiserver docker image to use ]
    [-w | --scheduled_workflow_image ml-pipeline scheduled workflow controller image]
    [-p | --persistence_agent_image ml-pipeline persistence agent image]
    [-u | --ui_image ml-pipeline frontend UI docker image]
    [-r | --report_usage deploy roles or not. Roles are needed for GKE]
    [--with_kubeflow whether to include kubeflow or not]
    [--deploy_argo whether to deploy argo or not]
    [--uninstall uninstall ml pipeline]
    [-h help]"
}

while [ "$1" != "" ]; do
    case $1 in
        -n | --namespace )                   shift
                                             NAMESPACE=$1
                                             ;;
        -a | --api_image )                   shift
                                             API_SERVER_IMAGE=$1
                                             ;;
        -w | --scheduled_workflow_image )    shift
                                             SCHEDULED_WORKFLOW_IMAGE=$1
                                             ;;
        -p | --persistence_agent_image )     shift
                                             PERSISTENCE_AGENT_IMAGE=$1
                                             ;;
        -u | --ui_image )                    shift
                                             UI_IMAGE=$1
                                             ;;
        -r | --report_usage )                shift
                                             REPORT_USAGE=$1
                                             ;;
        --with_kubeflow )                    shift
                                             WITH_KUBEFLOW=$1
                                             ;;
        --deploy_argo )                      shift
                                             DEPLOY_ARGO=$1
                                             ;;
        --uninstall )                        UNINSTALL=true
                                             ;;
        -h | --help )                        usage
                                             exit
                                             ;;
        * )                                  usage
                                             exit 1
    esac
    shift
done

echo "Configure ksonnet ..."
/ml-pipeline/bootstrapper.sh
echo "Configure ksonnet completed successfully"

echo "Initialize a ksonnet APP ..."
ks init ${APP_DIR}
echo "Initialized ksonnet APP completed successfully"


# Import pipeline registry
# Note: Since the repository is not yet public, and ksonnet can't add a local registry due to
# an known issue: https://github.com/ksonnet/ksonnet/issues/232, we are working around by creating
# a symbolic links in ./vendor and manually modifying app.yaml
# when the repo is public we can do following:
# ks registry add ml-pipeline github.com/googleprivate/ml/tree/master/ml-pipeline
# ks pkg install ml-pipeline/ml-pipeline
BASEDIR=$(cd $(dirname "$0") && pwd)
ln -s ${BASEDIR} ${APP_DIR}/vendor/ml-pipeline

# Modifying the app.yaml
sed '/kind: ksonnet.io\/app/r '<(cat<<'EOF'
libraries:
  ml-pipeline:
    name: ml-pipeline
    registry: ml-pipeline
EOF
)  ${APP_DIR}/app.yaml > ${APP_DIR}/tmp.yaml
mv ${APP_DIR}/tmp.yaml ${APP_DIR}/app.yaml


kubectl get ns ${NAMESPACE} &>/dev/null
if [ $? == 0 ]; then
  echo "namespace ${NAMESPACE} exist"
else
  echo "Creating a new kubernetes namespace ..."
  kubectl create ns ${NAMESPACE}
fi

# Generate a ksonnet component manifest and assign parameters
( cd ${APP_DIR} && ks generate ml-pipeline ml-pipeline --namespace=${NAMESPACE} )
( cd ${APP_DIR} && ks param set ml-pipeline api_image ${API_SERVER_IMAGE} )
( cd ${APP_DIR} && ks param set ml-pipeline scheduledworkflow_image ${SCHEDULED_WORKFLOW_IMAGE} )
( cd ${APP_DIR} && ks param set ml-pipeline persistenceagent_image ${PERSISTENCE_AGENT_IMAGE} )
( cd ${APP_DIR} && ks param set ml-pipeline ui_image ${UI_IMAGE} )
( cd ${APP_DIR} && ks param set ml-pipeline deploy_argo ${DEPLOY_ARGO} )
( cd ${APP_DIR} && ks param set ml-pipeline report_usage ${REPORT_USAGE} )
( cd ${APP_DIR} && ks param set ml-pipeline usage_id $(uuidgen) )

if [ "$WITH_KUBEFLOW" = true ]; then
  # v0.2 non-gke deploy script doesn't create a namespace. This would be fixed in the later version.
  # https://github.com/kubeflow/kubeflow/blob/master/scripts/deploy.sh#L43
  mkdir -p ${KF_DIR}
  # We use kubeflow v0.3.0 by default
  KUBEFLOW_VERSION=${KUBEFLOW_VERSION:-"v0.3.0"}
  (cd ${KF_DIR} && curl -L -o kubeflow.tar.gz https://github.com/kubeflow/kubeflow/archive/${KUBEFLOW_VERSION}.tar.gz)
  tar -xzf ${KF_DIR}/kubeflow.tar.gz  -C ${KF_DIR}
  KUBEFLOW_REPO=$(find ${KF_DIR} -maxdepth 1 -type d -name "kubeflow*")
  if [[ ${REPORT_USAGE} != "true" ]]; then
      (cd ${KUBEFLOW_REPO} && sed -i -e 's/--reportUsage\=true/--reportUsage\=false/g' scripts/util.sh)
  fi
  (${KUBEFLOW_REPO}/scripts/kfctl.sh init ${KFAPP} --platform none)
  (cd ${KFAPP} && ${KUBEFLOW_REPO}/scripts/kfctl.sh generate k8s && ${KUBEFLOW_REPO}/scripts/kfctl.sh apply k8s)
fi

if ${UNINSTALL} ; then
  ( cd ${APP_DIR} && ks delete default)
  if [ "$WITH_KUBEFLOW" = true ]; then
    KUBEFLOW_REPO=$(find ${KF_DIR} -maxdepth 1 -type d -name "kubeflow*")
    # Uninstall Kubeflow
    (cd ${KUBEFLOW_REPO}${KFAPP} && ks delete default)
  fi
  exit 0
fi

# Install ML pipeline
( cd ${APP_DIR} && ks apply default -c ml-pipeline)

# Wait for service to be ready

MAX_ATTEMPT=60
READY_KEYWORD="\"apiServerReady\":true"
CA_CERT=/var/run/secrets/kubernetes.io/serviceaccount/ca.crt
TOKEN=$(cat /var/run/secrets/kubernetes.io/serviceaccount/token)

# probing the UI healthz/ status until it's ready. Timeout after 4 minutes
echo "Waiting for ML pipeline to be ready..."
for i in $(seq 1 ${MAX_ATTEMPT})
do
  echo -n .
  UI_STATUS=`curl -sS --cacert $CA_CERT -H "Authorization: Bearer $TOKEN" https://${KUBERNETES_SERVICE_HOST}:${KUBERNETES_SERVICE_PORT}/api/v1/namespaces/${NAMESPACE}/services/ml-pipeline-ui:80/proxy/apis/v1beta1/healthz`
  echo $UI_STATUS | grep -q ${READY_KEYWORD} && s=0 && break || s=$? && sleep 4
done

if [[ $s != 0 ]]
  then echo "ML Pipeline not start successfully after 4 minutes. Timeout..." && exit $s
else
  echo "ML Pipeline Is Now Ready" && exit $s
fi
