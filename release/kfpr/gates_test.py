"""Release mutations must use a qualified, merged commit and immutable tags."""

import json
import subprocess
import unittest
from pathlib import Path
from tempfile import TemporaryDirectory
from unittest import mock

from kfpr import core, steps

SHA = 'a' * 40
HEAD = 'b' * 40


class ReleaseGatesTest(unittest.TestCase):
    def setUp(self):
        tmp = TemporaryDirectory()
        self.addCleanup(tmp.cleanup)
        self.root = Path(tmp.name)
        self.runner = mock.Mock(spec=core.CommandRunner)
        self.runner.dry_run = False
        self.runner.run.return_value = subprocess.CompletedProcess([], 1)
        self.runner.capture.return_value = ''
        self.context = core.ReleaseContext(
            root=self.root,
            state=core.ReleaseState(self.root / 'state.json'),
            runner=self.runner,
            metadata=core.ReleaseMetadata.from_version('patch', '2.17.3'),
            fork_remote='https://github.com/user/pipelines.git',
            include_backend=True,
            include_sdk=True,
        )
        self.context.state.answers.update(
            version_pr_url='https://github.com/kubeflow/pipelines/pull/1',
            release_sha=SHA,
        )

    def pr(self, **changes):
        pr = dict(
            state='MERGED',
            baseRefName='release-2.17',
            headRefName='2.17.3-update-version-tags',
            headRefOid=HEAD,
            mergeCommit={'oid': SHA},
            statusCheckRollup=[
                {'name': 'ci', 'status': 'COMPLETED', 'conclusion': 'SUCCESS'}
            ],
        )
        pr.update(changes)
        self.runner.capture.return_value = json.dumps(pr)

    def test_merged_latest_head_is_bound(self):
        self.pr()
        self.assertEqual(steps.validate_release_pr(self.context), SHA)
        self.assertEqual(
            self.context.state.answers['version_pr_head_sha'], HEAD
        )
        self.runner.run.assert_not_called()

    def test_explicit_followup_pr_binds_its_merged_commit(self):
        self.context.state.answers = {
            'version_pr_url': 'https://github.com/kubeflow/pipelines/pull/15041',
        }
        merge_sha = 'a0733cb51b0484a6050cafee989ffe3312f808ad'
        head_sha = 'a6680eafcd9d1b8a258571d5ee5a5a90c88a7b42'
        self.pr(
            headRefName='codex/2.17.3-ci-sdk-source',
            headRefOid=head_sha,
            mergeCommit={'oid': merge_sha},
        )
        self.assertEqual(steps.validate_release_pr(self.context), merge_sha)
        self.assertEqual(self.context.state.answers['release_sha'], merge_sha)
        self.assertEqual(
            self.context.state.answers['version_pr_head_sha'], head_sha
        )
        self.assertIn(
            self.context.state.answers['version_pr_url'],
            self.runner.capture.call_args.args[0],
        )
        self.runner.run.assert_not_called()

    def test_rejects_unmerged_wrong_base_changed_head_and_non_green_ci(self):
        for changes in (
            {'state': 'OPEN'},
            {'baseRefName': 'master'},
            {'headRefOid': 'invalid'},
            {'mergeCommit': {'oid': 'invalid'}},
            {'mergeCommit': {'oid': 'c' * 40}},
            {'mergeCommit': None},
            {'statusCheckRollup': []},
            {'statusCheckRollup': [{'status': 'IN_PROGRESS', 'name': 'ci'}]},
            {
                'statusCheckRollup': [
                    {
                        'status': 'COMPLETED',
                        'conclusion': 'FAILURE',
                        'name': 'ci',
                    }
                ]
            },
            {
                'statusCheckRollup': [
                    {
                        'status': 'COMPLETED',
                        'conclusion': 'SKIPPED',
                        'name': 'ci',
                    }
                ]
            },
        ):
            with self.subTest(changes=changes):
                self.pr(headRefName='codex/2.17.3-ci-sdk-source', **changes)
                with self.assertRaises(RuntimeError):
                    steps.validate_release_pr(self.context)
        self.context.state.answers['version_pr_head_sha'] = 'c' * 40
        self.pr()
        with self.assertRaisesRegex(RuntimeError, 'changed'):
            steps.validate_release_pr(self.context)

    def test_followup_requires_explicit_upstream_pr_url(self):
        for url in ('', 'https://github.com/other/pipelines/pull/15041'):
            with self.subTest(url=url):
                self.context.state.answers['version_pr_url'] = url
                with self.assertRaisesRegex(RuntimeError, 'must be recorded'):
                    steps.validate_release_pr(self.context)
        self.runner.capture.assert_not_called()

    def test_merged_pr_ignores_pending_prow_merge_queue_status(self):
        self.pr()
        pr = json.loads(self.runner.capture.return_value)
        pr['statusCheckRollup'].append(
            {
                '__typename': 'StatusContext',
                'context': 'tide',
                'state': 'PENDING',
                'targetUrl': 'https://oss.gprow.dev/tide',
            }
        )
        self.runner.capture.return_value = json.dumps(pr)
        self.assertEqual(steps.validate_release_pr(self.context), SHA)

    def test_merge_queue_exception_cannot_bypass_ci(self):
        tide = {
            '__typename': 'StatusContext',
            'context': 'tide',
            'state': 'PENDING',
            'targetUrl': 'https://oss.gprow.dev/tide',
        }
        for changes in (
            {'__typename': 'CheckRun', 'name': 'tide', 'status': 'IN_PROGRESS'},
            {'context': 'test'},
            {'targetUrl': 'https://example.com/tide'},
            {'state': 'FAILURE'},
        ):
            with self.subTest(changes=changes):
                self.pr()
                pr = json.loads(self.runner.capture.return_value)
                pr['statusCheckRollup'].append({**tide, **changes})
                self.runner.capture.return_value = json.dumps(pr)
                with self.assertRaisesRegex(RuntimeError, 'CI is not green'):
                    steps.validate_release_pr(self.context)
        for checks in (
            [tide],
            [
                tide,
                {
                    'name': 'ci',
                    'status': 'COMPLETED',
                    'conclusion': 'SKIPPED',
                },
            ],
        ):
            with self.subTest(checks=checks):
                self.pr(statusCheckRollup=checks)
                with self.assertRaises(RuntimeError):
                    steps.validate_release_pr(self.context)
        self.pr(state='OPEN', statusCheckRollup=[tide])
        with self.assertRaisesRegex(RuntimeError, 'must be merged'):
            steps.validate_release_pr(self.context)

    def test_duplicate_check_names_cannot_hide_failure(self):
        self.pr(
            statusCheckRollup=[
                {
                    'name': 'test',
                    'status': 'COMPLETED',
                    'conclusion': 'FAILURE',
                },
                {
                    'name': 'test',
                    'status': 'COMPLETED',
                    'conclusion': 'SUCCESS',
                },
            ]
        )
        with self.assertRaises(RuntimeError):
            steps.validate_release_pr(self.context)

    def test_every_publication_entrypoint_requires_qualification_first(self):
        for function in (
            steps.step_publish_images,
            steps.step_publish_sdks,
            steps.step_create_sdk_tag,
            steps.step_create_sdk_release,
            steps.step_create_backend_release,
            steps.step_sync_master,
        ):
            with (
                self.subTest(function=function),
                mock.patch.object(
                    steps.qualification,
                    'require_qualification',
                    side_effect=RuntimeError('not qualified'),
                ),
            ):
                self.runner.reset_mock()
                with self.assertRaisesRegex(RuntimeError, 'not qualified'):
                    function(self.context)
                self.runner.run.assert_not_called()
                self.runner.capture.assert_not_called()

    def test_resume_external_version_pr_never_pushes_preparation_or_tags(self):
        self.context.state.answers['version_pr_url'] = (
            'https://github.com/kubeflow/pipelines/pull/15217'
        )
        self.pr(headRefName='codex/218-release-version')
        with (
            mock.patch.object(steps, 'watch_pr_ci'),
            mock.patch.object(steps, 'wait_for_pr_merge'),
        ):
            steps.step_merge_version_pr(self.context)
            resumed = core.ReleaseState.load(self.context.state.path)
            self.context.state = resumed
            steps.step_merge_version_pr(self.context)
        self.runner.run.assert_not_called()
        self.assertEqual(resumed.answers['release_sha'], SHA)
        self.assertEqual(resumed.answers['version_pr_head_sha'], HEAD)
        # Recording a merge does not qualify it, even if an old checkpoint says done.
        resumed.mark_done('qualify-release')
        self.runner.reset_mock()
        with self.assertRaisesRegex(ValueError, 'qualify-release'):
            steps.step_create_sdk_tag(self.context)
        self.runner.run.assert_not_called()

    def test_recorded_merge_with_pending_tide_does_not_poll_forever(self):
        self.pr()
        pr = json.loads(self.runner.capture.return_value)
        pr['statusCheckRollup'].append(
            {
                '__typename': 'StatusContext',
                'context': 'tide',
                'targetUrl': 'https://oss.gprow.dev/tide',
                'state': 'PENDING',
            }
        )

        def capture(command, **kwargs):
            if command[-1] == '.statusCheckRollup':
                return json.dumps(pr['statusCheckRollup'])
            return json.dumps(pr)

        self.runner.capture.side_effect = capture
        with (
            mock.patch.object(steps, 'wait_for_pr_merge'),
            mock.patch.object(
                core.time,
                'sleep',
                side_effect=AssertionError('Unexpected CI wait'),
            ),
        ):
            steps.step_merge_version_pr(self.context)
        self.assertEqual(self.context.state.answers['release_sha'], SHA)
        self.runner.run.assert_not_called()

    def test_recorded_unmerged_pr_cannot_reach_tag_creation(self):
        self.pr(state='OPEN', mergeCommit=None)
        with (
            mock.patch.object(steps, 'watch_pr_ci'),
            mock.patch.object(steps, 'wait_for_pr_merge'),
        ):
            with self.assertRaisesRegex(RuntimeError, 'must be merged'):
                steps.step_merge_version_pr(self.context)
        self.runner.run.assert_not_called()
        with self.assertRaisesRegex(RuntimeError, 'must be merged'):
            steps.step_create_sdk_tag(self.context)
        self.runner.run.assert_not_called()

    def test_preparation_does_not_create_tags_on_a_real_run(self):
        self.runner.run.return_value = subprocess.CompletedProcess([], 1)
        with (
            mock.patch.object(steps, '_preparation_range'),
            mock.patch.object(steps, '_update_changelog'),
            mock.patch.object(steps, '_update_sdk_version_files'),
            mock.patch.object(steps, '_run_reviewed_commit'),
        ):
            steps.step_update_version_tags(self.context)
        commands = [call.args[0] for call in self.runner.run.call_args_list]
        self.assertFalse(
            any(command[:2] == ['git', 'tag'] for command in commands)
        )
        self.assertFalse(
            any(command[:2] == ['git', 'push'] for command in commands)
        )

    def test_qualification_checkpoint_precedes_publication(self):
        ids = [step.step_id for step in steps.build_steps('patch', True, True)]
        self.assertLess(
            ids.index('merge-version-pr'), ids.index('qualify-release')
        )
        self.assertLess(
            ids.index('qualify-release'), ids.index('publish-images')
        )
        with mock.patch.object(
            steps.qualification, 'qualify_release'
        ) as qualify:
            steps.step_qualify_release(self.context)
            qualify.assert_called_once_with(self.context)

    def test_tag_created_on_exact_merge_sha_without_force(self):
        steps._ensure_release_tag(self.context, '2.17.3')
        commands = [call.args[0] for call in self.runner.run.call_args_list]
        self.assertIn(
            [
                'git',
                'tag',
                '-a',
                '2.17.3',
                SHA,
                '-m',
                'Kubeflow Pipelines 2.17.3 release',
            ],
            commands,
        )
        self.assertIn(
            [
                'git',
                'push',
                'https://github.com/kubeflow/pipelines.git',
                'refs/tags/2.17.3:refs/tags/2.17.3',
            ],
            commands,
        )
        self.assertFalse(any('--force' in c or '-f' in c for c in commands))

    def test_published_annotated_tag_is_verified_not_replaced(self):
        self.runner.capture.return_value = (
            f'{"c" * 40}\trefs/tags/2.17.3\n{SHA}\trefs/tags/2.17.3^{{}}\n'
        )
        steps._ensure_release_tag(self.context, '2.17.3')
        self.assertFalse(
            any(
                call.args[0][:2] == ['git', 'push']
                for call in self.runner.run.call_args_list
            )
        )
        self.runner.reset_mock()
        self.runner.capture.return_value = f'{"c" * 40}\trefs/tags/2.17.3\n'
        with self.assertRaisesRegex(RuntimeError, 'refusing replacement'):
            steps._ensure_release_tag(self.context, '2.17.3')
        self.runner.run.assert_not_called()

    def test_only_unpublished_stale_local_tag_is_repaired(self):
        def capture(command, cwd=None):
            return 'c' * 40 if command[1] == 'rev-parse' else ''

        self.runner.capture.side_effect = capture
        self.runner.run.return_value = subprocess.CompletedProcess([], 0)
        steps._ensure_release_tag(self.context, '2.17.3')
        self.assertIn(
            mock.call(['git', 'tag', '-d', '2.17.3'], cwd=self.root),
            self.runner.run.call_args_list,
        )
        self.runner.reset_mock()

        def published(command, cwd=None):
            if self.context.fork_remote in command:
                return f'{SHA}\trefs/tags/2.17.3\n'
            return capture(command, cwd)

        self.runner.capture.side_effect = published
        with self.assertRaisesRegex(RuntimeError, 'published tag'):
            steps._ensure_release_tag(self.context, '2.17.3')
        self.assertNotIn(
            mock.call(['git', 'tag', '-d', '2.17.3'], cwd=self.root),
            self.runner.run.call_args_list,
        )

    def test_publication_dispatch_uses_immutable_ref_and_source(self):
        self.runner.capture.return_value = '[]'
        with (
            mock.patch.object(steps.qualification, 'require_qualification'),
            mock.patch.object(steps, '_ensure_release_tag'),
            mock.patch.object(steps, 'watch_latest_workflow_run') as watch,
        ):
            steps.step_publish_images(self.context)
        command = self.runner.run.call_args.args[0]
        self.assertEqual(command[command.index('--ref') + 1], '2.17.3')
        self.assertIn('src_branch=' + SHA, command)
        self.assertEqual(watch.call_args.kwargs['expected_sha'], SHA)

    def test_workflow_source_mismatch_is_rejected_before_watch(self):
        self.runner.capture.side_effect = [
            '1\thttps://example.invalid/run/1\t2026-01-01T00:00:00Z',
            'c' * 40,
        ]
        with self.assertRaisesRegex(RuntimeError, 'qualified release commit'):
            core.watch_latest_workflow_run(
                self.runner,
                'workflow.yml',
                '2.17.3',
                fresh_only=False,
                expected_sha=SHA,
            )
        self.runner.run.assert_not_called()

        command = self.runner.capture.call_args_list[0].args[0]
        self.assertNotIn('--branch', command)
        self.assertEqual(command[command.index('--commit') + 1], SHA)
        self.assertEqual(
            command[command.index('--event') + 1], 'workflow_dispatch'
        )

    def test_existing_version_pr_is_reused_without_duplicate_or_tag_push(self):
        url = self.context.state.answers.pop('version_pr_url')
        self.runner.run.return_value = subprocess.CompletedProcess([], 1)
        self.runner.capture.return_value = json.dumps([{'url': url}])
        with (
            mock.patch.object(steps, 'watch_pr_ci'),
            mock.patch.object(steps, 'wait_for_pr_merge'),
            mock.patch.object(steps, 'validate_release_pr', return_value=SHA),
        ):
            steps.step_merge_version_pr(self.context)
        self.assertEqual(self.context.state.answers['version_pr_url'], url)
        self.assertFalse(
            any(
                call.args[0][:3] == ['gh', 'pr', 'create']
                for call in self.runner.capture.call_args_list
            )
        )
        self.assertFalse(
            any(
                'refs/tags/' in ' '.join(call.args[0])
                for call in self.runner.run.call_args_list
            )
        )

    def test_ambiguous_version_pr_reuse_fails_closed(self):
        self.context.state.answers.pop('version_pr_url')
        self.runner.capture.return_value = json.dumps(
            [{'url': 'one'}, {'url': 'two'}]
        )
        with self.assertRaisesRegex(RuntimeError, 'Multiple version PRs'):
            steps.step_merge_version_pr(self.context)

    def test_correct_remote_tag_does_not_hide_wrong_local_tag(self):
        self.runner.run.return_value = subprocess.CompletedProcess([], 0)
        self.runner.capture.side_effect = lambda command, cwd=None: (
            'c' * 40
            if command[1] == 'rev-parse'
            else f'{SHA}\trefs/tags/2.17.3\n'
        )
        with self.assertRaisesRegex(RuntimeError, 'published tag'):
            steps._ensure_release_tag(self.context, '2.17.3')
        self.assertFalse(
            any(
                call.args[0][:3]
                in (['git', 'tag', '-d'], ['git', 'push', '--force'])
                for call in self.runner.run.call_args_list
            )
        )

    def test_image_command_defaults_to_no_latest_without_explicit_selection(
        self,
    ):
        command = core.image_workflow_command(
            self.context.metadata, source_sha=SHA, workflow_ref='2.17.3'
        )
        self.assertIn('set_latest=false', command)

    def test_real_git_unpublished_tag_is_repaired_to_exact_commit(self):
        subprocess.run(['git', 'init', '-q', str(self.root)], check=True)
        commit = [
            'git',
            '-C',
            str(self.root),
            '-c',
            'user.name=Test',
            '-c',
            'user.email=test@example.invalid',
            'commit',
            '--allow-empty',
            '-qm',
        ]
        subprocess.run(commit + ['old'], check=True)
        subprocess.run(
            ['git', '-C', str(self.root), 'tag', '2.17.3'], check=True
        )
        subprocess.run(commit + ['merged'], check=True)
        sha = subprocess.check_output(
            ['git', '-C', str(self.root), 'rev-parse', 'HEAD'], text=True
        ).strip()
        self.context.state.answers['release_sha'] = sha

        class LocalRunner(core.CommandRunner):
            def capture(self, command, cwd=None):
                if command[:2] == ['git', 'ls-remote']:
                    return ''
                return super().capture(command, cwd)

            def run(self, command, cwd=None, check=True):
                if command[:2] in (['git', 'fetch'], ['git', 'push']):
                    self.commands.append(command)
                    return subprocess.CompletedProcess(command, 0)
                if command[:3] == ['git', 'tag', '-a']:
                    command = [
                        'git',
                        '-c',
                        'user.name=Test',
                        '-c',
                        'user.email=test@example.invalid',
                        *command[1:],
                    ]
                return super().run(command, cwd, check)

        self.context.runner = LocalRunner()
        steps._ensure_release_tag(self.context, '2.17.3')
        actual = subprocess.check_output(
            ['git', '-C', str(self.root), 'rev-parse', '2.17.3^{commit}'],
            text=True,
        ).strip()
        self.assertEqual(actual, sha)

    def test_backend_notes_exclude_operator_checkout_changes_after_merge(self):
        subprocess.run(['git', 'init', '-q', str(self.root)], check=True)
        commit = [
            'git',
            '-C',
            str(self.root),
            '-c',
            'user.name=Test',
            '-c',
            'user.email=test@example.invalid',
            'commit',
            '--allow-empty',
            '-qm',
        ]
        subprocess.run(commit + ['base'], check=True)
        subprocess.run(
            ['git', '-C', str(self.root), 'tag', '2.17.2'], check=True
        )
        subprocess.run(commit + ['fix(backend): approved change'], check=True)
        sha = subprocess.check_output(
            ['git', '-C', str(self.root), 'rev-parse', 'HEAD'], text=True
        ).strip()
        subprocess.run(
            commit + ['fix(backend): unrelated checkout change'], check=True
        )
        notes = steps._generate_backend_release_notes(
            self.root, steps.resolve_range(self.root, '2.17.2', sha)
        )
        self.assertIn('approved change', notes)
        self.assertNotIn('unrelated checkout change', notes)
