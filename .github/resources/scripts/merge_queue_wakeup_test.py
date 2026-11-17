#!/usr/bin/env python3
# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
# http://www.apache.org/licenses/LICENSE-2.0
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Exercise the production finalizer caller's targeted admission notification."""

import json
import subprocess
import unittest
from pathlib import Path

import yaml
from merge_queue_admission_test import pr

ROOT = Path(__file__).resolve().parents[3]


def run_wakeup(**options: object) -> dict:
    workflow = yaml.load(
        (ROOT / '.github/workflows/ci-checks.yml').read_text(),
        Loader=yaml.BaseLoader,
    )
    script = workflow['jobs']['check_ci_status']['steps'][-1]['with']['script']
    source = r"""
const fs = require('node:fs');
const f = JSON.parse(fs.readFileSync(0, 'utf8'));
const calls = [], warnings = [];
const github = {rest: {repos: {get: async () => {
  calls.push(['repository']);
  if (f.readFailure) throw Error('repository unavailable');
  return {data:{default_branch:f.branch || 'main'}};
}}, actions: {createWorkflowDispatch: async input => {
  calls.push(['dispatch',input]);
  if(f.dispatchFailure) throw Error('temporary dispatch failure');
}}}};
const context = {repo:{owner:'owner',repo:'repo'}};
const core = {warning:message => warnings.push(message), info:()=>{}};
const realRequire = require;
function load(path) {
  if(path.endsWith('/ci_passed.js')) return {finalize:async () => {
    calls.push(['verified-finalize']); return f.pr;
  }};
  if(path.endsWith('/github_request_accounting.js')) return {withRequestAccounting:async (args, callback) => {
    calls.push(['accounting-start']);
    try {return await callback();} finally {calls.push(['accounting-end']);}
  }};
  return realRequire(process.cwd() + '/' + path);
}
(async()=>{
  await new (Object.getPrototypeOf(async function(){}).constructor)('require','github','context','core',f.script)(load,github,context,core);
  console.log(JSON.stringify({calls,warnings}));
})().catch(error=>{console.error(error);process.exit(1)});
"""
    result = subprocess.run(
        ['node', '-e', source],
        input=json.dumps({'script': script, **options}),
        text=True,
        capture_output=True,
        cwd=ROOT,
        check=True,
    )
    return json.loads(result.stdout)


class AdmissionWakeupTest(unittest.TestCase):
    def test_verified_publisher_wakes_default_branch_inside_accounting(
        self,
    ) -> None:
        for branch in ['main', 'master']:
            with self.subTest(branch=branch):
                result = run_wakeup(pr=pr(), branch=branch)
                self.assertEqual(
                    [c[0] for c in result['calls']],
                    [
                        'accounting-start',
                        'verified-finalize',
                        'repository',
                        'dispatch',
                        'accounting-end',
                    ],
                )
                self.assertEqual(
                    result['calls'][3][1],
                    {
                        'owner': 'owner',
                        'repo': 'repo',
                        'workflow_id': 'merge-queue-recovery.yml',
                        'ref': branch,
                        'inputs': {'pr_number': '7', 'head_sha': 'a' * 40},
                    },
                )

    def test_only_verified_policy_eligible_results_wake_admission(self) -> None:
        for candidate in [
            None,
            pr(labels=[]),
            pr(draft=True),
            pr(
                labels=[
                    {'name': name}
                    for name in ['lgtm', 'approved', 'do-not-merge/hold']
                ]
            ),
        ]:
            with self.subTest(candidate=candidate):
                result = run_wakeup(pr=candidate)
                self.assertEqual(
                    [c[0] for c in result['calls']],
                    ['accounting-start', 'verified-finalize', 'accounting-end'],
                )

    def test_dependabot_uses_existing_exemption(self) -> None:
        result = run_wakeup(
            pr=pr(labels=[], user={'login': 'dependabot[bot]', 'type': 'Bot'})
        )
        self.assertEqual(
            len([c for c in result['calls'] if c[0] == 'dispatch']), 1
        )

    def test_notification_failure_preserves_finalization_and_allows_recovery(
        self,
    ) -> None:
        for failure in [
            {'readFailure': True},
            {'dispatchFailure': True},
            {'branch': 'feature'},
        ]:
            with self.subTest(failure=failure):
                result = run_wakeup(pr=pr(), **failure)
                self.assertEqual(len(result['warnings']), 1)
                self.assertIn(
                    'scheduled recovery will retry', result['warnings'][0]
                )
                self.assertEqual(result['calls'][-1], ['accounting-end'])


if __name__ == '__main__':
    unittest.main()
