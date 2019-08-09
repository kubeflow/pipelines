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
#
# This script is picked by PROW for testing purpose.
#TODO(numerology): Further unification of the naming convention across pipeline code repo, in order
# to reduce risk of unintended error.

set -xe

# Usage information of this script.
usage() {
  echo "usage: run_test.sh
    [--results-gcs-dir              GCS directory for the test results]
    [--target-image-prefix          image prefix]
    [--dataflow-tft-image           image path to the dataflow tft]
    [--dataflow-predict-image       image path to the dataflow predict]
    [--dataflow-tfma-image          image path to the dataflow tfma]
    [--dataflow-tfdv-image          image path to the dataflow tfdv]
    [--dataproc-create-cluster-image        image path to the dataproc create cluster]
    [--dataproc-delete-cluster-image        image path to the dataproc delete cluster]
    [--dataproc-analyze-image               image path to the dataproc analyze]
    [--dataproc-transform-image             image path to the dataproc transform]
    [--dataproc-train-image                 image path to the dataproc train]
    [--dataproc-predict-image       image path to the dataproc predict]
    [--kubeflow-dnntrainer-image    image path to the kubeflow dnntrainer]
    [--kubeflow-deployer-image      image path to the kubeflow deployer]
    [--local-confusionmatrix-image  image path to the confusion matrix]
    [--local-roc-image              image path to the roc]
    [--namespace                    namespace for the deployed pipeline system]
    [--test-name                    test name: tf-training, xgboost]
    [-h help]"
}

while [ "$1" != "" ]; do
  case $1 in
             --results-gcs-dir )                shift
                                                RESULTS_GCS_DIR=$1
                                                ;;
             --target-image-prefix )            shift
                                                TARGET_IMAGE_PREFIX=$1
                                                ;;
             --dataflow-tft-image )             shift
                                                DATAFLOW_TFT_IMAGE=$1
                                                ;;
             --dataflow-predict-image )         shift
                                                DATAFLOW_PREDICT_IMAGE=$1
                                                ;;
             --dataflow-tfma-image )            shift
                                                DATAFLOW_TFMA_IMAGE=$1
                                                ;;
             --dataflow-tfdv-image )            shift
                                                DATAFLOW_TFDV_IMAGE=$1
                                                ;;
             --dataproc-create-cluster-image )  shift
                                                DATAPROC_CREATE_CLUSTER_IMAGE=$1
                                                ;;
             --dataproc-delete-cluster-image )  shift
                                                DATAPROC_DELETE_CLUSTER_IMAGE=$1
                                                ;;
             --dataproc-analyze-image )         shift
                                                DATAPROC_ANALYZE_IMAGE=$1
                                                ;;
             --dataproc-transform-image )       shift
                                                DATAPROC_TRANSFORM_IMAGE=$1
                                                ;;
             --dataproc-train-image )           shift
                                                DATAPROC_TRAIN_IMAGE=$1
                                                ;;
             --dataproc-predict-image )         shift
                                                DATAPROC_PREDICT_IMAGE=$1
                                                ;;
             --kubeflow-dnntrainer-image )      shift
                                                KUBEFLOW_DNNTRAINER_IMAGE=$1
                                                ;;
             --kubeflow-deployer-image )        shift
                                                KUBEFLOW_DEPLOYER_IMAGE=$1
                                                ;;
             --local-confusionmatrix-image )    shift
                                                LOCAL_CONFUSIONMATRIX_IMAGE=$1
                                                ;;
             --local-roc-image )                shift
                                                LOCAL_ROC_IMAGE=$1
                                                ;;
             --namespace )                      shift
                                                NAMESPACE=$1
                                                ;;
             --test-name )                      shift
                                                TEST_NAME=$1
                                                ;;
             -h | --help )                      usage
                                                exit
                                                ;;
             * )                                usage
                                                exit 1
  esac
  shift
done

GITHUB_REPO=kubeflow/pipelines
BASE_DIR=/python/src/github.com/${GITHUB_REPO}
TEST_DIR=${BASE_DIR}/test/sample-test

