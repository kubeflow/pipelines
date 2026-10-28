#!/usr/bin/env python3
"""Tests for the kfpr CLI."""

import contextlib
import re
import subprocess
import sys
import unittest
from pathlib import Path
from tempfile import TemporaryDirectory
from unittest import mock

from typer.testing import CliRunner

from kfpr import core, steps
from kfpr.cli import app
from kfpr.core import ReleaseState


def _strip_ansi(text: str) -> str:
    return re.sub(r'\x1b\[[0-?]*[ -/]*[@-~]', '', text)


class PackageImportTest(unittest.TestCase):
    def test_kfpr_package_imports(self):
        self.assertTrue(hasattr(core, 'ReleaseMetadata'))
        self.assertTrue(hasattr(steps, 'build_steps'))

    def test_kfpr_cli_imports(self):
        from kfpr.cli import app

        self.assertIsNotNone(app)
        self.assertTrue(hasattr(app, 'command'))

    def test_kfpr_cli_help_runnable(self):
        runner = CliRunner()
        result = runner.invoke(app, ['--help'])
        self.assertEqual(
            result.exit_code, 0, f'CLI --help failed with: {result.stdout}'
        )

    def test_kfpr_cli_short_help_runnable(self):
        runner = CliRunner()
        result = runner.invoke(app, ['-h'])
        self.assertEqual(
            result.exit_code, 0, f'CLI -h failed with: {result.stdout}'
        )
        self.assertIn('run', result.stdout)

    def test_qualification_command_exposes_python_matrix_configuration(self):
        from kfpr import cli

        with (
            mock.patch.object(cli, '_context') as context,
            mock.patch.dict(
                cli.STEP_HANDLERS, {'qualify-release': mock.Mock()}
            ),
        ):
            result = CliRunner().invoke(
                app,
                [
                    'run',
                    'qualify-release',
                    '--dry-run',
                    '--qualification-pythons',
                    '/python39,/python311',
                    '--security-policy',
                    '2.17.3-approved-exceptions-v1',
                ],
            )
        self.assertEqual(result.exit_code, 0, result.stdout)
        self.assertEqual(
            context.call_args.kwargs['qualification_pythons'],
            '/python39,/python311',
        )
        self.assertEqual(
            context.call_args.kwargs['security_policy'],
            '2.17.3-approved-exceptions-v1',
        )

    def test_general_security_exception_file_option(self):
        from kfpr import cli

        with (
            mock.patch.object(cli, '_context') as context,
            mock.patch.dict(
                cli.STEP_HANDLERS, {'qualify-release': mock.Mock()}
            ),
        ):
            result = CliRunner().invoke(
                app,
                [
                    'run',
                    'qualify-release',
                    '--dry-run',
                    '--security-exceptions',
                    '/tmp/exceptions.json',
                ],
            )
        self.assertEqual(result.exit_code, 0, result.stdout)
        self.assertEqual(
            context.call_args.kwargs['security_exceptions'],
            Path('/tmp/exceptions.json'),
        )

    def test_local_image_qualification_is_rejected(self):
        from kfpr import cli

        with (
            mock.patch.object(cli, '_context') as context,
            mock.patch.dict(
                cli.STEP_HANDLERS, {'qualify-release': mock.Mock()}
            ),
        ):
            result = CliRunner().invoke(
                app,
                [
                    'run',
                    'qualify-release',
                    '--dry-run',
                    '--image-qualification',
                    'local',
                ],
            )
            self.assertNotEqual(result.exit_code, 0)
            context.assert_not_called()
            rejected = CliRunner().invoke(
                app,
                [
                    'run',
                    'qualify-release',
                    '--dry-run',
                    '--image-qualification',
                    'automatic',
                ],
            )
            self.assertNotEqual(rejected.exit_code, 0)

    def test_image_mode_defaults_to_ci_and_rejects_saved_local(self):
        from kfpr import cli

        with TemporaryDirectory() as temporary:
            state = ReleaseState(Path(temporary) / 'state.json')
            with (
                mock.patch.object(cli, '_load_state', return_value=state),
                mock.patch.object(cli, 'collect_context'),
            ):
                args = [state.path, False, None, None, None, None, None, None]
                cli._context(*args)
                self.assertEqual(state.answers['image_qualification'], 'ci')
                state.answers['image_qualification'] = 'local'
                with self.assertRaises(cli.typer.BadParameter):
                    cli._context(*args)
                cli._context(*args, image_qualification='ci')
                self.assertEqual(state.answers['image_qualification'], 'ci')
                with self.assertRaises(cli.typer.BadParameter):
                    cli._context(
                        *args, qualification_pythons='/usr/bin/python3'
                    )

    def test_exception_selection_persists_and_strict_selection_clears_it(self):
        from kfpr import cli

        with TemporaryDirectory() as temporary:
            state = ReleaseState(Path(temporary) / 'state.json')
            exceptions = Path(temporary) / 'exceptions.json'
            exceptions.write_text('{"schema": 1, "exceptions": []}')
            with (
                mock.patch.object(cli, '_load_state', return_value=state),
                mock.patch.object(cli, 'collect_context'),
            ):
                args = [state.path, False, None, None, None, None, None, None]
                cli._context(*args, security_exceptions=exceptions)
                self.assertEqual(
                    state.answers['security_policy'], 'approved-exceptions'
                )
                self.assertEqual(
                    state.answers['security_exceptions'],
                    str(exceptions.resolve()),
                )
                cli._context(*args)
                self.assertEqual(
                    state.answers['security_policy'], 'approved-exceptions'
                )
                cli._context(*args, security_policy='zero-fixable')
                self.assertNotIn('security_exceptions', state.answers)
                with self.assertRaises(cli.typer.BadParameter):
                    cli._context(*args, security_policy='approved-exceptions')

    def test_promote_latest_dispatches_ci_without_release_checkpoint(self):
        promotion = mock.Mock()
        with mock.patch.dict(sys.modules, {'kfpr.promotion': promotion}):
            result = CliRunner().invoke(
                app,
                [
                    'promote-latest',
                    '--version',
                    '2.17.3',
                    '--dry-run',
                ],
            )
        self.assertEqual(result.exit_code, 0, result.stdout)
        promotion.dispatch.assert_called_once_with('2.17.3', dry_run=True)

    def test_kfpr_cli_has_module_entrypoint(self):
        result = subprocess.run(
            [sys.executable, '-m', 'kfpr.cli', '--help'],
            check=False,
            capture_output=True,
            text=True,
        )
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn('run', result.stdout)

    def test_installed_console_script_is_kfpr(self):
        pyproject = Path(__file__).parents[1] / 'pyproject.toml'
        self.assertRegex(pyproject.read_text(), r'(?m)^kfpr = "kfpr\.cli:app"$')

    def test_pyproject_declares_build_backend(self):
        pyproject = Path(__file__).parents[1] / 'pyproject.toml'
        pyproject_text = pyproject.read_text()
        self.assertIn('[build-system]', pyproject_text)
        self.assertIn('"typer>=0.16,<1.0"', pyproject_text)

    def test_sdk_client_ci_installs_local_server_api_after_requirements(self):
        workflow = (
            Path(__file__).parents[2]
            / '.github/workflows/kfp-sdk-client-tests.yml'
        )
        text = workflow.read_text()
        self.assertLess(
            text.index('name: Install Test dependencies'),
            text.index('name: Build and install the unified SDK'),
        )

    def test_readthedocs_ci_uses_nested_run_step_command(self):
        workflow = (
            Path(__file__).parents[2]
            / '.github/workflows/readthedocs-builds.yml'
        )
        workflow_text = workflow.read_text()
        self.assertIn('kfpr run create-sdk-tag', workflow_text)
        self.assertIn(
            'uv run --with-editable ./release kfpr run', workflow_text
        )
        self.assertIn(
            'uv run --with-editable ./release python -m unittest', workflow_text
        )

    def test_kubernetes_release_guide_uses_nested_step_command(self):
        guide = (
            Path(__file__).parents[2] / 'kubernetes_platform/python/RELEASE.md'
        )
        guide_text = guide.read_text()
        self.assertIn('single `kfp` distribution', guide_text)
        self.assertIn('release/README.md', guide_text)
        self.assertNotIn('source release.sh', guide_text)


