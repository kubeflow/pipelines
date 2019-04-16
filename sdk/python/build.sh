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


# The scripts creates a Pipelines client python package.
#
# Usage:
#   ./build.sh [output_dir]
#
# Setup:
#   apt-get update -y
#   apt-get install --no-install-recommends -y -q default-jdk
#   wget http://central.maven.org/maven2/io/swagger/swagger-codegen-cli/2.4.1/swagger-codegen-cli-2.4.1.jar -O /tmp/swagger-codegen-cli.jar
#   # where /tmp/swagger-codegen-cli.jar in the wget line above is the same as the variable SWAGGER_CODEGEN_CLI_FILE defined below


set -e
set -o nounset
set -o pipefail

usage()
{
    echo "Usage: build.sh
     ( [-o, --output OUTPUT_PKG_FILE] | [-i, --install ] )
     [-h help]

    The -o flag and -i flag should not both be set at the same time.  No flags is equivalent to -o ./kfp.tar.gz

    Examples:
    build.sh                               # build and output a package, kfp.tar.gz, in this directory
    build.sh --output ~/my_kfp_pgk.tar.gz  # build and output a package, my_kfp_pkg.tar.gz in your home dir
    build.sh --install                     # install an editable copy of this kfp package in this directory
    "
}

SWAGGER_CODEGEN_CLI_FILE=/tmp/swagger-codegen-cli.jar
if [[ ! -f ${SWAGGER_CODEGEN_CLI_FILE} ]]; then
    echo "You don't have swagger codegen installed.  See the comments on this file for install instructions."
fi


DO_INSTALL=false
while (($#)); do
    arg=$1; shift
    case "$arg" in
        -o | --output )
                                OUTPUT_FILE=$1; shift
                                ;;
        -i | --install )
                                DO_INSTALL=true
                                ;;
        -h | --help )           usage
                                exit
                                ;;
        * )                     usage
                                exit 1
    esac
done

if [ ! -z ${OUTPUT_FILE+x} ] && [ "$DO_INSTALL" = true ] ; then
  usage;
  exit 1;
fi

TMP_DIR=$(mktemp -d)
# the directory containing this script
THIS_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

get_abs_filename() {
  # $1 : relative filename
  echo "$(cd "$(dirname "$1")" && pwd)/$(basename "$1")"
}

OUTPUT_FILE=${OUTPUT_FILE:-kfp.tar.gz}
OUTPUT_FILE=$(get_abs_filename "$OUTPUT_FILE")

if $DO_INSTALL ; then

  for SWAGGER_PACKAGE in kfp_experiment kfp_run kfp_pipeline kfp_uploadpipeline kfp_job
  do
      TARGET_DIR=${THIS_DIR}/${SWAGGER_PACKAGE}
      if [ -d  $TARGET_DIR ]; then
          echo "$TARGET_DIR already exists.  Aborting."
          exit 1
      fi
  done
fi


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

if $DO_INSTALL ; then
  echo "Installing ..."
  # copy the generated python packages back into this directory
  for SWAGGER_PACKAGE in kfp_experiment kfp_run kfp_pipeline kfp_uploadpipeline kfp_job
  do
      TARGET_DIR=${THIS_DIR}/${SWAGGER_PACKAGE}
      if [ -d  $TARGET_DIR ]; then
          echo "$TARGET_DIR already exists.  Aborting."
          exit 1
      fi
      mv ${TMP_DIR}/${SWAGGER_PACKAGE} ${THIS_DIR}
  done

  # install the full Python package, which includes both the raw KFP code and the generated swagger packages, in editable mode
  pip3 install --editable $THIS_DIR

else
  echo "Building ..."
  # Merge generated code with the rest code (setup.py, seira_client, etc).
  cp -r kfp $TMP_DIR
  cp ./setup.py $TMP_DIR

  # Build tarball package.
  cd $TMP_DIR
  python setup.py sdist --format=gztar
  cp $TMP_DIR/dist/*.tar.gz "$OUTPUT_FILE"
  cd -
fi

rm -rf $TMP_DIR

