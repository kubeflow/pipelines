#!/bin/bash -e
#
# Copyright 2018-2020 Google LLC
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

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null && pwd)"
REPO_ROOT="$DIR/../.."
VERSION="$(cat $REPO_ROOT/VERSION)"
if [ -z "$VERSION" ]; then
    echo "ERROR: $REPO_ROOT/VERSION is empty"
    exit 1
fi

codegen_file=/tmp/openapi-generator-cli.jar
# Browse all versions in: https://repo1.maven.org/maven2/org/openapitools/openapi-generator-cli/
codegen_uri="https://repo1.maven.org/maven2/org/openapitools/openapi-generator-cli/4.3.1/openapi-generator-cli-4.3.1.jar"
if ! [ -f "$codegen_file" ]; then
    curl -L "$codegen_uri" -o "$codegen_file"
fi

pushd "$(dirname "$0")"

CURRENT_DIR="$(pwd)"
DIR="$CURRENT_DIR/python_http_client"
swagger_file="$CURRENT_DIR/swagger/kfp_api_single_file.swagger.json"

echo "Removing old content in DIR first."
rm -rf "$DIR"

echo "Generating python code from swagger json in $DIR."
java -jar "$codegen_file" generate -g python -t "$CURRENT_DIR/python_http_client_template" -i "$swagger_file" -o "$DIR" -c <(echo '{
    "packageName": "kfp_server_api",
    "packageVersion": "'"$VERSION"'",
    "packageUrl": "https://github.com/kubeflow/pipelines"
}')

echo "Copying LICENSE to $DIR"
cp "$CURRENT_DIR/../../LICENSE" "$DIR"

echo "Building the python package in $DIR."
pushd "$DIR"
python3 setup.py --quiet sdist
popd

echo "Adding license header for generated python files in $DIR."
go get -u github.com/google/addlicense
addlicense "$DIR"

echo "Run the following commands to update the package on PyPI"
echo "python3 -m pip install twine"
echo "python3 -m twine upload --username kubeflow-pipelines $DIR/dist/*"

echo "Please also push local changes to github.com/kubeflow/pipelines"

popd
