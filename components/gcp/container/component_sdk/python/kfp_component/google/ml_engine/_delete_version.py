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

from kfp_component.core import KfpExecutionContext
from ._client import MLEngineClient
from .. import common as gcp_common
from ._common_ops import wait_existing_version, wait_for_operation_done

def delete_version(version_name, wait_interval=30):
    """Deletes a MLEngine version and wait.

    Args:
        version_name (str): required, the name of the version.
        wait_interval (int): the interval to wait for a long running operation.
    """
    DeleteVersionOp(version_name, wait_interval).execute_and_wait()

class DeleteVersionOp:
    def __init__(self, version_name, wait_interval):
        self._ml = MLEngineClient()
        self._version_name = version_name
        self._wait_interval = wait_interval
        self._delete_operation_name = None

    def execute_and_wait(self):
        with KfpExecutionContext(on_cancel=self._cancel):
            existing_version = wait_existing_version(self._ml, 
                self._version_name, 
                self._wait_interval)
            if not existing_version:
                logging.info('The version has already been deleted.')
                return None

            logging.info('Deleting existing version...')
            operation = self._ml.delete_version(self._version_name)
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

    def _cancel(self):
        if self._delete_operation_name:
            self._ml.cancel_operation(self._delete_operation_name)