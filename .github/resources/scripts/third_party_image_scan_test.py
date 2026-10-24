#!/usr/bin/env python3
# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
# http://www.apache.org/licenses/LICENSE-2.0
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import json
import os
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path
from unittest import mock

import third_party_image_scan as scanner
import yaml

ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT / 'release'))


class ThirdPartyInventoryTest(unittest.TestCase):
    def setUp(self):
        self.sources = {'master': 'a' * 40, 'release-2.18': 'b' * 40}

    def test_shared_exact_refs_scan_once_and_candidates_keep_branch_identity(
        self,
    ):
        common = 'mysql:8.4@sha256:' + 'c' * 64
        old, new = (
            'registry.example:5000/object:4',
            'registry.example:5000/object:5',
        )
        result = scanner.create_inventory(
            self.sources,
            {
                'master': {common, new},
                'release-2.18': {common, old},
            },
        )
        self.assertEqual(result['sources'], self.sources)
        self.assertEqual(len(result['scans']), 3)
        rows = {row['image']: row for row in result['images']}
        self.assertEqual(rows[common]['branches'], ['master', 'release-2.18'])
        self.assertEqual(rows[old]['candidate_key'], rows[new]['key'])
        self.assertNotIn('candidate_key', rows[new])
        self.assertEqual(
            {row['image'] for row in result['scans']}, {common, old, new}
        )

    def test_candidate_requires_same_repository(self):
        result = scanner.create_inventory(
            self.sources,
            {
                'master': {'registry.example/new:5'},
                'release-2.18': {'registry.example/old:4'},
            },
        )
        self.assertTrue(
            all('candidate_key' not in row for row in result['images'])
        )

    def test_empty_or_ambiguous_branch_coverage_fails(self):
        for images in (set(), {'mysql:8.0', 'mysql:8.4'}):
            with self.subTest(images=images), self.assertRaises(ValueError):
                scanner.create_inventory(
                    self.sources,
                    {
                        'master': {'mysql:8.4'},
                        'release-2.18': images,
                    },
                )

    def test_discovery_reads_exact_branch_commits_and_preserves_rendered_evidence(
        self,
    ):
        manifest = (
            'kind: Pod\nspec:\n  containers:\n'
            '  - image: ghcr.io/kubeflow/kfp-metadata-writer:2.18.0\n'
            '  - image: ghcr.io/chrislusf/seaweedfs:4.34\n'
        )
        with tempfile.TemporaryDirectory() as temporary:
            output = Path(temporary)
            env = {
                'GITHUB_WORKFLOW_SHA': 'c' * 40,
                'GITHUB_RUN_ID': '123',
                'GITHUB_RUN_ATTEMPT': '2',
                'GITHUB_OUTPUT': str(output / 'outputs'),
            }
            with (
                mock.patch.dict(os.environ, env),
                mock.patch.object(
                    scanner,
                    'github_api',
                    side_effect=[{'sha': sha} for sha in self.sources.values()],
                ) as api,
                mock.patch.object(scanner, 'download_manifests') as download,
                mock.patch.object(
                    scanner, 'render_overlay', return_value=manifest
                ) as render,
            ):
                scanner.discover(
                    'kubeflow/pipelines', Path('/tools/kustomize'), output
                )
            self.assertEqual(
                api.call_args_list,
                [
                    mock.call('repos/kubeflow/pipelines/commits/' + branch)
                    for branch in self.sources
                ],
            )
            self.assertEqual(
                download.call_args_list,
                [
                    mock.call('kubeflow/pipelines', sha, output / branch)
                    for branch, sha in self.sources.items()
                ],
            )
            self.assertEqual(render.call_count, 4)
            inventory = json.loads((output / 'inventory.json').read_text())
            self.assertEqual(inventory['sources'], self.sources)
            self.assertEqual(
                inventory['workflow_sha'], env['GITHUB_WORKFLOW_SHA']
            )
            self.assertEqual(
                [row['image'] for row in inventory['scans']],
                ['ghcr.io/chrislusf/seaweedfs:4.34'],
            )
            self.assertEqual(len(list(output.glob('*.yaml'))), 4)
            self.assertIn('matrix=', (output / 'outputs').read_text())


