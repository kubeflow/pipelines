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
"""Keep CI tokens independent of repository defaults and bound writer jobs."""

import re
import unittest
from pathlib import Path

import yaml

ROOT = Path(__file__).resolve().parents[3]
WORKFLOWS = ROOT / '.github/workflows'

# Each write grant must have an identified consumer. The generated analyzer's
# issue output, cancellation, and inference grants stay in its source contract.
WRITE_PERMISSIONS = {
    'release-branch-image-scanning.yml': {'upload': {'security-events'}},
    'release-promote-latest.yml': {'promote': {'packages'}},
    'ai-analyzer.lock.yml': {
        'activation': {'issues'},
        'agent': {'copilot-requests'},
        'classify_title_event': {'actions'},
        'conclusion': {'issues'},
        'detection': {'copilot-requests'},
        'pre_activation': {'actions', 'issues'},
        'safe_outputs': {'issues'},
    },
    'build-and-push.yml': {
        'build-and-push-images': {'packages', 'attestations', 'id-token'},
    },
    'build-tools-images.yml': {
        'build-tools': {'packages'},
        'publish-tools': {'packages'},
    },
    'ci-checks.yml': {
        'check_ci_status': {'pull-requests', 'statuses'},
    },
    'ci-health-report.yml': {
        'report': {'issues'},
    },
    'ci-passed-stale-revoke.yml': {
        'revoke_stale': {'statuses'},
    },
    'codeql.yml': {
        'analyze': {'security-events'},
    },
    'contributor-report.yml': {
        'contributor-report': {'pull-requests'},
    },
    'create-manifest.yml': {
        'create-manifest': {'packages'},
    },
    'gh-workflow-approve.yml': {
        'ok-to-test': {'actions'},
        'scheduled-recovery': {'actions'},
    },
    'image-builds-master.yml': {
        'build-images-for-master': {'packages', 'attestations', 'id-token'},
        'create-manifests': {'packages'},
    },
    'image-builds-release.yml': {
        'build-images-for-release': {'packages', 'attestations', 'id-token'},
        'create-manifests': {'packages'},
    },
    'osv-scanner.yml': {
        'scan-source': {'security-events'},
        'scan-images': {'security-events'},
    },
    'pr-gate.yml': {
        'check-pr-author': {'actions', 'pull-requests'},
    },
    'publish-packages.yml': {
        'publish': {'id-token'},
    },
    'runtime-base-images.yml': {
        'prune-runtime-base-image-caches': {'actions'},
    },
    'stale.yml': {
        'stale': {'issues', 'pull-requests'},
    },
}


def effective_permissions(workflow, job):
    # Job permissions replace the workflow map; unspecified scopes become none.
    permissions = job.get('permissions', workflow.get('permissions'))
    if not isinstance(permissions, dict):
        raise AssertionError(
            'Declare an explicit permissions map; do not '
            'inherit repository defaults or use read/write-all'
        )
    if not all(
        value in {'none', 'read', 'write'} for value in permissions.values()
    ):
        raise AssertionError(f'Invalid token permissions: {permissions}')
    return permissions


class WorkflowPermissionsTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.workflows = {
            path.name: yaml.safe_load(path.read_text(encoding='utf-8'))
            for path in sorted(WORKFLOWS.iterdir())
            if path.suffix in {'.yml', '.yaml'}
        }

    def permissions(self, workflow_name, job_name):
        workflow = self.workflows[workflow_name]
        return effective_permissions(workflow, workflow['jobs'][job_name])

    def test_every_job_has_explicit_permissions_and_only_expected_writes(self):
        actual_writers = {}
        for workflow_name, workflow in self.workflows.items():
            for job_name, job in workflow['jobs'].items():
                with self.subTest(workflow=workflow_name, job=job_name):
                    permissions = effective_permissions(workflow, job)
                    writes = {
                        scope
                        for scope, level in permissions.items()
                        if level == 'write'
                    }
                    if writes:
                        actual_writers.setdefault(workflow_name, {})[
                            job_name
                        ] = writes
        self.assertEqual(actual_writers, WRITE_PERMISSIONS)

    def test_reusable_callers_allow_each_callee_required_permission(self):
        rank = {'none': 0, 'read': 1, 'write': 2}
        prefix = './.github/workflows/'
        for workflow_name, workflow in self.workflows.items():
            for job_name, job in workflow['jobs'].items():
                reference = job.get('uses', '')
                if not reference.startswith(prefix):
                    continue
                caller = effective_permissions(workflow, job)
                callee = self.workflows[reference.removeprefix(prefix)]
                for callee_job_name, callee_job in callee['jobs'].items():
                    with self.subTest(
                        workflow=workflow_name,
                        job=job_name,
                        callee=callee_job_name,
                    ):
                        required = effective_permissions(callee, callee_job)
                        for scope, level in required.items():
                            self.assertGreaterEqual(
                                rank[caller.get(scope, 'none')],
                                rank[level],
                                f'{reference} needs {scope}: {level}',
                            )

    def test_checkouts_do_not_leave_credentials_for_later_commands(self):
        documents = list(self.workflows.items())
        for path in (ROOT / '.github/actions').rglob('action.y*ml'):
            documents.append(
                (
                    str(path.relative_to(ROOT)),
                    yaml.safe_load(path.read_text(encoding='utf-8')),
                )
            )
        for name, document in documents:
            jobs = document.get('jobs', {'composite': document.get('runs', {})})
            for job_name, job in jobs.items():
                for step in job.get('steps', []):
                    if step.get('uses', '').split('@')[0] != 'actions/checkout':
                        continue
                    with self.subTest(document=name, job=job_name):
                        self.assertIs(
                            step.get('with', {}).get('persist-credentials'),
                            False,
                        )

    def test_external_actions_and_reusable_workflows_use_immutable_refs(self):
        documents = list(self.workflows.items())
        for path in (ROOT / '.github/actions').rglob('action.y*ml'):
            documents.append(
                (
                    str(path.relative_to(ROOT)),
                    yaml.safe_load(path.read_text(encoding='utf-8')),
                )
            )
        for name, document in documents:
            jobs = document.get('jobs', {'composite': document.get('runs', {})})
            for job_name, job in jobs.items():
                for item in (job, *job.get('steps', [])):
                    reference = item.get('uses', '')
                    if not reference or reference.startswith('./'):
                        continue
                    with self.subTest(
                        document=name, job=job_name, reference=reference
                    ):
                        self.assertIsNotNone(
                            re.fullmatch(r'[^@\s]+@[0-9a-f]{40}', reference)
                        )

    def test_test_image_builds_and_callers_are_read_only(self):
        self.assertEqual(
            self.permissions('image-builds.yml', 'image-build'),
            {'contents': 'read'},
        )
        self.assertEqual(
            self.permissions('image-builds.yml', 'runtime-base-images'),
            {
                'contents': 'read',
                'actions': 'read',
            },
        )
        for workflow_name, workflow in self.workflows.items():
            for job_name, job in workflow['jobs'].items():
                if job.get('uses') == './.github/workflows/image-builds.yml':
                    with self.subTest(workflow=workflow_name, job=job_name):
                        self.assertEqual(
                            effective_permissions(workflow, job),
                            {
                                'contents': 'read',
                                'actions': 'read',
                            },
                        )
        image_workflow = self.workflows['image-builds.yml']
        self.assertFalse(
            {'GITHUB_TOKEN', 'GH_TOKEN'} & image_workflow.get('env', {}).keys()
        )
        for job in image_workflow['jobs'].values():
            self.assertFalse(
                {'GITHUB_TOKEN', 'GH_TOKEN'} & job.get('env', {}).keys()
            )

    def test_deployment_jobs_keep_artifact_polling_access(self):
        for workflow_name, workflow in self.workflows.items():
            for job_name, job in workflow['jobs'].items():
                if not any(
                    step.get('uses') == './.github/actions/deploy'
                    for step in job.get('steps', [])
                ):
                    continue
                with self.subTest(workflow=workflow_name, job=job_name):
                    self.assertEqual(
                        effective_permissions(workflow, job),
                        {
                            'contents': 'read',
                            'actions': 'read',
                        },
                    )

    def test_metadata_jobs_do_not_request_unrelated_scopes(self):
        self.assertNotIn(
            'issues', self.permissions('ci-checks.yml', 'check_ci_status')
        )
        self.assertEqual(
            self.permissions('pr-gate.yml', 'check-pr-author')['issues'], 'read'
        )
        for job_name in self.workflows['gh-workflow-approve.yml']['jobs']:
            self.assertEqual(
                self.permissions('gh-workflow-approve.yml', job_name),
                {
                    'actions': 'write',
                    'pull-requests': 'read',
                },
            )

    def test_summary_and_same_run_artifact_jobs_need_no_extra_token_scopes(
        self,
    ):
        self.assertEqual(
            self.permissions(
                'validate-generated-files.yml', 'validate-generated-files'
            ),
            {},
        )
        self.assertEqual(
            self.permissions('arm64-presubmit.yml', 'smoke'),
            {'contents': 'read'},
        )
        self.assertEqual(
            self.permissions(
                'runtime-base-images.yml', 'produce-runtime-base-images'
            ),
            {'contents': 'read'},
        )

    def test_fixture_backed_integration_tests_do_not_receive_a_github_token(
        self,
    ):
        action = yaml.safe_load(
            (ROOT / '.github/actions/test-and-report/action.yml').read_text()
        )
        fixture_steps = [
            step
            for step in action['runs']['steps']
            if 'KFP_TEST_FIXTURE_BASE_URL' in step.get('env', {})
        ]
        self.assertTrue(fixture_steps)
        for step in fixture_steps:
            self.assertFalse({'GITHUB_TOKEN', 'GH_TOKEN'} & step['env'].keys())


if __name__ == '__main__':
    unittest.main()
