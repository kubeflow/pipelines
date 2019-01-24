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

from kfp.gcp.core.base_op import BaseOp
from kfp.gcp.core import name_utils
from ml_engine import utils

class CreateModelOp(BaseOp):

    def __init__(self, project_id, model):
        super().__init__()
        self._ml = utils.create_ml_client()
        self._project_id = project_id
        self._model_name = name_utils.normalize(model['name'])
        model['name'] = self._model_name
        self._model = model

    def on_executing(self):
        self._dump_metadata()
        created_model = self._ml.projects().models().create(
            parent = 'projects/{}'.format(self._project_id),
            body = self._model
        ).execute()
        self._dump_model(created_model)

    def _dump_metadata(self):
        metadata = {
            'outputs' : [{
                'type': 'link',
                'name': 'model details',
                'href': 'https://console.cloud.google.com/mlengine/models/{}?project={}'.format(self._model_name, self._project_id)
            }]
        }
        logging.info('Dumping UI metadata: {}'.format(metadata))
        with open('/tmp/mlpipeline-ui-metadata.json', 'w') as f:
            json.dump(metadata, f)

    def _dump_model(self, model):
        logging.info('Dumping model: {}'.format(model))
        with open('/tmp/model.json', 'w') as f:
            json.dump(model, f)