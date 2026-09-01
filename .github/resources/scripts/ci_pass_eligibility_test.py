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

"""Executable regression tests for the ci-passed state machine."""

import json
import os
from pathlib import Path
import subprocess
import tempfile
import textwrap
import unittest

ROOT = Path(__file__).resolve().parents[3]
CI_CHECKS_PATH = ROOT / '.github/workflows/ci-checks.yml'
ADD_LABEL_PATH = ROOT / '.github/workflows/add-ci-passed-label.yml'
TRUSTED_ASSOCIATIONS = {'MEMBER', 'OWNER', 'COLLABORATOR'}
AUTHORS = {
    'human_member': ('alice', 'MEMBER'),
    'human_contributor': ('bob', 'CONTRIBUTOR'),
    'dependabot': ('dependabot[bot]', 'NONE'),
    'other_bot': ('renovate[bot]', 'CONTRIBUTOR'),
}


def extract_step_script(workflow: str, step_name: str,
                        next_step_name: str | None = None) -> str:
    start = workflow.index(f'      - name: {step_name}')
    if next_step_name:
        end = workflow.index(f'      - name: {next_step_name}', start)
    else:
        end = len(workflow)
    step = workflow[start:end]
    marker = 'run: |' if 'run: |' in step else 'script: |'
    marker_idx = step.index(marker)
    # Indentation of the marker key (``run:``/``script:``). The literal block
    # body is indented further and ends at the first line indented no deeper
    # than the key itself (a following step, job header, or EOF).
    line_start = step.rfind('\n', 0, marker_idx) + 1
    marker_indent = len(step[line_start:marker_idx]) - len(
        step[line_start:marker_idx].lstrip())
    lines = step[marker_idx + len(marker):].split('\n')
    block: list[str] = []
    for line in lines:
        if line.strip() == '':
            block.append('')
            continue
        indent = len(line) - len(line.lstrip())
        if indent <= marker_indent:
            break
        block.append(line)
    while block and block[-1].strip() == '':
        block.pop()
    return textwrap.dedent('\n'.join(block)).strip()


def expected_eligible(author: str, association: str, has_ok: bool,
                      has_needs: bool, base_changed: bool = False) -> bool:
    return (not has_needs and not base_changed and
            (has_ok or author == 'dependabot[bot]' or
             association in TRUSTED_ASSOCIATIONS))


def execute_eligibility(workflow: str, author: str, association: str,
                        has_ok: bool, has_needs: bool, action: str = 'opened',
                        old_base: str = '') -> bool:
    script = extract_step_script(
        workflow, 'Determine whether CI polling is needed',
        'Mark ci-passed pending while current head is revalidated')
    script = script.replace(
        "${{ contains(github.event.pull_request.labels.*.name, 'ok-to-test') }}",
        'true' if has_ok else 'false')
    script = script.replace(
        "${{ contains(github.event.pull_request.labels.*.name, 'needs-ok-to-test') }}",
        'true' if has_needs else 'false')
    with tempfile.TemporaryDirectory() as directory:
        output = Path(directory) / 'output'
        env = {
            **os.environ,
            'PR_AUTHOR': author,
            'AUTHOR_ASSOCIATION': association,
            'EVENT_ACTION': action,
            'LABEL_NAME': '',
            'OLD_BASE_REF': old_base,
            'GITHUB_OUTPUT': str(output),
        }
        subprocess.run(['bash', '-c', script], env=env, check=True,
                       capture_output=True, text=True)
        values = dict(line.split('=', 1)
                      for line in output.read_text().splitlines()
                      if '=' in line)
    return values['should_poll'] == 'true'


def execute_javascript(script: str, setup: str, workspace: Path) -> list:
    harness = f'''
const calls = [];
const core = {{
  info: (message) => calls.push(["info", String(message)]),
  setOutput: (name, value) => calls.push(["setOutput", String(name), String(value)]),
}};
{setup}
(async () => {{
{textwrap.indent(script, '  ')}
}})().then(() => console.log(JSON.stringify(calls))).catch((error) => {{
  console.error(error.stack || String(error));
  process.exit(1);
}});
'''
    path = workspace / 'workflow-test.js'
    path.write_text(harness, encoding='utf-8')
    result = subprocess.run(['node', str(path)], cwd=workspace, check=True,
                            capture_output=True, text=True)
    return json.loads(result.stdout)


