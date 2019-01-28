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

import gcp_common
from kfp_component import BaseOp
from ml_engine.client import MLEngineClient
from ._common_ops import wait_existing_version, wait_for_operation_done

class DeleteVersionOp(BaseOp):
    """Operation for deleting a MLEngine version.

    Args:
        project_id: the ID of the parent project.
        model_name: the name of the parent model.
        version_name: the name of the version.
        wait_interval: the interval to wait for a long running operation.
    """
    def __init__(self, project_id, model_name, version_name, wait_interval):
        super().__init__()
        self._ml = MLEngineClient()
        self._project_id = project_id
        self._model_name = gcp_common.normalize_name(model_name)
        self._version_name = gcp_common.normalize_name(version_name)
        self._wait_interval = wait_interval
        self._delete_operation_name = None

    def on_executing(self):
        existing_version = wait_existing_version(self._ml, 
            self._project_id, self._model_name, self._version_name, 
            self._wait_interval)
        if not existing_version:
            logging.info('The version has already been deleted.')
            return None

        logging.info('Deleting existing version...')
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
        return None

    def on_cancelling(self):
        if self._delete_operation_name:
            self._ml.cancel_operation(self._delete_operation_name)
