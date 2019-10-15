#!/bin/bash -e
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


# The scripts creates a the KF Pipelines API python package.
# Requirements: jq and Java
# To install the prerequisites run the following:
#
# # Debian / Ubuntu:
# sudo apt-get install --no-install-recommends -y -q default-jdk jq
#
# # OS X
# brew tap caskroom/cask
# brew cask install caskroom/versions/java8
# brew install jq

VERSION="$1"

if [ -z "$VERSION" ]; then
    echo "Usage: build_kfp_server_api_python_package.sh <version>"
    exit 1
fi

codegen_file=/tmp/swagger-codegen-cli.jar
codegen_uri=http://central.maven.org/maven2/io/swagger/swagger-codegen-cli/2.4.7/swagger-codegen-cli-2.4.7.jar
if ! [ -f "$codegen_file" ]; then
    wget --no-verbose "$codegen_uri" -O "$codegen_file"
fi

pushd "$(dirname "$0")"

DIR=$(mktemp -d)

swagger_file=$(mktemp)

echo "Merging all Swagger API definitions to $swagger_file."
jq -s '
    reduce .[] as $item ({}; . * $item) |
    .info.title = "KF Pipelines API" |
    .info.description = "Generated python client for the KF Pipelines server API"
' ./swagger/{run,job,pipeline,experiment,pipeline.upload}.swagger.json > "$swagger_file"

echo "Generating python code from swagger json in $DIR."
java -jar "$codegen_file" generate -l python -i "$swagger_file" -o "$DIR" -c <(echo '{
    "packageName": "kfp_server_api",
    "projectName": "kfp-server-api",
    "packageVersion": "'"$VERSION"'",
    "packageUrl": "https://github.com/kubeflow/pipelines"
}')

echo "Building the python package in $DIR."
pushd "$DIR"
python3 setup.py --quiet sdist
popd

echo "Run the following commands to update the package on PyPI"
echo "python3 -m pip install twine"
echo "python3 -m twine upload --username kubeflow-pipelines $DIR/dist/*"

popd
