#!/usr/bin/env python3
"""Tests for release steps."""

import unittest
from pathlib import Path
from tempfile import TemporaryDirectory
from unittest import mock

from kfpr import core, steps


class StepSelectionTest(unittest.TestCase):

  def test_patch_includes_cherry_pick_steps(self):
    steps_list = [step.step_id for step in steps.build_steps('patch', include_backend=True, include_sdk=True)]
    self.assertIn('cherry-pick-prs', steps_list)
    self.assertIn('publish-images', steps_list)
    self.assertIn('publish-sdks', steps_list)

  def test_major_skips_cherry_pick_steps(self):
    steps_list = [step.step_id for step in steps.build_steps('major', include_backend=True, include_sdk=True)]
    self.assertNotIn('cherry-pick-prs', steps_list)

  def test_backend_only_patch_skips_sdk_steps(self):
    steps_list = [step.step_id for step in steps.build_steps('patch', include_backend=True, include_sdk=False)]
    self.assertIn('publish-images', steps_list)
    self.assertNotIn('publish-sdks', steps_list)


class StepRegistryTest(unittest.TestCase):

  def test_step_registry_contains_every_built_step(self):
    for release_type in ('major', 'minor', 'patch'):
      for step in steps.build_steps(release_type, include_backend=True, include_sdk=True):
        self.assertIn(step.step_id, steps.STEP_HANDLERS)
        self.assertIs(steps.STEP_HANDLERS[step.step_id], getattr(steps, step.handler))


class CherryPickStepTest(unittest.TestCase):

  def test_cherry_pick_skips_pr_without_merge_commit(self):
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

      with mock.patch('builtins.print'):
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
    steps_list = steps.build_steps('patch', include_backend=True, include_sdk=True)
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

      self.assertIn(['git', 'checkout', 'release-candidate'], context.runner.commands)
      self.assertNotIn(['git', 'checkout', 'master'], context.runner.commands)

  def test_prepare_release_branch_reuses_existing_branch_when_confirmed(self):
    with TemporaryDirectory() as tmpdir:
      state = core.ReleaseState(Path(tmpdir) / 'state.json')
      state.answers['release_source_branch'] = 'master'

      class ExistingBranchRunner(core.CommandRunner):

        def __init__(self):
          super().__init__(dry_run=True)

        def capture(self, command, cwd=None):
          self.commands.append(command)
          return 'release-3.2'

      context = core.ReleaseContext(
          root=Path(tmpdir),
          state=state,
          runner=ExistingBranchRunner(),
          metadata=core.ReleaseMetadata.from_version('minor', '3.2.0'),
          fork_remote='origin',
          include_backend=True,
          include_sdk=True,
      )

      with mock.patch('builtins.input', return_value='y'):
        steps.step_prepare_release_branch(context)

      self.assertIn(['git', 'checkout', 'release-3.2'], context.runner.commands)
      self.assertNotIn(['git', 'checkout', '-b', 'release-3.2'], context.runner.commands)


class OrchestrationTest(unittest.TestCase):

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
          with self.subTest(release_type=release_type, backend=include_backend, sdk=include_sdk):
            steps_list = steps.build_steps(release_type, include_backend, include_sdk)
            step_ids = [step.step_id for step in steps_list]

            self.assertIn('create-backend-release', step_ids)
            self.assertIn('sync-master', step_ids)
            self.assertIn('confirm-website-and-slack', step_ids)

  def test_run_steps_persists_completed_state_after_each_step(self):
    with TemporaryDirectory() as tmpdir:
      state_file = Path(tmpdir) / 'state.json'
      state = core.ReleaseState(state_file)
      state.answers['version'] = '3.0.0'
      state.answers['release_type'] = 'major'
      state.answers['fork_remote'] = 'git@github.com:testuser/pipelines.git'
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
      self.assertGreater(len(loaded_state.completed_steps), 0)

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

      with mock.patch.object(steps, 'build_steps', return_value=fake_steps):
        with mock.patch.object(steps, 'noop_step', create=True, side_effect=lambda context: None):
          with mock.patch('builtins.print') as print_mock:
            steps.run_steps(context)

      output = '\n'.join(str(call.args[0]) if call.args else '' for call in print_mock.call_args_list)
      self.assertIn('Running step: First step\n\nRunning step: Second step', output)


class StepPRCreationTest(unittest.TestCase):

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

        def capture(self, command, cwd=None):
          self.commands.append(command)
          if 'view' in command and any('mergeStateStatus' in part for part in command):
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
      self.assertEqual(head_value, 'testuser:release-1.3')


