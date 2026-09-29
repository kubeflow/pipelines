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


def _evaluate_label_condition(expression: str,
                              action: str,
                              label_name: str,
                              event_name: str = 'pull_request_target') -> bool:
    python_expression = expression.replace('github.event.action', 'action')
    python_expression = python_expression.replace('github.event.label.name',
                                                  'label_name')
    python_expression = python_expression.replace('github.event_name',
                                                  'event_name')
    python_expression = python_expression.replace('&&', ' and ')
    python_expression = python_expression.replace('||', ' or ')
    parsed = ast.parse(python_expression, mode='eval')
    return bool(
        _evaluate_condition_node(parsed, {
            'action': action,
            'label_name': label_name,
            'event_name': event_name,
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
        concurrency_group = _plain_scalar(concurrency, 'group', 6)

        self.assertIn("github.event.workflow_run.event == 'pull_request'",
                      condition)
        self.assertIn("github.event.label.name == 'ok-to-test'", condition)
        self.assertIn("github.event.label.name == 'needs-ok-to-test'",
                      condition)

        self.assertFalse(_has_mapping(workflow, 'concurrency', 0))
        self.assertIn('github.event.workflow_run.head_sha', concurrency_group)
        self.assertIn('github.event.pull_request.head.sha', concurrency_group)
        self.assertEqual(
            _plain_scalar(concurrency, 'cancel-in-progress', 6), 'false')

    def test_scheduled_recovery_uses_same_short_writer(self):
        workflow = self._read_workflow('ci-checks.yml')
        jobs = _mapping_block(workflow, 'jobs', 0)
        writer = _mapping_block(jobs, 'check_ci_status', 2)
        discovery = _mapping_block(jobs, 'recovery_candidates', 2)
        self.assertIn("cron: '7,22,37,52 * * * *'", workflow)
        self.assertIn("github.event_name == 'schedule'", discovery)
        self.assertNotIn('concurrency:', discovery)
        self.assertIn(
            'matrix.candidate.head || github.event.workflow_run.head_sha',
            writer)
        self.assertIn('name: check_ci_status', writer)
        self.assertIn('max-parallel: 4', writer)
        self.assertIn('fail-fast: false', writer)
        self.assertIn('CI_RECOVERY_NUMBER: ${{ matrix.candidate.number }}',
                      writer)
        self.assertIn('CI_RECOVERY_HEAD: ${{ matrix.candidate.head }}', writer)
        self.assertIn("poll: 'false'", writer)
        # poll: 'false' alone is not enough: the pinned action still waits a
        # minute before its first API call, holding the shared writer lock.
        self.assertIn("delay: '0'", writer)
        self.assertNotIn('sleep', writer)

    def test_publisher_has_pr_label_write_permission(self):
        workflow = self._read_workflow('ci-checks.yml')
        jobs = _mapping_block(workflow, 'jobs', 0)
        for name, expected in [('check_ci_status', 'write'),
                               ('recovery_candidates', 'read')]:
            with self.subTest(job=name):
                job = _mapping_block(jobs, name, 2)
                permissions = _mapping_block(job, 'permissions', 4)
                self.assertEqual(
                    _plain_scalar(permissions, 'pull-requests', 6), expected)

    def test_publisher_checkout_uses_only_trusted_refs(self):
        workflow = self._read_workflow('ci-checks.yml')
        writer = _mapping_block(
            _mapping_block(workflow, 'jobs', 0), 'check_ci_status', 2)
        checkouts = [
            step for step in writer.split('      - ')
            if re.search(r'(?m)^        uses: actions/checkout@', step)
        ]
        cases = [
            ('pull_request_target', 'opened', 'base-sha'),
            ('pull_request_target', 'synchronize', 'base-sha'),
            ('pull_request_target', 'reopened', 'base-sha'),
            ('pull_request_target', 'closed', 'merged-pr-sha'),
            ('pull_request_target', 'closed', 'base-sha'),
            ('workflow_run', 'completed', 'default-sha'),
            ('schedule', '', 'default-sha'),
        ]
        for event_name, action, sha in cases:
            for default_branch in ['master', 'release/main']:
                with self.subTest(
                        event=event_name,
                        action=action,
                        sha=sha,
                        default_branch=default_branch):
                    active = []
                    for step in checkouts:
                        condition = _plain_scalar(step, 'if', 8)
                        condition = condition.replace('github.event_name',
                                                      'event_name')
                        condition = condition.replace('github.event.action',
                                                      'action')
                        condition = condition.replace('&&', ' and ')
                        condition = condition.replace('||', ' or ')
                        if _evaluate_condition_node(
                                ast.parse(condition, mode='eval'), {
                                    'event_name': event_name,
                                    'action': action,
                                }):
                            active.append(step)
                    self.assertEqual(len(active), 1)
                    step = active[0]
                    ref = _plain_scalar(step, 'ref', 10)
                    ref = ref.replace('${{ github.sha }}', sha).replace(
                        '${{ github.event.repository.default_branch }}',
                        default_branch)
                    expected = (f'refs/heads/{default_branch}'
                                if event_name == 'pull_request_target' and
                                action == 'closed' else sha)
                    self.assertEqual(ref, expected)
                    self.assertEqual(
                        _plain_scalar(step, 'repository', 10),
                        '${{ github.repository }}')
                    self.assertEqual(
                        _plain_scalar(step, 'persist-credentials', 10), 'false')
                    self.assertNotIn('allow-unsafe-pr-checkout', step)

    def test_publisher_exclusion_matches_hosted_matrix_job_name(self):
        workflow = self._read_workflow('ci-checks.yml')
        exclusions = re.search(r"checks_exclude: '([^']+)'", workflow).group(1)
        pattern = exclusions.split(',')[0]
        for name in [
                'check_ci_status', 'check_ci_status (0)',
                'check_ci_status (14111, abc123)'
        ]:
            self.assertIsNotNone(re.fullmatch(pattern, name))
        self.assertIsNone(re.fullmatch(pattern, 'check_ci_status_unrelated'))

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
            "github.event_name == 'pull_request_target' && "
            "(github.event.action != 'labeled' || "
            "github.event.label.name == 'ok-to-test')",
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
        self.assertFalse(
            _evaluate_label_condition(condition, '', '', 'schedule'))

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
        self.assertIn('            .github/scripts/kubeflow_membership.py\n',
                      workflow)
        self.assertIn('            .github/scripts/pr_gate_issue.py\n',
                      workflow)
        self.assertIn('            .github/scripts/requirements.txt\n',
                      workflow)
        self.assertIn('run: python3 .github/scripts/kubeflow_membership.py',
                      workflow)
        self.assertIn('PR_AUTHOR: ${{ github.event.pull_request.user.login }}',
                      workflow)
        self.assertIn("'.github/workflows/pr-gate.yml'",
                      self._read_workflow('ci-scripts-tests.yml'))

    def test_membership_consumers_install_shared_requirements(self):
        install = 'run: python3 -m pip install -r .github/scripts/requirements.txt'
        consumers = {
            'pr-gate.yml':
                'run: python3 .github/scripts/kubeflow_membership.py',
            'contributor-report.yml':
                'run: python3 .github/scripts/contributor-report.py',
            'ci-scripts-tests.yml':
                'run: python3 -m unittest discover',
        }
        for name, consumer in consumers.items():
            with self.subTest(workflow=name):
                workflow = self._read_workflow(name)
                self.assertLess(
                    workflow.index('uses: actions/setup-python@'),
                    workflow.index(install))
                self.assertLess(
                    workflow.index(install), workflow.index(consumer))
                if name != 'ci-scripts-tests.yml':
                    self.assertIn(
                        '            .github/scripts/requirements.txt\n',
                        workflow)
                    self.assertIn('ref: ${{ github.workflow_sha }}', workflow)
        requirements = (ROOT / '.github/scripts/requirements.txt').read_text()
        self.assertRegex(requirements, r'(?m)^PyYAML==\d+\.\d+\.\d+$')

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
            'PRs from external contributors must reference an issue in this repository that a maintainer has labeled `ready`.\n\n'
            'This PR does not currently meet that requirement. This is not a judgment on the quality of your contribution.\n\n'
            'To continue, please work with maintainers to get the associated issue triaged and labeled `ready`, then add `Fixes #1234` on its own line in the PR description and reopen this PR.\n\n'
            f'See [{heading}](https://github.com/kubeflow/pipelines/blob/master/CONTRIBUTING.md#{anchor}) for the updated contribution guidelines.',
        )
        admission = workflow.split(
            '      - name: Check linked issue admission for external contributors\n',
            1)[1].split(
                '      - name: Approve pending workflow runs after admission\n',
                1)[0]
        script = textwrap.dedent(admission.split('        run: |\n', 1)[1])
        fake_gh = '''
        gh() {
          case "$1 $2" in
            'pr view')
              if [[ "$PR_VIEW_ERROR" == '1' ]]; then return 22; fi
              if [[ " $* " == *' --json labels '* ]]; then
                printf '%s' "$PR_LABELS_JSON"
              else
                printf '%s' "$ISSUE_JSON"
              fi ;;
            'issue view')
              if [[ "$ISSUE_VIEW_ERROR" == '1' ]]; then return 22; fi
              printf '%s' "$LABELS_JSON" ;;
            'pr comment') printf '%s' "$5" > "$COMMENT_FILE" ;;
            'pr close') printf 'closed' > "$STATE_FILE" ;;
            'pr edit')
              if [[ " $* " == *' --remove-label needs-ok-to-test '* ]]; then
                printf 'removed' > "$REMOVED_FILE"
              else
                printf 'admitted' > "$STATE_FILE"
              fi ;;
            *) return 99 ;;
          esac
        }
        '''
        same_repo = {'owner': {'login': 'kubeflow'}, 'name': 'pipelines'}
        other_repo = {'owner': {'login': 'other'}, 'name': 'project'}
        cases = [
            ('master', '', [], [], 'closed'),
            ('master', 'Fixes #123', [], [{
                'name': 'ready'
            }], 'closed'),
            ('master', '', [{
                'number': 123,
                'repository': same_repo
            }], [], 'closed'),
            ('master', '', [{
                'number': 123,
                'repository': same_repo
            }], [{
                'name': 'ready'
            }], 'admitted'),
            ('release-2.18', 'Fixes #123 for release-2.18', [], [{
                'name': 'ready'
            }], 'admitted'),
            ('release-2.18', 'Fixes #123', [], [{
                'name': 'not-ready'
            }], 'closed'),
            ('release-2.18', 'Fixes other/project#123', [], [{
                'name': 'ready'
            }], 'closed'),
            ('release-2.18', '', [{
                'number': 123,
                'repository': other_repo
            }], [{
                'name': 'ready'
            }], 'closed'),
            ('release-2.18', 'Fixes #123', [], None, 'error'),
            ('release-2.18', 'Fixes #123', [], [{
                'name': 'ready'
            }], 'malformed_json'),
            ('release-2.18', 'Fixes #123', [], [{
                'name': 'ready'
            }], 'pr_view_error'),
        ]
        exit_codes = {
            'admitted': 0,
            'closed': 1,
            'error': 22,
            'malformed_json': 1,
            'pr_view_error': 22,
        }
        for base, body, issues, labels, expected_state in cases:
            with self.subTest(
                    base=base,
                    body=body,
                    issues=issues,
                    labels=labels,
                    expected_state=expected_state), tempfile.TemporaryDirectory(
                    ) as directory:
                comment_file = Path(directory) / 'comment'
                state_file = Path(directory) / 'state'
                removed_file = Path(directory) / 'removed'
                needs_label = (
                    base == 'release-2.18' and
                    body == 'Fixes #123 for release-2.18')
                issue_json = json.dumps({
                    'closingIssuesReferences': issues,
                    'baseRefName': base,
                    'body': body,
                })
                if expected_state == 'malformed_json':
                    issue_json = '{malformed'
                result = subprocess.run(
                    ['bash', '-eo', 'pipefail', '-c', fake_gh + script],
                    env={
                        **os.environ,
                        'PR_NUMBER':
                            '456',
                        'GITHUB_REPOSITORY':
                            'kubeflow/pipelines',
                        'GH_REPO':
                            'kubeflow/pipelines',
                        'DEFAULT_BRANCH':
                            'master',
                        'CLOSURE_MESSAGE':
                            message,
                        'ISSUE_JSON':
                            issue_json,
                        'PR_VIEW_ERROR':
                            '1' if expected_state == 'pr_view_error' else '0',
                        'LABELS_JSON':
                            json.dumps({'labels': labels}),
                        'PR_LABELS_JSON':
                            json.dumps({
                                'labels': [{
                                    'name': 'needs-ok-to-test'
                                }] if needs_label else []
                            }),
                        'ISSUE_VIEW_ERROR':
                            '1' if labels is None else '0',
                        'COMMENT_FILE':
                            str(comment_file),
                        'STATE_FILE':
                            str(state_file),
                        'REMOVED_FILE':
                            str(removed_file),
                    },
                    capture_output=True,
                    text=True,
                    timeout=10,
                    cwd=ROOT,
                )
                closed = expected_state == 'closed'
                self.assertEqual(result.returncode, exit_codes[expected_state],
                                 result.stderr)
                if expected_state in ('error', 'malformed_json',
                                      'pr_view_error'):
                    self.assertFalse(state_file.exists())
                else:
                    self.assertEqual(state_file.read_text(), expected_state)
                self.assertEqual(comment_file.exists(), closed)
                self.assertEqual(removed_file.exists(), needs_label)
                if closed:
                    self.assertEqual(comment_file.read_text(), message)

    @unittest.skipUnless(shutil.which('node'), 'Node.js is required')
    def test_gate_approves_only_current_head_after_admission(self):
        workflow = self._read_workflow('pr-gate.yml')
        admission = workflow.index('      - name: Check linked issue admission')
        approval = workflow.index(
            '      - name: Approve pending workflow runs after admission')
        self.assertLess(admission, approval)
        self.assertLess(workflow.index('gh pr edit "$PR_NUMBER"'), approval)
        self.assertIn('        id: admission\n', workflow)
        self.assertIn("        if: steps.admission.outcome == 'success'\n",
                      workflow)
        self.assertIn('  actions: write\n', workflow)
        approval_step = workflow[approval:]
        script = textwrap.dedent(
            approval_step.split('          script: |\n', 1)[1])
        harness = r'''
        const fs = require('fs');
        const AsyncFunction = Object.getPrototypeOf(async function() {}).constructor;
        const script = fs.readFileSync(0, 'utf8');
        const scenario = JSON.parse(process.argv[1]);
        const head = {sha: 'validated-head', ref: 'feature',
          repo: {id: 42, owner: {login: 'contributor'}}};
        const context = {repo: {owner: 'kubeflow', repo: 'pipelines'},
          payload: {pull_request: {number: 14562, head}}};
        const current = {state: scenario.closed ? 'closed' : 'open',
          head: structuredClone(head), labels: scenario.labelRemoved ? [] :
            [{name: 'ok-to-test'}]};
        if (scenario.changedHead) current.head.sha = 'new-head';
        const ownRun = id => ({id, name: `run-${id}`,
          event: 'pull_request', head_sha: 'validated-head', head_branch: 'feature',
          head_repository: {id: 42}, pull_requests: []});
        const foreignRepo = {...ownRun(3), head_repository: {id: 99}};
        const foreignBranch = {...ownRun(4), head_branch: 'other'};
        const otherPR = {...ownRun(5), pull_requests: [{number: 7}]};
        const foreignSha = {...ownRun(6), head_sha: 'other-head'};
        const thisPR = {number: 14562, head};
        const duplicatePR = {number: 14563, head: structuredClone(head)};
        const approved = [], approvedElsewhere = new Set();
        const failures = [], notices = [], warnings = [];
        let polls = 0;
        const methods = {runs: () => {}, pulls: () => {}};
        const github = {rest: {
          pulls: {get: async () => ({data: current}), list: methods.pulls},
          actions: {listWorkflowRunsForRepo: methods.runs,
            approveWorkflowRun: async ({run_id}) => {
              if (run_id === scenario.failId || run_id === scenario.concurrentId) {
                if (run_id === scenario.concurrentId) approvedElsewhere.add(run_id);
                throw Error('approval conflict');
              }
              approved.push(run_id);
            }},
        }, paginate: async (method, request) => {
          if (method === methods.pulls) {
            if (request.head !== 'contributor:feature' ||
                request.state !== 'open') throw Error('wrong PR query');
            return [thisPR, ...(scenario.ambiguous ? [duplicatePR] : [])];
          }
          if (method !== methods.runs || request.head_sha !== 'validated-head' ||
              request.event !== 'pull_request' ||
              request.status !== 'action_required') throw Error('wrong run query');
          polls++;
          if (scenario.noRuns) return [];
          return [ownRun(1), ...(polls >= 2 ? [ownRun(2)] : []),
            foreignRepo, foreignBranch, otherPR, foreignSha].filter(run =>
              !approved.includes(run.id) && !approvedElsewhere.has(run.id));
        }};
        const core = {info: () => {}, warning: message => warnings.push(message),
          notice: message => notices.push(message),
          setFailed: message => failures.push(message)};
        (async () => {
          await new AsyncFunction('github', 'context', 'core', 'setTimeout',
            script)(github, context, core, callback => callback());
          process.stdout.write(JSON.stringify({approved, failures, notices,
            warnings, polls}));
        })().catch(error => {console.error(error); process.exitCode = 1;});
        '''
        cases = [
            ({}, [1, 2], False, False, False),
            ({
                'changedHead': True
            }, [], False, True, True),
            ({
                'closed': True
            }, [], False, True, True),
            ({
                'labelRemoved': True
            }, [], False, True, True),
            ({
                'ambiguous': True
            }, [], True, False, True),
            ({
                'noRuns': True
            }, [], False, False, False),
            ({
                'failId': 2
            }, [1], True, False, False),
            ({
                'concurrentId': 2
            }, [1], False, False, False),
        ]
        for scenario, expected_approved, should_fail, should_skip, no_polls in cases:
            with self.subTest(scenario=scenario):
                result = subprocess.run(
                    [
                        'node', '-e',
                        textwrap.dedent(harness),
                        json.dumps(scenario)
                    ],
                    input=script,
                    capture_output=True,
                    text=True,
                    timeout=10,
                    check=True,
                )
                outcome = json.loads(result.stdout)
                self.assertEqual(outcome['approved'], expected_approved)
                self.assertEqual(bool(outcome['failures']), should_fail)
                self.assertEqual(bool(outcome['notices']), should_skip)
                self.assertEqual(outcome['polls'] == 0, no_polls)

    def test_publisher_runs_trusted_code_without_pr_interpolation(self):
        workflow = self._read_workflow('ci-checks.yml')
        self.assertIn('ref: ${{ github.sha }}', workflow)
        self.assertIn('persist-credentials: false', workflow)
        self.assertNotIn('ref: ${{ github.event.pull_request.head', workflow)
        self.assertNotIn('${{ github.event.label.name }}', workflow)
        self.assertIn("poll: 'false'", workflow)
        self.assertIn("delay: '0'", workflow)
        self.assertIn('types: [requested, in_progress, completed]', workflow)
        self.assertIn('reopened, edited, labeled', workflow)
        selector = workflow.split('    workflows:',
                                  1)[1].split('    types:', 1)[0]
        self.assertNotIn("'CI Check'", selector)
        self.assertIn("'Frontend Tests'", selector)


if __name__ == '__main__':
    unittest.main()
