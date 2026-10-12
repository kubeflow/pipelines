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
"""Exercise prefetch supervision with real subprocesses and local downloads."""

import os
from pathlib import Path
import shutil
import signal
import subprocess
import sys
import tempfile
import time
import unittest

SCRIPTS = Path(__file__).resolve().parent


class RuntimePrefetchLifecycleTest(unittest.TestCase):

    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory(prefix='prefetch-test-')
        self.root = Path(self.temporary.name).resolve()
        self.state = self.root / 'state'
        self.script = self.root / 'runtime_prefetch.py'
        shutil.copyfile(SCRIPTS / 'runtime_prefetch.py', self.script)
        self.environment = dict(
            os.environ,
            TEST_ROOT=str(self.root),
            GH_TOKEN='sentinel-secret-not-in-state')
        self.pids = []
        self.addCleanup(self.cleanup)

    def cleanup(self):
        try:
            self.command('stop', timeout=20)
        except (subprocess.TimeoutExpired, OSError):
            pass
        for pid in self.pids:
            try:
                os.kill(pid, signal.SIGKILL)
            except ProcessLookupError:
                pass
        self.temporary.cleanup()

    def command(self, command, timeout=20):
        return subprocess.run(
            [sys.executable,
             str(self.script), command,
             str(self.state)],
            env=self.environment,
            capture_output=True,
            text=True,
            timeout=timeout,
            check=False)

    def waiter(self, body):
        (self.root / 'wait-for-image-artifacts.sh'
        ).write_text('#!/usr/bin/env bash\nset -eu\n' + body)

    def wait_for(self, path, timeout=5):
        deadline = time.monotonic() + timeout
        while not path.exists():
            if time.monotonic() >= deadline:
                self.fail(f'Timed out waiting for {path}: ' +
                          (self.state / 'log').read_text())
            time.sleep(0.02)
        return path.read_text()

    def remember_pid(self, path):
        pid = int(self.wait_for(path))
        self.pids.append(pid)
        return pid

    def assert_process_stopped(self, pid):
        # A reparented zombie has stopped executing but may await its OS reaper.
        deadline = time.monotonic() + 3
        while time.monotonic() < deadline:
            status = subprocess.run(
                ['ps', '-p', str(pid), '-o', 'stat='],
                capture_output=True,
                text=True,
                check=False)
            if status.returncode or status.stdout.strip().startswith('Z'):
                return
            time.sleep(0.05)
        self.fail(f'Prefetch left process {pid} running')

    def test_background_download_finishes_and_never_persists_credentials(self):
        self.waiter('test "$RUNTIME_PREFETCH_ONLY" = true\n'
                    'test "$RUNTIME_DOWNLOAD_ONLY" = true\n'
                    'sleep 0.2\nprintf downloaded > "$TEST_ROOT/download"\n')
        started = self.command('start')
        self.assertEqual(started.returncode, 0, started.stderr)
        finished = self.command('finish')
        self.assertEqual(finished.returncode, 0,
                         finished.stdout + finished.stderr)
        self.assertEqual((self.root / 'download').read_text(), 'downloaded')
        for path in self.state.iterdir():
            self.assertNotIn(self.environment['GH_TOKEN'], path.read_text())
        self.assertFalse(Path((self.state / 'address').read_text()).exists())
        self.assertEqual(self.command('stop').returncode, 0)

    def test_download_failure_propagates_to_finish(self):
        self.waiter('exit 17\n')
        self.assertEqual(self.command('start').returncode, 0)
        self.assertEqual(self.command('finish').returncode, 17)
        self.assertEqual(self.command('stop').returncode, 0)

    def test_duplicate_start_does_not_replace_active_state(self):
        self.waiter('sleep 20\n')
        self.assertEqual(self.command('start').returncode, 0)
        address = (self.state / 'address').read_text()
        self.assertNotEqual(self.command('start').returncode, 0)
        self.assertEqual((self.state / 'address').read_text(), address)
        self.assertEqual(self.command('stop').returncode, 0)

    def test_stop_without_start_is_safe(self):
        self.assertEqual(self.command('stop').returncode, 0)
        self.assertNotEqual(self.command('finish').returncode, 0)

    def test_stop_kills_descendant_even_when_shell_exits_first(self):
        child = self.root / 'child.py'
        child.write_text(
            'import os, pathlib, signal, time\n'
            'signal.signal(signal.SIGTERM, signal.SIG_IGN)\n'
            'pathlib.Path(os.environ["TEST_ROOT"], "child.pid").write_text(str(os.getpid()))\n'
            'time.sleep(60)\n')
        self.waiter(
            f'"{sys.executable}" "$TEST_ROOT/child.py" &\n'
            'while [[ ! -e "$TEST_ROOT/child.pid" ]]; do sleep 0.02; done\n'
            'exit 0\n')
        self.assertEqual(self.command('start').returncode, 0)
        pid = self.remember_pid(self.root / 'child.pid')
        self.assertEqual(self.command('finish').returncode, 0)
        self.assert_process_stopped(pid)

    def test_stop_reaches_real_download_only_loader_descendants(self):
        # Exercise the production download-only path, including its gh child.
        (self.root / 'load-ready-image-artifacts.sh'
        ).write_text('exec bash "' +
                     str(SCRIPTS / 'load-ready-image-artifacts.sh') +
                     '" "$@"\n')
        self.waiter('bash "$1" runtime-base-images:1\n')
        bin_path = self.root / 'bin'
        bin_path.mkdir()
        timeout = bin_path / 'timeout'
        timeout.write_text('#!/usr/bin/env bash\nshift 3\nexec "$@"\n')
        timeout.chmod(0o755)
        gh = bin_path / 'gh'
        gh.write_text(
            '#!' + sys.executable + '\n'
            'import os, pathlib, subprocess, sys, time\n'
            'pathlib.Path(os.environ["TEST_ROOT"], "worker.pid").write_text(str(os.getpid()))\n'
            'child = subprocess.Popen([sys.executable, "-c", '
            '"import signal,time; signal.signal(signal.SIGTERM,signal.SIG_IGN); time.sleep(60)"])\n'
            'pathlib.Path(os.environ["TEST_ROOT"], "grandchild.pid").write_text(str(child.pid))\n'
            'time.sleep(60)\n')
        gh.chmod(0o755)
        self.environment.update(
            PATH=str(bin_path) + os.pathsep + os.environ['PATH'],
            GITHUB_REPOSITORY='example/repo',
            GITHUB_RUN_ID='123',
            IMAGE_PATH=str(self.root / 'images'),
            CLUSTER_NAME='test',
            IMAGE_REGISTRY='registry.test',
            IMAGE_TAG='ci')
        self.assertEqual(self.command('start').returncode, 0)
        worker = self.remember_pid(self.root / 'worker.pid')
        child = self.remember_pid(self.root / 'grandchild.pid')
        time.sleep(0.1)
        self.assertEqual(self.command('stop').returncode, 0)
        self.assert_process_stopped(worker)
        self.assert_process_stopped(child)
        self.assertNotEqual(self.command('finish').returncode, 0)

    def test_start_timeout_stops_owned_supervisor(self):
        source = self.script.read_text().replace(
            'deadline = time.monotonic() + 10',
            'deadline = time.monotonic() + 0.25')
        source = source.replace(
            "if args.command == 'serve':", """if args.command == 'serve':
        Path(os.environ['TEST_ROOT'], 'supervisor.pid').write_text(str(os.getpid()))
        time.sleep(60)""")
        self.script.write_text(source)
        self.assertNotEqual(self.command('start').returncode, 0)
        pid = self.remember_pid(self.root / 'supervisor.pid')
        self.assert_process_stopped(pid)
        # No live supervisor owns this deliberately failed initialization.
        (self.state / 'result').write_text('1')

    def test_stale_state_never_signals_an_unrelated_pid(self):
        process = subprocess.Popen(
            [sys.executable, '-c', 'import time; time.sleep(60)'])
        self.addCleanup(process.wait)
        self.addCleanup(process.kill)
        self.state.mkdir()
        (self.state / 'address').write_text(str(self.root / 'missing-socket'))
        (self.state / 'pid').write_text(str(process.pid))
        self.script.write_text(self.script.read_text().replace(
            'deadline = time.monotonic() + 15',
            'deadline = time.monotonic() + 0.2'))
        self.assertNotEqual(self.command('stop').returncode, 0)
        self.assertIsNone(process.poll())
        (self.state / 'result').write_text('1')

    def test_supervisor_deadline_stops_download(self):
        source = self.script.read_text().replace('LIMIT_SECONDS = 45 * 60',
                                                 'LIMIT_SECONDS = 0.25')
        self.script.write_text(source)
        self.waiter('echo $$ > "$TEST_ROOT/waiter.pid"\nsleep 60\n')
        self.assertEqual(self.command('start').returncode, 0)
        pid = self.remember_pid(self.root / 'waiter.pid')
        self.assertNotEqual(self.command('finish').returncode, 0)
        self.assert_process_stopped(pid)


if __name__ == '__main__':
    unittest.main()
