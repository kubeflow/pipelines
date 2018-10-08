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
    [--kubeflow-launcher-image      image path to the kubeflow launcher]
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
             --kubeflow-launcher-image )        shift
                                                KUBEFLOW_LAUNCHER_IMAGE=$1
                                                ;;
             --local-confusionmatrix-image )    shift
                                                LOCAL_CONFUSIONMATRIX_IMAGE=$1
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

GITHUB_REPO=googleprivate/ml
BASE_DIR=/python/src/github.com/${GITHUB_REPO}

# Add github to SSH known host.
ssh-keygen -F github.com || ssh-keyscan github.com >>~/.ssh/known_hosts
cp ~/.ssh/github/* ~/.ssh

echo "Clone ML pipeline code in COMMIT SHA ${COMMIT_SHA}..."
git clone git@github.com:${GITHUB_REPO}.git ${BASE_DIR}
cd ${BASE_DIR}
git checkout ${COMMIT_SHA}

echo "Run the sample tests..."

# Install dsl, dsl-compiler
pip3 install ./dsl/ --upgrade
pip3 install ./dsl-compiler/ --upgrade

SAMPLE_KUBEFLOW_TEST_RESULT=junit_SampleKubeflowOutput.xml

# Compile samples
cd samples/
cd kubeflow-tf
DATAFLOW_TFT_IMAGE_FOR_SED=$(echo ${DATAFLOW_TFT_IMAGE}|sed -e "s/\//\\\\\//g"|sed -e "s/\./\\\\\./g")
DATAFLOW_PREDICT_IMAGE_FOR_SED=$(echo ${DATAFLOW_PREDICT_IMAGE}|sed -e "s/\//\\\\\//g"|sed -e "s/\./\\\\\./g")
KUBEFLOW_LAUNCHER_IMAGE_FOR_SED=$(echo ${KUBEFLOW_LAUNCHER_IMAGE}|sed -e "s/\//\\\\\//g"|sed -e "s/\./\\\\\./g")
LOCAL_CONFUSIONMATRIX_IMAGE_FOR_SED=$(echo ${LOCAL_CONFUSIONMATRIX_IMAGE}|sed -e "s/\//\\\\\//g"|sed -e "s/\./\\\\\./g")

sed -i -e "s/gcr.io\/ml-pipeline\/ml-pipeline-dataflow-tft:\([a-zA-Z0-9_.-]\)\+/${DATAFLOW_TFT_IMAGE_FOR_SED}/g" kubeflow-training-classification.py
sed -i -e "s/gcr.io\/ml-pipeline\/ml-pipeline-kubeflow-tf:\([a-zA-Z0-9_.-]\)\+/${KUBEFLOW_LAUNCHER_IMAGE_FOR_SED}/g" kubeflow-training-classification.py
sed -i -e "s/gcr.io\/ml-pipeline\/ml-pipeline-dataflow-tf-predict:\([a-zA-Z0-9_.-]\)\+/${DATAFLOW_PREDICT_IMAGE_FOR_SED}/g" kubeflow-training-classification.py
sed -i -e "s/gcr.io\/ml-pipeline\/ml-pipeline-local-confusion-matrix:\([a-zA-Z0-9_.-]\)\+/${LOCAL_CONFUSIONMATRIX_IMAGE_FOR_SED}/g" kubeflow-training-classification.py

dsl-compile --py kubeflow-training-classification.py --output kubeflow-training-classification.yaml

# Generate API Python library
cd ${BASE_DIR}/backend/api
echo "{\"packageName\": \"swagger_pipeline_upload\"}" > config.json
java -jar /swagger-codegen-cli.jar generate -l python -i swagger/pipeline.upload.swagger.json -o ./swagger_pipeline_upload -c config.json
pip3 install ./swagger_pipeline_upload/ --upgrade
echo "{\"packageName\": \"swagger_pipeline\"}" > config.json
java -jar /swagger-codegen-cli.jar generate -l python -i swagger/pipeline.swagger.json -o ./swagger_pipeline -c config.json
pip3 install ./swagger_pipeline/ --upgrade
echo "{\"packageName\": \"swagger_run\"}" > config.json
java -jar /swagger-codegen-cli.jar generate -l python -i swagger/run.swagger.json -o ./swagger_run -c config.json
pip3 install ./swagger_run/ --upgrade
echo "{\"packageName\": \"swagger_job\"}" > config.json
java -jar /swagger-codegen-cli.jar generate -l python -i swagger/job.swagger.json -o ./swagger_job -c config.json
pip3 install ./swagger_job/ --upgrade
rm -rf config.json swagger_pipeline_upload swagger_pipeline swagger_run swagger_job

# Run the tests
#TODO: update the job output directory
cd /
python3 run_kubeflow_test.py --input ${BASE_DIR}/samples/kubeflow-tf/kubeflow-training-classification.yaml --output $SAMPLE_KUBEFLOW_TEST_RESULT

echo "Copy the test results to GCS ${RESULTS_GCS_DIR}/"
gsutil cp ${SAMPLE_KUBEFLOW_TEST_RESULT} ${RESULTS_GCS_DIR}/${SAMPLE_KUBEFLOW_TEST_RESULT}
