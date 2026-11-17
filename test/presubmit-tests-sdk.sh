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

set -eo pipefail

source_root=$(pwd)
SETUP_ENV="${SETUP_ENV:-true}"
JUNIT_XML="${JUNIT_XML:-sdk-regression.xml}"
PYTEST_LOG="${PYTEST_LOG:-sdk-regression.log}"
PYTEST_PARALLEL_WORKERS="${PYTEST_PARALLEL_WORKERS:-2}"

if [ "${SETUP_ENV}" = "true" ]; then
  # Create a virtual environment and activate it
  python3 -m venv venv
  source venv/bin/activate

  python3 -m pip install --upgrade pip
  python3 -m pip install -r sdk/python/requirements.txt
  python3 -m pip install -r sdk/python/requirements-dev.txt
  python3 -m pip install setuptools
  python3 -m pip install wheel==0.42.0
  python3 -m pip install pytest-cov
  python3 -m pip install pytest-xdist
  python3 -m pip install pytest
  python3 -m pip install google_cloud_pipeline_components
  python3 -m pip install docker
  python3 -m pip install --upgrade protobuf
  python3 -m pip install sdk/python

  # regenerate the kfp-pipeline-spec
  cd api/
  make clean python
  cd ..
  # install the local kfp-pipeline-spec
  python3 -m pip install -I api/v2alpha1/python
fi

# Build the exact checkout once; every component still installs the wheel in
# its own environment. Docker tests mount this file read-only.
runtime_dist_dir=$(mktemp -d)
trap 'rm -rf "$runtime_dist_dir"' EXIT
python -m pip wheel --no-deps --wheel-dir "$runtime_dist_dir" ./sdk/python
runtime_wheels=("$runtime_dist_dir"/kfp-*.whl)
if [[ ${#runtime_wheels[@]} -ne 1 || ! -f "${runtime_wheels[0]}" ]]; then
  echo "Expected exactly one freshly built kfp wheel in $runtime_dist_dir" >&2
  exit 1
fi
export KFP_PACKAGE_PATH="${runtime_wheels[0]}"

python -m pytest sdk/python/test -v -m regression --cov=kfp -n "${PYTEST_PARALLEL_WORKERS}" --junitxml="${JUNIT_XML}" --durations=30 --durations-min=0.1 2>&1 | tee "${PYTEST_LOG}"

if [ "${SETUP_ENV}" = "true" ]; then
  # Deactivate the virtual environment
  deactivate
fi