################################################################################
# Utility function to setup working dir, input and output locations.
# Globals:
#   SAMPLE_TEST_RESULT, test output file.
#   SAMPLE_TEST_OUTPUT, test output location.
# Arguments:
#   Test name.
################################################################################
preparation() {
  SAMPLE_TEST_RESULT="junit_Sample$1Output.xml"
  SAMPLE_TEST_OUTPUT=${RESULTS_GCS_DIR}
  WORK_DIR=${BASE_DIR}/samples/core/$1
  cd ${WORK_DIR}
}

################################################################################
# Utility function to generate formatted result, and move it to a place for PROW
# to pick up.
# Arguments:
#   Test name.
################################################################################
check_result() {
  cd "${TEST_DIR}"
  python3 run_sample_test.py --input ${WORK_DIR}/$1.yaml \
  --result ${SAMPLE_TEST_RESULT} --output ${SAMPLE_TEST_OUTPUT} --testname $1 \
  --namespace ${NAMESPACE}

  echo "Copy the test results to GCS ${RESULTS_GCS_DIR}/"
  gsutil cp ${SAMPLE_TEST_RESULT} ${RESULTS_GCS_DIR}/${SAMPLE_TEST_RESULT}
}

################################################################################
# Utility function to prepare and validate the results for sample test based on
# ipynb.
# Arguments:
#   Test name.
################################################################################
check_notebook_result() {
  jupyter nbconvert --to python $1.ipynb
  #TODO(numerology): move repeated package installation into .ipynb notebook.
  pip3 install tensorflow==1.8.0
  ipython $1.py
  EXIT_CODE=$?
  cd ${TEST_DIR}

  if [[ $1 != "dsl_static_type_checking" ]]; then
    python3 check_notebook_results.py --experiment "$1-test" --testname \
    $1 --result ${SAMPLE_TEST_RESULT} --namespace ${NAMESPACE} \
    --exit-code ${EXIT_CODE}
  else
    # If the sample does not require pipeline running under an experiment, put it
    # here.
    python3 check_notebook_results.py --testname $1 --result \
    ${SAMPLE_TEST_RESULT} --exit-code ${EXIT_CODE}
  fi

  echo "Copy the test results to GCS ${RESULTS_GCS_DIR}/"
  gsutil cp ${SAMPLE_TEST_RESULT} ${RESULTS_GCS_DIR}/${SAMPLE_TEST_RESULT}
  # Need to make sure CMLE models are cleaned inside the notebook.
}

################################################################################
# Utility function to inject correct images to generated yaml files for
# tfx-cab-classification test.
################################################################################
tfx_cab_classification_injection() {
  # Update the image tag in the yaml.
  sed -i "s|gcr.io/ml-pipeline/ml-pipeline-dataflow-tft:\([a-zA-Z0-9_.-]\)\+|${DATAFLOW_TFT_IMAGE}|g" ${TEST_NAME}.yaml
  sed -i "s|gcr.io/ml-pipeline/ml-pipeline-dataflow-tf-predict:\([a-zA-Z0-9_.-]\)\+|${DATAFLOW_PREDICT_IMAGE}|g" ${TEST_NAME}.yaml
  sed -i "s|gcr.io/ml-pipeline/ml-pipeline-dataflow-tfdv:\([a-zA-Z0-9_.-]\)\+|${DATAFLOW_TFDV_IMAGE}|g" ${TEST_NAME}.yaml
  sed -i "s|gcr.io/ml-pipeline/ml-pipeline-dataflow-tfma:\([a-zA-Z0-9_.-]\)\+|${DATAFLOW_TFMA_IMAGE}|g" ${TEST_NAME}.yaml
  sed -i "s|gcr.io/ml-pipeline/ml-pipeline-kubeflow-tf-trainer:\([a-zA-Z0-9_.-]\)\+|${KUBEFLOW_DNNTRAINER_IMAGE}|g" ${TEST_NAME}.yaml
  sed -i "s|gcr.io/ml-pipeline/ml-pipeline-kubeflow-deployer:\([a-zA-Z0-9_.-]\)\+|${KUBEFLOW_DEPLOYER_IMAGE}|g" ${TEST_NAME}.yaml
  sed -i "s|gcr.io/ml-pipeline/ml-pipeline-local-confusion-matrix:\([a-zA-Z0-9_.-]\)\+|${LOCAL_CONFUSIONMATRIX_IMAGE}|g" ${TEST_NAME}.yaml
  sed -i "s|gcr.io/ml-pipeline/ml-pipeline-local-roc:\([a-zA-Z0-9_.-]\)\+|${LOCAL_ROC_IMAGE}|g" ${TEST_NAME}.yaml
}

