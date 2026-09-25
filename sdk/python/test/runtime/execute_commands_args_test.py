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
import shlex
import subprocess
import sys
from typing import Any, Dict

import pytest
import yaml

from ..test_utils.file_utils import FileUtils


@dataclasses.dataclass
class RuntimeTestConfig:
    """A compiled executor and its runtime inputs."""
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


def get_sdk_wheel() -> Path:
    """Require a local SDK wheel rather than falling back to released code."""
    package_path = os.environ.get('KFP_PACKAGE_PATH')
    guidance = ('Build with "uv build --package kfp --wheel" and set '
                'KFP_PACKAGE_PATH to the resulting wheel.')
    if not package_path:
        raise ValueError(f'KFP_PACKAGE_PATH is required. {guidance}')
    wheel = Path(package_path).expanduser().resolve()
    if not (wheel.is_file() and wheel.name.startswith('kfp-') and
            wheel.suffix == '.whl'):
        raise ValueError(
            f'KFP_PACKAGE_PATH must identify an existing kfp wheel: '
            f'{package_path}. {guidance}')
    return wheel


def use_sdk_wheel(command: list[str], wheel: Path) -> list[str]:
    """Replace only the bootstrap's SDK requirement, preserving execution."""
    if command[:2] != ['sh', '-c'] or len(command) < 3:
        raise ValueError(
            'Expected the compiled component installation wrapper.')
    bootstrap, replacements = re.subn(
        r"""(?P<quote>['"])kfp==[^'"\s]+(?P=quote)""",
        lambda match: shlex.quote(str(wheel)),
        command[2],
    )
    if replacements != 1:
        raise ValueError(
            'Expected exactly one pinned kfp requirement in the bootstrap.')
    return command[:2] + [bootstrap] + command[3:]


def run_commands_and_args(
    config: RuntimeTestConfig,
    temp_dir: Path,
    wheel: Path,
    environment: Dict[str, str],
) -> subprocess.CompletedProcess[str]:
    """Execute the compiled command in an isolated, wheel-backed runtime."""
    with open(config.pipeline_file_relpath) as source:
        pipeline_spec_dict = yaml.safe_load(source)
    container = pipeline_spec_dict['deploymentSpec']['executors'][
        config.executor_name]['container']
    command_and_args = use_sdk_wheel(container['command'],
                                     wheel) + container['args']
    executor_input_json = json.dumps(config.executor_input).replace(
        '/gcs/', f'{temp_dir}/')
    command_and_args = [
        value.replace('{{$}}', executor_input_json)
        for value in command_and_args
    ]
    return subprocess.run(
        command_and_args,
        cwd=temp_dir,
        env=environment,
        capture_output=True,
        text=True,
    )


