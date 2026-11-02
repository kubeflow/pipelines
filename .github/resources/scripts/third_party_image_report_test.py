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

import copy
import json
import tempfile
import unittest
from pathlib import Path

from third_party_image_report import aggregate, markdown, repository


class ImageReportTest(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.evidence = Path(self.temporary.name)
        self.key = 'a' * 16
        self.candidate = 'b' * 16
        self.document = {
            'schema': 1,
            'sources': {'master': 'a' * 40, 'release-2.18': 'b' * 40},
            'images': [
                {
                    'key': self.key,
                    'image': 'docker.io/example/tool:1',
                    'branches': ['release-2.18'],
                }
            ],
            'scans': [{'key': self.key, 'image': 'docker.io/example/tool:1'}],
        }
        self.write_scan(
            self.key,
            self.document['images'][0]['image'],
            [('GHSA-one', ['CVE-2026-1'])],
        )

    def test_default_branch_marker_must_match_immutable_sources(self):
        for branch in ('main', 'feature'):
            with self.subTest(branch=branch):
                self.document['default_branch'] = branch
                with self.assertRaisesRegex(ValueError, 'default branch'):
                    aggregate(self.document, self.evidence)

    def test_unknown_lane_retains_other_counts_and_blocks_coverage(self):
        second = 'c' * 16
        image = 'example/unknown:latest'
        self.document['scans'].append({'key': second, 'image': image})
        self.document['images'].append(
            {'key': second, 'image': image, 'branches': ['release-2.18']}
        )
        self.write_scan(second, image, [])
        lane = self.evidence / second
        (lane / 'scan.json').unlink()
        (lane / 'osv-report.json').unlink()
        (lane / 'coverage.json').write_text(
            json.dumps(
                {
                    'status': 'unknown',
                    'image_id': 'sha256:' + second * 4,
                    'reason': 'No package metadata',
                }
            )
        )
        summary = aggregate(self.document, self.evidence)
        self.assertFalse(summary['coverage_complete'])
        self.assertEqual(summary['scans'][self.key]['raw_finding_count'], 1)
        self.assertIsNone(summary['scans'][second]['raw_finding_count'])
        self.assertIn('Unknown', markdown(summary))
        self.assertIn('uploads blocked', markdown(summary))

    def write_scan(self, key, image, advisories, duplicate=False):
        lane = self.evidence / key
        lane.mkdir(exist_ok=True)
        vulnerability = lambda identifier, aliases: {
            'id': identifier,
            'aliases': aliases,
            'affected': [
                {
                    'package': {'name': 'curl', 'ecosystem': 'Alpine:v3.24'},
                    'ranges': [
                        {
                            'type': 'ECOSYSTEM',
                            'events': [
                                {'introduced': '0'},
                                {'fixed': '9.0-r0'},
                            ],
                        }
                    ],
                }
            ],
        }
        package = {
            'package': {
                'name': 'curl',
                'ecosystem': 'Alpine:v3.24',
                'version': '8.0-r0',
            },
            'vulnerabilities': [
                vulnerability(*advisory) for advisory in advisories
            ],
        }
        raw = {
            'results': [
                {
                    'source': {'path': '/lib/apk/db/installed', 'type': 'os'},
                    'packages': [package] * (2 if duplicate else 1),
                }
            ]
        }
        image_id = 'sha256:' + key * 4
        envelope = {
            'schema': 1,
            'scanner': {'name': 'osv-scanner', 'version': '2.5.0'},
            'image_id': image_id,
            'archive_sha256': 'f' * 64,
            'exit_code': int(bool(advisories)),
            'report_format': 'json',
            'report': raw,
        }
        for name, value in [
            (
                'subject.json',
                {
                    'image': image,
                    'image_id': image_id,
                    'platform': 'linux/amd64',
                },
            ),
            ('scan.json', envelope),
            ('osv-report.json', raw),
        ]:
            (lane / name).write_text(json.dumps(value))

    def add_candidate(self, advisories):
        image = 'docker.io/example/tool:2'
        self.document['images'][0]['candidate_key'] = self.candidate
        self.document['scans'].append({'key': self.candidate, 'image': image})
        self.document['images'].append(
            {'key': self.candidate, 'image': image, 'branches': ['master']}
        )
        self.write_scan(self.candidate, image, advisories)

    def test_no_candidate_is_not_an_available_image_fix(self):
        report = aggregate(self.document, self.evidence)
        row = report['images'][0]
        self.assertEqual(row['package_fix_count'], 1)
        self.assertNotIn('candidate_comparison', row)
        self.assertIn('no candidate image verified', row['action'])
        self.assertIn('Package fix versions do not establish', markdown(report))
        self.assertEqual(len(row['evidence_sha256']['scan.json']), 64)

    def test_duplicate_packages_and_alias_id_changes_are_not_fixes(self):
        self.write_scan(
            self.key,
            self.document['images'][0]['image'],
            [('GHSA-one', ['CVE-2026-1'])],
            duplicate=True,
        )
        self.add_candidate([('ALPINE-one', ['CVE-2026-1'])])
        report = aggregate(self.document, self.evidence)
        row = report['images'][0]
        self.assertEqual(row['raw_finding_count'], 2)
        self.assertEqual(row['unique_advisory_count'], 1)
        self.assertEqual(row['candidate_comparison']['resolved'], [])
        self.assertEqual(row['candidate_comparison']['added'], [])
        self.assertEqual(
            row['candidate_comparison']['remaining'], ['CVE-2026-1']
        )

    def test_candidate_reports_added_and_resolved_advisories(self):
        self.add_candidate([('CVE-2026-2', [])])
        comparison = aggregate(self.document, self.evidence)['images'][0][
            'candidate_comparison'
        ]
        self.assertEqual(comparison['resolved'], ['CVE-2026-1'])
        self.assertEqual(comparison['added'], ['CVE-2026-2'])
        self.assertIn('Review required', comparison['compatibility'])

    def test_empty_findings_with_valid_package_inventory(self):
        self.write_scan(self.key, self.document['images'][0]['image'], [])
        row = aggregate(self.document, self.evidence)['images'][0]
        self.assertEqual(row['raw_finding_count'], 0)
        self.assertEqual(len(row['packages']), 1)

    def test_missing_and_extra_lanes_fail(self):
        self.add_candidate([])
        (self.evidence / self.candidate / 'osv-report.json').unlink()
        with self.assertRaises(ValueError):
            aggregate(self.document, self.evidence)
        self.write_scan(self.candidate, self.document['scans'][1]['image'], [])
        (self.evidence / 'unexpected').mkdir()
        with self.assertRaisesRegex(ValueError, 'lanes'):
            aggregate(self.document, self.evidence)

    def test_corrupt_subject_envelope_raw_and_inventory_fail(self):
        for name, change in [
            (
                'subject.json',
                lambda value: value.update(image='other/image:1'),
            ),
            (
                'subject.json',
                lambda value: value.update(image_id='sha256:wrong'),
            ),
            (
                'subject.json',
                lambda value: value.update(platform='linux/arm64'),
            ),
            (
                'scan.json',
                lambda value: value.update(image_id='sha256:' + 'f' * 64),
            ),
            ('scan.json', lambda value: value.update(exit_code=0)),
            ('scan.json', lambda value: value.update(exit_code=2)),
            ('osv-report.json', lambda value: value.update(results=[])),
        ]:
            with self.subTest(file=name, change=change):
                path = self.evidence / self.key / name
                original = path.read_text()
                value = json.loads(original)
                change(value)
                path.write_text(json.dumps(value))
                with self.assertRaises(ValueError):
                    aggregate(self.document, self.evidence)
                path.write_text(original)
        document = copy.deepcopy(self.document)
        document['scans'].append(copy.deepcopy(document['scans'][0]))
        with self.assertRaises(ValueError):
            aggregate(document, self.evidence)

    def test_incomplete_extraction_cannot_be_empty_success(self):
        for name in ('scan.json', 'osv-report.json'):
            path = self.evidence / self.key / name
            value = json.loads(path.read_text())
            if name == 'scan.json':
                value['report']['results'] = []
                value['exit_code'] = 0
            else:
                value['results'] = []
            path.write_text(json.dumps(value))
        with self.assertRaisesRegex(ValueError, 'no dependency targets'):
            aggregate(self.document, self.evidence)

    def test_candidate_other_repository_rejected(self):
        self.add_candidate([])
        self.document['scans'][1]['image'] = 'docker.io/example/other:2'
        with self.assertRaisesRegex(ValueError, 'same repository'):
            aggregate(self.document, self.evidence)

    def test_candidate_requires_master_provenance(self):
        self.add_candidate([])
        self.document['images'][1]['branches'] = ['release-2.18']
        with self.assertRaisesRegex(ValueError, 'deployed default-branch'):
            aggregate(self.document, self.evidence)

    def test_support_channel_evidence_is_visible_in_summary(self):
        report = aggregate(self.document, self.evidence)
        row = report['images'][0]
        row['support_channel_review_count'] = 3
        self.assertIn(
            '3 findings need support-channel review', markdown(report)
        )
        self.assertIn('not a standard-channel fix', markdown(report))

    def test_registry_port_is_not_a_tag(self):
        self.assertEqual(
            repository('registry.example:5000/path/tool:1'),
            'registry.example:5000/path/tool',
        )
        self.assertEqual(
            repository('registry.example:5000/path/tool@sha256:abc'),
            'registry.example:5000/path/tool',
        )


if __name__ == '__main__':
    unittest.main()
