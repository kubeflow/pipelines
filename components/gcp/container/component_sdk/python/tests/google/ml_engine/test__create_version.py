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
from kfp_component.google.ml_engine import create_version

CREATE_VERSION_MODULE = 'kfp_component.google.ml_engine._create_version'

@mock.patch(CREATE_VERSION_MODULE + '.display.display')
@mock.patch(CREATE_VERSION_MODULE + '.gcp_common.dump_file')
@mock.patch(CREATE_VERSION_MODULE + '.KfpExecutionContext')
@mock.patch(CREATE_VERSION_MODULE + '.MLEngineClient')
class TestCreateVersion(unittest.TestCase):

    def test_create_version_succeed(self, mock_mlengine_client,
        mock_kfp_context, mock_dump_json, mock_display):
        version = {
            'description': 'the mock version'
        }
        mock_mlengine_client().get_version.return_value = None
        mock_mlengine_client().create_version.return_value = {
            'name': 'mock_operation_name'
        }
        mock_mlengine_client().get_operation.return_value = {
            'done': True,
            'response': version
        }

        result = create_version('projects/mock_project/models/mock_model', 
            deployemnt_uri = 'gs://test-location', version_id = 'mock_version',
            version = version, 
            replace_existing = True)

        self.assertEqual(version, result)

    def test_create_version_fail(self, mock_mlengine_client,
        mock_kfp_context, mock_dump_json, mock_display):
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

        with self.assertRaises(RuntimeError) as context:
            create_version('projects/mock_project/models/mock_model',
                version = version, replace_existing = True, wait_interval = 30)

        self.assertEqual(
            'Failed to complete create version operation mock_operation_name: 400 bad request', 
            str(context.exception))

    def test_create_version_dup_version_succeed(self, mock_mlengine_client,
        mock_kfp_context, mock_dump_json, mock_display):
        version = {
            'name': 'mock_version',
            'description': 'the mock version',
            'deploymentUri': 'gs://test-location'
        }
        pending_version = {
            'state': 'CREATING'
        }
        pending_version.update(version)
        ready_version = { 
            'state': 'READY'
        }
        ready_version.update(version)
        mock_mlengine_client().get_version.side_effect = [
            pending_version, ready_version]

        result = create_version('projects/mock_project/models/mock_model', version = version, 
            replace_existing = True, wait_interval = 0)

        self.assertEqual(ready_version, result)

    def test_create_version_failed_state(self, mock_mlengine_client,
        mock_kfp_context, mock_dump_json, mock_display):
        version = {
            'name': 'mock_version',
            'description': 'the mock version',
            'deploymentUri': 'gs://test-location'
        }
        pending_version = {
            'state': 'CREATING'
        }
        pending_version.update(version)
        failed_version = {
            'state': 'FAILED',
            'errorMessage': 'something bad happens'
        }
        failed_version.update(version)
        mock_mlengine_client().get_version.side_effect = [
            pending_version, failed_version]

        with self.assertRaises(RuntimeError) as context:
            create_version('projects/mock_project/models/mock_model', version = version, 
                replace_existing = True, wait_interval = 0)

        self.assertEqual(
            'Version is in failed state: something bad happens', 
            str(context.exception))

    def test_create_version_conflict_version_replace_succeed(self, mock_mlengine_client,
        mock_kfp_context, mock_dump_json, mock_display):
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

        result = create_version('projects/mock_project/models/mock_model', version = version, 
            replace_existing = True, wait_interval = 0)

        self.assertEqual(version, result)

    def test_create_version_conflict_version_delete_fail(self, mock_mlengine_client,
        mock_kfp_context, mock_dump_json, mock_display):
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

        with self.assertRaises(RuntimeError) as context:
            create_version('projects/mock_project/models/mock_model', version = version, 
                replace_existing = True, wait_interval = 0)

        self.assertEqual(
            'Failed to complete delete version operation delete_operation_name: 400 bad request', 
            str(context.exception))

    def test_create_version_conflict_version_fail(self, mock_mlengine_client,
        mock_kfp_context, mock_dump_json, mock_display):
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

        with self.assertRaises(RuntimeError) as context:
            create_version('projects/mock_project/models/mock_model', version = version, 
                replace_existing = False, wait_interval = 0)

        self.assertEqual(
            'Existing version conflicts with the name of the new version.', 
            str(context.exception))