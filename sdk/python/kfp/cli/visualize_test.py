# Copyright 2026 The Kubeflow Authors
# SPDX-License-Identifier: Apache-2.0
"""Native export validation and filesystem/HTTP safety. Browser behavior has a real smoke test."""

import json
import os
import py_compile
import subprocess
import urllib.error
import urllib.request

import pytest
import yaml
from click.testing import CliRunner

from kfp.cli import visualization, visualize


@pytest.fixture
def spec():
    return {
        'apiVersion': 'argoproj.io/v1alpha1',
        'kind': 'WorkflowTemplate',
        'metadata': {'name': 'native'},
        'spec': {
            'entrypoint': 'root',
            'templates': [
                {
                    'name': 'root',
                    'dag': {
                        'tasks': [
                            {'name': 'prepare', 'template': 'step'},
                            {
                                'name': 'train',
                                'template': 'step',
                                'dependencies': ['prepare'],
                            },
                        ]
                    },
                },
                {
                    'name': 'step',
                    'container': {'image': 'busybox', 'command': ['true']},
                },
            ],
        },
    }


@pytest.fixture
def source(tmp_path, spec):
    path = tmp_path / 'pipeline.yaml'
    path.write_text(yaml.safe_dump(spec))
    return path


@pytest.mark.parametrize('suffix', ['.yaml', '.yml', '.json'])
def test_load_native_formats(tmp_path, spec, suffix):
    path = tmp_path / ('pipeline' + suffix)
    path.write_text(
        json.dumps(spec) if suffix == '.json' else yaml.safe_dump(spec)
    )
    assert visualization.load_spec(path) == spec
    assert visualization.build_view(spec)['graphs'][0]['edges'] == [
        {'source': 'prepare', 'target': 'train'}
    ]


@pytest.mark.parametrize(
    'kind', ['Workflow', 'WorkflowTemplate', 'CronWorkflow']
)
def test_resource_kinds(spec, tmp_path, kind):
    spec['kind'] = kind
    if kind == 'CronWorkflow':
        spec['spec'] = {
            'workflowSpec': spec['spec'],
            'schedules': ['0 9 * * *'],
        }
    path = tmp_path / 'resource.yaml'
    path.write_text(yaml.safe_dump(spec))
    assert visualization.load_spec(path)['kind'] == kind


@pytest.mark.parametrize('broken', ['missing', 'cycle', 'recursive', 'steps'])
def test_invalid_native_graphs_fail(spec, broken):
    root = spec['spec']['templates'][0]
    if broken == 'missing':
        root['dag']['tasks'][0]['template'] = 'missing'
    elif broken == 'cycle':
        root['dag']['tasks'][0]['dependencies'] = ['train']
    elif broken == 'recursive':
        root['dag']['tasks'][0]['template'] = 'root'
    else:
        root['steps'] = [[]]
    with pytest.raises(ValueError):
        visualization.validate_spec(spec)


@pytest.mark.parametrize(
    'content', ['not yaml: [', '{}', '{"root":{"dag":{}}}', 'null']
)
def test_invalid_inputs_preserve_existing_exports(source, content):
    output = source.with_suffix('.html')
    output.write_text('previous')
    source.write_text(content)
    result = CliRunner().invoke(
        visualize.visualize, [str(source), '-o', str(output)]
    )
    assert result.exit_code != 0
    assert output.read_text() == 'previous'


@pytest.mark.parametrize(
    'format_,suffix',
    [('html', '.html'), ('markdown', '.md'), ('both', '.html')],
)
def test_export_without_a_cluster(source, format_, suffix):
    result = CliRunner().invoke(
        visualize.visualize, [str(source), '--format', format_]
    )
    assert result.exit_code == 0, result.output
    assert source.with_suffix(suffix).exists()
    if format_ == 'both':
        assert source.with_suffix('.md').exists()


def test_labels_cannot_escape_exports(spec):
    payload = (
        '</script><script>alert(1)</script>```\nclick n0 "javascript:alert(1)"'
    )
    spec['metadata']['name'] = payload
    spec['spec']['templates'][0]['dag']['tasks'][1]['name'] = payload
    markdown = visualization.render_markdown(visualization.build_view(spec))
    assert '<script>' not in markdown
    assert markdown.count('```') == 2
    html = visualization.render_html(
        {'spec': spec, 'error': payload, 'revision': 1}
    )
    assert payload not in html
    assert '\\u003c/script\\u003e' in html
    assert '<script src=' not in html