def execute_reconcile(workflow: str, should_poll: str, poll_outcome: str,
                      labels: list[str] | None = None,
                      association: str = 'MEMBER', author: str = 'alice',
                      live_head: str = 'event-sha',
                      live_base: str = 'master',
                      post_label_head: str = 'event-sha',
                      post_label_base: str | None = None,
                      post_label_state: str = 'open',
                      post_label_labels: list[str] | None = None,
                      prior_statuses: list | None = None,
                      action: str = 'opened',
                      changes_base_from: str = '') -> list:
    script = extract_step_script(
        workflow, 'Reconcile ci-passed status and informational label')
    script = script.replace(
        '${{ steps.eligibility.outputs.should_poll }}', should_poll)
    script = script.replace('${{ steps.poll.outcome }}', poll_outcome)
    labels = labels or []
    post_label_labels = labels if post_label_labels is None else post_label_labels
    post_label_base = live_base if post_label_base is None else post_label_base
    prior_statuses = [] if prior_statuses is None else prior_statuses
    setup = f'''
const context = {{
  repo: {{ owner: "kubeflow", repo: "pipelines" }},
  payload: {{
    action: {json.dumps(action)},
    changes: {{ base: {{ ref: {{ from: {json.dumps(changes_base_from)} }} }} }},
    pull_request: {{
      number: 7, head: {{ sha: "event-sha" }}, base: {{ ref: "master" }}
    }},
  }},
}};
let pullReads = 0;
const github = {{ rest: {{
  pulls: {{ get: async () => {{
    pullReads += 1;
    const isFirst = pullReads === 1;
    return {{ data: {{
      number: 7, state: isFirst ? "open" : {json.dumps(post_label_state)},
      head: {{ sha: isFirst ? {json.dumps(live_head)} : {json.dumps(post_label_head)} }},
      base: {{ ref: isFirst ? {json.dumps(live_base)} : {json.dumps(post_label_base)} }},
      labels: isFirst ? {json.dumps([{'name': label} for label in labels])} : {json.dumps([{'name': label} for label in post_label_labels])},
      user: {{ login: {json.dumps(author)} }},
      author_association: {json.dumps(association)},
    }} }};
  }} }},
  repos: {{
    createCommitStatus: async (options) => {{
      calls.push(["status", options.state, options.sha]);
      return {{ data: options }};
    }},
    listCommitStatusesForRef: async () => {{
      return {{ data: {json.dumps(prior_statuses)} }};
    }},
  }},
  issues: {{
    addLabels: async () => calls.push(["add-label"]),
    removeLabel: async () => calls.push(["remove-label"]),
  }},
}} }};
'''
    with tempfile.TemporaryDirectory() as directory:
        return execute_javascript(script, setup, Path(directory))


def execute_pending(workflow: str, prior_statuses: list | None = None,
                    head_sha: str = 'event-sha') -> list:
    script = extract_step_script(
        workflow, 'Mark ci-passed pending while current head is revalidated',
        'Wait for action_required workflow runs to be approved')
    prior_statuses = [] if prior_statuses is None else prior_statuses
    setup = f'''
const context = {{
  repo: {{ owner: "kubeflow", repo: "pipelines" }},
  payload: {{ pull_request: {{ head: {{ sha: {json.dumps(head_sha)} }} }} }},
}};
const github = {{ rest: {{
  repos: {{
    createCommitStatus: async (options) => {{
      calls.push(["status", options.state, options.sha]);
      return {{ data: options }};
    }},
    listCommitStatusesForRef: async () => {{
      return {{ data: {json.dumps(prior_statuses)} }};
    }},
  }},
}} }};
'''
    with tempfile.TemporaryDirectory() as directory:
        return execute_javascript(script, setup, Path(directory))


def execute_workflow_run_resolve(workflow: str, pulls: list,
                                 head_sha: str = 'head-sha',
                                 head_repo_owner: str = 'kubeflow',
                                 head_branch: str = 'dependabot-ci-passed-eligibility') -> dict:
    script = extract_step_script(
        workflow, 'Resolve pull request for workflow_run head SHA',
        'Read current ci-passed status for the workflow_run head SHA')
    setup = f'''
const context = {{
  repo: {{ owner: "kubeflow", repo: "pipelines" }},
  payload: {{ workflow_run: {{
    head_sha: {json.dumps(head_sha)},
    head_repository: {{ owner: {{ login: {json.dumps(head_repo_owner)} }} }},
    head_branch: {json.dumps(head_branch)},
  }} }},
}};
const github = {{ rest: {{
  pulls: {{ list: async () => {{ return {{ data: {json.dumps(pulls)} }}; }} }},
}} }};
'''
    with tempfile.TemporaryDirectory() as directory:
        calls = execute_javascript(script, setup, Path(directory))
    return {call[1]: call[2] for call in calls if call[0] == 'setOutput'}


