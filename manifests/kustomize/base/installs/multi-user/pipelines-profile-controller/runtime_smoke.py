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
"""Start the actual profile webhook with no cloud or Kubernetes
dependencies."""

import json
import os
from pathlib import Path
import socket
import subprocess
import sys
import tempfile
import time
import urllib.error
import urllib.request


def check_runtime(script):
    with socket.socket() as listener:
        listener.bind(('127.0.0.1', 0))
        port = listener.getsockname()[1]
    environment = {
        **os.environ,
        'AWS_EC2_METADATA_DISABLED': 'true',
        'AWS_ACCESS_KEY_ID': 'runtime-smoke',
        'AWS_SECRET_ACCESS_KEY': 'runtime-smoke',
        'AWS_ENDPOINT_URL': 'http://127.0.0.1:1',
        'S3_ENDPOINT_URL': 'http://127.0.0.1:1',
        'KFP_VERSION': 'runtime-smoke',
        'CONTROLLER_PORT': str(port),
    }
    body = json.dumps({
        'object': {
            'metadata': {
                'name': 'disabled',
                'labels': {}
            }
        },
        'attachments': {},
    }).encode()
    request = urllib.request.Request(
        f'http://127.0.0.1:{port}/sync',
        data=body,
        headers={'Content-Type': 'application/json'})
    opener = urllib.request.build_opener(urllib.request.ProxyHandler({}))
    with tempfile.TemporaryFile(mode='w+') as log:
        process = subprocess.Popen([sys.executable, str(script)],
                                   env=environment,
                                   stdout=log,
                                   stderr=subprocess.STDOUT)
        try:
            deadline = time.monotonic() + 15
            while time.monotonic() < deadline:
                if process.poll() is not None:
                    break
                try:
                    with opener.open(request, timeout=1) as response:
                        result = json.load(response)
                    if result != {'status': {}, 'attachments': []}:
                        raise RuntimeError(
                            f'Unexpected profile response: {result}')
                    print(
                        'Profile controller runtime startup and webhook passed')
                    return
                except (urllib.error.URLError, TimeoutError):
                    time.sleep(0.1)
            log.seek(0)
            raise RuntimeError(
                f'Profile controller failed to start: {log.read()}')
        finally:
            process.terminate()
            try:
                process.wait(timeout=5)
            except subprocess.TimeoutExpired:
                process.kill()
                process.wait(timeout=5)


if __name__ == '__main__':
    script = Path(sys.argv[1]) if len(
        sys.argv) > 1 else Path(__file__).with_name('sync.py')
    check_runtime(script)
