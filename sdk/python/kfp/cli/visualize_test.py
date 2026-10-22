# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import json
import os
import py_compile
import subprocess
import urllib.error
import urllib.request
from pathlib import Path

import pytest
import yaml
from click.testing import CliRunner

from kfp.cli import cli, visualization, visualize


@pytest.fixture
def spec():
    return {
        'pipelineInfo': {'name': 'example'},
        'schemaVersion': '2.1.0',
        'root': {
            'dag': {
                'tasks': {
                    'load': {
                        'componentRef': {'name': 'step'},
                        'taskInfo': {'name': 'Load data'},
                    },
                    'train': {
                        'componentRef': {'name': 'step'},
                        'taskInfo': {'name': 'Train'},
                        'dependentTasks': ['load'],
                        'inputs': {
                            'parameters': {
                                'data': {
                                    'taskOutputParameter': {
                                        'producerTask': 'load',
                                        'outputParameterKey': 'data',
                                    }
                                }
                            }
                        },
                    },
                }
            }
        },
        'components': {'step': {'executorLabel': 'exec-step'}},
    }


@pytest.fixture
def source(tmp_path, spec):
    path = tmp_path / 'pipeline.yaml'
    path.write_text(yaml.safe_dump(spec))
    return path


def test_task_dependencies_and_nested_dags(spec):
    spec['components']['nested'] = {
        'dag': {
            'tasks': {
                'child': {
                    'componentRef': {'name': 'step'},
                    'taskInfo': {'name': 'Child'},
                }
            }
        }
    }
    spec['root']['dag']['tasks']['loop'] = {
        'taskInfo': {'name': 'Loop'},
        'componentRef': {'name': 'nested'},
        'parameterIterator': {},
        'inputs': {
            'artifacts': {
                'model': {
                    'taskOutputArtifact': {
                        'producerTask': 'train',
                        'outputArtifactKey': 'model',
                    }
                }
            }
        },
    }
    spec['root']['dag']['tasks']['condition'] = {
        'taskInfo': {'name': 'Condition'},
        'componentRef': {'name': 'nested'},
        'triggerPolicy': {'condition': 'true'},
    }
    view = visualization.build_view(spec)
    assert len(view['graphs']) == 3
    root = view['graphs'][0]
    assert root['edges'] == [
        {'source': 'load', 'target': 'train'},
        {'source': 'train', 'target': 'loop'},
    ]
    nodes = {node['id']: node for node in root['nodes']}
    assert nodes['loop']['kind'] == 'Loop'
    assert nodes['condition']['kind'] == 'Condition'
    assert {graph['label'] for graph in view['graphs'][1:]} == {
        'loop',
        'condition',
    }
    markdown = visualization.render_markdown(view)
    assert markdown.count('```mermaid') == 3
    assert markdown.count('-->') == 2


def test_graph_order_is_stable_across_spec_key_reordering(spec):
    expected = visualization.build_view(spec)
    spec['root']['dag']['tasks'] = dict(
        reversed(list(spec['root']['dag']['tasks'].items()))
    )
    assert visualization.build_view(spec) == expected


def test_final_status_dependency_and_runtime_constants(spec):
    task = spec['root']['dag']['tasks']['train']
    task.pop('dependentTasks')
    task['inputs'] = {
        'parameters': {
            'status': {'taskFinalStatus': {'producerTask': 'load'}},
            'config': {
                'runtimeValue': {
                    'constant': {
                        'taskOutputParameter': {'producerTask': 'not-a-task'}
                    }
                }
            },
        }
    }
    assert visualization.build_view(spec)['graphs'][0]['edges'] == [
        {'source': 'load', 'target': 'train'}
    ]


@pytest.mark.parametrize(
    'broken', ['component', 'dependency', 'cycle', 'recursive']
)
def test_invalid_graphs_rejected(spec, broken):
    tasks = spec['root']['dag']['tasks']
    if broken == 'component':
        tasks['load']['componentRef']['name'] = 'missing'
    elif broken == 'dependency':
        tasks['load']['dependentTasks'] = ['missing']
    elif broken == 'cycle':
        tasks['load']['dependentTasks'] = ['train']
    else:
        spec['components']['recursive'] = {
            'dag': {
                'tasks': {
                    'recurse': {
                        'componentRef': {'name': 'recursive'},
                        'taskInfo': {'name': 'Recurse'},
                    }
                }
            }
        }
        tasks['load']['componentRef']['name'] = 'recursive'
    messages = {
        'component': 'missing component',
        'dependency': 'invalid dependency',
        'cycle': 'Cyclic task dependencies',
        'recursive': 'Recursive DAG components',
    }
    with pytest.raises(ValueError, match=messages[broken]):
        visualization.build_view(spec)


