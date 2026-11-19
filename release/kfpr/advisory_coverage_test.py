"""Coverage of release-scoped, public security advisories in release notes."""

import json
import subprocess
import unittest
from unittest.mock import Mock

from kfpr.advisory_coverage import (
    ReleaseAdvisory,
    fetch_release_advisories,
    validate_advisory_coverage,
)


def affected(name, patched, ecosystem='pip'):
    return {
        'package': {'name': name, 'ecosystem': ecosystem},
        'patched_versions': patched,
    }


def advisory(ghsa_id, packages, **overrides):
    return {
        'ghsa_id': ghsa_id,
        'summary': 'Public advisory title',
        'state': 'published',
        'published_at': '2026-11-18T17:33:20Z',
        'vulnerabilities': packages,
        **overrides,
    }


class ReleaseAdvisoryTest(unittest.TestCase):
    sdk_id = 'GHSA-qv29-qxgh-jcw8'
    backend_id = 'GHSA-j53r-x5f5-j4hr'

    def test_matches_per_package_across_pages(self):
        sdk = advisory(
            self.sdk_id,
            [
                affected('KFP_Server.API', 'v2.17.5, >= 2.18.1'),
                affected('kfp-backend', '2.18.10', 'other'),
            ],
        )
        backend = advisory(
            self.backend_id,
            [
                affected('kfp-backend', '2.18.1', 'other'),
                affected('kfp', '>= 2.17.5, < 2.18.1'),
                affected('kfp', '2.18.1-rc.1'),
            ],
        )
        runner = Mock()
        runner.capture.return_value = json.dumps([[sdk], [backend]])
        rows = fetch_release_advisories(runner, '2.18.1')
        self.assertEqual(
            [(r.ghsa_id, r.products) for r in rows],
            [(self.backend_id, {'backend'}), (self.sdk_id, {'sdk'})],
        )
        self.assertIn('--paginate', runner.capture.call_args.args[0])
        self.assertIn('state=published', runner.capture.call_args.args[0][-1])

    def test_private_withdrawn_and_unpatched_are_excluded(self):
        packages = [affected('kfp', '2.18.1')]
        rows = [
            advisory(self.sdk_id, packages, state='draft'),
            advisory(self.sdk_id, packages, published_at=None),
            advisory(self.sdk_id, packages, withdrawn_at='2026-11-20'),
            advisory(self.sdk_id, [affected('kfp', None)]),
        ]
        runner = Mock()
        runner.capture.return_value = json.dumps([rows])
        self.assertEqual(fetch_release_advisories(runner, '2.18.1'), [])

    def test_empty_feed_and_fetch_failure_are_distinct(self):
        runner = Mock()
        runner.capture.return_value = '[[]]'
        self.assertEqual(fetch_release_advisories(runner, '2.18.1'), [])
        runner.capture.side_effect = subprocess.CalledProcessError(1, 'gh api')
        with self.assertRaises(subprocess.CalledProcessError):
            fetch_release_advisories(runner, '2.18.1')

    def test_backend_covers_release_sdk_covers_python_packages(self):
        sdk = ReleaseAdvisory(self.sdk_id, 'SDK', frozenset({'sdk'}))
        backend = ReleaseAdvisory(
            self.backend_id, 'Server', frozenset({'backend'})
        )
        docs = {
            'backend': f'[SDK fix]({sdk.url}) and <{backend.url}>',
            'sdk': f'Fixed in this patch. {sdk.url}.',
        }
        validate_advisory_coverage([sdk, backend], docs)
        validate_advisory_coverage([sdk, backend], {'sdk': docs['sdk']})
        docs['backend'] = backend.url
        with self.assertRaisesRegex(ValueError, f'backend: {sdk.url}'):
            validate_advisory_coverage([sdk, backend], docs)

    def test_reference_link_and_autolink_point_to_published_advisory(self):
        sdk = ReleaseAdvisory(self.sdk_id, 'SDK', frozenset({'sdk'}))
        for text in (
            f'[Read about the SDK fix][sdk]\n\n[sdk]: {sdk.url}',
            f'Visit <{sdk.url}>.',
            f'Full advisory: {sdk.url}.',
            f'Impact: {sdk.url}#impact',
            f'Fixed ({sdk.url}), see the next advisory.',
        ):
            with self.subTest(text=text):
                validate_advisory_coverage([sdk], {'sdk': text})

    def test_hidden_links_and_other_destinations_do_not_count(self):
        sdk = ReleaseAdvisory(self.sdk_id, 'SDK', frozenset({'sdk'}))
        for text in (
            f'![image]({sdk.url})',
            f'[More information {sdk.url} here](https://example.org/)',
            f'[unused]: {sdk.url}',
            f'    {sdk.url}',
            f'<!-- {sdk.url} -->',
            f'```md\n{sdk.url}\n```',
            f'`{sdk.url}`',
            f'{sdk.url},extra',
            f'{sdk.url};another',
            f'{sdk.url})suffix',
            f'[fixed]({sdk.url}-unrelated)',
            f'[fixed](https://example.org/?url={sdk.url})',
            self.sdk_id,
        ):
            with (
                self.subTest(text=text),
                self.assertRaisesRegex(ValueError, self.sdk_id),
            ):
                validate_advisory_coverage([sdk], {'sdk': text})


if __name__ == '__main__':
    unittest.main()
