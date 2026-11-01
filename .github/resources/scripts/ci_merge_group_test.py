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
"""Exercise the trusted merge-group publisher against GitHub API fixtures."""

import json
import subprocess
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
MODULE = ROOT / '.github/resources/scripts/ci_merge_group.js'
HEAD = 'a' * 40
BASE = 'b' * 40
REBUILT = 'c' * 40
BRANCH = 'gh-readonly-queue/master/pr-7-abcdef'


def lane(path='.github/workflows/frontend.yml', **overrides):
    return {
        'path': path,
        'name': 'Frontend Tests',
        'header_sha256': 'header',
        'pull_request': {'branches': ['master'], 'paths': ['frontend/**']},
        'merge_group': {'types': ['checks_requested'], 'branches': ['master']},
        **overrides,
    }


def good_run(**overrides):
    return {
        'id': 100,
        'path': '.github/workflows/frontend.yml',
        'run_attempt': 1,
        'event': 'merge_group',
        'head_sha': HEAD,
        'head_branch': BRANCH,
        'head_repository': {'full_name': 'owner/repo'},
        'status': 'completed',
        'conclusion': 'success',
        'created_at': '2026-11-01T12:00:00Z',
        'run_started_at': '2026-11-01T12:00:00Z',
        'pull_requests': [],
        **overrides,
    }


def reconcile(runs=None, lanes=None, event=None, refs=None, **overrides):
    workflows = [lane()] if lanes is None else lanes
    fixture = {
        'runs': [good_run()] if runs is None else runs,
        'refs': {} if refs is None else refs,
        'context': {
            'eventName': 'workflow_run',
            'repo': {'owner': 'owner', 'repo': 'repo'},
            'runId': 123,
            'payload': {'workflow_run': good_run() if event is None else event},
        },
        'inventory': {'version': 1, 'workflows': workflows},
        'workflowFiles': [
            {'path': item['path'], 'header_sha256': item['header_sha256']}
            for item in workflows
        ],
        **overrides,
    }
    script = f"""
const gate = require({json.dumps(str(MODULE))});
const fixture = {json.dumps(fixture)};
const writes = [], requests = [], inventoryRequests = [], infos = [];
const refCounts = new Map();
let runCount = 0;
const github = {{
  rest: {{
    git: {{getRef: async options => {{
      requests.push({{route: 'ref', options}});
      const count = refCounts.get(options.ref) || 0;
      refCounts.set(options.ref, count + 1);
      const values = fixture.refs[options.ref] ||
        [options.ref === 'heads/master' ? {json.dumps(BASE)} : {json.dumps(HEAD)}];
      const sha = values[Math.min(count, values.length - 1)];
      if (sha === null) throw Object.assign(new Error('Ref not found'), {{status: 404}});
      return {{data: {{ref: 'refs/' + options.ref, object: {{type: 'commit', sha}}}}}};
    }}}},
    actions: {{listWorkflowRunsForRepo: 'runs'}},
    repos: {{createCommitStatus: async options => {{
      writes.push(options);
      return {{data: options}};
    }}}},
  }},
  paginate: async (route, options) => {{
    requests.push({{route, options}});
    if (route !== 'runs') throw new Error('Unexpected API route: ' + route);
    if (fixture.runsError) throw new Error('API unavailable');
    if (fixture.runSnapshots) {{
      return fixture.runSnapshots[Math.min(runCount++, fixture.runSnapshots.length - 1)];
    }}
    return fixture.runs;
  }},
}};
const loadInventory = async options => {{
  inventoryRequests.push(options.pullRequest);
  if (fixture.inventoryError) throw new Error('Invalid trusted inventory');
  return {{inventory: fixture.inventory, workflowFiles: fixture.workflowFiles}};
}};
const core = {{info: value => infos.push(value), warning: value => infos.push(value),
  setOutput: () => {{}}, error: value => infos.push(value)}};
gate.reconcile({{github, context: fixture.context, core, loadInventory, root: {json.dumps(str(ROOT))}}})
  .then(result => console.log(JSON.stringify({{result, writes, requests, inventoryRequests, infos}})))
  .catch(error => console.log(JSON.stringify({{error: error.message, writes, requests, inventoryRequests, infos}})));
"""
    completed = subprocess.run(
        ['node'],
        input=script,
        text=True,
        capture_output=True,
        check=True,
        cwd=ROOT,
    )
    return json.loads(completed.stdout)