def execute_workflow_run_status(workflow: str, conclusion: str,
                                prior_statuses: list | None = None) -> dict:
    script = extract_step_script(
        workflow, 'Read current ci-passed status for the workflow_run head SHA',
        'Re-poll all CI checks for the workflow_run head SHA')
    prior_statuses = [] if prior_statuses is None else prior_statuses
    setup = f'''
const context = {{
  repo: {{ owner: "kubeflow", repo: "pipelines" }},
  payload: {{ workflow_run: {{
    head_sha: "head-sha",
    conclusion: {json.dumps(conclusion)},
  }} }},
}};
const github = {{ rest: {{
  repos: {{ listCommitStatusesForRef: async () => {{
    return {{ data: {json.dumps(prior_statuses)} }};
  }} }},
}} }};
'''
    with tempfile.TemporaryDirectory() as directory:
        calls = execute_javascript(script, setup, Path(directory))
    return {call[1]: call[2] for call in calls if call[0] == 'setOutput'}


def execute_workflow_run_reconcile(workflow: str, matched: str,
                                   pr_number: str, pr_base_ref: str,
                                   repoll_outcome: str,
                                   conclusion: str = 'failure',
                                   live_state: str = 'open',
                                   live_head: str = 'head-sha',
                                   live_base: str = 'master',
                                   prior_statuses: list | None = None,
                                   labels: list[str] | None = None,
                                   author: str = 'dependabot[bot]',
                                   association: str = 'NONE') -> list:
    script = extract_step_script(
        workflow, 'Reconcile ci-passed for the workflow_run head SHA')
    script = script.replace('${{ steps.resolve.outputs.matched }}', matched)
    script = script.replace('${{ steps.resolve.outputs.pr_number }}', pr_number)
    script = script.replace('${{ steps.resolve.outputs.pr_base_ref }}', pr_base_ref)
    script = script.replace('${{ steps.repoll.outcome }}', repoll_outcome)
    prior_statuses = [] if prior_statuses is None else prior_statuses
    labels = labels or []
    setup = f'''
const context = {{
  repo: {{ owner: "kubeflow", repo: "pipelines" }},
  payload: {{ workflow_run: {{
    head_sha: "head-sha",
    conclusion: {json.dumps(conclusion)},
  }} }},
}};
const github = {{ rest: {{
  pulls: {{ get: async () => {{ return {{ data: {{
    number: 7, state: {json.dumps(live_state)},
    head: {{ sha: {json.dumps(live_head)} }},
    base: {{ ref: {json.dumps(live_base)} }},
    labels: {json.dumps([{'name': label} for label in labels])}, user: {{ login: {json.dumps(author)} }}, author_association: {json.dumps(association)},
  }} }}; }} }},
  repos: {{ createCommitStatus: async (options) => {{
    calls.push(["status", options.state, options.sha]);
    return {{ data: options }};
  }}, listCommitStatusesForRef: async () => {{
    return {{ data: {json.dumps(prior_statuses)} }};
  }} }},
  issues: {{
    addLabels: async () => calls.push(["add-label"]),
    removeLabel: async () => calls.push(["remove-label"]),
  }},
}} }};
'''
    with tempfile.TemporaryDirectory() as directory:
        return execute_javascript(script, setup, Path(directory))


