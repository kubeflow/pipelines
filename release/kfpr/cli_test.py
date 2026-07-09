#!/usr/bin/env python3
"""Tests for the kfpr CLI."""

import subprocess
import sys
import unittest
from pathlib import Path
from tempfile import TemporaryDirectory

from typer.testing import CliRunner

from kfpr import core, steps
from kfpr.cli import app
from kfpr.core import ReleaseState


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
    self.assertEqual(result.exit_code, 0, f'CLI --help failed with: {result.stdout}')

  def test_kfpr_cli_short_help_runnable(self):
    runner = CliRunner()
    result = runner.invoke(app, ['-h'])
    self.assertEqual(result.exit_code, 0, f'CLI -h failed with: {result.stdout}')
    self.assertIn('run', result.stdout)

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
    self.assertIn('[build-system]', pyproject.read_text())

  def test_sdk_client_ci_installs_local_server_api_after_requirements(self):
    workflow = Path(__file__).parents[2] / '.github/workflows/kfp-sdk-client-tests.yml'
    text = workflow.read_text()
    self.assertLess(
        text.index('name: Install Test dependencies'),
        text.index('name: Build & install kfp-server-api dist'),
    )

  def test_readthedocs_ci_uses_nested_run_step_command(self):
    workflow = Path(__file__).parents[2] / '.github/workflows/readthedocs-builds.yml'
    self.assertIn('kfpr run create-kfp-kubernetes-docs-branch', workflow.read_text())


