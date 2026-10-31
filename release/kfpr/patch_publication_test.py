"""Maintenance releases must preserve defaults for newer release lines."""

import subprocess
import unittest
from pathlib import Path
from tempfile import TemporaryDirectory
from unittest import mock

from kfpr import core, steps
from kfpr.ranges import ReleaseRange


class PatchPublicationTest(unittest.TestCase):
    def setUp(self):
        directory = TemporaryDirectory()
        self.addCleanup(directory.cleanup)
        self.root = Path(directory.name)
        self.runner = mock.Mock(spec=core.CommandRunner)
        self.runner.dry_run = False
        self.runner.run.return_value = subprocess.CompletedProcess([], 1)
        self.context = core.ReleaseContext(
            root=self.root,
            state=core.ReleaseState(self.root / 'state.json'),
            runner=self.runner,
            metadata=core.ReleaseMetadata.from_version('patch', '2.17.3'),
            fork_remote='https://github.com/testuser/pipelines.git',
            include_backend=True,
            include_sdk=True,
        )
        self.context.previous_release = '2.17.2'
        self.context.previous_sdk_release = 'sdk-2.17.2'
        self.context.release_target = 'release-candidate'
        self.context.state.answers['release_sha'] = 'a' * 40
        for target, result in (
            (
                'resolve_range',
                ReleaseRange('2.17.2', 'release-candidate', 'b' * 40, 'a' * 40),
            ),
            ('read_commits', []),
            ('_remote_release_sha', 'a' * 40),
            ('confirm', None),
        ):
            patcher = mock.patch('kfpr.steps.' + target, return_value=result)
            patcher.start()
            self.addCleanup(patcher.stop)
        for target in (
            'qualification.require_qualification',
            '_ensure_release_tag',
        ):
            patcher = mock.patch('kfpr.steps.' + target)
            patcher.start()
            self.addCleanup(patcher.stop)

    def test_patch_does_not_change_master_version_or_changelog(self):
        (self.root / 'VERSION').write_text('3.0.0')
        (self.root / 'CHANGELOG.md').write_text('Newer release history')
        with (
            mock.patch.object(steps, 'watch_pr_ci'),
            mock.patch.object(steps, 'wait_for_pr_merge'),
        ):
            steps.step_sync_master(self.context)
        self.runner.run.assert_not_called()
        self.runner.capture.assert_not_called()
        self.assertEqual((self.root / 'VERSION').read_text(), '3.0.0')
        self.assertEqual(
            (self.root / 'CHANGELOG.md').read_text(), 'Newer release history'
        )

    def test_patch_builds_docs_without_resetting_global_defaults(self):
        for consolidated in (False, True):
            with self.subTest(consolidated=consolidated):
                client = mock.Mock()
                client.get_build.return_value = {
                    'state': {'code': 'finished'},
                    'success': True,
                }
                with (
                    mock.patch.dict('os.environ', {'RTD_TOKEN': 'test-token'}),
                    mock.patch.object(
                        steps,
                        '_sdk_release_is_consolidated',
                        return_value=consolidated,
                    ),
                    mock.patch.object(
                        steps.rtd, 'ReadTheDocsClient', return_value=client
                    ),
                ):
                    steps.step_confirm_rtd(self.context)
                self.assertEqual(
                    client.trigger_build.call_count, 1 if consolidated else 2
                )
                self.assertEqual(
                    client.activate_version.call_count, 1 if consolidated else 2
                )
                client.patch_project.assert_not_called()
                checklist = steps.manual_checklist(
                    'confirm-rtd', self.context.metadata, consolidated
                )
                self.assertNotIn('Set Default', checklist)
                self.assertIn('Preserve', checklist)

    def test_older_patch_github_releases_are_not_latest(self):
        self.runner.capture.return_value = (
            '[[{"tag_name":"3.0.0","draft":false,"prerelease":false}]]'
        )
        with (
            mock.patch.object(
                steps, '_sdk_release_is_consolidated', return_value=False
            ),
            mock.patch.object(steps, 'prompt_required', return_value='2.17.2'),
            mock.patch.object(
                steps, '_generate_backend_release_notes', return_value='Fixes'
            ),
        ):
            steps.step_create_sdk_release(self.context)
            steps.step_create_backend_release(self.context)
        commands = [
            call.args[0]
            for call in self.runner.run.call_args_list
            if call.args[0][:3] == ['gh', 'release', 'create']
        ]
        self.assertEqual(len(commands), 2)
        for command in commands:
            self.assertIn('--latest=false', command)

    def test_minor_github_releases_retain_latest_selection(self):
        self.context.metadata = core.ReleaseMetadata.from_version(
            'minor', '2.18.0'
        )
        self.runner.capture.return_value = '[]'
        with (
            mock.patch.object(
                steps, '_sdk_release_is_consolidated', return_value=False
            ),
            mock.patch.object(steps, 'prompt_required', return_value='2.17.2'),
            mock.patch.object(
                steps,
                '_generate_backend_release_notes',
                return_value='Features',
            ),
        ):
            steps.step_create_sdk_release(self.context)
            steps.step_create_backend_release(self.context)
        for call in self.runner.run.call_args_list:
            if call.args[0][:3] == ['gh', 'release', 'create']:
                expected = (
                    '--latest=false'
                    if call.args[0][3].startswith('sdk-')
                    else '--latest=true'
                )
                self.assertIn(expected, call.args[0])

    def test_patch_website_checklist_preserves_latest(self):
        checklist = steps.manual_checklist(
            'confirm-website-and-slack', self.context.metadata
        )
        self.assertNotIn('echo -n 2.17.3', checklist)
        self.assertIn('Preserve', checklist)
        self.assertIn('latest-version.html', checklist)
        self.assertIn('releases/tag/2.17.3', checklist)
