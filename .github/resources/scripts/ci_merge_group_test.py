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
import tempfile
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
        'repository': {'full_name': 'owner/repo'},
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
let runCount = 0, triggerCount = 0;
let queueCount = 0, prCount = 0;
const github = {{
  rest: {{
    pulls: {{get: async () => ({{data: fixture.queuePRs[Math.min(prCount++, fixture.queuePRs.length - 1)]}})}},
    git: {{getRef: async options => {{
      requests.push({{route: 'ref', options}});
      const count = refCounts.get(options.ref) || 0;
      refCounts.set(options.ref, count + 1);
      const values = fixture.refs[options.ref] ||
        [['heads/master', 'heads/main'].includes(options.ref) ? {json.dumps(BASE)} : {json.dumps(HEAD)}];
      const sha = values[Math.min(count, values.length - 1)];
      if (sha === null) throw Object.assign(new Error('Ref not found'), {{status: 404}});
      return {{data: {{ref: 'refs/' + options.ref, object: {{type: 'commit', sha}}}}}};
    }}}},
    actions: {{listWorkflowRunsForRepo: 'runs', getWorkflowRun: async options => {{
      requests.push({{route: 'trigger', options}});
      if (fixture.triggerError) throw new Error('Trigger API unavailable');
      const snapshots = fixture.triggerSnapshots || [fixture.context.payload.workflow_run];
      return {{data: snapshots[Math.min(triggerCount++, snapshots.length - 1)]}};
    }}}},
    repos: {{createCommitStatus: async options => {{
      writes.push(options);
      return {{data: options}};
    }}}},
  }},
  paginate: async (route, options) => {{
    requests.push({{route, options}});
    if (route.includes('/rules/branches/')) {{
      if (fixture.rulesError) throw new Error('Rules API unavailable');
      return fixture.rules || [{{type: 'merge_queue'}}];
    }}
    if (route !== 'runs') throw new Error('Unexpected API route: ' + route);
    if (fixture.runsError) throw new Error('API unavailable');
    if (fixture.runSnapshots) {{
      return fixture.runSnapshots[Math.min(runCount++, fixture.runSnapshots.length - 1)];
    }}
    return fixture.runs;
  }},
  graphql: async query => {{
    if (query.includes('query QueuePolicies')) {{
      const repository = {{nameWithOwner: 'owner/repo'}};
      for (const match of query.matchAll(/pr([0-9]+): pullRequest/g)) {{
        const pr = fixture.queuePRs[Math.min(prCount++, fixture.queuePRs.length - 1)];
        repository['pr' + match[1]] = {{id: pr.node_id, number: pr.number,
          state: pr.state.toUpperCase(), isDraft: pr.draft, merged: pr.merged,
          headRefOid: pr.head.sha, baseRefName: pr.base.ref,
          baseRepository: {{nameWithOwner: pr.base.repo.full_name}},
          author: {{__typename: pr.user.type, login: pr.user.login}},
          labels: {{nodes: pr.labels, totalCount: pr.labels.length,
            pageInfo: {{hasNextPage: false}}}}}};
      }}
      return {{repository}};
    }}
    return fixture.queues[Math.min(queueCount++, fixture.queues.length - 1)];
  }},
}};
const loadInventory = async options => {{
  inventoryRequests.push(options.pullRequest);
  if (fixture.inventoryError) throw new Error('Invalid trusted inventory');
  return {{inventory: fixture.inventory, workflowFiles: fixture.workflowFiles}};
}};
const core = {{info: value => infos.push(value), warning: value => infos.push(value),
  setOutput: () => {{}}, error: value => infos.push(value)}};
gate.reconcile({{github, context: fixture.context, core, loadInventory,
  root: fixture.policyRoot || process.argv[2], group: fixture.group}})
  .then(result => console.log(JSON.stringify({{result, writes, requests, inventoryRequests, infos}})))
  .catch(error => console.log(JSON.stringify({{error: error.message, writes, requests, inventoryRequests, infos}})));