################################################################################
# Utility function to inject correct images to generated yaml files for
# xgboost-training-cm test.
################################################################################
xgboost_training_cm_injection() {
  sed -i "s|gcr.io/ml-pipeline/ml-pipeline-dataproc-create-cluster:\([a-zA-Z0-9_.-]\)\+|${DATAPROC_CREATE_CLUSTER_IMAGE}|g" ${TEST_NAME}.yaml
  sed -i "s|gcr.io/ml-pipeline/ml-pipeline-dataproc-delete-cluster:\([a-zA-Z0-9_.-]\)\+|${DATAPROC_DELETE_CLUSTER_IMAGE}|g" ${TEST_NAME}.yaml
  sed -i "s|gcr.io/ml-pipeline/ml-pipeline-dataproc-analyze:\([a-zA-Z0-9_.-]\)\+|${DATAPROC_ANALYZE_IMAGE}|g" ${TEST_NAME}.yaml
  sed -i "s|gcr.io/ml-pipeline/ml-pipeline-dataproc-transform:\([a-zA-Z0-9_.-]\)\+|${DATAPROC_TRANSFORM_IMAGE}|g" ${TEST_NAME}.yaml
  sed -i "s|gcr.io/ml-pipeline/ml-pipeline-dataproc-train:\([a-zA-Z0-9_.-]\)\+|${DATAPROC_TRAIN_IMAGE}|g" ${TEST_NAME}.yaml
  sed -i "s|gcr.io/ml-pipeline/ml-pipeline-dataproc-predict:\([a-zA-Z0-9_.-]\)\+|${DATAPROC_PREDICT_IMAGE}|g" ${TEST_NAME}.yaml
  sed -i "s|gcr.io/ml-pipeline/ml-pipeline-local-confusion-matrix:\([a-zA-Z0-9_.-]\)\+|${LOCAL_CONFUSIONMATRIX_IMAGE}|g" ${TEST_NAME}.yaml
  sed -i "s|gcr.io/ml-pipeline/ml-pipeline-local-roc:\([a-zA-Z0-9_.-]\)\+|${LOCAL_ROC_IMAGE}|g" ${TEST_NAME}.yaml
}

################################################################################
# Utility function to inject correct images to python files for
# kubeflow_training_classification test.
################################################################################
kubeflow_training_classification_injection() {
  sed -i "s|gcr.io/ml-pipeline/ml-pipeline-dataflow-tft:\([a-zA-Z0-9_.-]\)\+|${DATAFLOW_TFT_IMAGE}|g" ${TEST_NAME}.py
  sed -i "s|gcr.io/ml-pipeline/ml-pipeline-kubeflow-tf-trainer:\([a-zA-Z0-9_.-]\)\+|${KUBEFLOW_DNNTRAINER_IMAGE}|g" ${TEST_NAME}.py
  sed -i "s|gcr.io/ml-pipeline/ml-pipeline-dataflow-tf-predict:\([a-zA-Z0-9_.-]\)\+|${DATAFLOW_PREDICT_IMAGE}|g" ${TEST_NAME}.py
  sed -i "s|gcr.io/ml-pipeline/ml-pipeline-local-confusion-matrix:\([a-zA-Z0-9_.-]\)\+|${LOCAL_CONFUSIONMATRIX_IMAGE}|g" ${TEST_NAME}.py
}

