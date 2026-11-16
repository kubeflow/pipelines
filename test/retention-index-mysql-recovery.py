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
"""Exercise retention-index cancellation and crash recovery on disposable
MySQL.

Requires Docker and Go. Never accepts an existing container or database
endpoint. The crash uses SIGKILL and restarts the same container with
its original data.
"""

import argparse
import os
import socket
import subprocess
import tempfile
import time
import uuid
from pathlib import Path


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--image', default='mysql:8.0')
    args = parser.parse_args()
    name = 'kfp-retention-recovery-' + uuid.uuid4().hex[:12]
    password = uuid.uuid4().hex
    repo = Path(__file__).resolve().parents[1]
    process = None
    created = False

    def docker(*arguments, **kwargs):
        return subprocess.run(['docker', *arguments], check=True, **kwargs)

    try:
        # Docker can reassign an ephemeral published port on restart. Reserve a
        # concrete port so the application's DSN remains stable across the crash.
        with socket.socket() as listener:
            listener.bind(('127.0.0.1', 0))
            port = str(listener.getsockname()[1])
        docker(
            'run',
            '-d',
            '--name',
            name,
            '-p',
            f'127.0.0.1:{port}:3306',
            '-e',
            'MYSQL_ROOT_PASSWORD=' + password,
            args.image,
            '--performance-schema=ON',
            '--innodb-buffer-pool-size=134217728',
        )
        created = True
        port = docker(
            'inspect',
            '--format',
            '{{(index (index .NetworkSettings.Ports "3306/tcp") 0).HostPort}}',
            name,
            capture_output=True,
            text=True,
        ).stdout.strip()
        deadline = time.monotonic() + 120
        while True:
            ready = subprocess.run(
                [
                    'docker',
                    'exec',
                    '-e',
                    'MYSQL_PWD=' + password,
                    name,
                    'mysql',
                    '--protocol=TCP',
                    '-h127.0.0.1',
                    '-uroot',
                    '-e',
                    'SELECT 1',
                ],
                stdout=subprocess.DEVNULL,
                stderr=subprocess.DEVNULL,
            )
            if ready.returncode == 0:
                break
            if time.monotonic() >= deadline:
                raise RuntimeError('Disposable MySQL did not become ready')
            time.sleep(1)
        docker(
            'exec',
            '-e',
            'MYSQL_PWD=' + password,
            name,
            'mysql',
            '--protocol=TCP',
            '-h127.0.0.1',
            '-uroot',
            '-e',
            "UPDATE performance_schema.setup_instruments SET ENABLED='YES', TIMED='YES' WHERE NAME LIKE 'stage/innodb/alter%'; UPDATE performance_schema.setup_consumers SET ENABLED='YES' WHERE NAME LIKE 'events_stages%'; SELECT VERSION();",
        )
        env = os.environ.copy()
        env['KFP_RECURRING_MYSQL_TEST_DSN'] = (
            f'root:{password}@tcp(127.0.0.1:{port})/mysql?parseTime=true&timeout=2s'
        )
        env['KFP_RETENTION_MYSQL_INTERRUPTION_TEST'] = 'true'
        command = [
            'go',
            'test',
            './backend/src/apiserver/client_manager',
            '-count=1',
            '-v',
            '-timeout=10m',
            '-run',
        ]
        subprocess.run(
            command + ['^TestRetentionIndexesMySQLInterruptedBuild$'],
            cwd=repo,
            env=env,
            check=True,
        )
        with tempfile.TemporaryDirectory(prefix='kfp-retention-crash-') as tmp:
            marker = Path(tmp) / 'ready'
            env['KFP_RETENTION_MYSQL_CRASH_READY_FILE'] = str(marker)
            process = subprocess.Popen(
                command + ['^TestRetentionIndexesMySQLCrashRecovery$'],
                cwd=repo,
                env=env,
            )
            deadline = time.monotonic() + 300
            while not marker.exists():
                if process.poll() is not None:
                    raise RuntimeError(
                        'Crash test exited before active-build signal'
                    )
                if time.monotonic() >= deadline:
                    raise RuntimeError(
                        'Timed out waiting for active index build'
                    )
                time.sleep(0.01)
            print(
                'Crashing disposable MySQL after measured build progress',
                flush=True,
            )
            docker('kill', '--signal=KILL', name)
            docker('start', name)
            restarted_port = docker(
                'inspect',
                '--format',
                '{{(index (index .NetworkSettings.Ports "3306/tcp") 0).HostPort}}',
                name,
                capture_output=True,
                text=True,
            ).stdout.strip()
            if restarted_port != port:
                raise RuntimeError(
                    'Disposable database endpoint changed on restart'
                )
            if process.wait(timeout=300) != 0:
                raise RuntimeError('Crash recovery test failed')
    finally:
        if process is not None and process.poll() is None:
            process.terminate()
            try:
                process.wait(timeout=10)
            except subprocess.TimeoutExpired:
                process.kill()
                process.wait()
        if created:
            subprocess.run(
                ['docker', 'logs', '--tail', '35', name], check=False
            )
            subprocess.run(['docker', 'rm', '-fv', name], check=False)


if __name__ == '__main__':
    main()
