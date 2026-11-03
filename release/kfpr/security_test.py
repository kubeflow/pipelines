"""Regression tests for release security qualification."""

import hashlib
import json
import subprocess
import tempfile
import unittest
from pathlib import Path
from unittest import mock

from kfpr import security
from kfpr.security import _pip_findings, _trivy_findings, qualify_security


def trivy(fixed=None):
    vulnerabilities = []
    if fixed is not None:
        vulnerabilities = [
            {
                'VulnerabilityID': 'CVE-test',
                'PkgName': 'example',
                'InstalledVersion': '1',
                'FixedVersion': fixed,
            }
        ]
    return {
        'SchemaVersion': 2,
        'Results': [{'Target': 'go.mod', 'Vulnerabilities': vulnerabilities}],
    }


def audit(fixes=None):
    return {
        'dependencies': [
            {
                'name': 'urllib3',
                'version': '2.2',
                'vulns': []
                if fixes is None
                else [{'id': 'CVE-test', 'fix_versions': fixes}],
            }
        ]
    }


class SecurityTest(unittest.TestCase):
    def test_historical_approvals_preserve_every_exact_identity_and_reason(
        self,
    ):
        # Fingerprint of the original 80 explicitly approved rows. Grouping must
        # not introduce a cross-product of versions, targets, or advisories.
        rows = security.APPROVED_2173
        self.assertEqual(len(rows), 80)
        encoded = json.dumps(sorted(rows), separators=(',', ':')).encode()
        self.assertEqual(
            hashlib.sha256(encoded).hexdigest(),
            '9b12315f2aca83d17a6fe2d3e6769a6ba214c86d76609b15beb60ec039e29a59',
        )

    def test_ci_image_reports_are_reparsed_and_bound_to_image_ids(self):
        for fixed, valid_id in [(None, True), ('2', True), (None, False)]:
            with (
                self.subTest(fixed=fixed, valid_id=valid_id),
                tempfile.TemporaryDirectory() as tmp,
            ):
                root = Path(tmp)
                image_id = 'sha256:' + 'a' * 64
                data = trivy(fixed)
                data.update(
                    ArtifactType='container_image',
                    Metadata={'ImageID': image_id if valid_id else 'wrong'},
                )
                raw = root / 'ci.json'
                raw.write_text(json.dumps(data))
                with mock.patch.object(security, '_scan', return_value=trivy()):
                    if fixed or not valid_id:
                        with self.assertRaises(RuntimeError):
                            qualify_security(
                                root,
                                root / 'reports',
                                [],
                                [],
                                image_reports=[(image_id, raw)],
                            )
                    else:
                        result = qualify_security(
                            root,
                            root / 'reports',
                            [],
                            [],
                            image_reports=[(image_id, raw)],
                        )
                        self.assertEqual(result['scans'][1]['target'], image_id)
                        self.assertTrue(result['passed'])
                self.assertEqual(
                    (root / 'reports/image-0.json').read_bytes(),
                    raw.read_bytes(),
                )

    def approved_source(self):
        data = trivy('2.7.0')
        result = data['Results'][0]
        result['Target'] = (
            '.github/resources/scripts/kfp-readiness/requirements.txt'
        )
        result['Vulnerabilities'][0].update(
            VulnerabilityID='CVE-2026-44431',
            PkgName='urllib3',
            InstalledVersion='2.6.3',
        )
        return data

    def test_explicit_2173_policy_accepts_only_documented_source_finding(self):
        with (
            tempfile.TemporaryDirectory() as temporary,
            mock.patch('kfpr.security.subprocess.run') as run,
        ):
            root = Path(temporary)
            data = self.approved_source()
            run.return_value = subprocess.CompletedProcess(
                [], 0, json.dumps(data), ''
            )
            report = qualify_security(
                root,
                root / 'reports',
                [],
                [],
                release_version='2.17.3',
                policy='2.17.3-approved-exceptions-v1',
            )
            self.assertTrue(report['passed'])
            self.assertEqual(report['fixable_count'], 1)
            self.assertEqual(report['accepted_count'], 1)
            self.assertEqual(report['blocking_count'], 0)
            self.assertEqual(
                report['accepted_findings'][0]['reason'], 'nonrelease-tooling'
            )
            self.assertEqual(
                json.loads((root / 'reports/source.json').read_text()), data
            )

    def test_2173_exceptions_are_exact_and_never_apply_to_images(self):
        finding = _trivy_findings(self.approved_source())[0]
        scan = {'name': 'source', 'findings': [finding]}
        for changed in (
            {**scan, 'name': 'image-0'},
            {**scan, 'findings': [{**finding, 'id': 'CVE-new'}]},
            {**scan, 'findings': [{**finding, 'package': 'other'}]},
            {**scan, 'findings': [{**finding, 'installed_version': '2.6.4'}]},
            {
                **scan,
                'findings': [
                    {**finding, 'target': 'sdk/python/requirements.txt'}
                ],
            },
        ):
            with self.subTest(scan=changed):
                result = security.evaluate_policy(
                    [changed],
                    release_version='2.17.3',
                    policy='2.17.3-approved-exceptions-v1',
                )
                self.assertEqual(result['blocking_count'], 1)
                self.assertEqual(result['accepted_count'], 0)
        strict = security.evaluate_policy([scan], release_version='2.17.3')
        self.assertEqual(strict['blocking_count'], 1)
        for version in ('', '2.17.4', '2.18.0'):
            with self.subTest(version=version), self.assertRaises(ValueError):
                security.evaluate_policy(
                    [scan],
                    release_version=version,
                    policy='2.17.3-approved-exceptions-v1',
                )

    def test_python39_exceptions_require_interpreter_and_exact_advisory(self):
        finding = {
            'id': 'PYSEC-2026-2275',
            'package': 'requests',
            'installed_version': '2.32.5',
            'fixable': True,
            'fixed_versions': ['2.33.0'],
        }
        for version, package, expected in (
            ('3.9', 'requests', 1),
            ('3.10', 'requests', 0),
            ('3.9', 'pip', 0),
            ('3.9', 'setuptools', 0),
        ):
            with self.subTest(version=version, package=package):
                result = security.evaluate_policy(
                    [
                        {
                            'name': 'python-0',
                            'python_version': version,
                            'findings': [{**finding, 'package': package}],
                        },
                    ],
                    release_version='2.17.3',
                    policy='2.17.3-approved-exceptions-v1',
                )
                self.assertEqual(result['accepted_count'], expected)
                self.assertEqual(result['blocking_count'], 1 - expected)

    def test_images_are_scanned_before_generator_advances(self):
        events = []

        def images():
            for image in ('sha256:first', 'sha256:second'):
                events.append(('build', image))
                yield image
                self.assertIn(('scan', image), events)
                events.append(('remove', image))

        def scan(command, **kwargs):
            if command[1] == 'image':
                events.append(('scan', command[-1]))
            return subprocess.CompletedProcess(
                command, 0, json.dumps(trivy()), ''
            )

        with (
            tempfile.TemporaryDirectory() as temporary,
            mock.patch('kfpr.security.subprocess.run', side_effect=scan),
        ):
            root = Path(temporary)
            qualify_security(root, root / 'reports', [], images())
        self.assertEqual(
            events,
            [
                ('build', 'sha256:first'),
                ('scan', 'sha256:first'),
                ('remove', 'sha256:first'),
                ('build', 'sha256:second'),
                ('scan', 'sha256:second'),
                ('remove', 'sha256:second'),
            ],
        )

    def test_policy_does_not_accept_installer_or_falsified_fixability(self):
        finding = _trivy_findings(self.approved_source())[0]
        with self.assertRaises(ValueError):
            security.evaluate_policy(
                [
                    {
                        'name': 'source',
                        'findings': [
                            {**finding, 'fixable': False},
                        ],
                    }
                ],
                release_version='2.17.3',
                policy='2.17.3-approved-exceptions-v1',
            )
        with self.assertRaises(ValueError):
            security.evaluate_policy(
                [], release_version='2.17.3', policy='accept-everything'
            )
        self.assertTrue(
            all(
                scope != 'image' and package not in ('pip', 'setuptools')
                for scope, target, package, advisory, version, reason in security.APPROVED_2173
            )
        )

    def test_rejects_invalid_trivy_schema(self):
        for report in (
            {},
            {'SchemaVersion': 3},
            {'SchemaVersion': 2, 'Results': []},
            {'SchemaVersion': 2, 'Results': [{}]},
        ):
            with self.subTest(report=report), self.assertRaises(ValueError):
                _trivy_findings(report)
        report = trivy()
        report['Results'][0]['Vulnerabilities'] = {}
        with self.assertRaises(ValueError):
            _trivy_findings(report)

    def test_requires_explicit_pip_fix_metadata(self):
        report = audit([])
        del report['dependencies'][0]['vulns'][0]['fix_versions']
        with self.assertRaises(ValueError):
            _pip_findings(report)

    def test_third_party_skip_blocks_but_local_skip_is_recorded(self):
        dependency = {'name': 'kfp', 'skip_reason': 'Unpublished release'}
        findings, skipped = _pip_findings({'dependencies': [dependency]})
        self.assertEqual(findings, [])
        self.assertEqual(
            skipped, [{'package': 'kfp', 'reason': 'Unpublished release'}]
        )
        dependency['name'] = 'urllib3'
        with self.assertRaisesRegex(ValueError, 'skipped third-party'):
            _pip_findings({'dependencies': [dependency]})

    def test_unfixable_findings_preserved(self):
        with (
            tempfile.TemporaryDirectory() as temporary,
            mock.patch('kfpr.security.subprocess.run') as run,
        ):
            root = Path(temporary)
            run.return_value = subprocess.CompletedProcess(
                [], 0, json.dumps(trivy('')), ''
            )
            report = qualify_security(
                root, root / 'reports', [], ['sha256:built']
            )
            self.assertTrue(report['passed'])
            self.assertEqual(report['residual_count'], 2)
            saved = json.loads((root / 'reports/security.json').read_text())
            self.assertEqual(saved, report)
            self.assertEqual(saved['scans'][1]['target'], 'sha256:built')
            self.assertTrue((root / 'reports/image-0.json').exists())

    def test_os_packaged_go_binary_findings_block_release(self):
        # Trivy's precise default omits Go modules in OS-owned binaries.
        def scan(command, **kwargs):
            comprehensive = (
                '--detection-priority' in command
                and command[command.index('--detection-priority') + 1]
                == 'comprehensive'
            )
            report = (
                trivy('2')
                if command[1] == 'image' and comprehensive
                else trivy()
            )
            return subprocess.CompletedProcess(
                command, 0, json.dumps(report), ''
            )

        with (
            tempfile.TemporaryDirectory() as temporary,
            mock.patch('kfpr.security.subprocess.run', side_effect=scan),
        ):
            root = Path(temporary)
            with self.assertRaisesRegex(RuntimeError, '1 fixable'):
                qualify_security(root, root / 'reports', [], ['sha256:built'])
            saved = json.loads((root / 'reports/security.json').read_text())
            self.assertEqual(saved['fixable_count'], 1)
            self.assertFalse(saved['passed'])

    def test_fixable_findings_block_and_save_report(self):
        with (
            tempfile.TemporaryDirectory() as temporary,
            mock.patch('kfpr.security.subprocess.run') as run,
        ):
            root = Path(temporary)
            run.return_value = subprocess.CompletedProcess(
                [], 0, json.dumps(trivy('2')), ''
            )
            with self.assertRaisesRegex(RuntimeError, '1 fixable'):
                qualify_security(root, root / 'reports', [], [])
            report = json.loads((root / 'reports/security.json').read_text())
            self.assertFalse(report['passed'])
            self.assertEqual(report['fixable_count'], 1)

    def test_python_env_site_packages_audited_without_installing_scanner(self):
        with (
            tempfile.TemporaryDirectory() as temporary,
            mock.patch('kfpr.security.subprocess.run') as run,
        ):
            root = Path(temporary)
            run.side_effect = [
                subprocess.CompletedProcess([], 0, json.dumps(trivy()), ''),
                subprocess.CompletedProcess(
                    [],
                    0,
                    '{"version": "3.9", "paths": ["/venv/lib/python3.9/site-packages"]}',
                    '',
                ),
                subprocess.CompletedProcess(
                    [], 1, json.dumps(audit([])), 'Found one vulnerability'
                ),
            ]
            report = qualify_security(
                root, root / 'reports', [root / 'py39'], []
            )
            self.assertEqual(report['residual_count'], 1)
            self.assertEqual(
                run.call_args_list[1].args[0][0], str(root / 'py39/bin/python')
            )
            self.assertIn(
                '/venv/lib/python3.9/site-packages',
                run.call_args_list[2].args[0],
            )
            self.assertEqual(run.call_args_list[2].args[0][0], 'pip-audit')

    def test_scanner_failures_and_malformed_output_fail_closed(self):
        failures = [
            FileNotFoundError('trivy'),
            subprocess.CompletedProcess([], 2, '{}', 'failed'),
            subprocess.CompletedProcess([], 0, 'not JSON', ''),
        ]
        for failure in failures:
            with (
                self.subTest(failure=failure),
                tempfile.TemporaryDirectory() as temporary,
                mock.patch('kfpr.security.subprocess.run') as run,
            ):
                root = Path(temporary)
                if isinstance(failure, Exception):
                    run.side_effect = failure
                else:
                    run.return_value = failure
                with self.assertRaisesRegex(RuntimeError, 'scanner errors'):
                    qualify_security(root, root / 'reports', [], [])
                self.assertTrue(
                    json.loads((root / 'reports/security.json').read_text())[
                        'errors'
                    ]
                )

    def test_missing_source_ecosystem_blocks(self):
        with (
            tempfile.TemporaryDirectory() as temporary,
            mock.patch('kfpr.security.subprocess.run') as run,
        ):
            root = Path(temporary)
            (root / 'frontend').mkdir()
            (root / 'frontend/package-lock.json').write_text('{}')
            run.return_value = subprocess.CompletedProcess(
                [], 0, json.dumps(trivy()), ''
            )
            with self.assertRaises(RuntimeError):
                qualify_security(root, root / 'reports', [], [])
            self.assertIn(
                'omitted required dependency target',
                (root / 'reports/security.json').read_text(),
            )


if __name__ == '__main__':
    unittest.main()
