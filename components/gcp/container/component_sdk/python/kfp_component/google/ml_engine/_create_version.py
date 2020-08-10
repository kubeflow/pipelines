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
import re

from googleapiclient import errors
from fire import decorators

from kfp_component.core import KfpExecutionContext, display
from ._client import MLEngineClient
from .. import common as gcp_common
from ._common_ops import wait_existing_version, wait_for_operation_done

@decorators.SetParseFns(python_version=str, runtime_version=str)
def create_version(model_name, deployemnt_uri=None, version_id=None, 
    runtime_version=None, python_version=None, version=None, 
    replace_existing=False, wait_interval=30,
    version_name_output_path='/tmp/kfp/output/ml_engine/version_name.txt',
    version_object_output_path='/tmp/kfp/output/ml_engine/version.json',
):
    """Creates a MLEngine version and wait for the operation to be done.

    Args:
        model_name (str): required, the name of the parent model.
        deployment_uri (str): optional, the Google Cloud Storage location of 
            the trained model used to create the version.
        version_id (str): optional, the user provided short name of 
            the version. If it is not provided, the operation uses a random name.
        runtime_version (str): optinal, the Cloud ML Engine runtime version 
            to use for this deployment. If not set, Cloud ML Engine uses 
            the default stable version, 1.0. 
        python_version (str): optinal, the version of Python used in prediction. 
            If not set, the default version is '2.7'. Python '3.5' is available
            when runtimeVersion is set to '1.4' and above. Python '2.7' works 
            with all supported runtime versions.
        version (dict): optional, the payload of the new version.
        replace_existing (boolean): boolean flag indicates whether to replace 
            existing version in case of conflict.
        wait_interval (int): the interval to wait for a long running operation.
    """
    if not version:
        version = {}
    if deployemnt_uri:
        version['deploymentUri'] = deployemnt_uri
    if version_id:
        version['name'] = version_id
    if runtime_version:
        version['runtimeVersion'] = runtime_version
    if python_version:
        version['pythonVersion'] = python_version

    return CreateVersionOp(model_name, version, 
        replace_existing, wait_interval,
        version_name_output_path=version_name_output_path,
        version_object_output_path=version_object_output_path,
    ).execute_and_wait()

class CreateVersionOp:
    def __init__(self, model_name, version, 
        replace_existing, wait_interval,
        version_name_output_path,
        version_object_output_path,
    ):
        self._ml = MLEngineClient()
        self._model_name = model_name
        self._project_id, self._model_id = self._parse_model_name(model_name)
        # The name of the version resource, which is in the format 
        # of projects/*/models/*/versions/*
        self._version_name = None
        # The user provide short name of the version.
        self._version_id = None
        # The full payload of the version resource.
        self._version = version
        self._replace_existing = replace_existing
        self._wait_interval = wait_interval
        self._create_operation_name = None
        self._delete_operation_name = None
        self._version_name_output_path = version_name_output_path
        self._version_object_output_path = version_object_output_path

    def execute_and_wait(self):
        with KfpExecutionContext(on_cancel=self._cancel) as ctx:
            self._set_version_name(ctx.context_id())
            self._dump_metadata()
            existing_version = wait_existing_version(self._ml, 
                self._version_name, 
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
    
    def _parse_model_name(self, model_name):
        match = re.search(r'^projects/([^/]+)/models/([^/]+)$', model_name)
        if not match:
            raise ValueError('model name "{}" is not in desired format.'.format(model_name))
        return (match.group(1), match.group(2))

    def _set_version_name(self, context_id):
        name = self._version.get('name', None)
        if not name:
            name = 'ver_' + context_id
        name = gcp_common.normalize_name(name)
        self._version_id = name
        self._version['name'] = name
        self._version_name = '{}/versions/{}'.format(self._model_name, name)

    def _cancel(self):
        if self._delete_operation_name:
            self._ml.cancel_operation(self._delete_operation_name)

        if self._create_operation_name:
            self._ml.cancel_operation(self._create_operation_name)

    def _create_version_and_wait(self):
        operation = self._ml.create_version(self._model_name, self._version)
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
        
    def _handle_completed_version(self, version):
        state = version.get('state', None)
        if state == 'FAILED':
            error_message = version.get('errorMessage', 'Unknown failure')
            raise RuntimeError('Version is in failed state: {}'.format(
                error_message))
        # Workaround issue that CMLE doesn't return the full version name.
        version['name'] = self._version_name
        self._dump_version(version)
        return version

    def _dump_metadata(self):
        display.display(display.Link(
            'https://console.cloud.google.com/mlengine/models/{}/versions/{}?project={}'.format(
                self._model_id, self._version_id, self._project_id),
            'Version Details'
        ))
        display.display(display.Markdown('''
## Online Prediction

### REST endpoint
The REST endpoint for online prediction is as follows:
```
POST https://ml.googleapis.com/v1/{}:predict
```
Try the REST endpoint in [Google OAuth 2.0 Playgound](https://developers.google.com/oauthplayground/#step3\
&apisSelect=https://www.googleapis.com/auth/cloud-platform&postData={{"instances":[]}}\
&url=https://ml.googleapis.com/v1/{}:predict&content_type=application/json&http_method=POST).

### GCloud command
```bash
gcloud ai-platform predict --model {}  \
                   --version {} \
                   --json-instances instances.json
```
        '''.format(self._version_name, self._version_name, self._model_id, self._version_id)))

    def _dump_version(self, version):
        logging.info('Dumping version: {}'.format(version))
        gcp_common.dump_file(self._version_name_output_path, json.dumps(version))
        gcp_common.dump_file(self._version_object_output_path, version['name'])

    def _is_dup_version(self, existing_version):
        return not gcp_common.check_resource_changed(
            self._version,
            existing_version,
            ['description', 'deploymentUri', 
                'runtimeVersion', 'machineType', 'labels',
                'framework', 'pythonVersion', 'autoScaling',
                'manualScaling'])
