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

from itertools import product
import json
from pathlib import Path
import re
import unittest

import yaml

ROOT = Path(__file__).resolve().parents[3]
WORKFLOW_JOBS = {
    '.github/workflows/legacy-native-transfer.yml': ('destination',),
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
    end = (
        start + len(marker) + next_job.start() if next_job else len(workflow))
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
                    self.assertIn(
                        'permissions:\n      actions: read\n'
                        '      contents: read', job)
                    self.assertIn('image_path: images_${{ github.run_id }}',
                                  job)
                    self.assertIn('image_tag: latest', job)
                    self.assertIn('image_registry: kind-registry:5000', job)

    def test_deploy_waits_before_downloading_images(self):
        deploy_action = yaml.safe_load(
            (ROOT /
             '.github/actions/deploy/action.yml').read_text(encoding='utf-8'))
        steps = deploy_action['runs']['steps']
        wait_steps = [
            step for step in steps
            if './.github/resources/scripts/wait-for-image-artifacts.sh' in
            [line.strip() for line in step.get('run', '').splitlines()]
        ]
        download_steps = [
            step for step in steps if step.get('uses') ==
            './.github/actions/download-artifact-with-retry'
        ]
        self.assertEqual(len(wait_steps), 1)
        self.assertEqual(len(download_steps), 1)
        wait = wait_steps[0]
        self.assertLess(steps.index(wait), steps.index(download_steps[0]))
        self.assertEqual(
            wait.get('env', {}).get('GH_TOKEN'), '${{ github.token }}')

    def test_multi_user_artifact_proxy_lane_uses_critical_shards(self):
        workflow = (ROOT / '.github/workflows/e2e-test.yml').read_text(
            encoding='utf-8')
        job = _job_block(workflow,
                         'end-to-end-critical-scenario-multi-user-tests')

        self.assertIn("'[\"E2ECriticalShardA\", \"E2ECriticalShardB\"]'", job)
        matrix = yaml.safe_load(
            job
        )['end-to-end-critical-scenario-multi-user-tests']['strategy']['matrix']
        self.assertEqual(matrix['exclude'], [{
            'db_type': 'pgx',
            'cache_enabled': 'false'
        }])
        self.assertIn('db_type: "pgx"', job)
        # pgx include overrides artifact_proxy because the
        # multiuser/postgresql/artifact-proxy overlay does not exist yet.
        # Revert to assertIn('artifact_proxy: ${{ matrix.cache_enabled }}', job)
        # once that overlay is added and the include can drop its override.
        self.assertIn(
            'artifact_proxy: ${{ matrix.artifact_proxy || matrix.cache_enabled }}',
            job)
        self.assertIn(
            "(matrix.artifact_proxy || matrix.cache_enabled) == 'true'", job)
        self.assertIn('Multi User ${{ matrix.test_label }} Tests', job)
        self.assertIn('E2EMultiUser${{ matrix.test_label }}Tests', job)

    def test_multi_user_database_shards_preserve_coverage_and_report_names(
            self):
        workflow = yaml.safe_load(
            (ROOT /
             '.github/workflows/e2e-test.yml').read_text(encoding='utf-8'))
        job = workflow['jobs']['end-to-end-critical-scenario-multi-user-tests']
        matrix = job['strategy']['matrix']
        self.assertEqual(matrix['db_type'], ['mysql', 'pgx'])
        self.assertEqual(matrix['multi_user'], ['true'])
        self.assertEqual(matrix['include'], [{
            'db_type': 'pgx',
            'artifact_proxy': 'false'
        }])
        # Read this workflow's two literal label branches, without implementing
        # a general GitHub Actions expression evaluator.
        branches = re.fullmatch(
            r"\$\{\{ fromJSON\(github\.event_name == 'workflow_dispatch' && "
            r"'([^']+)' \|\| '([^']+)'\) \}\}", matrix['test_label'])
        self.assertIsNotNone(branches)
        report_template = next(
            step for step in job['steps']
            if step.get('id') == 'test-run')['with']['report_name']
        expected_configs = {
            ('mysql', 'true', 'true'),
            ('mysql', 'false', 'false'),
            ('pgx', 'true', 'false'),
        }
        for branch, expected_labels in ((1, ['E2ECritical']), (2, [
                'E2ECriticalShardA', 'E2ECriticalShardB'
        ])):
            with self.subTest(manual_dispatch=branch == 1):
                labels = json.loads(branches.group(branch))
                self.assertEqual(labels, expected_labels)
                rows = []
                for db_type, cache_enabled, label in product(
                        matrix['db_type'], matrix['cache_enabled'], labels):
                    row = dict(
                        db_type=db_type,
                        cache_enabled=cache_enabled,
                        test_label=label,
                        multi_user='true',
                        k8s_version=matrix['k8s_version'][0])
                    if any(
                            all(row[key] == value
                                for key, value in rule.items())
                            for rule in matrix['exclude']):
                        continue
                    if db_type == 'pgx':
                        row.update(matrix['include'][0])
                    rows.append(row)
                self.assertCountEqual([(row['db_type'], row['cache_enabled'],
                                        row.get('artifact_proxy') or
                                        row['cache_enabled'], row['test_label'])
                                       for row in rows],
                                      [(*config, label)
                                       for config in expected_configs
                                       for label in expected_labels])
                reports = [
                    re.sub(r'\$\{\{ matrix\.(\w+) \}\}',
                           lambda match: row[match.group(1)], report_template)
                    for row in rows
                ]
                self.assertFalse(any('${{' in report for report in reports))
                self.assertEqual(len(reports), len(set(reports)))


if __name__ == '__main__':
    unittest.main()
