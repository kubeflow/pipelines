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
import sys

import yaml

ROOT = Path(__file__).resolve().parents[3]
OUTPUT = ROOT / '.github/resources/ci-workflow-inventory.json'


class UniqueKeyLoader(yaml.SafeLoader):
    """Reject ambiguous workflow mappings, including merge overrides."""

    def construct_mapping(self, node, deep=False):
        self.flatten_mapping(node)
        keys = set()
        for key_node, _ in node.value:
            key = self.construct_object(key_node, deep=deep)
            if key in keys:
                raise ValueError(f'Duplicate workflow key: {key}')
            keys.add(key)
        return super().construct_mapping(node, deep=deep)


def literal_bool(loader, node):
    """Keep YAML 1.1 words such as off/no from becoming a literal false."""
    if node.value == 'false':
        return False
    if node.value == 'true':
        return True
    return node.value


UniqueKeyLoader.add_constructor('tag:yaml.org,2002:bool', literal_bool)


def upgrade_paused(path, definition):
    """Recognize only the checked-in whole-workflow pause for #14029."""
    if path != '.github/workflows/upgrade-test.yml':
        return False
    jobs = definition.get('jobs')
    if not isinstance(jobs, dict) or not jobs:
        return False
    for job in jobs.values():
        condition = job.get('if') if isinstance(job, dict) else None
        if condition is not False and not (isinstance(condition, str) and
                                           condition.strip() == '${{ false }}'):
            return False
    return True


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


def build_inventory(records):
    """Parse workflow data without executing repository code."""
    if not isinstance(records, list) or not records:
        raise ValueError('Workflow records must be a nonempty list')
    workflows = []
    seen = set()
    for record in records:
        path = record['path']
        if (not isinstance(path, str) or not re.fullmatch(
                r'\.github/workflows/[A-Za-z0-9_.-]+\.ya?ml', path) or
                path in seen):
            raise ValueError('Invalid or duplicate workflow path')
        seen.add(path)
        content = record['content'].encode('utf-8')
        definition = yaml.load(content, Loader=UniqueKeyLoader)
        if not isinstance(definition, dict):
            raise ValueError(f'Invalid workflow definition: {path}')
        header = trigger_header(content)
        header_definition = yaml.load(header, Loader=UniqueKeyLoader)
        if (header_definition.get('name') != definition.get('name') or
                header_definition.get('on', header_definition.get(True))
                != definition.get('on', definition.get(True))):
            raise ValueError(
                f'Workflow header extraction is incomplete: {path}')
        # Preserve support for both quoted and unquoted workflow event keys.
        events = definition.get('on', definition.get(True, {}))
        if isinstance(events, str):
            events = {events: None}
        elif isinstance(events, list):
            events = {event: None for event in events}
        if not isinstance(events, dict):
            raise ValueError(f'Unsupported workflow trigger: {path}')
        if not isinstance(definition.get('name'),
                          str) or not definition['name']:
            raise ValueError(f'Invalid workflow name: {path}')
        workflow = {
            'path':
                path,
            'name':
                definition['name'],
            'header_sha256':
                hashlib.sha256(header).hexdigest(),
            'pull_request': (events['pull_request'] or {})
                            if 'pull_request' in events else None,
        }
        if upgrade_paused(path, definition):
            workflow['disabled_for_migration'] = True
        workflows.append(workflow)
    workflows.sort(key=lambda workflow: workflow['path'])
    return {'version': 1, 'workflows': workflows}


def main():
    if sys.argv[1:] == ['--stdin']:
        inventory = build_inventory(json.load(sys.stdin))
        json.dump(
            {
                'inventory':
                    inventory,
                'workflowFiles': [{
                    'path': workflow['path'],
                    'header_sha256': workflow['header_sha256'],
                } for workflow in inventory['workflows']],
            }, sys.stdout)
    elif not sys.argv[1:]:
        inventory = build_inventory(
            [{
                'path': path.relative_to(ROOT).as_posix(),
                'content': path.read_text(encoding='utf-8'),
            }
             for path in sorted((ROOT / '.github/workflows').iterdir())
             if path.suffix in ('.yml', '.yaml')])
        OUTPUT.write_text(
            json.dumps(inventory, indent=2) + '\n', encoding='utf-8')
    else:
        raise ValueError('Expected no arguments or --stdin')


if __name__ == '__main__':
    main()
