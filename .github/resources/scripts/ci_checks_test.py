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

"""Truth-table regression tests for the ci-passed label pipeline.

The CI Check workflow (ci-checks.yml) decides whether to poll CI statuses,
records a ci-check-result artifact, and the Add CI Passed Label workflow
(add-ci-passed-label.yml) turns that result into a 'ci-passed' label and a
SHA-bound commit status. Both workflows are YAML with embedded bash/JS, so
these tests mirror the eligibility logic in Python and additionally pin the
structural facts that make the mirror valid (branch order, status publish,
head re-verification, path filter coverage).
"""

from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[3]
CI_CHECKS_PATH = ROOT / '.github/workflows/ci-checks.yml'
ADD_LABEL_PATH = ROOT / '.github/workflows/add-ci-passed-label.yml'
CI_SCRIPTS_PATH = ROOT / '.github/workflows/ci-scripts-tests.yml'

DEPENDABOT = 'dependabot[bot]'
ANOTHER_BOT = 'renovate[bot]'
HUMAN = 'aholten'
OK_TO_TEST = 'ok-to-test'
NEEDS_OK_TO_TEST = 'needs-ok-to-test'


def _ci_checks_should_poll(author, labels, event_action, label_name):
    """Mirror of the eligibility step in ci-checks.yml.

    Order is load-bearing: needs-ok-to-test beats everything, then
    ok-to-test, then the trusted dependabot[bot] author, then the
    labeled/unlabeled event guards, then reject.
    """
    if NEEDS_OK_TO_TEST in labels:
        return False
    if OK_TO_TEST in labels:
        return True
    if author == DEPENDABOT:
        return True
    if event_action == 'labeled' and label_name != OK_TO_TEST:
        return False
    if event_action == 'unlabeled' and label_name != NEEDS_OK_TO_TEST:
        return False
    return False


def _ci_checks_should_add_label(author, labels, event_action, label_name):
    """Mirror of the Record CI check result step (poll must also succeed)."""
    should_poll = _ci_checks_should_poll(author, labels, event_action,
                                         label_name)
    return should_poll  # poll_outcome==success is the only pass path


def _downstream_eligible(author, labels):
    """Mirror of the eligible expression in add-ci-passed-label.yml."""
    return (OK_TO_TEST in labels or author == DEPENDABOT) and \
        NEEDS_OK_TO_TEST not in labels


def _downstream_should_add_label(recorded_should_add_label, author, labels):
    """Mirror of the fetch_data job in add-ci-passed-label.yml.

    The early return on a no-op CI Check result (should_add_label != true)
    is what makes the downstream author-aware expression reachable only
    after the upstream eligibility change.
    """
    if recorded_should_add_label is not True:
        return False
    return _downstream_eligible(author, labels)


def _label_add_allowed(expected_sha, current_sha):
    """Mirror of the add_ci_passed_label job head re-verification."""
    return bool(expected_sha) and expected_sha == current_sha


