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

from pathlib import Path
import unittest

import yaml

WORKFLOWS = Path(__file__).resolve().parents[2] / 'workflows'


def jobs(name: str) -> dict:
    return yaml.safe_load(
        (WORKFLOWS / name).read_text(encoding='utf-8'))['jobs']


class ReleaseImageWorkflowPermissionsTest(unittest.TestCase):

    def test_ci_scripts_run_when_release_permission_contract_changes(self):
        workflow = yaml.load(
            (WORKFLOWS / 'ci-scripts-tests.yml').read_text(encoding='utf-8'),
            Loader=yaml.BaseLoader)
        paths = workflow['on']['pull_request']['paths']
        for name in ('image-builds-release.yml', 'build-and-push.yml',
                     'create-manifest.yml'):
            with self.subTest(workflow=name):
                self.assertIn(f'.github/workflows/{name}', paths)

    def test_reusable_callers_grant_only_required_permissions(self):
        release_jobs = jobs('image-builds-release.yml')
        build = release_jobs['build-images-for-release']
        manifest = release_jobs['create-manifests']

        self.assertEqual(
            build['permissions'], {
                'contents': 'read',
                'packages': 'write',
                'attestations': 'write',
                'id-token': 'write',
            })
        self.assertEqual(
            build['permissions'],
            jobs('build-and-push.yml')['build-and-push-images']['permissions'])
        self.assertEqual(manifest['permissions'], {
            'contents': 'read',
            'packages': 'write',
        })
        self.assertEqual(
            manifest['permissions'],
            jobs('create-manifest.yml')['create-manifest']['permissions'])

        for caller in (build, manifest):
            with self.subTest(caller=caller['uses']):
                self.assertNotIn('secrets', caller)

        self.assertEqual(build['with']['push'], '${{ !inputs.dry_run }}')
        self.assertIn('!inputs.dry_run', manifest['if'])

    def test_profile_qualification_gates_release_tag_publication(self):
        release_jobs = jobs('image-builds-release.yml')
        release_check = release_jobs['qualify-profile-upgrade']
        manifest = release_jobs['create-manifests']
        build = release_jobs['build-images-for-release']
        self.assertIn('qualify-profile-upgrade', manifest['needs'])
        self.assertIn("needs.qualify-profile-upgrade.result == 'success'", manifest['if'])
        self.assertIn("!startsWith(inputs.target_tag, '2.18.')", manifest['if'])
        self.assertEqual(
            build['with']['src_branch'], '${{ needs.resolve-candidate.outputs.sha }}')
        self.assertEqual(
            release_check['steps'][0]['with']['ref'],
            build['with']['src_branch'])
        self.assertEqual(release_check['steps'][1]['with']['pattern'], 'digests-*-amd64')
        self.assertEqual(release_check['steps'][2]['uses'], './.github/actions/profile-upgrade')
        self.assertEqual(release_check['steps'][2]['with']['release-digests'],
                         release_check['steps'][1]['with']['path'])
        self.assertEqual(
            release_check['steps'][2]['with']['metadata-isolation'],
            "${{ inputs.target_tag == '2.18.2' && 'report' || 'enforce' }}")

    def test_pr_and_release_use_the_same_upgrade_harness(self):
        profile = jobs('upgrade-test.yml')['profile-upgrade']
        self.assertIn('build', profile['needs'])
        self.assertEqual(profile['steps'][1]['uses'], './.github/actions/profile-upgrade')
        self.assertNotIn('continue-on-error', profile)



if __name__ == '__main__':
    unittest.main()
