#!/usr/bin/env python3
# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Regression coverage for migration evidence (including silent skips)."""

import unittest
from pathlib import Path

import yaml
from verify_migrated_specs import verify


class MigratedSpecsTest(unittest.TestCase):
    def report(self, state='passed', label='LegacyCachePVC'):
        return [
            {
                'SpecReports': [
                    {
                        'ContainerHierarchyLabels': [
                            ['ApiServerTests', 'LegacyMigration']
                        ],
                        'LeafNodeLabels': [label],
                        'State': state,
                    }
                ]
            }
        ]

    def test_passed_leaf_and_container_labels(self):
        self.assertEqual(
            verify(self.report(), ['LegacyCachePVC', 'LegacyMigration']), {}
        )

    def test_missing_report_or_filtered_spec_fails(self):
        self.assertEqual(verify([], ['LegacyCachePVC']), {'LegacyCachePVC': []})
        self.assertIn(
            'LegacyCachePVC',
            verify(self.report(label='Other'), ['LegacyCachePVC']),
        )

    def test_pending_skipped_failed_and_invalid_state_fail(self):
        for state in ('pending', 'skipped', 'failed', 'timedout', None):
            with self.subTest(state=state):
                self.assertIn(
                    'LegacyCachePVC',
                    verify(self.report(state), ['LegacyCachePVC']),
                )

    def test_duplicate_label_cannot_hide_a_missing_behavior(self):
        self.assertIn(
            'LegacyCachePVC', verify(self.report() * 2, ['LegacyCachePVC'])
        )

    def test_unlabeled_specs_use_null_label_arrays(self):
        reports = self.report()
        reports[0]['SpecReports'].append(
            {
                'LeafNodeLabels': None,
                'ContainerHierarchyLabels': None,
                'State': 'skipped',
            }
        )
        self.assertEqual(verify(reports, ['LegacyCachePVC']), {})

    def test_unlabeled_nested_containers_use_null_entries(self):
        reports = self.report()
        reports[0]['SpecReports'][0]['ContainerHierarchyLabels'] = [
            None,
            ['ApiServerTests', 'LegacyMigration'],
            None,
        ]
        self.assertEqual(
            verify(reports, ['LegacyCachePVC', 'LegacyMigration']), {}
        )
        reports[0]['SpecReports'][0]['State'] = 'skipped'
        self.assertEqual(
            verify(reports, ['LegacyMigration']),
            {'LegacyMigration': ['skipped']},
        )

    def test_empty_or_duplicate_manifest_fails(self):
        for required in ([], ['LegacyCachePVC', 'LegacyCachePVC']):
            with self.assertRaises(ValueError):
                verify(self.report(), required)


class MigrationReportingTest(unittest.TestCase):
    def test_failed_verification_preserves_reports_but_fails_the_job(self):
        root = Path(__file__).resolve().parents[3]
        action = yaml.safe_load(
            (root / '.github/actions/test-and-report/action.yml').read_text()
        )
        steps = action['runs']['steps']
        verifier = next(
            step for step in steps if step.get('id') == 'verify-migration'
        )
        self.assertTrue(verifier['continue-on-error'])
        reporter = next(step for step in steps if step.get('id') == 'name_gen')
        self.assertLess(steps.index(verifier), steps.index(reporter))
        failure = steps[-1]
        self.assertEqual(failure['run'], 'exit 1')
        self.assertEqual(
            failure['if'],
            "(steps.run-tests.outcome != 'success' || "
            "steps.verify-migration.outcome == 'failure') && !cancelled()",
        )
        for name in (
            'Collect Pod logs in case of Test Failures',
            'Upload JUnit XML on failure',
            'Upload HTML Report',
        ):
            step = next(step for step in steps if step['name'] == name)
            self.assertIn(
                "steps.verify-migration.outcome == 'failure'", step['if']
            )


if __name__ == '__main__':
    unittest.main()