def _reused_dag_spec(depth):
    components = {'step': {'executorLabel': 'exec-step'}}
    reference = 'step'
    for index in range(depth):
        name = f'dag-{index}'
        components[name] = {
            'dag': {
                'tasks': {
                    key: {
                        'componentRef': {'name': reference},
                        'taskInfo': {'name': key},
                    }
                    for key in ('left', 'right')
                }
            }
        }
        reference = name
    return {
        'pipelineInfo': {'name': 'reused-dags'},
        'deploymentSpec': {
            'executors': {'exec-step': {'container': {'image': 'alpine'}}}
        },
        'components': components,
        'root': {
            'dag': {
                'tasks': {
                    'start': {
                        'componentRef': {'name': reference},
                        'taskInfo': {'name': 'Start'},
                    }
                }
            }
        },
    }


def test_html_validates_reusable_components_once(tmp_path, monkeypatch):
    depth = 30
    source = tmp_path / 'reused.json'
    source.write_text(json.dumps(_reused_dag_spec(depth)))
    producers = visualization._producers
    calls = 0

    def count_visits(inputs):
        nonlocal calls
        calls += 1
        # Bound the old implementation too: fail before it exhausts memory.
        assert calls <= 2 * depth + 1, 'Repeated validation of reusable DAGs'
        return producers(inputs)

    monkeypatch.setattr(visualization, '_producers', count_visits)
    result = CliRunner().invoke(visualize.visualize, [str(source)])
    assert result.exit_code == 0, (result.output, result.exception)
    assert calls == 2 * depth + 1
    assert source.with_suffix('.html').exists()


def test_reusable_component_cycle_is_rejected(tmp_path):
    spec = _reused_dag_spec(30)
    spec['components']['dag-0']['dag']['tasks']['left']['componentRef'][
        'name'
    ] = 'dag-29'
    source = tmp_path / 'recursive.json'
    source.write_text(json.dumps(spec))
    result = CliRunner().invoke(visualize.visualize, [str(source)])
    assert result.exit_code == 1
    assert 'Recursive DAG components' in result.output
    assert not source.with_suffix('.html').exists()


def test_html_does_not_project_markdown(source, monkeypatch):
    def unexpected_projection(_spec):
        pytest.fail('HTML must not construct the path-expanded Markdown view')

    monkeypatch.setattr(visualization, 'build_view', unexpected_projection)
    result = CliRunner().invoke(visualize.visualize, [str(source)])
    assert result.exit_code == 0, (result.output, result.exception)


def test_markdown_expansion_limit_preserves_last_export(source, monkeypatch):
    monkeypatch.setattr(
        visualization, '_MAX_MARKDOWN_ELEMENTS', 20, raising=False
    )
    state = {'current': {'spec': None, 'error': None, 'revision': 0}}
    output = source.with_suffix('.md')
    visualize._update(source, None, output, 'markdown', state, watch=True)
    previous_file = output.read_bytes()
    previous_spec = state['current']['spec']
    source.write_text(json.dumps(_reused_dag_spec(6)))
    visualize._update(source, None, output, 'markdown', state, watch=True)
    assert 'Markdown expansion' in (state['current']['error'] or '')
    assert output.read_bytes() == previous_file
    assert state['current']['spec'] == previous_spec
    visualize._update(
        source, None, source.with_suffix('.html'), 'html', state, watch=True
    )
    assert state['current']['error'] is None
    assert source.with_suffix('.html').exists()


def test_markdown_limit_counts_dags_tasks_and_dependencies(spec, monkeypatch):
    # One DAG, two tasks, and one deduplicated dependency edge.
    monkeypatch.setattr(visualization, '_MAX_MARKDOWN_ELEMENTS', 3)
    with pytest.raises(ValueError, match='Markdown expansion'):
        visualization.build_view(spec)
    monkeypatch.setattr(visualization, '_MAX_MARKDOWN_ELEMENTS', 4)
    assert len(visualization.build_view(spec)['graphs']) == 1


