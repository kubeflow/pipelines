"""Package-specific OSV policy and exception regression tests."""

import copy
import unittest

from . import osv_reports
from .security import evaluate_policy


def report(name='requests', ecosystem='PyPI'):
    return {
        'results': [
            {
                'source': {'path': 'requirements.txt', 'type': 'lockfile'},
                'packages': [
                    {
                        'package': {
                            'name': name,
                            'version': '1.0',
                            'ecosystem': ecosystem,
                        },
                        'vulnerabilities': [
                            {
                                'id': 'GHSA-test',
                                'aliases': ['CVE-2026-1234'],
                                'affected': [
                                    {
                                        'package': {
                                            'name': name,
                                            'ecosystem': ecosystem,
                                        },
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
                        ],
                    }
                ],
            }
        ]
    }


class ReportsTest(unittest.TestCase):
    def test_clean_inventory_preserved(self):
        data = report()
        del data['results'][0]['packages'][0]['vulnerabilities']
        self.assertEqual(osv_reports.findings(data), [])
        self.assertEqual(
            osv_reports.inventory(data),
            [('requirements.txt', 'PyPI', 'requests', '1.0')],
        )

    def test_fixes_are_package_and_ecosystem_scoped(self):
        for ecosystem in ('PyPI', 'Go', 'npm', 'Debian:12', 'Alpine:v3.23'):
            with self.subTest(ecosystem=ecosystem):
                data = report(ecosystem=ecosystem)
                vuln = data['results'][0]['packages'][0]['vulnerabilities'][0]
                other = copy.deepcopy(vuln['affected'][0])
                other['package']['name'] = 'unrelated'
                other['ranges'][0]['events'][1]['fixed'] = '99.0'
                distro = copy.deepcopy(other)
                distro['package'] = {
                    'name': 'requests',
                    'ecosystem': 'Debian:11',
                }
                vuln['affected'].extend([other, distro])
                finding = osv_reports.findings(data)[0]
                self.assertEqual(finding['fixed_versions'], ['2.0'])
                self.assertEqual(finding['aliases'], ['CVE-2026-1234'])

    def test_ubuntu_lts_alias_retains_release_and_support_channel(self):
        data = report('util-linux', 'Ubuntu:22.04')
        affected = data['results'][0]['packages'][0]['vulnerabilities'][0][
            'affected'
        ]
        affected[0]['package']['ecosystem'] = 'Ubuntu:22.04:LTS'
        for ecosystem in (
            'Ubuntu:20.04:LTS',
            'Ubuntu:Pro:22.04:LTS',
            'Ubuntu',
            'Debian:22.04',
        ):
            unrelated = copy.deepcopy(affected[0])
            unrelated['package']['ecosystem'] = ecosystem
            unrelated['ranges'][0]['events'][1]['fixed'] = '99.0'
            affected.append(unrelated)
        self.assertEqual(
            osv_reports.findings(data)[0]['fixed_versions'], ['2.0']
        )
        del affected[0]
        with self.assertRaises(ValueError):
            osv_reports.findings(data)
        data = report('util-linux', 'Ubuntu:Pro:22.04')
        data['results'][0]['packages'][0]['vulnerabilities'][0]['affected'][0][
            'package'
        ]['ecosystem'] = 'Ubuntu:Pro:22.04:LTS'
        self.assertEqual(
            osv_reports.findings(data)[0]['fixed_versions'], ['2.0']
        )

    def test_last_affected_limit_and_versions_are_not_fixes(self):
        for event in ({'last_affected': '2.0'}, {'limit': '2.0'}):
            data = report()
            affected = data['results'][0]['packages'][0]['vulnerabilities'][0][
                'affected'
            ][0]
            affected['ranges'][0]['events'][1] = event
            self.assertFalse(osv_reports.findings(data)[0]['fixable'])
        affected.pop('ranges')
        affected['versions'] = ['1.0']
        self.assertFalse(osv_reports.findings(data)[0]['fixable'])

    def test_normalizes_only_pypi_names(self):
        data = report('Some_Package')
        data['results'][0]['packages'][0]['vulnerabilities'][0]['affected'][0][
            'package'
        ]['name'] = 'some-package'
        self.assertTrue(osv_reports.findings(data)[0]['fixable'])
        data = report('Some_Package', 'npm')
        data['results'][0]['packages'][0]['vulnerabilities'][0]['affected'][0][
            'package'
        ]['name'] = 'some-package'
        with self.assertRaises(ValueError):
            osv_reports.findings(data)

    def test_malformed_or_unmatched_advisories_fail_closed(self):
        for mutation in (
            'affected',
            'package',
            'ranges',
            'events',
            'introduced',
            'aliases',
            'version',
            'generic',
        ):
            data = report()
            package = data['results'][0]['packages'][0]
            vuln = package['vulnerabilities'][0]
            affected = vuln['affected'][0]
            if mutation == 'affected':
                vuln['affected'] = []
            elif mutation == 'package':
                affected['package']['name'] = 'unrelated'
            elif mutation == 'ranges':
                affected['ranges'] = None
            elif mutation == 'events':
                affected['ranges'][0]['events'] = [
                    {'introduced': '0', 'fixed': '2'}
                ]
            elif mutation == 'introduced':
                affected['ranges'][0]['events'] = [{'fixed': '2'}]
            elif mutation == 'aliases':
                vuln['aliases'] = 'CVE-2026-1234'
            elif mutation == 'version':
                package['package']['version'] = ''
            else:
                data['experimental_generic_findings'] = [{}]
            with self.subTest(mutation=mutation), self.assertRaises(ValueError):
                osv_reports.findings(data)

    def test_unresolved_source_entries_are_explicit_and_not_audited(self):
        # Hosted release-2.17.3 source output includes these unpinned declarations
        # and local Go replacements, without vulnerability findings.
        cases = [
            ('PyPI', name)
            for name in (
                'future',
                'future',
                'fire',
                'setuptools',
                'setuptools',
                'pytest',
                'pytest-lazy-fixture',
                'requests',
                'requests',
                'absl-py',
                'cloudpickle',
                'deprecated',
                'fire',
                'jsonschema',
                'strip-hints',
                'uritemplate',
                'typer',
            )
        ]
        cases += [
            ('Go', name)
            for name in ('./api', './kubernetes_platform', '../api')
        ]
        for ecosystem, name in cases:
            data = report(name, ecosystem)
            item = data['results'][0]['packages'][0]
            item['package']['version'] = ''
            with self.assertRaisesRegex(ValueError, 'unresolved dependency'):
                osv_reports.findings(data, allow_unresolved=True)
            del item['vulnerabilities']
            with self.assertRaises(ValueError):
                osv_reports.findings(data)
            self.assertEqual(
                osv_reports.findings(data, allow_unresolved=True), []
            )
            self.assertEqual(
                osv_reports.inventory(data, allow_unresolved=True),
                [('requirements.txt', ecosystem, name, '')],
            )
            self.assertEqual(
                osv_reports.unresolved(data),
                [
                    {
                        'target': 'requirements.txt',
                        'ecosystem': ecosystem,
                        'package': name,
                        'installed_version': '',
                    }
                ],
            )
            data['results'][0]['source']['type'] = 'artifact'
            with self.assertRaises(ValueError):
                osv_reports.inventory(data, allow_unresolved=True)

    def test_empty_targets_fail(self):
        for data in (
            {},
            {'results': []},
            {
                'results': [
                    {
                        'source': {'path': 'x', 'type': 'lockfile'},
                        'packages': [],
                    }
                ]
            },
        ):
            with self.assertRaises(ValueError):
                osv_reports.inventory(data)


class AliasPolicyTest(unittest.TestCase):
    def test_legacy_exception_matches_alias_only_at_exact_tuple(self):
        from .security_policy_2173 import APPROVED_2173

        row = next(row for row in APPROVED_2173 if row[0] == 'source')
        scope, target, package, identifier, installed = row[:5]
        finding = {
            'id': 'OSV-new-advisory',
            'aliases': [identifier],
            'target': target,
            'package': package,
            'installed_version': installed,
            'fixed_versions': ['fixed'],
            'fixable': True,
        }

        def evaluate(item):
            return evaluate_policy(
                [{'name': 'source', 'findings': [item]}],
                release_version='2.17.3',
                policy='2.17.3-approved-exceptions-v1',
            )

        self.assertEqual(evaluate(finding)['accepted_count'], 1)
        for key in ('target', 'package', 'installed_version'):
            changed = {**finding, key: 'different'}
            self.assertEqual(evaluate(changed)['blocking_count'], 1)
        self.assertEqual(
            evaluate({**finding, 'aliases': []})['blocking_count'], 1
        )

    def test_explicit_alias_approvals_keep_scope_and_expiry(self):
        from datetime import datetime, timezone

        from .security_exceptions_test import NOW, SecurityExceptionsTest

        fixture = SecurityExceptionsTest()
        for scope, artifact in [
            ('source', 'requirements.txt'),
            ('python', '3.13'),
            ('image', 'sha256:' + 'a' * 64),
        ]:
            document = fixture.document(scope, artifact)
            scans = fixture.scan(scope, artifact)
            scans[0]['findings'][0].update(
                id='GHSA-other', aliases=['CVE-test']
            )
            result = evaluate_policy(
                scans,
                policy='approved-exceptions',
                exceptions=document,
                now=NOW,
            )
            self.assertEqual(result['accepted_count'], 1)
            self.assertEqual(
                result['accepted_findings'][0]['expires'], '2026-10-22'
            )
            changed = copy.deepcopy(scans)
            changed[0]['name'] = 'python-0' if scope == 'source' else 'source'
            self.assertEqual(
                evaluate_policy(
                    changed,
                    policy='approved-exceptions',
                    exceptions=document,
                    now=NOW,
                )['blocking_count'],
                1,
            )
            with self.assertRaises(ValueError):
                evaluate_policy(
                    scans,
                    policy='approved-exceptions',
                    exceptions=document,
                    now=datetime(2026, 10, 23, tzinfo=timezone.utc),
                )

    def test_alias_cannot_bypass_zero_fixable(self):
        finding = osv_reports.findings(report())[0]
        self.assertEqual(
            evaluate_policy([{'name': 'source', 'findings': [finding]}])[
                'blocking_count'
            ],
            1,
        )


if __name__ == '__main__':
    unittest.main()
