"""Regression tests for dependency resolution during release preparation."""

import subprocess
import unittest
from pathlib import Path
from tempfile import TemporaryDirectory
from unittest import mock

from kfpr import core, steps
from kfpr.dependency_pins import preserve_dependency_pins


class DependencyPinsTest(unittest.TestCase):
    def setUp(self):
        temp = TemporaryDirectory()
        self.addCleanup(temp.cleanup)
        self.root = Path(temp.name)
        self.sdk = self.root / 'sdk/python/requirements.txt'
        self.sdk.parent.mkdir(parents=True)
        self.sdk.write_text('kfp-server-api==2.17.3\nrequests==2.33.0\n')
        self.kubernetes = (
            self.root / 'kubernetes_platform/python/requirements.txt'
        )
        self.kubernetes.parent.mkdir(parents=True)
        self.kubernetes.write_text('kfp==2.17.3\nrequests==2.33.0\n')

    def test_companion_versions_and_formatting_can_change(self):
        with preserve_dependency_pins(self.root):
            self.sdk.write_text(
                '# New annotations\nkfp_server_api==2.18.0\n'
                'Requests[security]==2.33.0 ; python_version >= "3.10"\n'
            )
            self.kubernetes.write_text('kfp==2.18.0\nrequests==2.33.0\n')
        self.assertIn('kfp==2.18.0', self.kubernetes.read_text())

    def test_upgrade_addition_and_removal_restore_both_locks(self):
        before = {p: p.read_bytes() for p in (self.sdk, self.kubernetes)}
        for resolved in (
            'requests==2.34.0\n',
            'requests==2.33.0\nnew-package==1.0\n',
            '',
        ):
            with self.subTest(resolved=resolved):
                with self.assertRaisesRegex(ValueError, 'separately reviewed'):
                    with preserve_dependency_pins(self.root):
                        self.sdk.write_text(
                            'kfp-server-api==2.18.0\n' + resolved
                        )
                        self.kubernetes.write_text(
                            'kfp==2.18.0\nrequests==2.33.0\n'
                        )
                for path, original in before.items():
                    self.assertEqual(path.read_bytes(), original)

    def test_failed_and_interrupted_resolution_is_safe_to_resume(self):
        for error in (RuntimeError('resolver failed'), KeyboardInterrupt()):
            with self.subTest(error=error):
                with self.assertRaises(type(error)):
                    with preserve_dependency_pins(self.root):
                        self.sdk.write_text('requests==99.0\n')
                        raise error
                with preserve_dependency_pins(self.root):
                    self.assertIn('requests==2.33.0', self.sdk.read_text())
                    self.sdk.write_text(
                        'kfp-server-api==2.18.0\nrequests==2.33.0\n'
                    )

    def test_uv_lock_checks_versions_and_sources_and_restores_exports(self):
        lock = self.root / 'uv.lock'
        original = (
            'version = 1\n[[package]]\nname = "kfp"\nversion = "3.0.0"\n'
            'source = { editable = "sdk/python" }\n'
            '[[package]]\nname = "requests"\nversion = "2.33.0"\n'
            'source = { registry = "https://pypi.org/simple" }\n'
        )
        lock.write_text(original)
        with preserve_dependency_pins(self.root):
            lock.write_text(original.replace('3.0.0', '3.0.1'))
        accepted = lock.read_text()
        for changed in (
            accepted.replace('2.33.0', '2.34.0'),
            accepted.replace(
                'https://pypi.org/simple', 'https://other.example/simple'
            ),
        ):
            with self.subTest(changed=changed):
                with self.assertRaisesRegex(ValueError, 'requests'):
                    with preserve_dependency_pins(self.root):
                        lock.write_text(changed)
                        self.sdk.write_text('requests==2.34.0\n')
                self.assertEqual(lock.read_text(), accepted)
                self.assertIn('requests==2.33.0', self.sdk.read_text())

    def test_unpinned_requirements_are_not_silently_ignored(self):
        self.sdk.write_text('requests>=2\n')
        with self.assertRaisesRegex(ValueError, 'fully pinned'):
            with preserve_dependency_pins(self.root):
                self.fail('Must validate the baseline before resolving')

    def test_legacy_sdk_stops_before_building_with_changed_pins(self):
        runner = mock.Mock(spec=core.CommandRunner)
        runner.dry_run = False

        def run(command, cwd=None, **kwargs):
            if command[:2] == ['bash', '-c']:
                self.sdk.write_text(
                    'kfp-server-api==2.18.0\nrequests==2.34.0\n'
                )
            return subprocess.CompletedProcess(command, 0)

        runner.run.side_effect = run
        context = core.ReleaseContext(
            self.root,
            core.ReleaseState(self.root / 'state.json'),
            runner,
            core.ReleaseMetadata.from_version('minor', '2.18.0'),
            'https://github.com/user/pipelines.git',
            True,
            True,
        )
        with self.assertRaisesRegex(ValueError, 'requests'):
            steps._update_sdk_version_files(context)
        self.assertIn('requests==2.33.0', self.sdk.read_text())
        self.assertFalse(
            any(
                call.kwargs.get('cwd') == self.sdk.parent
                and '-m' in call.args[0]
                for call in runner.run.call_args_list
            )
        )

    def test_uv_sdk_checks_pins_before_exporting_or_building(self):
        lock = self.root / 'uv.lock'
        original = (
            'version = 1\n[[package]]\nname = "requests"\nversion = "2.33.0"\n'
        )
        lock.write_text(original)
        runner = mock.Mock(spec=core.CommandRunner)
        runner.dry_run = False

        def run(command, **kwargs):
            if command == ['uv', 'lock']:
                lock.write_text(original.replace('2.33.0', '2.34.0'))
            return subprocess.CompletedProcess(command, 0)

        runner.run.side_effect = run
        context = core.ReleaseContext(
            self.root,
            core.ReleaseState(self.root / 'state.json'),
            runner,
            core.ReleaseMetadata.from_version('major', '3.0.0'),
            'https://github.com/user/pipelines.git',
            True,
            True,
        )
        for consolidated in (False, True):
            with self.subTest(consolidated=consolidated):
                if consolidated:
                    module = self.root / 'sdk/python/kfp/server_api/__init__.py'
                    module.parent.mkdir(parents=True)
                    module.touch()
                runner.reset_mock()
                with self.assertRaisesRegex(ValueError, 'requests'):
                    steps._update_sdk_version_files(context)
                self.assertEqual(lock.read_text(), original)
                self.assertFalse(
                    any(
                        call.args[0][:2] in (['uv', 'export'], ['uv', 'build'])
                        for call in runner.run.call_args_list
                    )
                )

    def test_dry_run_does_not_read_or_change_locks(self):
        self.sdk.write_text('not a lock\n')
        with preserve_dependency_pins(self.root, dry_run=True):
            pass
        self.assertEqual(self.sdk.read_text(), 'not a lock\n')


if __name__ == '__main__':
    unittest.main()
