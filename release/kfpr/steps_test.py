#!/usr/bin/env python3
"""Tests for release steps."""

import contextlib
import io
import json
import subprocess
import sys
import unittest
from pathlib import Path
from tempfile import TemporaryDirectory
from unittest import mock

from kfpr import core, steps
from kfpr.notes import Commit
from kfpr.ranges import ReleaseRange

SELECTED_RANGE = ReleaseRange(
    'previous', 'release-candidate', 'a' * 40, 'b' * 40
)


class SdkReleaseNotesTest(unittest.TestCase):
    def test_preserves_custom_heading_only_warning(self):
        with TemporaryDirectory() as tmpdir:
            path = Path(tmpdir) / 'RELEASE.md'
            warning = '## Upgrade the backend before installing this SDK'
            path.write_text(
                '# Current Version (in development)\n\n'
                + warning
                + '\n\n# 3.1.0\n\nPrevious notes.\n'
            )
            steps._update_sdk_release_notes(
                path, '3.2.0', '## Features\n\n* New SDK feature'
            )
            self.assertIn(warning, path.read_text().split('\n# 3.2.0\n')[1])

    def test_template_only_notes_use_generated_content(self):
        template = (
            '## Features\n\n## Breaking changes\n\n'
            '## Deprecations\n\n## Bug fixes and other changes'
        )
        generated = '## Features\n\n* New SDK feature'
        with TemporaryDirectory() as tmpdir:
            path = Path(tmpdir) / 'RELEASE.md'
            path.write_text(
                '# Current Version (in development)\n\n'
                + template
                + '\n\n# 3.1.0\n\nPrevious notes.\n'
            )

            steps._update_sdk_release_notes(path, '3.2.0', generated)

            text = path.read_text()
            development, released = text.split('\n# 3.2.0\n', 1)
            self.assertIn(template, development)
            self.assertIn(generated, released)
            self.assertNotIn('## Generated changes', released)
            self.assertNotIn('## Deprecations', released)

    def test_preserves_handwritten_notes_alongside_generated_changes(self):
        handwritten = (
            '## Breaking changes\n\n'
            'Upgrade the backend before installing this SDK.\n\n'
            '### Migration\n\n'
            '1. Export existing configuration.\n'
            '2. Follow the [upgrade guide](https://example.com/upgrade).'
        )
        historical = '\n# 3.1.0\n\n## Features\n\nPrevious release notes.\n'
        with TemporaryDirectory() as tmpdir:
            path = Path(tmpdir) / 'RELEASE.md'
            path.write_text(
                '# Current Version (in development)\n\n'
                + handwritten
                + '\n'
                + historical
            )

            steps._update_sdk_release_notes(
                path, '3.2.0', '## Bug fixes and other changes\n\n* Fix retries'
            )

            text = path.read_text()
            development, released = text.split('\n# 3.2.0\n', 1)
            self.assertNotIn('Upgrade the backend', development)
            self.assertIn(handwritten, released)
            self.assertIn(
                '## Generated changes\n\n'
                '### Bug fixes and other changes\n\n* Fix retries',
                released,
            )
            self.assertTrue(text.endswith(historical))
            self.assertEqual(text.count(handwritten), 1)

    def test_repeating_release_update_is_idempotent(self):
        with TemporaryDirectory() as tmpdir:
            path = Path(tmpdir) / 'RELEASE.md'
            path.write_text(
                '# Current Version (in development)\n\n'
                'Handwritten migration guidance.\n\n'
                '# 3.1.0\n\nPrevious release notes.\n'
            )
            generated = '## Features\n\n* New SDK feature'

            steps._update_sdk_release_notes(path, '3.2.0', generated)
            first_update = path.read_text()
            steps._update_sdk_release_notes(path, '3.2.0', generated)

            self.assertEqual(path.read_text(), first_update)
            self.assertEqual(first_update.count('# 3.2.0\n'), 1)
            self.assertEqual(first_update.count('* New SDK feature'), 1)

    def test_empty_generated_notes_preserve_handwritten_notes(self):
        handwritten = '## Breaking changes\n\nKeep this migration guidance.'
        historical = '\n# 3.1.0\n\nPrevious release notes.\n'
        with TemporaryDirectory() as tmpdir:
            path = Path(tmpdir) / 'RELEASE.md'
            path.write_text(
                '# Current Version (in development)\n\n'
                + handwritten
                + '\n'
                + historical
            )

            steps._update_sdk_release_notes(path, '3.2.0', '')

            text = path.read_text()
            released = text.split('\n# 3.2.0\n', 1)[1]
            self.assertIn(handwritten, released)
            self.assertNotIn('## Generated changes', released)
            self.assertTrue(text.endswith(historical))


class QualifiedPublicationTest(unittest.TestCase):
    """Isolate existing layout/notes tests from the separately tested gates."""

    def setUp(self):
        for target in (
            'qualification.require_qualification',
            '_ensure_release_tag',
            '_should_set_backend_latest',
        ):
            patch = mock.patch(
                'kfpr.steps.' + target,
                **(
                    {'return_value': True}
                    if target == '_should_set_backend_latest'
                    else {}
                ),
            )
            patch.start()
            self.addCleanup(patch.stop)


class StepSelectionTest(unittest.TestCase):
    def test_patch_includes_cherry_pick_steps(self):
        steps_list = [
            step.step_id
            for step in steps.build_steps(
                'patch', include_backend=True, include_sdk=True
            )
        ]
        self.assertIn('cherry-pick-prs', steps_list)
        self.assertIn('publish-images', steps_list)
        self.assertIn('create-sdk-tag', steps_list)
        self.assertIn('publish-sdks', steps_list)
        self.assertIn('create-sdk-release', steps_list)
        self.assertLess(
            steps_list.index('create-sdk-tag'), steps_list.index('publish-sdks')
        )
        self.assertLess(
            steps_list.index('publish-sdks'),
            steps_list.index('create-sdk-release'),
        )
        self.assertIn('create-kfp-kubernetes-docs-branch', steps_list)
        self.assertIn('confirm-rtd', steps_list)
        self.assertNotIn('update-sdk-versions', steps_list)
        self.assertNotIn('merge-sdk-pr', steps_list)

    def test_minor_keeps_release_order_and_updates_docs(self):
        steps_list = [
            step.step_id
            for step in steps.build_steps(
                'minor', include_backend=True, include_sdk=True
            )
        ]
        self.assertLess(
            steps_list.index('publish-sdks'),
            steps_list.index('create-backend-release'),
        )
        self.assertLess(
            steps_list.index('create-sdk-tag'), steps_list.index('publish-sdks')
        )
        self.assertLess(
            steps_list.index('publish-sdks'), steps_list.index('confirm-rtd')
        )
        self.assertLess(
            steps_list.index('confirm-rtd'),
            steps_list.index('create-sdk-release'),
        )

    def test_major_skips_cherry_pick_steps(self):
        steps_list = [
            step.step_id
            for step in steps.build_steps(
                'major', include_backend=True, include_sdk=True
            )
        ]
        self.assertNotIn('cherry-pick-prs', steps_list)

    def test_backend_only_patch_skips_sdk_steps(self):
        steps_list = [
            step.step_id
            for step in steps.build_steps(
                'patch', include_backend=True, include_sdk=False
            )
        ]
        self.assertIn('publish-images', steps_list)
        self.assertNotIn('publish-sdks', steps_list)


class StepRegistryTest(unittest.TestCase):
    def test_step_registry_contains_every_built_step(self):
        for release_type in ('major', 'minor', 'patch'):
            for step in steps.build_steps(
                release_type, include_backend=True, include_sdk=True
            ):
                self.assertIn(step.step_id, steps.STEP_HANDLERS)
                self.assertIs(
                    steps.STEP_HANDLERS[step.step_id],
                    getattr(steps, step.handler),
                )


class ReleaseLayoutTest(QualifiedPublicationTest):
    """Exercise the imported CLI against both target release layouts."""

    def setUp(self) -> None:
        super().setUp()
        """Mock external operations without changing the current checkout."""
        for name, value in (
            ('_notes_range', SELECTED_RANGE),
            ('read_commits', []),
            ('_remote_release_sha', 'b' * 40),
        ):
            patcher = mock.patch.object(steps, name, return_value=value)
            patcher.start()
            self.addCleanup(patcher.stop)
        approval = mock.patch('builtins.input', return_value='y')
        approval.start()
        self.addCleanup(approval.stop)
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
            metadata=core.ReleaseMetadata.from_version('patch', '2.18.1'),
            fork_remote='origin',
            include_backend=False,
            include_sdk=True,
        )

    def select_layout(self, consolidated: bool) -> None:
        """Return the target tag's layout, opposite to the local checkout."""
        contents = 'version.py\n' + ('server_api\n' if consolidated else '')

        def capture(command: list[str], cwd: Path | None = None) -> str:
            return '' if command == ['git', 'status', '--short'] else contents

        self.runner.capture.side_effect = capture
        local_client = self.root / 'sdk/python/kfp/server_api/__init__.py'
        local_client.parent.mkdir(parents=True, exist_ok=True)
        if consolidated:
            local_client.unlink(missing_ok=True)
        else:
            local_client.write_text('from kfp.version import __version__\n')
        self.runner.run.reset_mock()
        self.runner.capture.reset_mock()

    def test_install_instructions_follow_target_layout(self) -> None:
        """Never remove Kubernetes helpers from a split-package release."""
        for version in ('2.18.1', '3.0.0'):
            for consolidated in (False, True):
                with self.subTest(version=version, consolidated=consolidated):
                    self.context.metadata = core.ReleaseMetadata.from_version(
                        'patch', version
                    )
                    self.select_layout(consolidated)
                    steps.step_create_sdk_release(self.context)
                    command = self.runner.run.call_args.args[0]
                    notes = Path(
                        command[command.index('--notes-file') + 1]
                    ).read_text()
                    Path(command[command.index('--notes-file') + 1]).unlink()
                    self.assertIn(f'pip install kfp=={version}', notes)
                    if consolidated:
                        self.assertIn('unified KFP SDK', notes)
                        self.assertIn('pip uninstall -y', notes)
                        self.assertNotIn(
                            f'pip install kfp-kubernetes=={version}', notes
                        )
                    else:
                        self.assertNotIn('pip uninstall', notes)
                        for package in (
                            'kfp-pipeline-spec',
                            'kfp-server-api',
                            'kfp-kubernetes',
                        ):
                            self.assertIn(
                                f'pip install {package}=={version}', notes
                            )
                    self.runner.capture.assert_called_once_with(
                        [
                            'gh',
                            'api',
                            f'repos/{core.REPO}/contents/sdk/python/kfp?ref=sdk-{version}',
                            '--jq',
                            '.[].name',
                        ],
                        cwd=self.root,
                    )

    def test_docs_branch_is_conditional_at_execution_time(self) -> None:
        """Keep the checkpoint, but mutate branches only for split tags."""
        step_ids = [
            step.step_id for step in steps.build_steps('patch', False, True)
        ]
        self.assertLess(
            step_ids.index('publish-sdks'),
            step_ids.index('create-kfp-kubernetes-docs-branch'),
        )
        self.assertLess(
            step_ids.index('create-kfp-kubernetes-docs-branch'),
            step_ids.index('confirm-rtd'),
        )
        for consolidated in (False, True):
            with self.subTest(consolidated=consolidated):
                self.select_layout(consolidated)
                steps.step_create_kfp_kubernetes_docs_branch(self.context)
                if consolidated:
                    self.runner.run.assert_not_called()
                else:
                    self.runner.run.assert_any_call(
                        [
                            'git',
                            'push',
                            '--set-upstream',
                            'upstream',
                            'kfp-kubernetes-2.18',
                        ],
                        cwd=self.root,
                    )

    def test_legacy_docs_reject_dirty_tree_before_branch_operations(
        self,
    ) -> None:
        """Reject staged, unstaged, and untracked changes before any
        mutation."""
        for status in (
            'M  unrelated.txt',
            ' M .readthedocs.yml',
            '?? untracked.txt',
        ):
            with self.subTest(status=status):
                self.select_layout(False)
                self.runner.capture.side_effect = ['version.py\n', status]
                with self.assertRaisesRegex(
                    RuntimeError, 'working tree is dirty'
                ):
                    steps.step_create_kfp_kubernetes_docs_branch(self.context)
                self.runner.capture.assert_called_with(
                    ['git', 'status', '--short'], cwd=self.root
                )
                self.runner.run.assert_not_called()

    def test_unified_docs_skip_working_tree_check(self) -> None:
        """A no-op for unified tags must not require a clean checkout."""
        self.select_layout(True)
        self.runner.capture.side_effect = [
            'version.py\nserver_api\n',
            AssertionError('Unified docs must not inspect the working tree'),
        ]
        steps.step_create_kfp_kubernetes_docs_branch(self.context)
        self.assertEqual(self.runner.capture.call_count, 1)
        self.runner.run.assert_not_called()

    def test_legacy_docs_stop_when_working_tree_check_fails(self) -> None:
        """A failed Git status command must not be treated as a clean tree."""
        self.select_layout(False)
        self.runner.capture.side_effect = [
            'version.py\n',
            subprocess.CalledProcessError(128, ['git', 'status', '--short']),
        ]
        with self.assertRaises(subprocess.CalledProcessError):
            steps.step_create_kfp_kubernetes_docs_branch(self.context)
        self.runner.run.assert_not_called()

    def test_resumed_legacy_docs_recheck_working_tree(self) -> None:
        """Completed preflight cannot authorize later dirty branch changes."""
        self.select_layout(False)
        step_ids = [
            step.step_id for step in steps.build_steps('patch', False, True)
        ]
        completed = step_ids[
            : step_ids.index('create-kfp-kubernetes-docs-branch')
        ]
        self.context.state.completed_steps = completed.copy()
        self.context.state.save()
        self.runner.capture.side_effect = [
            'version.py\n',
            ' M .readthedocs.yml',
        ]
        with self.assertRaisesRegex(RuntimeError, 'working tree is dirty'):
            steps.run_steps(self.context)
        self.runner.run.assert_not_called()
        self.assertEqual(self.context.state.completed_steps, completed)
        self.assertEqual(
            core.ReleaseState.load(self.context.state.path).completed_steps,
            completed,
        )

    def test_rtd_automation_and_fallback_follow_target_layout(self) -> None:
        """Automated and manual paths must cover the same projects."""
        for consolidated in (False, True):
            for fails in (False, True):
                with self.subTest(consolidated=consolidated, fails=fails):
                    self.select_layout(consolidated)
                    with (
                        mock.patch.dict('os.environ', {'RTD_TOKEN': 'test'}),
                        mock.patch.object(
                            steps.rtd, 'ReadTheDocsClient'
                        ) as client,
                        mock.patch.object(
                            steps.rtd, 'update_release_docs'
                        ) as update,
                        mock.patch.object(steps, 'confirm'),
                        contextlib.redirect_stdout(io.StringIO()) as output,
                    ):
                        if fails:
                            update.side_effect = steps.rtd.ReadTheDocsError(
                                'unavailable'
                            )
                        steps.step_confirm_rtd(self.context)
                    update.assert_called_once_with(
                        client.return_value,
                        '2.18.1',
                        'release-2.18',
                        consolidated=consolidated,
                        set_defaults=False,
                    )
                    if fails:
                        self.assertIn(
                            'ReadTheDocs manual checkpoint:', output.getvalue()
                        )
                        self.assertEqual(
                            'projects/kfp-kubernetes/' in output.getvalue(),
                            not consolidated,
                        )

    def test_pypi_polling_follows_target_layout(self) -> None:
        """Check every split distribution, but only kfp for unified tags."""
        for consolidated in (False, True):
            with self.subTest(consolidated=consolidated):
                self.select_layout(consolidated)
                with (
                    mock.patch.object(steps, 'watch_latest_workflow_run'),
                    mock.patch.object(
                        steps, '_is_pypi_version_published', return_value=True
                    ) as published,
                ):
                    steps.step_publish_sdks(self.context)
                packages = (
                    ['kfp']
                    if consolidated
                    else [
                        'kfp-pipeline-spec',
                        'kfp-server-api',
                        'kfp',
                        'kfp-kubernetes',
                    ]
                )
                self.assertEqual(
                    published.call_args_list,
                    [mock.call(package, '2.18.1') for package in packages],
                )

    def test_legacy_kubernetes_publication_resumes_only_missing_package(
        self,
    ) -> None:
        """A missing optional distribution must not report release success."""
        self.select_layout(False)
        with (
            mock.patch.object(
                steps,
                '_is_pypi_version_published',
                side_effect=lambda package, version: (
                    package != 'kfp-kubernetes'
                ),
            ),
            mock.patch.object(steps, 'watch_latest_workflow_run'),
            mock.patch.object(
                steps,
                '_wait_for_pypi_packages',
                return_value=[],
            ) as wait,
        ):
            steps.step_publish_sdks(self.context)
        wait.assert_called_once_with('2.18.1', ['kfp-kubernetes'])
        self.runner.run.assert_any_call(
            core.sdk_workflow_command(
                self.context.metadata,
                packages='kfp-kubernetes',
                workflow_ref=self.context.metadata.sdk_tag,
            )
        )

    def test_missing_tag_layout_fails_before_mutations(self) -> None:
        """Do not guess a legacy layout if target inspection fails."""
        for handler in (
            steps.step_create_sdk_release,
            steps.step_publish_sdks,
            steps.step_create_kfp_kubernetes_docs_branch,
            steps.step_confirm_rtd,
        ):
            with self.subTest(handler=handler.__name__):
                self.runner.capture.return_value = ''
                self.runner.run.reset_mock()
                with self.assertRaisesRegex(RuntimeError, 'packaging layout'):
                    handler(self.context)
                self.runner.run.assert_not_called()

    def test_resume_runs_the_matching_sdk_release_flow(self) -> None:
        """Use current handlers for legacy and unified release checkpoints."""
        for version, consolidated in (('2.18.1', False), ('3.0.0', True)):
            with self.subTest(version=version):
                self.context.metadata = core.ReleaseMetadata.from_version(
                    'patch', version
                )
                self.select_layout(consolidated)
                step_ids = [
                    step.step_id
                    for step in steps.build_steps('patch', False, True)
                ]
                self.context.state.completed_steps = step_ids[
                    : step_ids.index('publish-sdks')
                ]
                with (
                    mock.patch.dict('os.environ', {'RTD_TOKEN': 'test'}),
                    mock.patch.object(steps, 'watch_latest_workflow_run'),
                    mock.patch.object(
                        steps, '_is_pypi_version_published', return_value=True
                    ),
                    mock.patch.object(steps.rtd, 'ReadTheDocsClient') as client,
                ):
                    client.return_value.get_build.return_value = {
                        'state': {'code': 'finished'},
                        'success': True,
                    }
                    steps.run_steps(self.context)
                self.assertEqual(
                    core.ReleaseState.load(
                        self.context.state.path
                    ).completed_steps,
                    step_ids,
                )
                expected_projects = ['kubeflow-pipelines']
                if not consolidated:
                    expected_projects.append('kfp-kubernetes')
                self.assertEqual(
                    client.return_value.sync_versions.call_args_list,
                    [mock.call(project) for project in expected_projects],
                )
                command = self.runner.run.call_args.args[0]
                self.assertEqual(
                    command[:4], ['gh', 'release', 'create', f'sdk-{version}']
                )
                notes = Path(
                    command[command.index('--notes-file') + 1]
                ).read_text()
                self.assertEqual('pip uninstall' in notes, consolidated)
                self.assertEqual(
                    any(
                        call.args[0][:2] == ['git', 'push']
                        for call in self.runner.run.call_args_list
                    ),
                    not consolidated,
                )

    def test_dry_run_defers_tag_inspection_and_external_operations(
        self,
    ) -> None:
        """An uncreated SDK tag must not prevent a read-only preview."""
        self.runner.dry_run = True
        with (
            mock.patch.object(steps, 'watch_latest_workflow_run'),
            mock.patch.object(
                steps.subprocess, 'check_output', return_value='version.py\n'
            ),
            mock.patch.object(steps.rtd, 'ReadTheDocsClient') as client,
            mock.patch.object(steps, '_wait_for_pypi_packages') as wait,
            contextlib.redirect_stdout(io.StringIO()) as output,
        ):
            steps.step_publish_sdks(self.context)
            self.runner.run.assert_not_called()
            self.runner.run.reset_mock()
            steps.step_create_kfp_kubernetes_docs_branch(self.context)
            steps.step_confirm_rtd(self.context)
            steps.step_create_sdk_release(self.context)
        self.runner.capture.assert_not_called()
        self.runner.run.assert_not_called()
        client.assert_not_called()
        wait.assert_not_called()
        self.assertIn('split-package SDK tags only', output.getvalue())
        self.assertIn('kfp-kubernetes-2.18', output.getvalue())


