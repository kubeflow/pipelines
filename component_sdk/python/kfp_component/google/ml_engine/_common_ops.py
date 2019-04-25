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

import logging
import time

from googleapiclient import errors

def wait_existing_version(ml_client, version_name, wait_interval):
    while True:
        existing_version = ml_client.get_version(version_name)
        if not existing_version:
            return None
        state = existing_version.get('state', None)
        if not state in ['CREATING', 'DELETING', 'UPDATING']:
            return existing_version
        logging.info('Version is in {} state. Wait for {}s'.format(
            state, wait_interval
        ))
        time.sleep(wait_interval)

def wait_for_operation_done(ml_client, operation_name, action, wait_interval):
    """Waits for an operation to be done.

    Args:
        operation_name: the name of the operation.
        action: the action name of the operation.
        wait_interval: the wait interview between pulling job
            status.

    Returns:
        The completed operation.

    Raises:
        RuntimeError if the operation has error.
    """
    operation = None
    while True:
        operation = ml_client.get_operation(operation_name)
        done = operation.get('done', False)
        if done:
            break
        logging.info('Operation {} is not done. Wait for {}s.'.format(operation_name, wait_interval))
        time.sleep(wait_interval)
    error = operation.get('error', None)
    if error:
        raise RuntimeError('Failed to complete {} operation {}: {} {}'.format(
            action,
            operation_name,
            error.get('code', 'Unknown code'),
            error.get('message', 'Unknown message'),
        ))
    return operation