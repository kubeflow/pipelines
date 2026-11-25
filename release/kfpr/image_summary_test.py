"""Tests for image summaries that leave release evidence untouched."""

import copy
import unittest

from kfpr.image_summary import summarize
from kfpr.osv_image_scan_test import sarif


def finding(advisory='CVE-test', score=None):
    report = sarif(True)
    rule = report['runs'][0]['tool']['driver']['rules'][0]
    rule['id'] = advisory
    if score is not None:
        rule['properties'] = {'security-severity': score}
    report['runs'][0]['results'][0]['ruleId'] = advisory
    return report


def summary(reports, **kwargs):
    defaults = {
        'identity': 'release-2.18-image-kfp-api-server',
        'image': 'registry.example/api:v1@sha256:abc',
        'evidence_url': 'https://github.com/kubeflow/pipelines/actions/runs/123',
        'platforms': ['linux/amd64'],
    }
    return summarize(reports, **(defaults | kwargs))


class ImageSummaryTest(unittest.TestCase):
    def test_multiple_architectures_produce_one_result_without_mutating_raw(
        self,
    ):
        first, second = finding(score='7.1'), finding(score='9.8')
        first['runs'][0]['results'] *= 2
        original = copy.deepcopy([first, second])
        result = summary(
            [first, second], platforms=['linux/amd64', 'linux/arm64']
        )
        self.assertEqual([first, second], original)
        run = result['runs'][0]
        self.assertEqual(len(run['results']), 1)
        properties = run['results'][0]['properties']
        self.assertEqual(properties['advisoryCount'], 1)
        self.assertEqual(properties['rawResultCount'], 3)
        self.assertEqual(
            run['tool']['driver']['rules'][0]['properties'][
                'security-severity'
            ],
            '9.8',
        )

    def test_result_score_does_not_lower_reported_rule_severity(self):
        raw = finding('CVE-TEST', '9.8')
        raw['runs'][0]['results'][0]['properties'] = {'security-severity': '2'}
        rule = summary([raw])['runs'][0]['tool']['driver']['rules'][0]
        self.assertEqual(rule['properties']['security-severity'], '9.8')

    def test_friendly_title_and_clickable_evidence(self):
        run = summary([finding()])['runs'][0]
        rule = run['tool']['driver']['rules'][0]
        self.assertEqual(
            rule['shortDescription']['text'],
            'Vulnerabilities in registry.example/api:v1@sha256:abc',
        )
        self.assertIn(
            '(<https://github.com/kubeflow/pipelines/actions/runs/123>)',
            rule['help']['markdown'],
        )
        long_name = summary([finding()], image='x' * 2000)['runs'][0]
        self.assertEqual(
            len(
                long_name['tool']['driver']['rules'][0]['shortDescription'][
                    'text'
                ]
            ),
            1024,
        )
        unusual_url = summary(
            [finding()], evidence_url='https://example.com/path(with) spaces'
        )['runs'][0]['tool']['driver']['rules'][0]['help']['markdown']
        self.assertIn(
            '(<https://example.com/path%28with%29%20spaces>)', unusual_url
        )

    def test_clean_scan_has_no_result_and_same_rule(self):
        clean = summary([sarif()])['runs'][0]
        dirty = summary([finding()])['runs'][0]
        self.assertEqual(clean['results'], [])
        self.assertEqual(
            clean['tool']['driver']['rules'][0]['id'],
            dirty['results'][0]['ruleId'],
        )

    def test_identity_survives_image_and_advisory_changes(self):
        old = summary([finding()])['runs'][0]['results'][0]
        new = summary([finding('OTHER', '5')], image='registry.example/api:v2')[
            'runs'
        ][0]['results'][0]
        for key in ('ruleId', 'locations', 'partialFingerprints'):
            self.assertEqual(old[key], new[key])
        other = summary([finding()], identity='other-image')['runs'][0][
            'results'
        ][0]
        self.assertNotEqual(old['ruleId'], other['ruleId'])

    def test_missing_or_invalid_scores_are_explicitly_unknown(self):
        for score in (None, 'NaN', 'infinity', 'bad', -1, 11, True):
            with self.subTest(score=score):
                run = summary([finding(score=score)])['runs'][0]
                self.assertNotIn(
                    'security-severity',
                    run['tool']['driver']['rules'][0]['properties'],
                )
                self.assertEqual(
                    run['results'][0]['properties']['unknownSeverityCount'], 1
                )
                self.assertIn(
                    'unknown does not mean low risk',
                    run['results'][0]['message']['text'],
                )

    def test_mixed_known_and_unknown_retains_highest_known(self):
        run = summary(
            [finding(score='9.4'), finding('OTHER')],
            platforms=['linux/amd64', 'linux/arm64'],
        )['runs'][0]
        self.assertEqual(
            run['tool']['driver']['rules'][0]['properties'][
                'security-severity'
            ],
            '9.4',
        )
        self.assertEqual(
            run['results'][0]['properties']['unknownSeverityCount'], 1
        )
        self.assertEqual(run['results'][0]['properties']['advisoryCount'], 2)

    def test_incomplete_or_invalid_reports_cannot_clear_alert(self):
        for bad in ({}, {'version': '2.1.0', 'runs': []}):
            with self.subTest(bad=bad), self.assertRaises(ValueError):
                summary(
                    [sarif(), bad], platforms=['linux/amd64', 'linux/arm64']
                )
        incomplete = sarif()
        del incomplete['runs'][0]['results']
        with self.assertRaises(ValueError):
            summary([incomplete])
        with self.assertRaises(ValueError):
            summary([])
        with self.assertRaises(ValueError):
            summary([sarif(), sarif()])
        with self.assertRaises(ValueError):
            summary(
                [sarif(), sarif()], platforms=['linux/amd64', 'linux/amd64']
            )

    def test_failed_invocation_and_error_notifications_are_rejected(self):
        for invocation in (
            {'executionSuccessful': False},
            {'toolExecutionNotifications': [{'level': 'error'}]},
            {'toolConfigurationNotifications': [{'level': 'error'}]},
        ):
            report = sarif()
            report['runs'][0]['invocations'] = [invocation]
            with (
                self.subTest(invocation=invocation),
                self.assertRaises(ValueError),
            ):
                summary([report])

    def test_unsafe_evidence_url_rejected(self):
        with self.assertRaises(ValueError):
            summary([sarif()], evidence_url='javascript:alert(1)')


if __name__ == '__main__':
    unittest.main()
