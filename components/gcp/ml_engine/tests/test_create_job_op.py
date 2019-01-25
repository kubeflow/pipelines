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

from ml_engine.create_job_op import CreateJobOp

@mock.patch('kubernetes.config.load_incluster_config')
@mock.patch('kubernetes.client.CoreV1Api')
@mock.patch('ml_engine.mlengine_client.MLEngineClient')
class TestCreateJobOp(unittest.TestCase):

    def test_init(self, mock_mlengine_client,
        mock_core_v1_api, mock_load_incluster_config):
        job = {
            'jobId': 'mock_job'
        }
        op = CreateJobOp('mock_project', job)

        self.assertEqual('mock_project', op._project_id)
        self.assertEqual(job['jobId'], op._job_id)
        self.assertEqual(job, op._job)

    def test_execute_submit_job(self, mock_mlengine_client,
        mock_core_v1_api, mock_load_incluster_config):
        job = {
            'jobId': 'mock_job'
        }
        mock_mlengine_client().get_job.return_value = {
            'state': 'SUCCEEDED'
        }
        op = CreateJobOp('mock_project', job)

        op.execute()



