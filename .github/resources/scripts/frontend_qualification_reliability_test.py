#!/usr/bin/env python3
# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
import json
from pathlib import Path
import tempfile
import unittest
from unittest import mock

from frontend_qualification_reliability import collect
from frontend_qualification_reliability import report


class ReliabilityTest(unittest.TestCase):

    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.directory = Path(self.temp.name)
        self.workflow = 'frontend-browser-qualification.yml'
        self.run = dict(
            id=12,
            path=f'.github/workflows/{self.workflow}',
            head_sha='abc',
            html_url='https://github.com/run/12',
            run_attempt=1,
            conclusion='cancelled')

    def write(self, attempt, conclusion, **overrides):
        job = dict(
            run_id=12,
            run_attempt=attempt,
            head_sha='abc',
            name='Safari (iPad)',
            conclusion=conclusion,
            html_url=f'https://github.com/job/{attempt}')
        job.update(overrides)
        (self.directory / f'12-attempt-{attempt}.json').write_text(
            json.dumps([{
                'jobs': [job]
            }]))

    def result(self):
        (self.directory / 'runs.json').write_text(
            json.dumps({'workflow_runs': [self.run]}))
        return report(self.directory, [self.workflow])

    def test_failure_inside_cancelled_run_counts(self):
        self.write(1, 'failure')
        result = self.result()
        self.assertEqual(result['lanes'][0]['failure'], 1)
        self.assertEqual(result['evidence'][0]['cause'], 'unclassified')

    def test_stale_job_counts_as_terminal_failure(self):
        self.write(1, 'stale')
        result = self.result()
        self.assertEqual(result['lanes'][0]['failure'], 1)
        self.assertEqual(result['lanes'][0]['unknown'], 0)

    def test_rerun_success_preserves_first_failure(self):
        self.run.update(run_attempt=2, conclusion='success')
        self.write(1, 'failure')
        self.write(2, 'success')
        result = self.result()
        self.assertEqual(result['lanes'][0]['failure'], 1)
        self.assertEqual(result['lanes'][0]['success'], 0)
        self.assertEqual(result['lanes'][0]['same_sha_recoveries'], 1)
        self.assertEqual(
            result['evidence'][0]['successful_reruns'][0]['attempt'], 2)

    def test_only_latest_attempt_is_unknown(self):
        self.run['run_attempt'] = 2
        self.write(2, 'success')
        result = self.result()
        self.assertEqual(result['lanes'][0]['unknown'], 1)
        self.assertEqual(result['lanes'][0]['same_sha_recoveries'], 0)
        self.assertEqual(result['incomplete_attempts'][0]['attempt'], 1)

    def test_missing_jobs_does_not_mean_success(self):
        result = self.result()
        self.assertEqual(result['lanes'], [])
        self.assertEqual(len(result['incomplete_attempts']), 1)
        self.assertEqual(result['missing_workflows'], [self.workflow])

    def test_cancelled_skipped_and_running_jobs_are_unknown(self):
        for conclusion in ['cancelled', 'skipped', 'action_required', None]:
            with self.subTest(conclusion=conclusion):
                self.write(1, conclusion)
                self.assertEqual(self.result()['lanes'][0]['unknown'], 1)

    def test_changed_sha_cannot_establish_recovery(self):
        self.run['run_attempt'] = 2
        self.write(1, 'failure')
        self.write(2, 'success', head_sha='different')
        with self.assertRaisesRegex(ValueError, 'identity mismatch'):
            self.result()

    def test_missing_later_attempt_keeps_known_first_outcome(self):
        self.run['run_attempt'] = 2
        self.write(1, 'success')
        result = self.result()
        self.assertEqual(result['lanes'][0]['success'], 1)
        self.assertEqual(result['incomplete_attempts'][0]['attempt'], 2)

    def test_paginated_jobs_are_not_dropped(self):
        self.write(1, 'success')
        path = self.directory / '12-attempt-1.json'
        pages = json.loads(path.read_text())
        second = dict(pages[0]['jobs'][0], name='Firefox')
        pages.append({'jobs': [second]})
        path.write_text(json.dumps(pages))
        self.assertEqual(len(self.result()['lanes']), 2)

    def test_duplicate_job_names_are_rejected(self):
        self.write(1, 'success')
        path = self.directory / '12-attempt-1.json'
        pages = json.loads(path.read_text())
        pages.append(pages[0])
        path.write_text(json.dumps(pages))
        with self.assertRaisesRegex(ValueError, 'duplicate job name'):
            self.result()


