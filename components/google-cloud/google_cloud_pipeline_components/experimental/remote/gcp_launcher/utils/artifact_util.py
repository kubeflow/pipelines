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


def update_output_artifact(executor_input: str, target_artifact_name: str,
                           uri: str):
    """Update the output artifact with the new uri."""
    executor_input_json = json.loads(executor_input)
    executor_output = {}
    executor_output['artifacts'] = {}
    for name, artifacts in executor_input_json.get('outputs',
                                                   {}).get('artifacts',
                                                           {}).items():
        artifacts_list = artifacts.get('artifacts')
        if name == target_artifact_name and artifacts_list:
            updated_runtime_artifact = artifacts_list[0]
            updated_runtime_artifact['uri'] = uri
            artifacts_list = {'artifacts': [updated_runtime_artifact]}

        executor_output['artifacts'][name] = artifacts_list

    # update the output artifacts.
    os.makedirs(
        os.path.dirname(executor_input_json['outputs']['outputFile']),
        exist_ok=True)
    with open(executor_input_json['outputs']['outputFile'], 'w') as f:
        f.write(json.dumps(executor_output))
