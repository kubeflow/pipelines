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
"""Verify timing correlation and exclusion of raw diagnostic contents."""
import io
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

import filter_server_timings as collector

ID = '10000000-0000-4000-8000-000000000001'
OTHER = '10000000-0000-4000-8000-000000000002'


class FilterServerTimingsTest(unittest.TestCase):

    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name)
        folder = self.root / 'flip_coin.yaml-test'
        folder.mkdir()
        self.log = folder / 'run-0-0-pod-0.log'
        self.log.write_text('prefix KFP_RUNTIME_CLIENT_TIMING ' + json.dumps({
            'correlation_id': ID,
            'method': '/api.RunService/CreateTask',
            'total_ms': 10,
            'status': 'OK'
        }) + '\n')

    def server(self, **overrides):
        record = {
            'correlation_id': ID,
            'method': 'CreateTask',
            'total_ms': 9,
            'phases_ms': {
                'token_review': 5,
                'kubernetes_http_roundtrip': 1
            }
        }
        record.update(overrides)
        return 'glog prefix KFP_RUNTIME_SERVER_TIMING ' + json.dumps(
            record) + '\n'

    def test_only_matching_allowlisted_records_persist(self):
        raw = ('user secret-token unrelated line\n' +
               self.server(extra='secret-token') +
               self.server(correlation_id=OTHER))
        summary = collector.collect(self.root, io.BytesIO(raw.encode()))
        self.assertEqual(summary['matched_server_records'], 1)
        self.assertEqual(summary['missing_server_ids'], [])
        self.assertTrue(summary['complete'])
        self.assertEqual(summary['incomplete_reasons'], [])
        data = (self.root / 'server-timings.jsonl').read_text()
        self.assertNotIn('secret-token', data)
        self.assertNotIn(OTHER, data)
        self.assertEqual(
            set(json.loads(data)),
            {'correlation_id', 'method', 'total_ms', 'phases_ms'})

    def test_bad_shapes_and_nonfinite_timings_rejected(self):
        raw = ''.join([
            self.server(total_ms=float('nan')),
            self.server(total_ms=True),
            self.server(total_ms=-1),
            self.server(method='GetRun'),
            self.server(phases_ms={'token': 'secret'}),
            self.server(phases_ms={'bad\nkey': 1}),
            self.server(phases_ms={'private_identifier': 123}),
            'KFP_RUNTIME_SERVER_TIMING not-json\n'
        ])
        summary = collector.collect(self.root, io.BytesIO(raw.encode()))
        self.assertEqual(summary['matched_server_records'], 0)
        self.assertEqual(summary['rejected_records'], 8)
        self.assertEqual(summary['missing_server_ids'], [ID])

    def test_malformed_client_metadata_cannot_select_server(self):
        self.log.write_text(
            'KFP_RUNTIME_CLIENT_TIMING {"correlation_id":"invalid"}\n')
        summary = collector.collect(self.root,
                                    io.BytesIO(self.server().encode()))
        self.assertEqual(summary['selected_client_ids'], 0)
        self.assertFalse(summary['complete'])
        self.assertIn('no_selected_client_records',
                      summary['incomplete_reasons'])
        self.assertIn('no_matching_server_records',
                      summary['incomplete_reasons'])
        self.assertEqual(summary['matched_server_records'], 0)

    def test_missing_server_records_explicitly_incomplete(self):
        summary = collector.collect(self.root, io.BytesIO(b''))
        self.assertFalse(summary['complete'])
        self.assertEqual(summary['missing_server_ids'], [ID])
        self.assertIn('no_matching_server_records',
                      summary['incomplete_reasons'])
        self.assertIn('missing_server_records', summary['incomplete_reasons'])

    def test_exact_byte_limit_is_conservatively_incomplete(self):
        raw = self.server().encode()
        with patch.object(collector, 'MAX_BYTES', len(raw)):
            summary = collector.collect(self.root, io.BytesIO(raw))
        self.assertEqual(summary['matched_server_records'], 1)
        self.assertTrue(summary['server_input_truncated'])
        self.assertFalse(summary['complete'])
        self.assertIn('server_input_at_byte_limit',
                      summary['incomplete_reasons'])

    def test_bounded_input_reports_truncation(self):
        with patch.object(collector, 'MAX_BYTES', 1024):
            raw = self.server() + ('unrelated secret\n' * 100)
            summary = collector.collect(self.root, io.BytesIO(raw.encode()))
        self.assertTrue(summary['server_input_truncated'])
        self.assertEqual(summary['matched_server_records'], 1)
        self.assertNotIn('secret',
                         (self.root / 'server-timings.jsonl').read_text())


if __name__ == '__main__':
    unittest.main()
