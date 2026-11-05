#!/usr/bin/env python3
"""Tests for GitHub lookups and polling."""

import contextlib
import datetime
import io
import json
import subprocess
import time
import unittest
from pathlib import Path
from unittest import mock

from kfpr import core


class GithubPollingTest(unittest.TestCase):
    def test_wait_for_pr_merge_dry_run_returns_immediately(self):
        runner = core.CommandRunner(dry_run=True)
        core.wait_for_pr_merge(
            runner, 'https://github.com/kubeflow/pipelines/pull/1'
        )
        self.assertEqual(len(runner.commands), 0)

    def test_wait_for_pr_merge_fails_when_pr_is_closed(self):

        class ClosedPrRunner:
            dry_run = False

            def capture(self, command, cwd=None):
                return 'CLOSED UNKNOWN'

        with self.assertRaisesRegex(RuntimeError, 'closed without merging'):
            core.wait_for_pr_merge(
                ClosedPrRunner(), 'https://github.com/kubeflow/pipelines/pull/1'
            )

    def test_watch_pr_ci_fails_fast_on_failed_gate(self):

        class FailingCiRunner:
            dry_run = False

            def __init__(self):
                self.commands = []

            def capture(self, command, cwd=None):
                self.commands.append(command)
                return '[{"name":"unit-tests","status":"COMPLETED","conclusion":"FAILURE"}]'

        runner = FailingCiRunner()

        with self.assertRaises(RuntimeError) as cm:
            core.watch_pr_ci(
                runner, 'https://github.com/kubeflow/pipelines/pull/1'
            )

        self.assertIn('unit-tests failed', str(cm.exception))

    def test_watch_pr_ci_uses_latest_duplicate_check_status(self):

        class DuplicateCiRunner:
            dry_run = False

            def __init__(self):
                self.commands = []

            def capture(self, command, cwd=None):
                self.commands.append(command)
                return (
                    '[{"name":"check_ci_status","status":"COMPLETED","conclusion":"CANCELLED",'
                    '"startedAt":"2026-07-09T17:16:58Z"},'
                    '{"name":"check_ci_status","status":"COMPLETED","conclusion":"SUCCESS",'
                    '"startedAt":"2026-07-09T17:17:09Z"}]'
                )

        core.watch_pr_ci(
            DuplicateCiRunner(), 'https://github.com/kubeflow/pipelines/pull/1'
        )

    def test_watch_pr_ci_returns_when_all_gates_succeed(self):

        class PassingCiRunner:
            dry_run = False

            def __init__(self):
                self.commands = []

            def capture(self, command, cwd=None):
                self.commands.append(command)
                return (
                    '[{"name":"unit-tests","status":"COMPLETED","conclusion":"SUCCESS"},'
                    '{"name":"lint","status":"COMPLETED","conclusion":"SUCCESS"}]'
                )

        runner = PassingCiRunner()

        core.watch_pr_ci(runner, 'https://github.com/kubeflow/pipelines/pull/1')

        self.assertEqual(
            runner.commands[0],
            [
                'gh',
                'pr',
                'view',
                'https://github.com/kubeflow/pipelines/pull/1',
                '--json',
                'statusCheckRollup',
                '--jq',
                '.statusCheckRollup',
            ],
        )

    def test_watch_pr_ci_prints_in_progress_check_url_once(self):

        class RunningCiRunner:
            dry_run = False

            def __init__(self):
                self.commands = []
                self.calls = 0

            def capture(self, command, cwd=None):
                self.commands.append(command)
                self.calls += 1
                if self.calls == 1:
                    return (
                        '[{"name":"slow-job","status":"IN_PROGRESS","detailsUrl":"https://github.com/kubeflow/pipelines/actions/runs/1/job/2"},'
                        '{"name":"done-job","status":"COMPLETED","conclusion":"SUCCESS","detailsUrl":"https://github.com/kubeflow/pipelines/actions/runs/1/job/3"}]'
                    )
                return (
                    '[{"name":"slow-job","status":"COMPLETED","conclusion":"SUCCESS","detailsUrl":"https://github.com/kubeflow/pipelines/actions/runs/1/job/2"},'
                    '{"name":"done-job","status":"COMPLETED","conclusion":"SUCCESS","detailsUrl":"https://github.com/kubeflow/pipelines/actions/runs/1/job/3"}]'
                )

        with (
            contextlib.redirect_stdout(io.StringIO()) as output,
            mock.patch('time.sleep'),
        ):
            core.watch_pr_ci(
                RunningCiRunner(),
                'https://github.com/kubeflow/pipelines/pull/1',
            )

        self.assertIn('slow-job:', output.getvalue())
        self.assertEqual(
            output.getvalue().count(
                'https://github.com/kubeflow/pipelines/actions/runs/1/job/2'
            ),
            1,
        )
        self.assertNotIn('done-job:', output.getvalue())

    def test_pending_merge_queue_alone_is_not_successful_ci(self):
        runner = mock.Mock(spec=core.CommandRunner)
        runner.dry_run = False
        runner.capture.return_value = json.dumps(
            [
                {
                    '__typename': 'StatusContext',
                    'context': 'tide',
                    'targetUrl': 'https://oss.gprow.dev/tide',
                    'state': 'PENDING',
                }
            ]
        )
        with mock.patch.object(
            time, 'sleep', side_effect=RuntimeError('still waiting')
        ):
            with self.assertRaisesRegex(RuntimeError, 'still waiting'):
                core.watch_pr_ci(
                    runner, 'https://github.com/kubeflow/pipelines/pull/1'
                )

    def test_watch_pr_ci_waits_for_pending_ci_passed_status(self):
        runner = mock.Mock(dry_run=False)
        runner.capture.side_effect = [
            '[{"context":"ci-passed","state":"PENDING"},'
            '{"name":"unit-tests","status":"IN_PROGRESS"}]',
            '[{"context":"ci-passed","state":"PENDING"},'
            '{"name":"unit-tests","status":"COMPLETED","conclusion":"SUCCESS"}]',
            '[{"context":"ci-passed","state":"SUCCESS"},'
            '{"name":"unit-tests","status":"COMPLETED","conclusion":"SUCCESS"}]',
        ]

        with (
            contextlib.redirect_stdout(io.StringIO()) as output,
            mock.patch('time.sleep') as sleep,
        ):
            core.watch_pr_ci(
                runner, 'https://github.com/kubeflow/pipelines/pull/1'
            )

        self.assertEqual(runner.capture.call_count, 3)
        self.assertEqual(sleep.call_args_list, [mock.call(60), mock.call(60)])
        self.assertIn(
            'All PR CI gates completed successfully (2/2)', output.getvalue()
        )

    def test_watch_pr_ci_fails_fast_on_failed_status_context(self):

        class FailingStatusRunner:
            dry_run = False

            def capture(self, command, cwd=None):
                return '[{"context":"license/cla","state":"FAILURE"}]'

        with mock.patch(
            'time.sleep', side_effect=AssertionError('slept instead of failing')
        ):
            with self.assertRaises(RuntimeError) as cm:
                core.watch_pr_ci(
                    FailingStatusRunner(),
                    'https://github.com/kubeflow/pipelines/pull/1',
                )

        self.assertIn('license/cla failed', str(cm.exception))

    def test_watch_pr_ci_returns_when_status_contexts_succeed(self):

        class PassingStatusRunner:
            dry_run = False

            def capture(self, command, cwd=None):
                return '[{"context":"license/cla","state":"SUCCESS"}]'

        with mock.patch(
            'time.sleep',
            side_effect=AssertionError('slept instead of returning'),
        ):
            core.watch_pr_ci(
                PassingStatusRunner(),
                'https://github.com/kubeflow/pipelines/pull/1',
            )

    def test_watch_latest_workflow_run_dry_run_returns_immediately(self):
        runner = core.CommandRunner(dry_run=True)
        core.watch_latest_workflow_run(runner, 'test-workflow.yml', 'main')
        for command in runner.commands:
            self.assertNotEqual(command, ['gh', 'run', 'watch', ''])

    def test_watch_latest_workflow_run_retries_until_run_appears(self):

        class DelayedRunRunner:
            dry_run = False

            def __init__(self):
                self.commands = []
                self.captures = 0

            def capture(self, command, cwd=None):
                self.commands.append(command)
                self.captures += 1
                return (
                    ''
                    if self.captures == 1
                    else '12345\thttps://github.com/kubeflow/pipelines/actions/runs/12345\t2026-07-08T19:05:01Z'
                )

            def run(self, command, cwd=None, check=True):
                self.commands.append(command)

        runner = DelayedRunRunner()
        with (
            mock.patch('time.time', return_value=1783537500),
            mock.patch('time.sleep'),
        ):
            core.watch_latest_workflow_run(runner, 'test-workflow.yml', 'main')

        self.assertEqual(
            runner.commands[-1],
            ['gh', 'run', 'watch', '12345', '--exit-status'],
        )
        self.assertNotIn(['gh', 'run', 'watch', ''], runner.commands)

    def test_watch_latest_workflow_run_ignores_runs_created_before_dispatch(
        self,
    ):

        class DelayedNewRunRunner:
            dry_run = False

            def __init__(self):
                self.commands = []
                self.captures = 0

            def capture(self, command, cwd=None):
                self.commands.append(command)
                self.captures += 1
                if self.captures == 1:
                    return '111\thttps://github.com/kubeflow/pipelines/actions/runs/111\t2026-07-08T19:00:00Z'
                return '222\thttps://github.com/kubeflow/pipelines/actions/runs/222\t2026-07-08T19:05:01Z'

            def run(self, command, cwd=None, check=True):
                self.commands.append(command)

        runner = DelayedNewRunRunner()
        with (
            mock.patch('time.time', return_value=1783537500),
            mock.patch('time.sleep'),
        ):
            core.watch_latest_workflow_run(runner, 'test-workflow.yml', 'main')

        self.assertEqual(
            runner.commands[-1], ['gh', 'run', 'watch', '222', '--exit-status']
        )
        self.assertNotIn(['gh', 'run', 'watch', '111'], runner.commands)

    def test_watch_latest_workflow_run_can_resume_existing_run(self):

        class ExistingRunRunner:
            dry_run = False

            def __init__(self):
                self.commands = []

            def capture(self, command, cwd=None):
                self.commands.append(command)
                return '111\thttps://github.com/kubeflow/pipelines/actions/runs/111\t2026-07-08T19:00:00Z'

            def run(self, command, cwd=None, check=True):
                self.commands.append(command)

        runner = ExistingRunRunner()
        with mock.patch('time.time', return_value=1783537500):
            core.watch_latest_workflow_run(
                runner, 'test-workflow.yml', 'main', fresh_only=False
            )

        self.assertEqual(
            runner.commands[-1], ['gh', 'run', 'watch', '111', '--exit-status']
        )

    def test_watch_latest_workflow_run_prints_run_url(self):

        class RunUrlRunner:
            dry_run = False

            def __init__(self):
                self.commands = []

            def capture(self, command, cwd=None):
                self.commands.append(command)
                return '12345\thttps://github.com/kubeflow/pipelines/actions/runs/12345\t2026-07-08T19:05:01Z'

            def run(self, command, cwd=None, check=True):
                self.commands.append(command)

        runner = RunUrlRunner()
        with (
            mock.patch('time.time', return_value=1783537500),
            mock.patch('builtins.print') as print_mock,
        ):
            core.watch_latest_workflow_run(runner, 'test-workflow.yml', 'main')

        output = '\n'.join(
            str(call.args[0]) for call in print_mock.call_args_list
        )
        self.assertIn(
            'Workflow run: \033[4mhttps://github.com/kubeflow/pipelines/actions/runs/12345\033[0m',
            output,
        )
        self.assertEqual(
            runner.commands[-1],
            ['gh', 'run', 'watch', '12345', '--exit-status'],
        )

    def test_watch_latest_workflow_run_supports_python_without_datetime_utc(
        self,
    ):

        class RunRunner:
            dry_run = False

            def capture(self, command, cwd=None):
                return '12345\thttps://github.com/kubeflow/pipelines/actions/runs/12345\t2026-07-08T19:05:01Z'

            def run(self, command, cwd=None, check=True):
                pass

        utc = getattr(datetime, 'UTC', None)
        if utc is not None:
            del datetime.UTC
        try:
            with mock.patch('time.time', return_value=1783537500):
                core.watch_latest_workflow_run(
                    RunRunner(), 'test-workflow.yml', 'main'
                )
        finally:
            if utc is not None:
                datetime.UTC = utc


class DefaultBranchTest(unittest.TestCase):
    def test_only_supported_upstream_defaults_are_accepted(self):
        for branch in ('master', 'main'):
            with self.subTest(branch=branch):
                self.assertEqual(
                    core.default_branch(lambda _: {'default_branch': branch}),
                    branch,
                )
        for metadata in (
            {},
            {'default_branch': 'release-3.0'},
            {'default_branch': None},
        ):
            with (
                self.subTest(metadata=metadata),
                self.assertRaisesRegex(ValueError, 'default branch'),
            ):
                core.default_branch(lambda _: metadata)

    def test_api_failure_does_not_fall_back_to_master(self):
        api = mock.Mock(side_effect=RuntimeError('API unavailable'))
        with self.assertRaisesRegex(RuntimeError, 'API unavailable'):
            core.default_branch(api)

    def test_dry_run_uses_placeholder_without_live_lookup(self):
        runner = core.CommandRunner(dry_run=True)
        with mock.patch.object(
            subprocess, 'check_output', side_effect=AssertionError('live API')
        ):
            self.assertEqual(
                core.upstream_default_branch(runner, Path('.')),
                '<repository-default>',
            )


if __name__ == '__main__':
    unittest.main()
