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

    def test_uploads_keep_branch_commits_and_share_exact_ref_evidence(self):
        image = 'mysql:8.4@sha256:' + 'c' * 64
        result = scanner.create_inventory(
            self.sources, {branch: {image} for branch in scanner.BRANCHES}
        )
        self.assertEqual(len(result['scans']), 1)
        self.assertEqual(len(result['uploads']), 2)
        category = next(iter(scanner.create_image_map({image})))
        for row in result['uploads']:
            self.assertEqual(row['image'], image)
            self.assertEqual(row['key'], result['scans'][0]['key'])
            self.assertEqual(row['sha'], self.sources[row['branch']])
            expected = (
                f'kubeflow-pipelines-osv-image-{category}'
                if row['branch'] == 'master'
                else f'release-2.18-third-party-image-{category}-amd64'
            )
            self.assertEqual(row['category'], expected)
        self.assertEqual(
            {row['branch'] for row in result['uploads']}, set(scanner.BRANCHES)
        )

    def test_upload_categories_survive_tag_digest_and_commit_changes(self):
        previous = None
        keys = set()
        for version in ('4', '5', '5@sha256:' + 'd' * 64):
            image = 'registry.example:5000/object:' + version
            result = scanner.create_inventory(
                {branch: version for branch in scanner.BRANCHES},
                {branch: {image} for branch in scanner.BRANCHES},
            )
            categories = {
                row['branch']: row['category'] for row in result['uploads']
            }
            if previous is not None:
                self.assertEqual(categories, previous)
            previous = categories
            keys.add(result['scans'][0]['key'])
        self.assertEqual(len(keys), 3)

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

    def test_logical_targets_allow_multiple_versions_and_share_scans(self):
        targets = [
            {
                'target': 'mysql',
                'image': 'mysql:8.4',
                'scope': 'standard',
                'profiles': ['normal'],
            },
            {
                'target': 'legacy:mysql',
                'image': 'mysql:8.0',
                'scope': 'legacy',
                'profiles': ['legacy'],
            },
            {
                'target': 'runtime:cache:mysql',
                'image': 'mysql:8.4',
                'scope': 'runtime',
                'profiles': [],
            },
        ]
        result = scanner.create_inventory(
            self.sources, {branch: targets for branch in scanner.BRANCHES}
        )
        self.assertEqual(len(result['scans']), 2)
        self.assertEqual(len(result['uploads']), 6)
        self.assertEqual(len({row['category'] for row in result['uploads']}), 6)
        prior = {row['category'] for row in result['uploads']}
        for target in targets:
            target['image'] += '-patched'
        changed = scanner.create_inventory(
            self.sources, {branch: targets for branch in scanner.BRANCHES}
        )
        self.assertEqual(prior, {row['category'] for row in changed['uploads']})

    def test_scoped_duplicate_target_fails_before_uploads(self):
        targets = [
            {'target': 'mysql', 'image': 'mysql:8.4'},
            {'target': 'mysql', 'image': 'mysql:8.0'},
        ]
        with self.assertRaisesRegex(ValueError, 'Duplicate stable scan target'):
            scanner.create_inventory(
                self.sources, {branch: targets for branch in scanner.BRANCHES}
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
                    scanner,
                    'capture',
                    return_value={
                        'targets': [
                            {
                                'image': 'ghcr.io/chrislusf/seaweedfs:4.34',
                                'target': 'ghcr.io/chrislusf/seaweedfs',
                                'scope': 'standard',
                                'profiles': ['platform-agnostic'],
                            }
                        ]
                    },
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
                    mock.call(
                        'kubeflow/pipelines',
                        sha,
                        output / branch,
                        extra_paths=scanner.SOURCE_PATHS,
                    )
                    for branch, sha in self.sources.items()
                ],
            )
            self.assertEqual(render.call_count, 2)
            inventory = json.loads((output / 'inventory.json').read_text())
            self.assertEqual(inventory['sources'], self.sources)
            self.assertEqual(
                inventory['workflow_sha'], env['GITHUB_WORKFLOW_SHA']
            )
            self.assertEqual(
                [row['image'] for row in inventory['scans']],
                ['ghcr.io/chrislusf/seaweedfs:4.34'],
            )
            outputs = dict(
                line.split('=', 1)
                for line in (output / 'outputs').read_text().splitlines()
            )
            self.assertEqual(
                json.loads(outputs['matrix']), {'include': inventory['scans']}
            )
            self.assertEqual(
                json.loads(outputs['uploads']),
                {'include': inventory['uploads']},
            )


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
        self.assertEqual(
            scan_image.call_args_list,
            [
                mock.call(
                    self.metadata['Id'],
                    'linux/amd64',
                    self.output,
                    report_format='json',
                ),
                mock.call(
                    self.metadata['Id'],
                    'linux/amd64',
                    self.output / 'github',
                    report_format='sarif',
                ),
            ],
        )
        subject = json.loads((self.output / 'subject.json').read_text())
        self.assertEqual(subject['image'], self.image)
        self.assertEqual(subject['repo_digests'], self.metadata['RepoDigests'])

    def test_sarif_scan_preserves_raw_json_and_its_logs(self):
        def write_report(image_id, platform, output, report_format):
            self.assertEqual(image_id, self.metadata['Id'])
            self.assertEqual(platform, 'linux/amd64')
            self.assertTrue(output.is_dir())
            (output / 'scan.json').write_text(report_format)
            (output / 'scan.stderr.log').write_text(report_format + ' log')
            report = (
                'osv-report.json' if report_format == 'json' else 'scan.sarif'
            )
            (output / report).write_text(report_format + ' raw')

        with (
            mock.patch.object(scanner.subprocess, 'run'),
            mock.patch.object(
                scanner.subprocess,
                'check_output',
                return_value=json.dumps([self.metadata]),
            ),
            mock.patch(
                'kfpr.osv_image_scan.scan_image', side_effect=write_report
            ),
        ):
            scanner.scan(self.image, self.output)
        for directory, format_name, report in (
            (self.output, 'json', 'osv-report.json'),
            (self.output / 'github', 'sarif', 'scan.sarif'),
        ):
            self.assertEqual((directory / 'scan.json').read_text(), format_name)
            self.assertEqual(
                (directory / 'scan.stderr.log').read_text(),
                format_name + ' log',
            )
            self.assertEqual(
                (directory / report).read_text(), format_name + ' raw'
            )

    def test_unknown_scan_preserves_diagnostics_and_fails(self):
        with (
            mock.patch.object(scanner.subprocess, 'run'),
            mock.patch.object(
                scanner.subprocess,
                'check_output',
                return_value=json.dumps([self.metadata]),
            ),
            mock.patch(
                'kfpr.osv_image_scan.scan_image',
                side_effect=ValueError('No package metadata'),
            ),
            self.assertRaisesRegex(ValueError, 'No package metadata'),
        ):
            scanner.scan(self.image, self.output)
        coverage = json.loads((self.output / 'coverage.json').read_text())
        self.assertEqual(coverage['status'], 'unknown')
        self.assertEqual(coverage['image_id'], self.metadata['Id'])
        self.assertFalse((self.output / 'github').exists())

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

    def test_only_production_upload_has_security_write(self):
        self.assertEqual(self.workflow['permissions'], {'contents': 'read'})
        for name, job in self.workflow['jobs'].items():
            writes = {
                permission
                for permission, value in job.get('permissions', {}).items()
                if value == 'write'
            }
            self.assertEqual(
                writes, {'security-events'} if name == 'upload' else set()
            )
            for step in job['steps']:
                if step.get('uses', '').startswith('actions/checkout@'):
                    self.assertFalse(step['with']['persist-credentials'])
                    if step['with'].get('path') == 'source':
                        self.assertEqual(name, 'upload')
                        self.assertEqual(
                            step['with']['ref'], '${{ matrix.sha }}'
                        )
                    else:
                        self.assertEqual(
                            step['with']['ref'], '${{ github.workflow_sha }}'
                        )
        upload = self.workflow['jobs']['upload']
        self.assertEqual(upload['if'], "github.ref == 'refs/heads/master'")
        self.assertEqual(set(upload['needs']), {'inventory', 'scan', 'report'})
        self.assertEqual(
            upload['strategy']['matrix'],
            '${{ fromJSON(needs.inventory.outputs.uploads) }}',
        )
        publish = [
            step
            for step in upload['steps']
            if step.get('uses', '').startswith(
                'github/codeql-action/upload-sarif@'
            )
        ]
        self.assertEqual(len(publish), 1)
        self.assertEqual(
            publish[0]['with'],
            {
                'sarif_file': 'findings.sarif',
                'category': '${{ matrix.category }}',
                'sha': '${{ matrix.sha }}',
                'ref': 'refs/heads/${{ matrix.branch }}',
                'checkout_path': 'source',
            },
        )

    def test_upload_validates_raw_evidence_and_only_removes_exact_duplicates(
        self,
    ):
        from kfpr.osv_image_scan import SCANNER

        steps = self.workflow['jobs']['upload']['steps']
        validation = next(
            step for step in steps if 'validate_report' in step.get('run', '')
        )
        self.assertEqual(validation['env']['IMAGE'], '${{ matrix.image }}')
        self.assertEqual(validation['env']['IMAGE_KEY'], '${{ matrix.key }}')
        download = next(
            step
            for step in steps
            if 'download-artifact-with-retry' in step.get('uses', '')
        )
        self.assertEqual(download['with']['path'], 'evidence')
        self.assertIn(
            '${{ matrix.key }}/github/scan.sarif',
            download['with']['required-files'],
        )
        result = {
            'ruleId': 'CVE-2026-1234',
            'message': {'text': 'Vulnerable package'},
        }
        distinct = {**result, 'message': {'text': 'Another vulnerable package'}}
        sarif = {
            'version': '2.1.0',
            'runs': [
                {
                    'tool': {
                        'driver': {
                            **SCANNER,
                            'rules': [{'id': result['ruleId']}],
                        }
                    },
                    'results': [result, result, distinct],
                }
            ],
        }
        image, image_id = 'example/image:1', 'sha256:' + 'a' * 64
        envelope = {
            'schema': 1,
            'scanner': SCANNER,
            'image_id': image_id,
            'archive_sha256': 'b' * 64,
            'exit_code': 1,
            'report_format': 'sarif',
            'sarif': sarif,
        }
        for mismatch in (None, 'image', 'image_id', 'sarif'):
            with (
                self.subTest(mismatch=mismatch),
                tempfile.TemporaryDirectory() as temporary,
            ):
                directory = Path(temporary)
                root = directory / 'evidence/key'
                (root / 'github').mkdir(parents=True)
                subject = {'image': image, 'image_id': image_id}
                if mismatch in ('image', 'image_id'):
                    subject[mismatch] = 'wrong'
                (root / 'subject.json').write_text(json.dumps(subject))
                (root / 'github/scan.json').write_text(json.dumps(envelope))
                raw = json.dumps({} if mismatch == 'sarif' else sarif)
                (root / 'github/scan.sarif').write_text(raw)
                scripts = directory / '.github/resources/scripts'
                scripts.parent.mkdir(parents=True)
                scripts.symlink_to(
                    ROOT / '.github/resources/scripts', target_is_directory=True
                )
                env = {
                    **os.environ,
                    'PATH': str(Path(sys.executable).parent)
                    + os.pathsep
                    + os.environ['PATH'],
                    'PYTHONPATH': str(ROOT / 'release'),
                    'IMAGE': image,
                    'IMAGE_KEY': 'key',
                }
                process = subprocess.run(
                    ['bash', '-e', '-o', 'pipefail', '-c', validation['run']],
                    cwd=directory,
                    env=env,
                    capture_output=True,
                    text=True,
                )
                self.assertEqual((root / 'github/scan.sarif').read_text(), raw)
                if mismatch:
                    self.assertNotEqual(process.returncode, 0)
                    self.assertFalse((directory / 'findings.sarif').exists())
                else:
                    self.assertEqual(process.returncode, 0, process.stderr)
                    uploaded = json.loads(
                        (directory / 'findings.sarif').read_text()
                    )
                    self.assertEqual(
                        uploaded['runs'][0]['results'], [result, distinct]
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
            self.assertEqual(job['permissions']['security-events'], 'write')


if __name__ == '__main__':
    unittest.main()
