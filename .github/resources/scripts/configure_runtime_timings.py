#!/usr/bin/env python3
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
"""Enable runtime timings in a disposable profiling cluster, preserving
defaults."""

from datetime import datetime
from datetime import timezone
import json
import os
from pathlib import Path
import subprocess
import time

import yaml

NAME = 'KFP_RUNTIME_PROFILE_TIMINGS'


def workflow_patch(configmap):
    defaults = yaml.safe_load(
        configmap.get('data', {}).get('workflowDefaults', '{}'))
    if defaults is None:
        defaults = {}
    if not isinstance(defaults, dict):
        raise ValueError('workflowDefaults must be a mapping')
    spec = defaults.setdefault('spec', {})
    if not isinstance(spec, dict):
        raise ValueError('workflow spec must be a mapping')
    patch = yaml.safe_load(spec.get('podSpecPatch', '{}'))
    if patch is None:
        patch = {}
    if not isinstance(patch, dict):
        raise ValueError('podSpecPatch must be a mapping')
    containers = patch.setdefault('containers', [])
    if not isinstance(containers, list) or any(
            not isinstance(c, dict) for c in containers):
        raise ValueError('containers must be mappings')
    main = [c for c in containers if c.get('name') == 'main']
    if len(main) > 1:
        raise ValueError('duplicate main containers')
    if main:
        container = main[0]
    else:
        container = {'name': 'main'}
        containers.append(container)
    environment = container.setdefault('env', [])
    if not isinstance(environment, list) or any(
            not isinstance(e, dict) for e in environment):
        raise ValueError('env must be mappings')
    # Only this reserved profiling variable changes; all fixture env entries,
    # including valueFrom and custom containers, retain their original values.
    container['env'] = [e for e in environment if e.get('name') != NAME]
    container['env'].append({'name': NAME, 'value': 'true'})
    spec['podSpecPatch'] = json.dumps(patch)
    return {'data': {'workflowDefaults': yaml.safe_dump(defaults)}}


def main():
    if os.environ.get('PROFILE_RUNTIME_TIMINGS') != 'true':
        return
    root = Path(os.environ['KFP_E2E_PROFILE_DIR']).resolve(strict=True)
    namespace = os.environ['PROFILE_NAMESPACE']
    started = time.monotonic()
    timings = {'enabled': False}

    def kubectl(*args, input_data=None, timeout=20):
        return subprocess.run(
            ['kubectl', '--request-timeout=15s', '-n', namespace, *args],
            input=input_data,
            capture_output=True,
            text=True,
            check=True,
            timeout=timeout).stdout

    try:
        configmap = json.loads(
            kubectl('get', 'configmap', 'workflow-controller-configmap', '-o',
                    'json'))
        patch = workflow_patch(configmap)
        before = time.monotonic()
        kubectl(
            'patch',
            'configmap',
            'workflow-controller-configmap',
            '--type=merge',
            '--patch-file=/dev/stdin',
            input_data=json.dumps(patch))
        timings['workflow_patch_ms'] = (time.monotonic() - before) * 1000
        timings['capture_since'] = datetime.now(timezone.utc).isoformat()
        timings['enabled'] = True
    finally:
        timings['configure_ms'] = (time.monotonic() - started) * 1000
        (root / 'runtime-timing-configuration.json').write_text(
            json.dumps(timings))


if __name__ == '__main__':
    main()
