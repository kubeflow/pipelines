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
    end = (workflow.index(f'      - name: {next_step_name}', start)
           if next_step_name else len(workflow))
    step = workflow[start:end]
    marker = 'run: |' if 'run: |' in step else 'script: |'
    return textwrap.dedent(step.split(marker, 1)[1]).strip()


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
const core = {{ info: (message) => calls.push(["info", String(message)]) }};
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
                      post_label_head: str = 'event-sha') -> list:
    script = extract_step_script(
        workflow, 'Reconcile ci-passed status and informational label')
    script = script.replace(
        '${{ steps.eligibility.outputs.should_poll }}', should_poll)
    script = script.replace('${{ steps.poll.outcome }}', poll_outcome)
    labels = labels or []
    setup = f'''
const context = {{
  repo: {{ owner: "kubeflow", repo: "pipelines" }},
  payload: {{ pull_request: {{
    number: 7, head: {{ sha: "event-sha" }}, base: {{ ref: "master" }}
  }} }},
}};
let pullReads = 0;
const github = {{ rest: {{
  pulls: {{ get: async () => {{
    pullReads += 1;
    return {{ data: {{
      number: 7, state: "open",
      head: {{ sha: pullReads === 1 ? {json.dumps(live_head)} : {json.dumps(post_label_head)} }},
      base: {{ ref: {json.dumps(live_base)} }},
      labels: {json.dumps([{'name': label} for label in labels])},
      user: {{ login: {json.dumps(author)} }},
      author_association: {json.dumps(association)},
    }} }};
  }} }},
  repos: {{ createCommitStatus: async (options) => {{
    calls.push(["status", options.state, options.sha]);
    return {{ data: options }};
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
            'group: ${{ github.workflow }}-${{ github.event.pull_request.number }}-${{ github.event.pull_request.head.sha }}',
            self.workflow)
        self.assertIn('cancel-in-progress: true', self.workflow)
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


if __name__ == '__main__':
    unittest.main()
