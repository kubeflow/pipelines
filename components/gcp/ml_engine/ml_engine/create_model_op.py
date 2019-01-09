
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