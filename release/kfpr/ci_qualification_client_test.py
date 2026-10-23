"""Verify remote qualification provenance, complete inventories, and policy."""

import copy
import hashlib
import json
import tempfile
import unittest
from pathlib import Path
from types import SimpleNamespace
from unittest import mock

from . import ci_qualification_client as client
from .ci_security_runner_test import clean_report
from .osv_image_scan_test import json_envelope

IMAGE = 'sha256:' + 'd' * 64
IMAGES = [
    {'name': 'api', 'platform': 'linux/amd64'},
    {'name': 'api', 'platform': 'linux/arm64'},
]


class ArtifactTest(unittest.TestCase):
    def setUp(self):
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        self.root = Path(temporary.name)
        self.identity = {
            'version': '2.18.0',
            'source_sha': 'a' * 40,
            'workflow_sha': 'b' * 40,
            'request_id': 'c' * 32,
            'run_id': '123',
            'run_attempt': 1,
            'repository': client.REPO,
            'workflow': client.WORKFLOW,
            'title': 'request title',
        }
        self.expected = {
            'schema': 1,
            **{
                key: self.identity[key]
                for key in (
                    'version',
                    'source_sha',
                    'workflow_sha',
                    'request_id',
                    'run_id',
                    'run_attempt',
                )
            },
        }

    def write(self, relative, data):
        path = self.root / relative
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(json.dumps(data))
        return hashlib.sha256(path.read_bytes()).hexdigest()

    def artifact(self):
        data = clean_report(
            '/ci/source/go.mod',
            [{'name': 'example', 'version': '1', 'ecosystem': 'Go'}],
        )
        data['results'].extend(
            clean_report(
                '/ci/source/frontend/package-lock.json',
                [{'name': 'example', 'version': '1', 'ecosystem': 'npm'}],
            )['results']
        )
        digest = self.write('security/scans/source.json', data)
        source = {'raw_report': 'scans/source.json', 'sha256': digest}
        security = {
            **self.expected,
            'scanner': client.SCANNER,
            'include_sdk': False,
            'passed': True,
            'wheels': None,
            'source': source,
            'scans': [
                {
                    'name': 'source',
                    'target': 'source',
                    'python_version': None,
                    'source_root': '/ci/source',
                    'exit_code': 0,
                    **source,
                }
            ],
        }
        manifest = {
            **self.expected,
            'scanner': client.SCANNER,
            'report_format': 'json',
            'include_backend': True,
            'include_sdk': False,
            'images': [],
            'security_manifest_sha256': self.write(
                'security/security.json', security
            ),
        }
        for image in IMAGES:
            filename = f'scans/api-{image["platform"].split("/")[1]}.json'
            manifest['images'].append(
                {
                    **image,
                    'scanner': client.SCANNER,
                    'report_format': 'json',
                    'image_id': IMAGE,
                    'scan_file': filename,
                    'scan_sha256': self.write(filename, json_envelope(IMAGE)),
                }
            )
        self.write('manifest.json', manifest)
        return manifest

    def test_all_images_and_source_are_verified_without_local_ci_paths(self):
        self.artifact()
        result = client.read_artifact(self.root, IMAGES, [], self.identity)
        self.assertEqual(len(result['images']), 2)
        self.assertEqual(len(result['scans']), 3)
        self.assertIsNone(result['wheels'])

    def test_identity_and_scope_mutations_fail_before_acceptance(self):
        baseline = self.artifact()
        changes = [
            ('source_sha', 'e' * 40),
            ('workflow_sha', 'e' * 40),
            ('run_attempt', 2),
            ('request_id', 'f' * 32),
            ('include_sdk', True),
            ('include_backend', False),
            ('scanner', {'name': 'trivy', 'version': '0.76.0'}),
            ('report_format', 'sarif'),
        ]
        for key, value in changes:
            self.write('manifest.json', {**baseline, key: value})
            with self.subTest(key=key), self.assertRaises(ValueError):
                client.read_artifact(self.root, IMAGES, [], self.identity)

    def test_missing_duplicate_or_unexpected_image_is_rejected(self):
        baseline = self.artifact()
        for images in (
            [baseline['images'][0]],
            [baseline['images'][0]] * 2,
            [
                {**baseline['images'][0], 'name': 'unexpected'},
                baseline['images'][1],
            ],
        ):
            self.write('manifest.json', {**baseline, 'images': images})
            with (
                self.subTest(images=images),
                self.assertRaisesRegex(ValueError, 'inventory'),
            ):
                client.read_artifact(self.root, IMAGES, [], self.identity)

    def test_image_checksum_format_and_subject_are_verified(self):
        baseline = self.artifact()
        path = baseline['images'][0]['scan_file']
        for change in ('checksum', 'trivy', 'subject', 'sarif'):
            manifest = copy.deepcopy(baseline)
            envelope = json_envelope(IMAGE)
            if change == 'trivy':
                envelope = {'SchemaVersion': 2, 'Results': []}
            elif change == 'subject':
                envelope['image_id'] = 'sha256:' + 'f' * 64
            elif change == 'sarif':
                envelope['report_format'] = 'sarif'
            digest = self.write(path, envelope)
            manifest['images'][0]['scan_sha256'] = (
                'f' * 64 if change == 'checksum' else digest
            )
            self.write('manifest.json', manifest)
            with self.subTest(change=change), self.assertRaises(ValueError):
                client.read_artifact(self.root, IMAGES, [], self.identity)

    def test_security_envelope_and_raw_report_checksums_are_verified(self):
        baseline = self.artifact()
        for target in ('security/security.json', 'security/scans/source.json'):
            path = self.root / target
            original = path.read_bytes()
            path.write_text('{}')
            with (
                self.subTest(target=target),
                self.assertRaisesRegex(ValueError, 'checksum mismatch'),
            ):
                client.read_artifact(self.root, IMAGES, [], self.identity)
            path.write_bytes(original)
        self.assertTrue(baseline['security_manifest_sha256'])

    def test_expected_minors_are_forwarded_exactly_to_security_verifier(self):
        manifest = self.artifact()
        manifest['include_sdk'] = True
        self.write('manifest.json', manifest)
        with mock.patch.object(
            client, 'verify_security', return_value={'wheels': {}, 'scans': []}
        ) as verify:
            client.read_artifact(
                self.root, IMAGES, ['3.10', '3.13'], self.identity
            )
        self.assertEqual(
            verify.call_args.args,
            (self.root / 'security', self.expected, {'3.10', '3.13'}),
        )

    def test_symlinked_evidence_is_rejected(self):
        self.artifact()
        path = self.root / 'scans/api-amd64.json'
        payload = path.read_bytes()
        path.unlink()
        outside = self.root / 'outside.json'
        outside.write_bytes(payload)
        path.symlink_to(outside)
        with self.assertRaisesRegex(ValueError, 'escaping path'):
            client.read_artifact(self.root, IMAGES, [], self.identity)


