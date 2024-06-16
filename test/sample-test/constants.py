# Copyright 2019 The Kubeflow Authors
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

import os

# String msgs.
PAPERMILL_ERR_MSG = 'An Exception was encountered at'

# Common paths
GITHUB_REPO = 'kubeflow/pipelines'
BASE_DIR = os.path.abspath('./')
TEST_DIR = os.path.join(BASE_DIR, 'test/sample-test')
CONFIG_DIR = os.path.join(TEST_DIR, 'configs')
DEFAULT_CONFIG = os.path.join(CONFIG_DIR, 'default.config.yaml')
SCHEMA_CONFIG = os.path.join(CONFIG_DIR, 'schema.config.yaml')

# Common test params
RUN_LIST_PAGE_SIZE = 1000
