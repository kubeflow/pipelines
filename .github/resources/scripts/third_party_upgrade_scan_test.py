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
import hashlib
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
                if target in scan.CERT_MANAGER_TARGETS:
                    continue
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

    def cert_manifest(self, version):
        documents = []
        for component in ('controller', 'cainjector', 'webhook'):
            container = {
                'name':
                    'cert-manager-' + component,
                'image':
                    'quay.io/jetstack/cert-manager-' + component + ':' +
                    version,
            }
            if component == 'controller':
                container['args'] = [
                    '--acme-http01-solver-image=quay.io/jetstack/cert-manager-acmesolver:'
                    + version,
                ]
            documents.append({
                'kind': 'Deployment',
                'metadata': {
                    'namespace': 'cert-manager'
                },
                'spec': {
                    'template': {
                        'spec': {
                            'containers': [container]
                        }
                    }
                },
            })
        return documents

    def test_cert_manager_targets_follow_each_checkout_and_preserve_manifest(
            self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            for version in ('v1.16.2', 'v1.21.3'):
                checkout = root / version
                (checkout / 'backend').mkdir(parents=True)
                (checkout /
                 'backend/Makefile').write_text('CERT_MANAGER_VERSION ?= ' +
                                                version + '\n')
                raw = yaml.safe_dump_all(self.cert_manifest(version)).encode()
                for target in scan.CERT_MANAGER_TARGETS:
                    output = checkout / target
                    output.mkdir()
                    with mock.patch.object(scan.urllib.request,
                                           'urlopen') as download:
                        download.return_value.__enter__.return_value.read.return_value = raw
                        reference, provenance = scan.cert_manager_reference(
                            checkout, target, output)
                        expected_url = (
                            'https://github.com/cert-manager/cert-manager/releases/download/'
                            + version + '/cert-manager.yaml')
                        download.assert_called_once_with(
                            expected_url, timeout=60)
                    self.assertEqual(
                        reference, 'quay.io/jetstack/' + target + ':' + version)
                    self.assertEqual(
                        (output / 'cert-manager.yaml').read_bytes(), raw)
                    self.assertEqual(
                        provenance['upstream_manifest'], {
                            'url': expected_url,
                            'file': 'cert-manager.yaml',
                            'sha256': hashlib.sha256(raw).hexdigest(),
                        })

    def test_cert_manager_rejects_unpinned_or_ambiguous_version_before_download(
            self):
        with tempfile.TemporaryDirectory() as temporary:
            checkout = Path(temporary)
            (checkout / 'backend').mkdir()
            for text in (
                    'CERT_MANAGER_VERSION ?= latest',
                    'CERT_MANAGER_VERSION ?= $(VERSION)',
                    'CERT_MANAGER_VERSION ?= v1.21.3\nCERT_MANAGER_VERSION ?= v1.16.2'
            ):
                (checkout / 'backend/Makefile').write_text(text)
                with mock.patch.object(scan.urllib.request,
                                       'urlopen') as download:
                    with self.assertRaisesRegex(ValueError,
                                                'pinned cert-manager version'):
                        scan.cert_manager_reference(checkout,
                                                    'cert-manager-controller',
                                                    checkout)
                    download.assert_not_called()

    def test_cert_manager_missing_ambiguous_and_invalid_images_fail_closed(
            self):
        with tempfile.TemporaryDirectory() as temporary:
            checkout = Path(temporary)
            (checkout / 'backend').mkdir()
            (checkout /
             'backend/Makefile').write_text('CERT_MANAGER_VERSION ?= v1.21.3\n')
            valid = self.cert_manifest('v1.21.3')
            cases = [
                ('cert-manager-controller', valid[1:]),
                ('cert-manager-controller', valid + [valid[0]]),
            ]
            for args in ([], ['--acme-http01-solver-image='], [
                    '--acme-http01-solver-image=one:1',
                    '--acme-http01-solver-image=two:2'
            ], ['--acme-http01-solver-image', '--another-flag']):
                malformed = copy.deepcopy(valid)
                malformed[0]['spec']['template']['spec']['containers'][0][
                    'args'] = args
                cases.append(('cert-manager-acmesolver', malformed))
            malformed = copy.deepcopy(valid)
            malformed[0]['spec']['template']['spec']['containers'][0][
                'image'] = 'bad image'
            cases.append(('cert-manager-controller', malformed))
            for target, documents in cases:
                raw = yaml.safe_dump_all(documents).encode()
                with self.subTest(
                        target=target, documents=documents), mock.patch.object(
                            scan.urllib.request, 'urlopen') as download:
                    download.return_value.__enter__.return_value.read.return_value = raw
                    with self.assertRaises(ValueError):
                        scan.cert_manager_reference(checkout, target, checkout)
                    self.assertEqual(
                        (checkout / 'cert-manager.yaml').read_bytes(), raw)

    @unittest.skipUnless(
        importlib.util.find_spec('kfpr'),
        'Requires release tooling on PYTHONPATH',
    )
    def test_cert_manager_capture_binds_manifest_provenance_to_image_subject(
            self):
        image_id = 'sha256:' + 'a' * 64
        provenance = {
            'upstream_manifest': {
                'file': 'cert-manager.yaml',
                'sha256': 'b' * 64,
                'url': 'https://example/manifest'
            }
        }
        metadata = [{
            'Id': image_id,
            'Os': 'linux',
            'Architecture': 'amd64',
            'RepoDigests': ['registry/image@sha256:' + 'c' * 64]
        }]

        def scanner(actual, platform, output, report_format):
            self.assertEqual(actual, image_id)
            (output / 'scan.json').write_text(
                json.dumps({'report': report([])}))

        with (tempfile.TemporaryDirectory() as temporary,
              mock.patch.object(
                  scan,
                  'cert_manager_reference',
                  return_value=('registry/image:1', provenance)) as reference,
              mock.patch.object(scan, 'image_reference') as ordinary_reference,
              mock.patch.object(
                  scan.subprocess,
                  'check_output',
                  side_effect=['d' * 40, json.dumps(metadata)]),
              mock.patch.object(scan.subprocess, 'run'),
              mock.patch('kfpr.osv_image_scan.scan_image', side_effect=scanner),
              mock.patch('kfpr.osv_image_scan.validate_report')):
            checkout = Path(temporary)
            output = checkout / 'out'
            subject, _ = scan.capture(checkout, 'cert-manager-controller',
                                      output)
            reference.assert_called_once_with(checkout,
                                              'cert-manager-controller', output)
            ordinary_reference.assert_not_called()
            self.assertEqual(subject['source_sha'], 'd' * 40)
            self.assertEqual(subject['manifest'], 'backend/Makefile')
            self.assertEqual(subject['upstream_manifest'],
                             provenance['upstream_manifest'])
            self.assertEqual(subject['image_id'], image_id)
            self.assertEqual(
                json.loads((output / 'subject.json').read_text()), subject)

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
