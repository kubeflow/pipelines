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

# Download and install protoc binary (from your branch)
PROTOC_ZIP=protoc-3.19.6-linux-x86_64.zip
curl -OL https://github.com/protocolbuffers/protobuf/releases/download/v3.19.6/$PROTOC_ZIP
sudo unzip -o $PROTOC_ZIP -d /usr/local bin/protoc
sudo unzip -o $PROTOC_ZIP -d /usr/local 'include/*'
rm -f $PROTOC_ZIP

protoc --version

if [ "${SETUP_ENV}" = "true" ]; then
  # Create a virtual environment and activate it
  python3 -m venv venv
  source venv/bin/activate

  # Master branch approach for pip installations
  python3 -m pip install --upgrade pip
  python3 -m pip install -r sdk/python/requirements.txt 
  python3 -m pip install -r sdk/python/requirements-dev.txt
  python3 -m pip install setuptools
  python3 -m pip install wheel==0.42.0
  python3 -m pip install coveralls==4.0.1
  python3 -m pip install --upgrade protobuf
  python3 -m pip install sdk/python

  # API regeneration steps (from your branch)
  #install kfp api
  pushd api && make clean-python python && popd
  python3 -m pip install api/v2alpha1/python

  # regenerate the kfp-pipeline-spec
  cd api/
  make clean python
  cd ..
  # install the local kfp-pipeline-spec
  python3 -m pip install -I api/v2alpha1/python
fi

pytest sdk/python/kfp --cov=kfp

if [ "${SETUP_ENV}" = "true" ]; then
  # Deactivate the virtual environment
  deactivate
fi