class ThirdPartyScanTest(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.output = Path(self.temporary.name) / 'scan'
        self.image = 'registry.example/tool:1'
        self.metadata = {
            'Os': 'linux',
            'Architecture': 'amd64',
            'Id': 'sha256:' + 'a' * 64,
            'RepoDigests': ['registry.example/tool@sha256:' + 'b' * 64],
        }

    def invoke(self, metadata):
        with (
            mock.patch.object(scanner.subprocess, 'run') as pull,
            mock.patch.object(
                scanner.subprocess,
                'check_output',
                return_value=json.dumps(metadata),
            ),
            mock.patch('kfpr.osv_image_scan.scan_image') as scan_image,
        ):
            scanner.scan(self.image, self.output)
        return pull, scan_image

    def test_scan_uses_immutable_id_and_keeps_subject_not_mutable_tag(self):
        pull, scan_image = self.invoke([self.metadata])
        pull.assert_called_once_with(
            [
                'bash',
                '.github/resources/scripts/docker-pull-with-retry.sh',
                self.image,
            ],
            check=True,
        )
        scan_image.assert_called_once_with(
            self.metadata['Id'],
            'linux/amd64',
            self.output,
            report_format='json',
        )
        subject = json.loads((self.output / 'subject.json').read_text())
        self.assertEqual(subject['image'], self.image)
        self.assertEqual(subject['repo_digests'], self.metadata['RepoDigests'])

    def test_foreign_platform_fails_before_scanning(self):
        self.metadata['Architecture'] = 'arm64'
        with self.assertRaisesRegex(ValueError, 'linux/amd64'):
            self.invoke([self.metadata])
        self.assertFalse((self.output / 'subject.json').exists())

    def test_ambiguous_inspect_fails(self):
        with self.assertRaisesRegex(ValueError, 'one pulled image'):
            self.invoke([self.metadata, self.metadata])

    def test_pull_failure_cannot_produce_success_evidence(self):
        with (
            mock.patch.object(
                scanner.subprocess,
                'run',
                side_effect=subprocess.CalledProcessError(1, 'docker pull'),
            ),
            mock.patch('kfpr.osv_image_scan.scan_image') as scan_image,
            self.assertRaises(subprocess.CalledProcessError),
        ):
            scanner.scan(self.image, self.output)
        scan_image.assert_not_called()
        self.assertFalse((self.output / 'subject.json').exists())


class ThirdPartyWorkflowTest(unittest.TestCase):
    def setUp(self):
        self.path = ROOT / '.github/workflows/third-party-image-scanning.yml'
        self.text = self.path.read_text()
        self.workflow = yaml.safe_load(self.text)

    def test_artifact_only_workflow_has_no_security_write_or_sarif_upload(self):
        self.assertNotIn('security-events', self.text)
        self.assertNotIn('upload-sarif', self.text)
        self.assertEqual(self.workflow['permissions'], {'contents': 'read'})
        for job in self.workflow['jobs'].values():
            self.assertFalse(
                any(
                    value == 'write'
                    for value in job.get('permissions', {}).values()
                )
            )
            for step in job['steps']:
                if step.get('uses', '').startswith('actions/checkout@'):
                    self.assertFalse(step['with']['persist-credentials'])
                    self.assertEqual(
                        step['with']['ref'], '${{ github.workflow_sha }}'
                    )

    def test_raw_failure_evidence_retained_and_report_requires_success(self):
        scan = self.workflow['jobs']['scan']
        self.assertFalse(scan['strategy']['fail-fast'])
        uploads = [
            step
            for step in scan['steps']
            if step.get('uses', '').startswith('actions/upload-artifact@')
        ]
        self.assertEqual(len(uploads), 1)
        self.assertEqual(uploads[0]['if'], 'always()')
        self.assertEqual(uploads[0]['with']['path'], 'evidence')
        self.assertEqual(uploads[0]['with']['if-no-files-found'], 'error')
        report = self.workflow['jobs']['report']
        self.assertEqual(set(report['needs']), {'inventory', 'scan'})
        command = '\n'.join(step.get('run', '') for step in report['steps'])
        self.assertIn('test "$SCAN_RESULT" = success', command)
        self.assertIn('test "$INVENTORY_RESULT" = success', command)
        self.assertIn('third_party_image_report.py', command)

    def test_native_sarif_keeps_scope_and_both_workflows_call_evidence(self):
        master = (ROOT / '.github/workflows/osv-scanner.yml').read_text()
        self.assertIn('--image-scope first-party', master)
        for name in ('osv-scanner.yml', 'release-branch-image-scanning.yml'):
            workflow = yaml.safe_load(
                (ROOT / '.github/workflows' / name).read_text()
            )
            job = workflow['jobs']['third-party-evidence']
            self.assertEqual(
                job['uses'],
                './.github/workflows/third-party-image-scanning.yml',
            )
            self.assertNotIn('security-events', job['permissions'])


if __name__ == '__main__':
    unittest.main()
