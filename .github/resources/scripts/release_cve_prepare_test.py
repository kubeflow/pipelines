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
"""Exercise bounded remediation with a real Git worktree and fake updaters."""

import json
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest import mock

import release_cve_prepare as prepare


class SourceFixture(unittest.TestCase):

    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        directory = Path(self.temporary.name)
        self.source = directory / 'source'
        self.source.mkdir()
        self.bundle = directory / 'bundle'
        self.bundle.mkdir()
        for name, content in {
                'go.mod': 'module example.test/fixture\n\ngo 1.26.1\n',
                'backend/Dockerfile': 'FROM scratch\n',
                prepare.LOCKFILE: '{"lockfileVersion":3}\n',
                'frontend/.nvmrc': '24.1.0\n',
                'README.md': 'fixture\n',
        }.items():
            path = self.source / name
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text(content)
        self.git('init', '--quiet')
        self.git('add', '.')
        self.git('-c', 'user.name=Fixture', '-c',
                 'user.email=fixture@example.test', 'commit', '--quiet', '-m',
                 'fixture')
        self.sha = self.git('rev-parse', 'HEAD').stdout.strip()
        self.plan = {
            'schema_version': 1,
            'source_sha': self.sha,
            'go_version': '',
            'npm_vulns': []
        }
        self.write_plan()

    def git(self, *arguments):
        return subprocess.run(['git', *arguments],
                              cwd=self.source,
                              check=True,
                              capture_output=True,
                              text=True)

    def write_plan(self):
        (self.bundle / 'plan.json').write_text(json.dumps(self.plan))