class CherryPickStepTest(unittest.TestCase):
    def test_cherry_pick_rejects_pr_without_merge_commit(self):
        with TemporaryDirectory() as tmpdir:
            state = core.ReleaseState(Path(tmpdir) / 'state.json')
            state.answers['patch_prs'] = '123'

            class NullMergeCommitRunner:
                dry_run = False

                def __init__(self):
                    self.commands = []

                def capture(self, command, cwd=None):
                    self.commands.append(command)
                    return '' if command[-1].endswith('// ""') else 'null'

                def run(self, command, cwd=None):
                    self.commands.append(command)

            runner = NullMergeCommitRunner()
            context = core.ReleaseContext(
                root=Path(tmpdir),
                state=state,
                runner=runner,
                metadata=core.ReleaseMetadata.from_version('patch', '1.0.1'),
                fork_remote='origin',
                include_backend=True,
                include_sdk=True,
            )

            with self.assertRaisesRegex(
                RuntimeError, 'Could not find merge commit for PR 123'
            ):
                steps.step_cherry_pick_prs(context)

            self.assertNotIn(['git', 'cherry-pick', 'null'], runner.commands)

    def test_cherry_pick_dry_run_no_malformed_command(self):
        with TemporaryDirectory() as tmpdir:
            state_file = Path(tmpdir) / 'state.json'
            state = core.ReleaseState(state_file)
            state.answers['patch_prs'] = '123'

            runner = core.CommandRunner(dry_run=True)
            metadata = core.ReleaseMetadata.from_version('patch', '1.0.1')
            context = core.ReleaseContext(
                root=Path(tmpdir),
                state=state,
                runner=runner,
                metadata=metadata,
                fork_remote='origin',
                include_backend=True,
                include_sdk=True,
            )

            steps.step_cherry_pick_prs(context)

            for command in runner.commands:
                self.assertNotEqual(command, ['git', 'cherry-pick', ''])


class HandlerExistenceTest(unittest.TestCase):
    def test_all_handlers_exist_for_patch_release(self):
        steps_list = steps.build_steps(
            'patch', include_backend=True, include_sdk=True
        )
        for step in steps_list:
            with self.subTest(handler=step.handler):
                self.assertTrue(
                    hasattr(steps, step.handler),
                    f'Handler {step.handler} for step {step.step_id} does not exist in steps module',
                )
                handler_fn = getattr(steps, step.handler)
                self.assertTrue(
                    callable(handler_fn),
                    f'Handler {step.handler} is not callable',
                )


class PreflightStepTest(unittest.TestCase):
    def test_preflight_checks_host_tools_used_by_selected_steps(self):
        with TemporaryDirectory() as tmpdir:
            context = core.ReleaseContext(
                root=Path(tmpdir),
                state=core.ReleaseState(Path(tmpdir) / 'state.json'),
                runner=core.CommandRunner(dry_run=True),
                metadata=core.ReleaseMetadata.from_version('minor', '3.2.0'),
                fork_remote='git@github.com:testuser/pipelines.git',
                include_backend=True,
                include_sdk=True,
            )

            steps.step_preflight(context)

            self.assertIn(['which', 'sed'], context.runner.commands)
            self.assertIn(['which', 'pip-compile'], context.runner.commands)

            (Path(tmpdir) / 'uv.lock').touch()
            context.runner.commands.clear()
            steps.step_preflight(context)
            self.assertIn(['which', 'uv'], context.runner.commands)
            self.assertNotIn(['which', 'pip-compile'], context.runner.commands)

    def test_preflight_rejects_dirty_working_tree(self):

        class DirtyRunner(core.CommandRunner):
            def capture(self, command, cwd=None):
                if command == ['git', 'status', '--short']:
                    return ' M release/kfpr/steps.py'
                return ''

        with TemporaryDirectory() as tmpdir:
            context = core.ReleaseContext(
                root=Path(tmpdir),
                state=core.ReleaseState(Path(tmpdir) / 'state.json'),
                runner=DirtyRunner(dry_run=True),
                metadata=core.ReleaseMetadata.from_version('minor', '3.2.0'),
                fork_remote='git@github.com:testuser/pipelines.git',
                include_backend=True,
                include_sdk=True,
            )

            with self.assertRaisesRegex(RuntimeError, 'working tree is dirty'):
                steps.step_preflight(context)


class PublishImagesStepTest(QualifiedPublicationTest):
    def test_failed_publication_does_not_complete_checkpoint(self):
        for release_type, version in (
            ('minor', '2.18.0'),
            ('major', '3.0.0'),
            ('patch', '3.0.1'),
        ):
            with self.subTest(version=version), TemporaryDirectory() as tmpdir:
                runner = mock.Mock(dry_run=False)
                runner.capture.return_value = (
                    '12345\thttps://github.com/kubeflow/pipelines/actions/runs/12345'
                    '\t2026-07-08T19:05:01Z'
                )

                def fail_watch(command):
                    if command[:3] == ['gh', 'run', 'watch']:
                        self.assertIn('--exit-status', command)
                        raise subprocess.CalledProcessError(1, command)

                runner.run.side_effect = fail_watch
                state = core.ReleaseState(Path(tmpdir) / 'state.json')
                selected = steps.build_steps(release_type, True, False)
                state.completed_steps = [
                    step.step_id
                    for step in selected[
                        : next(
                            i
                            for i, step in enumerate(selected)
                            if step.step_id == 'publish-images'
                        )
                    ]
                ]
                before = list(state.completed_steps)
                context = core.ReleaseContext(
                    root=Path(tmpdir),
                    state=state,
                    runner=runner,
                    metadata=core.ReleaseMetadata.from_version(
                        release_type, version
                    ),
                    fork_remote='origin',
                    include_backend=True,
                    include_sdk=False,
                )
                with mock.patch('time.time', return_value=1783537500):
                    with self.assertRaises(subprocess.CalledProcessError):
                        steps.run_steps(context)
                self.assertEqual(state.completed_steps, before)
                self.assertFalse(state.path.exists())

    def test_publish_images_prints_workflow_run_url(self):
        with TemporaryDirectory() as tmpdir:

            class WorkflowRunRunner:
                dry_run = False

                def __init__(self):
                    self.commands = []

                def capture(self, command, cwd=None):
                    self.commands.append(command)
                    return '12345\thttps://github.com/kubeflow/pipelines/actions/runs/12345\t2026-07-08T19:05:01Z'

                def run(self, command, cwd=None, check=True):
                    self.commands.append(command)

            context = core.ReleaseContext(
                root=Path(tmpdir),
                state=core.ReleaseState(Path(tmpdir) / 'state.json'),
                runner=WorkflowRunRunner(),
                metadata=core.ReleaseMetadata.from_version('minor', '3.2.0'),
                fork_remote='origin',
                include_backend=True,
                include_sdk=True,
            )

            with (
                mock.patch('time.time', return_value=1783537500),
                mock.patch('builtins.print') as print_mock,
            ):
                steps.step_publish_images(context)

            output = '\n'.join(
                str(call.args[0]) for call in print_mock.call_args_list
            )
            self.assertIn(
                'Workflow run: \033[4mhttps://github.com/kubeflow/pipelines/actions/runs/12345\033[0m',
                output,
            )
            self.assertIn(
                ['gh', 'run', 'watch', '12345', '--exit-status'],
                context.runner.commands,
            )


