#!/bin/bash
# Copyright 2021 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Usage: ./update_requirements.sh <requirements.in >requirements.txt

set -euo pipefail
IMAGE=${1:-"python:3.7"}
docker run -i --rm  --entrypoint "" "$IMAGE" sh -c '
  python3 -m pip install pip setuptools --upgrade --quiet
  python3 -m pip install pip-tools==5.4.0 --quiet
  pip-compile --verbose --output-file - -
'
