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

import unittest
from pathlib import Path

import yaml

ROOT = Path(__file__).resolve().parents[3]
WORKFLOW_PATH = ROOT / '.github/workflows/osv-scanner.yml'
CI_SCRIPTS_WORKFLOW_PATH = ROOT / '.github/workflows/ci-scripts-tests.yml'


class OsvScannerWorkflowTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.workflow = WORKFLOW_PATH.read_text(encoding='utf-8')
        cls.ci_scripts_workflow = CI_SCRIPTS_WORKFLOW_PATH.read_text(
            encoding='utf-8'
        )

    def test_producer_data_never_precedes_executable_steps(self):
        jobs = yaml.safe_load(self.workflow)['jobs']
        discovery = jobs['discover-images']['steps']
        for step in discovery:
            if step.get('uses', '').startswith('actions/checkout@'):
                self.assertEqual(
                    step['with']['ref'], '${{ github.workflow_sha }}'
                )
        self.assertTrue(
            any(
                'osv_manifest_source.py' in step.get('run', '')
                for step in discovery
            )
        )
        scan = jobs['scan-images']['steps']
        producer_index = next(
            i
            for i, step in enumerate(scan)
            if step.get('with', {}).get('path') == 'producer'
        )
        self.assertEqual(producer_index, len(scan) - 2)
        self.assertIn('upload-sarif@', scan[-1]['uses'])

    def test_scanner_release_is_pinned_and_checksum_verified(self):
        self.assertEqual(self.workflow.count("OSV_SCANNER_VERSION: '2.5.0'"), 2)
        scanner_checksum = (
            "OSV_SCANNER_SHA256: 'edcfc41d257db36148f065055655fe3f"
            "cfc434b0b423ea67468a84c207524e0c'"
        )
        self.assertEqual(self.workflow.count(scanner_checksum), 2)
        self.assertIn('sha256sum --check --strict', self.workflow)

    def test_scan_recursively_reports_all_supported_dependencies(self):
        self.assertIn('./osv-scanner scan source', self.workflow)
        self.assertIn('            --recursive', self.workflow)
        self.assertIn('            --no-resolve', self.workflow)
        self.assertIn(
            '            --experimental-exclude backend/api/v2/python_http_client',
            self.workflow,
        )
        self.assertIn('            --format sarif', self.workflow)
        self.assertIn(
            '            --output-file osv-results.sarif', self.workflow
        )
        self.assertIn('            . || scan_exit_code=$?', self.workflow)
        self.assertIn(
            '"${scan_exit_code}" -ne 0 && "${scan_exit_code}" -ne 1',
            self.workflow,
        )

    def test_scan_has_least_privilege_and_manual_dispatch(self):
        self.assertIn('  push:', self.workflow)
        self.assertIn('      - master', self.workflow)
        self.assertIn('  workflow_dispatch:', self.workflow)
        self.assertIn('  contents: read', self.workflow)
        self.assertIn('  security-events: write', self.workflow)
        self.assertEqual(self.workflow.count('      security-events: write'), 3)
        self.assertIn('          persist-credentials: false', self.workflow)
        self.assertNotIn('contents: write', self.workflow)
        self.assertNotIn('pull-requests: write', self.workflow)

    def test_deployed_images_are_discovered_and_scanned(self):
        self.assertIn("KUSTOMIZE_VERSION: '5.8.1'", self.workflow)
        self.assertIn(
            "KUSTOMIZE_SHA256: '029a7f0f4e1932c52a0476cf02a0fd855c0bb85694b82c338fc648dcb53a819d'",
            self.workflow,
        )
        self.assertIn('osv_manifest_images.py', self.workflow)
        self.assertIn(
            '--overlay producer/manifests/kustomize/env/platform-agnostic',
            self.workflow,
        )
        self.assertIn(
            '--overlay producer/manifests/kustomize/env/platform-agnostic-multi-user',
            self.workflow,
        )
        self.assertIn('third_party_image_scan.py scan', self.workflow)
        self.assertIn('--image "$IMAGE" --output evidence', self.workflow)
        self.assertIn(
            'category: kubeflow-pipelines-osv-image-${{ matrix.category }}',
            self.workflow,
        )
        self.assertIn('      fail-fast: false', self.workflow)

    def test_image_versions_are_not_part_of_the_analysis_environment(self):
        workflow = yaml.safe_load(self.workflow)
        discovery = workflow['jobs']['discover-images']
        scan = workflow['jobs']['scan-images']
        self.assertIn('images', discovery['outputs'])
        self.assertNotIn('matrix.image', self.workflow)
        scan_step = next(
            step
            for step in scan['steps']
            if step.get('name')
            == 'Scan immutable image and retain full JSON and SARIF'
        )
        self.assertEqual(
            scan_step['env']['IMAGE'],
            '${{ fromJSON(needs.discover-images.outputs.images)[matrix.category] }}',
        )

    def test_images_follow_successful_master_publication_and_keep_source_pushes(
        self,
    ):
        workflow = yaml.safe_load(self.workflow)
        triggers = workflow.get('on', workflow.get(True))
        self.assertEqual(
            triggers['workflow_run']['workflows'], ['Build and Push for master']
        )
        self.assertEqual(triggers['workflow_run']['types'], ['completed'])
        self.assertEqual(
            triggers['workflow_run']['branches'], ['master', 'main']
        )
        self.assertEqual(triggers['push']['branches'], ['master', 'main'])
        self.assertFalse(workflow['concurrency']['cancel-in-progress'])
        group = workflow['concurrency']['group']
        self.assertIn("github.event_name == 'push' && 'source'", group)
        self.assertIn("conclusion != 'success' && 'ignored'", group)
        self.assertIn(
            "github.event_name != 'workflow_run'",
            workflow['jobs']['scan-source']['if'],
        )
        resolver = workflow['jobs']['resolve-images']
        self.assertIn("github.event_name != 'push'", resolver['if'])
        self.assertEqual(
            resolver['permissions'], {'contents': 'read', 'actions': 'read'}
        )
        discovery = workflow['jobs']['discover-images']
        self.assertEqual(discovery['needs'], 'resolve-images')
        self.assertEqual(
            resolver['outputs']['branch'],
            '${{ steps.producer.outputs.branch }}',
        )
        self.assertEqual(
            discovery['outputs']['branch'],
            '${{ needs.resolve-images.outputs.branch }}',
        )
        self.assertIn('--published-records published-indexes', self.workflow)
        self.assertIn('--source-sha "${SOURCE_SHA}"', self.workflow)
        self.assertIn(
            '--overlay producer/manifests/kustomize/env/platform-agnostic',
            self.workflow,
        )
        download = next(
            step
            for step in discovery['steps']
            if step.get('uses', '').endswith('/download-artifact-with-retry')
        )
        self.assertEqual(
            download['with']['run-id'],
            '${{ needs.resolve-images.outputs.run-id }}',
        )
        self.assertEqual(
            download['with']['pattern'],
            'published-index-*-${{ needs.resolve-images.outputs.run-attempt }}',
        )
        upload = workflow['jobs']['scan-images']['steps'][-1]['with']
        self.assertEqual(
            upload['sha'], '${{ needs.discover-images.outputs.sha }}'
        )
        self.assertEqual(
            upload['ref'],
            'refs/heads/${{ needs.discover-images.outputs.branch }}',
        )
        self.assertEqual(upload['checkout_path'], 'producer')
        publisher = yaml.safe_load(
            (ROOT / '.github/workflows/create-manifest.yml').read_text()
        )
        record_uploads = [
            step
            for job in publisher['jobs'].values()
            for step in job.get('steps', [])
            if step.get('with', {})
            .get('name', '')
            .startswith('published-index-')
        ]
        self.assertEqual(len(record_uploads), 1)
        self.assertEqual(record_uploads[0]['with']['retention-days'], 90)

    def test_discovery_installs_yaml_dependency_before_rendering(self):
        workflow = yaml.safe_load(self.workflow)
        steps = workflow['jobs']['discover-images']['steps']
        install_index = next(
            index
            for index, step in enumerate(steps)
            if step.get('run')
            == 'python3 -m pip install -r .github/scripts/requirements.txt'
        )
        render_index = next(
            index
            for index, step in enumerate(steps)
            if 'osv_manifest_images.py' in step.get('run', '')
        )
        self.assertLess(install_index, render_index)
        self.assertTrue(
            any(
                step.get('uses', '').startswith('actions/setup-python@')
                for step in steps[:install_index]
            )
        )

    def test_images_are_summarized_but_source_findings_remain_individual(self):
        self.assertEqual(self.workflow.count('deduplicate_sarif.py'), 1)
        self.assertIn(
            'sarif_file: osv-results-deduplicated.sarif', self.workflow
        )
        self.assertIn('sarif_file: image-summary.sarif', self.workflow)
        self.assertIn('python -m kfpr.image_summary', self.workflow)
        self.assertIn('--input evidence/github/scan.sarif', self.workflow)
        self.assertIn('third_party_image_scan.py scan', self.workflow)
        steps = yaml.safe_load(self.workflow)['jobs']['scan-images']['steps']
        upload = next(
            step for step in steps if 'upload-sarif' in step.get('uses', '')
        )
        self.assertEqual(
            upload['if'],
            "github.ref == format('refs/heads/{0}', github.event.repository.default_branch)",
        )
        summary = next(
            step
            for step in steps
            if 'kfpr.image_summary' in step.get('run', '')
        )
        self.assertNotIn('if', summary)
        evidence = next(
            step
            for step in steps
            if 'actions/upload-artifact@' in step.get('uses', '')
        )
        self.assertEqual(evidence['if'], 'always()')
        self.assertIn('evidence', evidence['with']['path'])
        self.assertLess(steps.index(evidence), steps.index(upload))

    def test_ci_scripts_tests_run_for_osv_workflow_changes(self):
        self.assertIn(
            "      - '.github/workflows/osv-scanner.yml'",
            self.ci_scripts_workflow,
        )


if __name__ == '__main__':
    unittest.main()