class CreateSdkReleaseStepTest(QualifiedPublicationTest):
    def setUp(self) -> None:
        super().setUp()
        """Keep release-reuse tests independent of remote layout inspection."""
        layout = mock.patch.object(
            steps, '_sdk_release_is_consolidated', return_value=True
        )
        layout.start()
        self.addCleanup(layout.stop)

    def test_create_sdk_release_reuses_existing_release_when_user_accepts(self):
        with TemporaryDirectory() as tmpdir:

            class ExistingReleaseRunner:
                dry_run = False

                def __init__(self):
                    self.commands = []

                def run(self, command, cwd=None, check=True):
                    self.commands.append(command)
                    if command[:4] == ['gh', 'release', 'view', 'sdk-3.2.0']:
                        return subprocess.CompletedProcess(command, 0, '', '')
                    return subprocess.CompletedProcess(command, 0, '', '')

            context = core.ReleaseContext(
                root=Path(tmpdir),
                state=core.ReleaseState(Path(tmpdir) / 'state.json'),
                runner=ExistingReleaseRunner(),
                metadata=core.ReleaseMetadata.from_version('minor', '3.2.0'),
                fork_remote='origin',
                include_backend=True,
                include_sdk=True,
            )

            with mock.patch('builtins.input', return_value='u'):
                steps.step_create_sdk_release(context)

            self.assertIn(
                [
                    'gh',
                    'release',
                    'view',
                    'sdk-3.2.0',
                    '--repo',
                    'kubeflow/pipelines',
                ],
                context.runner.commands,
            )
            self.assertNotIn(
                ['gh', 'release', 'create', 'sdk-3.2.0'],
                [command[:4] for command in context.runner.commands],
            )


class CreateBackendReleaseStepTest(QualifiedPublicationTest):
    def setUp(self):
        super().setUp()
        patcher = mock.patch.object(
            steps,
            'resolve_range',
            side_effect=lambda root, base, target: ReleaseRange(
                base, target, 'a' * 40, 'b' * 40
            ),
        )
        patcher.start()
        self.addCleanup(patcher.stop)
        patcher = mock.patch.object(
            steps, '_remote_release_sha', return_value='b' * 40
        )
        patcher.start()
        self.addCleanup(patcher.stop)

    def test_architecture_checkpoint_and_communication_begin_with_backend_3(
        self,
    ):
        for release_type, version, supported in (
            ('minor', '2.18.0', False),
            ('patch', '2.18.1', False),
            ('major', '3.0.0', True),
            ('patch', '3.0.1', True),
        ):
            with self.subTest(version=version):
                metadata = core.ReleaseMetadata.from_version(
                    release_type, version
                )
                checklist = steps.manual_checklist(
                    'create-backend-release', metadata
                )
                announcement = steps.manual_checklist(
                    'confirm-website-and-slack', metadata
                )
                self.assertEqual(bool(checklist), supported)
                self.assertEqual('ARM64' in announcement, supported)
                if supported:
                    self.assertIn(f'target tag\n   {version}', checklist)
                    self.assertIn(
                        'native ARM64 installation/pipeline smoke', checklist
                    )
                sdk_only = [
                    step.step_id
                    for step in steps.build_steps(
                        release_type, include_backend=False, include_sdk=True
                    )
                ]
                self.assertNotIn('create-backend-release', sdk_only)
                self.assertNotIn('confirm-website-and-slack', sdk_only)
                self.assertNotIn('publish-images', sdk_only)

    def test_declining_architecture_evidence_stops_before_creating_release(
        self,
    ):
        with TemporaryDirectory() as tmpdir:
            runner = mock.Mock(dry_run=False)
            context = core.ReleaseContext(
                root=Path(tmpdir),
                state=core.ReleaseState(Path(tmpdir) / 'state.json'),
                runner=runner,
                metadata=core.ReleaseMetadata.from_version('major', '3.0.0'),
                fork_remote='origin',
                include_backend=True,
                include_sdk=False,
            )
            with mock.patch('builtins.input', return_value='n'):
                with self.assertRaises(SystemExit):
                    steps.step_create_backend_release(context)
            runner.run.assert_not_called()

    def test_2_x_release_has_no_architecture_claim_or_confirmation(self):
        with TemporaryDirectory() as tmpdir:
            runner = mock.Mock(dry_run=False)
            context = core.ReleaseContext(
                root=Path(tmpdir),
                state=core.ReleaseState(Path(tmpdir) / 'state.json'),
                runner=runner,
                metadata=core.ReleaseMetadata.from_version('minor', '2.18.0'),
                fork_remote='origin',
                include_backend=True,
                include_sdk=False,
            )
            with (
                mock.patch(
                    'builtins.input', side_effect=['2.17.0', 'y']
                ) as input_mock,
                mock.patch.object(
                    steps,
                    '_generate_backend_release_notes',
                    return_value='Fixes',
                ),
            ):
                steps.step_create_backend_release(context)
            self.assertEqual(input_mock.call_count, 2)
            self.assertFalse(
                any(
                    'architecture checks' in call.args[0]
                    for call in input_mock.call_args_list
                )
            )
            command = runner.run.call_args.args[0]
            notes_path = Path(command[command.index('--notes-file') + 1])
            self.assertNotIn('ARM64', notes_path.read_text())

    def test_create_backend_release_prompts_for_previous_tag_and_generates_notes(
        self,
    ):
        with TemporaryDirectory() as tmpdir:

            class BackendReleaseRunner:
                dry_run = False

                def __init__(self):
                    self.commands = []

                def run(self, command, cwd=None, check=True):
                    self.commands.append(command)
                    return subprocess.CompletedProcess(command, 0, '', '')

            context = core.ReleaseContext(
                root=Path(tmpdir),
                state=core.ReleaseState(Path(tmpdir) / 'state.json'),
                runner=BackendReleaseRunner(),
                metadata=core.ReleaseMetadata.from_version('minor', '3.2.0'),
                fork_remote='origin',
                include_backend=True,
                include_sdk=True,
            )

            with (
                mock.patch(
                    'builtins.input', side_effect=['y', '3.1.0', 'y']
                ) as input_mock,
                mock.patch.object(
                    steps,
                    '_generate_backend_release_notes',
                    return_value='## Features\n\n* add pipelines (#1)',
                ) as notes_mock,
            ):
                steps.step_create_backend_release(context)

            self.assertEqual(input_mock.call_count, 3)
            self.assertEqual(notes_mock.call_args.args[0], Path(tmpdir))
            self.assertEqual(notes_mock.call_args.args[1].base_ref, '3.1.0')
            release_command = context.runner.commands[-1]
            self.assertEqual(
                release_command[:4], ['gh', 'release', 'create', '3.2.0']
            )
            release_notes = Path(
                release_command[release_command.index('--notes-file') + 1]
            ).read_text()
            self.assertIn('## Features\n\n* add pipelines (#1)', release_notes)
            self.assertIn('Linux AMD64 and ARM64 are supported', release_notes)
            self.assertIn(
                'https://github.com/kubeflow/pipelines/compare/'
                + 'a' * 40
                + '...'
                + 'b' * 40,
                release_notes,
            )


class PublishSdksStepTest(QualifiedPublicationTest):
    def setUp(self) -> None:
        super().setUp()
        """Use unified tags for existing publication retry tests."""
        layout = mock.patch.object(
            steps, '_sdk_release_is_consolidated', return_value=True
        )
        layout.start()
        self.addCleanup(layout.stop)
        published = mock.patch.object(
            steps, '_is_pypi_version_published', return_value=False
        )
        self.published = published
        published.start()
        self.addCleanup(published.stop)

    def test_publish_sdks_publishes_missing_unified_package(self):
        with TemporaryDirectory() as tmpdir:

            class NoopRunner(core.CommandRunner):
                def run(self, command, cwd=None, check=True):
                    self.commands.append(command)
                    return subprocess.CompletedProcess(command, 0, '', '')

            runner = NoopRunner(dry_run=False)
            context = core.ReleaseContext(
                root=Path(tmpdir),
                state=core.ReleaseState(Path(tmpdir) / 'state.json'),
                runner=runner,
                metadata=core.ReleaseMetadata.from_version('minor', '3.2.0'),
                fork_remote='origin',
                include_backend=True,
                include_sdk=True,
            )

            with (
                mock.patch.object(steps, 'watch_latest_workflow_run'),
                mock.patch.object(
                    steps, '_wait_for_pypi_packages', return_value=[]
                ) as wait_mock,
            ):
                steps.step_publish_sdks(context)

            workflow_commands = [
                command
                for command in runner.commands
                if command[:4]
                == ['gh', 'workflow', 'run', 'publish-packages.yml']
            ]
            self.assertEqual(len(workflow_commands), 1)
            self.assertIn('packages=kfp', workflow_commands[0])
            wait_mock.assert_called_once_with('3.2.0', ['kfp'])

    def test_publish_sdks_fails_when_pypi_package_does_not_become_available(
        self,
    ):
        with TemporaryDirectory() as tmpdir:

            class NoopRunner(core.CommandRunner):
                def run(self, command, cwd=None, check=True):
                    self.commands.append(command)
                    return subprocess.CompletedProcess(command, 0, '', '')

            runner = NoopRunner(dry_run=False)
            context = core.ReleaseContext(
                root=Path(tmpdir),
                state=core.ReleaseState(Path(tmpdir) / 'state.json'),
                runner=runner,
                metadata=core.ReleaseMetadata.from_version('minor', '3.2.0'),
                fork_remote='origin',
                include_backend=True,
                include_sdk=True,
            )

            with (
                mock.patch.object(steps, 'watch_latest_workflow_run'),
                mock.patch.object(
                    steps, '_wait_for_pypi_packages', return_value=['kfp']
                ),
            ):
                with self.assertRaisesRegex(
                    RuntimeError, 'PyPI packages are still missing'
                ):
                    steps.step_publish_sdks(context)

    def test_pypi_package_check_uses_package_version_json(self):
        self.published.stop()
        opened_urls = []

        def fake_urlopen(request, timeout):
            opened_urls.append(request.full_url)
            return mock.Mock(status=200)

        with mock.patch('urllib.request.urlopen', side_effect=fake_urlopen):
            self.assertTrue(steps._is_pypi_version_published('kfp', '3.2.0'))

        self.assertEqual(opened_urls, ['https://pypi.org/pypi/kfp/3.2.0/json'])


class SyncMasterStepTest(QualifiedPublicationTest):
    def test_sync_master_opens_pr_and_waits_for_merge(self):
        with TemporaryDirectory() as tmpdir:
            root = Path(tmpdir)

            class SyncMasterRunner(core.CommandRunner):
                def capture(self, command, cwd=None):
                    self.commands.append(command)
                    if command[:3] == ['gh', 'pr', 'create']:
                        return 'https://github.com/kubeflow/pipelines/pull/123'
                    raise AssertionError(f'unexpected capture: {command}')

            context = core.ReleaseContext(
                root=root,
                state=core.ReleaseState(root / 'state.json'),
                runner=SyncMasterRunner(dry_run=True),
                metadata=core.ReleaseMetadata.from_version('minor', '3.2.0'),
                fork_remote='git@github.com:testuser/pipelines.git',
                include_backend=True,
                include_sdk=True,
            )

            with (
                mock.patch.object(steps, 'watch_pr_ci') as watch_ci,
                mock.patch.object(steps, 'wait_for_pr_merge') as wait_merge,
            ):
                steps.step_sync_master(context)

            self.assertIn(
                [
                    'git',
                    'push',
                    '--set-upstream',
                    'git@github.com:testuser/pipelines.git',
                    'chore-release-3.2.0-master',
                ],
                context.runner.commands,
            )
            self.assertIn(
                ['git', 'checkout', '3.2.0', '--', 'CHANGELOG.md'],
                context.runner.commands,
            )
            self.assertIn(
                ['gh', 'pr', 'create'],
                [command[:3] for command in context.runner.commands],
            )
            watch_ci.assert_called_once_with(
                context.runner, 'https://github.com/kubeflow/pipelines/pull/123'
            )
            wait_merge.assert_called_once_with(
                context.runner, 'https://github.com/kubeflow/pipelines/pull/123'
            )

    def test_sync_master_prompts_for_local_review_before_commit(self):
        with TemporaryDirectory() as tmpdir:
            root = Path(tmpdir)

            class SyncMasterRunner(core.CommandRunner):
                def run(self, command, cwd=None, check=True):
                    self.commands.append(command)
                    return subprocess.CompletedProcess(command, 0, '', '')

                def capture(self, command, cwd=None):
                    self.commands.append(command)
                    if command[:3] == ['gh', 'pr', 'create']:
                        return 'https://github.com/kubeflow/pipelines/pull/123'
                    return ''

            context = core.ReleaseContext(
                root=root,
                state=core.ReleaseState(root / 'state.json'),
                runner=SyncMasterRunner(dry_run=False),
                metadata=core.ReleaseMetadata.from_version('minor', '3.2.0'),
                fork_remote='git@github.com:testuser/pipelines.git',
                include_backend=True,
                include_sdk=True,
            )

            with (
                mock.patch('sys.stdin.isatty', return_value=True),
                mock.patch('builtins.input', return_value='y') as input_mock,
                mock.patch.object(steps, 'watch_pr_ci'),
                mock.patch.object(steps, 'wait_for_pr_merge'),
            ):
                steps.step_sync_master(context)

            self.assertEqual((root / 'VERSION').read_bytes(), b'3.2.0\n')

            status_index = context.runner.commands.index(
                ['git', 'status', '--short']
            )
            commit_index = context.runner.commands.index(
                [
                    'git',
                    'commit',
                    '-s',
                    '-m',
                    'chore(release): bump version to 3.2.0 on master branch',
                ]
            )
            self.assertLess(status_index, commit_index)
            input_mock.assert_called()

    def test_sync_master_skip_local_review_bypasses_commit_prompt(self):
        with TemporaryDirectory() as tmpdir:
            root = Path(tmpdir)
            state = core.ReleaseState(root / 'state.json')
            state.answers['skip_local_review'] = True

            class SyncMasterRunner(core.CommandRunner):
                def run(self, command, cwd=None, check=True):
                    self.commands.append(command)
                    return subprocess.CompletedProcess(command, 0, '', '')

                def capture(self, command, cwd=None):
                    self.commands.append(command)
                    if command[:3] == ['gh', 'pr', 'create']:
                        return 'https://github.com/kubeflow/pipelines/pull/123'
                    return ''

            context = core.ReleaseContext(
                root=root,
                state=state,
                runner=SyncMasterRunner(dry_run=False),
                metadata=core.ReleaseMetadata.from_version('minor', '3.2.0'),
                fork_remote='git@github.com:testuser/pipelines.git',
                include_backend=True,
                include_sdk=True,
            )

            with (
                mock.patch('builtins.input') as input_mock,
                mock.patch.object(steps, 'watch_pr_ci'),
                mock.patch.object(steps, 'wait_for_pr_merge'),
            ):
                steps.step_sync_master(context)

            self.assertNotIn(
                ['git', 'status', '--short'], context.runner.commands
            )
            input_mock.assert_not_called()


