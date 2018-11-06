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
    echo "usage: run_kubeflow_test.sh
    [--results-gcs-dir              GCS directory for the test results]
    [--commit_sha                   commit SHA to pull code from]
    [--dataflow-tft-image           image path to the dataflow tft]
    [--dataflow-predict-image       image path to the dataflow predict]
    [--dataflow-tfma-image          image path to the dataflow tfma]
    [--dataflow-tfdv-image          image path to the dataflow tfdv]
    [--kubeflow-dnntrainer-image    image path to the kubeflow dnntrainer]
    [--kubeflow-deployer-image      image path to the kubeflow deployer]
    [--local-confusionmatrix-image  image path to the confusion matrix]
    [-h help]"
}

while [ "$1" != "" ]; do
    case $1 in
             --results-gcs-dir )                shift
                                                RESULTS_GCS_DIR=$1
                                                ;;
             --commit_sha )                     shift
                                                COMMIT_SHA=$1
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
             --kubeflow-dnntrainer-image )      shift
                                                KUBEFLOW_DNNTRAINER_IMAGE=$1
                                                ;;
             --kubeflow-deployer-image )        shift
                                                KUBEFLOW_DEPLOYER_IMAGE=$1
                                                ;;
             --local-confusionmatrix-image )    shift
                                                LOCAL_CONFUSIONMATRIX_IMAGE=$1
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

GITHUB_REPO=kubeflow/pipelines
BASE_DIR=/python/src/github.com/${GITHUB_REPO}