if [[ -z "$RESULTS_GCS_DIR" ]]; then
  usage
  exit 1
fi

if [[ ! -z "${GOOGLE_APPLICATION_CREDENTIALS}" ]]; then
  gcloud auth activate-service-account \
  --key-file="${GOOGLE_APPLICATION_CREDENTIALS}"
fi

cd ${BASE_DIR}

#TODO(numerology): Move argo installation to Dockerfile to speedup test setup.

# Install argo
echo "install argo"
ARGO_VERSION=v2.3.0
mkdir -p ~/bin/
export PATH=~/bin/:$PATH
curl -sSL -o ~/bin/argo "https://github.com/argoproj/argo/releases/download/$ARGO_VERSION/argo-linux-amd64"
chmod +x ~/bin/argo

echo "Run the sample tests..."

# Run the tests
preparation ${TEST_NAME}

if [[ "${TEST_NAME}" == "kubeflow_training_classification" ]]; then
  #TODO(numerology): convert the sed commands to sed -e
  # 's|gcr.io/ml-pipeline/|gcr.io/ml-pipeline-test/' and tag replacement. Also
  # let the postsubmit tests refer to yaml files.
  if [ -n "${DATAFLOW_TFT_IMAGE}" ];then
    kubeflow_training_classification_injection
  fi

  dsl-compile --py "${TEST_NAME}.py" --output "${TEST_NAME}.yaml"
  check_result ${TEST_NAME}
elif [[ "${TEST_NAME}" == "tfx_cab_classification" ]]; then
  dsl-compile --py "${TEST_NAME}.py" --output "${TEST_NAME}.yaml"
  if [[ -n "${DATAFLOW_TFT_IMAGE}" ]]; then
    tfx_cab_classification_injection
  fi
  check_result ${TEST_NAME}
elif [[ "${TEST_NAME}" == "xgboost_training_cm" ]]; then
  dsl-compile --py "${TEST_NAME}.py" --output "${TEST_NAME}.yaml"
  if [[ -n "${DATAPROC_CREATE_CLUSTER_IMAGE}" ]]; then
    xgboost_training_cm_injection
  fi
  check_result ${TEST_NAME}
elif [[ "${TEST_NAME}" == "sequential" ]]; then
  dsl-compile --py "${TEST_NAME}.py" --output "${TEST_NAME}.yaml"
  check_result ${TEST_NAME}
elif [[ "${TEST_NAME}" == "condition" ]]; then
  dsl-compile --py "${TEST_NAME}.py" --output "${TEST_NAME}.yaml"
  check_result ${TEST_NAME}
elif [[ "${TEST_NAME}" == "exit_handler" ]]; then
  dsl-compile --py "${TEST_NAME}.py" --output "${TEST_NAME}.yaml"
  check_result ${TEST_NAME}
elif [[ "${TEST_NAME}" == "parallel_join" ]]; then
  dsl-compile --py "${TEST_NAME}.py" --output "${TEST_NAME}.yaml"
  check_result ${TEST_NAME}
elif [[ "${TEST_NAME}" == "recursion" ]]; then
  dsl-compile --py "${TEST_NAME}.py" --output "${TEST_NAME}.yaml"
  check_result ${TEST_NAME}
