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

from kfp_component.core import KfpExecutionContext, display
from ._client import MLEngineClient
from .. import common as gcp_common

def create_model(project_id, model_id=None, model=None,
    model_name_output_path='/tmp/kfp/output/ml_engine/model_name.txt',
    model_object_output_path='/tmp/kfp/output/ml_engine/model.json',
):
    """Creates a MLEngine model.

    Args:
        project_id (str): the ID of the parent project of the model.
        model_id (str): optional, the name of the model. If absent, a new name will 
            be generated.
        model (dict): the payload of the model.
    """
    return CreateModelOp(project_id, model_id, model,
        model_name_output_path=model_name_output_path,
        model_object_output_path=model_object_output_path,
    ).execute()

class CreateModelOp:
    def __init__(self, project_id, model_id, model,
        model_name_output_path,
        model_object_output_path,
    ):
        self._ml = MLEngineClient()
        self._project_id = project_id
        self._model_id = model_id
        self._model_name = None
        if model:
            self._model = model
        else:
            self._model = {}
        self._model_name_output_path = model_name_output_path
        self._model_object_output_path = model_object_output_path

    def execute(self):
        with KfpExecutionContext() as ctx:
            self._set_model_name(ctx.context_id())
            self._dump_metadata()
            try:
                created_model = self._ml.create_model(
                    project_id = self._project_id,
                    model = self._model)
            except errors.HttpError as e:
                if e.resp.status == 409:
                    existing_model = self._ml.get_model(self._model_name)
                    if not self._is_dup_model(existing_model):
                        raise
                    logging.info('The same model {} has been submitted'
                        ' before. Continue the operation.'.format(
                            self._model_name))
                    created_model = existing_model
                else:
                    raise
            self._dump_model(created_model)
            return created_model

    def _set_model_name(self, context_id):
        if not self._model_id:
            self._model_id = 'model_' + context_id
        self._model['name'] = gcp_common.normalize_name(self._model_id)
        self._model_name = 'projects/{}/models/{}'.format(
            self._project_id, self._model_id)


    def _is_dup_model(self, existing_model):
        return not gcp_common.check_resource_changed(
            self._model,
            existing_model,
            ['description', 'regions', 
                'onlinePredictionLogging', 'labels'])

    def _dump_metadata(self):
        display.display(display.Link(
            'https://console.cloud.google.com/mlengine/models/{}?project={}'.format(
                self._model_id, self._project_id),
            'Model Details'
        ))

    def _dump_model(self, model):
        logging.info('Dumping model: {}'.format(model))
        gcp_common.dump_file(self._model_object_output_path, json.dumps(model))
        gcp_common.dump_file(self._model_name_output_path, self._model_name)