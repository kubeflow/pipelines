#!/bin/bash

# K8s Namespace that all resources deployed to
NAMESPACE=default

# Ksonnet app name
APP_DIR=ml-pipeline

# Default ml pipeline api server image
API_SERVER_IMAGE=gcr.io/ml-pipeline/api-server

# Default ml pipeline ui image
UI_IMAGE=gcr.io/ml-pipeline/frontend

# Parameter supported:
# -n namespace
# -a ml-pipeline apiserver docker image
# -u ml-pipeline frontend docker image
while getopts "n:a:u:" opt; do
  case $opt in
    n)
      NAMESPACE="$OPTARG"
      ;;
    a)
      API_SERVER_IMAGE="$OPTARG"
      ;;
    u)
      UI_IMAGE="$OPTARG"
      ;;
    *)
      echo "-$opt not recognized"
      ;;
  esac
done


echo "Initialize a ksonnet APP ..."
ks init ${APP_DIR}
echo "Initialized ksonnet APP successfully"


# Import pipeline registry
# Note: Since the repository is not yet public, and ksonnet can't add a local registry due to
# an known issue: https://github.com/ksonnet/ksonnet/issues/232, we are working around by creating
# a symbolic links in ./vendor and manually modifying app.yaml
# when the repo is public we can do following:
# ks registry add ml-pipeline github.com/googleprivate/ml/tree/master/deploy
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

time="`date +%Y%m%d%H%M%S`"

# Generate a ksonnet component manifest and assign parameters
( cd ${APP_DIR} && ks generate ml-pipeline ml-pipeline-${time} --namespace=${NAMESPACE} )
( cd ${APP_DIR} && ks param set ml-pipeline-${time} api_image ${API_SERVER_IMAGE} )
( cd ${APP_DIR} && ks param set ml-pipeline-${time} ui_image ${UI_IMAGE} )
( cd ${APP_DIR} && ks param set ml-pipeline-${time} report_usage "true" )
( cd ${APP_DIR} && ks param set ml-pipeline-${time} usage_id $(uuidgen) )

# Deploy ml-pipeline
( cd ${APP_DIR} && ks apply default -c ml-pipeline-${time} )
