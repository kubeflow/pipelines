"""Fail-closed source and installed Python evidence collection tests."""

import copy
import json
import subprocess
import tempfile
import unittest
from pathlib import Path
from unittest import mock

from . import ci_security_runner as runner
from .osv_reports_test import report as vulnerable_report


def clean_report(path, packages):
    return {
        'results': [
            {
                'source': {'path': path, 'type': 'lockfile'},
                'packages': [{'package': package} for package in packages],
            }
        ]
    }


class SecurityRunnerTest(unittest.TestCase):
    def setUp(self):
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        self.root = Path(temporary.name).resolve()
        self.source = self.root / 'source'
        self.source.mkdir()
        (self.source / 'VERSION').write_text('2.18.0\n')
        (self.source / 'go.mod').write_text('module example.test\n')
        (self.source / 'frontend').mkdir()
        (self.source / 'frontend/package-lock.json').write_text('{}')
        self.output = self.root / 'output'
        self.identity = {
            'schema': 1,
            'version': '2.18.0',
            'source_sha': 'a' * 40,
            'workflow_sha': 'b' * 40,
            'request_id': 'c' * 32,
            'run_id': '123',
            'run_attempt': 1,
        }
        self.packages = [
            {'name': 'kfp', 'version': '2.18.0'},
            {'name': 'Some_Package', 'version': '1.0'},
        ]
        self.commands = []
        self.builds = []
        self.drop_python = False
        self.drop_source = False
        self.returncode = 0
        self.vulnerable = False
        self.patches = [
            mock.patch.object(
                runner.subprocess, 'check_output', side_effect=self.check_output
            ),
            mock.patch.object(runner.subprocess, 'run', side_effect=self.scan),
            mock.patch.object(
                runner.smoke,
                'default_interpreters',
                return_value=['python3.10', 'python3.13'],
            ),
            mock.patch.object(
                runner.qualification, 'build_wheels', side_effect=self.build
            ),
            mock.patch.object(
                runner.smoke, 'qualify_wheels', side_effect=self.qualify
            ),
        ]
        for patch in self.patches:
            patch.start()
            self.addCleanup(patch.stop)

    def check_output(self, command, **kwargs):
        if command[:2] == ['git', 'diff']:
            return ''
        if command[0] == 'git':
            return self.identity['source_sha'] + '\n'
        minor = '3.10' if 'python3.10' in command[0] else '3.13'
        return json.dumps({'python_version': minor, 'packages': self.packages})

    def build(self, source, version):
        self.builds.append((source, version))
        self.assertTrue((self.output / 'scans/source.json').is_file())
        self.wheel = self.source / 'kfp-2.18.0-py3-none-any.whl'
        self.wheel.write_bytes(b'qualified-wheel')

    def qualify(self, source, version, interpreters, output):
        self.assertFalse(output.is_relative_to(self.output))
        environments, paths = [], []
        for index, interpreter in enumerate(interpreters):
            environment = output / interpreter / 'venv'
            environment.mkdir(parents=True)
            (environment / 'secret-executable').write_text('not uploaded')
            paths.append(str(environment))
            environments.append(
                {'passed': True, 'distributions': {'kfp': version}}
            )
            lane = output / f'python-{index + 1}'
            lane.mkdir()
            for name in ('commands.log', 'requirements.txt', 'pipeline.yaml'):
                (lane / name).write_text('evidence')
        return {
            'passed': True,
            'wheels': {str(self.wheel): runner._digest(self.wheel)},
            'environments': environments,
            'python_environments': paths,
        }

    def scan(self, command, **kwargs):
        self.commands.append((command, kwargs))
        destination = Path(command[command.index('--output-file') + 1])
        if '--lockfile' in command:
            inventory_path = command[
                command.index('--lockfile') + 1
            ].removeprefix('osv-scanner:')
            packages = json.loads(Path(inventory_path).read_text())['results'][
                0
            ]['packages']
            if self.drop_python:
                packages.pop()
            data = {
                'results': [
                    {
                        'source': {'path': inventory_path, 'type': 'lockfile'},
                        'packages': packages,
                    }
                ]
            }
        else:
            data = clean_report(
                str(self.source / 'go.mod'),
                [{'name': 'example.test', 'version': '1.0', 'ecosystem': 'Go'}],
            )
            if not self.drop_source:
                data['results'].extend(
                    clean_report(
                        str(self.source / 'frontend/package-lock.json'),
                        [
                            {
                                'name': 'example',
                                'version': '1.0',
                                'ecosystem': 'npm',
                            }
                        ],
                    )['results']
                )
            if self.vulnerable:
                data['results'].extend(vulnerable_report()['results'])
        destination.write_text(json.dumps(data))
        return subprocess.CompletedProcess(command, self.returncode)

    def collect(self, include_sdk=True):
        return runner.collect(
            self.source, self.output, self.identity, include_sdk
        )

    def mutate(self, callback):
        path = self.output / 'security.json'
        data = json.loads(path.read_text())
        callback(data)
        path.write_text(json.dumps(data))

    def test_complete_collection_scans_exact_installed_packages_once_per_minor(
        self,
    ):
        with mock.patch.dict(
            'os.environ', {'OSV_CONFIG': 'unsafe', 'OSV_OFFLINE': 'true'}
        ):
            data = self.collect()
        self.assertEqual(len(self.builds), 1)
        self.assertEqual(len(self.commands), 3)
        verified = runner.verify_security(
            self.output, self.identity, ['3.10', '3.13']
        )
        self.assertEqual(len(verified['scans']), 3)
        self.assertEqual(
            data['wheels']['python_environments'],
            ['python-3.10', 'python-3.13'],
        )
        self.assertFalse(list(self.output.rglob('secret-executable')))
        self.assertTrue(
            (self.output / 'smoke/python-1/pipeline.yaml').is_file()
        )
        for command, kwargs in self.commands:
            for flag in ('--all-packages', '--all-vulns', '--config'):
                self.assertIn(flag, command)
            self.assertFalse(
                any(key.startswith('OSV_') for key in kwargs['env'])
            )
        self.assertIn('--no-ignore', self.commands[0][0])
        self.assertEqual(
            verified['scans'][1]['installed_inventory'], self.packages
        )

    def test_backend_only_does_not_build_or_require_sdk(self):
        self.collect(False)
        self.assertFalse(self.builds)
        self.assertIsNone(
            runner.verify_security(self.output, self.identity, [])['wheels']
        )
        with self.assertRaisesRegex(ValueError, 'SDK coverage'):
            runner.verify_security(self.output, self.identity, ['3.10'])

    def test_vulnerabilities_are_retained_without_policy_rejection(self):
        self.vulnerable, self.returncode = True, 1
        self.collect(False)
        result = runner.verify_security(self.output, self.identity, [])
        self.assertEqual(result['scans'][0]['findings'][0]['id'], 'GHSA-test')

    def test_source_scans_disable_supplemental_resolution_and_reachability(
        self,
    ):
        self.collect(False)
        command = self.commands[0][0]
        self.assertIn('--no-resolve', command)
        self.assertIn('--no-call-analysis=all', command)
        self.assertIn('--all-vulns', command)

    def test_partial_extraction_error_blocks_even_with_success_exit_code(self):
        original = self.scan

        def partial(command, **kwargs):
            result = original(command, **kwargs)
            kwargs['stderr'].write(
                'Error during extraction: malformed lockfile'
            )
            return result

        with mock.patch.object(runner.subprocess, 'run', side_effect=partial):
            with self.assertRaisesRegex(
                ValueError, 'incomplete source extraction'
            ):
                self.collect(False)
        self.assertEqual(self.builds, [])

    def test_source_coverage_fails_before_build_and_preserves_evidence(self):
        self.drop_source = True
        with self.assertRaisesRegex(ValueError, 'source dependency coverage'):
            self.collect()
        self.assertFalse(self.builds)
        self.assertFalse(
            json.loads((self.output / 'security.json').read_text())['passed']
        )
        self.assertTrue((self.output / 'scans/source.json').is_file())

    def test_scanner_failure_never_builds_and_retains_failure_envelope(self):
        self.returncode = 127
        with self.assertRaises(subprocess.CalledProcessError):
            self.collect()
        data = json.loads((self.output / 'security.json').read_text())
        self.assertFalse(data['passed'])
        self.assertIn('error', data)
        self.assertFalse(self.builds)

    def test_missing_installed_dependency_fails_closed(self):
        self.drop_python = True
        with self.assertRaisesRegex(ValueError, 'inventory is incomplete'):
            self.collect()
        self.assertTrue((self.output / 'smoke/python-1/commands.log').is_file())
        self.assertFalse(
            json.loads((self.output / 'security.json').read_text())['passed']
        )

    def test_installed_python_version_mismatch_fails(self):
        original = self.check_output

        def wrong(command, **kwargs):
            if command[0] == 'git':
                return original(command, **kwargs)
            return json.dumps(
                {'python_version': '3.9', 'packages': self.packages}
            )

        with (
            mock.patch.object(
                runner.subprocess, 'check_output', side_effect=wrong
            ),
            self.assertRaisesRegex(ValueError, 'minor mismatch'),
        ):
            self.collect()

    def test_altered_identity_or_incomplete_minors_are_rejected(self):
        self.collect()
        with self.assertRaisesRegex(ValueError, 'identity mismatch'):
            runner.verify_security(
                self.output,
                {**self.identity, 'run_attempt': 2},
                ['3.10', '3.13'],
            )
        for minors in (['3.10'], ['3.10', '3.13', '3.14']):
            with self.subTest(minors=minors), self.assertRaises(ValueError):
                runner.verify_security(self.output, self.identity, minors)

    def test_tampered_raw_report_and_wheel_are_rejected(self):
        self.collect()
        for relative in (
            'scans/source.json',
            'wheels/files/kfp-2.18.0-py3-none-any.whl',
        ):
            path = self.output / relative
            original = path.read_bytes()
            path.write_bytes(b'tampered')
            with (
                self.subTest(relative=relative),
                self.assertRaisesRegex(ValueError, 'checksum mismatch'),
            ):
                runner.verify_security(
                    self.output, self.identity, ['3.10', '3.13']
                )
            path.write_bytes(original)

    def test_duplicate_missing_skipped_or_wrong_version_evidence_rejected(self):
        data = self.collect()
        mutations = [
            lambda d: d['scans'].append(copy.deepcopy(d['scans'][0])),
            lambda d: d['scans'].pop(),
            lambda d: d['scans'][1]['excluded'].append(
                {'name': 'kfp', 'reason': 'local'}
            ),
            lambda d: d['wheels']['environments'][0]['distributions'].update(
                kfp='2.17.0'
            ),
            lambda d: d['wheels']['python_environments'].append('python-3.10'),
        ]
        for mutation in mutations:
            (self.output / 'security.json').write_text(json.dumps(data))
            self.mutate(mutation)
            with self.subTest(mutation=mutation), self.assertRaises(ValueError):
                runner.verify_security(
                    self.output, self.identity, ['3.10', '3.13']
                )

    def test_symlink_and_traversal_are_rejected(self):
        self.collect(False)
        path = self.output / 'scans/source.json'
        backup = self.root / 'source-scan.json'
        path.rename(backup)
        path.symlink_to(backup)
        with self.assertRaisesRegex(ValueError, 'Symlink'):
            runner.verify_security(self.output, self.identity, [])
        with self.assertRaisesRegex(ValueError, 'relative'):
            runner._verified_file(
                self.output, '../source-scan.json', runner._digest(backup)
            )

    def test_unversioned_source_declarations_remain_explicitly_unaudited(self):
        original = self.scan

        def unresolved(command, **kwargs):
            completed = original(command, **kwargs)
            if '--lockfile' not in command:
                path = Path(command[command.index('--output-file') + 1])
                data = json.loads(path.read_text())
                data['results'].extend(
                    clean_report(
                        str(self.source / 'requirements.txt'),
                        [
                            {
                                'name': 'unpinned',
                                'version': '',
                                'ecosystem': 'PyPI',
                            }
                        ],
                    )['results']
                )
                path.write_text(json.dumps(data))
            return completed

        with mock.patch.object(
            runner.subprocess, 'run', side_effect=unresolved
        ):
            document = self.collect(False)
        expected = [
            {
                'target': str(self.source / 'requirements.txt'),
                'ecosystem': 'PyPI',
                'package': 'unpinned',
                'installed_version': '',
            }
        ]
        self.assertEqual(document['scans'][0]['unresolved_inventory'], expected)
        verified = runner.verify_security(self.output, self.identity, [])
        self.assertEqual(verified['scans'][0]['unresolved_inventory'], expected)
        self.mutate(
            lambda data: data['scans'][0].update(unresolved_inventory=[])
        )
        with self.assertRaisesRegex(ValueError, 'Unresolved source inventory'):
            runner.verify_security(self.output, self.identity, [])

    def test_unresolved_packages_cannot_supply_required_manifest_coverage(self):
        data = clean_report(
            str(self.source / 'go.mod'),
            [{'name': 'local', 'version': '', 'ecosystem': 'Go'}],
        )
        data['results'].extend(
            clean_report(
                str(self.source / 'frontend/package-lock.json'),
                [{'name': 'package', 'version': '1.0', 'ecosystem': 'npm'}],
            )['results']
        )
        with self.assertRaisesRegex(ValueError, 'coverage: go.mod'):
            runner._source_coverage(data, str(self.source))

    def test_installed_python_unresolved_version_remains_invalid(self):
        self.packages[-1]['version'] = ''
        with self.assertRaisesRegex(
            ValueError, 'Invalid installed distribution'
        ):
            self.collect()

    def test_generated_tracked_changes_fail_qualification(self):
        original = self.check_output

        def changed(command, **kwargs):
            if command[:2] == ['git', 'diff']:
                return 'sdk/python/kfp/generated.py\n'
            return original(command, **kwargs)

        with mock.patch.object(
            runner.subprocess, 'check_output', side_effect=changed
        ):
            with self.assertRaisesRegex(ValueError, 'new version PR'):
                self.collect()
        self.assertFalse(
            json.loads((self.output / 'security.json').read_text())['passed']
        )
        self.assertTrue((self.output / 'scans/python-3.13.json').is_file())

    def test_source_revision_mismatch_stops_before_any_scan(self):
        with mock.patch.object(
            runner.subprocess, 'check_output', return_value='d' * 40
        ):
            with self.assertRaisesRegex(ValueError, 'Source checkout'):
                self.collect()
        self.assertFalse(self.commands)
        self.assertFalse(self.builds)

    def test_sdk_only_does_not_require_backend_version_alignment(self):
        (self.source / 'VERSION').write_text('2.19.0\n')
        self.assertTrue(self.collect()['passed'])

    def test_scan_exit_status_must_match_findings(self):
        self.vulnerable = True
        with self.assertRaisesRegex(ValueError, 'exit code'):
            self.collect(False)

    def test_source_findings_are_normalized_to_repository_relative_paths(self):
        self.collect(False)
        path = self.output / 'scans/source.json'
        data = json.loads(path.read_text())
        vulnerability = vulnerable_report()
        vulnerability['results'][0]['source']['path'] = str(
            self.source / 'requirements.txt'
        )
        data['results'].extend(vulnerability['results'])
        path.write_text(json.dumps(data))

        def update(document):
            document['scans'][0]['sha256'] = runner._digest(path)
            document['scans'][0]['exit_code'] = 1
            document['source']['sha256'] = runner._digest(path)

        self.mutate(update)
        result = runner.verify_security(self.output, self.identity, [])
        self.assertEqual(
            result['scans'][0]['findings'][0]['target'], 'requirements.txt'
        )

    def test_python_package_normalization_and_duplicate_rejection(self):
        report = clean_report(
            'inventory.json',
            [{'name': 'some-package', 'version': '1.0', 'ecosystem': 'PyPI'}],
        )
        runner._python_coverage(
            report, [{'name': 'Some_Package', 'version': '1.0'}]
        )
        with self.assertRaisesRegex(ValueError, 'Duplicate'):
            runner._packages(
                [
                    {'name': 'Some_Package', 'version': '1.0'},
                    {'name': 'some.package', 'version': '2.0'},
                ]
            )


if __name__ == '__main__':
    unittest.main()
