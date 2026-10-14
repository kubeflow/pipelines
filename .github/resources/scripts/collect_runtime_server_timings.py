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
"""Collect one bounded structured server timing stream per profiling window."""

from datetime import datetime
import json
import os
from pathlib import Path
import subprocess
import sys
import time

ROOT = Path(__file__).resolve().parents[3]
MAX_BYTES = 16 * 1024 * 1024


def main():
    if os.environ.get('PROFILE_RUNTIME_TIMINGS') != 'true':
        return
    directory = Path(os.environ['KFP_E2E_PROFILE_DIR']).resolve(strict=True)
    configuration = json.loads(
        (directory / 'runtime-timing-configuration.json').read_text())
    if not configuration.get('enabled'):
        return
    since = datetime.fromisoformat(configuration['capture_since'])
    if since.tzinfo is None:
        raise ValueError('Profiling capture needs an explicit timezone')
    started = time.monotonic()
    summary = {'timed_out': False, 'input_limit_bytes': MAX_BYTES}
    processes = []
    try:
        # Raw lines pass only through a pipe to the allowlisting filter. Never
        # retain the full server log in the artifact or in runner scratch files.
        logs = subprocess.Popen([
            'kubectl', '--request-timeout=20s', '-n',
            os.environ['PROFILE_NAMESPACE'], 'logs', 'deployment/ml-pipeline',
            '-c', 'ml-pipeline-api-server', '--timestamps=true',
            '--since-time=' + since.isoformat(),
            '--limit-bytes=' + str(MAX_BYTES)
        ],
                                stdout=subprocess.PIPE,
                                stderr=subprocess.DEVNULL)
        processes.append(logs)
        filtered = subprocess.Popen([
            sys.executable,
            str(ROOT / 'backend/test/runtimeprofile/filter_server_timings.py'),
            str(directory)
        ],
                                    stdin=logs.stdout)
        processes.append(filtered)
        logs.stdout.close()
        summary['filter_exit_code'] = filtered.wait(timeout=30)
        summary['kubectl_exit_code'] = logs.wait(timeout=5)
    except subprocess.TimeoutExpired:
        summary['timed_out'] = True
    finally:
        for process in processes:
            if process.poll() is None:
                process.kill()
            process.wait()
        summary['elapsed_ms'] = (time.monotonic() - started) * 1000
        (directory / 'server-timing-collection.json').write_text(
            json.dumps(summary))
    if summary.get('filter_exit_code') != 0 or summary.get(
            'kubectl_exit_code') != 0:
        raise RuntimeError(
            'Structured runtime timing capture was incomplete; see collection summary'
        )


if __name__ == '__main__':
    main()