class DispatchPolicyTest(unittest.TestCase):
    def setUp(self):
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        self.root = Path(temporary.name)
        self.output = self.root / 'qualification'
        self.context = SimpleNamespace(
            root=self.root,
            include_backend=False,
            include_sdk=True,
            state=SimpleNamespace(answers={'release_sha': 'a' * 40}),
            metadata=SimpleNamespace(tag='2.18.0'),
            runner=mock.Mock(),
        )
        self.result = {'images': [], 'wheels': {'passed': True}, 'scans': []}
        self.final = {'status': 'completed', 'conclusion': 'success'}
        self.inputs = None
        self.identity = None
        self.api = mock.patch.object(
            client, '_api', side_effect=self.fake_api
        ).start()
        self.dispatch = mock.patch.object(
            client, 'dispatch_and_watch', side_effect=self.fake_dispatch
        ).start()
        self.read = mock.patch.object(
            client,
            'read_artifact',
            side_effect=lambda *args: copy.deepcopy(self.result),
        ).start()
        self.verify = mock.patch.object(client, 'verify_run').start()
        self.addCleanup(mock.patch.stopall)

    def fake_api(self, context, endpoint):
        if endpoint == 'commits/master':
            return {'sha': 'b' * 40}
        return {'type': 'file', 'path': client.WORKFLOW}

    def fake_dispatch(self, api, run, identity, inputs):
        self.inputs = inputs
        identity.update(run_id='123', run_attempt=2)
        self.identity = identity
        return self.final

    def qualify(self):
        return client.qualify_ci(
            self.context, [], ['3.10', '3.13'], self.output
        )

    def finding(self, name, target):
        return {
            'name': name,
            'target': target,
            'python_version': target,
            'findings': [
                {
                    'id': 'GHSA-new',
                    'aliases': ['CVE-alias'],
                    'package': 'example',
                    'installed_version': '1.0',
                    'fixed_versions': ['2.0'],
                    'fixable': True,
                    'target': target,
                }
            ],
        }

    def test_sdk_only_scope_dispatches_and_live_run_is_reverified(self):
        result = self.qualify()
        self.assertEqual(self.inputs['include_backend'], 'false')
        self.assertEqual(self.inputs['include_sdk'], 'true')
        self.assertEqual(self.read.call_args.args[1:3], ([], ['3.10', '3.13']))
        self.verify.assert_called_once_with(self.context, self.identity)
        self.assertTrue(result['security']['passed'])
        download = self.context.runner.run.call_args.args[0]
        self.assertIn('--name', download)
        self.assertTrue(
            any(
                value.startswith('kfpr-images-') and value.endswith('-2')
                for value in download
            )
        )

    def test_fixable_findings_block_and_preserve_policy_evidence(self):
        self.result['scans'] = [self.finding('source', 'go.mod')]
        with self.assertRaisesRegex(ValueError, 'fixable blocking'):
            self.qualify()
        evidence = json.loads((self.output / 'security.json').read_text())
        self.assertFalse(evidence['passed'])
        self.assertEqual(evidence['blocking_count'], 1)
        self.verify.assert_called_once()

    def test_exact_alias_exceptions_apply_across_all_scopes(self):
        self.result['scans'] = [
            self.finding('source', 'go.mod'),
            self.finding('python-3.10', '3.10'),
            self.finding('image-api-amd64', IMAGE),
        ]
        document = {
            'schema': 1,
            'exceptions': [
                {
                    'scope': scope,
                    'artifact': artifact,
                    'vulnerability': 'CVE-alias',
                    'package': 'example',
                    'installed_version': '1.0',
                    'justification': 'Approved bounded remediation plan.',
                    'expires': '2099-01-01',
                }
                for scope, artifact in [
                    ('source', 'go.mod'),
                    ('python', '3.10'),
                    ('image', IMAGE),
                ]
            ],
        }
        path = self.root / 'exceptions.json'
        path.write_text(json.dumps(document))
        self.context.state.answers.update(
            security_policy='approved-exceptions', security_exceptions=str(path)
        )
        result = self.qualify()
        self.assertEqual(result['security']['accepted_count'], 3)
        self.assertEqual(result['security']['blocking_count'], 0)
        self.assertEqual(result['security']['exceptions'], document)

    def test_failed_workflow_downloads_diagnostics_without_acceptance(self):
        self.final['conclusion'] = 'failure'
        with self.assertRaisesRegex(RuntimeError, 'Qualification CI failed'):
            self.qualify()
        self.read.assert_not_called()
        self.verify.assert_not_called()
        call = self.context.runner.run.call_args
        self.assertFalse(call.kwargs['check'])
        self.assertIn(str(self.output.resolve() / 'failed-run'), call.args[0])
        self.assertFalse((self.output / 'security.json').exists())

    def test_live_run_changed_after_download_cannot_be_accepted(self):
        self.verify.side_effect = ValueError('Run attempt changed')
        with self.assertRaisesRegex(ValueError, 'Run attempt changed'):
            self.qualify()
        self.assertFalse((self.output / 'security.json').exists())

    def test_invalid_artifact_cannot_reach_policy_acceptance(self):
        self.read.side_effect = ValueError('Invalid artifact')
        with self.assertRaisesRegex(ValueError, 'Invalid artifact'):
            self.qualify()
        self.verify.assert_not_called()
        self.assertFalse((self.output / 'security.json').exists())


if __name__ == '__main__':
    unittest.main()
