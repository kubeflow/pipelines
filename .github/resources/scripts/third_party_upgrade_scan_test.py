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
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest
from unittest import mock

import third_party_upgrade_scan as scan
import yaml


def report(ids):
    return {
        'results': [{
            'source': {
                'path': '/packages',
                'type': 'os'
            },
            'packages': [{
                'package': {
                    'name': 'test',
                    'version': '1',
                    'ecosystem': 'Go',
                },
                'vulnerabilities': [{
                    'id':
                        identifier,
                    'aliases':
                        aliases,
                    'affected': [{
                        'package': {
                            'name': 'test',
                            'ecosystem': 'Go',
                        },
                        'versions': ['1'],
                    }],
                } for identifier, aliases in ids],
            }],
        }]
    }


class ImageUpgradeScanTest(unittest.TestCase):

    def test_manifest_selects_named_container_and_executor(self):
        with tempfile.TemporaryDirectory() as temporary:
            checkout = Path(temporary)
            for target, (relative, name) in scan.TARGETS.items():
                path = checkout / relative
                path.parent.mkdir(parents=True, exist_ok=True)
                manifest = {
                    'spec': {
                        'template': {
                            'spec': {
                                'containers': [
                                    {
                                        'name': 'unrelated',
                                        'image': 'wrong:1'
                                    },
                                    {
                                        'name':
                                            name,
                                        'image':
                                            'right:2',
                                        'args': [
                                            '--executor-image',
                                            'executor:2',
                                        ],
                                    },
                                ]
                            }
                        }
                    }
                }
                path.write_text(yaml.safe_dump(manifest))
                self.assertEqual(
                    scan.image_reference(checkout, target),
                    'executor:2' if target == 'argo-executor' else 'right:2',
                )
                container = manifest['spec']['template']['spec']['containers'][
                    1]
                if target == 'argo-executor':
                    container['env'] = [{
                        'name': 'ARGO_EXECUTOR_IMAGE',
                        'value': 'conflicting:3',
                    }]
                else:
                    manifest['spec']['template']['spec']['containers'].append(
                        container)
                path.write_text(yaml.safe_dump(manifest))
                with self.assertRaises(ValueError):
                    scan.image_reference(checkout, target)

    @unittest.skipUnless(
        importlib.util.find_spec('kfpr'),
        'Requires release tooling on PYTHONPATH',
    )
    def test_transitive_aliases_deduplicate_across_both_reports(self):
        old = report([('GO-1', ['CVE-1']), ('GO-1', ['CVE-1']), ('OLD', [])])
        new = report([('GHSA-1', ['CVE-1', 'ALIAS']), ('NEW', [])])
        preserved = copy.deepcopy((old, new))
        comparison = scan.compare_reports(old, new)
        self.assertEqual(comparison['baseline_occurrences'], 3)
        self.assertEqual(comparison['baseline_advisory_groups'], 2)
        self.assertEqual(comparison['candidate_advisory_groups'], 2)
        self.assertEqual(comparison['absent_from_candidate'], [['OLD']])
        self.assertEqual(comparison['new_in_candidate'], [['NEW']])
        self.assertEqual(
            comparison['present_in_both'],
            [['ALIAS', 'CVE-1', 'GHSA-1', 'GO-1']],
        )
        self.assertEqual((old, new), preserved)

    @unittest.skipUnless(
        importlib.util.find_spec('kfpr'),
        'Requires release tooling on PYTHONPATH',
    )
    def test_invalid_report_does_not_look_like_no_findings(self):
        for invalid in ({}, {'results': []}, {'results': [{'source': {}}]}):
            with self.subTest(invalid=invalid), self.assertRaises(ValueError):
                scan.compare_reports(report([]), invalid)

    @unittest.skipUnless(
        importlib.util.find_spec('kfpr'),
        'Requires release tooling on PYTHONPATH',
    )
    def test_scan_binds_both_formats_to_same_immutable_id(self):
        image_id = 'sha256:' + 'a' * 64
        metadata = [{
            'Id': image_id,
            'Os': 'linux',
            'Architecture': 'amd64',
            'RepoDigests': ['registry/image@sha256:' + 'b' * 64],
        }]
        envelopes = []

        def scanner(actual, platform, output, report_format):
            self.assertEqual(actual, image_id)
            self.assertEqual(platform, 'linux/amd64')
            envelope = {'format': report_format, 'report': report([])}
            envelopes.append(envelope)
            (output / 'scan.json').write_text(json.dumps(envelope))

        with (
                tempfile.TemporaryDirectory() as temporary,
                mock.patch.object(
                    scan, 'image_reference', return_value='registry/image:1'),
                mock.patch.object(
                    scan.subprocess,
                    'check_output',
                    side_effect=['c' * 40, json.dumps(metadata)],
                ),
                mock.patch.object(scan.subprocess, 'run') as pull,
                mock.patch(
                    'kfpr.osv_image_scan.scan_image', side_effect=scanner),
                mock.patch('kfpr.osv_image_scan.validate_report') as validate,
        ):
            subject, raw = scan.capture(
                Path(temporary), 'seaweedfs',
                Path(temporary) / 'out')
            self.assertEqual(subject['image_id'], image_id)
            self.assertEqual(raw, report([]))
            pull.assert_called_once_with(
                [
                    'docker',
                    'pull',
                    '--platform',
                    'linux/amd64',
                    'registry/image:1',
                ],
                check=True,
            )
            self.assertEqual(
                validate.call_args_list,
                [
                    mock.call(envelopes[0], image_id, 'json'),
                    mock.call(envelopes[1], image_id, 'sarif'),
                ],
            )

    @unittest.skipUnless(
        importlib.util.find_spec('kfpr'),
        'Requires release tooling on PYTHONPATH',
    )
    def test_scan_validation_failure_stops_comparison(self):
        image_id = 'sha256:' + 'a' * 64
        metadata = [{
            'Id': image_id,
            'Os': 'linux',
            'Architecture': 'amd64',
            'RepoDigests': ['registry/image@sha256:' + 'b' * 64],
        }]
        with (
                tempfile.TemporaryDirectory() as temporary,
                mock.patch.object(
                    scan, 'image_reference', return_value='registry/image:1'),
                mock.patch.object(
                    scan.subprocess,
                    'check_output',
                    side_effect=['c' * 40, json.dumps(metadata)],
                ),
                mock.patch.object(scan.subprocess, 'run'),
                mock.patch(
                    'kfpr.osv_image_scan.scan_image',
                    side_effect=ValueError('incomplete extraction'),
                ),
        ):
            with self.assertRaisesRegex(ValueError, 'incomplete extraction'):
                scan.capture(
                    Path(temporary), 'seaweedfs',
                    Path(temporary) / 'out')
            self.assertTrue((Path(temporary) / 'out/subject.json').is_file())
            self.assertFalse((Path(temporary) / 'out/sarif').exists())


if __name__ == '__main__':
    unittest.main()
