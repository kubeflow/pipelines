"""Test immutable archive binding and fail-closed OSV scan evidence."""

import copy
import hashlib
import io
import json
import subprocess
import tarfile
import tempfile
import unittest
from pathlib import Path
from unittest import mock

from kfpr import osv_image_scan as osv


def sarif(findings=False):
    driver = dict(osv.SCANNER)
    results = []
    if findings:
        driver['rules'] = [{'id': 'CVE-test'}]
        results = [
            {
                'ruleId': 'CVE-test',
                'ruleIndex': 0,
                'message': {'text': 'An unfixed vulnerability'},
                'locations': [
                    {
                        'physicalLocation': {
                            'artifactLocation': {'uri': 'usr/lib/package'}
                        }
                    }
                ],
            }
        ]
    return {
        'version': '2.1.0',
        'runs': [{'tool': {'driver': driver}, 'results': results}],
    }


def envelope(image_id, findings=False):
    return {
        'schema': 1,
        'scanner': dict(osv.SCANNER),
        'image_id': image_id,
        'archive_sha256': 'b' * 64,
        'exit_code': int(findings),
        'sarif': sarif(findings),
    }


def json_report(findings=False):
    item = {
        'package': {'name': 'sample', 'ecosystem': 'PyPI', 'version': '1.0'}
    }
    if findings:
        item['vulnerabilities'] = [
            {
                'id': 'GHSA-test',
                'affected': [
                    {
                        'package': {'name': 'sample', 'ecosystem': 'PyPI'},
                        'ranges': [
                            {
                                'type': 'ECOSYSTEM',
                                'events': [
                                    {'introduced': '0'},
                                    {'fixed': '2.0'},
                                ],
                            }
                        ],
                    }
                ],
            }
        ]
    return {
        'results': [
            {
                'source': {'path': '/usr/lib/sample', 'type': 'os'},
                'packages': [item],
            }
        ]
    }


def json_envelope(image_id, findings=False):
    report = envelope(image_id, findings)
    del report['sarif']
    report.update(report_format='json', report=json_report(findings))
    return report


