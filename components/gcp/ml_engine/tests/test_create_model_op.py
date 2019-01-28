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

import mock
import unittest

from googleapiclient import errors
from ml_engine.create_model_op import CreateModelOp
from ml_engine.client import MLEngineClient
import ml_engine

@mock.patch('ml_engine.create_model_op.gcp_common.dump_file')
@mock.patch('kubernetes.config.load_incluster_config')
@mock.patch('kubernetes.client.CoreV1Api')
@mock.patch('ml_engine.create_model_op.MLEngineClient')
class TestCreateModelOp(unittest.TestCase):

    def test_init(self, mock_mlengine_client,
        mock_core_v1_api, mock_load_incluster_config, 
        mock_dump_json):
        model = {
            'name': 'mock_model',
            'description': 'the mock model'
        }
        op = CreateModelOp('mock_project', model)

        self.assertEqual('mock_project', op._project_id)
        self.assertEqual(model['name'], op._model_name)
        self.assertEqual(model, op._model)

    def test_execute_succeed(self, mock_mlengine_client,
        mock_core_v1_api, mock_load_incluster_config, 
        mock_dump_json):
        model = {
            'name': 'mock_model',
            'description': 'the mock model'
        }
        mock_mlengine_client().create_model.return_value = model
        op = CreateModelOp('mock_project', model)

        returned = op.execute()

        self.assertEqual(model, returned)

    def test_execute_conflict_succeed(self, mock_mlengine_client,
        mock_core_v1_api, mock_load_incluster_config, 
        mock_dump_json):
        model = {
            'name': 'mock_model',
            'description': 'the mock model'
        }
        mock_mlengine_client().create_model.side_effect = errors.HttpError(
            resp = mock.Mock(status=409),
            content = b'conflict'
        )
        mock_mlengine_client().get_model.return_value = model
        op = CreateModelOp('mock_project', model)

        returned = op.execute()

        self.assertEqual(model, returned)

    def test_execute_conflict_fail(self, mock_mlengine_client,
        mock_core_v1_api, mock_load_incluster_config, 
        mock_dump_json):
        model = {
            'name': 'mock_model',
            'description': 'the mock model'
        }
        mock_mlengine_client().create_model.side_effect = errors.HttpError(
            resp = mock.Mock(status=409),
            content = b'conflict'
        )
        changed_model = {
            'name': 'mock_model',
            'description': 'the changed mock model'
        }
        mock_mlengine_client().get_model.return_value = changed_model
        op = CreateModelOp('mock_project', model)

        with self.assertRaises(errors.HttpError) as context:
            op.execute()

        self.assertEqual(409, context.exception.resp.status)