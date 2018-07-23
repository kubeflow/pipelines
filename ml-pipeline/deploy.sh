#!/bin/bash

# Copyright 2018 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# K8s Namespace that all resources deployed to
NAMESPACE=default

# Ksonnet app name
APP_DIR=ml-pipeline-app

# Version number of this release.
RELEASE_VERSION="${RELEASE_VERSION:-0.0.9}"

# Default ml pipeline api server image
API_SERVER_IMAGE="gcr.io/ml-pipeline/api-server:${RELEASE_VERSION}"

# Default ml pipeline scheduledworkflow CRD controller image
SCHEDULED_WORKFLOW_IMAGE="gcr.io/ml-pipeline/scheduledworkflow:${RELEASE_VERSION}"

# Default ml pipeline persistence agent image
PERSISTENCE_AGENT_IMAGE="gcr.io/ml-pipeline/persistenceagent:${RELEASE_VERSION}"

# Default ml pipeline ui image
UI_IMAGE="gcr.io/ml-pipeline/frontend:${RELEASE_VERSION}"

# Whether report usage or not. Default yes.
REPORT_USAGE="true"

# Whether to deploy argo or not. Argo might already exist in the cluster,
# installed by Kubeflow or exist in a test cluster.
DEPLOY_ARGO="true"

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

if ${UNINSTALL} ; then
  ( cd ${APP_DIR} && ks delete default)
else
  ( cd ${APP_DIR} && ks apply default -c ml-pipeline)
fi
