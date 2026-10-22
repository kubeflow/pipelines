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
"""Portable, read-only views of compiled KFP pipeline specifications."""

import functools
import html
import json
import re
import subprocess
import sys
import tempfile
from graphlib import CycleError, TopologicalSorter
from importlib import resources
from pathlib import Path
from typing import Any

import yaml
from google.protobuf import json_format

from kfp.pipeline_spec import pipeline_spec_pb2


def load_spec(path: Path, function: str | None = None) -> dict[str, Any]:
    """Read a spec, or compile Python without submitting a pipeline run.

    Python module-level code executes during compilation. Only compile
    trusted sources; no pipeline run is submitted to a cluster.
    """
    if path.suffix.lower() == '.py':
        with tempfile.TemporaryDirectory(prefix='kfp-visualize-') as directory:
            compiled = Path(directory) / 'pipeline.yaml'
            # Avoid stale timestamp-based .pyc files after rapid same-size edits.
            command = [
                sys.executable,
                '-B',
                '-X',
                f'pycache_prefix={directory}',
                '-c',
                'from kfp.cli.compile_ import compile_; compile_()',
                '--py',
                str(path),
                '--output',
                str(compiled),
            ]
            if function:
                command.extend(['--function', function])
            result = subprocess.run(
                command, capture_output=True, text=True, timeout=60
            )
            if result.returncode:
                raise ValueError(
                    result.stderr.strip()
                    or result.stdout.strip()
                    or 'Pipeline compilation failed.'
                )
            return load_spec(compiled)

    content = path.read_text(encoding='utf-8')
    documents = (
        [json.loads(content)]
        if path.suffix.lower() == '.json'
        else list(yaml.safe_load_all(content))
    )
    # Compiler YAML may have a second document containing platform configuration.
    candidates = []
    for document in documents:
        if not isinstance(document, dict):
            continue
        if document.get('kind') == 'PipelineVersion':
            manifest_spec = document.get('spec')
            if not isinstance(manifest_spec, dict):
                continue
            document = manifest_spec.get('pipelineSpec', {})
        if isinstance(document, dict) and 'root' in document:
            candidates.append(document)
    if len(candidates) != 1:
        raise ValueError(
            'Expected one KFP v2 pipeline spec with a root DAG. '
            'Choose a compiled pipeline YAML/JSON or a Python file.'
        )
    message = json_format.ParseDict(
        candidates[0], pipeline_spec_pb2.PipelineSpec()
    )
    spec = json_format.MessageToDict(message)
    if not spec.get('pipelineInfo', {}).get('name') or 'dag' not in spec.get(
        'root', {}
    ):
        raise ValueError(
            'Expected a named KFP v2 pipeline with a root DAG, '
            'not an individual component.'
        )
    return spec


def _producers(inputs: dict[str, Any]) -> set[str]:
    # Only inspect IR reference fields, never user JSON in runtime constants.
    return {
        source['producerTask']
        for kind in ('parameters', 'artifacts')
        for value in inputs.get(kind, {}).values()
        for key in (
            'taskOutputParameter',
            'taskOutputArtifact',
            'taskFinalStatus',
        )
        if (source := value.get(key)) and source.get('producerTask')
    }


def _task_dependencies(tasks: dict[str, Any]) -> dict[str, set[str]]:
    dependencies = {}
    for key, task in tasks.items():
        if not task.get('taskInfo', {}).get('name'):
            raise ValueError(
                f'Task {key!r} is missing a non-empty taskInfo.name. '
                'Restore its display name before exporting.'
            )
        producers = set(task.get('dependentTasks', []))
        producers.update(_producers(task.get('inputs', {})))
        for producer in producers:
            if producer not in tasks or producer == key:
                raise ValueError(
                    f'Task {key!r} has invalid dependency {producer!r}. '
                    'Correct its task inputs or dependentTasks.'
                )
        dependencies[key] = producers
    return dependencies


def validate_spec(spec: dict[str, Any]) -> None:
    """Validate reachable DAG definitions once, without expanding instances."""
    components = spec.get('components', {})
    pending: list[str | None] = [None]
    visited: set[str | None] = set()
    component_dependencies: dict[str | None, set[str]] = {}
    while pending:
        reference = pending.pop()
        if reference in visited:
            continue
        visited.add(reference)
        component = spec['root'] if reference is None else components[reference]
        tasks = component['dag'].get('tasks', {})
        try:
            TopologicalSorter(_task_dependencies(tasks)).prepare()
        except CycleError as error:
            raise ValueError(
                'Cyclic task dependencies. Remove the cycle '
                'before visualizing this pipeline.'
            ) from error
        nested = set()
        for key, task in tasks.items():
            target_reference = task.get('componentRef', {}).get('name')
            target = components.get(target_reference)
            if target is None:
                raise ValueError(
                    f'Task {key!r} references missing component '
                    f'{target_reference!r}. Correct its componentRef.'
                )
            if 'dag' in target:
                nested.add(target_reference)
            elif not target.get('executorLabel'):
                raise ValueError(
                    f'Component {target_reference!r} has no DAG or '
                    'executorLabel. Use a compiled pipeline spec.'
                )
        component_dependencies[reference] = nested
        pending.extend(nested - visited)
    try:
        TopologicalSorter(component_dependencies).prepare()
    except CycleError as error:
        raise ValueError(
            'Recursive DAG components. Remove the recursive component reference.'
        ) from error


