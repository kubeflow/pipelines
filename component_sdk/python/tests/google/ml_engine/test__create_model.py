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
from kfp_component.google.ml_engine import create_model

CREATE_MODEL_MODULE = 'kfp_component.google.ml_engine._create_model'

@mock.patch(CREATE_MODEL_MODULE + '.display.display')
@mock.patch(CREATE_MODEL_MODULE + '.gcp_common.dump_file')
@mock.patch(CREATE_MODEL_MODULE + '.KfpExecutionContext')
@mock.patch(CREATE_MODEL_MODULE + '.MLEngineClient')
class TestCreateModel(unittest.TestCase):

    def test_create_model_succeed(self, mock_mlengine_client,
        mock_kfp_context, mock_dump_json, mock_display):
        model = {
            'name': 'mock_model',
            'description': 'the mock model'
        }
        mock_mlengine_client().create_model.return_value = model

        result = create_model('mock_project', 'mock_model', model)

        self.assertEqual(model, result)

    def test_create_model_conflict_succeed(self, mock_mlengine_client,
        mock_kfp_context, mock_dump_json, mock_display):
        model = {
            'name': 'mock_model',
            'description': 'the mock model'
        }
        mock_mlengine_client().create_model.side_effect = errors.HttpError(
            resp = mock.Mock(status=409),
            content = b'conflict'
        )
        mock_mlengine_client().get_model.return_value = model

        result = create_model('mock_project', 'mock_model', model)

        self.assertEqual(model, result)

    def test_create_model_conflict_fail(self, mock_mlengine_client,
        mock_kfp_context, mock_dump_json, mock_display):
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

        with self.assertRaises(errors.HttpError) as context:
            create_model('mock_project', 'mock_model', model)

        self.assertEqual(409, context.exception.resp.status)

    def test_create_model_use_context_id_as_name(self, mock_mlengine_client,
        mock_kfp_context, mock_dump_json, mock_display):
        context_id = 'context1'
        model = {}
        returned_model = {
            'name': 'model_' + context_id
        }
        mock_mlengine_client().create_model.return_value = returned_model
        mock_kfp_context().__enter__().context_id.return_value = context_id

        create_model('mock_project', model=model)

        mock_mlengine_client().create_model.assert_called_with(
            project_id = 'mock_project',
            model = returned_model
        )