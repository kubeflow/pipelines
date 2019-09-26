
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

from kfp_component.google.dataproc import create_cluster

MODULE = 'kfp_component.google.dataproc._create_cluster'

@mock.patch(MODULE + '.display.display')
@mock.patch(MODULE + '.gcp_common.dump_file')
@mock.patch(MODULE + '.KfpExecutionContext')
@mock.patch(MODULE + '.DataprocClient')
class TestCreateCluster(unittest.TestCase):

    def test_create_cluster_succeed(self, mock_dataproc_client,
        mock_kfp_context, mock_dump_json, mock_display):
        mock_kfp_context().__enter__().context_id.return_value = 'ctx1'
        cluster = {
            'projectId': 'mock_project',
            'config': {},
            'clusterName': 'cluster-ctx1'
        }
        mock_dataproc_client().wait_for_operation_done.return_value = (
            {
                'response': cluster
            })

        result = create_cluster('mock_project', 'mock-region')

        self.assertEqual(cluster, result)
        mock_dataproc_client().create_cluster.assert_called_with(
            'mock_project',
            'mock-region',
            cluster,
            request_id = 'ctx1')
    
    def test_create_cluster_with_specs_succeed(self, mock_dataproc_client,
        mock_kfp_context, mock_dump_json, mock_display):
        mock_kfp_context().__enter__().context_id.return_value = 'ctx1'
        cluster = {
            'projectId': 'mock_project',
            'config': {
                'initializationActions': [
                    {
                        'executableFile': 'gs://action/1'
                    },
                    {
                        'executableFile': 'gs://action/2'
                    }
                ],
                'configBucket': 'gs://config/bucket',
                'softwareConfig': {
                    'imageVersion': '1.10'
                },
            },
            'labels': {
                'label-1': 'value-1'
            },
            'clusterName': 'test-cluster'
        }
        mock_dataproc_client().wait_for_operation_done.return_value = (
            {
                'response': cluster
            })

        result = create_cluster('mock_project', 'mock-region',
            name='test-cluster', 
            initialization_actions=['gs://action/1', 'gs://action/2'],
            config_bucket='gs://config/bucket',
            image_version='1.10',
            cluster={
                'labels':{
                    'label-1': 'value-1'
                }
            })

        self.assertEqual(cluster, result)
        mock_dataproc_client().create_cluster.assert_called_with(
            'mock_project',
            'mock-region',
            cluster,
            request_id = 'ctx1')

    def test_create_cluster_name_prefix_succeed(self, mock_dataproc_client,
        mock_kfp_context, mock_dump_json, mock_display):
        mock_kfp_context().__enter__().context_id.return_value = 'ctx1'
        cluster = {
            'projectId': 'mock_project',
            'config': {},
            'clusterName': 'my-cluster-ctx1'
        }
        mock_dataproc_client().wait_for_operation_done.return_value = (
            {
                'response': cluster
            })

        result = create_cluster('mock_project', 'mock-region', 
            name_prefix='my-cluster')

        self.assertEqual(cluster, result)
        mock_dataproc_client().create_cluster.assert_called_with(
            'mock_project',
            'mock-region',
            cluster,
            request_id = 'ctx1')

    def test_cancel_succeed(self, mock_dataproc_client,
        mock_kfp_context, mock_dump_json, mock_display):
        mock_kfp_context().__enter__().context_id.return_value = 'ctx1'
        operation = {
            'name': 'mock_operation'
        }
        mock_dataproc_client().create_cluster.return_value = (
            operation)
        cluster = {
            'projectId': 'mock_project',
            'config': {},
            'clusterName': 'my-cluster-ctx1'
        }
        mock_dataproc_client().wait_for_operation_done.return_value = (
            {
                'response': cluster
            })
        create_cluster('mock_project', 'mock-region')
        cancel_func = mock_kfp_context.call_args[1]['on_cancel']
        
        cancel_func()

        mock_dataproc_client().cancel_operation.assert_called_with(
            'mock_operation'
        )
