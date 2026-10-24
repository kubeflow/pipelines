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
"""Keep Debian triage additive, complete, and separate from acceptance."""

import copy
import json
from pathlib import Path
import tempfile
import unittest

from triage_debian import triage
from triage_debian import write_view


def entry(binary='libc6', version='2.41-12+deb13u4', ecosystem='Debian:13'):
    return {
        'package': {
            'name': 'glibc',
            'os_package_name': binary,
            'version': version,
            'ecosystem': ecosystem
        },
        'vulnerabilities': [{
            'id':
                'DEBIAN-CVE-2026-1234',
            'aliases': ['CVE-2026-1234'],
            'affected': [{
                'package': {
                    'name': 'glibc',
                    'ecosystem': ecosystem
                },
                'ranges': [{
                    'events': [{
                        'fixed': '2.43-7'
                    }]
                }]
            }]
        }]
    }


def scan(*entries):
    return {
        'schema': 1,
        'image_id': 'sha256:test',
        'source_sha': 'source-test',
        'report': {
            'results': [{
                'source': {
                    'path': '/var/lib/dpkg/status',
                    'type': 'os'
                },
                'packages': list(entries)
            }]
        }
    }


def tracker(**release):
    return {'glibc': {'CVE-2026-1234': {'releases': {'trixie': release}}}}


class DebianTriageTest(unittest.TestCase):

    def test_duplicate_binaries_retain_every_occurrence_and_old_fixes(self):
        result = triage(
            scan(entry(), entry('libc-bin')),
            tracker(status='open', urgency='unimportant'))
        self.assertEqual(result['counts']['deduplicated_source_findings'], 1)
        row = result['findings'][0]
        self.assertEqual(row['raw_occurrence_count'], 2)
        self.assertEqual({x['binary_package'] for x in row['raw_occurrences']},
                         {'libc6', 'libc-bin'})
        self.assertEqual(row['scan_fixed_versions'], ['2.43-7'])
        self.assertEqual(row['tracker']['interpretation'], 'open')
        self.assertEqual(row['tracker']['urgency'], 'unimportant')
        self.assertEqual(row['scan_advisory_ids'],
                         ['CVE-2026-1234', 'DEBIAN-CVE-2026-1234'])

    def test_versions_and_ecosystems_remain_distinct(self):
        result = triage(
            scan(entry(), entry(version='2.42'), entry(ecosystem='Debian:12')),
            tracker(status='resolved', fixed_version='0'))
        self.assertEqual(len(result['findings']), 3)
        self.assertEqual(
            sum(row['tracker']['applies_to_scanned_ecosystem']
                for row in result['findings']), 2)

    def test_other_debian_release_never_inherits_trixie_verdict(self):
        row = triage(
            scan(entry(ecosystem='Debian:12')),
            tracker(status='resolved', fixed_version='0'))['findings'][0]
        self.assertEqual(row['tracker']['lookup'], 'unsupported_ecosystem')
        self.assertEqual(row['tracker']['interpretation'], 'unknown')
        self.assertIsNone(row['tracker']['fixed_version'])
        self.assertEqual(row['reported_version'], '2.41-12+deb13u4')

    def test_empty_or_malformed_reports_fail_explicitly(self):
        examples = [
            None, {}, {
                'report': {}
            }, {
                'report': {
                    'results': []
                }
            }, {
                'report': {
                    'results': {}
                }
            }
        ]
        for mutate in (lambda x: x['report']['results'][0].update(packages=[]),
                       lambda x: x['report']['results'][0].update(source=None),
                       lambda x: x['report']['results'][0]['packages'][
                           0].update(vulnerabilities={}),
                       lambda x: x['report']['results'][0]['packages'][0][
                           'package'].update(version=None)):
            value = scan(entry())
            mutate(value)
            examples.append(value)
        for value in examples:
            with self.subTest(value=value), self.assertRaises(ValueError):
                triage(value, {})
        with self.assertRaises(ValueError):
            triage(scan(entry()), [])
        self.assertEqual(
            triage(scan(entry(), entry('libc-bin')),
                   {})['counts']['distinct_advisory_ids'], 1)

    def test_missing_release_unknown_and_zero_are_explicit(self):
        for data, expected in (({}, 'missing_advisory'), ({
                'glibc': {
                    'CVE-2026-1234': {
                        'releases': {}
                    }
                }
        }, 'missing_release')):
            row = triage(scan(entry()), data)['findings'][0]
            self.assertEqual(row['tracker']['lookup'], expected)
            self.assertEqual(row['tracker']['interpretation'], 'unknown')
        for data, expected in ((tracker(status='resolved',
                                        fixed_version='0'), 'not_affected'),
                               (tracker(
                                   status='undetermined',
                                   urgency='unimportant',
                                   nodsa='ignored',
                                   nodsa_reason='minor issue'), 'undetermined'),
                               (tracker(urgency='unimportant'), 'unknown')):
            self.assertEqual(
                triage(scan(entry()),
                       data)['findings'][0]['tracker']['interpretation'],
                expected)

    def test_sidecar_preserves_inputs_and_is_deterministic(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            source, database, output = root / 'scan.json', root / 'tracker.json', root / 'view.json'
            source.write_text(json.dumps(scan(entry())))
            database.write_text(json.dumps(tracker(status='open')))
            before = source.read_bytes(), database.read_bytes()
            result = write_view(source, database, output)
            first = output.read_bytes()
            write_view(source, database, output)
            self.assertEqual(output.read_bytes(), first)
            self.assertEqual((source.read_bytes(), database.read_bytes()),
                             before)
            self.assertEqual(result['scan_provenance']['image_id'],
                             'sha256:test')
            for path in (source, database):
                with self.assertRaises(ValueError):
                    write_view(source, database, path)
            link = root / 'linked.json'
            link.symlink_to(source)
            with self.assertRaises(ValueError):
                write_view(source, database, link)

    def test_raw_objects_are_unchanged(self):
        raw, database = scan(entry()), tracker(status='open')
        before = copy.deepcopy((raw, database))
        triage(raw, database)
        self.assertEqual((raw, database), before)


if __name__ == '__main__':
    unittest.main()
