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


# The scripts creates a Pipelines client python package.
#
# Usage:
#   ./build.sh [output_dir]
#
# Setup:
#   apt-get update -y
#   apt-get install --no-install-recommends -y -q default-jdk
#   wget http://central.maven.org/maven2/io/swagger/swagger-codegen-cli/2.3.1/swagger-codegen-cli-2.3.1.jar -O /tmp/swagger-codegen-cli.jar

get_abs_filename() {
  # $1 : relative filename
  echo "$(cd "$(dirname "$1")" && pwd)/$(basename "$1")"
}

target_archive_file=${1:-kfp.tar.gz}
target_archive_file=$(get_abs_filename "$target_archive_file")

DIR=$(mktemp -d)

# Generate python code from swagger json.
echo "{\"packageName\": \"kfp_experiment\"}" > /tmp/config.json
java -jar /tmp/swagger-codegen-cli.jar generate -l python -i ../../backend/api/swagger/experiment.swagger.json -o $DIR -c /tmp/config.json
echo "{\"packageName\": \"kfp_run\"}" > /tmp/config.json
java -jar /tmp/swagger-codegen-cli.jar generate -l python -i ../../backend/api/swagger/run.swagger.json -o $DIR -c /tmp/config.json
rm /tmp/config.json

# Merge generated code with the rest code (setup.py, seira_client, etc).
cp -r kfp $DIR
cp ./setup.py $DIR

# Build tarball package.
cd $DIR
python setup.py sdist --format=gztar
cp $DIR/dist/*.tar.gz "$target_archive_file"
rm -rf $DIR
