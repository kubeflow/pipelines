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

import os
from pathlib import Path
import re
import subprocess
import tempfile
import textwrap
from typing import Optional
import unittest

ROOT = Path(__file__).resolve().parents[3]
CI_CHECKS_PATH = ROOT / '.github/workflows/ci-checks.yml'
ADD_LABEL_PATH = ROOT / '.github/workflows/add-ci-passed-label.yml'
CI_SCRIPTS_TESTS_PATH = ROOT / '.github/workflows/ci-scripts-tests.yml'

# The author dimension of the truth table: each author is a (login,
# author_association) pair. Dependabot's exact bot identity is special-cased
# by login; every other author is gated by their association (trusted
# contributors) or by an explicit ok-to-test label. A different automation
# bot (renovate) is a CONTRIBUTOR and must NOT be treated as Dependabot.
AUTHORS = {
    'human_member': ('alice', 'MEMBER'),
    'human_contributor': ('bob', 'CONTRIBUTOR'),
    'dependabot': ('dependabot[bot]', 'NONE'),
    'other_bot': ('renovate[bot]', 'CONTRIBUTOR'),
}

# GitHub author_association values that are trusted to run CI without an
# explicit ok-to-test label.
TRUSTED_ASSOCIATIONS = {'MEMBER', 'OWNER', 'COLLABORATOR'}


def expected_eligible(author: str, association: str, has_ok_to_test: bool,
                      has_needs_ok_to_test: bool) -> bool:
    # The shared predicate both workflows must agree on: a PR is eligible for
    # CI polling / ci-passed only when it carries ok-to-test, is authored by
    # dependabot[bot], or is authored by a trusted contributor (MEMBER/OWNER/
    # COLLABORATOR) — AND it does not carry the blocking needs-ok-to-test label.
    return (not has_needs_ok_to_test) and (
        has_ok_to_test or author == 'dependabot[bot]' or
        association in TRUSTED_ASSOCIATIONS)


def extract_ci_checks_eligibility_script(workflow: str) -> str:
    start = workflow.index(
        '      - name: Determine whether CI polling is needed')
    end = workflow.index(
        '      - name: Wait for action_required workflow runs to be approved')
    step = workflow[start:end]
    return textwrap.dedent(step.split('run: |', 1)[1]).strip()


def ci_checks_should_poll(workflow: str, author: str, association: str,
                          has_ok_to_test: bool,
                          has_needs_ok_to_test: bool) -> bool:
    """Execute the real eligibility shell script for one truth-table row."""
    script = extract_ci_checks_eligibility_script(workflow)
    script = script.replace(
        "${{ contains(github.event.pull_request.labels.*.name, 'ok-to-test') }}",
        'true' if has_ok_to_test else 'false',
    )
    script = script.replace(
        "${{ contains(github.event.pull_request.labels.*.name, 'needs-ok-to-test') }}",
        'true' if has_needs_ok_to_test else 'false',
    )

    with tempfile.TemporaryDirectory() as temporary_directory:
        output_path = Path(temporary_directory) / 'output'
        env = {
            **os.environ,
            'PR_AUTHOR': author,
            'AUTHOR_ASSOCIATION': association,
            'EVENT_ACTION': 'opened',
            'LABEL_NAME': '',
            'GITHUB_OUTPUT': str(output_path),
        }
        subprocess.run(
            ['bash', '-c', script],
            env=env,
            check=True,
            capture_output=True,
            text=True,
        )
        output = output_path.read_text(encoding='utf-8')

    values = dict(
        line.split('=', 1) for line in output.splitlines() if '=' in line)
    return values.get('should_poll') == 'true'


def extract_js_eligible_expression(workflow: str) -> str:
    match = re.search(r'const eligible = (.*);', workflow)
    if match is None:
        raise AssertionError('eligible expression not found in workflow')
    return match.group(1)


def js_eligible(expression: str, author: str, association: str,
                labels: set[str]) -> bool:
    """Evaluate the extracted github-script eligibility expression in Python."""
    python_expression = re.sub(
        r'labels\.has\("([^"]+)"\)', r'("\1" in labels)', expression)
    python_expression = python_expression.replace('pullRequest.user.login',
                                                  'author')
    python_expression = python_expression.replace('authorAssociation',
                                                  'association')
    python_expression = python_expression.replace('===', '==')
    python_expression = python_expression.replace('&&', ' and ')
    python_expression = python_expression.replace('||', ' or ')
    python_expression = python_expression.replace('!', ' not ')
    return bool(
        eval(python_expression, {
            'author': author,
            'labels': labels,
            'association': association,
        }))