def test_json_preserves_surrogate_pairs(tmp_path, spec):
    source = tmp_path / 'pipeline.json'
    spec['root']['dag']['tasks']['load']['taskInfo']['name'] = 'Load 🚀'
    source.write_text(json.dumps(spec))
    loaded = visualization.load_spec(source)
    assert (
        loaded['root']['dag']['tasks']['load']['taskInfo']['name'] == 'Load 🚀'
    )


@pytest.mark.parametrize('value', [1e-6, 1e20])
def test_json_preserves_exponent_numbers(tmp_path, spec, value):
    source = tmp_path / 'pipeline.json'
    spec['root']['dag']['tasks']['load']['inputs'] = {
        'parameters': {'amount': {'runtimeValue': {'constant': value}}}
    }
    source.write_text(json.dumps(spec))
    loaded = visualization.load_spec(source)
    actual = loaded['root']['dag']['tasks']['load']['inputs']['parameters'][
        'amount'
    ]['runtimeValue']['constant']
    assert isinstance(actual, float)
    assert actual == value


def test_json_rejects_yaml_syntax(tmp_path, spec):
    source = tmp_path / 'pipeline.json'
    source.write_text(yaml.safe_dump(spec))
    with pytest.raises(json.JSONDecodeError):
        visualization.load_spec(source)


@pytest.mark.parametrize(
    'format_,suffix', [('html', '.html'), ('markdown', '.md')]
)
@pytest.mark.parametrize('nested', [False, True])
@pytest.mark.parametrize('task_info', [None, {}, {'name': ''}])
def test_watch_rejects_unnamed_tasks_and_recovers(
    source, spec, format_, suffix, nested, task_info
):
    task = spec['root']['dag']['tasks']['train']
    if nested:
        task['componentRef']['name'] = 'nested'
        spec['components']['nested'] = {
            'dag': {
                'tasks': {
                    'child': {
                        'componentRef': {'name': 'step'},
                        'taskInfo': {'name': 'Child'},
                    }
                }
            }
        }
        task = spec['components']['nested']['dag']['tasks']['child']
    source.write_text(yaml.safe_dump(spec))
    state = {'current': {'spec': None, 'error': None, 'revision': 0}}
    output = source.with_suffix(suffix)
    visualize._update(source, None, output, format_, state, watch=True)
    previous_file = output.read_bytes()
    previous_spec = state['current']['spec']
    original = task.pop('taskInfo')
    if task_info is not None:
        task['taskInfo'] = task_info
    source.write_text(yaml.safe_dump(spec))
    visualize._update(source, None, output, format_, state, watch=True)
    assert 'taskInfo.name' in (state['current']['error'] or '')
    assert state['current']['spec'] == previous_spec
    assert output.read_bytes() == previous_file
    task['taskInfo'] = original
    source.write_text(yaml.safe_dump(spec))
    visualize._update(source, None, output, format_, state, watch=True)
    assert state['current']['error'] is None
    assert state['current']['spec'] == previous_spec


@pytest.mark.parametrize('suffix', ['.yaml', '.yml', '.json'])
def test_load_spec_formats(tmp_path, spec, suffix):
    source = tmp_path / ('pipeline' + suffix)
    source.write_text(
        json.dumps(spec) if suffix == '.json' else yaml.safe_dump(spec)
    )
    assert visualization.load_spec(source)['pipelineInfo']['name'] == 'example'


def test_manifest_and_platform_documents(source, spec):
    source.write_text(
        yaml.safe_dump_all([spec, {'platforms': {'kubernetes': {}}}])
    )
    assert visualization.load_spec(source)['root'] == spec['root']
    source.write_text(
        yaml.safe_dump_all(
            [
                {'kind': 'Pipeline', 'spec': {}},
                {'kind': 'PipelineVersion', 'spec': {'pipelineSpec': spec}},
            ]
        )
    )
    assert visualization.load_spec(source)['root'] == spec['root']


@pytest.mark.parametrize(
    'content',
    [
        '[]',
        'null',
        'root: [',
        'root: {}',
        '{"pipelineInfo": {"name": "no-dag"}}',
        '{"pipelineInfo": {"name": "x"}, "root": null}',
        'kind: PipelineVersion\nspec: []',
    ],
)
def test_invalid_inputs_fail_without_overwriting(source, tmp_path, content):
    source.write_text(content)
    output = tmp_path / 'pipeline.md'
    output.write_text('previous export')
    result = CliRunner().invoke(
        visualize.visualize, [str(source), '-o', str(output)]
    )
    assert result.exit_code != 0
    assert output.read_text() == 'previous export'


