import json
import logging

from googleapiclient import errors

from kfp.gcp.core.base_op import BaseOp
from kfp.gcp.core import name_utils
from ml_engine import utils

class DeleteVersionOp(BaseOp):

    def __init__(self, project_id, model_name, version_name, staging_dir, wait_interval):
        super().__init__()
        if staging_dir:
            self.enable_staging(staging_dir, '{}/{}/{}'.format(project_id, model_name, version_name))
        self._ml = utils.create_ml_client()
        self._project_id = project_id
        self._model_name = model_name
        self._version_name = version_name
        self._wait_interval = wait_interval

    def on_executing(self):
        operation_name = self.staging_states.get('operation_name', None)
        if operation_name:
            self._wait_for_done(operation_name)
            return

        name = 'projects/{}/models/{}/versions/{}'.format(
            self._project_id,
            self._model_name,
            self._version_name
        )
        request = self._ml.projects().models().versions().delete(
            name = name
        )
        operation = None
        try:
            operation = request.execute()
        except errors.HttpError as e:
            if e.resp.status == 404:
                logging.warning('Version {} does not exist'.format(name))
                pass
            else:
                raise

        if operation:
            self._wait_for_done(operation.get('name'))

    def _wait_for_done(self, operation_name):
        self.staging_states['operation_name'] = operation_name
        operation = utils.wait_for_operation_done(self._ml, operation_name, self._wait_interval)
        if operation.error:
            raise RuntimeError('Failed to delete version. Error: {}'.format(operation.error))
        if operation.response:
            self._dump_response(operation.response)
        
    def _dump_response(self, response):
        logging.info('Dumping response: {}'.format(response))
        with open('/tmp/response.json', 'w') as f:
            json.dump(response, f)