class CiPassEligibilityTest(unittest.TestCase):

    @classmethod
    def setUpClass(cls):
        cls.workflow = CI_CHECKS_PATH.read_text(encoding='utf-8')

    def test_eligibility_truth_table_executes_real_shell(self):
        for name, (author, association) in AUTHORS.items():
            for has_ok in (False, True):
                for has_needs in (False, True):
                    with self.subTest(author=name, ok=has_ok,
                                      needs=has_needs):
                        self.assertEqual(
                            execute_eligibility(self.workflow, author,
                                                association, has_ok, has_needs),
                            expected_eligible(author, association, has_ok,
                                              has_needs))

    def test_base_retarget_executes_failure_path_until_synchronize(self):
        self.assertFalse(
            execute_eligibility(self.workflow, 'alice', 'MEMBER', False,
                                False, action='edited', old_base='master'))
        self.assertTrue(
            execute_eligibility(self.workflow, 'alice', 'MEMBER', False,
                                False, action='synchronize'))

    def test_needs_ok_to_test_precedes_all_success_branches(self):
        script = extract_step_script(
            self.workflow, 'Determine whether CI polling is needed',
            'Mark ci-passed pending while current head is revalidated')
        needs = script.index("'needs-ok-to-test'")
        self.assertLess(needs, script.index("'ok-to-test'"))
        self.assertLess(needs, script.index('dependabot[bot]'))
        self.assertLess(needs, script.index('AUTHOR_ASSOCIATION'))

    def test_single_writer_concurrency_and_permissions(self):
        self.assertFalse(ADD_LABEL_PATH.exists())
        self.assertIn(
            'group: ci-passed-reconcile-${{ github.event.workflow_run.head_sha || github.event.pull_request.head.sha }}',
            self.workflow)
        self.assertIn('cancel-in-progress: false', self.workflow)
        self.assertIn('statuses: write', self.workflow)
        self.assertIn('issues: write', self.workflow)

    def test_reconcile_executes_success_and_failure_transitions(self):
        success = execute_reconcile(self.workflow, 'true', 'success')
        self.assertIn(['status', 'success', 'event-sha'], success)
        self.assertIn(['add-label'], success)

        for should_poll, outcome, labels, association in (
                ('true', 'failure', [], 'MEMBER'),
                ('false', 'skipped', ['needs-ok-to-test'], 'MEMBER'),
                ('false', 'skipped', [], 'CONTRIBUTOR')):
            with self.subTest(should_poll=should_poll, outcome=outcome,
                              labels=labels, association=association):
                calls = execute_reconcile(self.workflow, should_poll, outcome,
                                          labels=labels,
                                          association=association)
                self.assertIn(['status', 'failure', 'event-sha'], calls)
                self.assertIn(['remove-label'], calls)

    def test_stale_head_cannot_publish(self):
        calls = execute_reconcile(self.workflow, 'true', 'success',
                                  live_head='new-sha')
        self.assertFalse(any(call[0] == 'status' for call in calls))
        self.assertNotIn(['add-label'], calls)

    def test_base_change_during_reconciliation_cannot_publish(self):
        calls = execute_reconcile(self.workflow, 'true', 'success',
                                  live_base='release-2.17')
        self.assertFalse(any(call[0] == 'status' for call in calls))
        self.assertNotIn(['add-label'], calls)

    def test_head_move_during_label_add_is_reconciled(self):
        calls = execute_reconcile(self.workflow, 'true', 'success',
                                  post_label_head='new-sha')
        self.assertIn(['add-label'], calls)
        self.assertIn(['remove-label'], calls)

    def test_only_relevant_events_enter_the_writer(self):
        condition = self.workflow[
            self.workflow.index('    if: >-'):
            self.workflow.index('    concurrency:')]
        self.assertIn("label.name == 'needs-ok-to-test'", condition)
        self.assertIn("label.name == 'ok-to-test'", condition)
        self.assertIn('github.event.changes.base.ref.from', condition)
        self.assertIn('types: [opened, synchronize, reopened, edited, labeled, unlabeled]',
                      self.workflow)

    def test_workflow_run_trigger_declared_without_check_run_suite(self):
        self.assertIn('workflow_run:', self.workflow)
        self.assertIn('types: [completed]', self.workflow)
        self.assertNotIn('check_run:', self.workflow)
        self.assertNotIn('check_suite:', self.workflow)

    def test_workflow_run_resolves_pr_by_head_sha(self):
        # Open PR whose head matches the event SHA.
        outputs = execute_workflow_run_resolve(self.workflow, [
            {'number': 7, 'head': {'sha': 'head-sha'},
             'base': {'ref': 'master'}},
        ])
        self.assertEqual(outputs['matched'], 'true')
        self.assertEqual(outputs['pr_number'], '7')
        self.assertEqual(outputs['pr_base_ref'], 'master')

        # Stale SHA: the PR head has moved past the event SHA.
        outputs = execute_workflow_run_resolve(self.workflow, [
            {'number': 7, 'head': {'sha': 'new-sha'},
             'base': {'ref': 'master'}},
        ])
        self.assertEqual(outputs['matched'], 'false')
        self.assertEqual(outputs['pr_number'], '')

        # No open PR at this head (direct push / already merged).
        outputs = execute_workflow_run_resolve(self.workflow, [])
        self.assertEqual(outputs['matched'], 'false')
        self.assertEqual(outputs['pr_number'], '')

    def test_workflow_run_rerun_failure_flips_red(self):
        calls = execute_workflow_run_reconcile(
            self.workflow, matched='true', pr_number='7', pr_base_ref='master',
            repoll_outcome='failure', conclusion='failure')
        self.assertIn(['status', 'failure', 'head-sha'], calls)
        self.assertIn(['remove-label'], calls)

    def test_workflow_run_rerun_success_keeps_green(self):
        calls = execute_workflow_run_reconcile(
            self.workflow, matched='true', pr_number='7', pr_base_ref='master',
            repoll_outcome='success', conclusion='failure')
        self.assertIn(['status', 'success', 'head-sha'], calls)
        self.assertIn(['add-label'], calls)

    def test_workflow_run_stale_sha_is_noop(self):
        calls = execute_workflow_run_reconcile(
            self.workflow, matched='false', pr_number='', pr_base_ref='',
            repoll_outcome='skipped', conclusion='failure')
        self.assertFalse(any(call[0] == 'status' for call in calls))

    def test_workflow_run_success_conclusion_already_green_is_noop(self):
        # A success conclusion on an already-green head has nothing to
        # re-derive and is a no-op.
        calls = execute_workflow_run_reconcile(
            self.workflow, matched='true', pr_number='7', pr_base_ref='master',
            repoll_outcome='skipped', conclusion='success',
            prior_statuses=[{'context': 'ci-passed', 'state': 'success'}])
        self.assertFalse(any(call[0] == 'status' for call in calls))
        self.assertNotIn(['add-label'], calls)
        self.assertNotIn(['remove-label'], calls)

    def test_workflow_run_status_only_failures_force_repoll(self):
        for conclusion in ('failure', 'timed_out'):
            with self.subTest(conclusion=conclusion):
                outputs = execute_workflow_run_status(self.workflow, conclusion)
                self.assertEqual(outputs['needs_repoll'], 'true')
        for conclusion in ('cancelled', 'skipped', 'neutral', 'action_required'):
            with self.subTest(conclusion=conclusion):
                outputs = execute_workflow_run_status(self.workflow, conclusion)
                self.assertEqual(outputs['needs_repoll'], 'false')

    def test_workflow_run_status_success_repolls_on_absent_pending(self):
        cases = (
            ([], 'true'),
            ([{'context': 'ci-passed', 'state': 'pending'}], 'true'),
            ([{'context': 'ci-passed', 'state': 'failure'}], 'true'),
            ([{'context': 'ci-passed', 'state': 'success'}], 'false'),
        )
        for prior_statuses, expected in cases:
            with self.subTest(prior_statuses=prior_statuses):
                outputs = execute_workflow_run_status(
                    self.workflow, 'success', prior_statuses=prior_statuses)
                self.assertEqual(outputs['needs_repoll'], expected)

    def test_workflow_run_cancelled_conclusion_does_not_redden(self):
        # An eligible, already-green head must not be reddened by a cancelled
        # re-run: the completion is not a check result.
        calls = execute_workflow_run_reconcile(
            self.workflow, matched='true', pr_number='7', pr_base_ref='master',
            repoll_outcome='skipped', conclusion='cancelled',
            prior_statuses=[{'context': 'ci-passed', 'state': 'success'}],
            labels=['ok-to-test'], author='alice', association='MEMBER')
        self.assertFalse(any(call[0] == 'status' for call in calls))
        self.assertNotIn(['remove-label'], calls)

    def test_workflow_run_success_conclusion_repolls_absent_or_pending(self):
        # A success completion with an absent or pending current status must
        # re-derive and publish, not leave the head stuck at pending.
        for prior in ([], [{'context': 'ci-passed', 'state': 'pending'}]):
            with self.subTest(prior=prior):
                calls = execute_workflow_run_reconcile(
                    self.workflow, matched='true', pr_number='7',
                    pr_base_ref='master', repoll_outcome='success',
                    conclusion='success', prior_statuses=prior)
                self.assertIn(['status', 'success', 'head-sha'], calls)
                self.assertIn(['add-label'], calls)

    def test_workflow_run_ineligible_never_flips_success(self):
        # needs-ok-to-test rejects unconditionally, even with a green re-poll.
        for labels in (['needs-ok-to-test'], ['needs-ok-to-test', 'ok-to-test']):
            with self.subTest(labels=labels):
                calls = execute_workflow_run_reconcile(
                    self.workflow, matched='true', pr_number='7',
                    pr_base_ref='master', repoll_outcome='success',
                    conclusion='failure', labels=labels,
                    author='alice', association='MEMBER')
                self.assertIn(['status', 'failure', 'head-sha'], calls)
                self.assertNotIn(['status', 'success', 'head-sha'], calls)
                self.assertIn(['remove-label'], calls)
                self.assertNotIn(['add-label'], calls)

        # An untrusted CONTRIBUTOR author with no ok-to-test is ineligible.
        calls = execute_workflow_run_reconcile(
            self.workflow, matched='true', pr_number='7', pr_base_ref='master',
            repoll_outcome='success', conclusion='failure',
            labels=[], author='bob', association='CONTRIBUTOR')
        self.assertIn(['status', 'failure', 'head-sha'], calls)
        self.assertNotIn(['status', 'success', 'head-sha'], calls)
        self.assertIn(['remove-label'], calls)
        self.assertNotIn(['add-label'], calls)

    def test_workflow_run_ineligible_after_green_leaves_failure(self):
        # needs-ok-to-test added after a green: any workflow_run completion
        # must leave the gate failure and drop the label.
        for conclusion in ('success', 'failure'):
            with self.subTest(conclusion=conclusion):
                calls = execute_workflow_run_reconcile(
                    self.workflow, matched='true', pr_number='7',
                    pr_base_ref='master', repoll_outcome='success',
                    conclusion=conclusion,
                    prior_statuses=[{'context': 'ci-passed',
                                     'state': 'success'}],
                    labels=['needs-ok-to-test'],
                    author='alice', association='MEMBER')
                self.assertIn(['status', 'failure', 'head-sha'], calls)
                self.assertNotIn(['status', 'success', 'head-sha'], calls)
                self.assertIn(['remove-label'], calls)
                self.assertNotIn(['add-label'], calls)

    def test_workflow_run_never_publishes_onto_live_head(self):
        # The event SHA no longer matches the live PR head: refuse to publish.
        calls = execute_workflow_run_reconcile(
            self.workflow, matched='true', pr_number='7', pr_base_ref='master',
            repoll_outcome='failure', conclusion='failure',
            live_head='new-sha')
        self.assertFalse(any(call[0] == 'status' for call in calls))

    def test_workflow_run_base_retarget_blocks_success_restore(self):
        # A base retarget left a persisted revalidation marker on the head SHA.
        # A non-success workflow_run completion whose re-poll is green must not
        # overwrite the marker with success.
        calls = execute_workflow_run_reconcile(
            self.workflow, matched='true', pr_number='7', pr_base_ref='master',
            repoll_outcome='success', conclusion='failure',
            prior_statuses=[{'context': 'ci-passed', 'state': 'failure',
                             'description': 'Base branch retargeted; CI revalidation is required.'}])
        self.assertIn(['status', 'failure', 'head-sha'], calls)
        self.assertNotIn(['status', 'success', 'head-sha'], calls)
        self.assertIn(['remove-label'], calls)
        self.assertNotIn(['add-label'], calls)

    def test_workflow_run_success_conclusion_clears_stale_red(self):
        # A success-conclusion workflow_run with a current ci-passed failure
        # status re-polls and re-derives success, clearing the stale red.
        calls = execute_workflow_run_reconcile(
            self.workflow, matched='true', pr_number='7', pr_base_ref='master',
            repoll_outcome='success', conclusion='success',
            prior_statuses=[{'context': 'ci-passed', 'state': 'failure',
                             'description': 'CI checks did not pass or the PR requires revalidation.'}])
        self.assertIn(['status', 'success', 'head-sha'], calls)
        self.assertIn(['add-label'], calls)

    def test_base_retarget_event_publishes_revalidation_marker(self):
        # A base change event must publish a persistent revalidation marker.
        calls = execute_reconcile(
            self.workflow, 'false', 'skipped',
            labels=['ok-to-test'], association='MEMBER',
            action='edited', changes_base_from='master')
        self.assertIn(['status', 'failure', 'event-sha'], calls)
        self.assertNotIn(['status', 'success', 'event-sha'], calls)

    def test_base_retarget_persistence_blocks_label_restore(self):
        # A base retarget left a persisted revalidation marker on the head SHA.
        # A later ok-to-test label on the unchanged head must not restore success.
        calls = execute_reconcile(
            self.workflow, 'true', 'success',
            labels=['ok-to-test'], association='MEMBER',
            prior_statuses=[{'context': 'ci-passed', 'state': 'failure',
                             'description': 'Base branch retargeted; CI revalidation is required.'}])
        self.assertIn(['status', 'failure', 'event-sha'], calls)
        self.assertNotIn(['status', 'success', 'event-sha'], calls)
        self.assertIn(['remove-label'], calls)
        self.assertNotIn(['add-label'], calls)

    def test_marker_pending_step_preserves_base_retarget_marker(self):
        # The pending step must not overwrite the base-retarget revalidation
        # marker with a pending status; otherwise a label/reopen event could
        # re-earn success without revalidating CI against the new base (I1).
        marker = [{'context': 'ci-passed', 'state': 'failure',
                   'description': 'Base branch retargeted; CI revalidation is required.'}]
        calls = execute_pending(self.workflow, prior_statuses=marker)
        self.assertFalse(any(call[0] == 'status' for call in calls))

    def test_marker_pending_step_writes_pending_without_marker(self):
        # Without a marker (fresh head), the pending step still writes pending.
        calls = execute_pending(self.workflow, prior_statuses=[])
        self.assertIn(['status', 'pending', 'event-sha'], calls)

    def test_base_retarget_label_event_repolls_but_publishes_failure(self):
        # After the marker is set, an ok-to-test label on the unchanged head
        # must re-poll (should_poll=true) yet still publish failure: the
        # pending step preserves the marker and reconcile re-publishes it.
        marker = [{'context': 'ci-passed', 'state': 'failure',
                   'description': 'Base branch retargeted; CI revalidation is required.'}]
        pending_calls = execute_pending(self.workflow, prior_statuses=marker)
        self.assertFalse(any(call[0] == 'status' for call in pending_calls))
        calls = execute_reconcile(
            self.workflow, 'true', 'success',
            labels=['ok-to-test'], association='MEMBER', action='labeled',
            prior_statuses=marker)
        self.assertIn(['status', 'failure', 'event-sha'], calls)
        self.assertNotIn(['status', 'success', 'event-sha'], calls)
        self.assertNotIn(['add-label'], calls)

    def test_base_retarget_reopen_repolls_but_publishes_failure(self):
        # A trusted author reopening the retargeted PR must re-poll yet still
        # publish failure; the marker is preserved on the unchanged head SHA.
        marker = [{'context': 'ci-passed', 'state': 'failure',
                   'description': 'Base branch retargeted; CI revalidation is required.'}]
        pending_calls = execute_pending(self.workflow, prior_statuses=marker)
        self.assertFalse(any(call[0] == 'status' for call in pending_calls))
        calls = execute_reconcile(
            self.workflow, 'true', 'success',
            labels=[], association='MEMBER', author='alice', action='reopened',
            prior_statuses=marker)
        self.assertIn(['status', 'failure', 'event-sha'], calls)
        self.assertNotIn(['status', 'success', 'event-sha'], calls)
        self.assertNotIn(['add-label'], calls)

    def test_base_retarget_synchronize_clears_requirement(self):
        # A synchronize moves the head to a new SHA that carries no marker, so
        # the pending step writes pending and CI may re-earn success on it.
        pending_calls = execute_pending(self.workflow, prior_statuses=[])
        self.assertIn(['status', 'pending', 'event-sha'], pending_calls)
        calls = execute_reconcile(
            self.workflow, 'true', 'success',
            labels=['ok-to-test'], association='MEMBER', action='synchronize',
            prior_statuses=[])
        self.assertIn(['status', 'success', 'event-sha'], calls)
        self.assertIn(['add-label'], calls)

    def test_final_publish_race_publishes_failure(self):
        # The post-mutation snapshot no longer matches (base retargeted during
        # reconciliation) -> an explicit failure is published.
        calls = execute_reconcile(
            self.workflow, 'true', 'success',
            labels=['ok-to-test'], association='MEMBER',
            post_label_base='release-2.17')
        self.assertIn(['add-label'], calls)
        self.assertIn(['remove-label'], calls)
        self.assertIn(['status', 'failure', 'event-sha'], calls)


if __name__ == '__main__':
    unittest.main()
