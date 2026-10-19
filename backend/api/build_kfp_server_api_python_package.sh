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

API_VERSION=${API_VERSION:-v2beta1}
if [[ "$API_VERSION" != "v2beta1" ]]; then
    echo "Only the v2beta1 API is supported." >&2
    exit 1
fi

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
package_name=kfp.server_api
# Python modules share the SDK release version, not the backend version.
VERSION="$(python3 - "$REPO_ROOT/sdk/python/kfp/version.py" <<'PY'
import runpy
import sys

print(runpy.run_path(sys.argv[1])["__version__"])
PY
)"
if [ -z "$VERSION" ]; then
    echo "ERROR: the package version must not be empty"
    exit 1
fi

codegen_file=/tmp/openapi-generator-cli.jar
# Browse all versions in: https://repo1.maven.org/maven2/org/openapitools/openapi-generator-cli/
codegen_uri="https://repo1.maven.org/maven2/org/openapitools/openapi-generator-cli/4.3.1/openapi-generator-cli-4.3.1.jar"
if ! [ -f "$codegen_file" ]; then
    curl --fail --location --retry 3 "$codegen_uri" -o "$codegen_file"
fi

pushd "$(dirname "$0")"

CURRENT_DIR="$(pwd)"
DIR="$CURRENT_DIR/$API_VERSION/python_http_client"
swagger_file="$CURRENT_DIR/$API_VERSION/swagger/kfp_api_single_file.swagger.json"

echo "Removing old content in DIR first."
rm -rf "$DIR"

# Generated test stubs have no assertions and instantiate invalid empty models.
# Real API coverage lives in SDK tests and python_http_client_smoke.
generator_options=(--global-property apiTests=false,modelTests=false)

echo "Generating python code from swagger json in $DIR."
java -jar "$codegen_file" generate -g python -t "$CURRENT_DIR/$API_VERSION/python_http_client_template" -i "$swagger_file" -o "$DIR" \
    "${generator_options[@]}" -c <(echo '{
    "packageName": "'"$package_name"'",
    "packageVersion": "'"$VERSION"'",
    "packageUrl": "https://github.com/kubeflow/pipelines"
}')


echo "Removing unnecessary GitLab and TravisCI generated files"
rm -f $CURRENT_DIR/$API_VERSION/python_http_client/.gitlab-ci.yml
rm -f $CURRENT_DIR/$API_VERSION/python_http_client/.travis.yml

# openapi-generator can emit a phantom GooglerpcStatus import alongside the
# real GoogleRpcStatus model. Drop the broken import so the package is
# importable without a missing googlerpc_status module.
CLIENT_ROOT="$CURRENT_DIR/$API_VERSION/python_http_client"
python3 - "$CLIENT_ROOT" "$package_name" <<'PY'
from pathlib import Path
import sys

root = Path(sys.argv[1])
package_name = sys.argv[2]
package_root = root.joinpath(*package_name.split("."))
bad_import = f"from {package_name}.models.googlerpc_status import GooglerpcStatus\n"
for path in [
    package_root / "__init__.py",
    package_root / "models" / "__init__.py",
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
if package_name == "kfp.server_api":
    import re
    initializer = package_root / "__init__.py"
    initializer.write_text(re.sub(
        r'^__version__ = .*$',
        'from kfp.version import __version__',
        initializer.read_text(),
        flags=re.MULTILINE,
    ))
    sdk_installation = """## Installation & Usage

This client is included in the unified `kfp` distribution and requires
Python 3.11 or later:

```sh
python -m pip install kfp
```

From a source checkout, install `sdk/python` from the repository root, not this
generated documentation directory. See the [SDK installation and migration
instructions](../../../../sdk/python/README.md).

```python
from kfp import server_api
```

"""
    text, replacements = re.subn(
        r'## Requirements\..*?(?=## Getting Started)',
        sdk_installation,
        readme.read_text(),
        count=1,
        flags=re.DOTALL,
    )
    if replacements != 1:
        raise RuntimeError("Generated README is missing its installation section")
    readme.write_text(text.replace("# kfp.server-api\n", "# kfp.server_api\n", 1))
PY

echo "Copying LICENSE to $DIR"
cp "$CURRENT_DIR/../../LICENSE" "$DIR"

# Generate only SDK-owned modules, never a second distribution.
SDK_CLIENT="$REPO_ROOT/sdk/python/kfp/server_api"
rm -rf "$SDK_CLIENT"
mv "$DIR/kfp/server_api" "$SDK_CLIENT"
rm -rf "$DIR/kfp" "$DIR/test" "$DIR/.openapi-generator"
rm -f "$DIR/setup.py" "$DIR/setup.cfg" "$DIR/tox.ini" \
    "$DIR/requirements.txt" "$DIR/test-requirements.txt" \
    "$DIR/git_push.sh" "$DIR/.gitignore" "$DIR/.openapi-generator-ignore"

popd
