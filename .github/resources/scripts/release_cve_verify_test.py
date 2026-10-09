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
"""Verify patch and CVE certificate gates using local command doubles."""

import json
from pathlib import Path
import subprocess
import unittest
from unittest import mock

from release_cve_prepare import patch_hash
from release_cve_prepare import run
from release_cve_prepare_test import SourceFixture
import release_cve_verify as verify


def report_for(cve):
    return {
        'results': [{
            'source': {
                'path': '/bin/app'
            },
            'packages': [{
                'package': {
                    'name': 'stdlib',
                    'ecosystem': 'Go',
                    'version': '1.26.1'
                },
                'vulnerabilities': [{
                    'id':
                        cve,
                    'affected': [{
                        'package': {
                            'name': 'stdlib',
                            'ecosystem': 'Go'
                        },
                        'ranges': [{
                            'events': [{
                                'introduced': '0'
                            }, {
                                'fixed': '1.26.2'
                            }]
                        }]
                    }]
                }]
            }]
        }]
    }


class VerifyTest(SourceFixture):

    def setUp(self):
        super().setUp()
        self.target = {
            'image': 'kfp-api-server',
            'target': '/bin/app',
            'cve': 'CVE-2026-12345',
            'ecosystem': 'Go',
            'package': 'stdlib'
        }
        self.other = {**self.target, 'cve': 'CVE-2026-98765'}
        self.plan.update({
            'targets': [self.target],
            'verify_images': [{
                'image': 'kfp-api-server',
                'dockerfile': 'backend/Dockerfile',
                'context': '.'
            }],
            'reports': [{
                'image': 'kfp-api-server',
                'platform': f'linux/{architecture}',
                'source_sha': self.sha,
                'outcome': 'blocked',
                'findings': [self.target, self.other]
            } for architecture in ('amd64', 'arm64')],
        })
        self.write_plan()
        (self.source /
         'go.mod').write_text('module example.test/fixture\n\ngo 1.26.2\n')
        (self.bundle / 'remediation.patch').write_text(
            self.git('diff', '--full-index').stdout)
        self.git('restore', 'go.mod')
        (self.bundle / 'patch.sha256').write_text(patch_hash(self.bundle))
        self.scan_report = report_for('CVE-2026-98765')
        self.scan_exit = 1
        self.output = self.bundle / 'certificate.json'
        self.commands = []

    def command(self, arguments, source, check=True):
        self.commands.append(arguments)
        if arguments[0] == 'git':
            return run(arguments, source, check=check)
        if arguments[:3] == ['docker', 'image', 'save']:
            Path(arguments[arguments.index('--output') +
                           1]).write_bytes(b'image')
        if arguments[0] == 'osv-scanner':
            Path(arguments[arguments.index('--output-file') + 1]).write_text(
                json.dumps(self.scan_report))
            return subprocess.CompletedProcess(arguments, self.scan_exit, '',
                                               '')
        return subprocess.CompletedProcess(arguments, 0, '', '')

    def verify(self, architecture='arm64'):
        with mock.patch.object(
                verify.platform,
                'machine',
                return_value={
                    'arm64': 'aarch64',
                    'amd64': 'x86_64'
                }[architecture]), mock.patch.object(
                    verify, 'run', side_effect=self.command):
            return verify.verify(self.source, self.bundle, architecture,
                                 self.output)

    def test_exact_patch_is_built_and_scanned_for_native_platform(self):
        certificate = self.verify()
        self.assertTrue(certificate['verified'])
        self.assertEqual(certificate['architecture'], 'arm64')
        self.assertEqual(certificate['patch_sha256'], patch_hash(self.bundle))
        self.assertEqual(certificate['remaining_baseline_findings'],
                         {'kfp-api-server': 1})
        self.assertIn('1.26.2', (self.source / 'go.mod').read_text())
        build = next(command for command in self.commands
                     if command[:3] == ['docker', 'buildx', 'build'])
        self.assertIn('--pull', build)
        self.assertIn('--load', build)
        self.assertNotIn('--push', build)
        self.assertEqual(build[build.index('--platform') + 1], 'linux/arm64')
        self.assertIn('NODE_VERSION=24.1.0', build)
        archive = next(command for command in self.commands
                       if command[:3] == ['docker', 'image', 'save'])
        self.assertEqual(archive[archive.index('--platform') + 1],
                         'linux/arm64')
        self.assertFalse(Path(archive[archive.index('--output') + 1]).exists())
        self.assertEqual(self.commands[-1][:4],
                         ['docker', 'image', 'rm', '--force'])

    def test_target_remaining_or_new_blocker_prevents_certificate(self):
        for cve, message in [('CVE-2026-12345', 'targeted'),
                             ('CVE-2026-11111', 'introduced')]:
            with self.subTest(cve=cve):
                self.scan_report = report_for(cve)
                with self.assertRaisesRegex(ValueError, message) as raised:
                    self.verify()
                self.assertIn(cve, str(raised.exception))
                self.assertIn('stdlib', str(raised.exception))
                self.assertFalse(self.output.exists())
                self.assertEqual(self.commands[-1][:4],
                                 ['docker', 'image', 'rm', '--force'])
                self.git('restore', 'go.mod')

    def test_scanner_errors_and_malformed_reports_prevent_certificate(self):
        for code in (2, 128):
            with self.subTest(code=code):
                self.scan_exit = code
                with self.assertRaisesRegex(ValueError, 'OSV-Scanner failed'):
                    self.verify()
                self.assertFalse(self.output.exists())
                self.git('restore', 'go.mod')
        self.scan_exit = 0
        self.scan_report = {}
        with self.assertRaisesRegex(ValueError, 'results'):
            self.verify()
        self.assertFalse(self.output.exists())

    def test_missing_architecture_baseline_fails_before_apply_or_build(self):
        self.plan['reports'].pop()
        self.write_plan()
        with self.assertRaisesRegex(ValueError, 'baseline'):
            self.verify()
        self.assertEqual(self.git('status', '--porcelain').stdout, '')
        self.assertEqual(self.commands, [])

    def test_patch_checksum_mismatch_fails_before_apply_or_build(self):
        (self.bundle / 'remediation.patch').write_text('tampered')
        with self.assertRaisesRegex(ValueError, 'checksum'):
            self.verify()
        self.assertFalse(self.output.exists())
        self.assertFalse(
            any(command[0] == 'docker' for command in self.commands))

    def test_failed_reverification_removes_old_certificate(self):
        self.output.write_text('{"verified":true}')
        self.scan_exit = 128
        with self.assertRaisesRegex(ValueError, 'OSV-Scanner failed'):
            self.verify()
        self.assertFalse(self.output.exists())

    def test_nonnative_runner_is_rejected(self):
        with mock.patch.object(
                verify.platform, 'machine', return_value='x86_64'):
            with self.assertRaisesRegex(ValueError, 'native architecture'):
                verify.verify(self.source, self.bundle, 'arm64', self.output)

    def test_only_image_with_target_can_be_certified(self):
        self.plan['targets'][0]['image'] = 'unbuilt-image'
        self.write_plan()
        with self.assertRaisesRegex(ValueError, 'must be rebuilt'):
            self.verify()


if __name__ == '__main__':
    unittest.main()
