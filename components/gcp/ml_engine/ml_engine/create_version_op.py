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
import time

from googleapiclient import errors

import gcp_common
from kfp_component import BaseOp
from ml_engine.client import MLEngineClient
from ._common_ops import wait_existing_version, wait_for_operation_done

class CreateVersionOp(BaseOp):
    """Operation for creating a MLEngine version.

    Args:
        project_id: the ID of the parent project.
        model_name: the name of the parent model.
        version: the payload of the new version. It must have a ``name`` in it.
        replace_existing: boolean flag indicates whether to replace existing 
            version in case of conflict.
        wait_interval: the interval to wait for a long running operation.
    """
    def __init__(self, project_id, model_name, version, replace_existing, wait_interval):
        super().__init__()
        self._ml = MLEngineClient()
        self._project_id = project_id
        self._model_name = gcp_common.normalize_name(model_name)
        self._version_name = gcp_common.normalize_name(version['name'])
        version['name'] = self._version_name
        self._version = version
        self._replace_existing = replace_existing
        self._wait_interval = wait_interval
        self._create_operation_name = None
        self._delete_operation_name = None

    def on_executing(self):
        self._dump_metadata()
        existing_version = wait_existing_version(self._ml, 
            self._project_id, self._model_name, self._version_name, 
            self._wait_interval)
        if existing_version and self._is_dup_version(existing_version):
            return self._handle_completed_version(existing_version)

        if existing_version and self._replace_existing:
            logging.info('Deleting existing version...')
            self._delete_version_and_wait()
        elif existing_version:
            raise RuntimeError(
                'Existing version conflicts with the name of the new version.')
        
        created_version = self._create_version_and_wait()
        return self._handle_completed_version(created_version)

    def on_cancelling(self):
        if self._delete_operation_name:
            self._ml.cancel_operation(self._delete_operation_name)

        if self._create_operation_name:
            self._ml.cancel_operation(self._create_operation_name)

    def _create_version_and_wait(self):
        operation = self._ml.create_version(self._project_id, 
            self._model_name, self._version)
        # Cache operation name for cancellation.
        self._create_operation_name = operation.get('name')
        try:
            operation = wait_for_operation_done(
                self._ml,
                self._create_operation_name, 
                'create version',
                self._wait_interval)
        finally:
            self._create_operation_name = None
        return operation.get('response', None)

    def _delete_version_and_wait(self):
        operation = self._ml.delete_version(
                self._project_id, self._model_name, self._version_name)
        # Cache operation name for cancellation.
        self._delete_operation_name = operation.get('name')
        try:
            wait_for_operation_done(
                self._ml,
                self._delete_operation_name, 
                'delete version',
                self._wait_interval)
        finally:
            self._delete_operation_name = None
        
    def _handle_completed_version(self, version):
        state = version.get('state', None)
        if state == 'FAILED':
            error_message = version.get('errorMessage', 'Unknown failure')
            raise RuntimeError('Version is in failed state: {}'.format(
                error_message))
        self._dump_version(version)
        return version

    def _dump_metadata(self):
        metadata = {
            'outputs' : [{
                'type': 'link',
                'name': 'version details',
                'href': 'https://console.cloud.google.com/mlengine/models/{}/versions/{}?project={}'.format(
                    self._model_name, self._version_name, self._project_id)
            }]
        }
        logging.info('Dumping UI metadata: {}'.format(metadata))
        gcp_common.dump_file('/mlpipeline-ui-metadata.json', json.dumps(metadata))

    def _dump_version(self, version):
        logging.info('Dumping version: {}'.format(version))
        gcp_common.dump_file('/output.txt', json.dumps(version))
        gcp_common.dump_file('/version_name.txt', version['name'])

    def _is_dup_version(self, existing_version):
        return not gcp_common.check_resource_changed(
            self._version,
            existing_version,
            ['description', 'deploymentUri', 
                'runtimeVersion', 'machineType', 'labels',
                'framework', 'pythonVersion', 'autoScaling',
                'manualScaling'])

    def _dump_response(self, response):
        logging.info('Dumping response: {}'.format(response))
        with open('/tmp/response.json', 'w') as f:
            json.dump(response, f)