#!/bin/bash -ex
# Copyright 2022 Kubeflow Pipelines contributors
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

BLUE='\033[0;34m'
GREEN='\033[0;32m'
NC='\033[0m'

echo "${BLUE}Cleaning up old run records...${NC}"
pip install kfp --pre
pip install tqdm==4.64.0
python3 "$source_root/test/tools/clean-mysql-runs/delete_old_runs.py"
echo "${BLUE}Deleting run cache...${NC}"
kubectl -n kubeflow exec -t mysql-client -- mysql -h mysql < "$source_root/test/tools/clean-mysql-runs/truncate_cache.sql"
echo "${GREEN}Successfully cleaned up old runs${NC}"
