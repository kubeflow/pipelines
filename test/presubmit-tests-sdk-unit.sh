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

set -o pipefail

source_root=$(pwd)
SETUP_ENV="${SETUP_ENV:-true}"
JUNIT_XML="${JUNIT_XML:-sdk-unit.xml}"
# Number of pytest-xdist workers; set to 0 to disable xdist (e.g. for bisecting).
PYTEST_LOG="${PYTEST_LOG:-sdk-unit.log}"
PYTEST_PARALLEL_WORKERS="${PYTEST_PARALLEL_WORKERS:-auto}"

if [ "${SETUP_ENV}" = "true" ]; then
  # Create a virtual environment and activate it
  python3 -m venv venv
  source venv/bin/activate

  python3 -m pip install --upgrade pip
  python3 -m pip install setuptools wheel==0.42.0
  # Generate the checkout's pipeline spec before resolving SDK dependencies.
  (cd api && make clean python)
fi

# Local execution starts fresh subprocess environments. Installing companions in
# the parent is insufficient when this checkout's versions are not on PyPI yet.
sdk_dependency_wheels=$(mktemp -d "${TMPDIR:-/tmp}/kfp-sdk-wheels.XXXXXX")
trap 'rm -rf -- "$sdk_dependency_wheels"' EXIT
python -m pip wheel --no-deps --wheel-dir "$sdk_dependency_wheels" \
  "$source_root/api/v2alpha1/python" \
  "$source_root/backend/api/v2beta1/python_http_client"
export PIP_FIND_LINKS="$sdk_dependency_wheels${PIP_FIND_LINKS:+ $PIP_FIND_LINKS}"

if [ "${SETUP_ENV}" = "true" ]; then
  python3 -m pip install -r sdk/python/requirements.txt
  python3 -m pip install -r sdk/python/requirements-dev.txt
  python3 -m pip install pytest-cov
  python3 -m pip install --upgrade protobuf
  python3 -m pip install sdk/python
  python3 -m pip install --force-reinstall "$sdk_dependency_wheels"/*.whl
fi

if [[ -n "${KFP_TEST_SOURCE_SHA:-}" ]]; then
  export KFP_PACKAGE_PATH="git+https://github.com/${REPO_NAME}@${KFP_TEST_SOURCE_SHA}#egg=kfp&subdirectory=sdk/python"
elif [[ "${GITHUB_EVENT_NAME:-}" == "merge_group" || "${GITHUB_REF:-}" == refs/heads/gh-readonly-queue/release-2.18/* ]]; then
  : "${GITHUB_SHA:?merge_group requires the queued commit SHA}"
  export KFP_PACKAGE_PATH="git+https://github.com/${REPO_NAME}@${GITHUB_SHA}#egg=kfp&subdirectory=sdk/python"
elif [[ -z "${PULL_NUMBER}" ]]; then
  export KFP_PACKAGE_PATH="git+https://github.com/${REPO_NAME}@release-2.18#egg=kfp&subdirectory=sdk/python"
else
  export KFP_PACKAGE_PATH="git+https://github.com/${REPO_NAME}@refs/pull/${PULL_NUMBER}/merge#egg=kfp&subdirectory=sdk/python"
fi

# Run tests in parallel with pytest-xdist. --dist loadfile keeps all tests in
# a file on the same worker so file-scoped fixtures and local-execution state
# stay isolated. -s is dropped because xdist workers do not forward live output.
python -m pytest -v -n "${PYTEST_PARALLEL_WORKERS}" --dist loadfile sdk/python/kfp --cov=kfp --junitxml="${JUNIT_XML}" --durations=30 --durations-min=0.1 2>&1 | tee "${PYTEST_LOG}"

if [ "${SETUP_ENV}" = "true" ]; then
  # Deactivate the virtual environment
  deactivate
fi