class CollectionTest(unittest.TestCase):

    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.directory = Path(self.temp.name)
        self.workflow = 'frontend-browser-qualification.yml'
        self.run = dict(
            id=12,
            path=f'.github/workflows/{self.workflow}',
            status='completed',
            head_sha='abc',
            run_attempt=2,
            conclusion='success',
            html_url='https://github.com/run/12')

    def api(self, endpoint, fields=None):
        if endpoint.endswith('/runs'):
            return dict(total_count=1, workflow_runs=[self.run])
        if endpoint.endswith('/jobs'):
            attempt = int(endpoint.split('/')[-2])
            return dict(
                total_count=1,
                jobs=[
                    dict(
                        id=attempt,
                        run_id=12,
                        run_attempt=attempt,
                        head_sha='abc',
                        name='Safari',
                        conclusion='failure' if attempt == 1 else 'success',
                        html_url=f'https://github.com/job/{attempt}')
                ])
        return dict(self.run)

    def collect(self):
        return collect(self.directory, [self.workflow], 'kubeflow/pipelines',
                       '2026-10-01', '2026-10-08')

    @mock.patch('frontend_qualification_reliability.api_json')
    def test_collects_every_attempt_and_preserves_first_failure(self, api):
        api.side_effect = self.api
        result = self.collect()
        self.assertEqual(result['lanes'][0]['failure'], 1)
        self.assertEqual(result['lanes'][0]['same_sha_recoveries'], 1)
        self.assertTrue(
            json.loads(
                (self.directory / 'collection.json').read_text())['complete'])

    @mock.patch('frontend_qualification_reliability.api_json')
    def test_paginated_jobs_remain_complete(self, api):

        def paginated(endpoint, fields=None):
            response = self.api(endpoint, fields)
            if endpoint.endswith('/jobs'):
                response['total_count'] = 2
                if fields['page'] == 2:
                    response['jobs'][0].update(
                        id=100 + response['jobs'][0]['id'], name='Firefox')
            return response

        api.side_effect = paginated
        self.assertEqual(len(self.collect()['lanes']), 2)

    @mock.patch('frontend_qualification_reliability.api_json')
    def test_run_changed_during_collection_is_not_certified(self, api):

        def changed(endpoint, fields=None):
            response = self.api(endpoint, fields)
            if endpoint.endswith('/runs/12'):
                response['run_attempt'] = 3
            return response

        api.side_effect = changed
        with self.assertRaisesRegex(ValueError, 'changed during collection'):
            self.collect()
        with self.assertRaisesRegex(ValueError, 'incomplete'):
            report(self.directory, [self.workflow])

    @mock.patch('frontend_qualification_reliability.api_json')
    def test_run_search_limit_fails_closed(self, api):
        api.return_value = dict(total_count=201, workflow_runs=[])
        with self.assertRaisesRegex(ValueError, 'limit exceeded'):
            self.collect()
        self.assertFalse(
            json.loads(
                (self.directory / 'collection.json').read_text())['complete'])

    @mock.patch('frontend_qualification_reliability.api_json')
    def test_active_runs_are_not_certified(self, api):
        self.run['status'] = 'in_progress'
        api.side_effect = self.api
        with self.assertRaisesRegex(ValueError, 'still active'):
            self.collect()

    @mock.patch('frontend_qualification_reliability.api_json')
    def test_duplicate_pages_fail_closed(self, api):

        def duplicate(endpoint, fields=None):
            response = self.api(endpoint, fields)
            if endpoint.endswith('/jobs'):
                response['total_count'] = 2
            return response

        api.side_effect = duplicate
        with self.assertRaisesRegex(ValueError, 'duplicate result'):
            self.collect()

    def test_nonempty_directory_cannot_mix_old_evidence(self):
        (self.directory / '12-attempt-1.json').write_text('{}')
        with self.assertRaisesRegex(ValueError, 'empty evidence directory'):
            self.collect()

    @mock.patch('frontend_qualification_reliability.api_json')
    def test_source_identity_failure_invalidates_collection(self, api):

        def changed(endpoint, fields=None):
            response = self.api(endpoint, fields)
            if endpoint.endswith('/jobs'):
                response['jobs'][0]['head_sha'] = 'different'
            return response

        api.side_effect = changed
        with self.assertRaisesRegex(ValueError, 'identity mismatch'):
            self.collect()
        self.assertFalse(
            json.loads(
                (self.directory / 'collection.json').read_text())['complete'])


if __name__ == '__main__':
    unittest.main()
