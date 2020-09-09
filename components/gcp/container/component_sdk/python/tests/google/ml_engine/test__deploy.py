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
from kfp_component.google.ml_engine import deploy

MODULE = 'kfp_component.google.ml_engine._deploy'

@mock.patch(MODULE + '.storage.Client')
@mock.patch(MODULE + '.create_model')
@mock.patch(MODULE + '.create_version')
@mock.patch(MODULE + '.set_default_version')
class TestDeploy(unittest.TestCase):

    def test_deploy_default_path(self, mock_set_default_version, mock_create_version, 
        mock_create_model, mock_storage_client):

        mock_storage_client().bucket().list_blobs().prefixes = []
        mock_storage_client().bucket().list_blobs().__iter__.return_value = []
        mock_create_model.return_value = {
            'name': 'projects/mock-project/models/mock-model'
        }
        expected_version = {
            'name': 'projects/mock-project/models/mock-model/version/mock-version'
        }
        mock_create_version.return_value = expected_version

        result = deploy('gs://model/uri', 'mock-project',
            model_uri_output_path='/tmp/kfp/output/ml_engine/model_uri.txt',
            model_name_output_path='/tmp/kfp/output/ml_engine/model_name.txt',
            version_name_output_path='/tmp/kfp/output/ml_engine/version_name.txt',
        )

        self.assertEqual(expected_version, result)
        mock_create_version.assert_called_with(
            'projects/mock-project/models/mock-model',
            'gs://model/uri',
            None, # version_name
            None, # runtime_version
            None, # python_version
            None, # version
            False, # replace_existing_version
            30,
            version_name_output_path='/tmp/kfp/output/ml_engine/version_name.txt',
        )

    def test_deploy_tf_exporter_path(self, mock_set_default_version, mock_create_version, 
        mock_create_model, mock_storage_client):

        prefixes_mock = mock.PropertyMock()
        prefixes_mock.return_value = set(['uri/012/', 'uri/123/'])
        type(mock_storage_client().bucket().list_blobs()).prefixes = prefixes_mock
        mock_storage_client().bucket().list_blobs().__iter__.return_value = []
        mock_storage_client().bucket().name = 'model'
        mock_create_model.return_value = {
            'name': 'projects/mock-project/models/mock-model'
        }
        expected_version = {
            'name': 'projects/mock-project/models/mock-model/version/mock-version'
        }
        mock_create_version.return_value = expected_version

        result = deploy('gs://model/uri', 'mock-project',
            model_uri_output_path='/tmp/kfp/output/ml_engine/model_uri.txt',
            model_name_output_path='/tmp/kfp/output/ml_engine/model_name.txt',
            version_name_output_path='/tmp/kfp/output/ml_engine/version_name.txt',
        )

        self.assertEqual(expected_version, result)
        mock_create_version.assert_called_with(
            'projects/mock-project/models/mock-model',
            'gs://model/uri/123/',
            None, # version_name
            None, # runtime_version
            None, # python_version
            None, # version
            False, # replace_existing_version
            30,
            version_name_output_path='/tmp/kfp/output/ml_engine/version_name.txt',
        )

    def test_deploy_set_default_version(self, mock_set_default_version, mock_create_version, 
        mock_create_model, mock_storage_client):
        mock_storage_client().bucket().list_blobs().prefixes = []
        mock_storage_client().bucket().list_blobs().__iter__.return_value = []
        mock_create_model.return_value = {
            'name': 'projects/mock-project/models/mock-model'
        }
        expected_version = {
            'name': 'projects/mock-project/models/mock-model/version/mock-version'
        }
        mock_create_version.return_value = expected_version
        mock_set_default_version.return_value = expected_version

        result = deploy('gs://model/uri', 'mock-project', set_default=True,
            model_uri_output_path='/tmp/kfp/output/ml_engine/model_uri.txt',
            model_name_output_path='/tmp/kfp/output/ml_engine/model_name.txt',
            version_name_output_path='/tmp/kfp/output/ml_engine/version_name.txt',
        )

        self.assertEqual(expected_version, result)
        mock_set_default_version.assert_called_with(
            'projects/mock-project/models/mock-model/version/mock-version')
