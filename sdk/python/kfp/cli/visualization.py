# Copyright 2026 The Kubeflow Authors
# SPDX-License-Identifier: Apache-2.0
"""Portable, read-only views of native Argo pipeline resources."""

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

from kfp.cli import native_visualization


def load_spec(path: Path, function: str | None = None) -> dict[str, Any]:
    """Compile trusted Python offline or read one native Argo resource."""
    if path.suffix.lower() == '.py':
        with tempfile.TemporaryDirectory(prefix='kfp-visualize-') as directory:
            compiled = Path(directory) / 'pipeline.yaml'
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
                '--disable-caching',
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
    candidates = [
        document
        for document in documents
        if isinstance(document, dict)
        and document.get('kind') in native_visualization.KINDS
    ]
    if len(candidates) != 1:
        raise ValueError(
            'Expected one native Argo Workflow, WorkflowTemplate or CronWorkflow. Recompile the Python source.'
        )
    validate_spec(candidates[0])
    return candidates[0]


def validate_spec(spec: dict[str, Any]) -> None:
    native_visualization.view(spec)


def build_view(spec: dict[str, Any]) -> dict[str, Any]:
    return native_visualization.view(spec)


def _markdown_text(text: str) -> str:
    text = html.escape(text).replace('\n', ' ').replace('\r', ' ')
    return re.sub(r'([\\`*_{}\[\]()#+.!|>-])', r'\\\1', text)


def render_markdown(view: dict[str, Any]) -> str:
    """Export one Mermaid diagram per DAG without executable HTML labels."""
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
            label = node['label'] + (
                f' ({node["kind"]})' if node['kind'] != 'Task' else ''
            )
            label = ''.join(
                character
                if character.isalnum() or character == ' '
                else f'#{ord(character)};'
                for character in label
            )
            lines.append(f'  {identifiers[node["id"]]}["{label}"]')
        for edge in graph['edges']:
            lines.append(
                f'  {identifiers[edge["source"]]} --> {identifiers[edge["target"]]}'
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
    """Embed an offline graph, or the loopback server's live view."""
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