def test_python_compilation_and_import_refresh(tmp_path):
    helper = tmp_path / 'helper.py'
    helper.write_text('NAME = "first"\n')
    source = tmp_path / 'pipeline.py'
    source.write_text("""from kfp import dsl
from helper import NAME
@dsl.container_component
def step() -> dsl.ContainerSpec:
    return dsl.ContainerSpec(image="alpine", command=["echo", NAME])
@dsl.pipeline(name=NAME)
def pipeline():
    step()
""")
    assert (
        visualization.load_spec(source, 'pipeline')['pipelineInfo']['name']
        == 'first'
    )
    cached_mtime = helper.stat().st_mtime
    py_compile.compile(str(helper))
    helper.write_text('NAME = "other"\n')
    os.utime(helper, (cached_mtime, cached_mtime))
    assert (
        visualization.load_spec(source, 'pipeline')['pipelineInfo']['name']
        == 'other'
    )
    source.write_text('this is invalid Python')
    with pytest.raises(ValueError, match='SyntaxError'):
        visualization.load_spec(source)


def test_labels_cannot_escape_exports(spec):
    payload = (
        '</script><script>alert(1)</script>```\nclick n0 "javascript:alert(1)"'
    )
    spec['pipelineInfo']['name'] = payload
    spec['root']['dag']['tasks']['load']['taskInfo']['name'] = payload
    view = visualization.build_view(spec)
    markdown = visualization.render_markdown(view)
    assert '<script>' not in markdown
    assert markdown.count('```') == 2
    assert '#60;' in markdown
    state = {'spec': spec, 'error': payload, 'revision': 1}
    artifact = visualization.render_html(state)
    assert payload not in artifact
    assert '\\u003c/script\\u003e' in artifact
    assert 'third-party-licenses' in artifact
    assert '"live": false' in artifact
    assert '<script src=' not in artifact


@pytest.mark.parametrize(
    'format_,suffix', [('html', '.html'), ('markdown', '.md')]
)
def test_export_command_without_cluster(source, format_, suffix, monkeypatch):

    def no_client(*args, **kwargs):
        pytest.fail('Visualization must not create a cluster client')

    monkeypatch.setattr(cli.client, 'Client', no_client)
    result = CliRunner().invoke(
        cli.cli, ['visualize', str(source), '--format', format_]
    )
    assert result.exit_code == 0, result.output
    assert source.with_suffix(suffix).exists()


def test_html_exports_full_spec_for_shared_kfp_renderer(source, spec):
    spec['deploymentSpec'] = {
        'executors': {
            'exec-step': {
                'container': {'image': 'alpine', 'command': ['echo', 'hello']}
            }
        }
    }
    source.write_text(yaml.safe_dump(spec))
    result = CliRunner().invoke(visualize.visualize, [str(source)])
    assert result.exit_code == 0, result.output
    artifact = source.with_suffix('.html').read_text()
    data = artifact.split('<script id="kfp-data" type="application/json">', 1)[
        1
    ]
    state = json.loads(data.split('</script>', 1)[0])['state']
    assert state['spec']['deploymentSpec'] == spec['deploymentSpec']
    assert state['spec']['components'] == spec['components']
    assert 'view' not in state


@pytest.mark.parametrize('flag', ['-w', '--watch'])
def test_watch_flag_aliases(source, monkeypatch, flag):
    calls = []
    monkeypatch.setattr(visualize, '_watch', lambda *args: calls.append(args))
    result = CliRunner().invoke(
        visualize.visualize, [str(source), '--format', 'markdown', flag]
    )
    assert result.exit_code == 0, result.output
    assert len(calls) == 1
    assert source.with_suffix('.md').exists()


def test_output_extension_infers_format(source):
    output = source.with_suffix('.md')
    result = CliRunner().invoke(
        visualize.visualize, [str(source), '-o', str(output)]
    )
    assert result.exit_code == 0, result.output
    assert '```mermaid' in output.read_text()


@pytest.mark.parametrize(
    'options',
    [
        ['--function', 'pipeline'],
        ['--format', 'html', '-o', 'out.md'],
        ['-o', 'out.txt'],
        ['-o', 'missing/out.html'],
    ],
)
def test_invalid_options(source, options):
    result = CliRunner().invoke(visualize.visualize, [str(source), *options])
    assert result.exit_code == 2


