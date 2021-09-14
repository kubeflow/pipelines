#!/bin/bash -e
# Copyright 2021 Google LLC
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


CONDA_ENV="${1:-build-yaml}"

is_conda_env() {
  check_env=$(conda env list | grep "${CONDA_ENV}\s")
  if [[ -n "$check_env" ]] ; then
    echo 1
  fi
}

is_conda_env_active() {
  check_env_active=$(conda env list | grep "${CONDA_ENV}\s*\*")
  if [[ -n "$check_env_active" ]] ; then
    echo 1
  fi
}

if [[ ! $(is_conda_env) ]]; then
  conda create -y -n "${CONDA_ENV}" --no-default-packages
  conda activate "${CONDA_ENV}"
  conda install -y pip
fi

if [[ ! $(is_conda_env_active) ]]; then
  conda activate "${CONDA_ENV}"
fi

pip install -r requirements.txt

python ./src/notebook_executor.py