elif [[ "${TEST_NAME}" == "kubeflow_pipeline_using_TFX_OSS_components" ]]; then
  # CMLE model name format: A name should start with a letter and contain only
  # letters, numbers and underscores.
  DEPLOYER_MODEL=`cat /proc/sys/kernel/random/uuid`
  DEPLOYER_MODEL=Notebook_tfx_taxi_`echo ${DEPLOYER_MODEL//-/_}`

  export LC_ALL=C.UTF-8
  export LANG=C.UTF-8
  if [[ -n "${DATAFLOW_TFT_IMAGE}" ]]; then
    papermill --prepare-only -p EXPERIMENT_NAME "${TEST_NAME}-test" -p OUTPUT_DIR \
    ${RESULTS_GCS_DIR} -p PROJECT_NAME ml-pipeline-test \
    -p BASE_IMAGE ${TARGET_IMAGE_PREFIX}pusherbase:dev -p TARGET_IMAGE \
    ${TARGET_IMAGE_PREFIX}pusher:dev -p TARGET_IMAGE_TWO \
    ${TARGET_IMAGE_PREFIX}pusher_two:dev \
    -p KFP_PACKAGE /tmp/kfp.tar.gz -p DEPLOYER_MODEL ${DEPLOYER_MODEL}  \
    -p DATAFLOW_TFDV_IMAGE ${DATAFLOW_TFDV_IMAGE} -p DATAFLOW_TFT_IMAGE \
    ${DATAFLOW_TFT_IMAGE} -p DATAFLOW_TFMA_IMAGE ${DATAFLOW_TFMA_IMAGE} -p \
    DATAFLOW_TF_PREDICT_IMAGE ${DATAFLOW_PREDICT_IMAGE} \
    -p KUBEFLOW_TF_TRAINER_IMAGE ${KUBEFLOW_DNNTRAINER_IMAGE} -p \
    KUBEFLOW_DEPLOYER_IMAGE ${KUBEFLOW_DEPLOYER_IMAGE} \
    -p TRAIN_DATA gs://ml-pipeline-dataset/sample-test/taxi-cab-classification/train50.csv \
    -p EVAL_DATA gs://ml-pipeline-dataset/sample-test/taxi-cab-classification/eval20.csv \
    -p HIDDEN_LAYER_SIZE 10 -p STEPS 50 \
    "KubeFlow Pipeline Using TFX OSS Components.ipynb" "${TEST_NAME}.ipynb"
  else
    papermill --prepare-only -p EXPERIMENT_NAME "${TEST_NAME}-test" -p \
    OUTPUT_DIR ${RESULTS_GCS_DIR} -p PROJECT_NAME ml-pipeline-test \
    -p BASE_IMAGE ${TARGET_IMAGE_PREFIX}pusherbase:dev -p TARGET_IMAGE \
    ${TARGET_IMAGE_PREFIX}pusher:dev -p TARGET_IMAGE_TWO \
    ${TARGET_IMAGE_PREFIX}pusher_two:dev \
    -p KFP_PACKAGE /tmp/kfp.tar.gz -p DEPLOYER_MODEL ${DEPLOYER_MODEL} \
    -p TRAIN_DATA gs://ml-pipeline-dataset/sample-test/taxi-cab-classification/train50.csv \
    -p EVAL_DATA gs://ml-pipeline-dataset/sample-test/taxi-cab-classification/eval20.csv \
    -p HIDDEN_LAYER_SIZE 10 -p STEPS 50 \
    "KubeFlow Pipeline Using TFX OSS Components.ipynb" "${TEST_NAME}.ipynb"
  fi
  check_notebook_result ${TEST_NAME}
elif [[ "${TEST_NAME}" == "lightweight_component" ]]; then
  export LC_ALL=C.UTF-8
  export LANG=C.UTF-8
  papermill --prepare-only -p EXPERIMENT_NAME "${TEST_NAME}-test" -p \
  PROJECT_NAME ml-pipeline-test -p KFP_PACKAGE /tmp/kfp.tar.gz  "Lightweight Python components - basics.ipynb" \
  "${TEST_NAME}.ipynb"

  check_notebook_result ${TEST_NAME}
elif [[ "${TEST_NAME}" == "dsl_static_type_checking" ]]; then

  cd ${BASE_DIR}/samples/core/dsl_static_type_checking
  export LC_ALL=C.UTF-8
  export LANG=C.UTF-8
  papermill --prepare-only -p KFP_PACKAGE /tmp/kfp.tar.gz \
  "DSL Static Type Checking.ipynb" "${TEST_NAME}.ipynb"

  check_notebook_result ${TEST_NAME}
fi
