# Copyright 2026 The Kubeflow Authors
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

import os
from pathlib import Path
import subprocess

from kfp import compiler
from kfp import components
from kfp import dsl
import pytest
import yaml

from backend.src.v2.test.components import download_gcs_tgz
from backend.src.v2.test.components import kaniko
from backend.src.v2.test.components import run_sample

COMPONENTS_DIR = Path(__file__).parent


@pytest.mark.parametrize('module', [download_gcs_tgz, kaniko, run_sample])
def test_native_component_compiles_and_reloads_ir(module, tmp_path):
    name = module.__name__.split('.')[-1]
    component = getattr(module, name)
    generated = tmp_path / f'{name}.yaml'
    compiler.Compiler().compile(component, package_path=str(generated))

    expected = yaml.safe_load((COMPONENTS_DIR / f'{name}.yaml').read_text())
    actual = yaml.safe_load(generated.read_text())
    assert actual == expected
    assert actual['schemaVersion'] == '2.1.0'
    assert 'implementation' not in actual

    loaded = components.load_component_from_file(str(generated))
    assert loaded.component_spec.inputs == component.component_spec.inputs
    assert loaded.component_spec.outputs == component.component_spec.outputs
    containers = [
        executor['container']
        for executor in actual['deploymentSpec']['executors'].values()
    ]
    assert len(containers) == 1
    assert containers[0][
        'image'] == component.component_spec.implementation.container.image
    assert containers[0]['command'][:3] == ['sh', '-exc', module.SHELL_COMMAND]


def test_ir_components_compose_with_uri_and_artifact_context(tmp_path):
    download = components.load_component_from_file(
        str(COMPONENTS_DIR / 'download_gcs_tgz.yaml'))
    build = components.load_component_from_file(
        str(COMPONENTS_DIR / 'kaniko.yaml'))
    run = components.load_component_from_file(
        str(COMPONENTS_DIR / 'run_sample.yaml'))

    @dsl.pipeline
    def sample_runner():
        folder = download(gcs_path='gs://bucket/source.tar.gz')
        build(
            dockerfile='Dockerfile',
            destination='registry.example/image',
            context_artifact=folder.outputs['folder'])
        build(
            dockerfile='Dockerfile',
            destination='registry.example/other-image',
            context_uri='gs://bucket/source.tar.gz')
        binary = dsl.importer(
            artifact_uri='gs://bucket/compiler', artifact_class=dsl.Artifact)
        run(name='hello',
            sample_path='samples.v2.hello_world',
            gcs_root='gs://bucket/output',
            external_host='http://pipeline-ui',
            backend_compiler=binary.output)

    path = tmp_path / 'runner.yaml'
    compiler.Compiler().compile(sample_runner, package_path=str(path))
    pipeline = yaml.safe_load(path.read_text())
    assert len(pipeline['root']['dag']['tasks']) == 5
    assert build.component_spec.inputs['context_artifact'].optional
    assert build.component_spec.inputs['cache'].default == 'true'
    assert build.component_spec.inputs['cache_ttl'].default == '24h'
    assert run.component_spec.inputs[
        'host'].default == 'http://ml-pipeline:8888'


def test_sample_runner_uses_uv_workspace_and_preserves_arguments(tmp_path):
    repo = tmp_path / 'checkout'
    repo.mkdir()
    source_root = COMPONENTS_DIR.parents[4]
    for name in ('pyproject.toml', 'uv.lock'):
        (repo / name).write_text((source_root / name).read_text())
    (repo / 'sdk/python').mkdir(parents=True)
    binaries = tmp_path / 'bin'
    binaries.mkdir()
    scripts = {
        'chmod':
            'exit 0\n',
        'cp':
            'exit 0\n',
        'pip':
            'printf "%s\\n" "$@" > "$TRACE_DIR/install"\n',
        'uv': ('test -f pyproject.toml\n'
               'test -f uv.lock\n'
               'test -d sdk/python\n'
               'printf "%s\\n" "$KF_PIPELINES_ENDPOINT" '
               '"$KF_PIPELINES_UI_ENDPOINT" "$@" > "$TRACE_DIR/run"\n'),
    }
    for name, body in scripts.items():
        executable = binaries / name
        executable.write_text('#!/bin/sh\nset -eu\n' + body)
        executable.chmod(0o755)
    env = dict(
        os.environ,
        PATH=f'{binaries}:{os.environ["PATH"]}',
        TRACE_DIR=str(tmp_path))
    subprocess.run([
        'sh', '-ec', run_sample.SHELL_COMMAND, '/artifacts/compiler',
        'samples.v2.hello_world', 'gs://bucket/output/hello',
        'http://ml-pipeline:8888', 'http://pipeline-ui', 'launcher:test',
        'driver:test'
    ],
                   cwd=repo,
                   env=env,
                   check=True)
    assert (tmp_path / 'install').read_text().splitlines() == ['install', 'uv']
    assert (tmp_path / 'run').read_text().splitlines() == [
        'http://ml-pipeline:8888', 'http://pipeline-ui', 'run', '--frozen',
        '--extra', 'backend-v2-test', 'python3', '-u', '-m',
        'samples.v2.hello_world', '--pipeline_root', 'gs://bucket/output/hello',
        '--launcher_v2_image', 'launcher:test', '--driver_image', 'driver:test'
    ]
