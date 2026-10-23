"""Qualification evidence binds the exact approved exception document."""

import copy
import hashlib
import json
import unittest
from datetime import datetime, timezone
from pathlib import Path
from tempfile import TemporaryDirectory
from unittest import mock

from kfpr import qualification as q
from kfpr.core import (
    CommandRunner,
    ReleaseContext,
    ReleaseMetadata,
    ReleaseState,
)

SHA = 'a' * 40


class QualificationExceptionsTest(unittest.TestCase):
    def setUp(self):
        directory = TemporaryDirectory()
        self.addCleanup(directory.cleanup)
        self.root = Path(directory.name)
        self.context = ReleaseContext(
            self.root,
            ReleaseState(self.root / 'state.json'),
            CommandRunner(),
            ReleaseMetadata.from_version('patch', '2.17.3'),
            'origin',
            False,
            False,
        )
        self.document = {
            'schema': 1,
            'exceptions': [
                {
                    'scope': 'source',
                    'artifact': 'tools/requirements.txt',
                    'vulnerability': 'CVE-test',
                    'package': 'example',
                    'installed_version': '1.0',
                    'justification': 'Reviewed tooling dependency.',
                    'expires': '2999-01-01',
                }
            ],
        }
        self.path = self.root / 'exceptions.json'
        self.write_document()
        self.context.state.answers.update(
            security_policy='approved-exceptions',
            security_exceptions=str(self.path),
        )
        patcher = mock.patch('kfpr.steps.validate_release_pr', return_value=SHA)
        patcher.start()
        self.addCleanup(patcher.stop)
        ci = mock.patch('kfpr.ci_qualification_client.verify_run')
        ci.start()
        self.addCleanup(ci.stop)

    def write_document(self):
        self.path.write_text(json.dumps(self.document))

    def save(self, report):
        path = self.root / 'qualification.json'
        path.write_text(json.dumps(report))
        self.context.state.answers.update(
            qualification_path=str(path),
            qualification_digest=hashlib.sha256(path.read_bytes()).hexdigest(),
        )

    def security(self, legacy=False):
        from kfpr.security_exceptions import exception_digest

        scans = [{'name': 'source', 'target': '/source', 'findings': []}]
        if not legacy:
            scans[0]['findings'] = [
                {
                    'target': 'tools/requirements.txt',
                    'id': 'CVE-test',
                    'package': 'example',
                    'installed_version': '1.0',
                    'fixed_versions': ['1.1'],
                    'fixable': True,
                }
            ]
        policy = 'zero-fixable' if legacy else 'approved-exceptions'
        security = {
            'passed': True,
            'scanner': q.SCANNER,
            'errors': [],
            'scans': scans,
            'policy': policy,
            'release_version': '2.17.3',
            **q.evaluate_policy(
                scans,
                release_version='2.17.3',
                policy=policy,
                **({} if legacy else {'exceptions': self.document}),
            ),
        }
        if not legacy:
            security.update(
                exceptions=copy.deepcopy(self.document),
                exceptions_digest=exception_digest(self.document),
            )
        return security

    def evidence(self, legacy=False):
        report = {
            **q._identity(self.context, SHA),
            'qualified_at': datetime.now(timezone.utc).isoformat(),
            'security': self.security(legacy),
            'images': [],
        }
        self.save(report)
        return report

    def test_identity_binds_configured_document(self):
        identity = q._identity(self.context, SHA)
        self.assertEqual(identity.get('exceptions'), self.document)
        self.assertRegex(
            identity.get('exceptions_digest', ''), r'^[0-9a-f]{64}$'
        )

    def test_changed_or_expired_file_rejected_at_publication(self):
        self.evidence()
        q.require_qualification(self.context)
        self.document['exceptions'][0]['justification'] = 'Changed approval'
        self.write_document()
        with self.assertRaisesRegex(ValueError, 'different|changed'):
            q.require_qualification(self.context)
        self.document['exceptions'][0]['expires'] = '2000-01-01'
        self.write_document()
        with self.assertRaisesRegex(ValueError, 'expired'):
            q.require_qualification(self.context)

    def test_embedded_security_document_and_digest_must_match(self):
        valid = self.evidence()
        for field, value in (
            ('exceptions', {'schema': 1, 'exceptions': []}),
            ('exceptions_digest', 'b' * 64),
        ):
            report = copy.deepcopy(valid)
            report['security'][field] = value
            self.save(report)
            with self.assertRaisesRegex(ValueError, '[Ee]xception'):
                q.require_qualification(self.context)

    def test_missing_file_and_wrong_policy_fail_closed(self):
        self.evidence()
        self.path.unlink()
        with self.assertRaises((ValueError, OSError)):
            q.require_qualification(self.context)
        self.write_document()
        self.context.state.answers['security_policy'] = 'zero-fixable'
        with self.assertRaises(ValueError):
            q._identity(self.context, SHA)
        self.context.state.answers['security_policy'] = 'approved-exceptions'
        self.context.state.answers.pop('security_exceptions')
        with self.assertRaises(ValueError):
            q._identity(self.context, SHA)

    def test_strict_policy_identity_has_no_exception_fields(self):
        self.context.state.answers.clear()
        report = self.evidence(legacy=True)
        self.assertNotIn('exceptions', report)
        self.assertNotIn('exceptions_digest', report)
        q.require_qualification(self.context)

    def test_qualification_passes_document_to_scan_and_records_identity(self):
        source = self.root / 'source'
        source.mkdir()
        with (
            mock.patch.object(
                q.tempfile, 'mkdtemp', return_value=str(self.root)
            ),
            mock.patch.object(q, '_run'),
            mock.patch.object(q.subprocess, 'check_output', return_value=''),
            mock.patch(
                'kfpr.ci_qualification_client.qualify_ci',
                return_value={
                    'images': [],
                    'wheels': None,
                    'security': self.security(),
                    'run': {'id': 123},
                },
            ) as scan,
        ):
            q.qualify_release(self.context)
        self.assertIs(scan.call_args.args[0], self.context)
        report = json.loads(
            Path(self.context.state.answers['qualification_path']).read_text()
        )
        self.assertEqual(report['exceptions'], self.document)
        self.assertEqual(
            report['exceptions_digest'], report['security']['exceptions_digest']
        )
        q.require_qualification(self.context)

    def test_unchanged_document_expiry_is_rechecked_before_publication(self):
        self.evidence()
        with mock.patch('kfpr.security_exceptions.datetime') as clock:
            clock.now.return_value = datetime(2999, 1, 2, tzinfo=timezone.utc)
            with self.assertRaisesRegex(ValueError, 'expired'):
                q.require_qualification(self.context)

    def test_new_finding_is_not_accepted_by_saved_approval_counts(self):
        report = self.evidence()
        self.assertEqual(report['security']['accepted_count'], 1)
        report['security']['scans'][0]['findings'][0]['id'] = 'CVE-other'
        self.save(report)
        with self.assertRaisesRegex(ValueError, 'findings do not satisfy'):
            q.require_qualification(self.context)