def _labels(has_ok_to_test: bool, has_needs_ok_to_test: bool) -> set[str]:
    labels = set()
    if has_ok_to_test:
        labels.add('ok-to-test')
    if has_needs_ok_to_test:
        labels.add('needs-ok-to-test')
    return labels


class CiPassEligibilityTest(unittest.TestCase):

    @classmethod
    def setUpClass(cls):
        cls.ci_checks = CI_CHECKS_PATH.read_text(encoding='utf-8')
        cls.add_label = ADD_LABEL_PATH.read_text(encoding='utf-8')
        cls.ci_scripts_tests = CI_SCRIPTS_TESTS_PATH.read_text(encoding='utf-8')

    def truth_table_rows(self):
        for author_name, (author, association) in AUTHORS.items():
            for has_ok_to_test in (False, True):
                for has_needs_ok_to_test in (False, True):
                    yield (author_name, author, association, has_ok_to_test,
                           has_needs_ok_to_test)

    def test_ci_checks_eligibility_truth_table(self):
        for author_name, author, association, has_ok, has_needs in (
                self.truth_table_rows()):
            with self.subTest(author=author_name,
                              ok_to_test=has_ok,
                              needs_ok_to_test=has_needs):
                self.assertEqual(
                    ci_checks_should_poll(self.ci_checks, author, association,
                                          has_ok, has_needs),
                    expected_eligible(author, association, has_ok, has_needs),
                )

    def test_add_label_eligibility_truth_table(self):
        expression = extract_js_eligible_expression(self.add_label)
        for author_name, author, association, has_ok, has_needs in (
                self.truth_table_rows()):
            with self.subTest(author=author_name,
                              ok_to_test=has_ok,
                              needs_ok_to_test=has_needs):
                self.assertEqual(
                    js_eligible(expression, author, association,
                                _labels(has_ok, has_needs)),
                    expected_eligible(author, association, has_ok, has_needs),
                )

    def test_add_label_workflow_author_predicate_is_narrow(self):
        expression = extract_js_eligible_expression(self.add_label)
        # Only Dependabot's exact login bypasses the label/association gates;
        # other automation bots must earn eligibility like any contributor.
        self.assertIn('pullRequest.user.login === "dependabot[bot]"', expression)
        self.assertIn('labels.has("ok-to-test")', expression)
        self.assertIn('labels.has("needs-ok-to-test")', expression)
        self.assertIn('authorAssociation === "MEMBER"', expression)
        self.assertIn('authorAssociation === "OWNER"', expression)
        self.assertIn('authorAssociation === "COLLABORATOR"', expression)

    def test_ci_checks_eligibility_branch_order(self):
        # Scoped to the EXECUTABLE block (not the outer job filter): the
        # blocking needs-ok-to-test check must precede the ok-to-test branch,
        # the dependabot branch, and the trusted-association branch, otherwise
        # a blocked PR could still be polled.
        script = extract_ci_checks_eligibility_script(self.ci_checks)
        needs = script.index("'needs-ok-to-test'")
        ok = script.index("'ok-to-test'")
        dependabot = script.index('dependabot[bot]')
        association = script.index('AUTHOR_ASSOCIATION')
        self.assertLess(needs, ok)
        self.assertLess(needs, dependabot)
        self.assertLess(needs, association)

    def test_add_job_revalidates_head_immediately_before_label_mutation(self):
        workflow = self.add_label

        # fetch_data threads the verified head SHA to the mutating job.
        self.assertIn('head_sha: ${{ steps.ci_result.outputs.head_sha }}',
                      workflow)
        # ci_result records the head SHA it validated against the live PR.
        self.assertIn('core.setOutput("head_sha", recordedHeadSha);', workflow)

        # The add job re-reads the PR head and refuses to attach ci-passed to
        # a moved head (closes the validate-then-publish synchronize race).
        add_job = workflow[workflow.index('  add_ci_passed_label:'):]
        self.assertIn('recorded_head_sha', add_job)
        self.assertIn('gh pr view', add_job)
        self.assertIn('--json headRefOid', add_job)
        self.assertIn('current_head_sha', add_job)
        self.assertLess(add_job.index('gh pr view'), add_job.index('gh pr edit'))
        self.assertLess(add_job.index('head moved from'),
                        add_job.index('--add-label'))
        # The post-mutation reconcile re-reads the head and strips the label
        # again if it moved during the add-label call itself.
        self.assertIn('post_head_sha', add_job)
        self.assertIn('--remove-label', add_job)

    def test_state_machine_structural_pins(self):
        workflow = self.add_label

        # Success and failure commit-status paths exist, bound to the verified
        # head SHA and the same "ci-passed" context.
        self.assertIn('createCommitStatus', workflow)
        self.assertIn('context: "ci-passed"', workflow)
        self.assertIn('state: "success"', workflow)
        self.assertIn('state: "failure"', workflow)
        self.assertIn('CI checks did not pass on this commit.', workflow)

        # Workflow-level concurrency serialises runs for the same PR.
        self.assertIn('concurrency:', workflow)
        self.assertIn('group: ci-passed-', workflow)
        self.assertIn('cancel-in-progress: false', workflow)

        # pull_request_target must react to base-ref edits.
        on_block = workflow[workflow.index('pull_request_target:'):
                            workflow.index('workflow_run:')]
        self.assertIn('- edited', on_block)

        # The reset job invalidates on base-ref changes as well as the other
        # triggers.
        reset_job = workflow[workflow.index('  reset_ci_passed_label:'):
                             workflow.index('  add_ci_passed_label:')]
        self.assertIn('github.event.changes.base.ref', reset_job)
        self.assertIn('state=failure', reset_job)
        self.assertIn('context=ci-passed', reset_job)

    def test_add_and_reset_interleaving_preserves_invariant(self):
        """Model the add + reset jobs and inject a synchronize (head move) at
        every gap between their production API calls.

        Invariant: no 'ci-passed' success label is attached to a head that did
        not pass CI. The add job's re-check and reconcile handle a move that
        lands before its reconcile read; the reset job (triggered by the same
        synchronize) publishes a failure status and strips the label, closing
        the residual window after the reconcile read.
        """
        passed = 'sha-passed'
        moved = 'sha-moved'

        class PR:
            label: Optional[str]

            def __init__(self):
                self.head = passed
                self.label = None  # 'ci-passed' when the success label is on

            def invariant(self):
                return self.label != 'ci-passed' or self.head == passed

        def run_add_job(sync_at):
            pr = PR()
            recorded = passed

            def synchronize():
                pr.head = moved

            # Gap 0: synchronize before the first read.
            if sync_at == 0:
                synchronize()
            current = pr.head
            # Verify: abort the add if the head already moved.
            if current != recorded:
                return pr
            # Gap 1: synchronize between verify and mutate.
            if sync_at == 1:
                synchronize()
            pr.label = 'ci-passed'
            # Gap 2: synchronize between mutate and the reconcile read.
            if sync_at == 2:
                synchronize()
            post = pr.head
            if post != recorded:
                pr.label = None
            # Gap 3: synchronize after the reconcile read (residual window).
            if sync_at == 3:
                synchronize()
            return pr

        def run_reset_job(pr):
            # The reset job publishes a failure status on the current head
            # (safe) and strips any stale success label (unconditional).
            pr.label = None

        for sync_at in range(4):
            with self.subTest(sync_at=sync_at):
                pr = run_add_job(sync_at)
                # Re-check + reconcile must have removed the label whenever
                # the move landed before the reconcile read.
                if sync_at in (0, 1, 2):
                    self.assertIsNone(pr.label)
                # The residual window (move after reconcile read) is closed by
                # the reset job triggered by the same synchronize.
                if pr.head == moved:
                    run_reset_job(pr)
                self.assertTrue(pr.invariant(),
                                f'invariant violated at sync_at={sync_at}')

    def test_reset_read_then_mutate_is_safe(self):
        """A synchronize landing between the reset job's read and its mutate
        still leaves no success label attached: the failure publish targets a
        head (harmless) and the label removal is unconditional."""
        passed = 'sha-passed'
        moved = 'sha-moved'

        class PR:
            label: Optional[str]

            def __init__(self):
                self.head = passed
                self.label = 'ci-passed'  # stale success label to be reset

            def invariant(self):
                return self.label != 'ci-passed' or self.head == passed

        for sync_at in range(3):
            pr = PR()

            def synchronize():
                pr.head = moved

            # Gap 0: before read; gap 1: between read and mutate; gap 2:
            # after mutate.
            if sync_at == 0:
                synchronize()
            current_head = pr.head
            if sync_at == 1:
                synchronize()
            pr.label = None  # label removal is unconditional
            if sync_at == 2:
                synchronize()

            self.assertTrue(pr.invariant(),
                            f'invariant violated at sync_at={sync_at}')

    def test_ci_scripts_tests_covers_both_workflows(self):
        self.assertIn("      - '.github/workflows/ci-checks.yml'",
                      self.ci_scripts_tests)
        self.assertIn("      - '.github/workflows/add-ci-passed-label.yml'",
                      self.ci_scripts_tests)


if __name__ == '__main__':
    unittest.main()
