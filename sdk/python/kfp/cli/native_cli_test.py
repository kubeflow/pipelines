# Copyright 2026 The Kubeflow Authors
# SPDX-License-Identifier: Apache-2.0

from pathlib import Path

import pytest
import yaml
from click.testing import CliRunner

from kfp.cli import visualization
from kfp.cli.compile_ import compile_
from kfp.dsl.pipeline_context import Pipeline

ROOT = Path(__file__).resolve().parents[4]


def test_cli_native_resource_options(tmp_path):
    path = tmp_path / 'cron.yaml'
    before = Pipeline.get_execution_caching_default()
    result = CliRunner().invoke(
        compile_,
        [
            '--py',
            str(ROOT / 'samples/argo-native/components.py'),
            '--output',
            str(path),
            '--resource-kind',
            'CronWorkflow',
            '--schedule',
            '0 9 * * *',
            '--namespace',
            'test',
            '--disable-caching',
        ],
    )
    assert result.exit_code == 0, result.output
    manifest = yaml.safe_load(path.read_text())
    assert manifest['kind'] == 'CronWorkflow'
    assert manifest['metadata']['namespace'] == 'test'
    assert manifest['spec']['schedules'] == ['0 9 * * *']
    assert Pipeline.get_execution_caching_default() == before


def test_python_preview_is_native_and_does_not_require_image_digests():
    spec = visualization.load_spec(ROOT / 'samples/argo-native/components.py')
    assert spec['kind'] == 'WorkflowTemplate'
    assert 'root' not in spec
    view = visualization.build_view(spec)
    assert [node['label'] for node in view['graphs'][0]['nodes']] == [
        'prepare',
        'train',
        'evaluate',
    ]
    assert 'prepare' in visualization.render_markdown(view)


def test_native_preview_rejects_recursive_templates():
    spec = {
        'kind': 'Workflow',
        'metadata': {'name': 'bad'},
        'spec': {
            'entrypoint': 'root',
            'templates': [
                {
                    'name': 'root',
                    'dag': {'tasks': [{'name': 'loop', 'template': 'root'}]},
                }
            ],
        },
    }
    with pytest.raises(ValueError, match='recursive'):
        visualization.validate_spec(spec)
