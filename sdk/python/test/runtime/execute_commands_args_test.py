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
from pathlib import Path
import re
import shutil
import subprocess
import tempfile
from typing import Any, Dict

import kfp
from kfp import compiler
from kfp import dsl
import pytest
import yaml

from ..test_utils.file_utils import FileUtils


@dataclasses.dataclass
class RuntimeTestConfig:
    pipeline_file_relpath: str
    executor_name: str
    executor_input: Dict[str, Any]


TEST_CONFIGS = [
    RuntimeTestConfig(
        pipeline_file_relpath=os.path.join(
            FileUtils.VALID_PIPELINE_FILES,
            'pipeline_with_task_final_status.yaml'),
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
            FileUtils.VALID_PIPELINE_FILES,
            'pipeline_with_task_final_status.yaml'),
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

        overriden_kfp_package_path_commands = list()
        for cmd in command_and_args:
            if re.search("kfp==[0-9].[0-9]+.[0-9]+", cmd) is not None:
                cmd = re.sub("kfp==[0-9].[0-9]+.[0-9]+",
                             f"kfp=={kfp.__version__}", cmd)
            overriden_kfp_package_path_commands.append(cmd)
        command_and_args = overriden_kfp_package_path_commands
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


@pytest.mark.regression
class TestRuntime:

    @pytest.fixture(autouse=True)
    def setup_and_teardown(self):
        self.temp_dir = tempfile.mkdtemp()
        yield
        shutil.rmtree(self.temp_dir)

    @pytest.mark.parametrize('config', TEST_CONFIGS)
    def test(self, config: RuntimeTestConfig):
        process = run_commands_and_args(
            config=config,
            temp_dir=self.temp_dir,
        )
        assert process.returncode == 0, f"Process failed with error={process.stderr}"

    @pytest.mark.parametrize(
        ('pipeline_enabled', 'component_enabled', 'env_passthrough'), [
            (True, True, False),
            (True, False, False),
            (False, True, False),
            (False, False, False),
            (False, True, True),
        ])
    def test_compiled_artifact_preview_policy(self, pipeline_enabled: bool,
                                              component_enabled: bool,
                                              env_passthrough: bool) -> None:
        """Execute the compiled preview policy.

        The harness runs compiled commands and arguments without task
        environment variables, as capture-only TaskConfig ENV
        passthrough does.
        """

        @dsl.component(
            base_image='python:3.11',
            artifact_previews=component_enabled,
            task_config_passthroughs=[dsl.TaskConfigField.ENV]
            if env_passthrough else None)
        def produce(dataset: dsl.Output[dsl.Dataset]) -> None:
            from pathlib import Path

            directory = Path(dataset.path)
            directory.mkdir(parents=True, exist_ok=True)
            (directory / 'data.csv').write_text('value\n1\n')

        @dsl.pipeline(
            pipeline_config=dsl.PipelineConfig(
                artifact_previews=pipeline_enabled))
        def pipeline() -> None:
            task = produce()
            if env_passthrough:
                task.set_env_variable('UNRELATED', 'captured')

        temp_dir = Path(self.temp_dir)
        package = temp_dir / 'preview-policy.yaml'
        compiler.Compiler().compile(pipeline, str(package))
        dataset = temp_dir / 'dataset'
        output_file = temp_dir / 'executor_output.json'
        config = RuntimeTestConfig(
            pipeline_file_relpath=str(package),
            executor_name='exec-produce',
            executor_input={
                'inputs': {},
                'outputs': {
                    'artifacts': {
                        'dataset': {
                            'artifacts': [{
                                'uri': str(dataset),
                                'type': {
                                    'schemaTitle': 'system.Dataset'
                                },
                            }],
                        },
                    },
                    'outputFile': str(output_file),
                },
            })
        process = run_commands_and_args(config, self.temp_dir)
        assert process.returncode == 0, process.stderr
        assert (dataset / 'data.csv').read_text() == 'value\n1\n'
        metadata = json.loads(output_file.read_text(
        ))['artifacts']['dataset']['artifacts'][0]['metadata']
        enabled = pipeline_enabled and component_enabled
        assert ('_kfp_preview' in metadata) is enabled
        assert (dataset / '_kfp_preview.json').exists() is enabled
        if enabled:
            assert json.loads(
                (dataset / '_kfp_preview.json').read_text())['rows'] == [['1']]