class PrepareTest(SourceFixture):

    def npm_run(self, arguments, source, check=True):
        if arguments[:3] == ['osv-scanner', 'scan', 'source']:
            report = {
                'results': [{
                    'source': {
                        'path': str(source / 'package-lock.json')
                    },
                    'packages': [{
                        'package': {
                            'name': 'fixture',
                            'ecosystem': 'npm',
                            'version': '1.0.0'
                        },
                        'vulnerabilities': [{
                            'id': 'GHSA-first',
                            'aliases': ['CVE-2026-12345', 'GHSA-second'],
                            'affected': []
                        }, {
                            'id': 'GHSA-second',
                            'aliases': ['CVE-2026-98765'],
                            'affected': []
                        }]
                    }]
                }]
            }
            Path(arguments[arguments.index('--output-file') + 1]).write_text(
                json.dumps(report))
            return subprocess.CompletedProcess(arguments, 1, '', '')
        if arguments[:2] == ['osv-scanner', 'fix']:
            Path(arguments[arguments.index('--lockfile') + 1]).write_text(
                '{"lockfileVersion":3,"packages":{}}\n')
            (source / 'package-lock.json.resolve.deps').write_text('cache')
            return subprocess.CompletedProcess(arguments, 0, '', '')
        return self.original_run(arguments, source, check=check)

    def test_npm_updates_only_lockfile_and_limits_upgrade_scope(self):
        self.plan['npm_vulns'] = ['CVE-2026-12345', 'CVE-2026-98765']
        self.write_plan()
        self.original_run = prepare.run
        with mock.patch.object(prepare, 'run', side_effect=self.npm_run) as run:
            outputs = prepare.prepare(self.source, self.bundle)
        self.assertEqual(outputs, {
            'has_patch': True,
            'go_changed': False,
            'npm_changed': True
        })
        command = next(call.args[0]
                       for call in run.call_args_list
                       if call.args[0][:2] == ['osv-scanner', 'fix'])
        self.assertIn('--strategy=in-place', command)
        self.assertIn('--upgrade-config=minor', command)
        self.assertIn('--no-introduce', command)
        self.assertEqual(command.count('--vulns'), 2)
        self.assertIn('GHSA-first', command)
        self.assertIn('GHSA-second', command)
        self.assertNotIn('CVE-2026-12345', command)
        self.assertIn(prepare.LOCKFILE,
                      (self.bundle / 'remediation.patch').read_text())
        self.assertEqual((self.bundle / 'patch.sha256').read_text().strip(),
                         prepare.patch_hash(self.bundle))

    def test_unplanned_file_edit_is_rejected(self):
        self.plan['npm_vulns'] = ['CVE-2026-12345']
        self.write_plan()
        self.original_run = prepare.run

        def mutate_other_file(arguments, source, check=True):
            result = self.npm_run(arguments, source, check)
            if arguments[0] == 'osv-scanner':
                (self.source / 'README.md').write_text('unexpected edit\n')
            return result

        with mock.patch.object(prepare, 'run', side_effect=mutate_other_file):
            with self.assertRaisesRegex(ValueError, 'unexpected files'):
                prepare.prepare(self.source, self.bundle)
        self.assertFalse((self.bundle / 'remediation.patch').exists())

    def test_unrecognized_npm_cve_fails_without_editing_source(self):
        self.plan['npm_vulns'] = ['CVE-2026-11111']
        self.write_plan()
        self.original_run = prepare.run
        with mock.patch.object(prepare, 'run', side_effect=self.npm_run):
            with self.assertRaisesRegex(ValueError, 'mapped to lockfile'):
                prepare.prepare(self.source, self.bundle)
        self.assertEqual(self.git('status', '--porcelain').stdout, '')

    def test_failed_lockfile_scan_does_not_attempt_fix(self):
        self.plan['npm_vulns'] = ['CVE-2026-12345']
        self.write_plan()
        original_run = prepare.run

        def fail_scan(arguments, source, check=True):
            if arguments[0] == 'osv-scanner':
                self.assertEqual(arguments[1], 'scan')
                return subprocess.CompletedProcess(arguments, 128, '', '')
            return original_run(arguments, source, check=check)

        with mock.patch.object(prepare, 'run', side_effect=fail_scan):
            with self.assertRaisesRegex(ValueError, 'lockfile scan failed'):
                prepare.prepare(self.source, self.bundle)
        self.assertEqual(self.git('status', '--porcelain').stdout, '')

    def test_go_calls_trusted_updater_on_source_and_checks_policy(self):
        self.plan['go_version'] = '1.26.2'
        self.write_plan()

        def update(source, version):
            self.assertEqual((source, version), (self.source, '1.26.2'))
            (source /
             'go.mod').write_text('module example.test/fixture\n\ngo 1.26.2\n')

        with mock.patch.object(prepare.update_go_version, 'MANAGED_DOCKERFILES',
                               []), mock.patch.object(
                                   prepare.update_go_version,
                                   'update_repository',
                                   side_effect=update), mock.patch.object(
                                       prepare.update_go_version,
                                       'check_repository') as check:
            outputs = prepare.prepare(self.source, self.bundle)
        check.assert_called_once_with(self.source)
        self.assertEqual(outputs, {
            'has_patch': True,
            'go_changed': True,
            'npm_changed': False
        })

    def test_source_mismatch_and_dirty_checkout_fail_before_updater(self):
        with mock.patch.object(prepare.update_go_version,
                               'update_repository') as update:
            self.plan['source_sha'] = 'a' * 40
            self.write_plan()
            with self.assertRaisesRegex(ValueError, 'blocked release SHA'):
                prepare.prepare(self.source, self.bundle)
            self.plan['source_sha'] = self.sha
            self.write_plan()
            (self.source / 'untracked').write_text('dirty\n')
            with self.assertRaisesRegex(ValueError, 'clean source'):
                prepare.prepare(self.source, self.bundle)
        update.assert_not_called()

    def test_symlink_source_is_rejected(self):
        path = self.source / prepare.LOCKFILE
        path.unlink()
        path.symlink_to(self.source / 'README.md')
        with self.assertRaisesRegex(ValueError, 'Symlink'):
            prepare.regular_source_path(self.source, prepare.LOCKFILE)

    def test_path_escape_is_rejected(self):
        for path in ('../outside', '/tmp/outside'):
            with self.subTest(path=path), self.assertRaisesRegex(
                    ValueError, 'Unsafe'):
                prepare.regular_source_path(self.source, path)

    def test_no_changes_produces_no_publishable_patch(self):
        outputs = prepare.prepare(self.source, self.bundle)
        self.assertFalse(outputs['has_patch'])
        self.assertEqual((self.bundle / 'remediation.patch').read_text(), '')


if __name__ == '__main__':
    unittest.main()
