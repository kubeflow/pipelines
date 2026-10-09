# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Check the release remediation permission and verification boundaries."""

import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

from generate_ci_workflow_inventory import UniqueKeyLoader
import yaml

ROOT = Path(__file__).resolve().parents[3]


def workflow(name):
    return yaml.load(
        (ROOT / '.github/workflows' / name).read_text(), Loader=UniqueKeyLoader)


class ReleaseCveRemediationWorkflowTest(unittest.TestCase):

    def setUp(self):
        self.workflow = workflow('release-cve-remediation.yml')
        self.jobs = self.workflow['jobs']

    def test_remediation_never_releases_images_or_bypasses_the_gate(self):
        release = workflow('image-builds-release.yml')
        job = release['jobs']['remediate-cves']
        self.assertEqual(job['needs'],
                         ['resolve-source', 'build-images-for-release'])
        self.assertIn("needs.build-images-for-release.result == 'failure'",
                      job['if'])
        self.assertIn('!inputs.dry_run', job['if'])
        self.assertIn('!cancelled()', job['if'])
        self.assertEqual(job['with']['policy_sha'],
                         '${{ github.workflow_sha }}')
        self.assertEqual(job['with']['source_sha'],
                         '${{ needs.resolve-source.outputs.sha }}')
        self.assertEqual(job['with']['source_branch'],
                         '${{ inputs.src_branch }}')
        self.assertEqual(release['jobs']['create-manifests']['needs'],
                         ['resolve-source', 'build-images-for-release'])
        self.assertTrue(release['on']['workflow_dispatch']['inputs']
                        ['create_cve_fix_pr']['default'])
        self.assertEqual(job['with']['create_pr'],
                         '${{ inputs.create_cve_fix_pr }}')

    def test_only_publisher_has_write_permissions_and_credentials(self):
        self.assertEqual(self.workflow['permissions'], {
            'contents': 'read',
            'actions': 'read'
        })
        for name in ('prepare', 'verify'):
            self.assertNotIn('permissions', self.jobs[name])
        self.assertEqual(self.jobs['publish']['permissions'], {
            'contents': 'write',
            'pull-requests': 'write',
            'actions': 'read'
        })
        for job in self.jobs.values():
            for step in job['steps']:
                if step.get('uses') == 'actions/checkout@v7':
                    self.assertIs(step['with']['persist-credentials'], False)
                if 'GH_TOKEN' in step.get('env', {}):
                    self.assertEqual(step['env']['GH_TOKEN'],
                                     '${{ github.token }}')
        self.assertNotIn('secrets.', json.dumps(self.workflow))

    def test_publisher_waits_for_tests_and_both_native_rescans(self):
        self.assertEqual(self.jobs['publish']['needs'], ['prepare', 'verify'])
        verify = self.jobs['verify']
        self.assertIs(verify['strategy']['fail-fast'], False)
        self.assertEqual(verify['strategy']['matrix']['include'], [{
            'architecture': 'amd64',
            'runner': 'ubuntu-latest'
        }, {
            'architecture': 'arm64',
            'runner': 'ubuntu-24.04-arm'
        }])
        self.assertEqual(verify['needs'], 'prepare')
        self.assertIn("needs.prepare.outputs.has_patch == 'true'", verify['if'])
        for job in self.jobs.values():
            self.assertNotIn('continue-on-error', job)
            for step in job['steps']:
                self.assertNotIn('continue-on-error', step)
        prepare_steps = self.jobs['prepare']['steps']
        commands = '\n'.join(step.get('run', '') for step in prepare_steps)
        self.assertIn('go test -mod=readonly ./backend/src/...', commands)
        self.assertIn('npm ci\n', commands)
        self.assertIn('npm --prefix server run build\n', commands)
        self.assertIn('npm --prefix server test', commands)
        frontend = next(
            step for step in prepare_steps
            if step.get('name') == 'Test the patched frontend server')
        self.assertEqual(frontend['working-directory'], 'source/frontend')
        self.assertIn('packageManager', frontend['run'])
        self.assertIn('cmp remediation/remediation.patch', commands)
        publish_commands = '\n'.join(
            step.get('run', '') for step in self.jobs['publish']['steps'])
        self.assertNotIn('npm ', publish_commands)
        self.assertNotIn('docker ', publish_commands)
        self.assertNotIn('go test', publish_commands)
        publisher = self.jobs['publish']['steps'][-1]
        self.assertEqual(publisher['env']['SOURCE_BRANCH'],
                         '${{ inputs.source_branch }}')
        self.assertIn('--source-branch "$SOURCE_BRANCH"', publisher['run'])
        evidence = next(
            step for step in self.jobs['publish']['steps']
            if step['name'] == 'Download both native verification records')
        self.assertEqual(
            set(evidence['with']['required-files'].split()),
            {'verification-amd64.json', 'verification-arm64.json'})

    def test_report_only_mode_does_not_prepare_updates(self):
        branch = next(step for step in self.jobs['prepare']['steps']
                      if step.get('id') == 'branch')
        patch = next(step for step in self.jobs['prepare']['steps']
                     if step.get('id') == 'patch')
        self.assertIn('inputs.create_pr', branch['if'])
        self.assertEqual(patch['if'], "steps.branch.outputs.eligible == 'true'")
        plan = next(step for step in self.jobs['prepare']['steps']
                    if step.get('id') == 'plan')
        self.assertNotIn('if', plan)

    def test_branch_guard_executes_exact_sha_check_and_encodes_branch(self):
        step = next(step for step in self.jobs['prepare']['steps']
                    if step.get('id') == 'branch')
        sha = 'a' * 40
        for status, remote_sha, expected in ((0, sha, True), (0, 'b' * 40,
                                                              False), (1, '',
                                                                       False)):
            with self.subTest(
                    status=status, remote_sha=remote_sha
            ), tempfile.TemporaryDirectory() as directory:
                root = Path(directory)
                stub = root / 'gh'
                stub.write_text(
                    f'#!{sys.executable}\n' +
                    'import json, os, sys\nfrom pathlib import Path\n' +
                    'Path(os.environ["CALL_LOG"]).write_text(json.dumps(sys.argv[1:]))\n'
                    + 'print(os.environ["REMOTE_SHA"])\n' +
                    'sys.exit(int(os.environ["REMOTE_STATUS"]))\n')
                stub.chmod(0o755)
                output, summary, calls = root / 'output', root / 'summary', root / 'calls'
                result = subprocess.run(
                    ['bash', '-eo', 'pipefail', '-c', step['run']],
                    text=True,
                    capture_output=True,
                    check=False,
                    env={
                        **os.environ, 'PATH':
                            f'{root}{os.pathsep}{os.environ["PATH"]}',
                        'GITHUB_OUTPUT':
                            str(output),
                        'GITHUB_STEP_SUMMARY':
                            str(summary),
                        'GITHUB_REPOSITORY':
                            'owner/repo',
                        'SOURCE_SHA':
                            sha,
                        'SOURCE_BRANCH':
                            'release/3.0',
                        'REMOTE_SHA':
                            remote_sha,
                        'REMOTE_STATUS':
                            str(status),
                        'CALL_LOG':
                            str(calls)
                    })
                self.assertEqual(result.returncode, 0, result.stderr)
                self.assertEqual(output.read_text(),
                                 f'eligible={str(expected).lower()}\n')
                self.assertEqual(
                    json.loads(calls.read_text())[1],
                    'repos/owner/repo/git/ref/heads/release%2F3.0')
                if not expected:
                    self.assertIn('No fix PR prepared', summary.read_text())


if __name__ == '__main__':
    unittest.main()