class MergeGroupTest(unittest.TestCase):
    def assert_not_successful(self, result):
        self.assertNotEqual(
            result['writes'][-1]['state'] if result['writes'] else None,
            'success',
            result,
        )

    def test_success_is_bound_to_queue_sha_and_trusted_master_policy(self):
        result = reconcile()
        self.assertNotIn('error', result)
        self.assertEqual(result['writes'][0]['state'], 'pending')
        self.assertEqual(result['writes'][-1]['state'], 'success')
        for status in result['writes']:
            self.assertEqual(status['sha'], HEAD)
            self.assertEqual(status['context'], 'ci-passed')
        self.assertTrue(result['inventoryRequests'])
        for request in result['inventoryRequests']:
            self.assertEqual(request['base']['sha'], BASE)
            self.assertEqual(request['base']['ref'], 'master')
            self.assertEqual(request['base']['repo']['full_name'], 'owner/repo')
        for request in result['requests']:
            if request['route'] == 'runs':
                self.assertEqual(request['options']['event'], 'merge_group')
                self.assertEqual(request['options']['head_sha'], HEAD)

    def test_non_queue_events_do_not_publish(self):
        for changes in [
            {'event': 'pull_request'},
            {'head_branch': 'gh-readonly-queue/release-2.18/pr-7-abcdef'},
            {'head_branch': 'feature'},
            {'head_repository': {'full_name': 'fork/repo'}},
            {'head_sha': 'not-a-sha'},
        ]:
            with self.subTest(changes=changes):
                self.assertEqual(
                    reconcile(event=good_run(**changes))['writes'], []
                )

    def test_missing_or_unsuccessful_lane_blocks(self):
        cases = [
            [],
            [good_run(status='queued', conclusion=None)],
            [good_run(status='in_progress', conclusion=None)],
        ]
        cases.extend(
            [good_run(conclusion=conclusion)]
            for conclusion in [
                'failure',
                'cancelled',
                'skipped',
                'timed_out',
                'neutral',
            ]
        )
        for runs in cases:
            with self.subTest(runs=runs):
                self.assert_not_successful(reconcile(runs=runs))

    def test_wrong_commit_branch_repository_event_and_path_do_not_count(self):
        for changes in [
            {'head_sha': REBUILT},
            {'head_branch': BRANCH + '-other'},
            {'head_repository': {'full_name': 'fork/repo'}},
            {'event': 'pull_request'},
            {'path': '.github/workflows/other.yml'},
        ]:
            with self.subTest(changes=changes):
                self.assert_not_successful(
                    reconcile(runs=[good_run(**changes)])
                )

    def test_rerun_of_older_execution_supersedes_newer_success(self):
        rerun = good_run(
            id=99,
            run_attempt=2,
            status='queued',
            conclusion=None,
            run_started_at='2026-11-01T13:00:00Z',
        )
        self.assert_not_successful(reconcile(runs=[good_run(), rerun]))

    def test_all_master_lanes_run_even_when_pr_paths_are_disjoint(self):
        second = lane(
            '.github/workflows/backend.yml',
            pull_request={'paths': ['backend/**']},
        )
        self.assert_not_successful(reconcile(lanes=[lane(), second]))
        result = reconcile(
            lanes=[lane(), second],
            runs=[good_run(), good_run(id=101, path=second['path'])],
        )
        self.assertEqual(result['writes'][-1]['state'], 'success', result)

    def test_missing_queue_trigger_in_master_lane_fails_closed(self):
        self.assert_not_successful(reconcile(lanes=[lane(merge_group=None)]))
        self.assert_not_successful(
            reconcile(
                lanes=[
                    lane(
                        merge_group={
                            'types': ['checks_requested'],
                            'branches': ['release-2.18'],
                        }
                    )
                ]
            )
        )

    def test_queued_run_without_start_timestamp_remains_pending(self):
        result = reconcile(
            runs=[
                good_run(status='queued', conclusion=None, run_started_at=None)
            ]
        )
        self.assertNotIn('error', result)
        self.assertEqual(result['writes'][-1]['state'], 'pending', result)

    def test_invalid_attempt_identity_and_started_run_timestamps_fail_closed(
        self,
    ):
        for changes in [
            {'id': 0},
            {'id': -1},
            {'run_attempt': 0},
            {'run_attempt': None},
            {'created_at': 0},
            {'run_started_at': None},
            {
                'status': 'in_progress',
                'conclusion': None,
                'run_started_at': None,
            },
            {
                'status': 'queued',
                'conclusion': None,
                'run_started_at': None,
                'run_attempt': 2,
            },
        ]:
            with self.subTest(changes=changes):
                result = reconcile(runs=[good_run(**changes)])
                self.assert_not_successful(result)
                self.assertEqual(
                    result['writes'][-1]['state'], 'failure', result
                )

    def test_empty_inventory_cannot_succeed(self):
        self.assert_not_successful(reconcile(lanes=[]))

    def test_only_explicit_upgrade_pause_can_skip_a_lane(self):
        paused = lane(
            '.github/workflows/upgrade-test.yml', disabled_for_migration=True
        )
        result = reconcile(lanes=[lane(), paused])
        self.assertEqual(result['writes'][-1]['state'], 'success', result)
        other = lane('.github/workflows/other.yml', disabled_for_migration=True)
        self.assert_not_successful(reconcile(lanes=[lane(), other]))

    def test_rebuilt_or_deleted_candidate_is_never_approved(self):
        for values in [
            [REBUILT],
            [HEAD, REBUILT],
            [HEAD, HEAD, REBUILT],
            [None],
            [HEAD, None],
        ]:
            with self.subTest(values=values):
                result = reconcile(refs={'heads/' + BRANCH: values})
                self.assert_not_successful(result)
                self.assertTrue(
                    all(write['sha'] == HEAD for write in result['writes'])
                )

    def test_deleted_queue_ref_after_success_is_lifecycle_completion(self):
        result = reconcile(refs={'heads/' + BRANCH: [HEAD, HEAD, None]})
        self.assertNotIn('error', result)
        self.assertEqual(result['writes'][-1]['state'], 'success', result)

    def test_master_policy_change_blocks_or_revokes_success(self):
        for values in [[BASE, REBUILT], [BASE, BASE, REBUILT]]:
            with self.subTest(values=values):
                self.assert_not_successful(
                    reconcile(refs={'heads/master': values})
                )

    def test_rerun_during_publication_revokes_success(self):
        good = [good_run()]
        pending = [
            good_run(status='in_progress', conclusion=None, run_attempt=2)
        ]
        self.assert_not_successful(
            reconcile(runSnapshots=[good, good, pending])
        )

    def test_api_and_inventory_errors_fail_closed(self):
        for options in [{'runsError': True}, {'inventoryError': True}]:
            with self.subTest(options=options):
                result = reconcile(**options)
                self.assert_not_successful(result)
                self.assertTrue(result['writes'], result)

    def test_malformed_inventory_and_attempt_metadata_fail_closed(self):
        for trigger in [
            [],
            {'paths': ['frontend/**']},
            {'types': ['destroyed']},
            {'branches': ['m?ster']},
        ]:
            with self.subTest(trigger=trigger):
                self.assert_not_successful(
                    reconcile(lanes=[lane(merge_group=trigger)])
                )
        for changes in [
            {'id': None},
            {'created_at': 'invalid'},
            {'run_started_at': 'invalid'},
        ]:
            with self.subTest(changes=changes):
                self.assert_not_successful(
                    reconcile(runs=[good_run(**changes)])
                )

    def test_publisher_executes_trusted_code_without_pr_write_permissions(self):
        import yaml

        # BaseLoader avoids YAML 1.1 interpreting the workflow's `on` as true.
        workflow = yaml.load(
            (ROOT / '.github/workflows/ci-checks.yml').read_text(),
            Loader=yaml.BaseLoader,
        )
        job = workflow['jobs']['check_merge_group_status']
        self.assertNotIn('merge_group', workflow['on'])
        self.assertIn(
            "github.event.workflow_run.event == 'merge_group'", job['if']
        )
        self.assertIn('gh-readonly-queue/master/', job['if'])
        self.assertIn(
            'github.event.workflow_run.head_sha', job['concurrency']['group']
        )
        self.assertEqual(job['concurrency']['cancel-in-progress'], 'false')
        self.assertEqual(
            job['permissions'],
            {'actions': 'read', 'contents': 'read', 'statuses': 'write'},
        )
        checkouts = [
            step
            for step in job['steps']
            if step.get('uses', '').startswith('actions/checkout@')
        ]
        self.assertEqual(len(checkouts), 1)
        self.assertEqual(
            checkouts[0]['with']['ref'], '${{ github.workflow_sha }}'
        )
        self.assertEqual(checkouts[0]['with']['persist-credentials'], 'false')

    def test_truncated_run_history_is_not_complete_evidence(self):
        self.assert_not_successful(reconcile(runs=[good_run()] * 1000))


if __name__ == '__main__':
    unittest.main()
