# Copyright 2023 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
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
import dataclasses
import json
import os
import re
import shutil
import subprocess
import tempfile
from typing import Any, Dict

from absl.testing import parameterized
import yaml

TEST_DATA_DIR = os.path.join(os.path.dirname(__file__), 'test_data')


@dataclasses.dataclass
class RuntimeTestConfig:
    pipeline_file_relpath: str
    executor_name: str
    executor_input: Dict[str, Any]


TEST_CONFIGS = [
    RuntimeTestConfig(
        pipeline_file_relpath=os.path.join(
            TEST_DATA_DIR, 'pipeline_with_task_final_status.yaml'),
        executor_name='exec-print-op',
        executor_input={
            'inputs': {
                'parameterValues': {
                    'message': 'Hello World!'
                },
                'parameters': {
                    'message': {
                        'stringValue': 'Hello World!'
                    }
                }
            },
            'outputs': {
                'outputFile':
                    '/gcs/cjmccarthy-kfp-default-bucket/271009669852/pipeline-with-task-final-status-07-14-2023-18-50-32/print-op_-9063136771365142528/executor_output.json'
            }
        },
    ),
    RuntimeTestConfig(
        pipeline_file_relpath=os.path.join(
            TEST_DATA_DIR, 'pipeline_with_task_final_status.yaml'),
        executor_name='exec-exit-op',
        executor_input={
            'inputs': {
                'parameterValues': {
                    'status': {
                        'error': {
                            'code':
                                9,
                            'message':
                                'The DAG failed because some tasks failed. The failed tasks are: [print-op, fail-op].'
                        },
                        'pipelineJobResourceName':
                            'projects/271009669852/locations/us-central1/pipelineJobs/pipeline-with-task-final-status-07-14-2023-19-07-11',
                        'pipelineTaskName':
                            'my-pipeline',
                        'state':
                            'FAILED'
                    },
                    'user_input': 'Hello World!'
                },
                'parameters': {
                    'status': {
                        'stringValue':
                            "{\"error\":{\"code\":9,\"message\":\"The DAG failed because some tasks failed. The failed tasks are: [print-op, fail-op].\"},\"pipelineJobResourceName\":\"projects/271009669852/locations/us-central1/pipelineJobs/pipeline-with-task-final-status-07-14-2023-19-07-11\",\"pipelineTaskName\":\"my-pipeline\",\"state\":\"FAILED\"}"
                    },
                    'user_input': {
                        'stringValue': 'Hello World!'
                    }
                }
            },
            'outputs': {
                'outputFile':
                    '/gcs/cjmccarthy-kfp-default-bucket/271009669852/pipeline-with-task-final-status-07-14-2023-19-07-11/exit-op_-6100894116462198784/executor_output.json'
            }
        },
    )
]


def run_commands_and_args(
    config: RuntimeTestConfig,
    temp_dir: str,
) -> subprocess.CompletedProcess:
    with open(config.pipeline_file_relpath) as f:
        pipline_spec_dict = yaml.safe_load(f)
        container = pipline_spec_dict['deploymentSpec']['executors'][
            config.executor_name]['container']

        command_and_args = container['command'] + container['args']
        # https://docs.prow.k8s.io/docs/jobs/#job-environment-variables
        # pip install from source in a container via a subprocess causes many
        # permissions issue
        # resolving by modifying the commands/args changes the commands/args
        # so much that it renders the test less valuable, since the
        # commands/args resemble the true runtime commands/args less well
        # prefer the less invasive approach of installing from a PR

        kfp_package_path = 'sdk/python'
        command_and_args = [
            re.sub(r"'kfp==(\d+).(\d+).(\d+)(-[a-z]+.\d+)?'", kfp_package_path,
                   cmd) for cmd in command_and_args
        ]
        executor_input_json = json.dumps(config.executor_input).replace(
            '/gcs/', temp_dir)
        command_and_args = [
            v.replace('{{$}}', executor_input_json) for v in command_and_args
        ]

        return subprocess.run(
            command_and_args,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
        )


class TestRuntime(parameterized.TestCase):

    @classmethod
    def setUp(cls):
        cls.temp_dir = tempfile.mkdtemp()

    @classmethod
    def tearDown(cls):
        shutil.rmtree(cls.temp_dir)

    @parameterized.parameters(TEST_CONFIGS)
    def test(self, config: RuntimeTestConfig):
        process = run_commands_and_args(
            config=config,
            temp_dir=self.temp_dir,
        )
        self.assertEqual(process.returncode, 0, process.stderr)
