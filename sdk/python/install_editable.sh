#!/bin/bash -ex
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


# This script installs the KFP Python SDK in editable mode.  It can be considered a sister-script to build.sh, but
# where build.sh creates a .tar.gz package which can be installed as a site package or uploaded for release, this
# script creates and installs a version of the KFP Python SDK on the local machine which is suitable for building
# new functionality into the KFP SDK itself.
#
# Usage:
#   ./build.sh [output_dir]
#
# Setup:
#   apt-get update -y
#   apt-get install --no-install-recommends -y -q default-jdk
#   wget http://central.maven.org/maven2/io/swagger/swagger-codegen-cli/2.4.1/swagger-codegen-cli-2.4.1.jar -O /tmp/swagger-codegen-cli.jar
#   # where /tmp/swagger-codegen-cli.jar in the wget line above is the same as the variable SWAGGER_CODEGEN_CLI_FILE defined below

SWAGGER_CODEGEN_CLI_FILE=/tmp/swagger-codegen-cli.jar
if [[ ! -f ${SWAGGER_CODEGEN_CLI_FILE} ]]; then
    echo "you don't have swagger codegen installed.  see the comments on this file for install instructions"
fi

TMP_DIR=$(mktemp -d)
# gets the directory containing this script
THIS_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

# Generate python code for the swagger-defined low-level api clients
echo "{\"packageName\": \"kfp_experiment\"}" > /tmp/config.json
java -jar /tmp/swagger-codegen-cli.jar generate -l python -i ../../backend/api/swagger/experiment.swagger.json -o $TMP_DIR -c /tmp/config.json
echo "{\"packageName\": \"kfp_run\"}" > /tmp/config.json
java -jar /tmp/swagger-codegen-cli.jar generate -l python -i ../../backend/api/swagger/run.swagger.json -o $TMP_DIR -c /tmp/config.json
echo "{\"packageName\": \"kfp_pipeline\"}" > /tmp/config.json
java -jar /tmp/swagger-codegen-cli.jar generate -l python -i ../../backend/api/swagger/pipeline.swagger.json -o $TMP_DIR -c /tmp/config.json
echo "{\"packageName\": \"kfp_uploadpipeline\"}" > /tmp/config.json
java -jar /tmp/swagger-codegen-cli.jar generate -l python -i ../../backend/api/swagger/pipeline.upload.swagger.json -o $TMP_DIR -c /tmp/config.json
echo "{\"packageName\": \"kfp_job\"}" > /tmp/config.json
java -jar /tmp/swagger-codegen-cli.jar generate -l python -i ../../backend/api/swagger/job.swagger.json -o $TMP_DIR -c /tmp/config.json
rm /tmp/config.json

# copy the generated python packages back into this directory
for SWAGGER_PACKAGE in kfp_experiment kfp_run kfp_pipeline kfp_uploadpipeline kfp_job
do
    mv ${TMP_DIR}/${SWAGGER_PACKAGE} ${THIS_DIR}
done

# install the full Python package, which includes both the raw KFP code and the generated swagger packages, in editable mode
pip3 install --editable $THIS_DIR

# clean up temp directory
rm -rf $TMP_DIR