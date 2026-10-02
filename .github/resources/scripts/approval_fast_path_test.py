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
"""Exercise the pull_request_target workflow approval script with GitHub
mocks."""

import copy
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[3]
WORKFLOW = ROOT / '.github/workflows/gh-workflow-approve.yml'

NODE_HARNESS = r'''
const fs = require('node:fs');
const AsyncFunction = Object.getPrototypeOf(async function() {}).constructor;
const {script, scenario} = JSON.parse(fs.readFileSync(0, 'utf8'));
const result = {approved: [], failures: [], notices: [], warnings: [], errors: [], requests: []};
const currentPRs = scenario.currentPRs || [scenario.eventPR];
const runBatches = scenario.runBatches || [[]];
let currentRead = 0;
let runRead = 0;
const pullsList = () => {};
const runsList = () => {};
const github = {
  rest: {
    pulls: {
      get: async request => {
        result.requests.push({kind: 'getPR', request});
        const index = Math.min(currentRead++, currentPRs.length - 1);
        return {data: currentPRs[index]};
      },
      list: pullsList,
    },
    actions: {
      listWorkflowRunsForRepo: runsList,
      approveWorkflowRun: async request => {
        result.requests.push({kind: 'approve', request});
        if ((scenario.failApprovals || []).includes(request.run_id)) {
          throw new Error('approval rejected');
        }
        result.approved.push(request.run_id);
      },
    },
  },
  paginate: async (endpoint, request) => {
    if (endpoint === pullsList) {
      result.requests.push({kind: 'listPRs', request});
      return scenario.openHeadPRs || [scenario.eventPR];
    }
    if (endpoint === runsList) {
      result.requests.push({kind: 'listRuns', request});
      const index = Math.min(runRead++, runBatches.length - 1);
      return runBatches[index];
    }
    throw new Error('Unexpected paginated endpoint');
  },
};
const context = {
  repo: {owner: 'kubeflow', repo: 'pipelines'},
  payload: {pull_request: scenario.eventPR},
};
const core = {
  info: () => {},
  error: message => result.errors.push(message),
  notice: message => result.notices.push(message),
  warning: message => result.warnings.push(message),
  setFailed: message => result.failures.push(message),
};
const immediateTimeout = callback => callback();
(async () => {
  try {
    await new AsyncFunction('github', 'context', 'core', 'setTimeout', 'require', script)(
      github, context, core, immediateTimeout, require);
  } catch (error) { result.error = error.message; }
  process.stdout.write(JSON.stringify(result));
})().catch(error => {
  process.stderr.write(error.stack || String(error));
  process.exitCode = 1;
});
'''


def _approval_script() -> str:
    workflow = WORKFLOW.read_text(encoding='utf-8')
    step = workflow.split(
        '      - name: Approve Pending Workflow Runs\n', maxsplit=1)[1]
    indented = step.split('          script: |\n', maxsplit=1)[1]
    lines = []
    for line in indented.splitlines():
        if line and not line.startswith('            '):
            break
        lines.append(line[12:] if line else '')
    return '\n'.join(lines)


def _pr(number=42,
        *,
        sha='head-sha',
        repo_id=700,
        branch='feature',
        state='open',
        labels=('ok-to-test',),
        author='contributor',
        association='CONTRIBUTOR'):
    return {
        'number': number,
        'state': state,
        'user': {
            'login': author
        },
        'author_association': association,
        'head': {
            'sha': sha,
            'ref': branch,
            'repo': {
                'id': repo_id,
                'owner': {
                    'login': 'contributor'
                },
            },
        },
        'labels': [{
            'name': label
        } for label in labels],
    }


def _run(run_id=101,
         *,
         sha='head-sha',
         repo_id=700,
         branch='feature',
         pr_numbers=(42,),
         event='pull_request'):
    return {
        'id': run_id,
        'name': f'Presubmit {run_id}',
        'event': event,
        'head_sha': sha,
        'head_branch': branch,
        'head_repository': {
            'id': repo_id
        },
        'pull_requests': [{
            'number': number
        } for number in pr_numbers],
    }


@unittest.skipUnless(
    shutil.which('node'), 'Node.js is required for workflow JS tests')
