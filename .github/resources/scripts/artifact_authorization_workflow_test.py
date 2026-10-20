#!/usr/bin/env python3
# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Guard automatic artifact acceptance and focused manual dispatch."""

import fnmatch
import unittest
from pathlib import Path

import yaml

ROOT = Path(__file__).resolve().parents[3]
WORKFLOW = ROOT / '.github/workflows/e2e-test.yml'
ACCEPTANCE_JOB = 'artifact-authorization-acceptance'


class ArtifactAuthorizationWorkflowTest(unittest.TestCase):
    def setUp(self):
        self.workflow = yaml.safe_load(WORKFLOW.read_text())
        self.jobs = self.workflow['jobs']

    def test_focused_dispatch_starts_only_build_and_one_acceptance_runner(self):
        event = self.workflow.get('on', self.workflow.get(True))
        option = event['workflow_dispatch']['inputs'][
            'artifact_authorization_only'
        ]
        self.assertEqual(option['type'], 'boolean')
        self.assertIs(option['default'], False)
        self.assertNotIn('if', self.jobs['build'])
        self.assertEqual(
            self.jobs[ACCEPTANCE_JOB]['if'],
            "${{ github.event_name != 'workflow_dispatch' || inputs.artifact_authorization_only }}",
        )
        self.assertEqual(self.jobs[ACCEPTANCE_JOB]['needs'], 'build')
        self.assertNotIn('strategy', self.jobs[ACCEPTANCE_JOB])

    def test_runtime_changes_trigger_pull_request_acceptance(self):
        event = self.workflow.get('on', self.workflow.get(True))
        self.assertNotIn('pull_request_target', event)
        paths = [
            'frontend/server/helpers/auth.ts',
            'frontend/server/helpers/launcher-config.ts',
            'frontend/server/minio-helper.ts',
            'frontend/server/k8s-helper.ts',
            'frontend/server/handlers/artifacts.ts',
            'frontend/server/package-lock.json',
            'frontend/Dockerfile',
            'frontend/.nvmrc',
            'frontend/.npmrc',
            'frontend/package.json',
            'frontend/package-lock.json',
            'frontend/tsconfig.json',
            'frontend/vite.config.mts',
            'frontend/scripts/build-server.mjs',
            'frontend/src/third_party/mlmd/generated/metadata_store_pb.js',
            'manifests/kustomize/base/installs/multi-user/pipelines-profile-controller/sync.py',
            'test/artifact-proxy/artifact_authorization_acceptance.py',
            'test/artifact-proxy/external_storage_acceptance.py',
            'test/seaweedfs/external_storage_test.sh',
            '.github/workflows/e2e-test.yml',
            '.github/workflows/image-builds.yml',
        ]
        cases = [(path, True) for path in paths] + [
            ('docs/operator-guides/artifact-authorization.md', False),
            ('frontend/server/README.md', False),
            ('test/artifact-proxy/README.md', False),
            ('frontend/server/OWNERS', False),
        ]
        # These literal and ** suffix/filename patterns use ordered exclusions.
        for path, expected in cases:
            matched = False
            for pattern in event['pull_request']['paths']:
                if fnmatch.fnmatchcase(path, pattern.lstrip('!')):
                    matched = not pattern.startswith('!')
            with self.subTest(path=path):
                self.assertEqual(matched, expected)

    def test_both_full_cluster_fixtures_run_without_extra_opt_in(self):
        steps = self.jobs[ACCEPTANCE_JOB]['steps']
        for command in (
            'test/artifact-proxy/test-artifact-authorization.sh',
            'test/artifact-proxy/external_storage_acceptance.py',
        ):
            with self.subTest(command=command):
                step = next(
                    step for step in steps if command in step.get('run', '')
                )
                self.assertNotIn('if', step)
                self.assertNotIn('continue-on-error', step)

    def test_every_other_job_skips_focused_dispatch_only(self):
        for name, job in self.jobs.items():
            if name in ('build', ACCEPTANCE_JOB):
                continue
            with self.subTest(job=name):
                self.assertEqual(
                    job['if'],
                    "${{ github.event_name != 'workflow_dispatch' || !inputs.artifact_authorization_only }}",
                )

    def test_acceptance_uses_real_multiuser_deployment_without_write_token(
        self,
    ):
        job = self.jobs[ACCEPTANCE_JOB]
        self.assertEqual(
            job['permissions'], {'actions': 'read', 'contents': 'read'}
        )
        self.assertEqual(
            self.jobs['build']['uses'], './.github/workflows/image-builds.yml'
        )
        steps = job['steps']
        self.assertTrue(
            any(
                step.get('uses') == './.github/actions/create-cluster'
                for step in steps
            )
        )
        deploy = next(
            step
            for step in steps
            if step.get('uses') == './.github/actions/deploy'
        )['with']
        self.assertEqual(deploy['multi_user'], 'true')
        self.assertEqual(deploy['artifact_proxy'], 'true')
        self.assertEqual(deploy['db_type'], 'mysql')
        self.assertEqual(deploy['image_path'], 'images_${{ github.run_id }}')
        self.assertTrue(
            any(
                'test/artifact-proxy/test-artifact-authorization.sh'
                in step.get('run', '')
                for step in steps
            )
        )
        upload = next(
            step
            for step in steps
            if step.get('uses')
            == './.github/actions/upload-artifact-with-retry'
        )
        self.assertIn('always()', upload['if'])
        self.assertNotIn('secrets', job)

    def test_all_multiuser_ci_overlays_use_candidate_receiver_image(self):
        base = ROOT / '.github/resources/manifests/multiuser'
        for overlay in (
            'default',
            'cache-disabled',
            'postgresql',
            'artifact-proxy',
        ):
            with self.subTest(overlay=overlay):
                definition = yaml.safe_load(
                    (base / overlay / 'kustomization.yaml').read_text()
                )
                patch = next(
                    item
                    for item in definition['patches']
                    if item.get('target', {}).get('name')
                    == 'kubeflow-pipelines-profile-controller'
                    and 'path' in item
                )
                controller = yaml.safe_load(
                    (base / overlay / patch['path']).read_text()
                )
                environment = {
                    item['name']: item['value']
                    for item in controller['spec']['template']['spec'][
                        'containers'
                    ][0]['env']
                }
                self.assertEqual(
                    environment['FRONTEND_IMAGE'], 'kind-registry:5000/frontend'
                )
                self.assertEqual(environment['FRONTEND_TAG'], 'latest')
                # Preserve the production sidecar setting in default overlays.
                self.assertNotIn('DISABLE_ISTIO_SIDECAR', environment)


if __name__ == '__main__':
    unittest.main()