class PrepareReleaseBranchStepTest(unittest.TestCase):
    def test_prepare_release_branch_uses_selected_source_branch(self):
        with TemporaryDirectory() as tmpdir:
            state = core.ReleaseState(Path(tmpdir) / 'state.json')
            state.answers['release_source_branch'] = 'release-candidate'
            context = core.ReleaseContext(
                root=Path(tmpdir),
                state=state,
                runner=core.CommandRunner(dry_run=True),
                metadata=core.ReleaseMetadata.from_version('minor', '3.2.0'),
                fork_remote='origin',
                include_backend=True,
                include_sdk=True,
            )

            steps.step_prepare_release_branch(context)

            self.assertIn(
                ['git', 'checkout', 'release-candidate'],
                context.runner.commands,
            )
            self.assertNotIn(
                ['git', 'checkout', 'master'], context.runner.commands
            )

    def test_prepare_release_branch_skips_pull_without_source_upstream(self):
        with TemporaryDirectory() as tmpdir:
            state = core.ReleaseState(Path(tmpdir) / 'state.json')
            state.answers['release_source_branch'] = 'release-candidate'

            class NoUpstreamRunner(core.CommandRunner):
                def __init__(self):
                    super().__init__(dry_run=True)

                def run(self, command, cwd=None, check=True):
                    self.commands.append(command)
                    if command == [
                        'git',
                        'rev-parse',
                        '--abbrev-ref',
                        '--symbolic-full-name',
                        '@{u}',
                    ]:
                        return subprocess.CompletedProcess(command, 1, '', '')
                    return subprocess.CompletedProcess(command, 0, '', '')

            context = core.ReleaseContext(
                root=Path(tmpdir),
                state=state,
                runner=NoUpstreamRunner(),
                metadata=core.ReleaseMetadata.from_version('minor', '3.2.0'),
                fork_remote='origin',
                include_backend=True,
                include_sdk=True,
            )

            steps.step_prepare_release_branch(context)

            self.assertNotIn(
                ['git', 'pull', '--ff-only'], context.runner.commands
            )
            self.assertIn(
                ['git', 'checkout', '-b', 'release-3.2'],
                context.runner.commands,
            )


class OrchestrationTest(unittest.TestCase):
    def test_full_dry_run_reaches_end_without_simulated_release_refs(self):
        for release_type, version in (('major', '4.0.0'), ('minor', '3.3.0')):
            with (
                self.subTest(release_type=release_type),
                TemporaryDirectory() as tmpdir,
            ):
                root = Path(tmpdir)
                state = core.ReleaseState(root / 'state.json')
                context = core.ReleaseContext(
                    root=root,
                    state=state,
                    runner=core.CommandRunner(dry_run=True),
                    metadata=core.ReleaseMetadata.from_version(
                        release_type, version
                    ),
                    fork_remote='git@github.com:testuser/pipelines.git',
                    include_backend=True,
                    include_sdk=True,
                    previous_release='3.2.0',
                    previous_sdk_release='sdk-3.2.0',
                )
                with (
                    contextlib.redirect_stdout(io.StringIO()) as output,
                    mock.patch.object(steps, '_notes_range') as notes_range,
                    mock.patch.object(
                        subprocess,
                        'run',
                        side_effect=AssertionError('unexpected live command'),
                    ),
                    mock.patch.object(
                        subprocess,
                        'check_output',
                        side_effect=AssertionError('unexpected live command'),
                    ),
                ):
                    steps.run_steps(context)
                notes_range.assert_not_called()
                self.assertIn(
                    'Confirm website PR and Slack announcement',
                    output.getvalue(),
                )
                self.assertIn(
                    'kfpr run create-sdk-release --dry-run', output.getvalue()
                )
                self.assertIn(
                    'kfpr run create-backend-release --dry-run',
                    output.getvalue(),
                )
                self.assertEqual(list(root.iterdir()), [])
                self.assertFalse(state.completed_steps)

    def test_run_steps_skips_completed_steps(self):
        with TemporaryDirectory() as tmpdir:
            state_file = Path(tmpdir) / 'state.json'
            state = core.ReleaseState(state_file)
            state.mark_done('preflight')
            state.save()

            runner = core.CommandRunner(dry_run=True)
            metadata = core.ReleaseMetadata.from_version('major', '3.0.0')
            context = core.ReleaseContext(
                root=Path(tmpdir),
                state=state,
                runner=runner,
                metadata=metadata,
                fork_remote='origin',
                include_backend=False,
                include_sdk=False,
            )

            self.assertTrue(context.state.is_done('preflight'))
            self.assertFalse(context.state.is_done('prepare-release-branch'))

    def test_build_steps_includes_all_final_steps(self):
        for release_type in ['major', 'minor', 'patch']:
            for include_backend in [True, False]:
                for include_sdk in [True, False]:
                    with self.subTest(
                        release_type=release_type,
                        backend=include_backend,
                        sdk=include_sdk,
                    ):
                        steps_list = steps.build_steps(
                            release_type, include_backend, include_sdk
                        )
                        step_ids = [step.step_id for step in steps_list]

                        if include_backend:
                            self.assertIn('create-backend-release', step_ids)
                            self.assertIn('sync-master', step_ids)
                            self.assertIn('confirm-website-and-slack', step_ids)
                        else:
                            self.assertNotIn('create-backend-release', step_ids)
                            self.assertNotIn('sync-master', step_ids)
                            self.assertNotIn(
                                'confirm-website-and-slack', step_ids
                            )

    def test_run_steps_dry_run_does_not_persist_completed_state(self):
        with TemporaryDirectory() as tmpdir:
            state_file = Path(tmpdir) / 'state.json'
            state = core.ReleaseState(state_file)
            state.answers['version'] = '3.0.0'
            state.answers['release_type'] = 'major'
            state.answers['fork_remote'] = (
                'git@github.com:testuser/pipelines.git'
            )
            state.answers['include_backend'] = False
            state.answers['include_sdk'] = False
            state.save()

            runner = core.CommandRunner(dry_run=True)
            metadata = core.ReleaseMetadata.from_version('major', '3.0.0')
            context = core.ReleaseContext(
                root=Path(tmpdir),
                state=state,
                runner=runner,
                metadata=metadata,
                fork_remote='git@github.com:testuser/pipelines.git',
                include_backend=False,
                include_sdk=False,
            )

            steps.run_steps(context)

            loaded_state = core.ReleaseState.load(state_file)
            self.assertEqual(loaded_state.completed_steps, [])

    def test_run_steps_dry_run_adds_blank_line_between_steps(self):
        with TemporaryDirectory() as tmpdir:
            root = Path(tmpdir)
            context = core.ReleaseContext(
                root=root,
                state=core.ReleaseState(root / 'state.json'),
                runner=core.CommandRunner(dry_run=True),
                metadata=core.ReleaseMetadata.from_version('major', '3.0.0'),
                fork_remote='git@github.com:testuser/pipelines.git',
                include_backend=False,
                include_sdk=False,
            )
            fake_steps = [
                steps.Step('first', 'First step', 'noop_step'),
                steps.Step('second', 'Second step', 'noop_step'),
            ]

            with mock.patch.object(
                steps, 'build_steps', return_value=fake_steps
            ):
                with mock.patch.object(
                    steps,
                    'noop_step',
                    create=True,
                    side_effect=lambda context: None,
                ):
                    with mock.patch('builtins.print') as print_mock:
                        steps.run_steps(context)

            output = '\n'.join(
                str(call.args[0]) if call.args else ''
                for call in print_mock.call_args_list
            )
            self.assertIn(
                'Running step: First step\n\nRunning step: Second step', output
            )


class StepPRCreationTest(unittest.TestCase):
    def setUp(self):
        patch = mock.patch(
            'kfpr.steps.validate_release_pr', return_value='a' * 40
        )
        patch.start()
        self.addCleanup(patch.stop)

    def test_step_merge_version_pr_dry_run_uses_fork_head(self):
        with TemporaryDirectory() as tmpdir:
            state_file = Path(tmpdir) / 'state.json'
            state = core.ReleaseState(state_file)

            class RecordingRunner:
                dry_run = False

                def __init__(self):
                    self.commands = []

                def run(self, command, cwd=None, check=True):
                    self.commands.append(command)
                    if command[:4] == [
                        'git',
                        'ls-remote',
                        '--exit-code',
                        '--heads',
                    ]:
                        return subprocess.CompletedProcess(command, 1, '', '')
                    if command[:4] == [
                        'git',
                        'ls-remote',
                        '--exit-code',
                        '--tags',
                    ]:
                        return subprocess.CompletedProcess(command, 1, '', '')
                    return subprocess.CompletedProcess(command, 0, '', '')

                def capture(self, command, cwd=None):
                    if command[:3] == ['gh', 'pr', 'list']:
                        return '[]'
                    self.commands.append(command)
                    if 'view' in command and any(
                        'statusCheckRollup' in part for part in command
                    ):
                        return '[{"name":"ci","status":"COMPLETED","conclusion":"SUCCESS"}]'
                    if 'view' in command and any(
                        'mergeStateStatus' in part for part in command
                    ):
                        return 'MERGED CLEAN'
                    return 'https://github.com/kubeflow/pipelines/pull/1'

            runner = RecordingRunner()
            metadata = core.ReleaseMetadata.from_version('minor', '1.3.0')
            context = core.ReleaseContext(
                root=Path(tmpdir),
                state=state,
                runner=runner,
                metadata=metadata,
                fork_remote='git@github.com:testuser/pipelines.git',
                include_backend=True,
                include_sdk=True,
            )

            steps.step_merge_version_pr(context)

            pr_create_cmd = None
            for command in runner.commands:
                if 'gh' in command and 'pr' in command and 'create' in command:
                    pr_create_cmd = command
                    break

            self.assertIsNotNone(pr_create_cmd)
            head_idx = pr_create_cmd.index('--head')
            head_value = pr_create_cmd[head_idx + 1]
            self.assertEqual(head_value, 'testuser:1.3.0-update-version-tags')

    def test_step_merge_version_pr_pushes_missing_upstream_release_branch(self):
        with TemporaryDirectory() as tmpdir:

            class MissingUpstreamReleaseBranchRunner:
                dry_run = False

                def __init__(self):
                    self.commands = []

                def run(self, command, cwd=None, check=True):
                    self.commands.append(command)
                    if command[:4] == [
                        'git',
                        'ls-remote',
                        '--exit-code',
                        '--heads',
                    ]:
                        return subprocess.CompletedProcess(command, 1, '', '')
                    if command[:4] == [
                        'git',
                        'ls-remote',
                        '--exit-code',
                        '--tags',
                    ]:
                        return subprocess.CompletedProcess(command, 1, '', '')
                    return subprocess.CompletedProcess(command, 0, '', '')

                def capture(self, command, cwd=None):
                    if command[:3] == ['gh', 'pr', 'list']:
                        return '[]'
                    self.commands.append(command)
                    if 'view' in command and any(
                        'statusCheckRollup' in part for part in command
                    ):
                        return '[{"name":"ci","status":"COMPLETED","conclusion":"SUCCESS"}]'
                    if 'view' in command and any(
                        'mergeStateStatus' in part for part in command
                    ):
                        return 'MERGED CLEAN'
                    return 'https://github.com/kubeflow/pipelines/pull/1'

            runner = MissingUpstreamReleaseBranchRunner()
            context = core.ReleaseContext(
                root=Path(tmpdir),
                state=core.ReleaseState(Path(tmpdir) / 'state.json'),
                runner=runner,
                metadata=core.ReleaseMetadata.from_version('minor', '1.3.0'),
                fork_remote='git@github.com:testuser/pipelines.git',
                include_backend=True,
                include_sdk=True,
            )

            steps.step_merge_version_pr(context)

            self.assertIn(
                [
                    'git',
                    'push',
                    'https://github.com/kubeflow/pipelines.git',
                    'release-1.3',
                ],
                runner.commands,
            )

    def test_step_merge_version_pr_skips_existing_upstream_release_branch_push(
        self,
    ):
        with TemporaryDirectory() as tmpdir:

            class ExistingUpstreamReleaseBranchRunner:
                dry_run = False

                def __init__(self):
                    self.commands = []

                def run(self, command, cwd=None, check=True):
                    self.commands.append(command)
                    if command == [
                        'git',
                        'ls-remote',
                        '--exit-code',
                        '--heads',
                        'https://github.com/kubeflow/pipelines.git',
                        'release-1.3',
                    ]:
                        return subprocess.CompletedProcess(command, 0, '', '')
                    if command[:4] == [
                        'git',
                        'ls-remote',
                        '--exit-code',
                        '--heads',
                    ]:
                        return subprocess.CompletedProcess(command, 1, '', '')
                    if command[:4] == [
                        'git',
                        'ls-remote',
                        '--exit-code',
                        '--tags',
                    ]:
                        return subprocess.CompletedProcess(command, 1, '', '')
                    return subprocess.CompletedProcess(command, 0, '', '')

                def capture(self, command, cwd=None):
                    if command[:3] == ['gh', 'pr', 'list']:
                        return '[]'
                    self.commands.append(command)
                    if 'view' in command and any(
                        'statusCheckRollup' in part for part in command
                    ):
                        return '[{"name":"ci","status":"COMPLETED","conclusion":"SUCCESS"}]'
                    if 'view' in command and any(
                        'mergeStateStatus' in part for part in command
                    ):
                        return 'MERGED CLEAN'
                    return 'https://github.com/kubeflow/pipelines/pull/1'

            runner = ExistingUpstreamReleaseBranchRunner()
            context = core.ReleaseContext(
                root=Path(tmpdir),
                state=core.ReleaseState(Path(tmpdir) / 'state.json'),
                runner=runner,
                metadata=core.ReleaseMetadata.from_version('minor', '1.3.0'),
                fork_remote='git@github.com:testuser/pipelines.git',
                include_backend=True,
                include_sdk=True,
            )

            steps.step_merge_version_pr(context)

            self.assertNotIn(
                [
                    'git',
                    'push',
                    'https://github.com/kubeflow/pipelines.git',
                    'release-1.3',
                ],
                runner.commands,
            )

    def test_step_merge_version_pr_uses_existing_remote_branch_when_requested(
        self,
    ):
        with TemporaryDirectory() as tmpdir:

            class ExistingRemoteBranchRunner:
                dry_run = False

                def __init__(self):
                    self.commands = []

                def run(self, command, cwd=None, check=True):
                    self.commands.append(command)
                    if command[:4] == [
                        'git',
                        'ls-remote',
                        '--exit-code',
                        '--heads',
                    ]:
                        return subprocess.CompletedProcess(command, 0, '', '')
                    if command[:4] == [
                        'git',
                        'ls-remote',
                        '--exit-code',
                        '--tags',
                    ]:
                        return subprocess.CompletedProcess(command, 1, '', '')
                    return subprocess.CompletedProcess(command, 0, '', '')

                def capture(self, command, cwd=None):
                    if command[:3] == ['gh', 'pr', 'list']:
                        return '[]'
                    self.commands.append(command)
                    if 'view' in command and any(
                        'statusCheckRollup' in part for part in command
                    ):
                        return '[{"name":"ci","status":"COMPLETED","conclusion":"SUCCESS"}]'
                    if 'view' in command and any(
                        'mergeStateStatus' in part for part in command
                    ):
                        return 'MERGED CLEAN'
                    return 'https://github.com/kubeflow/pipelines/pull/1'

            runner = ExistingRemoteBranchRunner()
            context = core.ReleaseContext(
                root=Path(tmpdir),
                state=core.ReleaseState(Path(tmpdir) / 'state.json'),
                runner=runner,
                metadata=core.ReleaseMetadata.from_version('minor', '1.3.0'),
                fork_remote='git@github.com:testuser/pipelines.git',
                include_backend=True,
                include_sdk=True,
            )

            with mock.patch('builtins.input', return_value='u'):
                steps.step_merge_version_pr(context)

            self.assertIn(
                [
                    'git',
                    'push',
                    'git@github.com:testuser/pipelines.git',
                    '1.3.0-update-version-tags',
                ],
                runner.commands,
            )
            self.assertNotIn(
                [
                    'git',
                    'push',
                    '--set-upstream',
                    'git@github.com:testuser/pipelines.git',
                    '1.3.0-update-version-tags',
                ],
                runner.commands,
            )
            self.assertNotIn(
                [
                    'git',
                    'push',
                    '--delete',
                    'git@github.com:testuser/pipelines.git',
                    '1.3.0-update-version-tags',
                ],
                runner.commands,
            )

    def test_step_merge_version_pr_recreates_existing_remote_branch_when_requested(
        self,
    ):
        with TemporaryDirectory() as tmpdir:

            class ExistingRemoteBranchRunner:
                dry_run = False

                def __init__(self):
                    self.commands = []

                def run(self, command, cwd=None, check=True):
                    self.commands.append(command)
                    if command[:4] == [
                        'git',
                        'ls-remote',
                        '--exit-code',
                        '--heads',
                    ]:
                        return subprocess.CompletedProcess(command, 0, '', '')
                    if command[:4] == [
                        'git',
                        'ls-remote',
                        '--exit-code',
                        '--tags',
                    ]:
                        return subprocess.CompletedProcess(command, 1, '', '')
                    return subprocess.CompletedProcess(command, 0, '', '')

                def capture(self, command, cwd=None):
                    if command[:3] == ['gh', 'pr', 'list']:
                        return '[]'
                    self.commands.append(command)
                    if 'view' in command and any(
                        'statusCheckRollup' in part for part in command
                    ):
                        return '[{"name":"ci","status":"COMPLETED","conclusion":"SUCCESS"}]'
                    if 'view' in command and any(
                        'mergeStateStatus' in part for part in command
                    ):
                        return 'MERGED CLEAN'
                    return 'https://github.com/kubeflow/pipelines/pull/1'

            runner = ExistingRemoteBranchRunner()
            context = core.ReleaseContext(
                root=Path(tmpdir),
                state=core.ReleaseState(Path(tmpdir) / 'state.json'),
                runner=runner,
                metadata=core.ReleaseMetadata.from_version('minor', '1.3.0'),
                fork_remote='git@github.com:testuser/pipelines.git',
                include_backend=True,
                include_sdk=True,
            )

            with mock.patch('builtins.input', return_value='r'):
                steps.step_merge_version_pr(context)

            self.assertIn(
                [
                    'git',
                    'push',
                    '--delete',
                    'git@github.com:testuser/pipelines.git',
                    '1.3.0-update-version-tags',
                ],
                runner.commands,
            )
            self.assertIn(
                [
                    'git',
                    'push',
                    '--set-upstream',
                    'git@github.com:testuser/pipelines.git',
                    '1.3.0-update-version-tags',
                ],
                runner.commands,
            )


