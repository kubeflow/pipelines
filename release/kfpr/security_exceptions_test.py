"""Explicit security exceptions remain exact, bounded, and auditable."""

import copy
import json
import unittest
from datetime import datetime, timezone
from pathlib import Path
from tempfile import TemporaryDirectory
from unittest.mock import patch

from kfpr import security
from kfpr import security_exceptions as exceptions

NOW = datetime(2026, 10, 22, 23, 59, tzinfo=timezone.utc)


class SecurityExceptionsTest(unittest.TestCase):
    def document(self, scope='source', artifact='tools/requirements.txt'):
        return {
            'schema': 1,
            'exceptions': [
                {
                    'scope': scope,
                    'artifact': artifact,
                    'vulnerability': 'CVE-test',
                    'package': 'example',
                    'installed_version': '1.0',
                    'justification': 'Tooling only; tracked for the next update.',
                    'expires': '2026-10-22',
                }
            ],
        }

    def scan(self, scope='source', artifact='tools/requirements.txt'):
        return [
            {
                'name': 'source' if scope == 'source' else scope + '-0',
                'target': artifact,
                'python_version': artifact,
                'findings': [
                    {
                        'target': artifact,
                        'id': 'CVE-test',
                        'package': 'example',
                        'installed_version': '1.0',
                        'fixed_versions': ['1.1'],
                        'fixable': True,
                    }
                ],
            }
        ]

    def test_exact_matches_all_scopes_and_preserves_reason_expiry(self):
        for scope, artifact in [
            ('source', 'tools/requirements.txt'),
            ('python', '3.9'),
            ('image', 'sha256:' + 'a' * 64),
        ]:
            document = self.document(scope, artifact)
            scans = self.scan(scope, artifact)
            result = security.evaluate_policy(
                scans,
                policy='approved-exceptions',
                exceptions=document,
                now=NOW,
            )
            self.assertEqual(result['blocking_count'], 0)
            self.assertEqual(
                result['accepted_findings'][0]['expires'], '2026-10-22'
            )
            self.assertEqual(
                result['accepted_findings'][0]['reason'],
                document['exceptions'][0]['justification'],
            )
            other = self.scan(scope, 'different')
            self.assertEqual(
                security.evaluate_policy(
                    other,
                    policy='approved-exceptions',
                    exceptions=document,
                    now=NOW,
                )['blocking_count'],
                1,
            )
            for field in ('id', 'package', 'installed_version'):
                changed = copy.deepcopy(scans)
                changed[0]['findings'][0][field] = 'different'
                result = security.evaluate_policy(
                    changed,
                    policy='approved-exceptions',
                    exceptions=document,
                    now=NOW,
                )
                self.assertEqual(result['blocking_count'], 1)

    def test_invalid_documents_fail_closed(self):
        valid = self.document()
        invalid = [
            None,
            [],
            {},
            {**valid, 'extra': 1},
            {**valid, 'schema': True},
            {**valid, 'exceptions': 'all'},
        ]
        for key, value in [
            ('scope', 'all'),
            ('artifact', '*'),
            ('justification', ''),
            ('expires', '2026-10-21'),
            ('expires', '20261022'),
            ('package', 3),
        ]:
            document = copy.deepcopy(valid)
            document['exceptions'][0][key] = value
            invalid.append(document)
        duplicate = copy.deepcopy(valid)
        duplicate['exceptions'] *= 2
        invalid.append(duplicate)
        for document in invalid:
            with self.subTest(document=document), self.assertRaises(ValueError):
                exceptions.validate_exceptions(document, now=NOW)

    def test_expiry_revalidated_and_other_policies_forbid_exceptions(self):
        document = self.document()
        exceptions.validate_exceptions(document, now=NOW)
        with self.assertRaisesRegex(ValueError, 'expired'):
            security.evaluate_policy(
                self.scan(),
                policy='approved-exceptions',
                exceptions=document,
                now=datetime(2026, 10, 23, tzinfo=timezone.utc),
            )
        for policy in ('zero-fixable', '2.17.3-approved-exceptions-v1'):
            with self.assertRaises(ValueError):
                security.evaluate_policy(
                    self.scan(),
                    policy=policy,
                    exceptions=document,
                    release_version='2.17.3',
                    now=NOW,
                )
        with self.assertRaises(ValueError):
            security.evaluate_policy(
                self.scan(), policy='approved-exceptions', now=NOW
            )

    def test_scanner_report_retains_exception_document_without_waiving_errors(
        self,
    ):
        document = self.document('image', 'sha256:' + 'a' * 64)
        document['exceptions'][0]['expires'] = '2099-01-01'
        data = {
            'SchemaVersion': 2,
            'Results': [
                {
                    'Target': 'node_modules/example',
                    'Vulnerabilities': [
                        {
                            'VulnerabilityID': 'CVE-test',
                            'PkgName': 'example',
                            'InstalledVersion': '1.0',
                            'FixedVersion': '1.1',
                        }
                    ],
                }
            ],
        }
        with TemporaryDirectory() as tmp:
            root = Path(tmp)
            with patch.object(
                security,
                '_scan',
                side_effect=[
                    {'SchemaVersion': 2, 'Results': [{'Target': 'go.mod'}]},
                    data,
                ],
            ):
                report = security.qualify_security(
                    root,
                    root / 'reports',
                    [],
                    ['sha256:' + 'a' * 64],
                    policy='approved-exceptions',
                    exceptions=document,
                )
            self.assertEqual(report['exceptions'], document)
            self.assertEqual(
                report['exceptions_digest'],
                exceptions.exception_digest(document),
            )
            self.assertEqual(report['accepted_count'], 1)
            with patch.object(
                security,
                '_scan',
                side_effect=RuntimeError('scanner unavailable'),
            ):
                with self.assertRaises(RuntimeError):
                    security.qualify_security(
                        root,
                        root / 'failed',
                        [],
                        [],
                        policy='approved-exceptions',
                        exceptions=document,
                    )
            self.assertFalse(
                json.loads((root / 'failed/security.json').read_text())[
                    'passed'
                ]
            )

    def test_load_and_canonical_digest(self):
        document = self.document()
        with TemporaryDirectory() as tmp:
            path = Path(tmp) / 'exceptions.json'
            path.write_text(json.dumps(document))
            self.assertEqual(
                exceptions.load_exceptions(path, now=NOW), document
            )
            self.assertEqual(
                exceptions.exception_digest(document),
                exceptions.exception_digest(
                    dict(reversed(list(document.items())))
                ),
            )
            path.write_text('{"schema":1,"schema":1,"exceptions":[]}')
            with self.assertRaises(ValueError):
                exceptions.load_exceptions(path, now=NOW)
