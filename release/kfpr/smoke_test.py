"""Tests for installed-wheel qualification and supported interpreter
selection."""

import json
import subprocess
import sys
import unittest
from pathlib import Path
from tempfile import TemporaryDirectory
from unittest import mock

from kfpr import smoke
from kfpr.packages import LEGACY_PACKAGES


class QualificationTest(unittest.TestCase):
    def test_defaults_preserve_declared_maintenance_matrix(self):
        with TemporaryDirectory() as temporary:
            root = Path(temporary)
            sdk = root / 'sdk/python'
            sdk.mkdir(parents=True)
            (sdk / 'setup.py').write_text(
                "classifiers=['Programming Language :: Python :: 3.9', 'Programming Language :: Python :: 3.11', 'Programming Language :: Python :: 3.10']"
            )
            self.assertEqual(
                smoke.default_interpreters(root),
                ['python3.9', 'python3.10', 'python3.11'],
            )
            (sdk / 'setup.py').unlink()
            (sdk / 'pyproject.toml').write_text('requires-python = ">=3.11"')
            self.assertEqual(smoke.default_interpreters(root), ['python3.11'])

    def test_explicit_interpreters_cannot_omit_declared_versions(self):
        with TemporaryDirectory() as temporary:
            root = Path(temporary)
            self.make_wheels(root, True)
            (root / 'sdk/python/pyproject.toml').write_text(
                'classifiers=["Programming Language :: Python :: 3.9", "Programming Language :: Python :: 3.11"]'
            )
            with (
                mock.patch.object(
                    smoke.subprocess, 'check_output', return_value='3.11'
                ),
                self.assertRaisesRegex(
                    ValueError, 'Missing supported Python versions: 3.9'
                ),
            ):
                smoke.qualify_wheels(
                    root, '2.17.3', [sys.executable], root / 'evidence'
                )

    def test_unknown_python_support_requires_explicit_metadata(self):
        with TemporaryDirectory() as temporary:
            with self.assertRaisesRegex(
                ValueError, 'Cannot determine supported Python'
            ):
                smoke.default_interpreters(Path(temporary))

    def test_missing_interpreter_is_not_skipped(self):
        with mock.patch.object(smoke.shutil, 'which', return_value=None):
            with self.assertRaisesRegex(
                ValueError, 'Missing Python interpreter'
            ):
                smoke.qualify_wheels(
                    Path('.'), '2.17.3', ['python3.9'], Path('/unused')
                )

    def make_wheels(self, root, consolidated):
        sdk = root / 'sdk/python'
        sdk.mkdir(parents=True, exist_ok=True)
        (sdk / 'pyproject.toml').write_text(
            f'requires-python = ">={sys.version_info.major}.{sys.version_info.minor}"'
        )
        if consolidated:
            marker = root / 'sdk/python/kfp/server_api/__init__.py'
            marker.parent.mkdir(parents=True)
            marker.touch()
        paths = []
        for directory, name, _ in LEGACY_PACKAGES:
            if consolidated and name != 'kfp':
                continue
            path = (
                root
                / directory
                / 'dist'
                / f'{name.replace("-", "_")}-2.17.3-py3-none-any.whl'
            )
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_bytes(b'wheel fixture')
            paths.append(str(path))
        return paths

    def test_qualifies_both_layouts_outside_source_tree(self):
        for consolidated in (False, True):
            with (
                self.subTest(consolidated=consolidated),
                TemporaryDirectory() as temporary,
            ):
                root = Path(temporary).resolve()
                wheels = self.make_wheels(root, consolidated)
                calls = []

                def run(command, **kwargs):
                    calls.append(command)
                    self.assertNotEqual(kwargs['cwd'], root)
                    self.assertNotIn('PYTHONPATH', kwargs['env'])
                    if command[1].endswith('smoke.py'):
                        self.assertEqual(
                            set(json.loads(command[2])),
                            {'kfp'}
                            if consolidated
                            else {name for _, name, _ in LEGACY_PACKAGES},
                        )
                        self.assertEqual(
                            command[3],
                            'kfp.server_api'
                            if consolidated
                            else 'kfp_server_api',
                        )
                        (kwargs['cwd'] / 'pipeline.yaml').write_text(
                            'platforms: {}'
                        )
                    return subprocess.CompletedProcess(
                        command,
                        0,
                        '["pip", "setuptools"]'
                        if smoke._BOOTSTRAP in command
                        else 'kfp==2.17.3\n',
                    )

                with (
                    mock.patch.object(
                        smoke.subprocess,
                        'check_output',
                        return_value=f'{sys.version_info.major}.{sys.version_info.minor}',
                    ),
                    mock.patch.object(smoke.subprocess, 'run', side_effect=run),
                    mock.patch.dict(
                        smoke.os.environ, {'PYTHONPATH': str(root)}
                    ),
                ):
                    report = smoke.qualify_wheels(
                        root,
                        '2.17.3',
                        [sys.executable, sys.executable],
                        root / 'evidence',
                    )
                self.assertTrue(report['passed'])
                self.assertEqual(len(report['environments']), 2)
                self.assertEqual(set(report['wheels']), set(wheels))
                installs = [
                    command for command in calls if 'install' in command
                ]
                self.assertEqual(len(installs), 2)
                for command in installs:
                    self.assertEqual(command[5:], wheels)
                self.assertEqual(
                    sum('check' in command for command in calls), 2
                )
                uninstalls = [
                    i
                    for i, command in enumerate(calls)
                    if 'uninstall' in command
                ]
                compiles = [
                    i
                    for i, command in enumerate(calls)
                    if command[1].endswith('smoke.py')
                ]
                self.assertEqual(len(uninstalls), 2)
                self.assertTrue(
                    all(
                        uninstall < compile
                        for uninstall, compile in zip(uninstalls, compiles)
                    )
                )
                self.assertEqual(
                    report['environments'][0]['removed_bootstrap'],
                    ['pip', 'setuptools'],
                )
                self.assertTrue(
                    (root / 'evidence/python-1/pipeline.yaml').exists()
                )
                self.assertEqual(
                    json.loads(
                        (root / 'evidence/qualification.json').read_text()
                    ),
                    report,
                )

    def test_bootstrap_dependencies_retained_conservatively(self):
        class Distribution:
            def __init__(self, name, requires):
                self.metadata = {'Name': name}
                self.requires = requires

        for requirements, removed in (
            ([], ['pip', 'setuptools']),
            (['setuptools>=1'], ['pip']),
            (['setuptools; extra == "optional"'], ['pip']),
            (['not valid !!!'], []),
        ):
            with (
                self.subTest(requirements=requirements),
                mock.patch(
                    'importlib.metadata.distributions',
                    return_value=[
                        Distribution('kfp', requirements),
                        Distribution('setuptools', []),
                        Distribution('pip', []),
                    ],
                ),
                mock.patch.object(sys, 'argv', ['bootstrap', '["kfp"]']),
                mock.patch('builtins.print') as output,
            ):
                exec(smoke._BOOTSTRAP, {})
                self.assertEqual(json.loads(output.call_args.args[0]), removed)

    def test_failed_dependency_check_retains_failure_evidence(self):
        with TemporaryDirectory() as temporary:
            root = Path(temporary)
            self.make_wheels(root, True)

            def run(command, **kwargs):
                return subprocess.CompletedProcess(
                    command,
                    1 if 'check' in command else 0,
                    'dependency conflict',
                )

            with (
                mock.patch.object(
                    smoke.subprocess,
                    'check_output',
                    return_value=f'{sys.version_info.major}.{sys.version_info.minor}',
                ),
                mock.patch.object(smoke.subprocess, 'run', side_effect=run),
                self.assertRaisesRegex(
                    RuntimeError, 'Wheel qualification failed'
                ),
            ):
                smoke.qualify_wheels(
                    root, '2.17.3', [sys.executable], root / 'evidence'
                )
            report = json.loads(
                (root / 'evidence/qualification.json').read_text()
            )
            self.assertFalse(report['passed'])
            self.assertFalse(report['environments'][0]['passed'])
            self.assertIn(
                'dependency conflict',
                (root / 'evidence/python-1/commands.log').read_text(),
            )

    def test_missing_wheel_fails_before_environment_creation(self):
        with (
            TemporaryDirectory() as temporary,
            mock.patch.object(
                smoke.subprocess,
                'check_output',
                return_value=f'{sys.version_info.major}.{sys.version_info.minor}',
            ),
            mock.patch.object(smoke.subprocess, 'run') as run,
        ):
            self.make_wheels(Path(temporary), True)
            next((Path(temporary) / 'sdk/python/dist').glob('*.whl')).unlink()
            with self.assertRaisesRegex(ValueError, 'Expected one'):
                smoke.qualify_wheels(
                    Path(temporary),
                    '2.17.3',
                    [sys.executable],
                    Path(temporary) / 'evidence',
                )
            run.assert_not_called()


if __name__ == '__main__':
    unittest.main()
