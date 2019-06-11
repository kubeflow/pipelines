# Copyright 2019 Google LLC
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
import fire

from ml_metadata.proto import metadata_store_pb2

from metadata import Metadata
import logging
import json
from typing import Dict
import copy
import hashlib

KFP_OUTPUTS_DIR = '/tmp/kfp/outputs'

# TODO(hongyes): remove pod ID to get stable digest
def _compute_digest(execution_json: str, workflow_id: str) -> str:
    stable_execution_json = execution_json.replace(workflow_id, '')
    return hashlib.md5(stable_execution_json.encode()).hexdigest()
    
def track(execution: Dict, mlmd: Metadata):
    logging.basicConfig(level=logging.INFO)
    logging.info('Execution: ' + str(execution))
    
    # TODO(hongyes): add more exec_properties: envvar, pod id, etc.
    # TODO(hongyes): log artifacts (untyped, dataset and model)
    execution_json =json.dumps(execution, sort_keys=True)
    execution_id = mlmd.prepare_execution('kubeflow.org/container-execution/v1', exec_properties={
        'image': execution['image'],
        'workflow_id': execution['workflow_id'],
        'digest': _compute_digest(execution_json, execution['workflow_id']),
        'configuration': execution_json,
    })
    logging.info('Execution ID: ' + str(execution_id))

    if 'outputs' in execution:
        if not os.path.exists(KFP_OUTPUTS_DIR):
            os.makedirs(KFP_OUTPUTS_DIR)
        for o_key, o_value in execution['outputs'].items():
            output_path = os.path.join(KFP_OUTPUTS_DIR, o_key)
            with open(output_path, 'w') as f:
                f.write(o_value)

    return execution_id

def track_wrapper(execution: Dict):
    connection_config = metadata_store_pb2.ConnectionConfig()
    # TODO(hongyes): find in-cluster DB connection for metadata service.
    connection_config.fake_database.SetInParent()
    track(execution, Metadata(connection_config))

if __name__ == '__main__':
    fire.Fire(track_wrapper)