# Add github to SSH known host.
ssh-keygen -F github.com || ssh-keyscan github.com >>~/.ssh/known_hosts
cp ~/.ssh/github/* ~/.ssh

echo "Clone ML pipeline code in COMMIT SHA ${COMMIT_SHA}..."
git clone git@github.com:${GITHUB_REPO}.git ${BASE_DIR}
cd ${BASE_DIR}
git checkout ${COMMIT_SHA}

# Install argo
echo "install argo"
ARGO_VERSION=v2.2.0
mkdir -p ~/bin/
export PATH=~/bin/:$PATH
curl -sSL -o ~/bin/argo https://github.com/argoproj/argo/releases/download/$ARGO_VERSION/argo-linux-amd64
chmod +x ~/bin/argo

echo "Run the sample tests..."

# Generate Python package
cd ./sdk/python
./build.sh /tmp/kfp.tar.gz

# Install python client, including DSL compiler.
pip3 install /tmp/kfp.tar.gz

# Run the tests
if [ "$TEST_NAME" == 'tf-training' ]; then
  SAMPLE_KUBEFLOW_TEST_RESULT=junit_SampleKubeflowOutput.xml
  SAMPLE_KUBEFLOW_TEST_OUTPUT=${RESULTS_GCS_DIR}

  # Compile samples
  cd ${BASE_DIR}/samples/kubeflow-tf
  DATAFLOW_TFT_IMAGE_FOR_SED=$(echo ${DATAFLOW_TFT_IMAGE}|sed -e "s/\//\\\\\//g"|sed -e "s/\./\\\\\./g")
  DATAFLOW_PREDICT_IMAGE_FOR_SED=$(echo ${DATAFLOW_PREDICT_IMAGE}|sed -e "s/\//\\\\\//g"|sed -e "s/\./\\\\\./g")
  KUBEFLOW_DNNTRAINER_IMAGE_FOR_SED=$(echo ${KUBEFLOW_DNNTRAINER_IMAGE}|sed -e "s/\//\\\\\//g"|sed -e "s/\./\\\\\./g")
  LOCAL_CONFUSIONMATRIX_IMAGE_FOR_SED=$(echo ${LOCAL_CONFUSIONMATRIX_IMAGE}|sed -e "s/\//\\\\\//g"|sed -e "s/\./\\\\\./g")

  sed -i -e "s/gcr.io\/ml-pipeline\/ml-pipeline-dataflow-tft:\([a-zA-Z0-9_.-]\)\+/${DATAFLOW_TFT_IMAGE_FOR_SED}/g" kubeflow-training-classification.py
  sed -i -e "s/gcr.io\/ml-pipeline\/ml-pipeline-kubeflow-tf-trainer:\([a-zA-Z0-9_.-]\)\+/${KUBEFLOW_DNNTRAINER_IMAGE_FOR_SED}/g" kubeflow-training-classification.py
  sed -i -e "s/gcr.io\/ml-pipeline\/ml-pipeline-dataflow-tf-predict:\([a-zA-Z0-9_.-]\)\+/${DATAFLOW_PREDICT_IMAGE_FOR_SED}/g" kubeflow-training-classification.py
  sed -i -e "s/gcr.io\/ml-pipeline\/ml-pipeline-local-confusion-matrix:\([a-zA-Z0-9_.-]\)\+/${LOCAL_CONFUSIONMATRIX_IMAGE_FOR_SED}/g" kubeflow-training-classification.py

  dsl-compile --py kubeflow-training-classification.py --output kubeflow-training-classification.tar.gz

  cd /
  python3 run_kubeflow_test.py --input ${BASE_DIR}/samples/kubeflow-tf/kubeflow-training-classification.tar.gz --result $SAMPLE_KUBEFLOW_TEST_RESULT --output $SAMPLE_KUBEFLOW_TEST_OUTPUT

  echo "Copy the test results to GCS ${RESULTS_GCS_DIR}/"
  gsutil cp ${SAMPLE_KUBEFLOW_TEST_RESULT} ${RESULTS_GCS_DIR}/${SAMPLE_KUBEFLOW_TEST_RESULT}
elif [ "$TEST_NAME" == "tfx" ]; then
  SAMPLE_TFX_TEST_RESULT=junit_SampleTFXOutput.xml
  SAMPLE_TFX_TEST_OUTPUT=${RESULTS_GCS_DIR}

  # Compile samples
  cd ${BASE_DIR}/samples/tfx
  DATAFLOW_TFT_IMAGE_FOR_SED=$(echo ${DATAFLOW_TFT_IMAGE}|sed -e "s/\//\\\\\//g"|sed -e "s/\./\\\\\./g")
  DATAFLOW_PREDICT_IMAGE_FOR_SED=$(echo ${DATAFLOW_PREDICT_IMAGE}|sed -e "s/\//\\\\\//g"|sed -e "s/\./\\\\\./g")
  DATAFLOW_TFDV_IMAGE_FOR_SED=$(echo ${DATAFLOW_TFDV_IMAGE}|sed -e "s/\//\\\\\//g"|sed -e "s/\./\\\\\./g")
  DATAFLOW_TFMA_IMAGE_FOR_SED=$(echo ${DATAFLOW_TFMA_IMAGE}|sed -e "s/\//\\\\\//g"|sed -e "s/\./\\\\\./g")
  KUBEFLOW_DNNTRAINER_IMAGE_FOR_SED=$(echo ${KUBEFLOW_DNNTRAINER_IMAGE}|sed -e "s/\//\\\\\//g"|sed -e "s/\./\\\\\./g")
  KUBEFLOW_DEPLOYER_IMAGE_FOR_SED=$(echo ${KUBEFLOW_DEPLOYER_IMAGE}|sed -e "s/\//\\\\\//g"|sed -e "s/\./\\\\\./g")

  sed -i -e "s/gcr.io\/ml-pipeline\/ml-pipeline-dataflow-tft:\([a-zA-Z0-9_.-]\)\+/${DATAFLOW_TFT_IMAGE_FOR_SED}/g" taxi-cab-classification-pipeline.py
  sed -i -e "s/gcr.io\/ml-pipeline\/ml-pipeline-dataflow-tf-predict:\([a-zA-Z0-9_.-]\)\+/${DATAFLOW_PREDICT_IMAGE_FOR_SED}/g" taxi-cab-classification-pipeline.py
  sed -i -e "s/gcr.io\/ml-pipeline\/ml-pipeline-dataflow-tfdv:\([a-zA-Z0-9_.-]\)\+/${DATAFLOW_TFDV_IMAGE_FOR_SED}/g" taxi-cab-classification-pipeline.py
  sed -i -e "s/gcr.io\/ml-pipeline\/ml-pipeline-dataflow-tfma:\([a-zA-Z0-9_.-]\)\+/${DATAFLOW_TFMA_IMAGE_FOR_SED}/g" taxi-cab-classification-pipeline.py
  sed -i -e "s/gcr.io\/ml-pipeline\/ml-pipeline-kubeflow-tf-trainer:\([a-zA-Z0-9_.-]\)\+/${KUBEFLOW_DNNTRAINER_IMAGE_FOR_SED}/g" taxi-cab-classification-pipeline.py
  sed -i -e "s/gcr.io\/ml-pipeline\/ml-pipeline-kubeflow-deployer:\([a-zA-Z0-9_.-]\)\+/${KUBEFLOW_DEPLOYER_IMAGE_FOR_SED}/g" taxi-cab-classification-pipeline.py

  dsl-compile --py taxi-cab-classification-pipeline.py --output taxi-cab-classification-pipeline.tar.gz

  cd /
  python3 run_tfx_test.py --input ${BASE_DIR}/samples/tfx/taxi-cab-classification-pipeline.tar.gz --result $SAMPLE_TFX_TEST_RESULT --output $SAMPLE_TFX_TEST_OUTPUT

  echo "Copy the test results to GCS ${RESULTS_GCS_DIR}/"
  gsutil cp ${SAMPLE_TFX_TEST_RESULT} ${RESULTS_GCS_DIR}/${SAMPLE_TFX_TEST_RESULT}
elif [ "$TEST_NAME" == "sequential" ]; then
  SAMPLE_SEQUENTIAL_TEST_RESULT=junit_SampleSequentialOutput.xml
  SAMPLE_SEQUENTIAL_TEST_OUTPUT=${RESULTS_GCS_DIR}

  # Compile samples
  cd ${BASE_DIR}/samples/basic
  dsl-compile --py sequential.py --output sequential.tar.gz

  cd /
  python3 run_sequential_test.py --input ${BASE_DIR}/samples/basic/sequential.tar.gz --result SAMPLE_SEQUENTIAL_TEST_RESULT --output SAMPLE_SEQUENTIAL_TEST_OUTPUT
elif [ "$TEST_NAME" == "condition" ]; then
  SAMPLE_CONDITION_TEST_RESULT=junit_SampleConditionOutput.xml
  SAMPLE_CONDITION_TEST_OUTPUT=${RESULTS_GCS_DIR}

  # Compile samples
  cd ${BASE_DIR}/samples/basic
  dsl-compile --py condition.py --output condition.tar.gz

  cd /
  python3 run_condition_test.py --input ${BASE_DIR}/samples/basic/condition.tar.gz --result SAMPLE_CONDITION_TEST_RESULT --output SAMPLE_CONDITION_TEST_OUTPUT

fi