class StepUpdateSdkVersionsTest(unittest.TestCase):

  def test_step_update_sdk_versions_replaces_version_without_trailing_garbage(self):
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

      steps.step_update_sdk_versions(context)

      self.assertEqual(version_file.read_text(), "__version__ = '1.2.4'\n")


class DryRunOutputTest(unittest.TestCase):

  def test_update_version_tags_dry_run_prints_version_write_and_tag(self):
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

      output = '\n'.join(str(call.args[0]) for call in print_mock.call_args_list)
      self.assertIn('[dry-run] would write VERSION: 3.2.0', output)
      self.assertIn('[dry-run] would create tag: 3.2.0', output)

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

      flattened_commands = [' '.join(command) for command in context.runner.commands]
      self.assertFalse(
          any('test/release/Makefile' in command for command in flattened_commands),
          flattened_commands,
      )

  def test_update_version_tags_edits_release_dockerfile(self):
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

      flattened_commands = [' '.join(command) for command in context.runner.commands]
      self.assertTrue(
          any('release/Dockerfile.release' in command for command in flattened_commands),
          flattened_commands,
      )
      self.assertTrue(
          any('ARG BASE_IMAGE=ghcr.io/kubeflow/kfp-api-generator:' in command for command in flattened_commands),
          flattened_commands,
      )
      self.assertFalse(
          any('test/release/Dockerfile.release' in command for command in flattened_commands),
          flattened_commands,
      )

  def test_update_version_tags_uses_master_release_image_for_minor_release(self):
    with TemporaryDirectory() as tmpdir:
      root = Path(tmpdir)
      state = core.ReleaseState(root / 'state.json')
      state.answers['release_source_branch'] = 'release-automation'
      context = core.ReleaseContext(
          root=root,
          state=state,
          runner=core.CommandRunner(dry_run=True),
          metadata=core.ReleaseMetadata.from_version('minor', '3.2.0'),
          fork_remote='git@github.com:testuser/pipelines.git',
          include_backend=True,
          include_sdk=True,
      )

      steps.step_update_version_tags(context)

      flattened_commands = [' '.join(command) for command in context.runner.commands]
      self.assertTrue(
          any('ghcr.io/kubeflow/kfp-release:master' in command for command in flattened_commands),
          flattened_commands,
      )
      self.assertFalse(
          any('ghcr.io/kubeflow/kfp-release:release-automation' in command for command in flattened_commands),
          flattened_commands,
      )

  def test_manual_checkpoint_text_can_be_reused_by_status(self):
    metadata = core.ReleaseMetadata.from_version('minor', '3.2.0')

    self.assertIn('ReadTheDocs manual checkpoint:', steps.manual_checklist('confirm-rtd', metadata))
    self.assertIn('https://app.readthedocs.org/projects/kubeflow-pipelines/', steps.manual_checklist('confirm-rtd', metadata))
    self.assertIn('Manual final checkpoint:', steps.manual_checklist('confirm-website-and-slack', metadata))

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

      with mock.patch('getpass.getpass', return_value='secret-token'), mock.patch.object(steps.rtd, 'ReadTheDocsClient') as client_cls, mock.patch.object(steps.rtd, 'update_release_docs') as update_docs, mock.patch('builtins.print') as print_mock:
        steps.step_confirm_rtd(context)

      client_cls.assert_called_once_with('secret-token')
      update_docs.assert_called_once_with(client_cls.return_value, '3.2.0', 'release-3.2')
      self.assertNotIn('rtd_token', context.state.answers)
      output = '\n'.join(str(call.args[0]) for call in print_mock.call_args_list)
      self.assertNotIn('ReadTheDocs manual checkpoint:', output)

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

      with mock.patch('getpass.getpass', side_effect=['', 'secret-token']), mock.patch.object(steps.rtd, 'ReadTheDocsClient') as client_cls, mock.patch.object(steps.rtd, 'update_release_docs'):
        steps.step_confirm_rtd(context)

      client_cls.assert_called_once_with('secret-token')

  def test_confirm_rtd_falls_back_to_manual_when_automation_fails_and_user_accepts(self):
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

      with mock.patch('getpass.getpass', return_value='secret-token'), mock.patch.object(steps.rtd, 'ReadTheDocsClient'), mock.patch.object(steps.rtd, 'update_release_docs', side_effect=steps.rtd.ReadTheDocsError('boom')), mock.patch('builtins.input', side_effect=['y', 'y']):
        steps.step_confirm_rtd(context)

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

      output = '\n'.join(str(call.args[0]) for call in print_mock.call_args_list)
      self.assertIn('[dry-run] would open PR: version bump for 3.2.0', output)
      self.assertNotIn('Version PR:', output)
      self.assertNotIn('Waiting for PR merge:', output)


if __name__ == '__main__':
  unittest.main()
