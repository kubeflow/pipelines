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
from kfp_component.google.ml_engine import delete_version

DELETE_VERSION_MODULE = 'kfp_component.google.ml_engine._delete_version'

@mock.patch(DELETE_VERSION_MODULE + '.gcp_common.dump_file')
@mock.patch(DELETE_VERSION_MODULE + '.KfpExecutionContext')
@mock.patch(DELETE_VERSION_MODULE + '.MLEngineClient')
class TestDeleteVersion(unittest.TestCase):

    def test_execute_succeed(self, mock_mlengine_client,
        mock_kfp_context, mock_dump_json):
        mock_mlengine_client().get_version.return_value = {
            'state': 'READY',
        }
        mock_mlengine_client().delete_version.return_value = {
            'name': 'mock_operation_name'
        }
        mock_mlengine_client().get_operation.return_value = {
            'done': True
        }

        delete_version('projects/mock_project/models/mock_model/versions/mock_version', 
            wait_interval = 30)

        mock_mlengine_client().delete_version.assert_called_once()

    def test_execute_retry_succeed(self, mock_mlengine_client,
        mock_kfp_context, mock_dump_json):
        pending_version = {
            'state': 'DELETING',
        }
        mock_mlengine_client().get_version.side_effect = [pending_version, None]

        delete_version('projects/mock_project/models/mock_model/versions/mock_version', 
            wait_interval = 0)

        self.assertEqual(2, mock_mlengine_client().get_version.call_count)