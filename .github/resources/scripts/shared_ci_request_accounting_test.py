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
"""Account for gh invocations without changing shared-image API behavior."""

import io
import json
import unittest
from unittest import mock

import shared_ci_images as shared


def response(status, headers=None, body=b'{}'):
    lines = [f'HTTP/2.0 {status} Result']
    lines.extend(f'{key}: {value}' for key, value in (headers or {}).items())
    return mock.Mock(
        returncode=0 if status < 400 else 1,
        stdout=('\r\n'.join(lines) + '\r\n\r\n').encode() + body,
        stderr=b'private diagnostic',
    )


class SharedRequestAccountingTest(unittest.TestCase):
    def summary(self, accounting):
        output = io.StringIO()
        with mock.patch.object(shared.sys, 'stderr', output):
            accounting.emit('verify')
        return json.loads(output.getvalue())

    def test_counts_paginated_pages_conditional_reads_and_headers(self):
        api = shared.GitHub('private/repository')
        replies = [
            response(
                200,
                {
                    'Link': '<https://api.github.com/repos/private/repository/actions/runs/10/artifacts?page=2>; rel="next"'
                },
                b'{"artifacts":[]}',
            ),
            response(200, body=b'{"artifacts":[]}'),
            response(200, {'ETag': 'private-etag'}, b'{}'),
            response(
                304,
                {
                    'X-RateLimit-Limit': '15000',
                    'X-RateLimit-Remaining': '14995',
                    'X-RateLimit-Used': '5',
                    'X-RateLimit-Reset': '1794536489',
                    'X-RateLimit-Resource': 'core',
                },
            ),
        ]
        with mock.patch.object(
            shared.subprocess, 'run', side_effect=replies
        ) as request:
            self.assertEqual(api.artifacts(10), [])
            self.assertEqual(
                api.conditional_page('actions/runs/10?secret=value'), ({}, None)
            )
            self.assertEqual(
                api.conditional_page('actions/runs/10?secret=value'), ({}, None)
            )
        result = self.summary(api.accounting)
        self.assertEqual(request.call_count, 4)
        self.assertEqual(result['requests'], 4)
        self.assertIn(
            {
                'method': 'GET',
                'category': 'artifacts',
                'status': '200',
                'count': 2,
            },
            result['counts'],
        )
        self.assertIn(
            {
                'method': 'GET',
                'category': 'workflow_runs',
                'status': '304',
                'count': 1,
            },
            result['counts'],
        )
        self.assertEqual(
            result['rate_limits']['core']['last']['remaining'], 14995
        )
        self.assertNotIn('private', json.dumps(result))
        self.assertNotIn('secret', json.dumps(result))

    def test_rate_limit_and_transport_failures_are_counted(self):
        api = shared.GitHub('owner/repo')
        with mock.patch.object(
            shared.subprocess,
            'run',
            return_value=response(
                403,
                {
                    'Retry-After': '9',
                    'X-RateLimit-Remaining': '0',
                    'X-RateLimit-Resource': 'core',
                },
            ),
        ):
            with self.assertRaises(shared.APIError) as raised:
                api.request('actions/runs/10')
            self.assertEqual(raised.exception.retry_after, 9)
        original = OSError('private path')
        with mock.patch.object(shared.subprocess, 'run', side_effect=original):
            with self.assertRaises(shared.APIError) as raised:
                api.request('actions/artifacts/2/zip', binary=True)
            self.assertIs(raised.exception.__cause__, original)
        result = self.summary(api.accounting)
        self.assertEqual(result['requests'], 2)
        self.assertEqual(
            {item['status'] for item in result['counts']},
            {'403', 'transport_error'},
        )
        self.assertEqual(result['rate_limits']['core']['last']['remaining'], 0)

    def test_no_command_after_deadline_and_malformed_response_counted(self):
        api = shared.GitHub('owner/repo')
        api.deadline = -1
        with mock.patch.object(shared.subprocess, 'run') as request:
            with self.assertRaises(shared.APIError):
                api.request('actions/runs/10')
            request.assert_not_called()
        self.assertEqual(self.summary(api.accounting)['requests'], 0)
        api.deadline = None
        with mock.patch.object(
            shared.subprocess,
            'run',
            return_value=mock.Mock(returncode=0, stdout=b'malformed'),
        ):
            with self.assertRaises(shared.InvalidEvidence):
                api.request('actions/runs/10')
        self.assertEqual(self.summary(api.accounting)['requests'], 1)

    def test_cli_error_emits_exactly_one_summary(self):
        output = io.StringIO()
        with (
            mock.patch.object(
                shared.sys,
                'argv',
                ['shared_ci_images.py', 'source', '--output', 'unused'],
            ),
            mock.patch.object(
                shared,
                'environment_identity',
                side_effect=shared.InvalidEvidence('identity failed'),
            ),
            mock.patch.object(shared.sys, 'stderr', output),
        ):
            self.assertEqual(shared.main(), 1)
        lines = output.getvalue().splitlines()
        self.assertEqual(lines[0], '::error::identity failed')
        self.assertEqual(len(lines), 2)
        self.assertEqual(json.loads(lines[1])['requests'], 0)


if __name__ == '__main__':
    unittest.main()
