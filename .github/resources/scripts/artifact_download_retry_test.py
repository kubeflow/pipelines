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
"""Regression tests for artifact-download retries and image preloading."""

from pathlib import Path
import re
import unittest

REPOSITORY_ROOT = Path(__file__).resolve().parents[3]
RETRY_ACTION = (
    REPOSITORY_ROOT / '.github/actions/download-artifact-with-retry/action.yml')
DEPLOY_ACTION = REPOSITORY_ROOT / '.github/actions/deploy/action.yml'
IMAGE_BUILDS_WORKFLOW = (REPOSITORY_ROOT / '.github/workflows/image-builds.yml')
CREATE_MANIFEST_WORKFLOW = (
    REPOSITORY_ROOT / '.github/workflows/create-manifest.yml')
CI_SCRIPTS_WORKFLOW = REPOSITORY_ROOT / '.github/workflows/ci-scripts-tests.yml'
RUNTIME_BASE_IMAGES = (
    REPOSITORY_ROOT / '.github/resources/runtime-base-images.txt')
E2E_WORKFLOW = REPOSITORY_ROOT / '.github/workflows/e2e-test.yml'
API_SERVER_WORKFLOW = (
    REPOSITORY_ROOT / '.github/workflows/api-server-tests.yml')
MYSQL_DEPLOYMENT = (
    REPOSITORY_ROOT /
    'manifests/kustomize/third-party/mysql/base/mysql-deployment.yaml')
ARGO_DEPLOYMENT_PATCH = (
    REPOSITORY_ROOT / 'manifests/kustomize/third-party/argo/base/'
    'workflow-controller-deployment-patch.yaml')
ARGO_CONFIG_PATCH = (
    REPOSITORY_ROOT / 'manifests/kustomize/third-party/argo/base/'
    'workflow-controller-configmap-patch.yaml')


def runtime_images():
    return {
        line.strip()
        for line in RUNTIME_BASE_IMAGES.read_text(
            encoding='utf-8').splitlines()
        if line.strip() and not line.startswith('#')
    }


def argo_matrix_versions():
    versions = set()
    for workflow in (E2E_WORKFLOW, API_SERVER_WORKFLOW):
        for line in workflow.read_text(encoding='utf-8').splitlines():
            if 'argo_version:' in line:
                versions.update(re.findall(r'"(v\d+\.\d+\.\d+)"', line))
    return versions


