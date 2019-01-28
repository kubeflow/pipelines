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
from ml_engine.create_version_op import CreateVersionOp
from ml_engine.client import MLEngineClient
import ml_engine

@mock.patch('ml_engine.create_version_op.gcp_common.dump_file')
@mock.patch('kubernetes.config.load_incluster_config')
@mock.patch('kubernetes.client.CoreV1Api')
@mock.patch('ml_engine.create_version_op.MLEngineClient')
class TestCreateVersionOp(unittest.TestCase):

    def test_init(self, mock_mlengine_client,
        mock_core_v1_api, mock_load_incluster_config, 
        mock_dump_json):
        version = {
            'name': 'mock_version',
            'description': 'the mock version',
            'deploymentUri': 'gs://test-location'
        }
        op = CreateVersionOp('mock_project', 'mock_model', version, 
            replace_existing = True, wait_interval = 30)

        self.assertEqual('mock_project', op._project_id)
        self.assertEqual('mock_model', op._model_name)
        self.assertEqual(version['name'], op._version_name)
        self.assertEqual(version, op._version)
        self.assertEqual(True, op._replace_existing)
        self.assertEqual(30, op._wait_interval)

    def test_execute_create_version_succeed(self, mock_mlengine_client,
        mock_core_v1_api, mock_load_incluster_config, 
        mock_dump_json):
        version = {
            'name': 'mock_version',
            'description': 'the mock version',
            'deploymentUri': 'gs://test-location'
        }
        mock_mlengine_client().get_version.return_value = None
        mock_mlengine_client().create_version.return_value = {
            'name': 'mock_operation_name'
        }
        mock_mlengine_client().get_operation.return_value = {
            'done': True,
            'response': version
        }
        op = CreateVersionOp('mock_project', 'mock_model', version, 
            replace_existing = True, wait_interval = 30)

        actual = op.execute()

        self.assertEqual(version, actual)

    def test_execute_create_version_fail(self, mock_mlengine_client,
        mock_core_v1_api, mock_load_incluster_config, 
        mock_dump_json):
        version = {
            'name': 'mock_version',
            'description': 'the mock version',
            'deploymentUri': 'gs://test-location'
        }
        mock_mlengine_client().get_version.return_value = None
        mock_mlengine_client().create_version.return_value = {
            'name': 'mock_operation_name'
        }
        mock_mlengine_client().get_operation.return_value = {
            'done': True,
            'error': {
                'code': 400,
                'message': 'bad request'
            }
        }
        op = CreateVersionOp('mock_project', 'mock_model', version, 
            replace_existing = True, wait_interval = 30)

        with self.assertRaises(RuntimeError) as context:
            op.execute()

        self.assertEqual(
            'Failed to complete create version operation mock_operation_name: 400 bad request', 
            str(context.exception))

    def test_execute_dup_version_succeed(self, mock_mlengine_client,
        mock_core_v1_api, mock_load_incluster_config, 
        mock_dump_json):
        version = {
            'name': 'mock_version',
            'description': 'the mock version',
            'deploymentUri': 'gs://test-location'
        }
        pending_version = { 
            **version,
            'state': 'CREATING'
        }
        ready_version = { 
            **version,
            'state': 'READY'
        }
        mock_mlengine_client().get_version.side_effect = [pending_version, ready_version]
        op = CreateVersionOp('mock_project', 'mock_model', version, 
            replace_existing = True, wait_interval = 0)

        actual = op.execute()

        self.assertEqual(ready_version, actual)

    def test_execute_dup_version_succeed(self, mock_mlengine_client,
        mock_core_v1_api, mock_load_incluster_config, 
        mock_dump_json):
        version = {
            'name': 'mock_version',
            'description': 'the mock version',
            'deploymentUri': 'gs://test-location'
        }
        pending_version = { 
            **version,
            'state': 'CREATING'
        }
        failed_version = { 
            **version,
            'state': 'FAILED',
            'errorMessage': 'something bad happens'
        }
        mock_mlengine_client().get_version.side_effect = [pending_version, failed_version]
        op = CreateVersionOp('mock_project', 'mock_model', version, 
            replace_existing = True, wait_interval = 0)

        with self.assertRaises(RuntimeError) as context:
            op.execute()

        self.assertEqual(
            'Version is in failed state: something bad happens', 
            str(context.exception))

    def test_execute_conflict_version_replace_succeed(self, mock_mlengine_client,
        mock_core_v1_api, mock_load_incluster_config, 
        mock_dump_json):
        version = {
            'name': 'mock_version',
            'description': 'the mock version',
            'deploymentUri': 'gs://test-location'
        }
        conflicting_version = {
            'name': 'mock_version',
            'description': 'the changed mock version',
            'deploymentUri': 'gs://changed-test-location',
            'state': 'READY'
        }
        mock_mlengine_client().get_version.return_value = conflicting_version
        mock_mlengine_client().delete_version.return_value = {
            'name': 'delete_operation_name'
        }
        mock_mlengine_client().create_version.return_value = {
            'name': 'create_operation_name'
        }
        delete_operation = { 'response': {}, 'done': True }
        create_operation = { 'response': version, 'done': True }
        mock_mlengine_client().get_operation.side_effect = [
            delete_operation,
            create_operation
        ]
        op = CreateVersionOp('mock_project', 'mock_model', version, 
            replace_existing = True, wait_interval = 0)

        actual = op.execute()

        self.assertEqual(version, actual)

    def test_execute_conflict_version_delete_fail(self, mock_mlengine_client,
        mock_core_v1_api, mock_load_incluster_config, 
        mock_dump_json):
        version = {
            'name': 'mock_version',
            'description': 'the mock version',
            'deploymentUri': 'gs://test-location'
        }
        conflicting_version = {
            'name': 'mock_version',
            'description': 'the changed mock version',
            'deploymentUri': 'gs://changed-test-location',
            'state': 'READY'
        }
        mock_mlengine_client().get_version.return_value = conflicting_version
        mock_mlengine_client().delete_version.return_value = {
            'name': 'delete_operation_name'
        }
        delete_operation = {
            'done': True,
            'error': {
                'code': 400,
                'message': 'bad request'
            } 
        }
        mock_mlengine_client().get_operation.return_value = delete_operation
        op = CreateVersionOp('mock_project', 'mock_model', version, 
            replace_existing = True, wait_interval = 0)

        with self.assertRaises(RuntimeError) as context:
            op.execute()

        self.assertEqual(
            'Failed to complete delete version operation delete_operation_name: 400 bad request', 
            str(context.exception))

    def test_execute_conflict_version_fail(self, mock_mlengine_client,
        mock_core_v1_api, mock_load_incluster_config, 
        mock_dump_json):
        version = {
            'name': 'mock_version',
            'description': 'the mock version',
            'deploymentUri': 'gs://test-location'
        }
        conflicting_version = {
            'name': 'mock_version',
            'description': 'the changed mock version',
            'deploymentUri': 'gs://changed-test-location',
            'state': 'READY'
        }
        mock_mlengine_client().get_version.return_value = conflicting_version
        op = CreateVersionOp('mock_project', 'mock_model', version, 
            replace_existing = False, wait_interval = 0)

        with self.assertRaises(RuntimeError) as context:
            op.execute()

        self.assertEqual(
            'Existing version conflicts with the name of the new version.', 
            str(context.exception))

