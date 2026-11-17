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
"""Cancel only obsolete, trusted merge-queue workflow runs."""

import json
import os
import subprocess
import unittest
from pathlib import Path

from ci_merge_group_test import good_run

MODULE = Path(__file__).with_name('merge_queue_dispatch.js')
HEAD = 'a' * 40
OTHER = 'b' * 40
BRANCH = 'gh-readonly-queue/main/pr-7-abcdef'
PATH = '.github/workflows/frontend.yml'


def run(**changes):
    return good_run(
        head_sha=HEAD,
        head_branch=BRANCH,
        path=PATH,
        event='workflow_dispatch',
        status='in_progress',
        conclusion=None,
        actor={'id': 41898282, 'login': 'github-actions[bot]', 'type': 'Bot'},
        **changes,
    )


def cleanup(**options):
    fixture = {'runs': [run()], 'refs': [None], 'event': 'schedule', **options}
    script = r"""
const gate = require(process.argv[1]);
const root = require('node:path').resolve(require('node:path').dirname(process.argv[1]), '../../..');
const f = JSON.parse(require('node:fs').readFileSync(0, 'utf8'));
const calls = [], cancelled = [];
let refIndex = 0;
const github = {rest: {
  repos: {listCommitStatusesForRef:'statuses'},
  actions: {listWorkflowRunsForRepo:'runs',
    getWorkflowRun:async o => { calls.push(['run',o]); return {data:f.live || f.runs.find(r=>r.id===o.run_id)}; },
    cancelWorkflowRun:async o => { calls.push(['cancel',o]); if(f.cancelError===o.run_id) throw new Error('cancel conflict'); cancelled.push(o.run_id); }},
  git: {getRef:async o => {
    calls.push(['ref',o]);
    const sha=f.refs[Math.min(refIndex++,f.refs.length-1)];
    if (sha===null) throw Object.assign(new Error('missing'),{status:404});
    if (sha==='error') throw Object.assign(new Error('unavailable'),{status:403});
    return {data:{ref:'refs/'+o.ref,object:{type:'commit',sha}}};
  }}
}, paginate:async (route,o)=> {
 if(route==='statuses') {
   calls.push(['statuses',o]);
   return f.statuses || f.runs.map(r=>({context:'ci-queue-dispatch/'+r.path.split('/').pop(),state:'pending',
     created_at:'2026-11-01T11:59:00Z', description:'Queue dispatch attempt 1; base '+ 'b'.repeat(40)+'.'}));
 }
 calls.push(['list',o]);
 if(f.listError) throw new Error('listing unavailable');
 return f.runs.filter(r=>!o.status || r.status===o.status);
}};
(async()=> {
 let error;
 try {
   if (gate.cancelObsoleteCandidates) await gate.cancelObsoleteCandidates({github,
     context:{repo:{owner:'owner',repo:'repo'},eventName:f.event,
       payload:{workflow_run:f.trigger}},core:{info:()=>{}},
     root:f.omitRoot ? undefined : root, branches:['main','release-2.18'], workflowPaths:f.localInventory ? undefined : new Set(['.github/workflows/frontend.yml'])});
 } catch(e) {error=e.message;}
 console.log(JSON.stringify({cancelled,calls,error}));
})();
"""
    result = subprocess.run(
        ['node', '-e', script, str(MODULE)],
        input=json.dumps(fixture),
        check=True,
        text=True,
        capture_output=True,
        env={
            **os.environ,
            'GITHUB_WORKSPACE': str(MODULE.resolve().parents[3]),
        },
    )
    return json.loads(result.stdout)