class StepSdkVersionFilesTest(unittest.TestCase):
    def setUp(self):
        for name, value in (
            ('_preparation_range', SELECTED_RANGE),
            ('_update_changelog', None),
            ('read_commits', []),
        ):
            patcher = mock.patch.object(steps, name, return_value=value)
            patcher.start()
            self.addCleanup(patcher.stop)

    def test_backend_notes_include_release_fixes_but_not_version_bumps(self):
        with TemporaryDirectory() as tmpdir:
            root = Path(tmpdir)
            subjects = (
                'fix(ci): pin pipeline SDK installs to checked-out commit (#15041)\n'
                'fix(frontend): prune development dependencies from release image (#15040)\n'
                'fix(release): patch 2.17.3 proxy and cache-deployer images (#15036)\n'
                'chore(release): bump version to 2.17.3 (#15034)\n'
                'fix(release): prepare 2.17.3 security backports (#15026)\n'
                'fix(sdk): SDK-only correction (#99999)\n'
            )
            with mock.patch.object(
                steps,
                'read_commits',
                return_value=[
                    Commit('c' * 40, subject)
                    for subject in subjects.splitlines()
                ],
            ):
                notes = steps._generate_backend_release_notes(
                    root, SELECTED_RANGE
                )
            for number in (15041, 15040, 15036, 15026):
                self.assertIn(f'/pull/{number}', notes)
            self.assertNotIn('15034', notes)
            self.assertNotIn('99999', notes)

    def test_sdk_and_backend_use_the_shared_classification(self):
        commits = [
            Commit('a' * 40, 'fix(backend/sdk): fix shared API (#1)'),
            Commit('b' * 40, 'chore(sdk): update test tools (#2)'),
            Commit('c' * 40, 'feat(frontend): show logs (#3)'),
            Commit(
                'd' * 40,
                'chore(sdk): new interface\n\nBREAKING CHANGE: migrate now',
            ),
        ]
        with mock.patch.object(steps, 'read_commits', return_value=commits):
            sdk = steps._generate_sdk_release_notes(
                Path('/repo'), SELECTED_RANGE
            )
            backend = steps._generate_backend_release_notes(
                Path('/repo'), SELECTED_RANGE
            )
        self.assertIn('Breaking changes', sdk)
        self.assertNotIn('update test tools', sdk + backend)
        self.assertIn('fix shared API', sdk)
        self.assertIn('fix shared API', backend)
        self.assertNotIn('show logs', sdk)
        self.assertIn('show logs', backend)

    def test_update_sdk_version_files_replaces_version_without_trailing_garbage(
        self,
    ):
        with TemporaryDirectory() as tmpdir:
            root = Path(tmpdir)
            version_file = root / 'sdk/python/kfp/version.py'
            version_file.parent.mkdir(parents=True)
            version_file.write_text("__version__ = '1.2.3'\n")

            class RecordingRunner:
                dry_run = False

                def __init__(self):
                    self.commands = []

                def run(self, command, cwd=None, check=True):
                    self.commands.append((command, cwd, check))

            state = core.ReleaseState(root / 'state.json')
            metadata = core.ReleaseMetadata.from_version('patch', '1.2.4')
            context = core.ReleaseContext(
                root=root,
                state=state,
                runner=RecordingRunner(),
                metadata=metadata,
                fork_remote='git@github.com:testuser/pipelines.git',
                include_backend=True,
                include_sdk=True,
            )

            with (
                mock.patch('kfpr.steps.clean_legacy_distributions'),
                mock.patch('kfpr.steps.validate_legacy_wheels'),
            ):
                steps._update_sdk_version_files(context)

            self.assertEqual(
                version_file.read_text(), "__version__ = '1.2.4'\n"
            )

    def test_step_update_version_tags_updates_sdk_files_when_sdk_included(self):
        with TemporaryDirectory() as tmpdir:
            root = Path(tmpdir)
            files = {
                'sdk/python/kfp/version.py': "__version__ = '1.2.3'\n",
                'kubernetes_platform/python/kfp/kubernetes/__init__.py': "__version__ = '1.2.3'\n",
                'api/v2alpha1/python/setup.py': "NAME = 'kfp-pipeline-spec'\nVERSION = '1.2.3'\n",
                'backend/api/v2beta1/python_http_client/setup.py': 'NAME = "kfp-server-api"\nVERSION = "1.2.3"\n',
                'backend/api/v2beta1/python_http_client/kfp_server_api/__init__.py': '__version__ = "1.2.3"\n',
                'sdk/python/requirements.in': (
                    'kfp-pipeline-spec>=1.2.3,<3\nkfp-server-api>=1.2.3,<3\n'
                ),
                'kubernetes_platform/python/requirements.in': 'kfp>=1.2.3,<3\n',
                'docs/sdk/versions.json': (
                    '[\n'
                    '    {\n'
                    '      "version": "https://kubeflow-pipelines.readthedocs.io/en/sdk-1.2.3/",\n'
                    '      "title": "1.2.3",\n'
                    '      "aliases": [\n'
                    '        "stable",\n'
                    '        "latest"\n'
                    '      ]\n'
                    '    }\n'
                    ']\n'
                ),
                'kubernetes_platform/python/docs/conf.py': (
                    'html_theme_options = {\n'
                    "    'version_info': [\n"
                    '        {\n'
                    "            'version':\n"
                    "                'https://kfp-kubernetes.readthedocs.io/en/kfp-kubernetes-1.2/',\n"
                    "            'title':\n"
                    "                '1.2',\n"
                    "            'aliases': ['stable'],\n"
                    '        },\n'
                    '    ],\n'
                    '}\n'
                ),
                'sdk/RELEASE.md': (
                    '# Current Version (in development)\n\n'
                    '## Features\n\n'
                    '* New SDK feature.\n\n'
                    '## Breaking changes\n\n'
                    '## Deprecations\n\n'
                    '## Bug fixes and other changes\n\n'
                    '# 1.2.3\n'
                ),
            }
            for relative_path, content in files.items():
                path = root / relative_path
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_text(content)

            class RecordingRunner:
                dry_run = False

                def __init__(self):
                    self.commands = []

                def run(self, command, cwd=None, check=True):
                    self.commands.append((command, cwd, check))
                    if command[:4] == [
                        'git',
                        'rev-parse',
                        '--verify',
                        'refs/heads/1.3.0-update-version-tags',
                    ]:
                        return subprocess.CompletedProcess(command, 1, '', '')
                    if command[:4] == [
                        'git',
                        'rev-parse',
                        '--verify',
                        'refs/tags/1.3.0',
                    ]:
                        return subprocess.CompletedProcess(command, 1, '', '')
                    return subprocess.CompletedProcess(command, 0, '', '')

            context = core.ReleaseContext(
                root=root,
                state=core.ReleaseState(root / 'state.json'),
                runner=RecordingRunner(),
                metadata=core.ReleaseMetadata.from_version('minor', '1.3.0'),
                fork_remote='git@github.com:testuser/pipelines.git',
                include_backend=True,
                include_sdk=True,
            )

            events = []
            original_run = context.runner.run

            def record_run(command, cwd=None, check=True):
                events.append(('command', command))
                return original_run(command, cwd=cwd, check=check)

            context.runner.run = record_run
            with (
                mock.patch(
                    'kfpr.steps.clean_legacy_distributions',
                    side_effect=lambda path: events.append(('clean', path)),
                ) as clean,
                mock.patch(
                    'kfpr.steps.validate_legacy_wheels',
                    side_effect=lambda path, version: events.append(
                        ('validate', path, version)
                    ),
                ) as validate,
            ):
                steps.step_update_version_tags(context)
            clean.assert_called_once_with(root)
            validate.assert_called_once_with(root, '1.3.0')
            self.assertLess(
                events.index(('clean', root)),
                events.index(('command', ['make', 'python'])),
            )
            self.assertLess(
                max(
                    index
                    for index, event in enumerate(events)
                    if event
                    == ('command', [sys.executable, '-m', 'build', '.'])
                ),
                events.index(('validate', root, '1.3.0')),
            )
            self.assertLess(
                events.index(('validate', root, '1.3.0')),
                events.index(('command', ['git', 'add', '--all'])),
            )

            self.assertEqual(
                (root / 'sdk/python/kfp/version.py').read_text(),
                "__version__ = '1.3.0'\n",
            )
            self.assertEqual(
                (
                    root
                    / 'kubernetes_platform/python/kfp/kubernetes/__init__.py'
                ).read_text(),
                "__version__ = '1.3.0'\n",
            )
            self.assertIn(
                "VERSION = '1.3.0'",
                (root / 'api/v2alpha1/python/setup.py').read_text(),
            )
            self.assertIn(
                'VERSION = "1.3.0"',
                (
                    root / 'backend/api/v2beta1/python_http_client/setup.py'
                ).read_text(),
            )
            self.assertIn(
                '__version__ = "1.3.0"',
                (
                    root
                    / 'backend/api/v2beta1/python_http_client/kfp_server_api/__init__.py'
                ).read_text(),
            )
            self.assertIn(
                'kfp-pipeline-spec>=1.3.0,<2',
                (root / 'sdk/python/requirements.in').read_text(),
            )
            self.assertIn(
                'kfp-server-api>=1.3.0,<2',
                (root / 'sdk/python/requirements.in').read_text(),
            )
            self.assertEqual(
                (
                    root / 'kubernetes_platform/python/requirements.in'
                ).read_text(),
                'kfp>=1.3.0,<2\n',
            )
            self.assertIn(
                '"title": "1.3.0"',
                (root / 'docs/sdk/versions.json').read_text(),
            )
            self.assertIn(
                '"aliases": []', (root / 'docs/sdk/versions.json').read_text()
            )
            self.assertIn(
                "'title':\n                '1.3'",
                (root / 'kubernetes_platform/python/docs/conf.py').read_text(),
            )
            self.assertIn(
                "'aliases': []",
                (root / 'kubernetes_platform/python/docs/conf.py').read_text(),
            )
            self.assertIn('# 1.3.0', (root / 'sdk/RELEASE.md').read_text())
            self.assertIn(
                '* New SDK feature.', (root / 'sdk/RELEASE.md').read_text()
            )
            commands = [command for command, _, _ in context.runner.commands]
            self.assertLess(
                commands.index([sys.executable, '-m', 'build', '.']),
                next(
                    index
                    for index, command in enumerate(commands)
                    if command[:2] == ['bash', '-c']
                ),
            )
            self.assertIn(['git', 'add', '--all'], commands)
            invocations = [
                (command, cwd) for command, cwd, _ in context.runner.commands
            ]
            self.assertLess(
                invocations.index((['make', 'python'], root / 'api')),
                invocations.index(
                    (
                        [sys.executable, '-m', 'build', '.'],
                        root / 'api/v2alpha1/python',
                    )
                ),
            )
            self.assertLess(
                invocations.index(
                    (
                        ['make', 'python', 'USE_FIND_LINKS=true'],
                        root / 'kubernetes_platform',
                    )
                ),
                invocations.index(
                    (
                        [sys.executable, '-m', 'build', '.'],
                        root / 'kubernetes_platform/python',
                    )
                ),
            )


