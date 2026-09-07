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
from pathlib import Path
import re
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

    def test_publisher_runs_trusted_code_without_pr_interpolation(self):
        workflow = self._read_workflow('ci-checks.yml')
        self.assertIn('ref: ${{ github.sha }}', workflow)
        self.assertIn('persist-credentials: false', workflow)
        self.assertNotIn('ref: ${{ github.event.pull_request.head', workflow)
        self.assertNotIn('${{ github.event.label.name }}', workflow)
        self.assertIn("poll: 'false'", workflow)
        self.assertIn('types: [requested, in_progress, completed]', workflow)
        self.assertIn('reopened, edited, labeled', workflow)
        selector = workflow.split('    workflows:',
                                  1)[1].split('    types:', 1)[0]
        self.assertNotIn("'CI Check'", selector)
        self.assertIn("'Frontend Tests'", selector)


if __name__ == '__main__':
    unittest.main()