class CiChecksEligibilityTest(unittest.TestCase):

    @classmethod
    def setUpClass(cls):
        cls.workflow = CI_CHECKS_PATH.read_text(encoding='utf-8')
        cls.lines = cls.workflow.splitlines()

    def _line_index(self, needle):
        for index, line in enumerate(self.lines):
            if needle in line:
                return index
        self.fail(f'expected line containing {needle!r} in ci-checks.yml')

    def test_dependabot_author_is_eligible_without_ok_to_test(self):
        self.assertTrue(
            _ci_checks_should_poll(DEPENDABOT, set(), 'opened', None))
        self.assertTrue(
            _downstream_should_add_label(True, DEPENDABOT, set()))

    def test_needs_ok_to_test_beats_author_and_label(self):
        for author in (DEPENDABOT, HUMAN):
            self.assertFalse(
                _ci_checks_should_poll(author, {NEEDS_OK_TO_TEST}, 'opened',
                                       None))
            self.assertFalse(
                _downstream_should_add_label(
                    True, author, {NEEDS_OK_TO_TEST}))
        self.assertFalse(
            _ci_checks_should_poll(HUMAN,
                                   {OK_TO_TEST, NEEDS_OK_TO_TEST}, 'opened',
                                   None))

    def test_human_requires_ok_to_test(self):
        self.assertFalse(_ci_checks_should_poll(HUMAN, set(), 'opened', None))
        self.assertFalse(
            _downstream_should_add_label(True, HUMAN, set()))
        self.assertTrue(
            _ci_checks_should_poll(HUMAN, {OK_TO_TEST}, 'opened', None))
        self.assertTrue(
            _downstream_should_add_label(True, HUMAN, {OK_TO_TEST}))

    def test_other_bots_are_not_trusted(self):
        self.assertFalse(
            _ci_checks_should_poll(ANOTHER_BOT, set(), 'opened', None))
        self.assertFalse(
            _downstream_should_add_label(True, ANOTHER_BOT, set()))

    def test_noop_ci_run_never_reaches_downstream_expression(self):
        # Jeff's blocker 1: without the upstream eligibility change the
        # artifact records should_add_label=false and the downstream
        # author-aware branch is unreachable.
        self.assertFalse(
            _downstream_should_add_label(False, DEPENDABOT, set()))

    def test_eligibility_step_author_branch_present_and_ordered(self):
        # Structural pin: needs-ok-to-test check must precede the dependabot
        # branch inside the eligibility step.
        needs_index = self._line_index("'needs-ok-to-test'")
        ok_index = self._line_index("'ok-to-test'")
        dependabot_index = self._line_index('dependabot[bot]')
        self.assertLess(needs_index, dependabot_index)
        self.assertLess(ok_index, dependabot_index)


class AddCiPassedLabelTest(unittest.TestCase):

    @classmethod
    def setUpClass(cls):
        cls.workflow = ADD_LABEL_PATH.read_text(encoding='utf-8')
        cls.lines = cls.workflow.splitlines()

    def test_dependabot_eligible_expression(self):
        self.assertIn('pullRequest.user.login === "dependabot[bot]"',
                      self.workflow)

    def test_publishes_sha_bound_commit_status(self):
        # Jeff's blocker 2: the merge gate must be a required commit status
        # on the verified head SHA, not the label.
        self.assertIn('createCommitStatus', self.workflow)
        self.assertIn('context: "ci-passed"', self.workflow)
        self.assertIn('state: "success"', self.workflow)
        self.assertIn('statuses: write', self.workflow)
        # The status step must run only after head verification.
        status_index = self._line_index('Publish')
        head_check_index = self._line_index("pullRequest.head.sha !== recordedHeadSha")
        self.assertGreater(status_index, head_check_index)

    def test_label_add_reverifies_head(self):
        # A synchronize between validation and publication must not add a
        # stale label: the add job re-reads the PR head.
        self.assertIn('headRefOid', self.workflow)
        self.assertIn('skipping stale ci-passed label', self.workflow)
        add_index = self._line_index("--add-label")
        recheck_index = self._line_index('headRefOid')
        self.assertGreater(add_index, recheck_index)

    def test_synchronize_between_validation_and_publication(self):
        # Race case: validated head A, then head moved to B before the add
        # job ran. Label must be skipped; status remains only on A.
        self.assertFalse(_label_add_allowed('sha-a', 'sha-b'))
        self.assertTrue(_label_add_allowed('sha-a', 'sha-a'))
        self.assertFalse(_label_add_allowed('', 'sha-b'))

    def test_head_sha_output_wired_through(self):
        self.assertIn('head_sha: ${{ steps.ci_result.outputs.head_sha }}',
                      self.workflow)
        self.assertIn('core.setOutput("head_sha", recordedHeadSha);',
                      self.workflow)

    def _line_index(self, needle):
        for index, line in enumerate(self.lines):
            if needle in line:
                return index
        self.fail(f'expected line containing {needle!r} in '
                  f'add-ci-passed-label.yml')


class CiScriptsTestsPathFilterTest(unittest.TestCase):

    def test_add_ci_passed_label_workflow_covered(self):
        # Jeff's ask: the path filter must include the changed workflow so
        # the truth-table coverage actually runs on this PR.
        workflow = CI_SCRIPTS_PATH.read_text(encoding='utf-8')
        self.assertIn("'.github/workflows/add-ci-passed-label.yml'", workflow)
        self.assertIn("'.github/workflows/ci-checks.yml'", workflow)


if __name__ == '__main__':
    unittest.main()