class CliTest(unittest.TestCase):
    def test_docs_step_preserves_dirty_checkout_and_checkpoint(self) -> None:
        """Exercise the CLI guard with real staged and unstaged Git changes."""
        for consolidated, dry_run in (
            (False, False),
            (True, False),
            (False, True),
        ):
            with (
                self.subTest(consolidated=consolidated, dry_run=dry_run),
                TemporaryDirectory() as directory,
            ):
                root = Path(directory) / 'repo'
                root.mkdir()
                state_file = Path(directory) / 'state.json'

                def git(*arguments: str) -> str:
                    return subprocess.check_output(
                        ['git', *arguments], cwd=root, text=True
                    )

                git('init', '--quiet')
                (root / '.readthedocs.yml').write_text('original docs\n')
                (root / 'unrelated.txt').write_text('original work\n')
                (root / 'kubernetes_platform').mkdir()
                gitignore = root / 'kubernetes_platform/.gitignore'
                gitignore.write_text('build\n')
                git('add', '.')
                git(
                    '-c',
                    'user.name=Release test',
                    '-c',
                    'user.email=release@example.invalid',
                    '-c',
                    'commit.gpgsign=false',
                    'commit',
                    '--quiet',
                    '-m',
                    'fixture',
                )
                (root / '.readthedocs.yml').write_text('uncommitted docs\n')
                (root / 'unrelated.txt').write_text('staged work\n')
                git('add', 'unrelated.txt')
                before = (
                    git('rev-parse', 'HEAD'),
                    git('symbolic-ref', 'HEAD'),
                    git('status', '--short'),
                    git('diff', '--cached'),
                )
                arguments = [
                    'run',
                    'create-kfp-kubernetes-docs-branch',
                    '--release-type',
                    'patch',
                    '--version',
                    '2.18.1',
                    '--fork-remote',
                    'testuser',
                    '--state-file',
                    str(state_file),
                    '--done',
                ]
                if dry_run:
                    arguments.append('--dry-run')
                with (
                    mock.patch.object(
                        steps,
                        '_sdk_release_is_consolidated',
                        return_value=consolidated,
                    ) as layout,
                    mock.patch.object(
                        core.CommandRunner,
                        'run',
                        side_effect=AssertionError(
                            'Unexpected branch operation'
                        ),
                    ) as operation,
                    contextlib.chdir(root),
                ):
                    result = CliRunner().invoke(app, arguments)

                operation.assert_not_called()
                if not consolidated and not dry_run:
                    self.assertNotEqual(result.exit_code, 0)
                    self.assertIsInstance(result.exception, RuntimeError)
                    self.assertIn(
                        'working tree is dirty', str(result.exception)
                    )
                else:
                    self.assertEqual(result.exit_code, 0, result.exception)
                self.assertEqual(layout.call_count, 0 if dry_run else 1)
                self.assertEqual(
                    ReleaseState.load(state_file).is_done(
                        'create-kfp-kubernetes-docs-branch'
                    ),
                    consolidated or dry_run,
                )
                self.assertEqual(
                    before,
                    (
                        git('rev-parse', 'HEAD'),
                        git('symbolic-ref', 'HEAD'),
                        git('status', '--short'),
                        git('diff', '--cached'),
                    ),
                )
                self.assertEqual(
                    (root / '.readthedocs.yml').read_text(),
                    'uncommitted docs\n',
                )
                self.assertEqual(
                    (root / 'unrelated.txt').read_text(), 'staged work\n'
                )
                self.assertEqual(gitignore.read_text(), 'build\n')

    def test_help_lists_full_flow_and_step_commands(self):
        result = CliRunner().invoke(app, ['--help'])
        self.assertEqual(result.exit_code, 0, result.stdout)
        self.assertIn('run', result.stdout)
        self.assertIn('steps', result.stdout)
        self.assertIn('clear', result.stdout)
        self.assertNotIn('update-version-tags', result.stdout)
        self.assertNotIn('publish-images', result.stdout)
        self.assertNotIn('confirm-rtd', result.stdout)

    def test_normal_command_short_help_runnable(self):
        result = CliRunner().invoke(app, ['steps', '-h'])
        self.assertEqual(result.exit_code, 0, result.stdout)
        self.assertIn('--diagram', _strip_ansi(result.stdout))

    def test_run_help_lists_release_source_branch_flag(self):
        result = CliRunner().invoke(app, ['run', '-h'], env={'COLUMNS': '200'})
        self.assertEqual(result.exit_code, 0, result.stdout)
        output = _strip_ansi(result.stdout)
        self.assertIn('--release-source-branch', output)
        self.assertIn('update-version-tags', output)
        self.assertIn('watch-publish-images', output)

    def test_run_step_command_short_help_runnable(self):
        result = CliRunner().invoke(app, ['run', 'preflight', '-h'])
        self.assertEqual(result.exit_code, 0, result.stdout)
        output = _strip_ansi(result.stdout)
        self.assertIn('--done', output)
        self.assertNotIn('--mark-done', output)

    def test_docs_branch_step_does_not_prompt_for_patch_prs(self):
        result = CliRunner().invoke(
            app,
            [
                'run',
                'create-sdk-tag',
                '--release-type',
                'patch',
                '--version',
                '2.16.1',
                '--fork-remote',
                'upstream',
                '--previous-release',
                '2.16.0',
                '--dry-run',
            ],
        )
        self.assertEqual(result.exit_code, 0, result.stdout)
        self.assertNotIn('Comma-separated PR numbers', result.stdout)

    def test_generated_step_without_mark_done_does_not_write_checkpoint(self):
        with TemporaryDirectory() as tmpdir:
            state_file = Path(tmpdir) / 'state.json'

            result = CliRunner().invoke(
                app,
                [
                    'run',
                    'create-sdk-tag',
                    '--state-file',
                    str(state_file),
                    '--release-type',
                    'patch',
                    '--version',
                    '2.16.1',
                    '--fork-remote',
                    'upstream',
                    '--previous-release',
                    '2.16.0',
                    '--dry-run',
                ],
            )

            self.assertEqual(result.exit_code, 0, result.stdout)
            self.assertFalse(state_file.exists())

    def test_watch_publish_images_watches_without_dispatching(self):
        with TemporaryDirectory() as tmpdir:
            state_file = Path(tmpdir) / 'state.json'
            state = ReleaseState(state_file)
            state.answers.update(
                {
                    'release_type': 'minor',
                    'version': '2.17.0',
                    'fork_remote': 'https://github.com/droctothorpe/pipelines.git',
                    'include_backend': True,
                    'include_sdk': True,
                }
            )
            state.save()

            result = CliRunner().invoke(
                app,
                [
                    'run',
                    'watch-publish-images',
                    '--state-file',
                    str(state_file),
                    '--dry-run',
                ],
            )

            self.assertEqual(result.exit_code, 0, result.stdout)
            self.assertIn(
                'Watching latest workflow run skipped: image-builds-release.yml on release-2.17',
                result.stdout,
            )
            self.assertNotIn(
                'workflow run image-builds-release.yml', result.stdout
            )

    def test_steps_lists_selected_flow(self):
        result = CliRunner().invoke(
            app,
            [
                'steps',
                '--release-type',
                'patch',
                '--include-backend',
                '--no-include-sdk',
            ],
        )
        self.assertEqual(result.exit_code, 0, result.stdout)
        self.assertIn('cherry-pick-prs', result.stdout)
        self.assertIn('publish-images', result.stdout)
        self.assertNotIn('publish-sdks', result.stdout)

    def test_steps_marks_manual_steps(self):
        result = CliRunner().invoke(app, ['steps', '--release-type', 'minor'])

        self.assertEqual(result.exit_code, 0, result.stdout)
        self.assertIn(
            'confirm-rtd [manual]: Confirm ReadTheDocs updates', result.stdout
        )
        self.assertIn(
            'confirm-website-and-slack [manual]: Confirm website PR and Slack announcement',
            result.stdout,
        )
        self.assertIn('preflight: Verify tools and GitHub auth', result.stdout)
        self.assertNotIn('preflight [manual]', result.stdout)
        self.assertIn(
            'create-kfp-kubernetes-docs-branch: '
            'Create Kubernetes docs branch for split SDK releases',
            result.stdout,
        )

    def test_steps_diagram_prints_release_flow_branches(self):
        result = CliRunner().invoke(app, ['steps', '--diagram'])

        self.assertEqual(result.exit_code, 0, result.stdout)
        self.assertIn('+-- patch --> prepare-patch-branch', result.stdout)
        self.assertIn(
            '+-- major/minor --> prepare-release-branch', result.stdout
        )
        self.assertIn('+-- include-backend --> publish-images', result.stdout)
        self.assertIn(
            '+-- include-sdk --> create-sdk-tag -> publish-sdks', result.stdout
        )
        self.assertIn('confirm-rtd -> create-sdk-release', result.stdout)
        self.assertIn(
            '[split SDK: create-kfp-kubernetes-docs-branch]', result.stdout
        )
        self.assertNotIn('preflight: Verify tools', result.stdout)

    def test_next_runs_and_marks_next_incomplete_step(self):
        with TemporaryDirectory() as tmpdir:
            state_file = Path(tmpdir) / 'state.json'
            state = ReleaseState(state_file)
            state.answers.update(
                {
                    'release_type': 'minor',
                    'version': '3.2.0',
                    'fork_remote': 'git@github.com:testuser/pipelines.git',
                    'include_backend': False,
                    'include_sdk': False,
                    'previous_release': '3.1.1',
                }
            )
            state.save()

            result = CliRunner().invoke(
                app, ['next', '--state-file', str(state_file), '--dry-run']
            )

            self.assertEqual(result.exit_code, 0, result.stdout)
            self.assertIn(
                'Running step: Verify tools and GitHub auth', result.stdout
            )
            self.assertFalse(ReleaseState.load(state_file).is_done('preflight'))

    def test_next_previous_release_flag_is_stored(self):
        with TemporaryDirectory() as tmpdir:
            state_file = Path(tmpdir) / 'state.json'
            state = ReleaseState(state_file)
            state.answers.update(
                {
                    'release_type': 'minor',
                    'version': '3.2.0',
                    'fork_remote': 'git@github.com:testuser/pipelines.git',
                    'include_backend': False,
                    'include_sdk': False,
                }
            )
            state.save()

            result = CliRunner().invoke(
                app,
                [
                    'next',
                    '--state-file',
                    str(state_file),
                    '--dry-run',
                    '--previous-release',
                    '3.1.1',
                ],
            )

            self.assertEqual(result.exit_code, 0, result.stdout)
            self.assertEqual(
                ReleaseState.load(state_file).answers['previous_release'],
                '3.1.1',
            )

    def test_next_saves_separate_sdk_base_and_target_options(self):
        with TemporaryDirectory() as tmpdir:
            state_file = Path(tmpdir) / 'state.json'
            state = ReleaseState(state_file)
            state.answers.update(
                {
                    'release_type': 'patch',
                    'version': '3.2.1',
                    'fork_remote': 'https://github.com/testuser/pipelines.git',
                    'patch_prs': 'not-needed',
                    'include_backend': False,
                    'include_sdk': True,
                }
            )
            state.save()
            result = CliRunner().invoke(
                app,
                [
                    'next',
                    '--state-file',
                    str(state_file),
                    '--dry-run',
                    '--previous-sdk-release',
                    'sdk-3.2.0',
                    '--release-target',
                    'release-candidate',
                ],
            )
            self.assertEqual(result.exit_code, 0, result.stdout)
            answers = ReleaseState.load(state_file).answers
            self.assertEqual(answers['previous_sdk_release'], 'sdk-3.2.0')
            self.assertEqual(answers['release_target'], 'release-candidate')
            self.assertNotIn('previous_release', answers)

    def test_next_without_backend_does_not_prompt_for_backend_base(self):
        with TemporaryDirectory() as tmpdir:
            state_file = Path(tmpdir) / 'state.json'
            state = ReleaseState(state_file)
            state.answers.update(
                {
                    'release_type': 'minor',
                    'version': '3.2.0',
                    'fork_remote': 'git@github.com:testuser/pipelines.git',
                    'include_backend': False,
                    'include_sdk': False,
                }
            )
            state.save()

            result = CliRunner().invoke(
                app,
                ['next', '--state-file', str(state_file), '--dry-run'],
                input='3.1.1\n',
            )

            self.assertEqual(result.exit_code, 0, result.stdout)
            self.assertNotIn(
                'previous_release', ReleaseState.load(state_file).answers
            )

    def test_status_prints_state_next_step_resume_command_and_manual_checklist(
        self,
    ):
        with TemporaryDirectory() as tmpdir:
            state_file = Path(tmpdir) / 'state.json'
            state = ReleaseState(state_file)
            state.answers.update(
                {
                    'release_type': 'minor',
                    'version': '3.2.0',
                    'fork_remote': 'git@github.com:testuser/pipelines.git',
                    'include_backend': True,
                    'include_sdk': True,
                }
            )
            for step in steps.build_steps('minor', True, True):
                if step.step_id == 'confirm-rtd':
                    break
                state.mark_done(step.step_id)
            state.save()

            result = CliRunner().invoke(
                app, ['status', '--state-file', str(state_file)]
            )

            self.assertEqual(result.exit_code, 0, result.stdout)
            self.assertIn('release_type: minor', result.stdout)
            self.assertIn('[done] preflight', result.stdout)
            self.assertIn('[next] confirm-rtd', result.stdout)
            self.assertIn(f'kfpr run --state-file {state_file}', result.stdout)
            self.assertIn('ReadTheDocs manual checkpoint:', result.stdout)

    def test_clear_deletes_checkpoint_file(self):
        with TemporaryDirectory() as tmpdir:
            state_file = Path(tmpdir) / 'state.json'
            state_file.write_text('{"answers": {}, "completed_steps": []}')

            result = CliRunner().invoke(
                app, ['clear', '--state-file', str(state_file)]
            )

            self.assertEqual(result.exit_code, 0, result.stdout)
            self.assertFalse(state_file.exists())
            self.assertIn(f'deleted checkpoint: {state_file}', result.stdout)

    def test_clear_is_ok_when_checkpoint_file_is_missing(self):
        with TemporaryDirectory() as tmpdir:
            state_file = Path(tmpdir) / 'state.json'

            result = CliRunner().invoke(
                app, ['clear', '--state-file', str(state_file)]
            )

            self.assertEqual(result.exit_code, 0, result.stdout)
            self.assertIn(
                f'no checkpoint to delete: {state_file}', result.stdout
            )

    def test_validate_state_rejects_unknown_completed_step(self):
        with TemporaryDirectory() as tmpdir:
            state_file = Path(tmpdir) / 'state.json'
            state = ReleaseState(state_file)
            state.answers.update(
                {
                    'release_type': 'minor',
                    'version': '3.2.0',
                    'fork_remote': 'git@github.com:testuser/pipelines.git',
                    'include_backend': True,
                    'include_sdk': True,
                }
            )
            state.mark_done('not-a-step')
            state.save()

            result = CliRunner().invoke(
                app, ['validate-state', '--state-file', str(state_file)]
            )

            self.assertNotEqual(result.exit_code, 0, result.stdout)
            self.assertIn('unknown completed step: not-a-step', result.stdout)

    def test_validate_state_reports_incompatible_json_shape(self):
        with TemporaryDirectory() as tmpdir:
            state_file = Path(tmpdir) / 'state.json'
            state_file.write_text('{"answers": [], "completed_steps": [123]}')

            result = CliRunner().invoke(
                app, ['validate-state', '--state-file', str(state_file)]
            )

            self.assertNotEqual(result.exit_code, 0, result.stdout)
            self.assertIn('answers must be an object', result.stdout)
            self.assertIn(
                'completed_steps must be a list of strings', result.stdout
            )
            self.assertNotIn('AttributeError', str(result.exception))

    def test_done_and_reset_step_edit_checkpoint(self):
        with TemporaryDirectory() as tmpdir:
            state_file = Path(tmpdir) / 'state.json'

            mark_result = CliRunner().invoke(
                app, ['done', 'preflight', '--state-file', str(state_file)]
            )
            self.assertEqual(mark_result.exit_code, 0, mark_result.stdout)
            self.assertTrue(ReleaseState.load(state_file).is_done('preflight'))

            reset_result = CliRunner().invoke(
                app,
                ['reset-step', 'preflight', '--state-file', str(state_file)],
            )
            self.assertEqual(reset_result.exit_code, 0, reset_result.stdout)
            self.assertFalse(ReleaseState.load(state_file).is_done('preflight'))

    def test_goto_marks_prior_steps_done_and_target_next(self):
        with TemporaryDirectory() as tmpdir:
            state_file = Path(tmpdir) / 'state.json'
            state = ReleaseState(state_file)
            state.answers.update(
                {
                    'release_type': 'minor',
                    'version': '3.2.0',
                    'fork_remote': 'git@github.com:testuser/pipelines.git',
                    'include_backend': True,
                    'include_sdk': True,
                }
            )
            state.mark_done('preflight')
            state.mark_done('create-sdk-release')
            state.save()

            result = CliRunner().invoke(
                app, ['goto', 'confirm-rtd', '--state-file', str(state_file)]
            )

            self.assertEqual(result.exit_code, 0, result.stdout)
            updated = ReleaseState.load(state_file)
            step_ids = [
                step.step_id for step in steps.build_steps('minor', True, True)
            ]
            self.assertEqual(
                updated.completed_steps,
                step_ids[: step_ids.index('confirm-rtd')],
            )
            self.assertFalse(updated.is_done('confirm-rtd'))
            self.assertFalse(updated.is_done('create-sdk-release'))
            self.assertIn('next step: confirm-rtd', result.stdout)

    def test_goto_allows_patch_docs_step(self):
        with TemporaryDirectory() as tmpdir:
            state_file = Path(tmpdir) / 'state.json'
            state = ReleaseState(state_file)
            state.answers.update(
                {
                    'release_type': 'patch',
                    'version': '3.2.1',
                    'fork_remote': 'git@github.com:testuser/pipelines.git',
                    'patch_prs': '12345',
                    'include_backend': True,
                    'include_sdk': True,
                }
            )
            state.save()

            result = CliRunner().invoke(
                app, ['goto', 'confirm-rtd', '--state-file', str(state_file)]
            )

            self.assertEqual(result.exit_code, 0, result.stdout)
            self.assertIn('next step: confirm-rtd', result.stdout)

    def test_doctor_rejects_bad_inputs_without_writing_state(self):
        with TemporaryDirectory() as tmpdir:
            state_file = Path(tmpdir) / 'state.json'
            result = CliRunner().invoke(
                app,
                [
                    'doctor',
                    '--release-type',
                    'minor',
                    '--version',
                    '3.2.1',
                    '--fork-remote',
                    '@droctothorpe',
                    '--state-file',
                    str(state_file),
                ],
            )

            self.assertNotEqual(result.exit_code, 0, result.stdout)
            self.assertIn('minor release version must be X.Y.0', result.stdout)
            self.assertIn('Cannot parse fork_remote', result.stdout)
            self.assertFalse(state_file.exists())

    def test_new_release_full_dry_run_completes_without_release_refs(self):
        for release_type, version in (('major', '4.0.0'), ('minor', '3.3.0')):
            with (
                self.subTest(release_type=release_type),
                TemporaryDirectory() as tmpdir,
            ):
                root = Path(tmpdir)
                state_file = root / 'state.json'
                with mock.patch.object(
                    core.subprocess, 'check_output', return_value=str(root)
                ) as git_read:
                    result = CliRunner().invoke(
                        app,
                        [
                            'run',
                            '--dry-run',
                            '--release-type',
                            release_type,
                            '--version',
                            version,
                            '--fork-remote',
                            'testuser',
                            '--release-source-branch',
                            'master',
                            '--previous-release',
                            '3.2.0',
                            '--previous-sdk-release',
                            'sdk-3.2.0',
                            '--state-file',
                            str(state_file),
                        ],
                    )
                self.assertEqual(result.exit_code, 0, result.output)
                self.assertIn(
                    f'Release flow complete for {version}', result.output
                )
                git_read.assert_called_once_with(
                    ['git', 'rev-parse', '--show-toplevel'], text=True
                )
                self.assertFalse((root / 'state-notes').exists())
                self.assertFalse(ReleaseState.load(state_file).completed_steps)

    def test_run_with_existing_state_prints_status_and_aborts_when_user_declines(
        self,
    ):
        with TemporaryDirectory() as tmpdir:
            state_file = Path(tmpdir) / 'state.json'
            state = ReleaseState(state_file)
            state.answers.update(
                {
                    'release_type': 'patch',
                    'version': '3.2.1',
                    'fork_remote': 'git@github.com:testuser/pipelines.git',
                    'patch_prs': '12345',
                    'include_backend': True,
                    'include_sdk': True,
                }
            )
            state.mark_done('preflight')
            state.save()

            result = CliRunner().invoke(
                app,
                ['run', '--state-file', str(state_file), '--dry-run'],
                input='n\n',
            )

            self.assertEqual(result.exit_code, 1, result.stdout)
            self.assertIn('answers:', result.stdout)
            self.assertIn('release_type: patch', result.stdout)
            self.assertIn('[done] preflight', result.stdout)
            self.assertIn('Continue from this checkpoint?', result.stdout)
            self.assertIn('Release flow aborted.', result.stdout)
            self.assertNotIn('Release flow complete', result.stdout)

    def test_run_force_skips_existing_state_prompt(self):
        with TemporaryDirectory() as tmpdir:
            state_file = Path(tmpdir) / 'state.json'
            state = ReleaseState(state_file)
            state.answers.update(
                {
                    'release_type': 'patch',
                    'version': '3.2.1',
                    'fork_remote': 'git@github.com:testuser/pipelines.git',
                    'patch_prs': '12345',
                    'include_backend': False,
                    'include_sdk': False,
                    'previous_release': '3.2.0',
                }
            )
            for step in steps.build_steps('patch', False, False):
                state.mark_done(step.step_id)
            state.save()

            result = CliRunner().invoke(
                app,
                [
                    'run',
                    '--state-file',
                    str(state_file),
                    '--dry-run',
                    '--force',
                ],
            )

            self.assertEqual(result.exit_code, 0, result.stdout)
            self.assertNotIn('Continue from this checkpoint?', result.stdout)
            self.assertIn('Release flow complete for 3.2.1', result.stdout)

    def test_single_step_does_not_mark_done_by_default(self):
        with TemporaryDirectory() as tmpdir:
            state_file = Path(tmpdir) / 'state.json'
            result = CliRunner().invoke(
                app,
                [
                    'run',
                    'preflight',
                    '--release-type',
                    'major',
                    '--version',
                    '3.0.0',
                    '--fork-remote',
                    'git@github.com:testuser/pipelines.git',
                    '--previous-release',
                    '2.16.1',
                    '--state-file',
                    str(state_file),
                    '--dry-run',
                ],
            )
            self.assertEqual(result.exit_code, 0, result.stdout)
            self.assertFalse(ReleaseState.load(state_file).is_done('preflight'))

    def test_single_step_mark_done_updates_checkpoint(self):
        with TemporaryDirectory() as tmpdir:
            state_file = Path(tmpdir) / 'state.json'
            result = CliRunner().invoke(
                app,
                [
                    'run',
                    'preflight',
                    '--release-type',
                    'major',
                    '--version',
                    '3.0.0',
                    '--fork-remote',
                    'git@github.com:testuser/pipelines.git',
                    '--previous-release',
                    '2.16.1',
                    '--state-file',
                    str(state_file),
                    '--dry-run',
                    '--done',
                ],
            )
            self.assertEqual(result.exit_code, 0, result.stdout)
            self.assertTrue(ReleaseState.load(state_file).is_done('preflight'))

    def test_single_step_skip_local_review_is_stored(self):
        with TemporaryDirectory() as tmpdir:
            state_file = Path(tmpdir) / 'state.json'
            result = CliRunner().invoke(
                app,
                [
                    'run',
                    'preflight',
                    '--release-type',
                    'major',
                    '--version',
                    '3.0.0',
                    '--fork-remote',
                    'git@github.com:testuser/pipelines.git',
                    '--previous-release',
                    '2.16.1',
                    '--state-file',
                    str(state_file),
                    '--dry-run',
                    '--done',
                    '--skip-local-review',
                ],
            )
            self.assertEqual(result.exit_code, 0, result.stdout)
            self.assertTrue(
                ReleaseState.load(state_file).answers['skip_local_review']
            )

    def test_single_step_without_changelog_does_not_prompt_for_previous_release(
        self,
    ):
        with TemporaryDirectory() as tmpdir:
            state_file = Path(tmpdir) / 'state.json'
            result = CliRunner().invoke(
                app,
                [
                    'run',
                    'create-sdk-tag',
                    '--release-type',
                    'minor',
                    '--version',
                    '3.2.0',
                    '--fork-remote',
                    'git@github.com:testuser/pipelines.git',
                    '--state-file',
                    str(state_file),
                    '--dry-run',
                ],
            )
            self.assertEqual(result.exit_code, 0, result.stdout)
            self.assertNotIn(
                'previous_release', ReleaseState.load(state_file).answers
            )

    def test_bare_fork_username_option_is_saved_as_https_remote(self):
        with TemporaryDirectory() as tmpdir:
            state_file = Path(tmpdir) / 'state.json'
            result = CliRunner().invoke(
                app,
                [
                    'run',
                    'preflight',
                    '--release-type',
                    'major',
                    '--version',
                    '3.0.0',
                    '--fork-remote',
                    'droctothorpe',
                    '--previous-release',
                    '2.16.1',
                    '--state-file',
                    str(state_file),
                    '--dry-run',
                    '--done',
                ],
            )
            self.assertEqual(result.exit_code, 0, result.stdout)
            state = ReleaseState.load(state_file)
            self.assertEqual(
                state.answers['fork_remote'],
                'https://github.com/droctothorpe/pipelines.git',
            )


