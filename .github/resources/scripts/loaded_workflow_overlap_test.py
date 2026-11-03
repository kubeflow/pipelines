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

import json
import re
import unittest
from itertools import product
from pathlib import Path

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
    '.github/workflows/kfp-kubernetes-native-migration-tests.yaml': (
        'kfp-kubernetes-native-migration-tests',
    ),
    '.github/workflows/kfp-sdk-client-tests.yml': ('sdk-client-tests',),
    '.github/workflows/kfp-webhooks.yml': ('webhook-tests',),
    '.github/workflows/upgrade-test.yml': ('upgrade-test',),
}


def _job_block(workflow: str, job_name: str) -> str:
    marker = f'  {job_name}:\n'
    start = workflow.index(marker)
    next_job = re.search(
        r'^  [a-zA-Z0-9_-]+:\n', workflow[start + len(marker) :], re.MULTILINE
    )
    end = start + len(marker) + next_job.start() if next_job else len(workflow)
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
                    parsed = yaml.safe_load(job)[job_name]
                    self.assertNotIn('needs', parsed)
                    steps = parsed['steps']
                    cluster = next(
                        i
                        for i, step in enumerate(steps)
                        if step.get('uses')
                        == './.github/actions/create-cluster'
                    )
                    deploy = next(
                        i
                        for i, step in enumerate(steps)
                        if step.get('uses') == './.github/actions/deploy'
                    )
                    self.assertLess(cluster, deploy)
                    self.assertIn(
                        'permissions:\n      actions: read\n'
                        '      contents: read',
                        job,
                    )
                    self.assertIn(
                        'image_path: images_${{ github.run_id }}', job
                    )
                    self.assertIn('image_tag: latest', job)
                    self.assertIn('image_registry: kind-registry:5000', job)

    def test_runtime_import_overlaps_branch_builds_without_skipping_images(
        self,
    ):
        deploy_action = yaml.safe_load(
            (ROOT / '.github/actions/deploy/action.yml').read_text(
                encoding='utf-8'
            )
        )
        steps = deploy_action['runs']['steps']
        downloads = [
            step
            for step in steps
            if step.get('uses') == './.github/actions/shared-ci-images'
        ]
        self.assertEqual(len(downloads), 2)
        runtime_download, branch_download = downloads
        self.assertEqual(runtime_download['with']['kind'], 'runtime')
        self.assertEqual(runtime_download['with']['layout'], 'flat')
        self.assertEqual(
            branch_download['with'].get('kind', 'images'), 'images'
        )
        self.assertEqual(branch_download['with']['layout'], 'nested')
        for download in downloads:
            self.assertEqual(download['with']['architecture'], 'amd64')
            self.assertNotIn('continue-on-error', download)
            # Both invocations use the same producer-state default.
            self.assertNotIn('state', download['with'])
        cache_restore = next(
            step for step in steps if step.get('id') == 'runtime-image-cache'
        )
        cache_stage = next(
            step
            for step in steps
            if step.get('id') == 'runtime-image-cache-files'
        )
        self.assertTrue(
            cache_restore['uses'].startswith('actions/cache/restore@')
        )
        self.assertIs(cache_restore.get('continue-on-error'), True)
        self.assertEqual(
            cache_stage['env']['CACHE_OUTCOME'],
            '${{ steps.runtime-image-cache.outcome }}',
        )
        self.assertEqual(
            cache_stage['env']['CACHE_HIT'],
            '${{ steps.runtime-image-cache.outputs.cache-hit }}',
        )
        self.assertEqual(
            cache_stage['env']['IMAGE_DIRECTORY'],
            runtime_download['with']['path'],
        )
        self.assertLess(steps.index(cache_restore), steps.index(cache_stage))
        self.assertLess(steps.index(cache_stage), steps.index(runtime_download))
        self.assertEqual(
            runtime_download['if'],
            "steps.runtime-image-cache-files.outputs.ready != 'true'",
        )
        self.assertNotIn('if', branch_download)
        self.assertEqual(
            runtime_download['with']['path'],
            '${{ inputs.image_path }}/runtime-base-images',
        )
        self.assertEqual(
            branch_download['with']['path'], '${{ inputs.image_path }}'
        )
        runtime_load = next(
            step
            for step in steps
            if step.get('name') == 'Load Runtime Base Images Into Kind'
        )
        modelcar_load = next(
            step
            for step in steps
            if step.get('name') == 'Load Modelcar Fixture Into Kind'
        )
        self.assertNotIn('if', runtime_load)
        self.assertIn('verify_runtime_base_images_in_kind', runtime_load['run'])
        self.assertEqual(
            modelcar_load['if'], "${{ inputs.load_modelcar_fixture == 'true' }}"
        )
        for load in (runtime_load, modelcar_load):
            self.assertLess(steps.index(runtime_download), steps.index(load))
            self.assertLess(steps.index(load), steps.index(branch_download))
        for name in (
            'Load Control Plane Images Into Kind',
            'Load Driver and Launcher Images Into Kind',
        ):
            load = next(step for step in steps if step.get('name') == name)
            self.assertLess(steps.index(branch_download), steps.index(load))

    def test_multi_user_artifact_proxy_lane_uses_critical_shards(self):
        workflow = (ROOT / '.github/workflows/e2e-test.yml').read_text(
            encoding='utf-8'
        )
        job = _job_block(
            workflow, 'end-to-end-critical-scenario-multi-user-tests'
        )

        self.assertIn('\'["E2ECriticalShardA", "E2ECriticalShardB"]\'', job)
        matrix = yaml.safe_load(job)[
            'end-to-end-critical-scenario-multi-user-tests'
        ]['strategy']['matrix']
        self.assertEqual(
            matrix['exclude'], [{'db_type': 'pgx', 'cache_enabled': 'false'}]
        )
        self.assertIn('db_type: "pgx"', job)
        # pgx include overrides artifact_proxy because the
        # multiuser/postgresql/artifact-proxy overlay does not exist yet.
        # Revert to assertIn('artifact_proxy: ${{ matrix.cache_enabled }}', job)
        # once that overlay is added and the include can drop its override.
        self.assertIn(
            'artifact_proxy: ${{ matrix.artifact_proxy || matrix.cache_enabled }}',
            job,
        )
        self.assertIn(
            "(matrix.artifact_proxy || matrix.cache_enabled) == 'true'", job
        )
        self.assertIn('Multi User ${{ matrix.test_label }} Tests', job)
        self.assertIn('E2EMultiUser${{ matrix.test_label }}Tests', job)

    def test_multi_user_database_shards_preserve_coverage_and_report_names(
        self,
    ):
        workflow = yaml.safe_load(
            (ROOT / '.github/workflows/e2e-test.yml').read_text(
                encoding='utf-8'
            )
        )
        job = workflow['jobs']['end-to-end-critical-scenario-multi-user-tests']
        matrix = job['strategy']['matrix']
        self.assertEqual(matrix['db_type'], ['mysql', 'pgx'])
        self.assertEqual(matrix['multi_user'], ['true'])
        self.assertEqual(
            matrix['include'], [{'db_type': 'pgx', 'artifact_proxy': 'false'}]
        )
        # Read this workflow's two literal label branches, without implementing
        # a general GitHub Actions expression evaluator.
        branches = re.fullmatch(
            r"\$\{\{ fromJSON\(github\.event_name == 'workflow_dispatch' && "
            r"'([^']+)' \|\| '([^']+)'\) \}\}",
            matrix['test_label'],
        )
        self.assertIsNotNone(branches)
        report_template = next(
            step for step in job['steps'] if step.get('id') == 'test-run'
        )['with']['report_name']
        expected_configs = {
            ('mysql', 'true', 'true'),
            ('mysql', 'false', 'false'),
            ('pgx', 'true', 'false'),
        }
        for branch, expected_labels in (
            (1, ['E2ECritical']),
            (2, ['E2ECriticalShardA', 'E2ECriticalShardB']),
        ):
            with self.subTest(manual_dispatch=branch == 1):
                labels = json.loads(branches.group(branch))
                self.assertEqual(labels, expected_labels)
                rows = []
                for db_type, cache_enabled, label in product(
                    matrix['db_type'], matrix['cache_enabled'], labels
                ):
                    row = dict(
                        db_type=db_type,
                        cache_enabled=cache_enabled,
                        test_label=label,
                        multi_user='true',
                        k8s_version=matrix['k8s_version'][0],
                    )
                    if any(
                        all(row[key] == value for key, value in rule.items())
                        for rule in matrix['exclude']
                    ):
                        continue
                    if db_type == 'pgx':
                        row.update(matrix['include'][0])
                    rows.append(row)
                self.assertCountEqual(
                    [
                        (
                            row['db_type'],
                            row['cache_enabled'],
                            row.get('artifact_proxy') or row['cache_enabled'],
                            row['test_label'],
                        )
                        for row in rows
                    ],
                    [
                        (*config, label)
                        for config in expected_configs
                        for label in expected_labels
                    ],
                )
                reports = [
                    re.sub(
                        r'\$\{\{ matrix\.(\w+) \}\}',
                        lambda match: row[match.group(1)],
                        report_template,
                    )
                    for row in rows
                ]
                self.assertFalse(any('${{' in report for report in reports))
                self.assertEqual(len(reports), len(set(reports)))


if __name__ == '__main__':
    unittest.main()
