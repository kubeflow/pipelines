# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Bounded disposable-fixture diagnostics; raw logs never leave memory."""

import re
import selectors
import subprocess
import time

from kfp_http import CollectionError
from provision_live_schedules import CONTEXT
from provision_live_schedules import NAMESPACE
from readiness import kill_process_group

MAX_BYTES = 65536
TIMEOUT = 15
MAX_CONTAINERS = 6
CATEGORIES = {
    'authorization':
        ('forbidden', 'permissiondenied', 'unauthorized', 'accessdenied'),
    'filesystem_permission': ('permission denied', 'read-only file system'),
    'dns': ('no such host', 'name resolution'),
    'connection': ('connection refused', 'connection reset', 'unavailable'),
    'timeout': ('deadline exceeded', 'timed out', 'timeout'),
    'metadata': ('mlmd', 'metadata-grpc', 'ml metadata'),
    'api': ('ml-pipeline', 'apiserver', 'api server'),
    'object_store':
        ('seaweedfs', 's3:', 's3 ', 'artifact repository', 'archive logs'),
    'missing_file': ('no such file', 'does not exist', 'not found'),
    'executor_output': ('output parameter', '/tmp/outputs', 'save outputs'),
    'invalid_argument':
        ('invalidargument', 'invalid argument', 'flag provided but not defined',
         'failed to parse', 'cannot parse', 'unmarshal', 'invalid character'),
    'image_pull': ('imagepull', 'errimagepull', 'pull image'),
    'out_of_memory': ('oomkilled', 'out of memory'),
}


def categories(text):
    text = text.lower() if isinstance(text, str) else ''
    return sorted(name for name, patterns in CATEGORIES.items()
                  if any(pattern in text for pattern in patterns))


def log_categories(pod, container):
    if not all(
            isinstance(name, str) and re.fullmatch(
                r'[a-z0-9]([-a-z0-9.]*[a-z0-9])?', name) and len(name) <= 253
            for name in (pod, container)):
        raise CollectionError('invalid_fixture_log_selector')
    command = [
        'kubectl', '--context', CONTEXT, '--namespace', NAMESPACE,
        '--request-timeout=10s', 'logs', pod, '--container=' + container,
        '--tail=100', '--limit-bytes=' + str(MAX_BYTES)
    ]
    chunks = bytearray()
    try:
        with subprocess.Popen(
                command,
                stdout=subprocess.PIPE,
                stderr=subprocess.DEVNULL,
                start_new_session=True) as process:
            with selectors.DefaultSelector() as selector:
                selector.register(process.stdout, selectors.EVENT_READ)
                deadline = time.monotonic() + TIMEOUT
                while True:
                    remaining = deadline - time.monotonic()
                    if remaining <= 0 or not selector.select(remaining):
                        kill_process_group(process)
                        raise CollectionError('fixture_log_timeout')
                    chunk = process.stdout.read1(
                        min(65536, MAX_BYTES + 1 - len(chunks)))
                    if not chunk:
                        break
                    chunks.extend(chunk)
                    if len(chunks) > MAX_BYTES:
                        kill_process_group(process)
                        raise CollectionError('fixture_log_limit')
                try:
                    code = process.wait(
                        timeout=max(.01, deadline - time.monotonic()))
                except subprocess.TimeoutExpired:
                    kill_process_group(process)
                    raise CollectionError('fixture_log_timeout') from None
                if code:
                    raise CollectionError('fixture_log_unavailable')
    except OSError:
        raise CollectionError('fixture_log_unavailable') from None
    return categories(chunks.decode('utf-8', errors='replace'))


def container_diagnostics(pods, workflows, logs=log_categories):
    workflow_names = {w.get('metadata', {}).get('name') for w in workflows}
    result = []
    for pod in pods:
        metadata = pod.get('metadata', {})
        if (metadata.get('namespace') != NAMESPACE or not metadata.get('uid') or
                metadata.get('labels', {}).get('workflows.argoproj.io/workflow')
                not in workflow_names):
            continue
        status = pod.get('status', {})
        for container in status.get('initContainerStatuses', []) + status.get(
                'containerStatuses', []):
            state = container.get('state', {})
            terminal = state.get('terminated', {})
            waiting = state.get('waiting', {})
            if terminal.get('exitCode', 0) == 0 and not waiting:
                continue
            if len(result) >= MAX_CONTAINERS:
                return dict(containers=result, truncated=True)
            name = container.get('name')
            record = dict(
                role=name if name in ('main', 'wait', 'init',
                                      'kfp-launcher') else 'other',
                state='terminated' if terminal else 'waiting',
                categories=categories(
                    terminal.get('message', '') + ' ' +
                    terminal.get('reason', '') + ' ' +
                    waiting.get('message', '') + ' ' +
                    waiting.get('reason', '')))
            reason = terminal.get('reason') or waiting.get('reason')
            record['reason'] = reason if reason in (
                'Error', 'Completed', 'OOMKilled', 'ContainerCannotRun',
                'StartError', 'DeadlineExceeded', 'ImagePullBackOff',
                'ErrImagePull', 'CrashLoopBackOff',
                'CreateContainerConfigError',
                'CreateContainerError') else 'other'
            code = terminal.get('exitCode')
            if type(code) is int:
                record['exit_code'] = code
            if terminal:
                try:
                    record['log_categories'] = logs(metadata.get('name'), name)
                    record['log_collection'] = 'bounded_tail'
                except (OSError, ValueError, TypeError):
                    record['log_collection'] = 'unavailable'
            result.append(record)
    return dict(containers=result, truncated=False)


def node_diagnostics(workflows):
    counts = {}
    total = 0
    for workflow in workflows:
        nodes = workflow.get('status', {}).get('nodes', {})
        if not isinstance(nodes, dict) or len(nodes) > 1000:
            return dict(collection='limit_exceeded')
        for node in nodes.values():
            total += 1
            if total > 1000:
                return dict(collection='limit_exceeded')
            phase = node.get('phase')
            phase = phase if phase in ('Pending', 'Running', 'Succeeded',
                                       'Skipped', 'Failed', 'Error',
                                       'Omitted') else 'other'
            kind = node.get('type')
            kind = kind if kind in ('Pod', 'DAG', 'TaskGroup', 'Retry', 'Steps',
                                    'StepGroup', 'Skipped') else 'other'
            template = node.get('templateName')
            template = template if template in (
                'system-dag-driver', 'system-container-driver',
                'system-container-impl') else 'other'
            key = (kind, phase, template)
            counts[key] = counts.get(key, 0) + 1
    return dict(
        collection='complete',
        nodes=[
            dict(type=kind, phase=phase, template=template, count=count)
            for (kind, phase, template), count in sorted(counts.items())
        ])
