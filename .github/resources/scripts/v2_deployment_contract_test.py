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
"""Deployment and generation wiring for the supported KFP API."""

import os
import re
import runpy
import subprocess
import tempfile
import unittest
from pathlib import Path

import yaml

ROOT = Path(__file__).resolve().parents[3]


class V2DeploymentContractTest(unittest.TestCase):
    def test_api_and_ui_probes_use_v2_healthz(self):
        for filename, count in (
            ('ml-pipeline-apiserver-deployment.yaml', 3),
            ('ml-pipeline-ui-deployment.yaml', 2),
        ):
            with self.subTest(deployment=filename):
                manifest = (
                    ROOT / 'manifests/kustomize/base/pipeline' / filename
                ).read_text()
                self.assertEqual(
                    manifest.count('path: /apis/v2/healthz'), count
                )
                self.assertNotIn('/apis/v1beta1', manifest)

    def test_modern_generation_only_generates_v2_clients(self):
        for path in (
            'Makefile',
            '.github/workflows/validate-generated-files.yml',
        ):
            with self.subTest(path=path):
                text = (ROOT / path).read_text()
                self.assertIn('v2', text)
                self.assertNotIn('v1beta1', text)

    def test_release_generates_legacy_api_only_when_protos_exist(self):
        release_command = runpy.run_path(str(ROOT / 'release/kfpr/core.py'))[
            'release_version_bump_command'
        ]
        for legacy_protos in (False, True):
            with (
                self.subTest(legacy_protos=legacy_protos),
                tempfile.TemporaryDirectory() as temporary,
            ):
                root = Path(temporary)
                api = root / 'backend/api'
                (api / 'v1beta1').mkdir(parents=True)
                (api / 'v2').mkdir()
                (api / 'v2/run.proto').write_text(
                    (ROOT / 'backend/api/v2/run.proto').read_text()
                )
                if legacy_protos:
                    (api / 'v1beta1/run.proto').write_text('package legacy;')
                for path in (
                    'hack/generator.sh',
                    'build_kfp_server_api_python_package.sh',
                ):
                    generator = api / path
                    generator.parent.mkdir(parents=True, exist_ok=True)
                    generator.write_text(
                        '#!/bin/bash\nprintf "%s\\n" "$API_VERSION"\n'
                    )
                    generator.chmod(0o755)
                script = release_command(root, 'release-3.0', '2.17.3')[-1]
                # Exercise generation after the unrelated tool and manifest checks.
                generation = script.split(
                    '"$REPO_ROOT/manifests/kustomize/hack/release.sh" "$TAG_NAME"\n',
                    1,
                )[1]
                result = subprocess.run(
                    ['bash', '-ec', generation],
                    env={
                        **os.environ,
                        'REPO_ROOT': str(root),
                        'API_VERSION': 'stale',
                    },
                    check=True,
                    capture_output=True,
                    text=True,
                    timeout=10,
                )
                self.assertEqual(
                    result.stdout.splitlines(),
                    (['v1beta1'] if legacy_protos else []) + ['v2', 'v2'],
                )

    def test_backend_dev_targets_submit_ir_from_existing_sources(self):
        makefile = (ROOT / 'backend/src/v2/Makefile').read_text()
        self.assertIn('kfp run submit -f "$${tmp}/$*-spec.yaml"', makefile)
        self.assertIn('build/compiler --spec "$${tmp}/$*-spec.yaml"', makefile)
        self.assertIn('argo lint "$${tmp}/$*.yaml"', makefile)
        targets = re.findall(r'^pipeline/([\w/-]+):$', makefile, re.MULTILINE)
        self.assertTrue(targets)
        for target in targets:
            with self.subTest(target=target):
                source = (
                    ROOT
                    / 'test_data/sdk_compiled_pipelines/valid'
                    / (target + '.py')
                )
                self.assertTrue(source.is_file(), source)

    def test_upgrade_preparation_uses_legacy_prefix_and_gates_verification(
        self,
    ):
        workflow = yaml.safe_load(
            (ROOT / '.github/workflows/upgrade-test.yml').read_text()
        )
        for job in workflow['jobs'].values():
            self.assertEqual(job['if'], '${{ false }}')
        steps = {
            step['name']: step
            for step in workflow['jobs']['upgrade-test']['steps']
        }
        prepare = steps['Prepare for Upgrade']
        self.assertIn('-- -useLegacyAPIPrefix=true', prepare['run'])
        self.assertFalse(prepare.get('continue-on-error', False))
        self.assertFalse(
            steps['Forward API port'].get('continue-on-error', False)
        )
        self.assertIn(
            "steps.prepare-upgrade.outcome == 'success'",
            steps['Deploy from Branch']['if'],
        )
        self.assertIn(
            "steps.prepare-upgrade.outcome == 'success'",
            steps['Verify Upgrade']['if'],
        )
        self.assertNotIn('useLegacyAPIPrefix', str(steps['Verify Upgrade']))
        self.assertEqual(steps['Stop port forwarding']['if'], '${{ always() }}')

    def test_legacy_api_specs_reuse_existing_cluster_jobs(self):
        workflow = yaml.safe_load(
            (ROOT / '.github/workflows/api-server-tests.yml').read_text()
        )
        for name in (
            'api-test-standalone',
            'api-test-k8s-native',
            'api-test-multi-user',
        ):
            with self.subTest(job=name):
                steps = workflow['jobs'][name]['steps']
                self.assertEqual(
                    sum(
                        step.get('uses') == './.github/actions/create-cluster'
                        for step in steps
                    ),
                    1,
                )
                self.assertEqual(
                    sum(
                        step.get('uses') == './.github/actions/deploy'
                        for step in steps
                    ),
                    1,
                )
                tests = next(
                    step for step in steps if step['name'] == 'Run Tests'
                )
                self.assertEqual(
                    tests['uses'], './.github/actions/test-and-report'
                )
                self.assertNotIn(
                    'LEGACY_API_COMPATIBILITY_TESTS', tests.get('env', {})
                )
                self.assertEqual(
                    tests['with']['test_directory'], '${{ env.API_TESTS_DIR }}'
                )
        multi_user = next(
            step
            for step in workflow['jobs']['api-test-multi-user']['steps']
            if step['name'] == 'Configure Input Variables'
        )
        self.assertIn(
            'TEST_LABEL="($TEST_LABEL) || LegacyAPICompatibility"',
            multi_user['run'],
        )
        specs = (
            ROOT / 'backend/test/v2/api/legacy_compatibility_api_test.go'
        ).read_text()
        self.assertNotIn('LEGACY_API_COMPATIBILITY_TESTS', specs)
        self.assertNotIn('Skip(', specs)
        self.assertIn('constants.APIServerTests', specs)
        self.assertNotIn('config.UseLegacyAPIPrefix', specs)

    def test_full_api_version_matrix_is_default_branch_only(self):
        workflow = yaml.safe_load(
            (ROOT / '.github/workflows/api-server-tests.yml').read_text()
        )
        job = workflow['jobs']['api-test-versions']
        self.assertEqual(
            job['if'],
            "${{ github.event_name == 'push' && github.ref == "
            "format('refs/heads/{0}', github.event.repository.default_branch) }}",
        )
        self.assertEqual(
            job['strategy']['matrix']['api_version'], ['v2', 'v2beta1']
        )
        self.assertEqual(
            set(job['strategy']['matrix']), {'api_version', 'k8s_version'}
        )
        self.assertEqual(len(job['strategy']['matrix']['k8s_version']), 1)
        steps = {step['name']: step for step in job['steps']}
        self.assertEqual(
            steps['Deploy KFP']['with']['pipeline_store'], 'database'
        )
        self.assertEqual(steps['Deploy KFP']['with']['db_type'], 'mysql')
        tests = steps['Run Tests']
        self.assertEqual(tests['uses'], './.github/actions/test-and-report')
        self.assertEqual(
            tests['with']['test_directory'], '${{ env.API_TESTS_DIR }}'
        )
        self.assertEqual(tests['with']['test_label'], '${{ env.TESTS_LABEL }}')
        self.assertEqual(
            tests['with']['use_legacy_api_prefix'],
            "${{ matrix.api_version == 'v2beta1' }}",
        )
        self.assertIn('${{ matrix.api_version }}', tests['with']['report_name'])

    def test_shared_action_passes_explicit_legacy_prefix_flag(self):
        action = yaml.safe_load(
            (ROOT / '.github/actions/test-and-report/action.yml').read_text()
        )
        self.assertEqual(
            action['inputs']['use_legacy_api_prefix']['default'], 'false'
        )
        step = next(
            step
            for step in action['runs']['steps']
            if step['name'] == 'Run Tests'
        )
        self.assertEqual(
            step['env']['INPUT_USE_LEGACY_API_PREFIX'],
            '${{ inputs.use_legacy_api_prefix }}',
        )
        script = step['run']
        prefix = script[
            script.index('API_PREFIX_FLAGS=()') : script.index(
                'BASE_IMAGE_FLAG=""'
            )
        ]
        self.assertIn(
            '"${API_PREFIX_FLAGS[@]}"',
            next(
                line
                for line in script.splitlines()
                if line.startswith('go run ')
            ),
        )
        for enabled, expected in [
            ('true', '-useLegacyAPIPrefix=true'),
            ('false', ''),
        ]:
            with self.subTest(enabled=enabled):
                result = subprocess.run(
                    [
                        'bash',
                        '--noprofile',
                        '--norc',
                        '-eo',
                        'pipefail',
                        '-c',
                        prefix + '\nprintf "%s" "${API_PREFIX_FLAGS[@]}"',
                    ],
                    env={**os.environ, 'INPUT_USE_LEGACY_API_PREFIX': enabled},
                    text=True,
                    capture_output=True,
                    check=True,
                )
                self.assertEqual(result.stdout, expected)

    def test_generator_rejects_missing_tasks_omitempty_fix(self):
        generator = (ROOT / 'backend/api/hack/generator.sh').read_text()
        guard = generator.split("if ! grep -Eq '^[[:space:]]*Tasks", 1)[1]
        guard = (
            "if ! grep -Eq '^[[:space:]]*Tasks"
            + guard.split('\nfi', 1)[0]
            + '\nfi'
        )
        subprocess.run(
            ['bash', '-eu', '-c', guard],
            cwd=ROOT,
            env={**os.environ, 'API_VERSION': 'v2'},
            check=True,
        )
        for field, succeeds in [
            ('Tasks []*V2PipelineTask `json:"tasks,omitempty"`', True),
            ('Tasks []*V2PipelineTask `json:"tasks"`', False),
            ('Tasks []*ChangedTask `json:"tasks,omitempty"`', False),
            ('Other string `json:"other"`', False),
        ]:
            with (
                self.subTest(field=field),
                tempfile.TemporaryDirectory() as directory,
            ):
                model = (
                    Path(directory)
                    / 'backend/api/v2/go_http_client/run_model/v2_run.go'
                )
                model.parent.mkdir(parents=True)
                model.write_text('\t' + field + '\n')
                result = subprocess.run(
                    ['bash', '-eu', '-c', guard],
                    cwd=directory,
                    env={**os.environ, 'API_VERSION': 'v2'},
                    text=True,
                    capture_output=True,
                )
                self.assertEqual(result.returncode == 0, succeeds)
                if not succeeds:
                    self.assertIn(
                        'update the go-swagger compatibility substitution',
                        result.stderr,
                    )

    def test_visualization_service_is_not_deployed_or_generated(self):
        for root in ('manifests/kustomize', '.github/resources/manifests'):
            for path in (ROOT / root).rglob('kustomization.yaml'):
                with self.subTest(path=path):
                    text = path.read_text()
                    for identifier in (
                        'ml-pipeline-visualization',
                        'kfp-visualization-server',
                        'Dockerfile.visualization',
                        'visualizationserver.yaml',
                    ):
                        self.assertNotIn(identifier, text)
        for path in (
            'backend/api/v2/visualization.proto',
            'backend/Dockerfile.visualization',
            'backend/src/apiserver/visualization/server.py',
            'frontend/src/apisv2/visualization',
        ):
            with self.subTest(path=path):
                self.assertFalse((ROOT / path).exists())
        for path in (
            'backend/api/v2/swagger/kfp_api_single_file.swagger.json',
            'docs/_static/kfp_api_single_file.swagger.json',
        ):
            with self.subTest(path=path):
                text = (ROOT / path).read_text()
                self.assertNotIn('VisualizationService', text)
                self.assertNotIn('/visualizations/', text)
        pipeline = (
            ROOT / 'manifests/kustomize/base/pipeline/kustomization.yaml'
        ).read_text()
        self.assertIn('ml-pipeline-viewer-crd-deployment.yaml', pipeline)
        self.assertIn('ml-pipeline-ui-deployment.yaml', pipeline)

    def test_v2_caching_is_configured_without_admission_webhook(self):
        self.assertFalse(
            (
                ROOT / 'manifests/kustomize/base/cache/kustomization.yaml'
            ).exists()
        )
        self.assertFalse(
            (
                ROOT
                / 'manifests/kustomize/base/cache-deployer/kustomization.yaml'
            ).exists()
        )
        overlay = (
            ROOT / '.github/resources/manifests/standalone/'
            'cache-disabled/cache-env.yaml'
        ).read_text()
        self.assertIn('name: CACHEENABLED', overlay)
        self.assertIn('value: "false"', overlay)


if __name__ == '__main__':
    unittest.main()