# Reusable components can describe exponentially many instance paths.
_MAX_MARKDOWN_ELEMENTS = 10_000


def build_view(spec: dict[str, Any]) -> dict[str, Any]:
    """Validate and project bounded instance paths for Mermaid Markdown.

    The budget counts DAGs, tasks, and dependencies. HTML uses the
    compact spec directly and does not need this expansion.
    """
    validate_spec(spec)
    graphs = []
    components = spec.get('components', {})
    pending = [(spec['root'], [])]
    remaining = _MAX_MARKDOWN_ELEMENTS
    while pending:
        component, path = pending.pop()
        tasks = component['dag'].get('tasks', {})
        dependencies = _task_dependencies(tasks)
        remaining -= 1 + len(tasks) + sum(map(len, dependencies.values()))
        if remaining < 0:
            raise ValueError(
                f'Markdown expansion exceeds {_MAX_MARKDOWN_ELEMENTS:,} '
                'DAGs, tasks, and dependencies. Use --format html for a '
                'compact visualization of this pipeline.'
            )
        nodes = []
        children = []
        for key, task in sorted(tasks.items()):
            target = components[task['componentRef']['name']]
            kind = 'Task'
            if 'dag' in target:
                kind = (
                    'Loop'
                    if 'parameterIterator' in task or 'artifactIterator' in task
                    else 'Condition'
                    if task.get('triggerPolicy', {}).get('condition')
                    else 'Sub-DAG'
                )
                children.append((target, [*path, key]))
            nodes.append(
                {
                    'id': key,
                    'label': task['taskInfo']['name'],
                    'kind': kind,
                }
            )
        graphs.append(
            {
                'label': ' / '.join(path) if path else 'Pipeline',
                'nodes': nodes,
                'edges': [
                    {'source': source, 'target': target}
                    for source, target in sorted(
                        (source, target)
                        for target, producers in dependencies.items()
                        for source in producers
                    )
                ],
            }
        )
        pending.extend(reversed(children))
    return {'name': spec['pipelineInfo']['name'], 'graphs': graphs}


def _markdown_text(text: str) -> str:
    text = html.escape(text).replace('\n', ' ').replace('\r', ' ')
    return re.sub(r'([\\`*_{}\[\]()#+.!|>-])', r'\\\1', text)


def render_markdown(view: dict[str, Any]) -> str:
    """Export one Mermaid diagram per DAG, without executable HTML labels."""
    lines = [
        f'# {_markdown_text(view["name"])}',
        '',
        'Static pipeline structure; loops are not unrolled.',
        '',
    ]
    for graph in view['graphs']:
        lines.extend(
            [
                f'## {_markdown_text(graph["label"])}',
                '',
                '```mermaid',
                'flowchart TD',
            ]
        )
        identifiers = {
            node['id']: f'n{index}' for index, node in enumerate(graph['nodes'])
        }
        for node in graph['nodes']:
            label = node['label']
            if node['kind'] != 'Task':
                label += f' ({node["kind"]})'
            # Mermaid numeric entities keep user labels out of its grammar.
            label = ''.join(
                character
                if character.isalnum() or character == ' '
                else f'#{ord(character)};'
                for character in label
            )
            lines.append(f'  {identifiers[node["id"]]}["{label}"]')
        for edge in graph['edges']:
            lines.append(
                f'  {identifiers[edge["source"]]} --> '
                f'{identifiers[edge["target"]]}'
            )
        lines.extend(['```', ''])
    return '\n'.join(lines)


@functools.lru_cache(maxsize=1)
def _viewer_assets() -> tuple[str, str, str]:
    assets = resources.files('kfp.cli').joinpath('viewer_assets')
    return (
        assets.joinpath('viewer.js').read_text(encoding='utf-8'),
        assets.joinpath('viewer.css').read_text(encoding='utf-8'),
        assets.joinpath('LICENSES.txt').read_text(encoding='utf-8'),
    )


def render_html(state: dict[str, Any], live: bool = False) -> str:
    """Render an offline HTML artifact, or the local server's live page."""
    script, style, licenses = _viewer_assets()
    data = json.dumps({'state': state, 'live': live}, ensure_ascii=True)
    data = (
        data.replace('<', '\\u003c')
        .replace('>', '\\u003e')
        .replace('&', '\\u0026')
    )
    script = re.sub(r'</script', r'<\\/script', script, flags=re.IGNORECASE)
    return f"""<!doctype html>
<html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>KFP pipeline viewer</title><style>{style}</style></head>
<body><div id="root"></div>
<template id="third-party-licenses"><pre>{html.escape(licenses)}</pre></template>
<script id="kfp-data" type="application/json">{data}</script>
<script>{script}</script></body></html>
"""
