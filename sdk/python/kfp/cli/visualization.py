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

    documents = list(yaml.safe_load_all(path.read_text(encoding='utf-8')))
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


def build_view(spec: dict[str, Any]) -> dict[str, Any]:
    """Validate DAG references and project task dependencies for Markdown.

    Each DAG is a separate view, preserving scope rather than pretending
    loops are unrolled executions. Edges include explicit ordering and
    data inputs.
    """
    graphs = []
    components = spec.get('components', {})

    def visit(
        component: dict[str, Any], path: list[str], ancestors: tuple[str, ...]
    ) -> None:
        tasks = component['dag'].get('tasks', {})
        nodes = []
        edges = set()
        children = []
        for key, task in sorted(tasks.items()):
            reference = task.get('componentRef', {}).get('name')
            target = components.get(reference)
            if target is None:
                raise ValueError(
                    f'Task {key!r} references missing component '
                    f'{reference!r}. Correct its componentRef.'
                )
            child_path = [*path, key]
            kind = 'Task'
            if 'dag' in target:
                if reference in ancestors:
                    raise ValueError(
                        f'Recursive DAG component {reference!r}. '
                        'Remove the recursive component reference.'
                    )
                kind = (
                    'Loop'
                    if 'parameterIterator' in task or 'artifactIterator' in task
                    else 'Condition'
                    if task.get('triggerPolicy', {}).get('condition')
                    else 'Sub-DAG'
                )
                children.append((target, child_path, (*ancestors, reference)))
            elif not target.get('executorLabel'):
                raise ValueError(
                    f'Component {reference!r} has no DAG or '
                    'executorLabel. Use a compiled pipeline spec.'
                )
            nodes.append(
                {
                    'id': key,
                    'label': task.get('taskInfo', {}).get('name', key),
                    'kind': kind,
                }
            )
            dependencies = set(task.get('dependentTasks', []))
            dependencies.update(_producers(task.get('inputs', {})))
            for producer in dependencies:
                if producer not in tasks or producer == key:
                    raise ValueError(
                        f'Task {key!r} has invalid dependency '
                        f'{producer!r}. Correct its task inputs '
                        'or dependentTasks.'
                    )
                edges.add((producer, key))
        # Reject dependency cycles instead of drawing a misleading pipeline.
        remaining = {key: set() for key in tasks}
        for source, target in edges:
            remaining[target].add(source)
        while remaining:
            ready = {
                key
                for key, dependencies in remaining.items()
                if not dependencies
            }
            if not ready:
                raise ValueError(
                    'Cyclic task dependencies. Remove the cycle '
                    'before visualizing this pipeline.'
                )
            remaining = {
                key: dependencies - ready
                for key, dependencies in remaining.items()
                if key not in ready
            }
        graphs.append(
            {
                'label': ' / '.join(path) if path else 'Pipeline',
                'nodes': nodes,
                'edges': [
                    {'source': source, 'target': target}
                    for source, target in sorted(edges)
                ],
            }
        )
        for target, child_path, ancestors in children:
            visit(target, child_path, ancestors)

    visit(spec['root'], [], ())
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