if __name__ == '__main__':
    unittest.main()


class RegenerateNotesTest(unittest.TestCase):
    def test_saved_context_and_overrides_do_not_write_checkpoint(self):
        for product in ('backend', 'sdk'):
            with (
                self.subTest(product=product),
                TemporaryDirectory() as directory,
            ):
                state_file = Path(directory) / 'state.json'
                state = ReleaseState(state_file)
                state.answers.update(
                    {
                        'version': '2.17.3',
                        'release_type': 'patch',
                        'previous_release': '2.17.2',
                        'previous_sdk_release': 'sdk-2.17.2',
                        'release_target': 'old-target',
                        'fork_remote': 'origin',
                    }
                )
                state.save()
                original = state_file.read_bytes()
                with (
                    mock.patch(
                        'kfpr.cli.regenerate_release_notes', return_value=True
                    ) as regenerate,
                    mock.patch(
                        'kfpr.cli.subprocess.check_output',
                        return_value=directory + '\n',
                    ) as git,
                    mock.patch.object(ReleaseState, 'save') as save,
                ):
                    result = CliRunner().invoke(
                        app,
                        [
                            'regenerate-notes',
                            '--product',
                            product,
                            '--state-file',
                            str(state_file),
                            '--version',
                            '2.17.4',
                            '--previous-release',
                            '2.17.3',
                            '--release-target',
                            'candidate',
                        ],
                    )
                self.assertEqual(result.exit_code, 0, result.output)
                context, selected_product = regenerate.call_args.args
                self.assertEqual(selected_product, product)
                self.assertEqual(context.metadata.tag, '2.17.4')
                self.assertEqual(context.previous_release, '2.17.3')
                self.assertEqual(context.previous_sdk_release, 'sdk-2.17.2')
                self.assertEqual(context.release_target, 'candidate')
                self.assertEqual(context.root, Path(directory))
                self.assertEqual(context.include_backend, product == 'backend')
                self.assertEqual(context.include_sdk, product == 'sdk')
                self.assertTrue(context.runner.dry_run)
                self.assertEqual(context.runner.commands, [])
                git.assert_called_once_with(
                    ['git', 'rev-parse', '--show-toplevel'], text=True
                )
                save.assert_not_called()
                self.assertEqual(state_file.read_bytes(), original)

    def test_new_preview_without_checkpoint_and_conflict_status(self):
        for clean in (True, False):
            with self.subTest(clean=clean), TemporaryDirectory() as directory:
                state_file = Path(directory) / 'missing.json'
                with (
                    mock.patch(
                        'kfpr.cli.regenerate_release_notes', return_value=clean
                    ) as regenerate,
                    mock.patch(
                        'kfpr.cli.subprocess.check_output',
                        return_value=directory,
                    ),
                ):
                    result = CliRunner().invoke(
                        app,
                        [
                            'regenerate-notes',
                            '--product',
                            'sdk',
                            '--state-file',
                            str(state_file),
                            '--version',
                            '2.17.3',
                            '--previous-sdk-release',
                            'sdk-2.17.2',
                            '--release-target',
                            'candidate',
                        ],
                    )
                self.assertEqual(
                    result.exit_code, 0 if clean else 1, result.output
                )
                context = regenerate.call_args.args[0]
                self.assertEqual(context.metadata.release_type, 'patch')
                self.assertEqual(context.fork_remote, '')
                self.assertEqual(context.previous_sdk_release, 'sdk-2.17.2')
                self.assertFalse(state_file.exists())
                self.assertIn(
                    'Review' if clean else 'conflict markers', result.output
                )

    def test_invalid_or_missing_inputs_never_regenerate(self):
        for options in (
            [],
            ['--version', 'invalid'],
            ['--version', '2.17.3', '--product', 'unknown'],
        ):
            with (
                self.subTest(options=options),
                TemporaryDirectory() as directory,
            ):
                with (
                    mock.patch(
                        'kfpr.cli.regenerate_release_notes'
                    ) as regenerate,
                    mock.patch.object(ReleaseState, 'save') as save,
                ):
                    result = CliRunner().invoke(
                        app,
                        [
                            'regenerate-notes',
                            '--product',
                            'backend',
                            '--state-file',
                            str(Path(directory) / 'missing.json'),
                            *options,
                        ],
                    )
                self.assertEqual(result.exit_code, 2, result.output)
                regenerate.assert_not_called()
                save.assert_not_called()


