
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
from ml_engine.delete_version_op import DeleteVersionOp
from ml_engine.client import MLEngineClient
import ml_engine

@mock.patch('kubernetes.config.load_incluster_config')
@mock.patch('kubernetes.client.CoreV1Api')
@mock.patch('ml_engine.delete_version_op.MLEngineClient')
class TestDeleteVersionOp(unittest.TestCase):

    def test_init(self, mock_mlengine_client,
        mock_core_v1_api, mock_load_incluster_config):
        op = DeleteVersionOp('mock_project', 'mock_model', 'mock_version', 
            wait_interval = 30)

        self.assertEqual('mock_project', op._project_id)
        self.assertEqual('mock_model', op._model_name)
        self.assertEqual('mock_version', op._version_name)
        self.assertEqual(30, op._wait_interval)

    def test_execute_succeed(self, mock_mlengine_client,
        mock_core_v1_api, mock_load_incluster_config):
        mock_mlengine_client().get_version.return_value = {
            'state': 'READY',
        }
        mock_mlengine_client().delete_version.return_value = {
            'name': 'mock_operation_name'
        }
        mock_mlengine_client().get_operation.return_value = {
            'done': True
        }
        op = DeleteVersionOp('mock_project', 'mock_model', 'mock_version', 
            wait_interval = 30)

        op.execute()

        mock_mlengine_client().delete_version.assert_called_once()

    def test_execute_retry_succeed(self, mock_mlengine_client,
        mock_core_v1_api, mock_load_incluster_config):
        pending_version = {
            'state': 'DELETING',
        }
        mock_mlengine_client().get_version.side_effect = [pending_version, None]
        op = DeleteVersionOp('mock_project', 'mock_model', 'mock_version', 
            wait_interval = 0)

        op.execute()

        self.assertEqual(2, mock_mlengine_client().get_version.call_count)