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
"""Own a bounded, cancellable download process across cluster action steps."""

import argparse
from datetime import datetime
from datetime import timezone
import os
from pathlib import Path
import select
import signal
import socket
import subprocess
import sys
import tempfile
import time

SCRIPTS = Path(__file__).resolve().parent
LIMIT_SECONDS = 45 * 60


def stamp(message):
    print(
        f'{datetime.now(timezone.utc).isoformat()} runtime prefetch: {message}',
        flush=True)


def stop(process):
    try:
        os.killpg(process.pid, signal.SIGTERM)
    except ProcessLookupError:
        pass
    try:
        process.wait(timeout=10)
    except subprocess.TimeoutExpired:
        pass
    # A shell may exit before a descendant, including one ignoring SIGTERM.
    try:
        os.killpg(process.pid, signal.SIGKILL)
    except ProcessLookupError:
        pass
    process.wait()


def serve(directory, address):
    cancelled = False

    def cancel(_signum, _frame):
        nonlocal cancelled
        cancelled = True

    signal.signal(signal.SIGTERM, cancel)
    signal.signal(signal.SIGINT, cancel)
    process = None
    result = 1
    with socket.socket(socket.AF_UNIX) as server:
        try:
            server.bind(address)
            server.listen(1)
            stamp('started; downloads only, cluster not required')
            env = dict(
                os.environ,
                RUNTIME_PREFETCH_ONLY='true',
                RUNTIME_DOWNLOAD_ONLY='true')
            process = subprocess.Popen([
                'bash',
                str(SCRIPTS / 'wait-for-image-artifacts.sh'),
                str(SCRIPTS / 'load-ready-image-artifacts.sh')
            ],
                                       env=env,
                                       start_new_session=True)
            deadline = time.monotonic() + LIMIT_SECONDS
            while not cancelled and time.monotonic() < deadline:
                result = process.poll()
                if result is not None:
                    break
                if select.select([server], [], [], 0.1)[0]:
                    with server.accept()[0] as client:
                        client.settimeout(1)
                        if client.recv(16) == b'stop':
                            cancelled = True
            if cancelled or result is None:
                result = 1
        finally:
            if process is not None:
                stop(process)
            stamp(f'finished with exit {result}')
            temporary = directory / 'result.tmp'
            temporary.write_text(str(result))
            temporary.replace(directory / 'result')
            Path(address).unlink(missing_ok=True)
            try:
                Path(address).parent.rmdir()
            except OSError:
                pass
    return result


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('command', choices=('start', 'finish', 'stop', 'serve'))
    parser.add_argument('directory', type=Path)
    parser.add_argument('--address')
    args = parser.parse_args()
    directory = args.directory.resolve()
    if args.command == 'serve':
        return serve(directory, args.address)
    if args.command == 'start':
        directory.mkdir(parents=True, exist_ok=False)
        address = str(
            Path(tempfile.mkdtemp(prefix='kfp-prefetch-', dir='/tmp')) /
            'control')
        (directory / 'address').write_text(address)
        with (directory / 'log').open('w') as log:
            process = subprocess.Popen([
                sys.executable,
                str(Path(__file__).resolve()), 'serve',
                str(directory), '--address', address
            ],
                                       stdout=log,
                                       stderr=subprocess.STDOUT,
                                       start_new_session=True)
        # Do not leave an untracked process if server initialization fails.
        deadline = time.monotonic() + 10
        while not Path(address).exists() and not (directory /
                                                  'result').exists():
            if time.monotonic() >= deadline:
                stop(process)
                raise RuntimeError('Prefetch supervisor did not start')
            time.sleep(0.05)
        stamp('background download launched')
        return 0
    if not directory.exists():
        return 0 if args.command == 'stop' else 1
    if args.command == 'stop':
        if (directory / 'result').exists():
            return 0
        try:
            with socket.socket(socket.AF_UNIX) as client:
                client.settimeout(2)
                client.connect((directory / 'address').read_text())
                client.sendall(b'stop')
        except (ConnectionRefusedError, FileNotFoundError):
            pass
        deadline = time.monotonic() + 15
    else:
        stamp('cluster ready; joining download')
        deadline = time.monotonic() + LIMIT_SECONDS + 15
    while not (directory / 'result').exists():
        if time.monotonic() >= deadline:
            raise RuntimeError('Prefetch supervisor did not finish')
        time.sleep(0.1)
    if args.command == 'finish':
        print((directory / 'log').read_text(), end='')
        stamp('join completed')
        return int((directory / 'result').read_text())
    return 0


if __name__ == '__main__':
    sys.exit(main())