class UvReleasePackagesTest(unittest.TestCase):
    def test_unified_release_generates_and_builds_only_kfp(self) -> None:
        """Keep codegen ahead of packaging without reviving retired
        projects."""
        with TemporaryDirectory() as tmpdir:
            root = Path(tmpdir)
            files = {
                'uv.lock': 'version = 1\n',
                'VERSION': '2.18.0\n',
                'sdk/python/kfp/version.py': "__version__ = '2.17.0'\n",
                'sdk/python/kfp/server_api/__init__.py': 'from kfp.version import __version__\n',
                'sdk/python/kfp/kubernetes/__init__.py': 'from kfp.version import __version__\n',
                '.github/resources/scripts/export_python_requirements.sh': '#!/bin/bash\n',
            }
            for relative, content in files.items():
                path = root / relative
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_text(content)
            runner = mock.Mock(spec=core.CommandRunner)
            runner.dry_run = False
            context = core.ReleaseContext(
                root=root,
                state=core.ReleaseState(root / 'state.json'),
                runner=runner,
                metadata=core.ReleaseMetadata.from_version('major', '3.0.0'),
                fork_remote='git@github.com:testuser/pipelines.git',
                include_backend=False,
                include_sdk=True,
            )
            with mock.patch.object(
                steps, '_generate_sdk_release_notes', return_value=''
            ):
                steps._update_sdk_version_files(context)
            self.assertEqual((root / 'VERSION').read_text(), '2.18.0\n')
            self.assertIn(
                '3.0.0', (root / 'sdk/python/kfp/version.py').read_text()
            )
            self.assertEqual(
                runner.run.call_args_list[:2],
                [
                    mock.call(
                        ['make', '-C', 'sdk', 'generate-python'], cwd=root
                    ),
                    mock.call(['uv', 'lock'], cwd=root),
                ],
            )
            self.assertFalse(
                any(
                    call.args[0][0] == 'bash'
                    for call in runner.run.call_args_list
                )
            )
            self.assertFalse(
                any(
                    call.args[0][:2] == ['uv', 'export']
                    for call in runner.run.call_args_list
                )
            )
            builds = [
                call.args[0]
                for call in runner.run.call_args_list
                if call.args[0][:2] == ['uv', 'build']
            ]
            self.assertEqual(
                builds,
                [
                    [
                        'uv',
                        'build',
                        '--package',
                        'kfp',
                        '--out-dir',
                        'sdk/python/dist',
                    ]
                ],
            )
            self.assertFalse((root / 'kubernetes_platform/python').exists())

    def test_older_unified_checkout_retains_two_exports(self):
        """Legacy exports do not depend on whether an old helper is present."""
        for legacy_exporter_present in (False, True):
            with (
                self.subTest(legacy_exporter_present=legacy_exporter_present),
                TemporaryDirectory() as tmpdir,
            ):
                root = Path(tmpdir)
                bundled_api = root / 'sdk/python/kfp/server_api/__init__.py'
                bundled_api.parent.mkdir(parents=True)
                bundled_api.touch()
                (root / 'requirements.txt').write_text('legacy export\n')
                if legacy_exporter_present:
                    exporter = (
                        root
                        / '.github/resources/scripts/export_python_requirements.sh'
                    )
                    exporter.parent.mkdir(parents=True)
                    exporter.write_text('#!/bin/bash\n')
                runner = mock.Mock(spec=core.CommandRunner)
                context = mock.Mock(root=root, runner=runner)

                steps._refresh_uv_release_packages(context)

                commands = [call.args[0] for call in runner.run.call_args_list]
                prefix = ['uv', 'export', '--frozen', '--no-dev', '--no-hashes']
                self.assertEqual(
                    commands,
                    [
                        ['make', '-C', 'sdk', 'generate-python'],
                        ['uv', 'lock'],
                        prefix
                        + [
                            '--format',
                            'requirements-txt',
                            '-o',
                            'requirements.txt',
                        ],
                        prefix
                        + [
                            '--package',
                            'kfp',
                            '--format',
                            'requirements-txt',
                            '-o',
                            'sdk/python/requirements.txt',
                        ],
                        [
                            'uv',
                            'build',
                            '--package',
                            'kfp',
                            '--out-dir',
                            'sdk/python/dist',
                        ],
                    ],
                )
                self.assertEqual(
                    (root / '.github').exists(), legacy_exporter_present
                )

    def test_modern_checkout_without_exports_or_helper_builds_only_kfp(self):
        with TemporaryDirectory() as tmpdir:
            root = Path(tmpdir)
            bundled_api = root / 'sdk/python/kfp/server_api/__init__.py'
            bundled_api.parent.mkdir(parents=True)
            bundled_api.touch()
            runner = mock.Mock(spec=core.CommandRunner)
            steps._refresh_uv_release_packages(
                mock.Mock(root=root, runner=runner)
            )
            commands = [call.args[0] for call in runner.run.call_args_list]
            self.assertEqual(
                [command[:2] for command in commands],
                [['make', '-C'], ['uv', 'lock'], ['uv', 'build']],
            )

    def test_update_sdk_versions_regenerates_server_api_and_refreshes_workspace(
        self,
    ) -> None:
        """Release all Python packages together without bumping the backend."""
        for dry_run in (False, True):
            with self.subTest(dry_run=dry_run), TemporaryDirectory() as tmpdir:
                root = Path(tmpdir)
                files = {
                    'uv.lock': 'version = 1\n',
                    'VERSION': '2.18.0\n',
                    'sdk/python/kfp/version.py': "__version__ = '2.17.0'\n",
                    'kubernetes_platform/python/kfp/kubernetes/__init__.py': "__version__ = '2.17.0'\n",
                    'api/v2alpha1/python/pyproject.toml': '[project]\nversion = "2.17.0"\n',
                    'sdk/python/pyproject.toml': (
                        '[project]\ndependencies = ["kfp-pipeline-spec>=2.17.0,<3", "kfp-server-api>=2.17.0,<3"]\n'
                        '[project.optional-dependencies]\nkubernetes = ["kfp-kubernetes==2.17.0"]\n'
                    ),
                    'kubernetes_platform/python/pyproject.toml': '[project]\ndependencies = ["kfp>=2.17.0,<3"]\n',
                }
                for relative_path, content in files.items():
                    path = root / relative_path
                    path.parent.mkdir(parents=True, exist_ok=True)
                    path.write_text(content)

                runner = mock.Mock(spec=core.CommandRunner)
                runner.dry_run = dry_run
                context = core.ReleaseContext(
                    root=root,
                    state=core.ReleaseState(root / 'state.json'),
                    runner=runner,
                    metadata=core.ReleaseMetadata.from_version(
                        'patch', '2.17.1'
                    ),
                    fork_remote='git@github.com:testuser/pipelines.git',
                    include_backend=False,
                    include_sdk=True,
                )
                with mock.patch.object(
                    steps, '_generate_sdk_release_notes', return_value=''
                ):
                    steps._update_sdk_version_files(context)

                for relative_path, original in files.items():
                    expected = (
                        original
                        if dry_run
                        else original.replace('2.17.0', '2.17.1')
                    )
                    self.assertEqual(
                        (root / relative_path).read_text(),
                        expected,
                        relative_path,
                    )
                self.assertEqual(
                    runner.run.call_args_list[:2],
                    [
                        mock.call(
                            [
                                'make',
                                'API_VERSION=v2beta1',
                                'generate-kfp-server-api-package',
                            ],
                            cwd=root / 'backend/api',
                        ),
                        mock.call(['uv', 'lock'], cwd=root),
                    ],
                )
                runner.run.assert_any_call(
                    [
                        'uv',
                        'export',
                        '--frozen',
                        '--no-dev',
                        '--no-hashes',
                        '--format',
                        'requirements-txt',
                        '-o',
                        'requirements.txt',
                    ],
                    cwd=root,
                )
                for package, path in (
                    ('kfp-pipeline-spec', 'api/v2alpha1/python'),
                    (
                        'kfp-server-api',
                        'backend/api/v2beta1/python_http_client',
                    ),
                    ('kfp', 'sdk/python'),
                    ('kfp-kubernetes', 'kubernetes_platform/python'),
                ):
                    runner.run.assert_any_call(
                        [
                            'uv',
                            'build',
                            '--package',
                            package,
                            '--out-dir',
                            f'{path}/dist',
                        ],
                        cwd=root,
                    )
                    if package != 'kfp-server-api':
                        runner.run.assert_any_call(
                            [
                                'uv',
                                'export',
                                '--frozen',
                                '--no-dev',
                                '--no-hashes',
                                '--package',
                                package,
                                '--format',
                                'requirements-txt',
                                '-o',
                                f'{path}/requirements.txt',
                            ],
                            cwd=root,
                        )


