# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import copy
from datetime import datetime
from datetime import timezone
import json
import unittest

import source_observation as observer


def evidence():
    return dict(
        schema_version='kfp-source-observation/v1',
        source_version='2.17.2',
        source_revision=observer.SOURCE_REVISION,
        started_at='2026-10-05T10:00:00Z',
        planned_end_at='2026-10-05T10:01:00Z',
        checkpoint_at='2026-10-05T10:01:00Z',
        ended_at='2026-10-05T10:01:00Z',
        status='completed',
        limits=dict(duration_seconds=60, max_records=512, max_bytes=1048576),
        health={key: 0 for key in observer.HEALTH},
        operations={key: 0 for key in observer.OPERATIONS},
        supported_operations=list(observer.OPERATIONS),
        unobserved_operations=list(observer.OPERATIONS),
        unsupported_checks=list(observer.UNSUPPORTED),
        records=[])


class SourceObservationTest(unittest.TestCase):

    def assess(self, data):
        return observer.assess(
            data, '2.17.2', now=datetime(2026, 10, 5, 11, tzinfo=timezone.utc))

    def test_quiet_complete_interval_never_passes(self):
        findings, summary = self.assess(evidence())
        self.assertTrue(all(row['status'] == 'unknown' for row in findings))
        self.assertEqual(summary['unobserved_operations'],
                         list(observer.OPERATIONS))
        self.assertEqual(summary['checkpoint_age_seconds'], 3540)
        self.assertIn('frontend_operations', summary['unsupported_checks'])

    def test_omitted_upload_and_authenticated_caller_are_not_correlated(self):
        data = evidence()
        data['records'] = [
            dict(
                operation='upload_pipeline',
                observed_at='2026-10-05T10:00:10Z',
                namespace_omitted=True),
            dict(
                operation='authorization',
                observed_at='2026-10-05T10:00:11Z',
                caller='private-user@example.test',
                namespace='private-namespace',
                result='allowed')
        ]
        for operation in ('upload_pipeline', 'authorization'):
            data['operations'][operation] = 1
            data['unobserved_operations'].remove(operation)
        original = copy.deepcopy(data)
        findings, summary = self.assess(data)
        self.assertEqual(data, original)
        self.assertEqual(summary['authenticated_callers_observed'], 1)
        self.assertEqual(summary['omitted_namespace_uploads_retained'], 1)
        self.assertTrue(
            any(row['rule'] == 'observation.uploadNamespace'
                for row in findings))
        for secret in ('private-user', 'private-namespace'):
            self.assertNotIn(secret, json.dumps([findings, summary]))

    def test_partial_and_unhealthy_evidence_retains_counters(self):
        for status in ('active', 'interrupted', 'completed'):
            data = evidence()
            data['status'] = status
            if status == 'active':
                del data['ended_at']
            data['health']['dropped_limit'] = 5
            data['health']['write_failures'] = 2
            findings, summary = self.assess(data)
            self.assertEqual(summary['health']['dropped_limit'], 5)
            self.assertEqual(summary['status'], status)
            self.assertEqual(findings[0]['status'], 'unknown')

    def test_delayed_checkpoint_cannot_extend_observation_duration(self):
        data = evidence()
        data.update(
            checkpoint_at='2026-10-05T10:30:00Z',
            ended_at='2026-10-05T10:30:00Z')
        data['operations']['create_run'] = 1
        data['unobserved_operations'].remove('create_run')
        data['records'] = [
            dict(operation='create_run', observed_at='2026-10-05T10:20:00Z')
        ]
        with self.assertRaisesRegex(ValueError, 'outside interval'):
            self.assess(data)

    def test_invalid_revision_interval_and_counter_claims_rejected(self):
        mutations = [
            lambda d: d.update(source_revision='a' * 40),
            lambda d: d.update(planned_end_at='2026-10-05T10:02:00Z'),
            lambda d: d.update(checkpoint_at='2026-10-06T10:02:00Z'),
            lambda d: d.update(started_at='2026-10-05T10:00:00'),
            lambda d: d['health'].update(dropped_limit=-1),
            lambda d: d['operations'].update(upload_pipeline=True), lambda d: d[
                'unsupported_checks'].remove('target_policy_evaluation'),
            lambda d: d.update(unobserved_operations=[]),
            lambda d: d.update(records=[
                dict(
                    operation='upload_pipeline',
                    observed_at='2026-10-05T10:00:10Z')
            ]), lambda d: d.update(records=[
                dict(
                    operation='authorization',
                    observed_at='2026-10-05T10:00:10Z',
                    token='never')
            ]), lambda d: d.update(records=[{}] * 513)
        ]
        for mutation in mutations:
            data = evidence()
            mutation(data)
            with self.subTest(data=data), self.assertRaises(ValueError):
                self.assess(data)
        with self.assertRaises(ValueError):
            observer.assess(evidence(), '2.17.1')


if __name__ == '__main__':
    unittest.main()