@pytest.mark.regression
class TestRuntimeConfiguration:
    """Keep artifact selection strict and generated shell commands intact."""

    def test_missing_wheel_fails(self, monkeypatch: pytest.MonkeyPatch) -> None:
        monkeypatch.delenv('KFP_PACKAGE_PATH', raising=False)
        with pytest.raises(ValueError, match='KFP_PACKAGE_PATH is required'):
            get_sdk_wheel()

    @pytest.mark.parametrize('package_path', [
        'kfp==2.17.0',
        'git+https://github.com/kubeflow/pipelines.git#subdirectory=sdk/python',
        'kfp-2.17.0-py3-none-any.whl',
        'unrelated-2.17.0-py3-none-any.whl',
        '.',
    ])
    def test_invalid_wheel_fails(
        self,
        package_path: str,
        tmp_path: Path,
        monkeypatch: pytest.MonkeyPatch,
    ) -> None:
        monkeypatch.chdir(tmp_path)
        (tmp_path / 'unrelated-2.17.0-py3-none-any.whl').touch()
        monkeypatch.setenv('KFP_PACKAGE_PATH', package_path)
        with pytest.raises(ValueError, match='existing kfp wheel'):
            get_sdk_wheel()

    @pytest.mark.parametrize('requirement', [
        "'kfp==2.1.3'",
        '"kfp==3.0.0rc1"',
        "'kfp==3.0.0.dev1+local'",
    ])
    def test_only_install_target_changes(
        self,
        requirement: str,
        tmp_path: Path,
        monkeypatch: pytest.MonkeyPatch,
    ) -> None:
        wheel = tmp_path / "SDK wheel's directory" / 'kfp-3.0.0-py3-none-any.whl'
        wheel.parent.mkdir()
        wheel.touch()
        monkeypatch.setenv('KFP_PACKAGE_PATH', str(wheel))
        assert get_sdk_wheel() == wheel
        bootstrap = (f'python3 -m pip install --quiet {requirement} '
                     '\'--no-deps\' && "$0" "$@"')
        command = [
            'sh',
            '-c',
            bootstrap,
            'sh',
            '-ec',
            '_KFP_RUNTIME=true python3 -m kfp.dsl.executor_main "$@"',
            f'print({requirement})',
        ]
        updated = use_sdk_wheel(command, wheel)
        assert updated[2] == bootstrap.replace(requirement,
                                               shlex.quote(str(wheel)))
        assert str(wheel) in shlex.split(updated[2])
        assert updated[:2] == command[:2]
        assert updated[3:] == command[3:]
        assert command[2] == bootstrap

    @pytest.mark.parametrize('command', [
        ['python3', '-m', 'kfp.dsl.executor_main'],
        ['sh', '-c', 'python3 -m pip install requests', "print('kfp==2.1.3')"],
        ['sh', '-c', "python3 -m pip install 'kfp==2.1.3' 'kfp==2.17.0'"],
    ])
    def test_unrecognized_bootstrap_fails(
        self,
        command: list[str],
        tmp_path: Path,
    ) -> None:
        with pytest.raises(ValueError, match='Expected'):
            use_sdk_wheel(command, tmp_path / 'kfp-3.0.0-py3-none-any.whl')


@pytest.mark.regression
class TestRuntime:
    """Run the golden executor commands without an editable SDK or its deps."""

    @pytest.fixture(autouse=True)
    def setup_runtime(self, tmp_path: Path) -> None:
        self.wheel = get_sdk_wheel()
        self.temp_dir = tmp_path
        runtime_venv = tmp_path / 'venv'
        subprocess.run(
            [
                'uv', 'venv', '--python', sys.executable, '--seed',
                str(runtime_venv)
            ],
            check=True,
        )
        self.runtime_python = runtime_venv / 'bin/python'
        self.environment = {
            key: value for key, value in os.environ.items() if key not in {
                'PYTHONPATH', 'PYTHONHOME', 'PIP_TARGET', 'PIP_PREFIX',
                'PIP_USER'
            }
        }
        self.environment.update({
            'PATH': f'{runtime_venv / "bin"}{os.pathsep}{os.environ["PATH"]}',
            'VIRTUAL_ENV': str(runtime_venv),
            'TMPDIR': str(tmp_path),
            'PIP_CONFIG_FILE': os.devnull,
        })

    @pytest.mark.parametrize('config', TEST_CONFIGS)
    def test(self, config: RuntimeTestConfig) -> None:
        process = run_commands_and_args(
            config=config,
            temp_dir=self.temp_dir,
            wheel=self.wheel,
            environment=self.environment,
        )
        assert process.returncode == 0, f"Process failed with error={process.stderr}"
        output_file = Path(
            config.executor_input['outputs']['outputFile'].replace(
                '/gcs/', f'{self.temp_dir}/'))
        assert output_file.is_file()
        installed = subprocess.run(
            [
                str(self.runtime_python), '-I', '-c', '''
import json
from importlib import metadata
print(json.dumps({
    "origin": json.loads(metadata.distribution("kfp").read_text("direct_url.json")),
    "packages": sorted(distribution.metadata["Name"].lower()
                       for distribution in metadata.distributions()),
}))
'''
            ],
            cwd=self.temp_dir,
            env=self.environment,
            capture_output=True,
            text=True,
            check=True,
        )
        installation = json.loads(installed.stdout)
        assert installation['origin']['url'] == self.wheel.as_uri()
        assert set(
            installation['packages']) <= {'kfp', 'pip', 'setuptools', 'wheel'}
