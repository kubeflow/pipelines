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

import ast
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import tempfile
import textwrap
import unittest

ROOT = Path(__file__).resolve().parents[3]

# CI Scripts Tests intentionally runs without third-party packages. These
# helpers parse only the small YAML subset needed to locate job-level scalars.


def _mapping_marker(key: str, indentation: int) -> re.Pattern:
    return re.compile(r'^' + (' ' * indentation) + re.escape(key) +
                      r':\s*(?:#.*)?$')


def _has_mapping(document: str, key: str, indentation: int) -> bool:
    marker = _mapping_marker(key, indentation)
    return any(marker.match(line) for line in document.splitlines())


def _mapping_block(document: str, key: str, indentation: int) -> str:
    lines = document.splitlines()
    marker = _mapping_marker(key, indentation)
    start = next(
        index for index, line in enumerate(lines) if marker.match(line))

    end = len(lines)
    for index in range(start + 1, len(lines)):
        line = lines[index]
        if not line.strip() or line.lstrip().startswith('#'):
            continue
        current_indentation = len(line) - len(line.lstrip())
        if current_indentation <= indentation:
            end = index
            break

    return '\n'.join(lines[start:end]) + '\n'


def _before_mapping(document: str, key: str, indentation: int) -> str:
    lines = document.splitlines()
    marker = _mapping_marker(key, indentation)
    end = next(index for index, line in enumerate(lines) if marker.match(line))
    return '\n'.join(lines[:end]) + '\n'


def _folded_scalar(document: str, key: str, indentation: int) -> str:
    lines = document.splitlines()
    marker = re.compile(r'^' + (' ' * indentation) + re.escape(key) +
                        r':\s*[>|][-+]?\s*$')
    start = next(
        index for index, line in enumerate(lines) if marker.match(line))

    values = []
    for line in lines[start + 1:]:
        if not line.strip():
            continue
        current_indentation = len(line) - len(line.lstrip())
        if current_indentation <= indentation:
            break
        values.append(line.strip())

    return ' '.join(values)


def _plain_scalar(document: str, key: str, indentation: int) -> str:
    marker = re.compile(r'^' + (' ' * indentation) + re.escape(key) +
                        r':\s*(.*?)\s*$')
    for line in document.splitlines():
        match = marker.match(line)
        if match:
            return match.group(1)
    raise ValueError(f'Missing scalar {key!r} at indentation {indentation}')


def _evaluate_condition_node(node, values):
    if isinstance(node, ast.Expression):
        return _evaluate_condition_node(node.body, values)
    if isinstance(node, ast.BoolOp):
        operands = (
            _evaluate_condition_node(value, values) for value in node.values)
        if isinstance(node.op, ast.And):
            return all(operands)
        if isinstance(node.op, ast.Or):
            return any(operands)
    if isinstance(node, ast.Compare):
        left = _evaluate_condition_node(node.left, values)
        for operator, comparator_node in zip(node.ops, node.comparators):
            right = _evaluate_condition_node(comparator_node, values)
            if isinstance(operator, ast.Eq):
                matches = left == right
            elif isinstance(operator, ast.NotEq):
                matches = left != right
            else:
                raise AssertionError(
                    f'Unsupported comparison: {ast.dump(operator)}')
            if not matches:
                return False
            left = right
        return True
    if isinstance(node, ast.Name) and node.id in values:
        return values[node.id]
    if isinstance(node, ast.Constant):
        return node.value
    raise AssertionError(f'Unsupported condition syntax: {ast.dump(node)}')


def _evaluate_label_condition(expression: str, action: str,
                              label_name: str) -> bool:
    python_expression = expression.replace('github.event.action', 'action')
    python_expression = python_expression.replace('github.event.label.name',
                                                  'label_name')
    python_expression = python_expression.replace('&&', ' and ')
    python_expression = python_expression.replace('||', ' or ')
    parsed = ast.parse(python_expression, mode='eval')
    return bool(
        _evaluate_condition_node(parsed, {
            'action': action,
            'label_name': label_name,
        }))


