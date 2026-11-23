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
"""Exercise passive transport accounting against real local HTTP requests."""

import json
import os
import shutil
import subprocess
import unittest
from pathlib import Path

import yaml

HELPER = Path(__file__).with_name('node_http_accounting.cjs')


@unittest.skipUnless(shutil.which('node'), 'Node.js is required')
class TransportAccountingTest(unittest.TestCase):
    def exercise(self, enabled=True):
        script = r"""
const http = require('node:http');
(async () => {
 const server = http.createServer((req, res) => {
  if (req.url.startsWith('/redirect')) {
   res.writeHead(302, {location: '/final?signature=secret'}); res.end();
  } else if (req.url.startsWith('/reset')) { req.socket.destroy(); }
  else { res.writeHead(req.url.startsWith('/retry') ? 503 : 200,
   {'x-ratelimit-limit':'5000', 'x-ratelimit-remaining':'4999',
    'x-ratelimit-resource':'core', 'authorization':'secret'}); res.end('secret-body'); }
 });
 await new Promise(resolve => server.listen(0, '127.0.0.1', resolve));
 const base = `http://127.0.0.1:${server.address().port}`;
 // Two pages, a redirect, then an explicit retry: all transport attempts count.
 for (const path of ['/page?token=secret&page=1', '/page?page=2', '/redirect', '/retry', '/final']) {
  await (await fetch(base + path)).text();
 }
 await fetch(base + '/reset').catch(() => {});
 await new Promise(resolve => http.get(base + '/native?token=secret', res => {
  res.resume(); res.on('end', resolve);
 }));
 await new Promise(resolve => http.get(base + '/reset').on('error', resolve));
 server.closeAllConnections(); await new Promise(resolve => server.close(resolve));
 console.log('operation-preserved');
})();
"""
        env = {**os.environ, 'CI_HTTP_ACCOUNTING_SCOPE': 'artifact_primary'}
        command = ['node']
        if enabled and HELPER.exists():
            command += ['--require', str(HELPER)]
        result = subprocess.run(
            command + ['-e', script],
            env=env,
            check=False,
            text=True,
            capture_output=True,
            timeout=20,
        )
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn('operation-preserved', result.stdout)
        return result.stdout

    def test_endpoint_categories_cover_gate_reads_and_writes(self):
        paths = {
            '/repos/o/r': 'repository_metadata',
            '/repos/o/r/rules/branches/release-2.18?page=2': 'branch_rules',
            '/repos/o/r/actions/workflows/1/dispatches': 'workflow_dispatches',
            '/repos/o/r/actions/workflows/1': 'workflow_definitions',
            '/repos/o/r/actions/workflows/1/runs': 'workflow_runs',
            '/repos/o/r/actions/runs/1/attempts/2/jobs': 'run_jobs',
            '/repos/o/r/actions/runs/1/artifacts': 'artifacts',
            '/repos/o/r/actions/artifacts/1/zip': 'artifacts',
            '/repos/o/r/statuses/sha': 'commit_statuses',
            '/repos/o/r/commits/sha/status': 'commit_statuses',
            '/repos/o/r/commits/sha/statuses': 'commit_statuses',
            '/repos/o/r/issues/1/labels': 'issue_labels',
            '/repos/o/r/pulls/1/files': 'pull_files',
        }
        script = r"""
const dc = require('node:diagnostics_channel');
for (const path of JSON.parse(process.argv[1])) {
 const request = {origin:'https://api.github.com', path, method:path.endsWith('/dispatches') ? 'POST' : 'GET'};
 dc.channel('undici:request:create').publish({request});
 dc.channel('undici:request:headers').publish({request, response:{statusCode:200, headers:[]}});
}
"""
        result = subprocess.run(
            [
                'node',
                '--require',
                str(HELPER),
                '-e',
                script,
                json.dumps(list(paths)),
            ],
            env={**os.environ, 'CI_HTTP_ACCOUNTING_SCOPE': 'prepare'},
            capture_output=True,
            text=True,
            check=True,
        )
        summary = json.loads(result.stdout)
        expected = {
            category: list(paths.values()).count(category)
            for category in paths.values()
        }
        self.assertEqual(
            {row['category']: row['count'] for row in summary['counts']},
            expected,
        )

    def test_unsupported_runtime_is_not_reported_as_measured_zero(self):
        script = "Object.defineProperty(process.versions, 'node', {value:'22.0.0'}); require(process.argv[1]);"
        result = subprocess.run(
            ['node', '-e', script, str(HELPER)],
            env={**os.environ, 'CI_HTTP_ACCOUNTING_SCOPE': 'all_checks'},
            capture_output=True,
            text=True,
            check=True,
        )
        summary = json.loads(result.stdout)
        self.assertFalse(summary['transport_supported'])

    def test_workflows_wire_all_external_and_logical_clients(self):
        root = HELPER.parents[3]
        covered = set()
        for name in [
            'ci-checks.yml',
            'ci-passed-stale-revoke.yml',
            'merge-queue-recovery.yml',
        ]:
            workflow = yaml.safe_load(
                (root / '.github/workflows' / name).read_text()
            )
            for job in workflow['jobs'].values():
                for step in job.get('steps', []):
                    script = step.get('with', {}).get('script', '')
                    if (
                        'withRequestAccounting' in script
                        or 'ci_merge_group.js' in script
                        or 'merge_queue_dispatch' in script
                    ):
                        self.assertIn(
                            'node_http_accounting.cjs',
                            step['env']['NODE_OPTIONS'],
                        )
                        covered.add(step['env']['CI_HTTP_ACCOUNTING_SCOPE'])
        self.assertEqual(
            covered,
            {
                'prepare',
                'finalize',
                'recovery',
                'merged_prepare',
                'merged_finalize',
                'revoke',
                'merge_group',
                'queue_admission',
            },
        )
        action = yaml.safe_load(
            (
                root / '.github/actions/download-artifact-with-retry/action.yml'
            ).read_text()
        )
        downloads = [
            step
            for step in action['runs']['steps']
            if step.get('uses', '').startswith('actions/download-artifact@')
        ]
        self.assertEqual(
            [step['env']['CI_HTTP_ACCOUNTING_SCOPE'] for step in downloads],
            ['artifact_primary', 'artifact_retry'],
        )
        for step in downloads:
            self.assertIn(
                'node_http_accounting.cjs', step['env']['NODE_OPTIONS']
            )

    def test_redacted_categories_pending_attempts_and_original_errors(self):
        script = r"""
const dc = require('node:diagnostics_channel');
function event(name, value) { dc.channel(name).publish(value); }
const first = {origin:'https://api.github.com', path:'/repos/secret/repo/commits/sha/check-runs?token=secret', method:'GET'};
event('undici:request:create', {request:first});
event('undici:request:create', {request:first}); // duplicate notification is not another attempt
const headers = [Buffer.from('x-ratelimit-limit'), Buffer.from('5000'),
 Buffer.from('x-ratelimit-resource'), Buffer.from('secret-resource')];
event('undici:request:headers', {request:first, response:{statusCode:200, headers}});
event('undici:request:error', {request:first}); // body failure must not double count HTTP response
const second = {origin:'https://results.example', path:'/twirp/private?token=secret', method:'POST'};
event('undici:request:create', {request:second});
event('undici:request:error', {request:second});
const pending = {origin:'https://storage.example', path:'/secret?sig=secret', method:'GET'};
event('undici:request:create', {request:pending});
event('undici:request:create', {}); // instrumentation errors cannot throw
process.env.CI_HTTP_ACCOUNTING_SCOPE = 'secret-mutated';
process.exit(7);
"""
        env = {
            **os.environ,
            'CI_HTTP_ACCOUNTING_SCOPE': 'all_checks',
            'ACTIONS_RESULTS_URL': 'https://results.example',
        }
        result = subprocess.run(
            ['node', '--require', str(HELPER), '-e', script],
            env=env,
            check=False,
            capture_output=True,
            text=True,
        )
        self.assertEqual(result.returncode, 7)
        summary = json.loads(result.stdout)
        self.assertEqual(summary['requests'], 3)
        self.assertEqual(summary['scope'], 'all_checks')
        self.assertEqual(
            {row['category']: row['status'] for row in summary['counts']},
            {
                'check_runs': '200',
                'artifact_service': 'transport_error',
                'other_http': 'incomplete',
            },
        )
        self.assertEqual(
            summary['rate_limits']['other']['first']['limit'], 5000
        )
        self.assertNotIn('secret', result.stdout)
        self.assertEqual(result.stderr, '')

    def test_real_transports_count_pages_redirects_retries_and_errors(self):
        output = self.exercise()
        records = [
            json.loads(line)
            for line in output.splitlines()
            if line.startswith('{')
        ]
        self.assertEqual(len(records), 1)
        result = records[0]
        self.assertEqual(result['measurement'], 'node_http_attempts')
        self.assertEqual(result['requests'], 9)
        counts = {row['status']: row['count'] for row in result['counts']}
        self.assertEqual(
            counts, {'200': 5, '302': 1, '503': 1, 'transport_error': 2}
        )
        self.assertEqual(result['rate_limits'], {})
        self.assertNotIn('secret', output)
        self.assertNotIn('127.0.0.1', output)
        self.assertNotIn('signature', output)
        self.assertEqual(self.exercise(False), 'operation-preserved\n')


if __name__ == '__main__':
    unittest.main()