class ApprovalFastPathTest(unittest.TestCase):

    def run_approval(self,
                     *,
                     event_pr=None,
                     current_prs=None,
                     open_head_prs=None,
                     run_batches=None,
                     members=('trusted-member',),
                     allow_error=False,
                     fail_approvals=()):
        event_pr = event_pr or _pr()
        scenario = {
            'eventPR':
                event_pr,
            'currentPRs':
                current_prs
                if current_prs is not None else [copy.deepcopy(event_pr)],
            'openHeadPRs':
                open_head_prs
                if open_head_prs is not None else [copy.deepcopy(event_pr)],
            'runBatches':
                run_batches if run_batches is not None else [[]],
            'failApprovals':
                list(fail_approvals),
        }
        with tempfile.TemporaryDirectory() as directory:
            members_file = Path(directory) / 'members.json'
            if members is not None:
                members_file.write_text(json.dumps(members), encoding='utf-8')
            completed = subprocess.run(
                ['node', '-e', NODE_HARNESS],
                input=json.dumps({
                    'script': _approval_script(),
                    'scenario': scenario,
                }),
                text=True,
                capture_output=True,
                check=False,
                timeout=10,
                cwd=ROOT,
                env={
                    **os.environ, 'KUBEFLOW_MEMBERS_FILE': str(members_file)
                },
            )
        self.assertEqual(completed.returncode, 0, completed.stderr)
        result = json.loads(completed.stdout)
        if not allow_error:
            self.assertIsNone(result.get('error'))
        return result

    def test_live_label_is_required_for_external_author(self):
        for labels in ((), ('ok-to-test', 'needs-ok-to-test')):
            with self.subTest(labels=labels):
                result = self.run_approval(
                    current_prs=[_pr(labels=labels)],
                    run_batches=[[_run()]],
                )
                self.assertEqual(result['approved'], [])
                self.assertEqual(result['failures'], [])
                self.assertFalse(
                    any(request['kind'] == 'listRuns'
                        for request in result['requests']))

    def test_changed_or_closed_pr_head_is_not_approved(self):
        changed = (
            _pr(sha='new-head'),
            _pr(repo_id=701),
            _pr(branch='renamed'),
            _pr(state='closed'),
        )
        for current_pr in changed:
            with self.subTest(current_pr=current_pr):
                result = self.run_approval(
                    current_prs=[current_pr], run_batches=[[_run()]])
                self.assertEqual(result['approved'], [])
                self.assertFalse(
                    any(request['kind'] == 'listRuns'
                        for request in result['requests']))

    def test_ambiguous_open_prs_with_same_head_need_manual_approval(self):
        result = self.run_approval(
            open_head_prs=[_pr(), _pr(number=43)],
            run_batches=[[_run()]],
        )
        self.assertEqual(result['approved'], [])
        self.assertEqual(len(result['failures']), 1)
        self.assertIn('exactly one open PR', result['failures'][0])

    def test_run_identity_and_pr_association_are_checked(self):
        runs = [
            _run(201, repo_id=701),
            _run(202, branch='other-branch'),
            _run(203, sha='other-sha'),
            _run(204, pr_numbers=(43,)),
            _run(205, event='workflow_dispatch'),
            _run(206),
        ]
        result = self.run_approval(run_batches=[runs])
        self.assertEqual(result['approved'], [206])
        self.assertEqual(result['failures'], [])
        run_requests = [
            request['request']
            for request in result['requests']
            if request['kind'] == 'listRuns'
        ]
        self.assertTrue(run_requests)
        self.assertTrue(
            all(request['head_sha'] == 'head-sha' and request['event'] ==
                'pull_request' and request['status'] == 'action_required'
                for request in run_requests))

    def test_member_approves_run_registered_after_first_poll(self):
        result = self.run_approval(
            current_prs=[_pr(labels=())],
            members=('CONTRIBUTOR',),
            run_batches=[[], [_run()]],
        )
        self.assertEqual(result['approved'], [101])
        self.assertEqual(result['failures'], [])
        self.assertGreaterEqual(
            len([
                request for request in result['requests']
                if request['kind'] == 'listRuns'
            ]), 2)

    def test_association_does_not_approve_unlisted_author(self):
        for association in ('MEMBER', 'OWNER', 'COLLABORATOR'):
            with self.subTest(association=association):
                result = self.run_approval(
                    event_pr=_pr(labels=(), association=association),
                    run_batches=[[_run()]],
                )
                self.assertEqual(result['approved'], [])
                self.assertFalse(
                    any(request['kind'] == 'listRuns'
                        for request in result['requests']))

    def test_needs_ok_to_test_blocks_members_and_dependabot(self):
        for author in ('contributor', 'dependabot[bot]'):
            with self.subTest(author=author):
                result = self.run_approval(
                    event_pr=_pr(
                        author=author,
                        labels=('ok-to-test', 'needs-ok-to-test')),
                    members=('contributor',),
                    run_batches=[[_run()]],
                )
                self.assertEqual(result['approved'], [])
                self.assertFalse(
                    any(request['kind'] == 'listRuns'
                        for request in result['requests']))

    def test_dependabot_without_label_is_approved(self):
        result = self.run_approval(
            event_pr=_pr(author='dependabot[bot]', labels=()),
            run_batches=[[_run()]],
        )
        self.assertEqual(result['approved'], [101])

    def test_live_needs_ok_to_test_stops_member_approval(self):
        result = self.run_approval(
            event_pr=_pr(labels=()),
            current_prs=[_pr(labels=()),
                         _pr(labels=('needs-ok-to-test',))],
            members=('contributor',),
            run_batches=[[], [_run()]],
        )
        self.assertEqual(result['approved'], [])
        self.assertEqual(
            len([
                request for request in result['requests']
                if request['kind'] == 'listRuns'
            ]), 1)

    def test_missing_or_invalid_membership_fails_before_approval(self):
        for members in (None, {}, []):
            with self.subTest(members=members):
                result = self.run_approval(
                    members=members, allow_error=True, run_batches=[[_run()]])
                self.assertTrue(result.get('error'))
                self.assertEqual(result['approved'], [])
                self.assertEqual(result['requests'], [])

    def test_unresolved_action_required_run_fails(self):
        result = self.run_approval(
            run_batches=[[_run()]], fail_approvals=(101,))
        self.assertEqual(result['approved'], [])
        self.assertEqual(len(result['failures']), 1)
        self.assertIn('101', result['failures'][0])


if __name__ == '__main__':
    unittest.main()