class DryRunOutputTest(unittest.TestCase):
    def setUp(self) -> None:
        """Retain legacy branch-operation coverage without GitHub requests."""
        layout = mock.patch.object(
            steps, '_sdk_release_is_consolidated', return_value=False
        )
        layout.start()
        self.addCleanup(layout.stop)
        for name in ('_preparation_range', '_update_changelog'):
            patcher = mock.patch.object(
                steps, name, return_value=SELECTED_RANGE
            )
            patcher.start()
            self.addCleanup(patcher.stop)

    def test_update_version_tags_cuts_update_branch_from_release_branch(self):

        class MissingBranchRunner(core.CommandRunner):
            def run(self, command, cwd=None, check=True):
                self.commands.append(command)
                if command[:4] == [
                    'git',
                    'rev-parse',
                    '--verify',
                    'refs/heads/3.2.0-update-version-tags',
                ]:
                    return subprocess.CompletedProcess(command, 1, '', '')
                if command[:4] == [
                    'git',
                    'rev-parse',
                    '--verify',
                    'refs/tags/3.2.0',
                ]:
                    return subprocess.CompletedProcess(command, 1, '', '')
                return subprocess.CompletedProcess(command, 0, '', '')

        with TemporaryDirectory() as tmpdir:
            root = Path(tmpdir)
            context = core.ReleaseContext(
                root=root,
                state=core.ReleaseState(root / 'state.json'),
                runner=MissingBranchRunner(dry_run=False),
                metadata=core.ReleaseMetadata.from_version('minor', '3.2.0'),
                fork_remote='git@github.com:testuser/pipelines.git',
                include_backend=True,
                include_sdk=True,
            )

            # This runner records builds without producing wheel artifacts.
            with mock.patch('kfpr.steps.validate_legacy_wheels'):
                steps.step_update_version_tags(context)

            self.assertEqual((root / 'VERSION').read_bytes(), b'3.2.0\n')

            self.assertEqual(
                context.runner.commands[0], ['git', 'checkout', 'release-3.2']
            )
            self.assertIn(['git', 'pull', '--ff-only'], context.runner.commands)
            self.assertIn(
                ['git', 'checkout', '-b', '3.2.0-update-version-tags'],
                context.runner.commands,
            )

    def test_update_version_tags_skips_pull_without_release_branch_upstream(
        self,
    ):

        class NoUpstreamRunner(core.CommandRunner):
            def run(self, command, cwd=None, check=True):
                self.commands.append(command)
                if command == [
                    'git',
                    'rev-parse',
                    '--abbrev-ref',
                    '--symbolic-full-name',
                    '@{u}',
                ]:
                    return subprocess.CompletedProcess(command, 1, '', '')
                if command[:4] == [
                    'git',
                    'rev-parse',
                    '--verify',
                    'refs/heads/3.2.0-update-version-tags',
                ]:
                    return subprocess.CompletedProcess(command, 1, '', '')
                if command[:4] == [
                    'git',
                    'rev-parse',
                    '--verify',
                    'refs/tags/3.2.0',
                ]:
                    return subprocess.CompletedProcess(command, 1, '', '')
                return subprocess.CompletedProcess(command, 0, '', '')

        with TemporaryDirectory() as tmpdir:
            root = Path(tmpdir)
            context = core.ReleaseContext(
                root=root,
                state=core.ReleaseState(root / 'state.json'),
                runner=NoUpstreamRunner(dry_run=False),
                metadata=core.ReleaseMetadata.from_version('minor', '3.2.0'),
                fork_remote='git@github.com:testuser/pipelines.git',
                include_backend=True,
                include_sdk=True,
            )

            # This runner records builds without producing wheel artifacts.
            with mock.patch('kfpr.steps.validate_legacy_wheels'):
                steps.step_update_version_tags(context)

            self.assertIn(
                ['git', 'checkout', 'release-3.2'], context.runner.commands
            )
            self.assertNotIn(
                ['git', 'pull', '--ff-only'], context.runner.commands
            )
            self.assertIn(
                ['git', 'checkout', '-b', '3.2.0-update-version-tags'],
                context.runner.commands,
            )

    def test_update_version_tags_recreates_existing_update_branch_when_requested(
        self,
    ):

        class ExistingBranchRunner(core.CommandRunner):
            def run(self, command, cwd=None, check=True):
                self.commands.append(command)
                if command[:4] == [
                    'git',
                    'rev-parse',
                    '--verify',
                    'refs/heads/3.2.0-update-version-tags',
                ]:
                    return subprocess.CompletedProcess(command, 0, '', '')
                if command[:4] == [
                    'git',
                    'rev-parse',
                    '--verify',
                    'refs/tags/3.2.0',
                ]:
                    return subprocess.CompletedProcess(command, 1, '', '')
                return subprocess.CompletedProcess(command, 0, '', '')

        with TemporaryDirectory() as tmpdir:
            root = Path(tmpdir)
            context = core.ReleaseContext(
                root=root,
                state=core.ReleaseState(root / 'state.json'),
                runner=ExistingBranchRunner(dry_run=False),
                metadata=core.ReleaseMetadata.from_version('minor', '3.2.0'),
                fork_remote='git@github.com:testuser/pipelines.git',
                include_backend=True,
                include_sdk=True,
            )

            with mock.patch('builtins.input', return_value='r'):
                # This runner records builds without producing wheel artifacts.
                with mock.patch('kfpr.steps.validate_legacy_wheels'):
                    steps.step_update_version_tags(context)

            self.assertIn(
                ['git', 'branch', '-D', '3.2.0-update-version-tags'],
                context.runner.commands,
            )
            self.assertIn(
                ['git', 'checkout', '-b', '3.2.0-update-version-tags'],
                context.runner.commands,
            )

    def test_update_version_tags_uses_existing_update_branch_when_requested(
        self,
    ):

        class ExistingBranchRunner(core.CommandRunner):
            def run(self, command, cwd=None, check=True):
                self.commands.append(command)
                if command[:4] == [
                    'git',
                    'rev-parse',
                    '--verify',
                    'refs/heads/3.2.0-update-version-tags',
                ]:
                    return subprocess.CompletedProcess(command, 0, '', '')
                if command[:4] == [
                    'git',
                    'rev-parse',
                    '--verify',
                    'refs/tags/3.2.0',
                ]:
                    return subprocess.CompletedProcess(command, 1, '', '')
                return subprocess.CompletedProcess(command, 0, '', '')

        with TemporaryDirectory() as tmpdir:
            root = Path(tmpdir)
            context = core.ReleaseContext(
                root=root,
                state=core.ReleaseState(root / 'state.json'),
                runner=ExistingBranchRunner(dry_run=False),
                metadata=core.ReleaseMetadata.from_version('minor', '3.2.0'),
                fork_remote='git@github.com:testuser/pipelines.git',
                include_backend=True,
                include_sdk=True,
            )

            with mock.patch('builtins.input', return_value='u'):
                # This runner records builds without producing wheel artifacts.
                with mock.patch('kfpr.steps.validate_legacy_wheels'):
                    steps.step_update_version_tags(context)

            self.assertIn(
                ['git', 'checkout', '3.2.0-update-version-tags'],
                context.runner.commands,
            )
            self.assertNotIn(
                ['git', 'branch', '-D', '3.2.0-update-version-tags'],
                context.runner.commands,
            )
            self.assertNotIn(
                ['git', 'checkout', '-b', '3.2.0-update-version-tags'],
                context.runner.commands,
            )

    def test_update_version_tags_dry_run_does_not_tag(self):
        with TemporaryDirectory() as tmpdir:
            root = Path(tmpdir)
            context = core.ReleaseContext(
                root=root,
                state=core.ReleaseState(root / 'state.json'),
                runner=core.CommandRunner(dry_run=True),
                metadata=core.ReleaseMetadata.from_version('minor', '3.2.0'),
                fork_remote='git@github.com:testuser/pipelines.git',
                include_backend=True,
                include_sdk=True,
            )

            with mock.patch('builtins.print') as print_mock:
                steps.step_update_version_tags(context)

            output = '\n'.join(
                str(call.args[0]) for call in print_mock.call_args_list
            )
            self.assertIn('[dry-run] would write VERSION: 3.2.0', output)
            self.assertNotIn('would create tag', output)

    def test_update_version_tags_no_longer_edits_release_makefile(self):
        with TemporaryDirectory() as tmpdir:
            root = Path(tmpdir)
            context = core.ReleaseContext(
                root=root,
                state=core.ReleaseState(root / 'state.json'),
                runner=core.CommandRunner(dry_run=True),
                metadata=core.ReleaseMetadata.from_version('minor', '3.2.0'),
                fork_remote='git@github.com:testuser/pipelines.git',
                include_backend=True,
                include_sdk=True,
            )

            steps.step_update_version_tags(context)

            flattened_commands = [
                ' '.join(command) for command in context.runner.commands
            ]
            self.assertFalse(
                any(
                    'test/release/Makefile' in command
                    for command in flattened_commands
                ),
                flattened_commands,
            )

    def test_update_version_tags_pins_release_dockerfile_image(self):
        with TemporaryDirectory() as tmpdir:
            root = Path(tmpdir)
            context = core.ReleaseContext(
                root=root,
                state=core.ReleaseState(root / 'state.json'),
                runner=core.CommandRunner(dry_run=True),
                metadata=core.ReleaseMetadata.from_version('minor', '3.2.0'),
                fork_remote='git@github.com:testuser/pipelines.git',
                include_backend=True,
                include_sdk=True,
            )

            steps.step_update_version_tags(context)

            flattened_commands = [
                ' '.join(command) for command in context.runner.commands
            ]
            self.assertTrue(
                any(
                    'release/Dockerfile.release' in command
                    and 'release-3.2' in command
                    for command in flattened_commands
                ),
                flattened_commands,
            )
            self.assertFalse(
                any(
                    'test/release/Dockerfile.release' in command
                    for command in flattened_commands
                ),
                flattened_commands,
            )

    def test_update_version_tags_pins_tool_images_to_release_branch(self):
        with TemporaryDirectory() as tmpdir:
            root = Path(tmpdir)
            context = core.ReleaseContext(
                root=root,
                state=core.ReleaseState(root / 'state.json'),
                runner=core.CommandRunner(dry_run=True),
                metadata=core.ReleaseMetadata.from_version('minor', '3.2.0'),
                fork_remote='git@github.com:testuser/pipelines.git',
                include_backend=True,
                include_sdk=True,
            )

            steps.step_update_version_tags(context)

            flattened_commands = [
                ' '.join(command) for command in context.runner.commands
            ]
            self.assertTrue(
                any(
                    'PREBUILT_REMOTE_IMAGE=ghcr.io/kubeflow/kfp-api-generator:'
                    in command
                    and 'release-3.2' in command
                    for command in flattened_commands
                ),
                flattened_commands,
            )
            self.assertTrue(
                any(
                    'RELEASE_IMAGE=ghcr.io/kubeflow/kfp-release:' in command
                    and 'release-3.2' in command
                    for command in flattened_commands
                ),
                flattened_commands,
            )
            self.assertFalse(
                any(
                    'ARG BASE_IMAGE=ghcr.io/kubeflow/kfp-api-generator:'
                    in command
                    for command in flattened_commands
                ),
                flattened_commands,
            )

    def test_update_version_tags_uses_existing_tag_when_requested(self):

        class ExistingTagRunner(core.CommandRunner):
            def run(self, command, cwd=None, check=True):
                self.commands.append(command)
                if command[:4] == [
                    'git',
                    'rev-parse',
                    '--verify',
                    'refs/heads/3.2.0-update-version-tags',
                ]:
                    return subprocess.CompletedProcess(command, 1, '', '')
                if command[:4] == [
                    'git',
                    'rev-parse',
                    '--verify',
                    'refs/tags/3.2.0',
                ]:
                    return subprocess.CompletedProcess(command, 0, '', '')
                return subprocess.CompletedProcess(command, 0, '', '')

        with TemporaryDirectory() as tmpdir:
            root = Path(tmpdir)
            context = core.ReleaseContext(
                root=root,
                state=core.ReleaseState(root / 'state.json'),
                runner=ExistingTagRunner(dry_run=False),
                metadata=core.ReleaseMetadata.from_version('minor', '3.2.0'),
                fork_remote='git@github.com:testuser/pipelines.git',
                include_backend=True,
                include_sdk=True,
            )

            with mock.patch('builtins.input', return_value='u'):
                # This runner records builds without producing wheel artifacts.
                with mock.patch('kfpr.steps.validate_legacy_wheels'):
                    steps.step_update_version_tags(context)

            self.assertNotIn(
                ['git', 'tag', '-d', '3.2.0'], context.runner.commands
            )
            self.assertNotIn(
                [
                    'git',
                    'tag',
                    '-a',
                    '3.2.0',
                    '-m',
                    'Kubeflow Pipelines 3.2.0 release',
                ],
                context.runner.commands,
            )

    def test_update_version_tags_uses_master_release_image_for_minor_release(
        self,
    ):
        with TemporaryDirectory() as tmpdir:
            root = Path(tmpdir)
            context = core.ReleaseContext(
                root=root,
                state=core.ReleaseState(root / 'state.json'),
                runner=core.CommandRunner(dry_run=True),
                metadata=core.ReleaseMetadata.from_version('minor', '3.2.0'),
                fork_remote='git@github.com:testuser/pipelines.git',
                include_backend=True,
                include_sdk=True,
            )

            steps.step_update_version_tags(context)

            flattened_commands = [
                ' '.join(command) for command in context.runner.commands
            ]
            self.assertTrue(
                any(
                    'ghcr.io/kubeflow/kfp-release:master' in command
                    for command in flattened_commands
                ),
                flattened_commands,
            )

    def test_manual_checkpoint_text_can_be_reused_by_status(self):
        metadata = core.ReleaseMetadata.from_version('minor', '3.2.0')

        self.assertIn(
            'ReadTheDocs manual checkpoint:',
            steps.manual_checklist('confirm-rtd', metadata),
        )
        self.assertIn(
            'https://app.readthedocs.org/projects/kubeflow-pipelines/',
            steps.manual_checklist('confirm-rtd', metadata),
        )
        final_checklist = steps.manual_checklist(
            'confirm-website-and-slack', metadata
        )
        self.assertIn('Manual final checkpoint:', final_checklist)
        self.assertIn(
            'https://github.com/kubeflow/website/edit/master/layouts/shortcodes/pipelines/latest-version.html',
            final_checklist,
        )
        self.assertIn(
            'KFP 3.2.0 has officially been released!', final_checklist
        )
        self.assertIn('Backend - ', final_checklist)
        self.assertIn(
            'https://github.com/kubeflow/pipelines/releases/tag/3.2.0',
            final_checklist,
        )
        self.assertIn('SDK - ', final_checklist)
        self.assertIn(
            'https://github.com/kubeflow/pipelines/releases/tag/sdk-3.2.0',
            final_checklist,
        )
        self.assertIn('Changes: ', final_checklist)
        self.assertIn(
            'https://github.com/kubeflow/pipelines/blob/master/CHANGELOG.md',
            final_checklist,
        )
        self.assertNotIn('does not support Argo Workflows 3.x', final_checklist)

    def test_kfp_3_0_checkpoint_announces_argo_3_support_removal(self):
        metadata = core.ReleaseMetadata.from_version('major', '3.0.0')

        final_checklist = steps.manual_checklist(
            'confirm-website-and-slack', metadata
        )

        self.assertIn('does not support Argo Workflows 3.x', final_checklist)
        self.assertIn('requires a', final_checklist)
        self.assertIn('supported Argo Workflows 4.x release', final_checklist)
        self.assertIn(
            'release notes list removal of Argo Workflows 3.x support as a breaking',
            final_checklist,
        )
        self.assertIn('7. After the website PR merges', final_checklist)

    def test_final_confirmation_prints_release_completion_message(self):
        with TemporaryDirectory() as tmpdir:
            context = core.ReleaseContext(
                root=Path(tmpdir),
                state=core.ReleaseState(Path(tmpdir) / 'state.json'),
                runner=core.CommandRunner(dry_run=False),
                metadata=core.ReleaseMetadata.from_version('minor', '3.2.0'),
                fork_remote='git@github.com:testuser/pipelines.git',
                include_backend=True,
                include_sdk=True,
            )

            with (
                contextlib.redirect_stdout(io.StringIO()) as output,
                mock.patch('builtins.input', return_value='y'),
            ):
                steps.step_confirm_website_and_slack(context)

            self.assertIn(
                "Congratulations! You've completed the release!",
                output.getvalue(),
            )

    def test_confirm_rtd_automates_with_prompted_token(self):
        with TemporaryDirectory() as tmpdir:
            root = Path(tmpdir)
            context = core.ReleaseContext(
                root=root,
                state=core.ReleaseState(root / 'state.json'),
                runner=core.CommandRunner(dry_run=False),
                metadata=core.ReleaseMetadata.from_version('minor', '3.2.0'),
                fork_remote='git@github.com:testuser/pipelines.git',
                include_backend=True,
                include_sdk=True,
            )

            with (
                mock.patch('getpass.getpass', return_value='secret-token'),
                mock.patch.object(steps.rtd, 'ReadTheDocsClient') as client_cls,
                mock.patch.object(
                    steps.rtd, 'update_release_docs'
                ) as update_docs,
                mock.patch('builtins.print') as print_mock,
            ):
                steps.step_confirm_rtd(context)

            client_cls.assert_called_once_with('secret-token')
            update_docs.assert_called_once_with(
                client_cls.return_value,
                '3.2.0',
                'release-3.2',
                consolidated=False,
                set_defaults=True,
            )
            self.assertNotIn('rtd_token', context.state.answers)
            output = '\n'.join(
                str(call.args[0]) for call in print_mock.call_args_list
            )
            self.assertNotIn('ReadTheDocs manual checkpoint:', output)

    def test_confirm_rtd_uses_env_token_without_prompting(self):
        with TemporaryDirectory() as tmpdir:
            root = Path(tmpdir)
            context = core.ReleaseContext(
                root=root,
                state=core.ReleaseState(root / 'state.json'),
                runner=core.CommandRunner(dry_run=False),
                metadata=core.ReleaseMetadata.from_version('minor', '3.2.0'),
                fork_remote='git@github.com:testuser/pipelines.git',
                include_backend=True,
                include_sdk=True,
            )

            with (
                mock.patch.dict('os.environ', {'RTD_TOKEN': 'env-token'}),
                mock.patch('getpass.getpass') as getpass_mock,
                mock.patch.object(steps.rtd, 'ReadTheDocsClient') as client_cls,
                mock.patch.object(
                    steps.rtd, 'update_release_docs'
                ) as update_docs,
            ):
                steps.step_confirm_rtd(context)

            getpass_mock.assert_not_called()
            client_cls.assert_called_once_with('env-token')
            update_docs.assert_called_once_with(
                client_cls.return_value,
                '3.2.0',
                'release-3.2',
                consolidated=False,
                set_defaults=True,
            )

    def test_confirm_rtd_reasks_for_empty_token(self):
        with TemporaryDirectory() as tmpdir:
            root = Path(tmpdir)
            context = core.ReleaseContext(
                root=root,
                state=core.ReleaseState(root / 'state.json'),
                runner=core.CommandRunner(dry_run=False),
                metadata=core.ReleaseMetadata.from_version('minor', '3.2.0'),
                fork_remote='git@github.com:testuser/pipelines.git',
                include_backend=True,
                include_sdk=True,
            )

            with (
                mock.patch('getpass.getpass', side_effect=['', 'secret-token']),
                mock.patch.object(steps.rtd, 'ReadTheDocsClient') as client_cls,
                mock.patch.object(steps.rtd, 'update_release_docs'),
            ):
                steps.step_confirm_rtd(context)

            client_cls.assert_called_once_with('secret-token')

    def test_confirm_rtd_falls_back_to_manual_when_automation_fails_and_user_accepts(
        self,
    ):
        with TemporaryDirectory() as tmpdir:
            root = Path(tmpdir)
            context = core.ReleaseContext(
                root=root,
                state=core.ReleaseState(root / 'state.json'),
                runner=core.CommandRunner(dry_run=False),
                metadata=core.ReleaseMetadata.from_version('minor', '3.2.0'),
                fork_remote='git@github.com:testuser/pipelines.git',
                include_backend=True,
                include_sdk=True,
            )

            with (
                mock.patch('getpass.getpass', return_value='secret-token'),
                mock.patch.object(steps.rtd, 'ReadTheDocsClient'),
                mock.patch.object(
                    steps.rtd,
                    'update_release_docs',
                    side_effect=steps.rtd.ReadTheDocsError('boom'),
                ),
                mock.patch('builtins.input', side_effect=['y', 'y']),
            ):
                steps.step_confirm_rtd(context)

    def test_kfp_kubernetes_docs_branch_reuses_existing_remote_branch(self):

        class ExistingBranchRunner(core.CommandRunner):
            def run(self, command, cwd=None, check=True):
                self.commands.append(command)
                if command[:4] == [
                    'git',
                    'ls-remote',
                    '--exit-code',
                    '--heads',
                ]:
                    return subprocess.CompletedProcess(command, 0, '', '')
                return subprocess.CompletedProcess(command, 0, '', '')

        with TemporaryDirectory() as tmpdir:
            root = Path(tmpdir)
            context = core.ReleaseContext(
                root=root,
                state=core.ReleaseState(root / 'state.json'),
                runner=ExistingBranchRunner(dry_run=False),
                metadata=core.ReleaseMetadata.from_version('minor', '3.2.0'),
                fork_remote='git@github.com:testuser/pipelines.git',
                include_backend=True,
                include_sdk=True,
            )

            with (
                mock.patch('builtins.input', return_value='reuse'),
                mock.patch.object(context.runner, 'capture', return_value=''),
            ):
                steps.step_create_kfp_kubernetes_docs_branch(context)

            self.assertEqual(
                context.runner.commands,
                [
                    [
                        'git',
                        'ls-remote',
                        '--exit-code',
                        '--heads',
                        'upstream',
                        'kfp-kubernetes-3.2',
                    ],
                ],
            )

    def test_kfp_kubernetes_docs_branch_can_replace_existing_remote_branch(
        self,
    ):

        class ExistingBranchRunner(core.CommandRunner):
            def run(self, command, cwd=None, check=True):
                self.commands.append(command)
                if command[:4] == [
                    'git',
                    'ls-remote',
                    '--exit-code',
                    '--heads',
                ]:
                    return subprocess.CompletedProcess(command, 0, '', '')
                return subprocess.CompletedProcess(command, 0, '', '')

        with TemporaryDirectory() as tmpdir:
            root = Path(tmpdir)
            (root / 'kubernetes_platform/python/docs').mkdir(parents=True)
            (
                root / 'kubernetes_platform/python/docs/.readthedocs.yml'
            ).write_text('version: 2\n')
            (root / 'kubernetes_platform').mkdir(exist_ok=True)
            (root / 'kubernetes_platform/.gitignore').write_text('build\n')
            context = core.ReleaseContext(
                root=root,
                state=core.ReleaseState(root / 'state.json'),
                runner=ExistingBranchRunner(dry_run=False),
                metadata=core.ReleaseMetadata.from_version('minor', '3.2.0'),
                fork_remote='git@github.com:testuser/pipelines.git',
                include_backend=True,
                include_sdk=True,
            )

            with (
                mock.patch('builtins.input', return_value='replace'),
                mock.patch.object(context.runner, 'capture', return_value=''),
            ):
                steps.step_create_kfp_kubernetes_docs_branch(context)

            self.assertIn(
                ['git', 'push', 'upstream', ':kfp-kubernetes-3.2'],
                context.runner.commands,
            )
            self.assertIn(
                ['git', 'fetch', 'upstream', 'release-3.2'],
                context.runner.commands,
            )
            self.assertIn(
                [
                    'git',
                    'checkout',
                    '-B',
                    'release-3.2',
                    'upstream/release-3.2',
                ],
                context.runner.commands,
            )
            self.assertIn(
                ['git', 'checkout', '-B', 'kfp-kubernetes-3.2'],
                context.runner.commands,
            )
            self.assertIn(
                [
                    'git',
                    'push',
                    '--set-upstream',
                    'upstream',
                    'kfp-kubernetes-3.2',
                ],
                context.runner.commands,
            )

    def test_kfp_kubernetes_docs_branch_can_force_push_existing_remote_branch(
        self,
    ):

        class ExistingBranchRunner(core.CommandRunner):
            def run(self, command, cwd=None, check=True):
                self.commands.append(command)
                if command[:4] == [
                    'git',
                    'ls-remote',
                    '--exit-code',
                    '--heads',
                ]:
                    return subprocess.CompletedProcess(command, 0, '', '')
                return subprocess.CompletedProcess(command, 0, '', '')

        with TemporaryDirectory() as tmpdir:
            root = Path(tmpdir)
            (root / 'kubernetes_platform/python/docs').mkdir(parents=True)
            (
                root / 'kubernetes_platform/python/docs/.readthedocs.yml'
            ).write_text('version: 2\n')
            context = core.ReleaseContext(
                root=root,
                state=core.ReleaseState(root / 'state.json'),
                runner=ExistingBranchRunner(dry_run=False),
                metadata=core.ReleaseMetadata.from_version('minor', '3.2.0'),
                fork_remote='git@github.com:testuser/pipelines.git',
                include_backend=True,
                include_sdk=True,
            )

            with (
                mock.patch('builtins.input', return_value='force-push'),
                mock.patch.object(context.runner, 'capture', return_value=''),
            ):
                steps.step_create_kfp_kubernetes_docs_branch(context)

            self.assertIn(
                ['git', 'checkout', '-B', 'kfp-kubernetes-3.2'],
                context.runner.commands,
            )
            self.assertIn(
                [
                    'git',
                    'push',
                    '--force-with-lease',
                    '--set-upstream',
                    'upstream',
                    'kfp-kubernetes-3.2',
                ],
                context.runner.commands,
            )

    def test_merge_version_pr_dry_run_does_not_print_empty_pr_wait(self):
        with TemporaryDirectory() as tmpdir:
            root = Path(tmpdir)
            context = core.ReleaseContext(
                root=root,
                state=core.ReleaseState(root / 'state.json'),
                runner=core.CommandRunner(dry_run=True),
                metadata=core.ReleaseMetadata.from_version('minor', '3.2.0'),
                fork_remote='git@github.com:testuser/pipelines.git',
                include_backend=True,
                include_sdk=True,
            )

            with mock.patch('builtins.print') as print_mock:
                steps.step_merge_version_pr(context)

            output = '\n'.join(
                str(call.args[0]) for call in print_mock.call_args_list
            )
            self.assertIn(
                '[dry-run] would open PR: version bump for 3.2.0', output
            )
            self.assertNotIn('Version PR:', output)
            self.assertNotIn('Waiting for PR merge:', output)