class ArtifactDownloadRetryTest(unittest.TestCase):

    def test_retry_action_preserves_inputs_and_output(self):
        action = RETRY_ACTION.read_text(encoding='utf-8')

        self.assertEqual(action.count('uses: actions/download-artifact@v7'), 2)
        self.assertIn("steps.primary.outcome == 'failure'", action)
        self.assertIn('sleep "$RETRY_DELAY_SECONDS"', action)
        self.assertIn('pattern: ${{ inputs.pattern }}', action)
        self.assertIn('merge-multiple: ${{ inputs.merge-multiple }}', action)
        self.assertIn('github-token: ${{ inputs.github-token }}', action)
        self.assertIn('run-id: ${{ inputs.run-id }}', action)
        self.assertIn(
            'steps.primary.outputs.download-path || '
            'steps.retry.outputs.download-path',
            action,
        )
        self.assertIn(
            '[[ "$PRIMARY_OUTCOME" == "success" || '
            '"$RETRY_OUTCOME" == "success" ]]',
            action,
        )

    def test_every_artifact_download_uses_retry_action(self):
        callers = (
            DEPLOY_ACTION,
            IMAGE_BUILDS_WORKFLOW,
            CREATE_MANIFEST_WORKFLOW,
        )

        for caller in callers:
            with self.subTest(caller=caller.relative_to(REPOSITORY_ROOT)):
                contents = caller.read_text(encoding='utf-8')
                self.assertIn(
                    'uses: ./.github/actions/download-artifact-with-retry',
                    contents,
                )
                self.assertNotIn(
                    'uses: actions/download-artifact@',
                    contents,
                )

    def test_runtime_archive_contains_external_deployment_images(self):
        images = runtime_images()
        current_version = (REPOSITORY_ROOT /
                           'third_party/argo/VERSION').read_text(
                               encoding='utf-8').strip()
        compatibility_version = (
            REPOSITORY_ROOT /
            'third_party/argo/COMPATIBILITY_VERSION').read_text(
                encoding='utf-8').strip()

        self.assertTrue({
            'docker.io/library/mysql:8.4',
            f'quay.io/argoproj/workflow-controller:{compatibility_version}',
            f'quay.io/argoproj/argoexec:{compatibility_version}',
            f'quay.io/argoproj/workflow-controller:{current_version}',
            f'quay.io/argoproj/argoexec:{current_version}',
        }.issubset(images))

    def test_argo_preloads_follow_ci_matrix_and_pull_policy(self):
        versions = argo_matrix_versions()
        expected_images = {
            f'quay.io/argoproj/{image}:{version}' for version in versions
            for image in ('workflow-controller', 'argoexec')
        }
        preloaded_images = {
            image for image in runtime_images()
            if image.startswith('quay.io/argoproj/')
        }

        self.assertEqual(preloaded_images, expected_images)
        current_version = (REPOSITORY_ROOT /
                           'third_party/argo/VERSION').read_text(
                               encoding='utf-8').strip()
        self.assertIn(current_version, versions)
        api_server_workflow = API_SERVER_WORKFLOW.read_text(encoding='utf-8')
        compatibility_version = re.search(
            r"ARGO_COMPATIBILITY_TESTS:.*matrix\.argo_version == '([^']+)'",
            api_server_workflow,
        )
        self.assertIsNotNone(compatibility_version)
        self.assertEqual(compatibility_version.group(1), current_version)
        deployment = ARGO_DEPLOYMENT_PATCH.read_text(encoding='utf-8')
        self.assertIn(
            f'quay.io/argoproj/workflow-controller:{current_version}',
            deployment,
        )
        self.assertIn(
            f'quay.io/argoproj/argoexec:{current_version}',
            deployment,
        )
        self.assertIn(
            'imagePullPolicy: IfNotPresent',
            ARGO_CONFIG_PATCH.read_text(encoding='utf-8'),
        )

    def test_mysql_preload_matches_active_deployment(self):
        deployment = MYSQL_DEPLOYMENT.read_text(encoding='utf-8')
        match = re.search(r'^\s*image:\s*(mysql:[^\s]+)', deployment,
                          re.MULTILINE)
        self.assertIsNotNone(match)

        canonical_image = f'docker.io/library/{match.group(1)}'
        self.assertIn(canonical_image, runtime_images())
        # A pinned tag with no override uses Kubernetes' IfNotPresent default.
        self.assertNotIn('imagePullPolicy: Always', deployment)

    def test_ci_runs_for_retry_wiring_and_runtime_image_changes(self):
        workflow = CI_SCRIPTS_WORKFLOW.read_text(encoding='utf-8')

        for path_filter in (
                "'.github/actions/download-artifact-with-retry/**'",
                "'.github/actions/deploy/**'",
                "'.github/resources/runtime-base-images.txt'",
                "'.github/workflows/api-server-tests.yml'",
                "'.github/workflows/create-manifest.yml'",
                "'.github/workflows/image-builds.yml'",
                "'manifests/kustomize/third-party/argo/base/"
                "workflow-controller-configmap-patch.yaml'",
                "'manifests/kustomize/third-party/argo/base/"
                "workflow-controller-deployment-patch.yaml'",
                "'manifests/kustomize/third-party/mysql/base/"
                "mysql-deployment.yaml'",
                "'third_party/argo/VERSION'",
        ):
            with self.subTest(path_filter=path_filter):
                self.assertIn(path_filter, workflow)


if __name__ == '__main__':
    unittest.main()
