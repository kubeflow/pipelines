# Copyright 2018 Google LLC
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

import json
import logging

from googleapiclient import errors

from kfp.gcp.core.base_op import BaseOp
from kfp.gcp.core import name_utils
from ml_engine import utils

class CreateVersionOp(BaseOp):

    def __init__(self, project_id, model_name, version, staging_dir, wait_interval):
        super().__init__()
        self._ml = utils.create_ml_client()
        self._project_id = project_id
        self._model_name = model_name
        self._version_name = name_utils.normalize(version['name'])
        version['name'] = self._version_name
        self._version = version
        self._wait_interval = wait_interval
        if staging_dir:
            self.enable_staging(staging_dir, '{}/{}/{}'.format(project_id, model_name, self._version_name))

    def on_executing(self):
        operation_name = self.staging_states.get('operation_name', None)
        if operation_name:
            return self._wait_for_done(operation_name)

        parent = 'projects/{}/models/{}'.format(self._project_id, self._model_name)
        request = self._ml.projects().models().versions().create(
            parent = parent,
            body = self._version
        )
        operation = request.execute()
        if operation:
            self._wait_for_done(operation.get('name'))

    def on_cancelling(self):
        operation_name = self.staging_states.get('operation_name', None)
        if operation_name:
            self._ml.projects().operations().cancel(name=operation_name).execute()

    def _wait_for_done(self, operation_name):
        self.staging_states['operation_name'] = operation_name
        operation = utils.wait_for_operation_done(self._ml, operation_name, self._wait_interval)
        error = operation.get('error', None)
        response = operation.get('response', None)
        if error:
            raise RuntimeError('Failed to create version. Error: {}'.format(error))
        if response:
            self._dump_response(response)
        
    def _dump_response(self, response):
        logging.info('Dumping response: {}'.format(response))
        with open('/tmp/response.json', 'w') as f:
            json.dump(response, f)