class OsvImageScanTest(unittest.TestCase):
    def setUp(self):
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        self.root = Path(temporary.name)
        self.config = json.dumps(
            {'os': 'linux', 'architecture': 'amd64'}
        ).encode()
        self.image_id = 'sha256:' + hashlib.sha256(self.config).hexdigest()

    def archive(self, path, count=1, config=None):
        config = self.config if config is None else config
        with tarfile.open(path, 'w') as archive:
            for name, contents in [
                (
                    'manifest.json',
                    json.dumps([{'Config': 'config.json'}] * count).encode(),
                ),
                ('config.json', config),
            ]:
                member = tarfile.TarInfo(name)
                member.size = len(contents)
                archive.addfile(member, io.BytesIO(contents))

    def test_archive_binds_single_config_and_native_platform(self):
        path = self.root / 'image.tar'
        self.archive(path)
        self.assertEqual(
            osv.verify_archive(path, self.image_id, 'linux/amd64'),
            hashlib.sha256(path.read_bytes()).hexdigest(),
        )
        with self.assertRaisesRegex(ValueError, 'platform'):
            osv.verify_archive(path, self.image_id, 'linux/arm64')
        with self.assertRaisesRegex(ValueError, 'image ID'):
            osv.verify_archive(path, 'sha256:' + 'f' * 64, 'linux/amd64')
        self.archive(path, count=2)
        with self.assertRaisesRegex(ValueError, 'exactly one'):
            osv.verify_archive(path, self.image_id, 'linux/amd64')

    def test_clean_and_unfiltered_findings_are_valid(self):
        for findings in (False, True):
            report = envelope(self.image_id, findings)
            original = copy.deepcopy(report)
            osv.validate_report(report, self.image_id)
            self.assertEqual(report, original)
        report = envelope(self.image_id)
        del report['sarif']['runs'][0]['results']
        osv.validate_report(report, self.image_id)

    def test_malformed_reports_fail_closed(self):
        mutations = [
            lambda r: r.update(scanner={'name': 'trivy', 'version': '0.76.0'}),
            lambda r: r.update(image_id='sha256:' + 'f' * 64),
            lambda r: r.update(archive_sha256=''),
            lambda r: r.update(exit_code=128),
            lambda r: r.update(exit_code=0),
            lambda r: r.update(sarif={}),
            lambda r: r['sarif'].update(runs=[]),
            lambda r: r['sarif']['runs'][0]['tool']['driver'].update(
                version='2.4.0'
            ),
            lambda r: r['sarif']['runs'][0].update(results=None),
            lambda r: r['sarif']['runs'][0]['results'][0].update(
                ruleId='unknown'
            ),
            lambda r: r['sarif']['runs'][0]['results'][0].update(ruleIndex=8),
            lambda r: r['sarif']['runs'][0]['results'][0].update(message={}),
        ]
        for mutation in mutations:
            with self.subTest(mutation=mutation):
                report = envelope(self.image_id, True)
                mutation(report)
                with self.assertRaises(ValueError):
                    osv.validate_report(report, self.image_id)

    def invoke(self, code=0, report=None, write=True, report_format='sarif'):
        def run(command, **kwargs):
            if command[0] == 'docker':
                self.assertEqual(command[-1], self.image_id)
                self.archive(Path(command[3]))
            else:
                self.assertEqual(
                    command[:4], ['osv-scanner', 'scan', 'image', '--archive']
                )
                self.assertIn('--all-vulns', command)
                self.assertEqual(
                    '--all-packages' in command, report_format == 'json'
                )
                self.assertEqual(
                    command[command.index('--format') + 1], report_format
                )
                self.assertNotIn('--allow-no-lockfiles', command)
                self.assertEqual(
                    Path(command[command.index('--config') + 1]).read_text(), ''
                )
                self.assertNotIn('OSV_TEST_OPTION', kwargs['env'])
                if write:
                    target = Path(command[command.index('--output-file') + 1])
                    target.write_text(
                        json.dumps(sarif() if report is None else report)
                    )
            return subprocess.CompletedProcess(
                command, code if command[0] != 'docker' else 0
            )

        with (
            mock.patch.dict('os.environ', {'OSV_TEST_OPTION': 'ignored'}),
            mock.patch.object(osv.subprocess, 'run', side_effect=run),
        ):
            osv.scan_image(
                self.image_id, 'linux/amd64', self.root, report_format
            )

    def test_scan_preserves_sarif_and_checksums(self):
        report = sarif(True)
        self.invoke(code=1, report=report)
        saved = json.loads((self.root / 'scan.json').read_text())
        osv.validate_report(saved, self.image_id)
        self.assertEqual(saved['sarif'], report)
        self.assertEqual(
            json.loads((self.root / 'scan.sarif').read_text()), report
        )

    def test_missing_and_stale_reports_and_error_exit_rejected(self):
        for code, write in ((0, False), (128, True), (127, True)):
            with self.subTest(code=code):
                (self.root / 'scan.sarif').write_text(json.dumps(sarif()))
                with self.assertRaises(
                    (ValueError, subprocess.CalledProcessError)
                ):
                    self.invoke(code=code, write=write)
        with self.assertRaises(ValueError):
            self.invoke(report={})

    def test_json_scanning_preserves_complete_report_and_requires_inventory(
        self,
    ):
        for findings in (False, True):
            report = json_report(findings)
            self.invoke(code=int(findings), report=report, report_format='json')
            saved = json.loads((self.root / 'scan.json').read_text())
            osv.validate_report(saved, self.image_id, 'json')
            self.assertEqual(saved['report'], report)
            self.assertEqual(saved['report_format'], 'json')
            self.assertEqual(
                json.loads((self.root / 'osv-report.json').read_text()), report
            )
            with self.assertRaisesRegex(ValueError, 'format'):
                osv.validate_report(saved, self.image_id)
        for report in ({}, {'results': []}):
            with self.assertRaises(ValueError):
                self.invoke(report=report, report_format='json')

    def test_json_missing_stale_error_and_wrong_format_fail(self):
        for code, write in ((0, False), (128, True), (1, True)):
            (self.root / 'osv-report.json').write_text(
                json.dumps(json_report())
            )
            with self.assertRaises((ValueError, subprocess.CalledProcessError)):
                self.invoke(
                    code=code,
                    write=write,
                    report=json_report(),
                    report_format='json',
                )
        report = json_envelope(self.image_id)
        del report['report_format']
        with self.assertRaisesRegex(ValueError, 'format'):
            osv.validate_report(report, self.image_id, 'json')
        with self.assertRaisesRegex(ValueError, 'subject'):
            osv.validate_report(
                json_envelope('sha256:' + 'f' * 64), self.image_id, 'json'
            )
