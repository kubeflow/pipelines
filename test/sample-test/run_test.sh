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

set -xe

usage()
{
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

if [ -z "$RESULTS_GCS_DIR" ]; then
    usage
    exit 1
fi

if [[ ! -z "${GOOGLE_APPLICATION_CREDENTIALS}" ]]; then
  gcloud auth activate-service-account --key-file="${GOOGLE_APPLICATION_CREDENTIALS}"
fi

GITHUB_REPO=kubeflow/pipelines
BASE_DIR=/python/src/github.com/${GITHUB_REPO}
TEST_DIR=${BASE_DIR}/test/sample-test

cd ${BASE_DIR}

# Install argo
echo "install argo"
ARGO_VERSION=v2.2.0
mkdir -p ~/bin/
export PATH=~/bin/:$PATH
curl -sSL -o ~/bin/argo https://github.com/argoproj/argo/releases/download/$ARGO_VERSION/argo-linux-amd64
chmod +x ~/bin/argo

echo "Run the sample tests..."

# Generate Python package
cd ${BASE_DIR}/sdk/python
./build.sh /tmp/kfp.tar.gz

# Install python client, including DSL compiler.
pip3 install /tmp/kfp.tar.gz

# Run the tests
if [ "$TEST_NAME" == 'tf-training' ]; then
  SAMPLE_KUBEFLOW_TEST_RESULT=junit_SampleKubeflowOutput.xml
  SAMPLE_KUBEFLOW_TEST_OUTPUT=${RESULTS_GCS_DIR}

  #TODO: convert the sed commands to sed -e 's|gcr.io/ml-pipeline/|gcr.io/ml-pipeline-test/' and tag replacement. 
  # Compile samples
  cd ${BASE_DIR}/samples/kubeflow-tf

  if [ -n "${DATAFLOW_TFT_IMAGE}" ];then
    sed -i -e "s|gcr.io/ml-pipeline/ml-pipeline-dataflow-tft:\([a-zA-Z0-9_.-]\)\+|${DATAFLOW_TFT_IMAGE}|g" kubeflow-training-classification.py
    sed -i -e "s|gcr.io/ml-pipeline/ml-pipeline-kubeflow-tf-trainer:\([a-zA-Z0-9_.-]\)\+|${KUBEFLOW_DNNTRAINER_IMAGE}|g" kubeflow-training-classification.py
    sed -i -e "s|gcr.io/ml-pipeline/ml-pipeline-dataflow-tf-predict:\([a-zA-Z0-9_.-]\)\+|${DATAFLOW_PREDICT_IMAGE}|g" kubeflow-training-classification.py
    sed -i -e "s|gcr.io/ml-pipeline/ml-pipeline-local-confusion-matrix:\([a-zA-Z0-9_.-]\)\+|${LOCAL_CONFUSIONMATRIX_IMAGE}|g" kubeflow-training-classification.py
  fi

  dsl-compile --py kubeflow-training-classification.py --output kubeflow-training-classification.zip

  cd "${TEST_DIR}"
  python3 run_kubeflow_test.py --input ${BASE_DIR}/samples/kubeflow-tf/kubeflow-training-classification.zip --result $SAMPLE_KUBEFLOW_TEST_RESULT --output $SAMPLE_KUBEFLOW_TEST_OUTPUT --namespace ${NAMESPACE}

  echo "Copy the test results to GCS ${RESULTS_GCS_DIR}/"
  gsutil cp ${SAMPLE_KUBEFLOW_TEST_RESULT} ${RESULTS_GCS_DIR}/${SAMPLE_KUBEFLOW_TEST_RESULT}
elif [ "$TEST_NAME" == "tfx" ]; then
  SAMPLE_TFX_TEST_RESULT=junit_SampleTFXOutput.xml
  SAMPLE_TFX_TEST_OUTPUT=${RESULTS_GCS_DIR}
  
  # Compile samples
  cd ${BASE_DIR}/samples/tfx

  dsl-compile --py taxi-cab-classification-pipeline.py --output taxi-cab-classification-pipeline.yaml

  if [ -n "${DATAFLOW_TFT_IMAGE}" ];then
    # Update the image tag in the yaml.
    sed -i -e "s|gcr.io/ml-pipeline/ml-pipeline-dataflow-tft:\([a-zA-Z0-9_.-]\)\+|${DATAFLOW_TFT_IMAGE}|g" taxi-cab-classification-pipeline.yaml
    sed -i -e "s|gcr.io/ml-pipeline/ml-pipeline-dataflow-tf-predict:\([a-zA-Z0-9_.-]\)\+|${DATAFLOW_PREDICT_IMAGE}|g" taxi-cab-classification-pipeline.yaml
    sed -i -e "s|gcr.io/ml-pipeline/ml-pipeline-dataflow-tfdv:\([a-zA-Z0-9_.-]\)\+|${DATAFLOW_TFDV_IMAGE}|g" taxi-cab-classification-pipeline.yaml
    sed -i -e "s|gcr.io/ml-pipeline/ml-pipeline-dataflow-tfma:\([a-zA-Z0-9_.-]\)\+|${DATAFLOW_TFMA_IMAGE}|g" taxi-cab-classification-pipeline.yaml
    sed -i -e "s|gcr.io/ml-pipeline/ml-pipeline-kubeflow-tf-trainer:\([a-zA-Z0-9_.-]\)\+|${KUBEFLOW_DNNTRAINER_IMAGE}|g" taxi-cab-classification-pipeline.yaml
    sed -i -e "s|gcr.io/ml-pipeline/ml-pipeline-kubeflow-deployer:\([a-zA-Z0-9_.-]\)\+|${KUBEFLOW_DEPLOYER_IMAGE}|g" taxi-cab-classification-pipeline.yaml
    sed -i -e "s|gcr.io/ml-pipeline/ml-pipeline-local-confusion-matrix:\([a-zA-Z0-9_.-]\)\+|${LOCAL_CONFUSIONMATRIX_IMAGE}|g" taxi-cab-classification-pipeline.yaml
    sed -i -e "s|gcr.io/ml-pipeline/ml-pipeline-local-roc:\([a-zA-Z0-9_.-]\)\+|${LOCAL_ROC_IMAGE}|g" taxi-cab-classification-pipeline.yaml
  fi

  cd "${TEST_DIR}"
  python3 run_tfx_test.py --input ${BASE_DIR}/samples/tfx/taxi-cab-classification-pipeline.yaml --result $SAMPLE_TFX_TEST_RESULT --output $SAMPLE_TFX_TEST_OUTPUT --namespace ${NAMESPACE}
  echo "Copy the test results to GCS ${RESULTS_GCS_DIR}/"
  gsutil cp ${SAMPLE_TFX_TEST_RESULT} ${RESULTS_GCS_DIR}/${SAMPLE_TFX_TEST_RESULT}
elif [ "$TEST_NAME" == "sequential" ]; then
  SAMPLE_SEQUENTIAL_TEST_RESULT=junit_SampleSequentialOutput.xml
  SAMPLE_SEQUENTIAL_TEST_OUTPUT=${RESULTS_GCS_DIR}

  # Compile samples
  cd ${BASE_DIR}/samples/basic
  dsl-compile --py sequential.py --output sequential.zip

  cd "${TEST_DIR}"
  python3 run_basic_test.py --input ${BASE_DIR}/samples/basic/sequential.zip --result $SAMPLE_SEQUENTIAL_TEST_RESULT --output $SAMPLE_SEQUENTIAL_TEST_OUTPUT  --testname sequential --namespace ${NAMESPACE}

  echo "Copy the test results to GCS ${RESULTS_GCS_DIR}/"
  gsutil cp ${SAMPLE_SEQUENTIAL_TEST_RESULT} ${RESULTS_GCS_DIR}/${SAMPLE_SEQUENTIAL_TEST_RESULT}
elif [ "$TEST_NAME" == "condition" ]; then
  SAMPLE_CONDITION_TEST_RESULT=junit_SampleConditionOutput.xml
  SAMPLE_CONDITION_TEST_OUTPUT=${RESULTS_GCS_DIR}

  # Compile samples
  cd ${BASE_DIR}/samples/basic
  dsl-compile --py condition.py --output condition.zip

  cd "${TEST_DIR}"
  python3 run_basic_test.py --input ${BASE_DIR}/samples/basic/condition.zip --result $SAMPLE_CONDITION_TEST_RESULT --output $SAMPLE_CONDITION_TEST_OUTPUT --testname condition --namespace ${NAMESPACE}

  echo "Copy the test results to GCS ${RESULTS_GCS_DIR}/"
  gsutil cp ${SAMPLE_CONDITION_TEST_RESULT} ${RESULTS_GCS_DIR}/${SAMPLE_CONDITION_TEST_RESULT}
elif [ "$TEST_NAME" == "exithandler" ]; then
  SAMPLE_EXIT_HANDLER_TEST_RESULT=junit_SampleExitHandlerOutput.xml
  SAMPLE_EXIT_HANDLER_TEST_OUTPUT=${RESULTS_GCS_DIR}

  # Compile samples
  cd ${BASE_DIR}/samples/basic
  dsl-compile --py exit_handler.py --output exit_handler.zip

  cd "${TEST_DIR}"
  python3 run_basic_test.py --input ${BASE_DIR}/samples/basic/exit_handler.zip --result $SAMPLE_EXIT_HANDLER_TEST_RESULT --output $SAMPLE_EXIT_HANDLER_TEST_OUTPUT --testname exithandler --namespace ${NAMESPACE}

  echo "Copy the test results to GCS ${RESULTS_GCS_DIR}/"
  gsutil cp ${SAMPLE_EXIT_HANDLER_TEST_RESULT} ${RESULTS_GCS_DIR}/${SAMPLE_EXIT_HANDLER_TEST_RESULT}
elif [ "$TEST_NAME" == "immediatevalue" ]; then
  SAMPLE_IMMEDIATE_VALUE_TEST_RESULT=junit_SampleImmediateValueOutput.xml
  SAMPLE_IMMEDIATE_VALUE_TEST_OUTPUT=${RESULTS_GCS_DIR}

  # Compile samples
  cd ${BASE_DIR}/samples/basic
  dsl-compile --py immediate_value.py --output immediate_value.zip

  cd "${TEST_DIR}"
  python3 run_basic_test.py --input ${BASE_DIR}/samples/basic/immediate_value.zip --result $SAMPLE_IMMEDIATE_VALUE_TEST_RESULT --output $SAMPLE_IMMEDIATE_VALUE_TEST_OUTPUT --testname immediatevalue --namespace ${NAMESPACE}

  echo "Copy the test results to GCS ${RESULTS_GCS_DIR}/"
  gsutil cp ${SAMPLE_IMMEDIATE_VALUE_TEST_RESULT} ${RESULTS_GCS_DIR}/${SAMPLE_IMMEDIATE_VALUE_TEST_RESULT}
elif [ "$TEST_NAME" == "paralleljoin" ]; then
  SAMPLE_PARALLEL_JOIN_TEST_RESULT=junit_SampleParallelJoinOutput.xml
  SAMPLE_PARALLEL_JOIN_TEST_OUTPUT=${RESULTS_GCS_DIR}

  # Compile samples
  cd ${BASE_DIR}/samples/basic
  dsl-compile --py parallel_join.py --output parallel_join.zip

  cd "${TEST_DIR}"
  python3 run_basic_test.py --input ${BASE_DIR}/samples/basic/parallel_join.zip --result $SAMPLE_PARALLEL_JOIN_TEST_RESULT --output $SAMPLE_PARALLEL_JOIN_TEST_OUTPUT --testname paralleljoin --namespace ${NAMESPACE}

  echo "Copy the test results to GCS ${RESULTS_GCS_DIR}/"
  gsutil cp ${SAMPLE_PARALLEL_JOIN_TEST_RESULT} ${RESULTS_GCS_DIR}/${SAMPLE_PARALLEL_JOIN_TEST_RESULT}
elif [ "$TEST_NAME" == "recursion" ]; then
  SAMPLE_RECURSION_TEST_RESULT=junit_SampleRecursionOutput.xml
  SAMPLE_RECURSION_TEST_OUTPUT=${RESULTS_GCS_DIR}

  # Compile samples
  cd ${BASE_DIR}/samples/basic
  dsl-compile --py recursion.py --output recursion.tar.gz

  cd "${TEST_DIR}"
  python3 run_basic_test.py --input ${BASE_DIR}/samples/basic/recursion.tar.gz --result $SAMPLE_RECURSION_TEST_RESULT --output $SAMPLE_RECURSION_TEST_OUTPUT --testname recursion --namespace ${NAMESPACE}

  echo "Copy the test results to GCS ${RESULTS_GCS_DIR}/"
  gsutil cp ${SAMPLE_RECURSION_TEST_RESULT} ${RESULTS_GCS_DIR}/${SAMPLE_RECURSION_TEST_RESULT}
elif [ "$TEST_NAME" == "xgboost" ]; then
  SAMPLE_XGBOOST_TEST_RESULT=junit_SampleXGBoostOutput.xml
  SAMPLE_XGBOOST_TEST_OUTPUT=${RESULTS_GCS_DIR}

  # Compile samples
  cd ${BASE_DIR}/samples/xgboost-spark

  dsl-compile --py xgboost-training-cm.py --output xgboost-training-cm.yaml

  if [ -n "${DATAPROC_CREATE_CLUSTER_IMAGE}" ];then
    sed -i -e "s|gcr.io/ml-pipeline/ml-pipeline-dataproc-create-cluster:\([a-zA-Z0-9_.-]\)\+|${DATAPROC_CREATE_CLUSTER_IMAGE}|g" xgboost-training-cm.yaml
    sed -i -e "s|gcr.io/ml-pipeline/ml-pipeline-dataproc-delete-cluster:\([a-zA-Z0-9_.-]\)\+|${DATAPROC_DELETE_CLUSTER_IMAGE}|g" xgboost-training-cm.yaml
    sed -i -e "s|gcr.io/ml-pipeline/ml-pipeline-dataproc-analyze:\([a-zA-Z0-9_.-]\)\+|${DATAPROC_ANALYZE_IMAGE}|g" xgboost-training-cm.yaml
    sed -i -e "s|gcr.io/ml-pipeline/ml-pipeline-dataproc-transform:\([a-zA-Z0-9_.-]\)\+|${DATAPROC_TRANSFORM_IMAGE}|g" xgboost-training-cm.yaml
    sed -i -e "s|gcr.io/ml-pipeline/ml-pipeline-dataproc-train:\([a-zA-Z0-9_.-]\)\+|${DATAPROC_TRAIN_IMAGE}|g" xgboost-training-cm.yaml
    sed -i -e "s|gcr.io/ml-pipeline/ml-pipeline-dataproc-predict:\([a-zA-Z0-9_.-]\)\+|${DATAPROC_PREDICT_IMAGE}|g" xgboost-training-cm.yaml
    sed -i -e "s|gcr.io/ml-pipeline/ml-pipeline-local-confusion-matrix:\([a-zA-Z0-9_.-]\)\+|${LOCAL_CONFUSIONMATRIX_IMAGE}|g" xgboost-training-cm.yaml
    sed -i -e "s|gcr.io/ml-pipeline/ml-pipeline-local-roc:\([a-zA-Z0-9_.-]\)\+|${LOCAL_ROC_IMAGE}|g" xgboost-training-cm.yaml
  fi
  cd "${TEST_DIR}"
  python3 run_xgboost_test.py --input ${BASE_DIR}/samples/xgboost-spark/xgboost-training-cm.yaml --result $SAMPLE_XGBOOST_TEST_RESULT --output $SAMPLE_XGBOOST_TEST_OUTPUT --namespace ${NAMESPACE}

  echo "Copy the test results to GCS ${RESULTS_GCS_DIR}/"
  gsutil cp ${SAMPLE_XGBOOST_TEST_RESULT} ${RESULTS_GCS_DIR}/${SAMPLE_XGBOOST_TEST_RESULT}
elif [ "$TEST_NAME" == "notebook-tfx" ]; then
  SAMPLE_NOTEBOOK_TFX_TEST_RESULT=junit_SampleNotebookTFXOutput.xml
  SAMPLE_NOTEBOOK_TFX_TEST_OUTPUT=${RESULTS_GCS_DIR}

  # CMLE model name format: A name should start with a letter and contain only letters, numbers and underscores.
  DEPLOYER_MODEL=`cat /proc/sys/kernel/random/uuid`
  DEPLOYER_MODEL=Notebook_tfx_taxi_`echo ${DEPLOYER_MODEL//-/_}`

  cd ${BASE_DIR}/samples/notebooks
  export LC_ALL=C.UTF-8
  export LANG=C.UTF-8
  if [ -n "${DATAFLOW_TFT_IMAGE}" ];then
    papermill --prepare-only -p EXPERIMENT_NAME notebook-tfx-test -p OUTPUT_DIR ${RESULTS_GCS_DIR} -p PROJECT_NAME ml-pipeline-test \
      -p BASE_IMAGE ${TARGET_IMAGE_PREFIX}pusherbase:dev -p TARGET_IMAGE ${TARGET_IMAGE_PREFIX}pusher:dev -p TARGET_IMAGE_TWO ${TARGET_IMAGE_PREFIX}pusher_two:dev \
      -p KFP_PACKAGE /tmp/kfp.tar.gz -p DEPLOYER_MODEL ${DEPLOYER_MODEL}  \
      -p DATAFLOW_TFDV_IMAGE ${DATAFLOW_TFDV_IMAGE} -p DATAFLOW_TFT_IMAGE ${DATAFLOW_TFT_IMAGE} -p DATAFLOW_TFMA_IMAGE ${DATAFLOW_TFMA_IMAGE} -p DATAFLOW_TF_PREDICT_IMAGE ${DATAFLOW_PREDICT_IMAGE} \
      -p KUBEFLOW_TF_TRAINER_IMAGE ${KUBEFLOW_DNNTRAINER_IMAGE} -p KUBEFLOW_DEPLOYER_IMAGE ${KUBEFLOW_DEPLOYER_IMAGE} \
      -p TRAIN_DATA gs://ml-pipeline-dataset/sample-test/taxi-cab-classification/train50.csv -p EVAL_DATA gs://ml-pipeline-dataset/sample-test/taxi-cab-classification/eval20.csv \
      -p HIDDEN_LAYER_SIZE 10 -p STEPS 50 KubeFlow\ Pipeline\ Using\ TFX\ OSS\ Components.ipynb notebook-tfx.ipynb
  else
    papermill --prepare-only -p EXPERIMENT_NAME notebook-tfx-test -p OUTPUT_DIR ${RESULTS_GCS_DIR} -p PROJECT_NAME ml-pipeline-test \
      -p BASE_IMAGE ${TARGET_IMAGE_PREFIX}pusherbase:dev -p TARGET_IMAGE ${TARGET_IMAGE_PREFIX}pusher:dev -p TARGET_IMAGE_TWO ${TARGET_IMAGE_PREFIX}pusher_two:dev \
      -p KFP_PACKAGE /tmp/kfp.tar.gz -p DEPLOYER_MODEL ${DEPLOYER_MODEL} \
      -p TRAIN_DATA gs://ml-pipeline-dataset/sample-test/taxi-cab-classification/train50.csv -p EVAL_DATA gs://ml-pipeline-dataset/sample-test/taxi-cab-classification/eval20.csv \
      -p HIDDEN_LAYER_SIZE 10 -p STEPS 50 KubeFlow\ Pipeline\ Using\ TFX\ OSS\ Components.ipynb notebook-tfx.ipynb
  fi
  jupyter nbconvert --to python notebook-tfx.ipynb
  pip3 install tensorflow==1.8.0
  ipython notebook-tfx.py
  EXIT_CODE=$?
  cd "${TEST_DIR}"
  python3 check_notebook_results.py --experiment notebook-tfx-test --testname notebooktfx --result $SAMPLE_NOTEBOOK_TFX_TEST_RESULT --namespace ${NAMESPACE} --exit-code ${EXIT_CODE}

  echo "Copy the test results to GCS ${RESULTS_GCS_DIR}/"
  gsutil cp $SAMPLE_NOTEBOOK_TFX_TEST_RESULT ${RESULTS_GCS_DIR}/$SAMPLE_NOTEBOOK_TFX_TEST_RESULT

  #Clean CMLE models. Not needed because we cleaned them up inside notebook.
  # python3 clean_cmle_models.py --project ml-pipeline-test --model ${DEV_DEPLOYER_MODEL} --version ${MODEL_VERSION}
  # python3 clean_cmle_models.py --project ml-pipeline-test --model ${PROD_DEPLOYER_MODEL} --version ${MODEL_VERSION}
elif [ "$TEST_NAME" == "notebook-lightweight" ]; then
  SAMPLE_NOTEBOOK_LIGHTWEIGHT_TEST_RESULT=junit_SampleNotebookLightweightOutput.xml
  SAMPLE_NOTEBOOK_LIGHTWEIGHT_TEST_OUTPUT=${RESULTS_GCS_DIR}

  cd ${BASE_DIR}/samples/notebooks
  export LC_ALL=C.UTF-8
  export LANG=C.UTF-8
  papermill --prepare-only -p EXPERIMENT_NAME notebook-lightweight -p PROJECT_NAME ml-pipeline-test -p KFP_PACKAGE /tmp/kfp.tar.gz  Lightweight\ Python\ components\ -\ basics.ipynb notebook-lightweight.ipynb
  jupyter nbconvert --to python notebook-lightweight.ipynb
  pip3 install tensorflow==1.8.0
  ipython notebook-lightweight.py
  EXIT_CODE=$?
  cd "${TEST_DIR}"
  python3 check_notebook_results.py --experiment notebook-lightweight --testname notebooklightweight --result $SAMPLE_NOTEBOOK_LIGHTWEIGHT_TEST_RESULT --namespace ${NAMESPACE} --exit-code ${EXIT_CODE}

  echo "Copy the test results to GCS ${RESULTS_GCS_DIR}/"
  gsutil cp $SAMPLE_NOTEBOOK_LIGHTWEIGHT_TEST_RESULT ${RESULTS_GCS_DIR}/$SAMPLE_NOTEBOOK_LIGHTWEIGHT_TEST_RESULT
elif [ "$TEST_NAME" == "notebook-typecheck" ]; then
  SAMPLE_NOTEBOOK_TYPECHECK_TEST_RESULT=junit_SampleNotebookTypecheckOutput.xml
  SAMPLE_NOTEBOOK_TYPECHECK_TEST_OUTPUT=${RESULTS_GCS_DIR}

  cd ${BASE_DIR}/samples/notebooks
  export LC_ALL=C.UTF-8
  export LANG=C.UTF-8
  papermill --prepare-only -p KFP_PACKAGE /tmp/kfp.tar.gz  DSL\ Static\ Type\ Checking.ipynb notebook-typecheck.ipynb
  jupyter nbconvert --to python notebook-typecheck.ipynb
  ipython notebook-typecheck.py
  EXIT_CODE=$?
  cd "${TEST_DIR}"
  python3 check_notebook_results.py --testname notebooktypecheck --result $SAMPLE_NOTEBOOK_TYPECHECK_TEST_RESULT --exit-code ${EXIT_CODE}

  echo "Copy the test results to GCS ${RESULTS_GCS_DIR}/"
  gsutil cp $SAMPLE_NOTEBOOK_TYPECHECK_TEST_RESULT ${RESULTS_GCS_DIR}/$SAMPLE_NOTEBOOK_TYPECHECK_TEST_RESULT
fi
