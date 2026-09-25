#!/usr/bin/env python3
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
"""Regenerate the trusted CI trigger inventory (requires PyYAML)."""

import hashlib
import json
from pathlib import Path
import re

import yaml

ROOT = Path(__file__).resolve().parents[3]
OUTPUT = ROOT / '.github/resources/ci-workflow-inventory.json'


def trigger_header(content):
    """Extract complete top-level name/on blocks without interpreting YAML."""
    blocks = []
    selected = False
    keys = set()
    for line in content.decode('utf-8').splitlines():
        if line and not line[0].isspace() and not line.startswith('#'):
            match = re.match(r'''^(?:([a-z_]+)|"([a-z_]+)"|'([a-z_]+)')\s*:''',
                             line)
            key = next(
                (part for part in match.groups() if part), '') if match else ''
            selected = key in ('name', 'on')
            if selected:
                if key in keys:
                    raise ValueError(f'Duplicate workflow header: {key}')
                keys.add(key)
        if selected:
            blocks.append(line)
    if keys != {'name', 'on'}:
        raise ValueError('Workflow must have top-level name and on headers')
    return '\n'.join(blocks).encode('utf-8')


def main():
    workflows = []
    for path in sorted((ROOT / '.github/workflows').iterdir()):
        if path.suffix not in ('.yml', '.yaml'):
            continue
        content = path.read_bytes()
        definition = yaml.safe_load(content)
        header = trigger_header(content)
        header_definition = yaml.safe_load(header)
        if (header_definition.get('name') != definition.get('name') or
                header_definition.get('on', header_definition.get(True))
                != definition.get('on', definition.get(True))):
            raise ValueError(
                f'Workflow header extraction is incomplete: {path}')
        # PyYAML's YAML 1.1 loader interprets unquoted `on` as True.
        events = definition.get('on', definition.get(True, {}))
        if isinstance(events, str):
            events = {events: None}
        elif isinstance(events, list):
            events = {event: None for event in events}
        if not isinstance(events, dict):
            raise ValueError(f'Unsupported workflow trigger: {path}')
        workflows.append({
            'path':
                path.relative_to(ROOT).as_posix(),
            'name':
                definition['name'],
            'header_sha256':
                hashlib.sha256(header).hexdigest(),
            'pull_request': (events['pull_request'] or {})
                            if 'pull_request' in events else None,
        })
    OUTPUT.write_text(
        json.dumps({
            'version': 1,
            'workflows': workflows
        }, indent=2) + '\n',
        encoding='utf-8')


if __name__ == '__main__':
    main()
