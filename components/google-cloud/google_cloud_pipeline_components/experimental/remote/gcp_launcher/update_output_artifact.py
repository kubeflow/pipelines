# Copyright 2021 The Kubeflow Authors. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import json
import os

def write_to_artifact(executor_input, target_artifact_name, uri):
    """Write output to local artifact and metadata path (uses GCSFuse)."""

    executor_output = {}
    executor_output['artifacts'] = {}
    for name, artifacts in executor_input.get('outputs', {}).get('artifacts',
                                                                 {}).items():
        artifacts_list = artifacts.get('artifacts')
        if name == target_artifact_name and artifacts_list:
            old_runtime_artifact = artifacts_list[0]
            updated_runtime_artifact = {
                "name": old_runtime_artifact.get('name'),
                "uri": uri,
                # todo allow override metadata
                "metadata": old_runtime_artifact.get('metadata', {})
            }
            artifacts_list = {'artifacts': [updated_runtime_artifact]}

        executor_output['artifacts'][name] = artifacts_list

    # update the output artifacts.
    os.makedirs(
        os.path.dirname(executor_input['outputs']['outputFile']), exist_ok=True
    )
    with open(executor_input['outputs']['outputFile'], 'w') as f:
        f.write(json.dumps(executor_output))