def test_watch_error_preserves_both_exports_and_recovers(source):
    state = {'current': {'spec': None, 'revision': 0, 'error': None}}
    output = source.with_suffix('.html')
    visualize._update(source, None, output, 'both', state, True)
    html, markdown = output.read_text(), output.with_suffix('.md').read_text()
    before = source.read_text()
    source.write_text('invalid: [')
    visualize._update(source, None, output, 'both', state, True)
    assert state['current']['error']
    assert output.read_text() == html
    assert output.with_suffix('.md').read_text() == markdown
    source.write_text(before)
    visualize._update(source, None, output, 'both', state, True)
    assert state['current']['error'] is None


def test_staging_failure_does_not_replace_either_file(tmp_path, monkeypatch):
    paths = [tmp_path / 'pipeline.html', tmp_path / 'pipeline.md']
    for path in paths:
        path.write_text('previous')
    original = visualize.tempfile.NamedTemporaryFile
    calls = 0

    def staging(*args, **kwargs):
        nonlocal calls
        calls += 1
        if calls == 2:
            raise OSError('disk full')
        return original(*args, **kwargs)

    monkeypatch.setattr(visualize.tempfile, 'NamedTemporaryFile', staging)
    with pytest.raises(OSError, match='disk full'):
        visualize._write_exports(dict.fromkeys(paths, 'new'))
    assert all(path.read_text() == 'previous' for path in paths)
    assert sorted(path.name for path in tmp_path.iterdir()) == [
        'pipeline.html',
        'pipeline.md',
    ]


def test_output_alias_and_symlink_rejection(source):
    with pytest.raises(ValueError, match='overwrite'):
        visualize._output_paths(source, source, 'html')
    link = source.with_suffix('.html')
    link.symlink_to(source)
    with pytest.raises(ValueError, match='overwrite'):
        visualize._output_paths(source, link, 'html')


def test_python_compilation_refreshes_same_size_imports(tmp_path):
    helper = tmp_path / 'helper.py'
    helper.write_text('NAME = "first"\n')
    source = tmp_path / 'pipeline.py'
    source.write_text(
        'from kfp import dsl\nfrom helper import NAME\n@dsl.container_component\ndef step():\n return dsl.ContainerSpec(image="busybox",command=["echo",NAME])\n@dsl.pipeline(name=NAME)\ndef pipeline():\n step()\n'
    )
    assert (
        visualization.load_spec(source)['metadata']['annotations'][
            'pipelines.kubeflow.org/display-name'
        ]
        == 'first'
    )
    stamp = helper.stat().st_mtime
    py_compile.compile(str(helper))
    helper.write_text('NAME = "other"\n')
    os.utime(helper, (stamp, stamp))
    assert (
        visualization.load_spec(source)['metadata']['annotations'][
            'pipelines.kubeflow.org/display-name'
        ]
        == 'other'
    )


def test_json_is_json_and_preserves_unicode(source, spec):
    path = source.with_suffix('.json')
    spec['metadata']['name'] = '\U0001f680'
    path.write_text(json.dumps(spec))
    assert visualization.load_spec(path)['metadata']['name'] == '\U0001f680'
    path.write_text(yaml.safe_dump(spec))
    with pytest.raises(ValueError):
        visualization.load_spec(path)


def test_timeout_is_reported(source, monkeypatch):
    source = source.with_suffix('.py')
    source.write_text('# trusted test')
    monkeypatch.setattr(
        visualization.subprocess,
        'run',
        lambda *args, **kwargs: (_ for _ in ()).throw(
            subprocess.TimeoutExpired('python', 60)
        ),
    )
    with pytest.raises(subprocess.TimeoutExpired):
        visualization.load_spec(source)


def test_fingerprint_tracks_imports_not_outputs(tmp_path):
    source = tmp_path / 'pipeline.py'
    source.write_text('# pipeline')
    before = visualize._fingerprint(source)
    (tmp_path / 'output.html').write_text('output')
    assert visualize._fingerprint(source) == before
    helper = tmp_path / 'helper.py'
    helper.write_text('# helper')
    assert visualize._fingerprint(source) != before
    helper.unlink()
    assert visualize._fingerprint(source) == before


@pytest.mark.parametrize('live', [True, False])
def test_loopback_server_rejects_other_paths_and_hosts(spec, live):
    state = {'current': {'spec': spec, 'revision': 1, 'error': None}}
    with visualize._live_server(state, live=live) as url:
        assert urllib.request.urlopen(url).status == 200
        assert (
            json.load(urllib.request.urlopen(url + 'state')) == state['current']
        )
        for path in ('../', 'secret', '../../../etc/passwd'):
            with pytest.raises(urllib.error.HTTPError) as error:
                urllib.request.urlopen(url + path)
            assert error.value.code == 404
        with pytest.raises(urllib.error.HTTPError) as error:
            urllib.request.urlopen(
                urllib.request.Request(
                    url, headers={'Host': 'attacker.invalid'}
                )
            )
        assert error.value.code == 403