class ValidateNotesTest(unittest.TestCase):
    def test_editorial_warnings_are_advisory_and_file_is_unchanged(self):
        for warnings in (
            [],
            ['Explain the user impact.', 'Add migration instructions.'],
        ):
            with (
                self.subTest(warnings=warnings),
                TemporaryDirectory() as directory,
            ):
                path = Path(directory) / 'notes.md'
                content = '# Release notes\n\nHandwritten notes.\n'
                path.write_text(content)
                with (
                    mock.patch(
                        'kfpr.cli.validate_notes', return_value=warnings
                    ) as validate,
                    mock.patch.object(ReleaseState, 'save') as save,
                    mock.patch.object(core.CommandRunner, 'run') as run,
                ):
                    result = CliRunner().invoke(
                        app, ['validate-notes', str(path)]
                    )
                self.assertEqual(result.exit_code, 0, result.output)
                validate.assert_called_once_with(content)
                for warning in warnings:
                    self.assertIn(f'Warning: {warning}', result.output)
                if not warnings:
                    self.assertIn('No editorial warnings', result.output)
                self.assertEqual(path.read_text(), content)
                save.assert_not_called()
                run.assert_not_called()

    def test_missing_or_invalid_utf8_file_has_friendly_error(self):
        for invalid_utf8 in (False, True):
            with (
                self.subTest(invalid_utf8=invalid_utf8),
                TemporaryDirectory() as directory,
            ):
                path = Path(directory) / 'notes.md'
                if invalid_utf8:
                    path.write_bytes(b'\xff')
                with mock.patch('kfpr.cli.validate_notes') as validate:
                    result = CliRunner().invoke(
                        app, ['validate-notes', str(path)]
                    )
                self.assertEqual(result.exit_code, 2, result.output)
                self.assertIn('Cannot read release notes', result.output)
                validate.assert_not_called()
