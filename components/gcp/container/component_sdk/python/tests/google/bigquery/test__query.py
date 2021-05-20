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

from google.cloud import bigquery
from google.cloud.bigquery.job import ExtractJobConfig, DestinationFormat
from google.api_core import exceptions
from kfp_component.google.bigquery import query

CREATE_JOB_MODULE = 'kfp_component.google.bigquery._query'

@mock.patch(CREATE_JOB_MODULE + '.display.display')
@mock.patch(CREATE_JOB_MODULE + '.gcp_common.dump_file')
@mock.patch(CREATE_JOB_MODULE + '.KfpExecutionContext')
@mock.patch(CREATE_JOB_MODULE + '.bigquery.Client')
class TestQuery(unittest.TestCase):

    def test_query_succeed(self, mock_client,
        mock_kfp_context, mock_dump_json, mock_display):
        mock_kfp_context().__enter__().context_id.return_value = 'ctx1'
        mock_client().get_job.side_effect = exceptions.NotFound('not found')
        mock_dataset = bigquery.DatasetReference('project-1', 'dataset-1')
        mock_client().dataset.return_value = mock_dataset
        mock_client().get_dataset.side_effect = exceptions.NotFound('not found')
        mock_response = {
            'configuration': {
                'query': {
                    'query': 'SELECT * FROM table_1'
                }
            }
        }
        mock_client().query.return_value.to_api_repr.return_value = mock_response

        result = query('SELECT * FROM table_1', 'project-1', 'dataset-1', 
            output_gcs_path='gs://output/path')

        self.assertEqual(mock_response, result)
        mock_client().create_dataset.assert_called()
        expected_job_config = bigquery.QueryJobConfig()
        expected_job_config.create_disposition = bigquery.job.CreateDisposition.CREATE_IF_NEEDED
        expected_job_config.write_disposition = bigquery.job.WriteDisposition.WRITE_TRUNCATE
        expected_job_config.destination = mock_dataset.table('query_ctx1')
        mock_client().query.assert_called_with('SELECT * FROM table_1',mock.ANY,
            job_id = 'query_ctx1')
        actual_job_config = mock_client().query.call_args_list[0][0][1]
        self.assertDictEqual(
            expected_job_config.to_api_repr(),
            actual_job_config.to_api_repr()
        )
        extract = mock_client().extract_table.call_args_list[0]
        self.assertEqual(extract[0], (mock_dataset.table('query_ctx1'), 'gs://output/path',))
        self.assertEqual(extract[1]["job_config"].destination_format, "CSV",)

    def test_query_no_output_path(self, mock_client,
        mock_kfp_context, mock_dump_json, mock_display):
        mock_kfp_context().__enter__().context_id.return_value = 'ctx1'
        mock_client().get_job.side_effect = exceptions.NotFound('not found')
        mock_dataset = bigquery.DatasetReference('project-1', 'dataset-1')
        mock_client().dataset.return_value = mock_dataset
        mock_client().get_dataset.return_value = bigquery.Dataset(mock_dataset)
        mock_response = {
            'configuration': {
                'query': {
                    'query': 'SELECT * FROM table_1'
                }
            }
        }
        mock_client().query.return_value.to_api_repr.return_value = mock_response

        result = query('SELECT * FROM table_1', 'project-1', 'dataset-1', 'table-1')

        self.assertEqual(mock_response, result)
        mock_client().create_dataset.assert_not_called()
        mock_client().extract_table.assert_not_called()

        expected_job_config = bigquery.QueryJobConfig()
        expected_job_config.create_disposition = bigquery.job.CreateDisposition.CREATE_IF_NEEDED
        expected_job_config.write_disposition = bigquery.job.WriteDisposition.WRITE_TRUNCATE
        expected_job_config.destination = mock_dataset.table('table-1')
        mock_client().query.assert_called_with('SELECT * FROM table_1',mock.ANY,
            job_id = 'query_ctx1')
        actual_job_config = mock_client().query.call_args_list[0][0][1]
        self.assertDictEqual(
            expected_job_config.to_api_repr(),
            actual_job_config.to_api_repr()
        )

    def test_query_output_json_format(self, mock_client,
                           mock_kfp_context, mock_dump_json, mock_display):
        mock_kfp_context().__enter__().context_id.return_value = 'ctx1'
        mock_client().get_job.side_effect = exceptions.NotFound('not found')
        mock_dataset = bigquery.DatasetReference('project-1', 'dataset-1')
        mock_client().dataset.return_value = mock_dataset
        mock_client().get_dataset.side_effect = exceptions.NotFound('not found')
        mock_response = {
            'configuration': {
                'query': {
                    'query': 'SELECT * FROM table_1'
                }
            }
        }
        mock_client().query.return_value.to_api_repr.return_value = mock_response

        result = query('SELECT * FROM table_1', 'project-1', 'dataset-1',
                       output_gcs_path='gs://output/path',
                       output_destination_format="NEWLINE_DELIMITED_JSON")

        self.assertEqual(mock_response, result)
        mock_client().create_dataset.assert_called()
        extract = mock_client().extract_table.call_args_list[0]
        self.assertEqual(extract[0], (mock_dataset.table('query_ctx1'), 'gs://output/path',))
        self.assertEqual(extract[1]["job_config"].destination_format, "NEWLINE_DELIMITED_JSON",)