class CliTest(unittest.TestCase):

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
    self.assertIn('--diagram', result.stdout)

  def test_run_help_lists_release_source_branch_flag(self):
    result = CliRunner().invoke(app, ['run', '-h'], env={'COLUMNS': '200'})
    self.assertEqual(result.exit_code, 0, result.stdout)
    self.assertIn('--release-source-branch', result.stdout)
    self.assertIn('update-version-tags', result.stdout)
    self.assertIn('watch-publish-images', result.stdout)

  def test_run_step_command_short_help_runnable(self):
    result = CliRunner().invoke(app, ['run', 'preflight', '-h'])
    self.assertEqual(result.exit_code, 0, result.stdout)
    self.assertIn('--done', result.stdout)
    self.assertNotIn('--mark-done', result.stdout)

  def test_docs_branch_step_does_not_prompt_for_patch_prs(self):
    result = CliRunner().invoke(
        app,
        [
            'run',
            'create-kfp-kubernetes-docs-branch',
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
              'create-kfp-kubernetes-docs-branch',
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
      state.answers.update({
          'release_type': 'minor',
          'version': '2.17.0',
          'fork_remote': 'https://github.com/droctothorpe/pipelines.git',
          'include_backend': True,
          'include_sdk': True,
      })
      state.save()

      result = CliRunner().invoke(
          app,
          ['run', 'watch-publish-images', '--state-file', str(state_file), '--dry-run'],
      )

      self.assertEqual(result.exit_code, 0, result.stdout)
      self.assertIn('Watching latest workflow run skipped: image-builds-release.yml on release-2.17', result.stdout)
      self.assertNotIn('workflow run image-builds-release.yml', result.stdout)

  def test_steps_lists_selected_flow(self):
    result = CliRunner().invoke(
        app,
        ['steps', '--release-type', 'patch', '--include-backend', '--no-include-sdk'],
    )
    self.assertEqual(result.exit_code, 0, result.stdout)
    self.assertIn('cherry-pick-prs', result.stdout)
    self.assertIn('publish-images', result.stdout)
    self.assertNotIn('publish-sdks', result.stdout)

  def test_steps_marks_manual_steps(self):
    result = CliRunner().invoke(app, ['steps', '--release-type', 'minor'])

    self.assertEqual(result.exit_code, 0, result.stdout)
    self.assertIn('confirm-rtd [manual]: Confirm ReadTheDocs updates', result.stdout)
    self.assertIn(
        'confirm-website-and-slack [manual]: Confirm website PR and Slack announcement',
        result.stdout,
    )
    self.assertIn('preflight: Verify tools and GitHub auth', result.stdout)
    self.assertNotIn('preflight [manual]', result.stdout)

  def test_steps_diagram_prints_release_flow_branches(self):
    result = CliRunner().invoke(app, ['steps', '--diagram'])

    self.assertEqual(result.exit_code, 0, result.stdout)
    self.assertIn('+-- patch --> prepare-patch-branch', result.stdout)
    self.assertIn('+-- major/minor --> prepare-release-branch', result.stdout)
    self.assertIn('+-- include-backend --> publish-images', result.stdout)
    self.assertIn('+-- include-sdk major/minor --> create-kfp-kubernetes-docs-branch', result.stdout)
    self.assertIn('confirm-rtd -> create-sdk-release -> publish-sdks', result.stdout)
    self.assertNotIn('preflight: Verify tools', result.stdout)

  def test_next_runs_and_marks_next_incomplete_step(self):
    with TemporaryDirectory() as tmpdir:
      state_file = Path(tmpdir) / 'state.json'
      state = ReleaseState(state_file)
      state.answers.update({
          'release_type': 'minor',
          'version': '3.2.0',
          'fork_remote': 'git@github.com:testuser/pipelines.git',
          'include_backend': False,
          'include_sdk': False,
          'previous_release': '3.1.1',
      })
      state.save()

      result = CliRunner().invoke(app, ['next', '--state-file', str(state_file), '--dry-run'])

      self.assertEqual(result.exit_code, 0, result.stdout)
      self.assertIn('Running step: Verify tools and GitHub auth', result.stdout)
      self.assertTrue(ReleaseState.load(state_file).is_done('preflight'))

  def test_next_previous_release_flag_is_stored(self):
    with TemporaryDirectory() as tmpdir:
      state_file = Path(tmpdir) / 'state.json'
      state = ReleaseState(state_file)
      state.answers.update({
          'release_type': 'minor',
          'version': '3.2.0',
          'fork_remote': 'git@github.com:testuser/pipelines.git',
          'include_backend': False,
          'include_sdk': False,
      })
      state.save()

      result = CliRunner().invoke(
          app,
          ['next', '--state-file', str(state_file), '--dry-run', '--previous-release', '3.1.1'],
      )

      self.assertEqual(result.exit_code, 0, result.stdout)
      self.assertEqual(ReleaseState.load(state_file).answers['previous_release'], '3.1.1')

  def test_next_prompts_for_previous_release_and_stores_it(self):
    with TemporaryDirectory() as tmpdir:
      state_file = Path(tmpdir) / 'state.json'
      state = ReleaseState(state_file)
      state.answers.update({
          'release_type': 'minor',
          'version': '3.2.0',
          'fork_remote': 'git@github.com:testuser/pipelines.git',
          'include_backend': False,
          'include_sdk': False,
      })
      state.save()

      result = CliRunner().invoke(
          app,
          ['next', '--state-file', str(state_file), '--dry-run'],
          input='3.1.1\n',
      )

      self.assertEqual(result.exit_code, 0, result.stdout)
      self.assertEqual(ReleaseState.load(state_file).answers['previous_release'], '3.1.1')

  def test_status_prints_state_next_step_resume_command_and_manual_checklist(self):
    with TemporaryDirectory() as tmpdir:
      state_file = Path(tmpdir) / 'state.json'
      state = ReleaseState(state_file)
      state.answers.update({
          'release_type': 'minor',
          'version': '3.2.0',
          'fork_remote': 'git@github.com:testuser/pipelines.git',
          'include_backend': True,
          'include_sdk': True,
      })
      for step in steps.build_steps('minor', True, True):
        if step.step_id == 'confirm-rtd':
          break
        state.mark_done(step.step_id)
      state.save()

      result = CliRunner().invoke(app, ['status', '--state-file', str(state_file)])

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

      result = CliRunner().invoke(app, ['clear', '--state-file', str(state_file)])

      self.assertEqual(result.exit_code, 0, result.stdout)
      self.assertFalse(state_file.exists())
      self.assertIn(f'deleted checkpoint: {state_file}', result.stdout)

  def test_clear_is_ok_when_checkpoint_file_is_missing(self):
    with TemporaryDirectory() as tmpdir:
      state_file = Path(tmpdir) / 'state.json'

      result = CliRunner().invoke(app, ['clear', '--state-file', str(state_file)])

      self.assertEqual(result.exit_code, 0, result.stdout)
      self.assertIn(f'no checkpoint to delete: {state_file}', result.stdout)

  def test_validate_state_rejects_unknown_completed_step(self):
    with TemporaryDirectory() as tmpdir:
      state_file = Path(tmpdir) / 'state.json'
      state = ReleaseState(state_file)
      state.answers.update({
          'release_type': 'minor',
          'version': '3.2.0',
          'fork_remote': 'git@github.com:testuser/pipelines.git',
          'include_backend': True,
          'include_sdk': True,
      })
      state.mark_done('not-a-step')
      state.save()

      result = CliRunner().invoke(app, ['validate-state', '--state-file', str(state_file)])

      self.assertNotEqual(result.exit_code, 0, result.stdout)
      self.assertIn('unknown completed step: not-a-step', result.stdout)

  def test_validate_state_reports_incompatible_json_shape(self):
    with TemporaryDirectory() as tmpdir:
      state_file = Path(tmpdir) / 'state.json'
      state_file.write_text('{"answers": [], "completed_steps": [123]}')

      result = CliRunner().invoke(app, ['validate-state', '--state-file', str(state_file)])

      self.assertNotEqual(result.exit_code, 0, result.stdout)
      self.assertIn('answers must be an object', result.stdout)
      self.assertIn('completed_steps must be a list of strings', result.stdout)
      self.assertNotIn('AttributeError', str(result.exception))

  def test_done_and_reset_step_edit_checkpoint(self):
    with TemporaryDirectory() as tmpdir:
      state_file = Path(tmpdir) / 'state.json'

      mark_result = CliRunner().invoke(app, ['done', 'preflight', '--state-file', str(state_file)])
      self.assertEqual(mark_result.exit_code, 0, mark_result.stdout)
      self.assertTrue(ReleaseState.load(state_file).is_done('preflight'))

      reset_result = CliRunner().invoke(app, ['reset-step', 'preflight', '--state-file', str(state_file)])
      self.assertEqual(reset_result.exit_code, 0, reset_result.stdout)
      self.assertFalse(ReleaseState.load(state_file).is_done('preflight'))

  def test_goto_marks_prior_steps_done_and_target_next(self):
    with TemporaryDirectory() as tmpdir:
      state_file = Path(tmpdir) / 'state.json'
      state = ReleaseState(state_file)
      state.answers.update({
          'release_type': 'minor',
          'version': '3.2.0',
          'fork_remote': 'git@github.com:testuser/pipelines.git',
          'include_backend': True,
          'include_sdk': True,
      })
      state.mark_done('preflight')
      state.mark_done('create-sdk-release')
      state.save()

      result = CliRunner().invoke(app, ['goto', 'confirm-rtd', '--state-file', str(state_file)])

      self.assertEqual(result.exit_code, 0, result.stdout)
      updated = ReleaseState.load(state_file)
      step_ids = [step.step_id for step in steps.build_steps('minor', True, True)]
      self.assertEqual(updated.completed_steps, step_ids[:step_ids.index('confirm-rtd')])
      self.assertFalse(updated.is_done('confirm-rtd'))
      self.assertFalse(updated.is_done('create-sdk-release'))
      self.assertIn('next step: confirm-rtd', result.stdout)

  def test_goto_rejects_step_outside_selected_flow(self):
    with TemporaryDirectory() as tmpdir:
      state_file = Path(tmpdir) / 'state.json'
      state = ReleaseState(state_file)
      state.answers.update({
          'release_type': 'patch',
          'version': '3.2.1',
          'fork_remote': 'git@github.com:testuser/pipelines.git',
          'patch_prs': '12345',
          'include_backend': True,
          'include_sdk': True,
      })
      state.save()

      result = CliRunner().invoke(app, ['goto', 'confirm-rtd', '--state-file', str(state_file)])

      self.assertNotEqual(result.exit_code, 0, result.stdout)
      self.assertIn('step is not in this release flow: confirm-rtd', result.stdout)

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

  def test_run_with_existing_state_prints_status_and_aborts_when_user_declines(self):
    with TemporaryDirectory() as tmpdir:
      state_file = Path(tmpdir) / 'state.json'
      state = ReleaseState(state_file)
      state.answers.update({
          'release_type': 'patch',
          'version': '3.2.1',
          'fork_remote': 'git@github.com:testuser/pipelines.git',
          'patch_prs': '12345',
          'include_backend': True,
          'include_sdk': True,
      })
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
      state.answers.update({
          'release_type': 'patch',
          'version': '3.2.1',
          'fork_remote': 'git@github.com:testuser/pipelines.git',
          'patch_prs': '12345',
          'include_backend': False,
          'include_sdk': False,
          'previous_release': '3.2.0',
      })
      for step in steps.build_steps('patch', False, False):
        state.mark_done(step.step_id)
      state.save()

      result = CliRunner().invoke(
          app,
          ['run', '--state-file', str(state_file), '--dry-run', '--force'],
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
      self.assertTrue(ReleaseState.load(state_file).answers['skip_local_review'])

  def test_single_step_without_changelog_does_not_prompt_for_previous_release(self):
    with TemporaryDirectory() as tmpdir:
      state_file = Path(tmpdir) / 'state.json'
      result = CliRunner().invoke(
          app,
          [
              'run',
              'create-kfp-kubernetes-docs-branch',
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
      self.assertNotIn('previous_release', ReleaseState.load(state_file).answers)

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
      self.assertEqual(state.answers['fork_remote'], 'https://github.com/droctothorpe/pipelines.git')


if __name__ == '__main__':
  unittest.main()
