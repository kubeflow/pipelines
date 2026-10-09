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
"""Execute the release scan shell with local Docker and OSV test doubles."""

import itertools
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

import yaml

ROOT = Path(__file__).resolve().parents[3]
IMAGE_REF = 'ghcr.io/kubeflow/kfp-api-server@sha256:' + 'a' * 64
TOOL_STUB = r'''
import json
import os
from pathlib import Path
import sys

tool = Path(sys.argv[0]).name
args = sys.argv[1:]
log = Path(os.environ['OSV_TEST_LOG'])
with log.open('a') as output:
    output.write(json.dumps({'tool': tool, 'args': args}) + '\n')

if tool == 'docker' and args[0] == 'pull':
    attempts = sum(
        record['tool'] == 'docker' and record['args'][0] == 'pull'
        for record in map(json.loads, log.read_text().splitlines()))
    sys.exit(1 if attempts <= int(os.environ['OSV_TEST_PULL_FAILURES']) else 0)
if tool == 'docker' and args[:2] == ['image', 'save']:
    Path(args[args.index('--output') + 1]).write_bytes(b'image archive')
    sys.exit(int(os.environ['OSV_TEST_SAVE_EXIT']))
if tool == 'osv-scanner':
    archive = Path(args[args.index('--archive') + 1])
    if archive.read_bytes() != b'image archive':
        sys.exit(99)
    Path(args[args.index('--output-file') + 1]).write_text('{"results": []}')
    sys.exit(int(os.environ['OSV_TEST_SCAN_EXIT']))
if tool == 'sleep':
    sys.exit(0)
sys.exit(98)
'''