class ReleaseNotesPreviewTest(QualifiedPublicationTest):
    def setUp(self):
        super().setUp()
        layout = mock.patch.object(
            steps, '_sdk_release_is_consolidated', return_value=True
        )
        layout.start()
        self.addCleanup(layout.stop)
        for name, value in (
            ('_notes_range', SELECTED_RANGE),
            ('read_commits', [Commit('c' * 40, 'fix(sdk): repair logs')]),
            ('_remote_release_sha', 'b' * 40),
        ):
            patcher = mock.patch.object(steps, name, return_value=value)
            patcher.start()
            self.addCleanup(patcher.stop)

    def context(self, root, dry_run=False):
        runner = mock.Mock(dry_run=dry_run)
        runner.run.return_value = subprocess.CompletedProcess([], 1)
        return core.ReleaseContext(
            root=root,
            state=core.ReleaseState(root / 'state.json'),
            runner=runner,
            metadata=core.ReleaseMetadata.from_version('patch', '2.17.3'),
            fork_remote='origin',
            include_backend=True,
            include_sdk=True,
            previous_release='2.17.2',
        )

    def test_dry_run_writes_backend_and_sdk_previews_without_publishing(self):
        with TemporaryDirectory() as tmpdir:
            root = Path(tmpdir)
            context = self.context(root, dry_run=True)
            with (
                mock.patch.object(
                    steps.subprocess,
                    'check_output',
                    return_value='version.py\n',
                ),
                mock.patch.object(
                    steps,
                    '_generate_backend_release_notes',
                    return_value='## Bug fixes\n\n* Repair logs.',
                ),
                mock.patch('builtins.input') as input_mock,
            ):
                steps.step_create_backend_release(context)
                steps.step_create_sdk_release(context)
            self.assertIn(
                'Repair logs.', (root / 'state-notes/2.17.3.md').read_text()
            )
            self.assertIn(
                'pip install kfp==2.17.3',
                (root / 'state-notes/sdk-2.17.3.md').read_text(),
            )
            context.runner.run.assert_not_called()
            input_mock.assert_not_called()

    def test_retry_publishes_edited_preview_without_regenerating(self):
        for sdk in (False, True):
            with self.subTest(sdk=sdk), TemporaryDirectory() as tmpdir:
                root = Path(tmpdir)
                context = self.context(root)
                tag = 'sdk-2.17.3' if sdk else '2.17.3'
                preview = root / 'state-notes' / f'{tag}.md'
                preview.parent.mkdir()
                preview.with_suffix('.json').write_text(
                    json.dumps(SELECTED_RANGE.as_dict())
                )
                preview.write_text(
                    '## Upgrade\n\nRetain this reviewed guidance.\n'
                )
                handler = (
                    steps.step_create_sdk_release
                    if sdk
                    else steps.step_create_backend_release
                )

                def edit_preview(_):
                    preview.write_text(
                        preview.read_text() + 'Final review edit.\n'
                    )
                    return 'y'

                with (
                    mock.patch('builtins.input', side_effect=edit_preview),
                    mock.patch.object(
                        steps, '_generate_backend_release_notes'
                    ) as generate,
                ):
                    handler(context)
                generate.assert_not_called()
                command = context.runner.run.call_args.args[0]
                self.assertEqual(command[:4], ['gh', 'release', 'create', tag])
                self.assertEqual(
                    Path(command[command.index('--notes-file') + 1]),
                    preview.resolve(),
                )
                self.assertIn(
                    'Retain this reviewed guidance.', preview.read_text()
                )
                self.assertIn('Final review edit.', preview.read_text())

    def test_declined_preview_does_not_publish_or_delete_existing_release(self):
        for sdk in (False, True):
            with self.subTest(sdk=sdk), TemporaryDirectory() as tmpdir:
                root = Path(tmpdir)
                context = self.context(root)
                context.runner.run.return_value.returncode = 0
                handler = (
                    steps.step_create_sdk_release
                    if sdk
                    else steps.step_create_backend_release
                )
                with (
                    mock.patch(
                        'builtins.input',
                        side_effect=['r', 'n'] if sdk else ['n'],
                    ),
                    mock.patch.object(
                        steps,
                        '_generate_backend_release_notes',
                        return_value='## Fixes\n\n* Fix logs.',
                    ),
                ):
                    with self.assertRaises(SystemExit):
                        handler(context)
                for call in context.runner.run.call_args_list:
                    self.assertNotIn(call.args[0][2], ('create', 'delete'))

    def test_empty_edited_preview_cannot_be_published(self):
        for sdk in (False, True):
            with self.subTest(sdk=sdk), TemporaryDirectory() as tmpdir:
                root = Path(tmpdir)
                context = self.context(root)
                tag = 'sdk-2.17.3' if sdk else '2.17.3'
                preview = root / 'state-notes' / f'{tag}.md'
                preview.parent.mkdir()
                preview.with_suffix('.json').write_text(
                    json.dumps(SELECTED_RANGE.as_dict())
                )
                preview.write_text('Reviewed notes')

                def clear_preview(_):
                    preview.write_text('   \n')
                    return 'y'

                handler = (
                    steps.step_create_sdk_release
                    if sdk
                    else steps.step_create_backend_release
                )
                with mock.patch('builtins.input', side_effect=clear_preview):
                    with self.assertRaisesRegex(ValueError, 'empty'):
                        handler(context)
                for call in context.runner.run.call_args_list:
                    self.assertNotIn(call.args[0][2], ('create', 'delete'))

    def test_empty_generated_backend_notes_stop_before_publication(self):
        with TemporaryDirectory() as tmpdir:
            context = self.context(Path(tmpdir))
            with mock.patch.object(
                steps, '_generate_backend_release_notes', return_value=''
            ):
                with self.assertRaisesRegex(
                    ValueError, 'No backend release notes'
                ):
                    steps.step_create_backend_release(context)
            context.runner.run.assert_not_called()


if __name__ == '__main__':
    unittest.main()
