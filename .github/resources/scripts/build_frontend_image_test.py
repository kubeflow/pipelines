"""Regression coverage for bounded frontend build recovery."""

import importlib.util
import os
from pathlib import Path
import signal
import subprocess
import sys
import tempfile
import time
import unittest
from unittest import mock

SPEC = importlib.util.spec_from_file_location(
    'build_frontend_image',
    Path(__file__).with_name('build_frontend_image.py'))
builder = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(builder)


class BuildTest(unittest.TestCase):

    def setUp(self):
        self.directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.directory.cleanup)
        self.output = Path(self.directory.name) / 'frontend.tar'
        self.partial = self.output.with_name('frontend.tar.partial')
        self.commands = []
        self.builds = []

    def execute(self, outcomes, cleanup_fails=False, empty=False):

        def run(command, _seconds, _grace, stdout=None):
            self.commands.append(command)
            if command[:3] == ['docker', 'buildx', 'build']:
                self.builds.append(command)
                self.partial.write_bytes(b'' if empty else b'image')
                outcome = outcomes[len(self.builds) - 1]
                if isinstance(outcome, BaseException):
                    raise outcome
                return outcome
            if cleanup_fails and command[:2] == ['docker', 'ps']:
                return 1
            if cleanup_fails and 'rm' in command:
                raise builder.CommandTimeout('cleanup')
            return 0

        with mock.patch.object(builder, 'run', side_effect=run):
            return builder.build('.', 'frontend/Dockerfile', 'test:latest',
                                 '22', self.output)

    def test_timeout_then_fresh_builder_success(self):
        self.output.write_bytes(b'stale')
        self.assertEqual(self.execute([builder.CommandTimeout('build'), 0]), 0)
        self.assertEqual(self.output.read_bytes(), b'image')
        self.assertFalse(self.partial.exists())
        self.assertNotEqual(self.builds[0][4], self.builds[1][4])
        self.assertNotIn('--no-cache', self.builds[0])
        self.assertIn('--no-cache', self.builds[1])
        first_remove = next(i for i, c in enumerate(self.commands) if 'rm' in c)
        second_setup = [
            i for i, c in enumerate(self.commands) if c[0] == 'bash'
        ][1]
        self.assertLess(first_remove, second_setup)
        self.assertTrue(any(c[:2] == ['docker', 'logs'] for c in self.commands))

    def test_two_timeouts_never_publish(self):
        self.assertEqual(self.execute([builder.CommandTimeout('build')] * 2), 1)
        self.assertFalse(self.output.exists())
        self.assertFalse(self.partial.exists())

    def test_nonzero_build_is_retried(self):
        self.assertEqual(self.execute([1, 0]), 0)
        self.assertEqual(len(self.builds), 2)

    def test_cleanup_failure_stops_retry_and_publication(self):
        self.assertEqual(self.execute([0], cleanup_fails=True), 1)
        self.assertEqual(len(self.builds), 1)
        self.assertFalse(self.output.exists())
        self.assertFalse(self.partial.exists())

    def test_empty_export_never_publishes(self):
        self.assertEqual(self.execute([0, 0], empty=True), 1)
        self.assertFalse(self.output.exists())

    def test_cancellation_cleans_without_retry(self):
        with self.assertRaises(KeyboardInterrupt):
            self.execute([KeyboardInterrupt()])
        self.assertEqual(len(self.builds), 1)
        self.assertTrue(any('rm' in c for c in self.commands))
        self.assertFalse(self.output.exists())
        self.assertFalse(self.partial.exists())

    def test_cancellation_during_diagnostics_removes_daemon(self):
        commands = []

        def run(command, _seconds, _grace):
            commands.append(command)
            if command[:2] == ['docker', 'top']:
                raise KeyboardInterrupt
            if command[:3] == ['docker', 'buildx', 'build']:
                return 1
            return 0

        with mock.patch.object(builder, 'run', side_effect=run):
            with self.assertRaises(KeyboardInterrupt):
                builder.build('.', 'Dockerfile', 'tag', '22', self.output)
        self.assertTrue(
            any(c[:3] == ['docker', 'rm', '--force'] for c in commands))
        self.assertEqual(sum(c[0] == 'bash' for c in commands), 1)
        self.assertFalse(self.output.exists())

    def test_setup_failure_before_creation_can_retry(self):
        setups = []

        def run(command, _seconds, _grace, stdout=None):
            if command[0] == 'bash':
                setups.append(command)
                return 1 if len(setups) == 1 else 0
            if 'rm' in command:
                return 1 if len(setups) == 1 else 0
            if command[:3] == ['docker', 'buildx', 'build']:
                self.partial.write_bytes(b'image')
            return 0

        with mock.patch.object(builder, 'run', side_effect=run):
            self.assertEqual(
                builder.build('.', 'Dockerfile', 'tag', '22', self.output), 0)
        self.assertEqual(len(setups), 2)

    def test_setup_consumes_attempt_budget(self):
        budgets = []

        def run(command, seconds, _grace):
            budgets.append(seconds)
            if command[0] == 'bash':
                time.sleep(0.05)
            else:
                self.partial.write_bytes(b'image')
            return 0

        with mock.patch.object(builder, 'run', side_effect=run):
            self.assertEqual(
                builder.build(
                    '.',
                    'Dockerfile',
                    'tag',
                    '22',
                    self.output,
                    attempt_seconds=1), 0)
        self.assertLess(budgets[1], budgets[0] - 0.04)

    def test_timeout_kills_term_ignoring_descendant(self):
        marker = Path(self.directory.name) / 'survived'
        child = ('import signal,time; from pathlib import Path; '
                 'signal.signal(signal.SIGTERM,signal.SIG_IGN); '
                 f'time.sleep(0.8); Path({str(marker)!r}).touch()')
        parent = ('import subprocess,sys,time; '
                  f'subprocess.Popen([sys.executable,"-c",{child!r}]); '
                  'time.sleep(10)')
        with self.assertRaises(builder.CommandTimeout):
            builder.run([sys.executable, '-c', parent], 0.3, grace=0.05)
        time.sleep(0.8)
        self.assertFalse(marker.exists())

    def test_sigterm_interrupts_running_client(self):
        ready = Path(self.directory.name) / 'ready'
        script = Path(self.directory.name) / 'cancel.py'
        script.write_text(
            f'import sys,signal; sys.path.insert(0,{str(Path(__file__).parent)!r})\n'
            'import build_frontend_image as b\n'
            'def stop(*args): raise KeyboardInterrupt\n'
            'signal.signal(signal.SIGTERM,stop)\n'
            f'b.run([sys.executable,"-c", "from pathlib import Path; import time; '
            f'Path({str(ready)!r}).touch(); time.sleep(30)"],30,grace=0.05)\n')
        process = subprocess.Popen([sys.executable, str(script)],
                                   stdout=subprocess.DEVNULL,
                                   stderr=subprocess.DEVNULL)
        self.addCleanup(lambda: process.poll() is None and process.kill())
        for _ in range(100):
            if ready.exists():
                break
            time.sleep(0.02)
        self.assertTrue(ready.exists())
        os.kill(process.pid, signal.SIGTERM)
        time.sleep(0.01)
        if process.poll() is None:
            os.kill(process.pid, signal.SIGINT)
        self.assertNotEqual(process.wait(timeout=3), 0)


if __name__ == '__main__':
    unittest.main()