class CleanupTest(unittest.TestCase):
    def test_manual_dispatch_after_controller_reservation_is_preserved(self):
        manual = run()
        manual['actor'] = {'id': 123, 'login': 'maintainer', 'type': 'User'}
        result = cleanup(runs=[manual])
        self.assertEqual(result['cancelled'], [])
        self.assertFalse(any(kind == 'statuses' for kind, _ in result['calls']))

    def test_dispatch_actor_is_rechecked_before_cancellation(self):
        for actor in (
            None,
            {'id': 123, 'login': 'maintainer', 'type': 'User'},
            {'id': 123, 'login': 'github-actions[bot]', 'type': 'Bot'},
            {'id': 41898282, 'login': 'other-bot', 'type': 'Bot'},
            {'id': 41898282, 'login': 'github-actions[bot]', 'type': 'User'},
        ):
            with self.subTest(actor=actor):
                current = run()
                current['actor'] = actor
                self.assertEqual(cleanup(live=current)['cancelled'], [])
                self.assertEqual(cleanup(runs=[current])['cancelled'], [])

    def test_human_rerun_keeps_original_controller_ownership(self):
        candidate = run()
        candidate['run_attempt'] = 2
        candidate['triggering_actor'] = {
            'id': 123,
            'login': 'maintainer',
            'type': 'User',
        }
        self.assertEqual(
            cleanup(runs=[candidate])['cancelled'], [candidate['id']]
        )

    def test_deleted_candidate_is_cancelled(self):
        result = cleanup()
        self.assertEqual(result['cancelled'], [run()['id']])
        self.assertNotIn('error', result)

    def test_replaced_candidate_is_cancelled(self):
        self.assertEqual(cleanup(refs=[OTHER])['cancelled'], [run()['id']])

    def test_current_candidate_is_preserved(self):
        self.assertEqual(cleanup(refs=[HEAD])['cancelled'], [])

    def test_local_inventory_includes_release_companion_and_empty_queue(self):
        candidate = run()
        candidate.update(
            head_branch='gh-readonly-queue/release-2.18/pr-7-abcdef',
            path='.github/workflows/build-tools-images-merge-group.yml',
        )
        result = cleanup(runs=[candidate], localInventory=True, omitRoot=True)
        self.assertNotIn('error', result)
        self.assertEqual(result['cancelled'], [candidate['id']])
        result = cleanup(runs=[], localInventory=True)
        self.assertNotIn('error', result)
        self.assertEqual(result['cancelled'], [])

    def test_native_and_waiting_runs_are_cancelled(self):
        for status in (
            'queued',
            'in_progress',
            'waiting',
            'pending',
            'requested',
        ):
            candidate = run()
            candidate.update(
                event='merge_group',
                status=status,
                actor={'id': 123, 'login': 'maintainer', 'type': 'User'},
            )
            self.assertEqual(
                cleanup(runs=[candidate])['cancelled'], [candidate['id']]
            )

    def test_fresh_malformed_evidence_cannot_cancel(self):
        current = run()
        current['created_at'] = None
        result = cleanup(live=current)
        self.assertEqual(result['cancelled'], [])
        self.assertIn('error', result)

    def test_cancel_race_does_not_strand_other_obsolete_runs(self):
        first, second = run(), run()
        second['id'] += 1
        result = cleanup(runs=[first, second], cancelError=first['id'])
        self.assertEqual(result['cancelled'], [second['id']])
        self.assertIn('error', result)

    def test_ref_lookup_failure_cannot_cancel(self):
        result = cleanup(refs=['error'])
        self.assertEqual(result['cancelled'], [])
        self.assertIn('error', result)

    def test_ref_restored_before_cancel_is_preserved(self):
        self.assertEqual(cleanup(refs=[None, HEAD])['cancelled'], [])

    def test_new_attempt_is_preserved(self):
        live = run()
        live['run_attempt'] += 1
        self.assertEqual(cleanup(live=live)['cancelled'], [])

    def test_completed_run_is_preserved(self):
        live = run()
        live.update(status='completed', conclusion='success')
        self.assertEqual(cleanup(live=live)['cancelled'], [])

    def test_foreign_and_untrusted_runs_are_preserved(self):
        for changes in (
            {'head_branch': 'main'},
            {'head_branch': 'gh-readonly-queue/main/pr-7-abc/extra'},
            {'event': 'pull_request'},
            {'path': '.github/workflows/untrusted.yml'},
            {'head_repository': {'full_name': 'fork/repo'}},
            {'head_sha': 'invalid'},
        ):
            with self.subTest(changes=changes):
                candidate = run()
                candidate.update(changes)
                self.assertEqual(cleanup(runs=[candidate])['cancelled'], [])

    def test_unrelated_event_does_not_scan(self):
        result = cleanup(event='pull_request_target')
        self.assertEqual(result['calls'], [])

    def test_completion_event_only_reads_affected_candidate(self):
        result = cleanup(event='workflow_run', trigger=run())
        self.assertEqual(result['cancelled'], [run()['id']])
        lists = [o for kind, o in result['calls'] if kind == 'list']
        self.assertEqual(len(lists), 2)
        self.assertTrue(all(o['head_sha'] == HEAD for o in lists))

    def test_active_candidates_share_ref_reads_and_discovery_is_bounded(self):
        first, second = run(), run()
        second['id'] += 1
        result = cleanup(runs=[first, second], refs=[HEAD])
        self.assertEqual(result['cancelled'], [])
        self.assertEqual(len([c for c in result['calls'] if c[0] == 'ref']), 1)
        lists = [o for kind, o in result['calls'] if kind == 'list']
        self.assertEqual(
            {o['status'] for o in lists},
            {'queued', 'in_progress', 'waiting', 'pending', 'requested'},
        )
        self.assertEqual(len(lists), 5)

    def test_manual_queue_ref_dispatch_without_receipt_is_preserved(self):
        candidate = run()
        candidate['path'] = '.github/workflows/release-qualification.yml'
        result = cleanup(runs=[candidate], localInventory=True, statuses=[])
        self.assertEqual(result['cancelled'], [])
        self.assertNotIn('error', result)

    def test_missing_or_malformed_reservation_is_preserved(self):
        for statuses in (
            [],
            [
                {
                    'context': 'ci-queue-dispatch/frontend.yml',
                    'state': 'success',
                    'description': 'anything',
                }
            ],
            [
                {
                    'context': 'ci-queue-dispatch/frontend.yml',
                    'state': 'pending',
                    'description': 'Queue dispatch attempt 4; base '
                    + OTHER
                    + '.',
                }
            ],
        ):
            self.assertEqual(cleanup(statuses=statuses)['cancelled'], [])

    def test_later_reservation_cannot_cancel_earlier_manual_run(self):
        status = {
            'context': 'ci-queue-dispatch/frontend.yml',
            'state': 'pending',
            'description': 'Queue dispatch attempt 1; base ' + OTHER + '.',
            'created_at': '2026-11-01T12:01:00Z',
        }
        self.assertEqual(cleanup(statuses=[status])['cancelled'], [])
        status['created_at'] = None
        self.assertEqual(cleanup(statuses=[status])['cancelled'], [])

    def test_obsolete_dispatches_share_reservation_read(self):
        first, second = run(), run()
        second['id'] += 1
        result = cleanup(runs=[first, second])
        self.assertEqual(result['cancelled'], [first['id'], second['id']])
        self.assertEqual(
            len([c for c in result['calls'] if c[0] == 'statuses']), 1
        )

    def test_discovery_is_bounded_and_failure_preserves_runs(self):
        result = cleanup(runs=[run()] * 1000)
        self.assertEqual(result['cancelled'], [])
        self.assertIn('error', result)


if __name__ == '__main__':
    unittest.main()
