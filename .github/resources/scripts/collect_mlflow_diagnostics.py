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
"""Collect bounded failure evidence without changing MLflow migration state."""

import re
import subprocess
import tempfile

MAX_OUTPUT_BYTES = 65536
MIGRATION_SELECTOR = 'mlflow.opendatahub.io/migration-job=true'


def commands():
    """Select status and logs, excluding Secret objects and workload specs."""
    status = '{range .items[*]}{.metadata.name}{" "}{.status}{"\\n"}{end}'
    return [
        [
            'get', 'mlflow', 'mlflow', '-o',
            'jsonpath={.metadata.generation}{" "}{.status}{"\\n"}'
        ],
        ['get', 'jobs', '-l', MIGRATION_SELECTOR, '-o', f'jsonpath={status}'],
        [
            'get', 'pods', '-l', 'component=mlflow-migration', '-o',
            f'jsonpath={status}'
        ],
        [
            'logs', '-l', 'component=mlflow-migration', '--all-containers=true',
            '--max-log-requests=4', '--tail=100', '--limit-bytes=32768',
            '--prefix'
        ],
        [
            'logs', 'deployment/mlflow-operator-controller-manager',
            '--all-containers=true', '--tail=100', '--limit-bytes=32768',
            '--prefix'
        ],
        [
            'get', 'events', '--field-selector=type=Warning',
            '--sort-by=.metadata.creationTimestamp'
        ],
    ]


def redact(text):
    """Remove URI credentials from errors emitted by database clients."""
    return re.sub(r'(://)[^\s/@]+:[^\s/@]+@', r'\1[redacted]@', text)


def collect(namespace='opendatahub', runner=subprocess.run, output=print):
    """Keep partial evidence after unavailable resources or command
    timeouts."""
    for args in commands():
        output(f'--- MLflow diagnostics: {" ".join(args[:2])} ---')
        with tempfile.TemporaryFile() as capture:
            try:
                result = runner(
                    [
                        'kubectl', '--request-timeout=10s', '-n', namespace,
                        *args
                    ],
                    stdin=subprocess.DEVNULL,
                    stdout=capture,
                    stderr=subprocess.STDOUT,
                    check=False,
                    timeout=15,
                )
                if result.returncode:
                    output(f'Diagnostic command exited {result.returncode}.')
            except (OSError, subprocess.SubprocessError) as error:
                output(
                    f'Diagnostic command unavailable: {type(error).__name__}.')
            capture.seek(0)
            text = capture.read(MAX_OUTPUT_BYTES + 1)
            output(
                redact(text[:MAX_OUTPUT_BYTES].decode(
                    'utf-8', errors='replace')))
            if len(text) > MAX_OUTPUT_BYTES:
                output('[diagnostic output truncated]')


if __name__ == '__main__':
    collect()