@pytest.mark.parametrize(
    'format_,suffix', [('html', '.html'), ('markdown', '.md')]
)
def test_watch_keeps_export_on_error_and_recovers(
    source, spec, format_, suffix
):
    state = {'current': {'spec': None, 'error': None, 'revision': 0}}
    output = source.with_suffix(suffix)
    visualize._update(source, None, output, format_, state, watch=True)
    previous = output.read_text()
    source.write_text('invalid: [')
    visualize._update(source, None, output, format_, state, watch=True)
    assert output.read_text() == previous
    assert state['current']['error']
    assert state['current']['spec']['pipelineInfo']['name'] == 'example'
    spec['pipelineInfo']['name'] = 'updated'
    source.write_text(yaml.safe_dump(spec))
    visualize._update(source, None, output, format_, state, watch=True)
    assert state['current']['error'] is None
    assert state['current']['spec']['pipelineInfo']['name'] == 'updated'
    assert output.read_text() != previous


def test_watch_debounces_and_detects_changes_during_export(source, monkeypatch):
    snapshots = iter([(1,), (2,), (2,), (3,), (3,)])
    monkeypatch.setattr(visualize, '_fingerprint', lambda _: next(snapshots))
    monkeypatch.setattr(visualize.time, 'sleep', lambda _: None)
    calls = []
    monkeypatch.setattr(
        visualize, '_update', lambda *args, **kwargs: calls.append(args)
    )
    with pytest.raises(StopIteration):
        visualize._watch(
            source, None, source.with_suffix('.md'), 'markdown', {}, (0,)
        )
    assert len(calls) == 2


def test_fingerprint_tracks_imports_deletion_not_outputs(tmp_path):
    source = tmp_path / 'pipeline.py'
    source.write_text('')
    initial = visualize._fingerprint(source)
    source.with_suffix('.html').write_text('export')
    assert visualize._fingerprint(source) == initial
    helper = tmp_path / 'nested' / 'helper.py'
    helper.parent.mkdir()
    helper.write_text('VALUE = 1')
    assert visualize._fingerprint(source) != initial
    helper.unlink()
    assert visualize._fingerprint(source) == initial
    source.unlink()
    assert visualize._fingerprint(source) != initial


def test_live_server_only_serves_viewer_and_state(spec):
    state = {'current': {'spec': spec, 'error': None, 'revision': 1}}
    with visualize._live_server(state) as url:
        with urllib.request.urlopen(url) as response:
            assert b'"live": true' in response.read()
        with urllib.request.urlopen(url + 'state') as response:
            assert json.load(response)['revision'] == 1
            assert response.headers['Cache-Control'] == 'no-store'
        state['current'] = {**state['current'], 'revision': 2}
        with urllib.request.urlopen(url + 'state') as response:
            assert json.load(response)['revision'] == 2
        for path in ('../../pipeline.py', 'pipeline.yaml'):
            with pytest.raises(urllib.error.HTTPError, match='404'):
                urllib.request.urlopen(url + path)
        request = urllib.request.Request(url, headers={'Host': 'attacker.test'})
        with pytest.raises(urllib.error.HTTPError, match='403'):
            urllib.request.urlopen(request)


@pytest.mark.parametrize(
    'name',
    [
        'pipeline_with_nested_loops',
        'pipeline_with_loops_and_conditions',
        'if_else_with_oneof_parameters',
        'pipeline_with_task_final_status',
        'parallel_and_nested/pipeline_in_pipeline_complex',
        'collected_artifacts',
        'scale/dag_120',
    ],
)
def test_compiler_goldens(name):
    root = Path(__file__).resolve().parents[4]
    source = root / 'test_data/sdk_compiled_pipelines/valid' / f'{name}.yaml'
    view = visualization.build_view(visualization.load_spec(source))
    assert view['graphs']
    assert '```mermaid' in visualization.render_markdown(view)


def test_compile_timeout_reported(source, monkeypatch):
    python_source = source.with_suffix('.py')
    python_source.write_text('')

    def timeout(*args, **kwargs):
        raise subprocess.TimeoutExpired('compile', 60)

    monkeypatch.setattr(visualization.subprocess, 'run', timeout)
    result = CliRunner().invoke(visualize.visualize, [str(python_source)])
    assert result.exit_code == 1
    assert 'timed out' in result.output
