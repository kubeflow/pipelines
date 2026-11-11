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

import os
from pathlib import Path
import re
import subprocess
import tempfile
import unittest

import yaml

ROOT = Path(__file__).resolve().parents[3]
WORKFLOW_JOBS = {
    '.github/workflows/api-server-tests.yml': (
        'api-test-standalone',
        'api-test-k8s-native',
        'api-test-multi-user',
    ),
    '.github/workflows/e2e-test-frontend.yml': ('frontend-integration-test',),
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
    '.github/workflows/upgrade-test.yml': (
        'upgrade-test',
        'readiness-schedules',
        'readiness-adoption',
        'custom-role-acceptance',
        'readiness-reporting',
    ),
    '.github/workflows/tensorboard-acceptance.yml': ('deployed',),
}


def _job_block(workflow: str, job_name: str) -> str:
    marker = f'  {job_name}:\n'
    start = workflow.index(marker)
    next_job = re.search(r'^  [a-zA-Z0-9_-]+:\n',
                         workflow[start + len(marker):], re.MULTILINE)
    end = (
        start + len(marker) + next_job.start() if next_job else len(workflow))
    return workflow[start:end]


class ImageArtifactReadinessTest(unittest.TestCase):

    def test_all_deploy_jobs_depend_on_the_single_build_readiness_check(self):
        callers = set()
        for path in (ROOT / '.github/workflows').glob('*.y*ml'):
            workflow = yaml.safe_load(path.read_text())
            jobs = workflow.get('jobs', {})
            for name, job in jobs.items():
                deployments = [
                    step for step in job.get('steps', [])
                    if step.get('uses') == './.github/actions/deploy'
                ]
                if not deployments:
                    continue
                relative = str(path.relative_to(ROOT))
                callers.add((relative, name))
                with self.subTest(workflow=relative, job=name):
                    self.assertEqual(job['needs'], 'build')
                    # The ordinary success condition prevents failed, cancelled
                    # or skipped producer jobs from enabling deployment.
                    for status_function in ('always(', 'failure(',
                                            'cancelled('):
                        self.assertNotIn(status_function, job.get('if', ''))
                    self.assertNotIn('continue-on-error', job)
                    self.assertEqual(jobs['build']['uses'],
                                     './.github/workflows/image-builds.yml')
                    for step in deployments:
                        self.assertEqual(
                            step['with']['image_artifacts_ready'],
                            '${{ needs.build.outputs.IMAGE_ARTIFACTS_READY }}')
                        self.assertEqual(step['with']['image_path'],
                                         'images_${{ github.run_id }}')
                        self.assertEqual(step['with']['image_tag'], 'latest')
                        self.assertEqual(step['with']['image_registry'],
                                         'kind-registry:5000')
        self.assertEqual(callers, {(path, name)
                                   for path, names in WORKFLOW_JOBS.items()
                                   for name in names})

    def test_readiness_depends_on_both_successful_producers(self):
        workflow = yaml.safe_load(
            (ROOT / '.github/workflows/image-builds.yml').read_text())
        job = workflow['jobs']['image-artifacts-ready']
        self.assertEqual(
            set(job['needs']), {'image-build', 'runtime-base-images'})
        for name in ('image-artifacts-ready', 'image-build',
                     'runtime-base-images'):
            self.assertNotIn('if', workflow['jobs'][name])
            self.assertNotIn('continue-on-error', workflow['jobs'][name])
        self.assertNotIn('strategy', job)
        self.assertEqual(job['outputs']['ready'],
                         '${{ steps.ready.outputs.ready }}')
        call = workflow[True]['workflow_call']
        self.assertEqual(call['outputs']['IMAGE_ARTIFACTS_READY']['value'],
                         '${{ jobs.image-artifacts-ready.outputs.ready }}')
        waits = [
            step for step in job['steps']
            if 'wait-for-image-artifacts.sh' in step.get('run', '')
        ]
        self.assertEqual(len(waits), 1)
        self.assertEqual(waits[0]['env']['WAIT_ATTEMPTS'], '1')

    def test_readiness_timeout_covers_api_recovery_and_setup(self):
        workflow = yaml.safe_load(
            (ROOT / '.github/workflows/image-builds.yml').read_text())
        waiter = (ROOT / '.github/resources/scripts/wait-for-image-artifacts.sh'
                 ).read_text()
        default = re.search(r'ARTIFACT_API_TIMEOUT_SECONDS:-([0-9]+)', waiter)
        self.assertIsNotNone(default)
        recovery_seconds = int(default.group(1))
        job_seconds = workflow['jobs']['image-artifacts-ready'][
            'timeout-minutes'] * 60
        self.assertGreaterEqual(job_seconds, recovery_seconds + 5 * 60)

    def test_readiness_receipt_requires_successful_inventory(self):
        workflow = yaml.safe_load(
            (ROOT / '.github/workflows/image-builds.yml').read_text())
        step = next(
            step for step in workflow['jobs']['image-artifacts-ready']['steps']
            if step.get('id') == 'ready')
        self.assertNotIn('continue-on-error', step)
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            script = root / '.github/resources/scripts/wait-for-image-artifacts.sh'
            script.parent.mkdir(parents=True)
            script.write_text('#!/usr/bin/env bash\nexit "$WAIT_RESULT"\n')
            script.chmod(0o755)
            for status in (0, 1):
                with self.subTest(inventory_status=status):
                    output = root / f'output-{status}'
                    result = subprocess.run(
                        ['bash', '-c', step['run']],
                        cwd=root,
                        check=False,
                        env={
                            **os.environ, 'WAIT_RESULT': str(status),
                            'GITHUB_OUTPUT': str(output)
                        })
                    self.assertEqual(result.returncode, status)
                    if status == 0:
                        self.assertEqual(output.read_text(), 'ready=true\n')
                    else:
                        self.assertFalse(output.exists())

    def test_viewer_workflows_include_tensorboard_in_shared_readiness(self):
        for relative in WORKFLOW_JOBS:
            workflow = yaml.safe_load((ROOT / relative).read_text())
            jobs = workflow['jobs']
            viewer = any(
                step.get('uses') == './.github/actions/deploy' and
                step.get('with', {}).get('load_tensorboard_image') == 'true'
                for job in jobs.values()
                for step in job.get('steps', []))
            with self.subTest(workflow=relative):
                self.assertEqual(
                    jobs['build'].get('with', {}).get('load_tensorboard_image',
                                                      False), viewer)
        workflow = yaml.safe_load(
            (ROOT / '.github/workflows/image-builds.yml').read_text())
        self.assertFalse(workflow[True]['workflow_call']['inputs']
                         ['load_tensorboard_image']['default'])
        ready = next(
            step for step in workflow['jobs']['image-artifacts-ready']['steps']
            if step.get('id') == 'ready')
        self.assertEqual(ready['env']['LOAD_TENSORBOARD_IMAGE'],
                         '${{ inputs.load_tensorboard_image }}')

    def test_standalone_deploy_still_waits_before_downloading(self):
        deploy = yaml.safe_load(
            (ROOT / '.github/actions/deploy/action.yml').read_text())
        self.assertEqual(deploy['inputs']['image_artifacts_ready']['default'],
                         'false')
        steps = deploy['runs']['steps']
        wait = next(
            step for step in steps
            if step.get('name') == 'Wait for branch image artifacts')
        download = next(
            step for step in steps
            if step.get('name') == 'Download Docker Images')
        self.assertLess(steps.index(wait), steps.index(download))
        self.assertEqual(wait['if'],
                         "${{ inputs.image_artifacts_ready != 'true' }}")
        self.assertEqual(wait['env']['GH_TOKEN'], '${{ github.token }}')

    def test_all_barrier_callers_trigger_workflow_graph_tests(self):
        workflow = yaml.safe_load(
            (ROOT / '.github/workflows/ci-scripts-tests.yml').read_text())
        paths = workflow[True]['pull_request']['paths']
        for path in WORKFLOW_JOBS:
            self.assertIn(path, paths)
        self.assertIn('.github/workflows/image-builds.yml', paths)
        self.assertIn('.github/actions/deploy/**', paths)

    def test_multi_user_artifact_proxy_lane_uses_critical_shards(self):
        workflow = (ROOT / '.github/workflows/e2e-test.yml').read_text(
            encoding='utf-8')
        job = _job_block(workflow,
                         'end-to-end-critical-scenario-multi-user-tests')

        self.assertIn("'[\"E2ECriticalShardA\", \"E2ECriticalShardB\"]'", job)
        self.assertNotIn('        exclude:', job)
        self.assertIn('artifact_proxy: ${{ matrix.cache_enabled }}', job)
        self.assertIn("matrix.cache_enabled == 'true'", job)
        self.assertIn('Multi User ${{ matrix.test_label }} Tests', job)
        self.assertIn('E2EMultiUser${{ matrix.test_label }}Tests', job)


if __name__ == '__main__':
    unittest.main()