"""
    # Native merge-group fixtures model the preactivation publisher. Explicit
    # policyRoot cases below exercise enabled admission and queue membership.
    with tempfile.TemporaryDirectory() as directory:
        policy = Path(directory) / '.github/merge-queue-policy.json'
        policy.parent.mkdir()
        policy.write_text(json.dumps({'version': 1, 'enabled_branches': []}))
        completed = subprocess.run(
            ['node', '-', directory],
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
                self.assertIn(
                    request['options']['event'],
                    ['merge_group', 'workflow_dispatch'],
                )
                self.assertEqual(request['options']['head_sha'], HEAD)

    def test_dispatch_evidence_requires_enabled_queue_and_latest_attempt(self):
        from merge_queue_admission_test import entry, pr, queue

        dispatch = good_run(event='workflow_dispatch')
        self.assertEqual(
            reconcile(event=dispatch, runs=[dispatch])['writes'], []
        )
        with tempfile.TemporaryDirectory() as directory:
            policy = Path(directory) / '.github/merge-queue-policy.json'
            policy.parent.mkdir()
            policy.write_text(
                json.dumps({'version': 1, 'enabled_branches': ['master']})
            )
            options = dict(
                policyRoot=directory,
                event=dispatch,
                queues=[queue([entry(headCommit={'oid': HEAD})])],
                queuePRs=[pr()],
            )
            recovery = reconcile(
                runs=[dispatch],
                **dict(
                    options,
                    context={
                        'eventName': 'workflow_run',
                        'repo': {'owner': 'owner', 'repo': 'repo'},
                        'runId': 123,
                        'payload': {
                            'workflow_run': good_run(event='pull_request')
                        },
                    },
                ),
                group={'head': HEAD, 'branch': BRANCH, 'baseRef': 'master'},
            )
            self.assertEqual(
                recovery['writes'][-1]['state'], 'success', recovery
            )
            result = reconcile(runs=[dispatch], **options)
            self.assertTrue(
                result['writes'],
                'Enabled dispatch must publish candidate status',
            )
            self.assertEqual(result['writes'][-1]['state'], 'success', result)
            for later_event in ['merge_group', 'workflow_dispatch']:
                newer = good_run(
                    id=101,
                    event=later_event,
                    created_at='2026-11-01T12:01:00Z',
                    run_started_at='2026-11-01T12:01:00Z',
                    conclusion='failure',
                )
                result = reconcile(runs=[dispatch, newer], **options)
                self.assertEqual(
                    result['writes'][-1]['state'], 'failure', result
                )

    def test_triggering_rerun_cannot_reuse_stale_list_success(self):
        for status, conclusion in [
            ('in_progress', None),
            ('completed', 'failure'),
            ('completed', 'success'),
        ]:
            with self.subTest(status=status):
                trigger = good_run(
                    run_attempt=2, status=status, conclusion=conclusion
                )
                result = reconcile(event=trigger, runs=[good_run()])
                self.assert_not_successful(result)
                self.assertNotIn(
                    'success', [write['state'] for write in result['writes']]
                )
        result = reconcile(
            event=good_run(run_attempt=2),
            runs=[good_run()],
            triggerSnapshots=[good_run()],
        )
        self.assertIn('error', result)
        self.assert_not_successful(result)

    def test_delayed_event_does_not_override_a_newer_completed_execution(self):
        older = good_run(
            id=99,
            conclusion='failure',
            created_at='2026-11-01T11:00:00Z',
            run_started_at='2026-11-01T11:00:00Z',
        )
        result = reconcile(event=older)
        self.assertEqual(result['writes'][-1]['state'], 'success', result)

    def test_trigger_attempt_is_rechecked_around_publication(self):
        active = good_run(run_attempt=2, status='in_progress', conclusion=None)
        for snapshots in [
            [good_run(), active],
            [good_run(), good_run(), active],
        ]:
            with self.subTest(snapshots=snapshots):
                result = reconcile(triggerSnapshots=snapshots)
                self.assert_not_successful(result)

    def test_invalid_trigger_identity_and_lookup_errors_fail_closed(self):
        for changes in [
            {'id': 0},
            {'run_attempt': 0},
            {'head_sha': REBUILT},
            {'event': 'pull_request'},
            {'path': '.github/workflows/other.yml'},
            {'repository': {'full_name': 'fork/repo'}},
            {'head_repository': {'full_name': 'fork/repo'}},
        ]:
            with self.subTest(changes=changes):
                result = reconcile(triggerSnapshots=[good_run(**changes)])
                self.assertIn('error', result)
                self.assert_not_successful(result)
        result = reconcile(triggerError=True)
        self.assertIn('error', result)
        self.assert_not_successful(result)

    def test_queue_policy_lookup_failure_invalidates_prior_success(self):
        with tempfile.TemporaryDirectory() as directory:
            policy = Path(directory) / '.github/merge-queue-policy.json'
            policy.parent.mkdir()
            policy.write_text(
                json.dumps({'version': 1, 'enabled_branches': ['master']})
            )
            for options in [{'rulesError': True}, {'rules': []}]:
                with self.subTest(options=options):
                    result = reconcile(policyRoot=directory, **options)
                    self.assertIn('error', result)
                    self.assertTrue(
                        result['writes'], 'Prior success must be invalidated'
                    )
                    self.assert_not_successful(result)

    def test_enabled_queue_policy_blocks_or_revokes_unapproved_candidates(self):
        from merge_queue_admission_test import entry, pr, queue

        for branch in ['master', 'main']:
            with (
                self.subTest(branch=branch),
                tempfile.TemporaryDirectory() as directory,
            ):
                policy = Path(directory) / '.github/merge-queue-policy.json'
                policy.parent.mkdir()
                policy.write_text(
                    json.dumps({'version': 1, 'enabled_branches': [branch]})
                )
                run = good_run(
                    head_branch=f'gh-readonly-queue/{branch}/pr-7-abcdef'
                )
                member = entry(branch=branch, headCommit={'oid': HEAD})
                eligible = pr(branch=branch)
                held = pr(branch=branch, labels=[{'name': 'do-not-merge/hold'}])
                options = dict(
                    policyRoot=directory,
                    event=run,
                    runs=[run],
                    lanes=[
                        lane(
                            pull_request={'branches': [branch]},
                            merge_group={'branches': [branch]},
                        )
                    ],
                    queues=[queue([member])],
                )
                result = reconcile(**options, queuePRs=[eligible])
                self.assertEqual(
                    result['writes'][-1]['state'], 'success', result
                )
                for snapshots in [
                    [held],
                    [eligible, held],
                    [eligible, eligible, held],
                ]:
                    result = reconcile(**options, queuePRs=snapshots)
                    self.assertEqual(
                        result['writes'][-1]['state'], 'failure', result
                    )
                changed = dict(options, queues=[queue([member]), queue([])])
                result = reconcile(**changed, queuePRs=[eligible])
                self.assertEqual(
                    result['writes'][-1]['state'], 'failure', result
                )

    def test_each_queue_uses_its_own_base_policy_and_ref(self):
        for base in ['master', 'main']:
            with self.subTest(base=base):
                branch = f'gh-readonly-queue/{base}/pr-7-abcdef'
                run = good_run(head_branch=branch)
                definition = lane(
                    pull_request={'branches': [base]},
                    merge_group={'branches': [base]},
                )
                result = reconcile(event=run, runs=[run], lanes=[definition])
                self.assertTrue(result['writes'], result)
                self.assertEqual(
                    result['writes'][-1]['state'], 'success', result
                )
                self.assertEqual(
                    result['inventoryRequests'][0]['base']['ref'], base
                )
                self.assertEqual(
                    result['inventoryRequests'][0]['base']['sha'], BASE
                )
                refs = [
                    r['options']['ref']
                    for r in result['requests']
                    if r['route'] == 'ref'
                ]
                self.assertIn('heads/' + base, refs)
                other = 'main' if base == 'master' else 'master'
                self.assertNotIn('heads/' + other, refs)
                for values in [[BASE, REBUILT], [BASE, BASE, REBUILT]]:
                    changed = reconcile(
                        event=run,
                        runs=[run],
                        lanes=[definition],
                        refs={'heads/' + base: values},
                    )
                    self.assert_not_successful(changed)
                missing = reconcile(
                    event=run,
                    runs=[run],
                    lanes=[
                        lane(
                            pull_request={'branches': [base]},
                            merge_group={'branches': [other]},
                        )
                    ],
                )
                self.assertEqual(
                    missing['writes'][-1]['state'], 'failure', missing
                )
                wrong_branch = reconcile(
                    event=run,
                    runs=[
                        good_run(
                            head_branch=f'gh-readonly-queue/{other}/pr-7-abcdef'
                        )
                    ],
                    lanes=[definition],
                )
                self.assert_not_successful(wrong_branch)

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
        for base in ['master', 'main']:
            self.assertIn(f'gh-readonly-queue/{base}/', job['if'])
        self.assertIn(
            'github.event.workflow_run.head_sha', job['concurrency']['group']
        )
        self.assertEqual(job['concurrency']['cancel-in-progress'], 'false')
        self.assertEqual(
            job['permissions'],
            {
                'actions': 'read',
                'contents': 'read',
                'statuses': 'write',
                'pull-requests': 'read',
            },
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
