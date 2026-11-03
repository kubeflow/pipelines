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
JUNIT_XML="${JUNIT_XML:-sdk-unit.xml}"
# Number of pytest-xdist workers; set to 0 to disable xdist (e.g. for bisecting).
PYTEST_PARALLEL_WORKERS="${PYTEST_PARALLEL_WORKERS:-auto}"

if [[ -z "${PULL_NUMBER}" ]]; then
  export KFP_PACKAGE_PATH="git+https://github.com/${REPO_NAME}#egg=kfp&subdirectory=sdk/python"
else
  export KFP_PACKAGE_PATH="git+https://github.com/${REPO_NAME}@refs/pull/${PULL_NUMBER}/merge#egg=kfp&subdirectory=sdk/python"
fi

# Run tests in parallel with pytest-xdist. --dist loadfile keeps all tests in
# a file on the same worker so file-scoped fixtures and local-execution state
# stay isolated. -s is dropped because xdist workers do not forward live output.
uv run pytest -v -n "${PYTEST_PARALLEL_WORKERS}" --dist loadfile sdk/python/kfp --cov=kfp --junitxml="${JUNIT_XML}"

if [ "${SETUP_ENV}" = "true" ]; then
  # Deactivate the virtual environment
  deactivate
fi
