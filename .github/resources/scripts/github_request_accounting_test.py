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
"""Exercise redacted API accounting without network access."""

import json
import shutil
import subprocess
import unittest
from pathlib import Path

import yaml

ROOT = Path(__file__).resolve().parents[3]
HELPER = Path(__file__).with_name('github_request_accounting.js')


@unittest.skipUnless(shutil.which('node'), 'Node.js is required')
class RequestAccountingTest(unittest.TestCase):
    def exercise(self, script):
        result = subprocess.run(
            ['node', '-e', script, str(HELPER)],
            capture_output=True,
            text=True,
            check=True,
        )
        return json.loads(result.stdout)

    def test_counts_pages_writes_errors_and_redacts_details(self):
        result = self.exercise(r"""
const {withRequestAccounting} = require(process.argv[1]);
let hook;
const summaries = [];
const github = {hook: {wrap: (_, fn) => {hook = fn;}, remove: (_, fn) => {
  if (fn !== hook) throw Error('Wrong hook removed'); hook = undefined;
}}};
const core = {info: line => summaries.push(JSON.parse(line))};
const headers = {'x-ratelimit-limit': '15000', 'x-ratelimit-remaining': '0',
 'x-ratelimit-used': '15000', 'x-ratelimit-reset': '1794536489',
 'x-ratelimit-resource': 'core', authorization: 'secret-token'};
const original = Object.assign(Error('secret-body'), {
 status: 403, response: {status: 403, headers}});
(async () => {
 let preserved = false;
 try { await withRequestAccounting({github, core, scope: 'prepare'}, async () => {
  for (let page = 1; page <= 2; page++) {
   await hook(async () => ({status: 200, headers: {}}), {
    method: 'GET', url: `/repos/secret-owner/secret-repo/actions/runs?token=secret-query&page=${page}`});
  }
  await hook(async () => ({status: 201, headers: {}}), {
   method: 'POST', url: '/repos/secret-owner/secret-repo/statuses/secret-sha', data: 'secret-body'});
  await hook(async () => {throw original;}, {
   method: 'GET', url: '/repos/secret-owner/secret-repo/pulls/23'});
 }); } catch (error) { preserved = error === original; }
 process.stdout.write(JSON.stringify({summaries, preserved, removed: !hook}));
})();
""")
        self.assertTrue(result['preserved'])
        self.assertTrue(result['removed'])
        self.assertEqual(len(result['summaries']), 1)
        summary = result['summaries'][0]
        self.assertEqual(summary['requests'], 4)
        self.assertIn(
            {
                'method': 'GET',
                'category': 'workflow_runs',
                'status': '200',
                'count': 2,
            },
            summary['counts'],
        )
        self.assertEqual(summary['rate_limits']['core']['last']['remaining'], 0)
        self.assertEqual(
            summary['rate_limits']['core']['last']['reset'], 1794536489
        )
        self.assertEqual(summary['rate_limits']['core']['observations'], 1)
        self.assertNotIn('secret', json.dumps(result))

    def test_transport_errors_and_logging_errors_preserve_original_error(self):
        result = self.exercise(r"""
const {withRequestAccounting} = require(process.argv[1]);
let hook;
const summaries = [];
const github = {hook: {wrap: (_, fn) => {hook = fn;}, remove: () => {}}};
const original = Object.assign(Error('transport detail must not be logged'), {status: 500});
(async () => {
 let preserved = false;
 try { await withRequestAccounting({github, scope: 'finalize', core: {
  info: line => {summaries.push(JSON.parse(line)); throw Error('logger failed');}
 }}, () => hook(async () => {throw original;}, {
  method: 'SECRET', url: '/unknown/secret?body=private'}));
 } catch (error) { preserved = error === original; }
 process.stdout.write(JSON.stringify({preserved, summaries}));
})();
""")
        self.assertTrue(result['preserved'])
        self.assertEqual(
            result['summaries'][0]['counts'],
            [
                {
                    'method': 'OTHER',
                    'category': 'other',
                    'status': 'transport_error',
                    'count': 1,
                }
            ],
        )

    def test_empty_unsupported_and_malformed_headers_are_explicit(self):
        result = self.exercise(r"""
const {withRequestAccounting} = require(process.argv[1]);
const summaries = [];
let hook;
const core = {info: line => summaries.push(JSON.parse(line))};
(async () => {
 const value = await withRequestAccounting({github: {}, core, scope: 'untrusted scope'}, () => 42);
 await withRequestAccounting({github: {hook: {wrap: (_, fn) => {hook = fn;}, remove: () => {}}},
  core, scope: 'recovery'}, async () => {
  await hook(async () => ({status: 200, headers: {
   'x-ratelimit-limit': 'secret', 'x-ratelimit-remaining': '-1',
   'x-ratelimit-reset': '9'.repeat(100), 'x-ratelimit-resource': 'secret-resource'}}),
   {method: 'POST', url: 'https://api.github.com/graphql?secret=true'});
 });
 process.stdout.write(JSON.stringify({value, summaries}));
})();
""")
        self.assertEqual(result['value'], 42)
        self.assertFalse(result['summaries'][0]['hook_available'])
        self.assertEqual(result['summaries'][0]['scope'], 'other')
        self.assertEqual(result['summaries'][1]['rate_limits'], {})
        self.assertNotIn('secret', json.dumps(result))

    def test_every_gate_script_is_wrapped(self):
        for filename in ('ci-checks.yml', 'ci-passed-stale-revoke.yml'):
            workflow = yaml.safe_load(
                (ROOT / '.github/workflows' / filename).read_text()
            )
            for job in workflow['jobs'].values():
                for step in job['steps']:
                    script = step.get('with', {}).get('script', '')
                    if 'ci_passed.js' in script:
                        with self.subTest(
                            workflow=filename, script=script[:80]
                        ):
                            self.assertIn('withRequestAccounting', script)
                            self.assertIn(
                                'return await withRequestAccounting', script
                            )


if __name__ == '__main__':
    unittest.main()
