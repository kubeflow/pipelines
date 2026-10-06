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
"""Missing trusted state must be observed, never inferred from absent runs."""

import copy
import unittest

import verify_legacy_schedules as legacy


class LegacyTests(unittest.TestCase):

    def setUp(self):
        self.case = dict(
            schedule_uid='uid',
            schedule_name='schedule',
            baseline_event_counts={'event': 1})
        self.start = legacy.timestamp('2026-10-05T00:00:00Z')
        self.event = dict(
            involvedObject=dict(
                uid='uid',
                name='schedule',
                namespace=legacy.NAMESPACE,
                kind='ScheduledWorkflow'),
            reason='Failed',
            type='Warning',
            source=dict(component='scheduled-workflow-controller'),
            metadata=dict(uid='event'),
            count=2,
            lastTimestamp='2026-10-05T00:00:01Z',
            message='rpc error: code = FailedPrecondition desc = Recurring run uid has no trusted scheduling state; recreate it through the KFP API'
        )

    def test_requires_exact_fresh_migration_event(self):
        self.assertTrue(legacy.rejected([self.event], self.case, self.start))
        for mutate in (lambda e: e.update(count=1),
                       lambda e: e.update(lastTimestamp='2026-10-04T00:00:00Z'),
                       lambda e: e['involvedObject'].update(uid='other'),
                       lambda e: e['source'].update(component='other'),
                       lambda e: e.update(message='generic FailedPrecondition'),
                       lambda e: e.update(message=e['message'].replace(
                           'run uid', 'run other'))):
            event = copy.deepcopy(self.event)
            mutate(event)
            self.assertFalse(legacy.rejected([event], self.case, self.start))

    def observe(self, get, collect):
        now = [0]

        def sleep(seconds):
            now[0] += seconds

        return legacy.observe(
            None, [self.case],
            self.start,
            10,
            get=get,
            collect=collect,
            clock=lambda: now[0],
            sleep=sleep)

    def test_each_round_gets_fresh_bounded_http_client(self):
        now = [0]
        clients = []
        collected = []

        def factory():
            client = object()
            clients.append(client)
            return client

        def collect(client, *args):
            collected.append(client)
            return []

        def sleep(seconds):
            now[0] += seconds

        legacy.observe(
            None, [self.case],
            self.start,
            10,
            get=lambda *args: [self.event],
            collect=collect,
            clock=lambda: now[0],
            sleep=sleep,
            client_factory=factory)
        self.assertEqual(len(clients), 3)
        self.assertEqual(clients, collected)
        self.assertEqual(len(set(clients)), 3)

    def test_no_event_is_inconclusive(self):
        result = self.observe(lambda *args: [], lambda *args: [])
        self.assertEqual(result['outcome'], 'inconclusive')

    def test_final_collection_catches_late_run_after_rejection(self):
        calls = []

        def collect(*args):
            calls.append(1)
            return ['unexpected'] if len(calls) == 3 else []

        result = self.observe(lambda *args: [self.event], collect)
        self.assertEqual(result['outcome'], 'failed')
        self.assertEqual(len(calls), 3)

    def test_pass_records_only_correlated_identifiers(self):
        result = self.observe(lambda *args: [self.event], lambda *args: [])
        self.assertEqual(result['outcome'], 'passed')
        self.assertEqual(result['schedule_uids'], ['uid'])
        self.assertNotIn('message', str(result))


if __name__ == '__main__':
    unittest.main()
