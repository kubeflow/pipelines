#!/bin/bash -ex
# Copyright 2020 Kubeflow Pipelines contributors
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

source_root=$(pwd)
SETUP_ENV="${SETUP_ENV:-true}"

if [ "${SETUP_ENV}" = "true" ]; then
  # Create a virtual environment and activate it
  python3 -m venv venv
  source venv/bin/activate

  python3 -m pip install --upgrade pip
  python3 -m pip install -r sdk/python/requirements.txt 
  python3 -m pip install -r sdk/python/requirements-dev.txt
  python3 -m pip install setuptools
  python3 -m pip install wheel==0.42.0
  python3 -m pip install build
  python3 -m pip install pytest
  python3 -m pip install pytest-cov
  python3 -m pip install --upgrade protobuf
  # Install local kfp-server-api first to avoid PyPI failures for unreleased versions (#12906).
  python3 -m pip install backend/api/v2beta1/python_http_client
  python3 -m pip install sdk/python

  # regenerate the kfp-pipeline-spec
  cd api/
  make clean python
  cd ..
  # install the local kfp-pipeline-spec
  python3 -m pip install -I api/v2alpha1/python
fi

# Build a local kfp wheel to use instead of pulling from PyPI (#12906).
pushd sdk/python
rm -rf dist/
python3 -m build .
kfp_wheel=$(find dist -maxdepth 1 -name "*.whl" -type f | head -n 1)
if [[ -z "${kfp_wheel}" ]]; then
  echo "Error: No wheel file found in dist/ after build." >&2
  exit 1
fi
popd
export KFP_PACKAGE_PATH="${source_root}/sdk/python/${kfp_wheel}"

python -m pytest sdk/python/test/client/ -v -s -m client --cov=kfp

if [ "${SETUP_ENV}" = "true" ]; then
  # Deactivate the virtual environment
  deactivate
fi