class ReleaseCveScanTest(unittest.TestCase):

    @classmethod
    def setUpClass(cls):
        workflow = yaml.safe_load(
            (ROOT / '.github/workflows/build-and-push.yml').read_text())
        steps = workflow['jobs']['build-and-push-images']['steps']
        cls.scan = next(step for step in steps if step.get('id') == 'osv_scan')
        cls.prepare = next(
            step for step in steps
            if step.get('name') == 'Prepare release policy checkout')
        cls.policy = next(
            step for step in steps
            if step.get('name') == 'Enforce fixable CVE policy')

    def run_scan(self,
                 platform='linux/arm64',
                 scanner_exit=0,
                 pull_failures=0,
                 save_exit=0,
                 source_has_policy=True):
        with tempfile.TemporaryDirectory(prefix='release cve scan ') as tmp:
            directory = Path(tmp)
            helper = directory / '.release-policy/.github/resources/scripts/helper-functions.sh'
            helper.parent.mkdir(parents=True)
            helper.write_text(
                (ROOT /
                 '.github/resources/scripts/helper-functions.sh').read_text())
            if source_has_policy:
                source_helper = directory / '.github/resources/scripts/helper-functions.sh'
                source_helper.parent.mkdir(parents=True)
                source_helper.write_text(
                    'echo "Untrusted source helper ran" >&2\nexit 97\n')
            binaries = directory / 'bin'
            binaries.mkdir()
            for name in ('docker', 'osv-scanner', 'sleep'):
                stub = binaries / name
                stub.write_text(f'#!{sys.executable}\n' + TOOL_STUB)
                stub.chmod(0o755)
            runner_temp = directory / 'runner temp'
            runner_temp.mkdir()
            log = directory / 'commands.jsonl'
            result = subprocess.run(
                [
                    'bash', '--noprofile', '--norc', '-eo', 'pipefail', '-c',
                    self.scan['run']
                ],
                cwd=directory,
                env={
                    **os.environ,
                    'PATH':
                        f'{binaries}{os.pathsep}{os.environ["PATH"]}',
                    'RUNNER_TEMP':
                        str(runner_temp),
                    'IMAGE_REF':
                        IMAGE_REF,
                    'PLATFORM':
                        platform,
                    'OSV_TEST_LOG':
                        str(log),
                    'OSV_TEST_SCAN_EXIT':
                        str(scanner_exit),
                    'OSV_TEST_PULL_FAILURES':
                        str(pull_failures),
                    'OSV_TEST_SAVE_EXIT':
                        str(save_exit),
                },
                text=True,
                capture_output=True,
                check=False)
            calls = [json.loads(line) for line in log.read_text().splitlines()]
            self.assertEqual(
                list(runner_temp.iterdir()), [],
                'The image archive must be removed after every scan')
            report_exists = (directory / 'osv-results.json').is_file()
            return result, calls, report_exists

    def test_clean_and_vulnerable_images_scan_the_exact_platform_and_digest(
            self):
        for platform in ('linux/amd64', 'linux/arm64'):
            for scanner_exit in (0, 1):
                with self.subTest(platform=platform, scanner_exit=scanner_exit):
                    result, calls, report_exists = self.run_scan(
                        platform=platform, scanner_exit=scanner_exit)
                    self.assertEqual(result.returncode, 0, result.stderr)
                    self.assertTrue(report_exists)
                    self.assertEqual(
                        calls[0], {
                            'tool': 'docker',
                            'args': ['pull', '--platform', platform, IMAGE_REF],
                        })
                    save = calls[1]['args']
                    self.assertEqual(save[:4],
                                     ['image', 'save', '--platform', platform])
                    self.assertEqual(save[-1], IMAGE_REF)
                    archive = save[save.index('--output') + 1]
                    self.assertEqual(
                        calls[2], {
                            'tool':
                                'osv-scanner',
                            'args': [
                                'scan', 'image', '--archive', archive,
                                '--all-vulns', '--format', 'json',
                                '--output-file', 'osv-results.json'
                            ],
                        })

    def test_scanner_errors_fail_even_when_a_report_exists(self):
        for scanner_exit in (2, 127, 128, 129, 130):
            with self.subTest(scanner_exit=scanner_exit):
                result, _, report_exists = self.run_scan(
                    scanner_exit=scanner_exit)
                self.assertEqual(result.returncode, scanner_exit)
                self.assertTrue(report_exists)
                self.assertIn('OSV-Scanner failed with exit code',
                              result.stdout)

    def test_transient_pull_failures_retry_the_same_digest_and_platform(self):
        result, calls, _ = self.run_scan(pull_failures=1)
        self.assertEqual(result.returncode, 0, result.stderr)
        pulls = [
            call for call in calls
            if call['tool'] == 'docker' and call['args'][0] == 'pull'
        ]
        self.assertEqual(len(pulls), 2)
        self.assertEqual(pulls[0], pulls[1])

    def test_pull_or_export_failure_stops_before_scanning(self):
        for failure in ({'pull_failures': 3}, {'save_exit': 1}):
            with self.subTest(failure=failure):
                result, calls, report_exists = self.run_scan(**failure)
                self.assertNotEqual(result.returncode, 0)
                self.assertFalse(report_exists)
                self.assertFalse(
                    any(call['tool'] == 'osv-scanner' for call in calls))

    def test_scan_supports_source_without_release_policy_files(self):
        result, _, report_exists = self.run_scan(source_has_policy=False)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertTrue(report_exists)

    def test_policy_checkout_requires_an_immutable_commit(self):
        for revision, expected in (('', 1), ('release-3.0', 1), ('a' * 39, 1),
                                   ('a' * 40, 0)):
            with self.subTest(
                    revision=revision), tempfile.TemporaryDirectory() as tmp:
                result = subprocess.run(
                    ['bash', '-eo', 'pipefail', '-c', self.prepare['run']],
                    cwd=tmp,
                    env={
                        **os.environ, 'RELEASE_POLICY_SHA': revision
                    },
                    text=True,
                    capture_output=True,
                    check=False)
                self.assertEqual(result.returncode, expected, result.stderr)

    def test_policy_checkout_rejects_source_symlinks_without_touching_target(
            self):
        with tempfile.TemporaryDirectory() as tmp:
            directory = Path(tmp)
            target = directory / 'source-target'
            target.mkdir()
            sentinel = target / 'keep'
            sentinel.write_text('unchanged')
            (directory / '.release-policy').symlink_to(
                target, target_is_directory=True)
            result = subprocess.run(
                ['bash', '-eo', 'pipefail', '-c', self.prepare['run']],
                cwd=directory,
                env={
                    **os.environ, 'RELEASE_POLICY_SHA': 'a' * 40
                },
                text=True,
                capture_output=True,
                check=False)
            self.assertEqual(result.returncode, 1, result.stderr)
            self.assertIn('.release-policy symlink', result.stdout)
            self.assertEqual(sentinel.read_text(), 'unchanged')

    def test_source_policy_cannot_replace_trusted_policy(self):
        report = {
            'results': [{
                'source': {
                    'path': '/app/package-lock.json',
                    'type': 'artifact'
                },
                'packages': [{
                    'package': {
                        'name': 'example',
                        'ecosystem': 'npm',
                        'version': '1.0.0'
                    },
                    'vulnerabilities': [{
                        'id':
                            'CVE-2026-12345',
                        'affected': [{
                            'package': {
                                'name': 'example',
                                'ecosystem': 'npm'
                            },
                            'ranges': [{
                                'type':
                                    'ECOSYSTEM',
                                'events': [{
                                    'introduced': '0'
                                }, {
                                    'fixed': '2.0.0'
                                }],
                            }],
                        }],
                    }],
                }],
            }],
        }
        for source_has_policy, platform in itertools.product(
            (False, True), ('linux/amd64', 'linux/arm64')):
            with self.subTest(
                    source_has_policy=source_has_policy,
                    platform=platform), tempfile.TemporaryDirectory() as tmp:
                directory = Path(tmp)
                helper_path = Path(
                    '.github/resources/scripts/check_fixable_cves.py')
                trusted_helper = directory / '.release-policy' / helper_path
                trusted_helper.parent.mkdir(parents=True)
                trusted_helper.write_text((ROOT / helper_path).read_text())
                if source_has_policy:
                    source_helper = directory / helper_path
                    source_helper.parent.mkdir(parents=True)
                    source_helper.write_text(
                        'print("Source policy silently allowed everything")\n')
                (directory / 'osv-results.json').write_text(json.dumps(report))
                result = subprocess.run(
                    ['bash', '-eo', 'pipefail', '-c', self.policy['run']],
                    cwd=directory,
                    env={
                        **os.environ,
                        **{
                            name: '' for name in self.policy.get('env', {})
                        },
                        'ALLOW_FIXABLE_CVES': 'false',
                        'IMAGE_NAME': 'kfp-api-server',
                        'PLATFORM': platform,
                        'IMAGE_REF': IMAGE_REF,
                        'SOURCE_SHA': 'b' * 40,
                        'GITHUB_STEP_SUMMARY': str(directory / 'summary.md'),
                    },
                    text=True,
                    capture_output=True,
                    check=False)
                self.assertEqual(result.returncode, 1, result.stderr)
                self.assertIn('CVE-2026-12345', result.stderr)
                self.assertNotIn('Source policy', result.stdout)
                record = directory / f"cve-result-kfp-api-server-{platform.rsplit('/', 1)[1]}.json"
                metadata = json.loads(record.read_text())
                self.assertEqual(metadata['image'], 'kfp-api-server')
                self.assertEqual(metadata['platform'], platform)
                self.assertEqual(metadata['source_sha'], 'b' * 40)
                self.assertEqual(metadata['outcome'], 'blocked')


if __name__ == '__main__':
    unittest.main()