class MetaWorkflowConcurrencyTest(unittest.TestCase):

    def _read_workflow(self, name: str) -> str:
        return (ROOT / '.github/workflows' / name).read_text(encoding='utf-8')

    def test_ci_check_filters_label_events_before_job_concurrency(self):
        workflow = self._read_workflow('ci-checks.yml')
        jobs = _mapping_block(workflow, 'jobs', 0)
        job = _mapping_block(jobs, 'check_ci_status', 2)
        pre_concurrency = _before_mapping(job, 'concurrency', 4)
        condition = _folded_scalar(pre_concurrency, 'if', 4)
        concurrency = _mapping_block(job, 'concurrency', 4)
        concurrency_group = _folded_scalar(concurrency, 'group', 6)

        self.assertEqual(
            condition,
            "(github.event.action != 'labeled' && github.event.action != 'unlabeled') || "
            "(github.event.action == 'labeled' && github.event.label.name == 'ok-to-test') || "
            "(github.event.action == 'unlabeled' && github.event.label.name == 'needs-ok-to-test')",
        )

        expected_results = {
            ('opened', ''): True,
            ('synchronize', ''): True,
            ('reopened', ''): True,
            ('labeled', 'ok-to-test'): True,
            ('unlabeled', 'needs-ok-to-test'): True,
            ('labeled', 'dependencies'): False,
            ('labeled', 'size/M'): False,
            ('labeled', 'needs-ok-to-test'): False,
            ('unlabeled', 'ok-to-test'): False,
        }
        for event, expected in expected_results.items():
            with self.subTest(action=event[0], label=event[1]):
                self.assertEqual(
                    _evaluate_label_condition(condition, *event), expected)

        self.assertFalse(_has_mapping(workflow, 'concurrency', 0))
        self.assertIn("&& 'head-update' || github.run_id", concurrency_group)
        self.assertEqual(
            _plain_scalar(concurrency, 'cancel-in-progress', 6),
            "${{ github.event.action == 'synchronize' }}",
        )

    def test_approval_runs_on_open_but_skips_unrelated_labels(self):
        workflow = self._read_workflow('gh-workflow-approve.yml')
        workflow_header = workflow.split('\njobs:', 1)[0]
        jobs = _mapping_block(workflow, 'jobs', 0)
        job = _mapping_block(jobs, 'ok-to-test', 2)
        pre_concurrency = _before_mapping(job, 'concurrency', 4)
        condition = _folded_scalar(pre_concurrency, 'if', 4)
        concurrency = _mapping_block(job, 'concurrency', 4)
        concurrency_group = _folded_scalar(concurrency, 'group', 6)

        self.assertEqual(
            condition,
            "github.event.action != 'labeled' || github.event.label.name == 'ok-to-test'",
        )

        expected_results = {
            ('opened', ''): True,
            ('synchronize', ''): True,
            ('reopened', ''): True,
            ('labeled', 'ok-to-test'): True,
            ('labeled', 'dependencies'): False,
            ('labeled', 'size/M'): False,
        }
        for event, expected in expected_results.items():
            with self.subTest(action=event[0], label=event[1]):
                self.assertEqual(
                    _evaluate_label_condition(condition, *event), expected)

        self.assertIn('      - opened\n', workflow_header)
        self.assertFalse(_has_mapping(workflow, 'concurrency', 0))
        self.assertIn("&& 'head-update' || github.run_id", concurrency_group)
        self.assertEqual(
            _plain_scalar(concurrency, 'cancel-in-progress', 6),
            "${{ github.event.action == 'synchronize' }}",
        )

    def test_gatekeeper_exempts_only_trusted_automation_authors(self):
        workflow = self._read_workflow('pr-gate.yml')
        jobs = _mapping_block(workflow, 'jobs', 0)
        job = _mapping_block(jobs, 'check-pr-author', 2)
        condition = _folded_scalar(_before_mapping(job, 'steps', 4), 'if', 4)
        self.assertEqual(
            condition,
            "github.event.pull_request.user.login != 'dependabot[bot]' && "
            "github.event.pull_request.user.login != 'copybara-service[bot]'",
        )

        expression = condition.replace('github.event.pull_request.user.login',
                                       'author').replace(
                                           'github.actor', 'actor')
        parsed = ast.parse(expression.replace('&&', ' and '), mode='eval')
        cases = [
            ('dependabot[bot]', 'dependabot[bot]', False),
            ('dependabot[bot]', 'maintainer', False),
            ('copybara-service[bot]', 'copybara-service[bot]', False),
            ('copybara-service[bot]', 'maintainer', False),
            ('contributor', 'dependabot[bot]', True),
            ('contributor', 'copybara-service[bot]', True),
            ('contributor', 'maintainer', True),
            ('renovate[bot]', 'renovate[bot]', True),
            ('dependabot-helper', 'dependabot-helper', True),
            ('copybara-service', 'copybara-service', True),
        ]
        for author, actor, should_run in cases:
            with self.subTest(author=author, actor=actor):
                self.assertEqual(
                    _evaluate_condition_node(parsed, {
                        'author': author,
                        'actor': actor,
                    }), should_run)

    def test_gatekeeper_uses_trusted_shared_membership_module(self):
        workflow = self._read_workflow('pr-gate.yml')
        self.assertIn("if: steps.membership-check.outputs.is_member == 'false'",
                      workflow)
        self.assertNotIn('continue-on-error', workflow)
        self.assertNotIn('author_association', workflow)
        self.assertNotIn('contributor-report.py', workflow)
        self.assertIn('ref: ${{ github.workflow_sha }}', workflow)
        self.assertIn('persist-credentials: false', workflow)
        self.assertIn('sparse-checkout: .github/scripts/kubeflow_membership.py',
                      workflow)
        self.assertIn('run: python3 .github/scripts/kubeflow_membership.py',
                      workflow)
        self.assertIn('PR_AUTHOR: ${{ github.event.pull_request.user.login }}',
                      workflow)
        self.assertIn("'.github/workflows/pr-gate.yml'",
                      self._read_workflow('ci-scripts-tests.yml'))

    @unittest.skipUnless(
        shutil.which('jq'), 'jq is required by the gate script')
    def test_gatekeeper_closure_message_and_guidelines_link(self):
        workflow = self._read_workflow('pr-gate.yml')
        message = textwrap.dedent(
            workflow.split('          CLOSURE_MESSAGE: |\n',
                           1)[1].split('        run: |', 1)[0]).strip()
        heading = 'Pull Request Admission for External Contributors'
        anchor = heading.lower().replace(' ', '-')
        self.assertIn('### ' + heading, (ROOT / 'CONTRIBUTING.md').read_text())
        self.assertEqual(
            message,
            'PRs from external contributors must link to an issue that a maintainer has labeled `ready`.\n\n'
            'This PR does not currently meet that requirement. This is not a judgment on the quality of your contribution.\n\n'
            'To continue, please work with maintainers to get the associated issue triaged and labeled `ready`, then link to it in the PR description.\n\n'
            f'See [{heading}](https://github.com/kubeflow/pipelines/blob/master/CONTRIBUTING.md#{anchor}) for the updated contribution guidelines.',
        )
        script = textwrap.dedent(workflow.rsplit('        run: |\n', 1)[1])
        fake_gh = '''
        gh() {
          case "$1 $2" in
            'pr view') printf '%s' "$ISSUE_JSON" ;;
            'issue view') printf '%s' "$LABELS_JSON" ;;
            'pr comment') printf '%s' "$5" > "$COMMENT_FILE" ;;
            'pr close') printf 'closed' > "$STATE_FILE" ;;
            'pr edit') printf 'admitted' > "$STATE_FILE" ;;
            *) return 99 ;;
          esac
        }
        '''
        cases = [([], [], 'closed'), ([{
            'number': 123
        }], [], 'closed'), ([{
            'number': 123
        }], [{
            'name': 'ready'
        }], 'admitted')]
        for issues, labels, expected_state in cases:
            with self.subTest(
                    issues=issues,
                    labels=labels), tempfile.TemporaryDirectory() as directory:
                comment_file = Path(directory) / 'comment'
                state_file = Path(directory) / 'state'
                result = subprocess.run(
                    ['bash', '-eo', 'pipefail', '-c', fake_gh + script],
                    env={
                        **os.environ,
                        'PR_NUMBER':
                            '456',
                        'GITHUB_REPOSITORY':
                            'kubeflow/pipelines',
                        'CLOSURE_MESSAGE':
                            message,
                        'ISSUE_JSON':
                            json.dumps({'closingIssuesReferences': issues}),
                        'LABELS_JSON':
                            json.dumps({'labels': labels}),
                        'COMMENT_FILE':
                            str(comment_file),
                        'STATE_FILE':
                            str(state_file),
                    },
                    capture_output=True,
                    text=True,
                    timeout=10,
                )
                closed = expected_state == 'closed'
                self.assertEqual(result.returncode, 1 if closed else 0,
                                 result.stderr)
                self.assertEqual(state_file.read_text(), expected_state)
                self.assertEqual(comment_file.exists(), closed)
                if closed:
                    self.assertEqual(comment_file.read_text(), message)

    def test_eligibility_shell_receives_label_name_through_environment(self):
        workflow = self._read_workflow('ci-checks.yml')
        step_start = workflow.index(
            '      - name: Determine whether CI polling is needed')
        step_end = workflow.index(
            '      - name: Wait for action_required workflow runs to be approved'
        )
        eligibility_step = workflow[step_start:step_end]
        shell_script = eligibility_step.split('        run: |', 1)[1]

        self.assertIn('LABEL_NAME: ${{ github.event.label.name }}',
                      eligibility_step)
        self.assertNotIn('${{ github.event.label.name }}', shell_script)
        self.assertIn("reason=\"Added label '${LABEL_NAME}'", shell_script)


if __name__ == '__main__':
    unittest.main()
