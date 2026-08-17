#!/bin/bash -e
#
# Copyright 2018-2021 The Kubeflow Authors
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
DIR="$CURRENT_DIR/$API_VERSION/python_http_client"
swagger_file="$CURRENT_DIR/$API_VERSION/swagger/kfp_api_single_file.swagger.json"

echo "Removing old content in DIR first."
rm -rf "$DIR"

echo "Generating python code from swagger json in $DIR."
java -jar "$codegen_file" generate -g python -t "$CURRENT_DIR/$API_VERSION/python_http_client_template" -i "$swagger_file" -o "$DIR" -c <(echo '{
    "packageName": "'"kfp_server_api"'",
    "packageVersion": "'"$VERSION"'",
    "packageUrl": "https://github.com/kubeflow/pipelines"
}')

echo "Removing unnecessary GitLab and TravisCI generated files"
rm $CURRENT_DIR/$API_VERSION/python_http_client/.gitlab-ci.yml
rm $CURRENT_DIR/$API_VERSION/python_http_client/.travis.yml

# openapi-generator can emit a phantom GooglerpcStatus import alongside the
# real GoogleRpcStatus model. Drop the broken import so the package is
# importable without a missing googlerpc_status module.
CLIENT_ROOT="$CURRENT_DIR/$API_VERSION/python_http_client"
python3 - "$CLIENT_ROOT" <<'PY'
from pathlib import Path
import sys

root = Path(sys.argv[1])
bad_import = "from kfp_server_api.models.googlerpc_status import GooglerpcStatus\n"
for path in [
    root / "kfp_server_api" / "__init__.py",
    root / "kfp_server_api" / "models" / "__init__.py",
]:
    text = path.read_text()
    if bad_import in text:
        path.write_text(text.replace(bad_import, ""))
readme = root / "README.md"
if readme.exists():
    readme.write_text(
        readme.read_text().replace(
            " - [GooglerpcStatus](docs/GooglerpcStatus.md)\n",
            "",
        )
    )
PY

echo "Copying LICENSE to $DIR"
cp "$CURRENT_DIR/../../LICENSE" "$DIR"

echo "Building the python package in $DIR."
pushd "$DIR"
python3 setup.py --quiet sdist
popd

echo "Run the following commands to update the package on PyPI"
echo "python3 -m pip install twine"
echo "python3 -m twine upload --username kubeflow-pipelines $DIR/dist/*"

echo "Please also push local changes to github.com/kubeflow/pipelines"

popd
