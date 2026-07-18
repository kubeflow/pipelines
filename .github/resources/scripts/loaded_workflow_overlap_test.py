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
import re
import unittest

ROOT = Path(__file__).resolve().parents[3]
WORKFLOW_JOBS = {
    '.github/workflows/api-server-tests.yml': (
        'api-test-standalone',
        'api-test-k8s-native',
        'api-test-multi-user',
    ),
    '.github/workflows/e2e-test-frontend.yml':
        ('frontend-integration-test',),
    '.github/workflows/e2e-test.yml': (
        'end-to-end-scenario-tests',
        'end-to-end-critical-scenario-multi-user-tests',
        'end-to-end-critical-mlflow-tests',
    ),
    '.github/workflows/integration-tests-v1.yml':
        ('initialization-integration-tests-v1',),
    '.github/workflows/kfp-kubernetes-native-migration-tests.yaml':
        ('kfp-kubernetes-native-migration-tests',),
    '.github/workflows/kfp-sdk-client-tests.yml': ('sdk-client-tests',),
    '.github/workflows/kfp-webhooks.yml': ('webhook-tests',),
    '.github/workflows/legacy-v2-api-integration-tests.yml':
        ('api-integration-tests-v2',),
    '.github/workflows/upgrade-test.yml': ('upgrade-test',),
}


def _job_block(workflow: str, job_name: str) -> str:
    marker = f'  {job_name}:\n'
    start = workflow.index(marker)
    next_job = re.search(r'^  [a-zA-Z0-9_-]+:\n',
                         workflow[start + len(marker):], re.MULTILINE)
    end = (start + len(marker) + next_job.start()
           if next_job else len(workflow))
    return workflow[start:end]


class LoadedWorkflowOverlapTest(unittest.TestCase):

    def test_all_deploy_callers_overlap_cluster_setup_with_image_builds(self):
        deploy_callers = set()
        for workflow_path in (ROOT / '.github/workflows').glob('*.y*ml'):
            workflow = workflow_path.read_text(encoding='utf-8')
            if 'uses: ./.github/actions/deploy' in workflow:
                deploy_callers.add(str(workflow_path.relative_to(ROOT)))

        self.assertEqual(deploy_callers, set(WORKFLOW_JOBS))

        for relative_path, job_names in WORKFLOW_JOBS.items():
            workflow = (ROOT / relative_path).read_text(encoding='utf-8')
            self.assertNotIn('needs.build.outputs', workflow, relative_path)
            for job_name in job_names:
                with self.subTest(workflow=relative_path, job=job_name):
                    job = _job_block(workflow, job_name)
                    self.assertNotIn('needs: build', job)
                    self.assertIn('permissions:\n      actions: read\n'
                                  '      contents: read', job)
                    self.assertIn('image_path: images_${{ github.run_id }}',
                                  job)
                    self.assertIn('image_tag: latest', job)
                    self.assertIn('image_registry: kind-registry:5000', job)

    def test_deploy_waits_before_downloading_images(self):
        deploy_action = (ROOT / '.github/actions/deploy/action.yml').read_text(
            encoding='utf-8')

        wait_position = deploy_action.index(
            'run: ./.github/resources/scripts/wait-for-image-artifacts.sh')
        download_position = deploy_action.index(
            '- name: Download Docker Images')
        self.assertLess(wait_position, download_position)
        self.assertIn('GH_TOKEN: ${{ github.token }}', deploy_action)


if __name__ == '__main__':
    unittest.main()
