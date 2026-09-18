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
import contextlib
import io
import json
from pathlib import Path
import unittest
from unittest import mock

from kfp_http import CollectionError
import kfp_inventory
import readiness


class InventoryTest(unittest.TestCase):

    def test_cli_replaces_manual_records_on_collection_failure(self):
        examples = Path(__file__).parent / 'examples'
        client = mock.Mock()
        client.get.side_effect = CollectionError('access_denied')
        with mock.patch.object(
                readiness, 'Client',
                return_value=client), contextlib.redirect_stdout(
                    io.StringIO()) as output:
            code = readiness.main([
                '--inventory',
                str(examples / 'schedules.json'), '--schedule-policy',
                str(examples / 'target-policy.json'), '--include-schedules',
                '--system-namespace', 'kubeflow', '--namespace', 'team-a',
                '--source-version', '2.17.2', '--kfp-endpoint',
                'https://kfp.example', '--format', 'json'
            ])
        self.assertEqual(code, 2)
        report = json.loads(output.getvalue())
        self.assertEqual(report['source']['kfp_collection']['failed_checks'], 2)
        self.assertFalse(
            any(f['status'] == 'policy_rejection' for f in report['findings']))
        self.assertTrue(
            any('access_denied' in f['evidence'] for f in report['findings']))

    def test_combined_record_budget_keeps_partial_evidence(self):
        client = mock.Mock()
        client.get.side_effect = [{
            'recurringRuns': [{
                'recurring_run_id': 'one',
                'experiment_id': 'exp',
                'service_account': 'custom'
            }, {
                'recurring_run_id': 'two',
                'experiment_id': 'exp',
                'service_account': 'custom'
            }]
        }, {
            'experiment_id': 'exp',
            'namespace': 'team'
        }]
        records, failures, _ = kfp_inventory.collect(
            client, ['team'], record_budget=2)
        self.assertEqual(len(records['recurring_runs']), 1)
        self.assertEqual(len(records['experiments']), 1)
        self.assertEqual(failures[0]['reason'], 'record_limit')

    def test_paging_and_deduplicated_experiments(self):
        client = mock.Mock()
        client.get.side_effect = [{
            'recurringRuns': [{
                'recurring_run_id': 'one',
                'experiment_id': 'exp',
                'service_account': 'custom'
            }],
            'next_page_token': 'next'
        }, {
            'experiment_id': 'exp',
            'namespace': 'team'
        }, {
            'recurringRuns': [{
                'recurringRunId': 'two',
                'experimentId': 'exp',
                'serviceAccount': 'custom'
            }]
        }]
        records, failures, coverage = kfp_inventory.collect(client, ['team'])
        self.assertEqual(len(records['recurring_runs']), 2)
        self.assertEqual(len(records['experiments']), 1)
        self.assertEqual(failures, [])
        self.assertEqual(coverage['list_completed_namespaces'], ['team'])
        self.assertEqual(client.get.call_args_list[-1].args[1]['page_token'],
                         'next')

    def test_pinned_v2_default_does_not_copy_spec(self):
        client = mock.Mock()
        client.get.side_effect = [{
            'recurringRuns': [{
                'recurring_run_id': 'one',
                'experiment_id': 'exp',
                'pipeline_version_reference': {
                    'pipeline_id': 'p',
                    'pipeline_version_id': 'v'
                },
                'runtime_config': {
                    'secret': 'do-not-copy'
                }
            }]
        }, {
            'experiment_id': 'exp',
            'namespace': 'team'
        }, {
            'pipeline_id': 'p',
            'pipeline_version_id': 'v',
            'pipeline_spec': {
                'pipelineInfo': {
                    'name': 'test'
                },
                'root': {},
                'secret': 'do-not-copy'
            }
        }]
        records, failures, _ = kfp_inventory.collect(client, ['team'])
        self.assertTrue(records['recurring_runs'][0]['_readiness_v2_default'])
        self.assertNotIn('do-not-copy', str(records))
        self.assertEqual(failures, [])

    def test_latest_and_missing_template_stay_unknown(self):
        for extra in ({'pipeline_version_reference': {'pipeline_id': 'p'}}, {}):
            client = mock.Mock()
            client.get.side_effect = [{
                'recurringRuns': [
                    dict(recurring_run_id='one', experiment_id='exp', **extra)
                ]
            }, {
                'experiment_id': 'exp',
                'namespace': 'team'
            }]
            records, failures, _ = kfp_inventory.collect(client, ['team'])
            self.assertNotIn('_readiness_v2_default',
                             records['recurring_runs'][0])
            self.assertEqual(len(failures), 1)

    def test_failure_retains_partial_inventory_and_scope(self):
        client = mock.Mock()
        client.get.side_effect = [{
            'recurringRuns': [{
                'recurring_run_id': 'one',
                'experiment_id': 'exp',
                'service_account': 'custom'
            }],
            'next_page_token': 'next'
        }, {
            'experiment_id': 'exp',
            'namespace': 'team'
        },
                                  CollectionError('request_failed')]
        records, failures, coverage = kfp_inventory.collect(client, ['team'])
        self.assertEqual(len(records['recurring_runs']), 1)
        self.assertEqual(failures[0]['reason'], 'request_failed')
        self.assertEqual(coverage['list_completed_namespaces'], [])

    def test_scope_mismatch_and_repeated_tokens(self):
        client = mock.Mock()
        client.get.return_value = {'recurringRuns': [], 'nextPageToken': 'same'}
        _, failures, _ = kfp_inventory.collect(client, ['team'])
        self.assertEqual(failures[0]['reason'], 'repeated_page_token')
        client.get.return_value = {
            'recurringRuns': [{
                'recurring_run_id': 'id',
                'namespace': 'elsewhere'
            }]
        }
        records, failures, _ = kfp_inventory.collect(client, ['team'])
        self.assertEqual(records['recurring_runs'], [])
        self.assertEqual(failures[0]['reason'], 'namespace_mismatch')

    def test_wrong_experiment_or_version_not_accepted(self):
        client = mock.Mock()
        client.get.side_effect = [{
            'recurringRuns': [{
                'recurring_run_id': 'one',
                'experiment_id': 'exp',
                'service_account': 'custom'
            }]
        }, {
            'experiment_id': 'exp',
            'namespace': 'other'
        }]
        records, failures, _ = kfp_inventory.collect(client, ['team'])
        self.assertEqual(records['experiments'], [])
        self.assertEqual(failures[0]['reason'], 'experiment_scope_mismatch')


if __name__ == '__main__':
    unittest.main()
