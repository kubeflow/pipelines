#!/usr/bin/env python3
"""Tests for release steps."""

import unittest
import subprocess
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
    self.assertIn('create-sdk-release', steps_list)
    self.assertLess(steps_list.index('create-sdk-release'), steps_list.index('publish-sdks'))
    self.assertNotIn('create-kfp-kubernetes-docs-branch', steps_list)
    self.assertNotIn('confirm-rtd', steps_list)
    self.assertNotIn('update-sdk-versions', steps_list)
    self.assertNotIn('merge-sdk-pr', steps_list)

  def test_minor_keeps_release_order_and_updates_docs(self):
    steps_list = [step.step_id for step in steps.build_steps('minor', include_backend=True, include_sdk=True)]
    self.assertLess(steps_list.index('create-sdk-release'), steps_list.index('publish-sdks'))
    self.assertLess(steps_list.index('publish-sdks'), steps_list.index('create-kfp-kubernetes-docs-branch'))
    self.assertLess(steps_list.index('create-kfp-kubernetes-docs-branch'), steps_list.index('confirm-rtd'))

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


class PublishImagesStepTest(unittest.TestCase):

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

      with mock.patch('time.time', return_value=1783537500), mock.patch('builtins.print') as print_mock:
        steps.step_publish_images(context)

      output = '\n'.join(str(call.args[0]) for call in print_mock.call_args_list)
      self.assertIn('Workflow run: \033[4mhttps://github.com/kubeflow/pipelines/actions/runs/12345\033[0m', output)
      self.assertIn(['gh', 'run', 'watch', '12345'], context.runner.commands)


class CreateSdkReleaseStepTest(unittest.TestCase):

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

      self.assertIn(['gh', 'release', 'view', 'sdk-3.2.0', '--repo', 'kubeflow/pipelines'], context.runner.commands)
      self.assertNotIn(['gh', 'release', 'create', 'sdk-3.2.0'], [command[:4] for command in context.runner.commands])

  def test_create_sdk_release_uses_existing_tag_when_user_accepts(self):
    with TemporaryDirectory() as tmpdir:

      class ExistingTagRunner:
        dry_run = False

        def __init__(self):
          self.commands = []

        def run(self, command, cwd=None, check=True):
          self.commands.append(command)
          if command[:4] == ['gh', 'release', 'view', 'sdk-3.2.0']:
            return subprocess.CompletedProcess(command, 1, '', '')
          if command[:3] == ['gh', 'api', '--silent']:
            return subprocess.CompletedProcess(command, 0, '', '')
          return subprocess.CompletedProcess(command, 0, '', '')

      context = core.ReleaseContext(
          root=Path(tmpdir),
          state=core.ReleaseState(Path(tmpdir) / 'state.json'),
          runner=ExistingTagRunner(),
          metadata=core.ReleaseMetadata.from_version('minor', '3.2.0'),
          fork_remote='origin',
          include_backend=True,
          include_sdk=True,
      )

      with mock.patch('builtins.input', return_value='u'):
        steps.step_create_sdk_release(context)

      self.assertIn(['gh', 'api', '--silent', 'repos/kubeflow/pipelines/git/ref/tags/sdk-3.2.0'], context.runner.commands)
      self.assertNotIn(['git', 'push', '--delete', 'origin', 'sdk-3.2.0'], context.runner.commands)
      self.assertIn(['gh', 'release', 'create', 'sdk-3.2.0'], [command[:4] for command in context.runner.commands])
      self.assertIn('--verify-tag', context.runner.commands[-1])

  def test_create_sdk_release_recreates_existing_tag_when_requested(self):
    with TemporaryDirectory() as tmpdir:

      class ExistingTagRunner:
        dry_run = False

        def __init__(self):
          self.commands = []

        def run(self, command, cwd=None, check=True):
          self.commands.append(command)
          if command[:4] == ['gh', 'release', 'view', 'sdk-3.2.0']:
            return subprocess.CompletedProcess(command, 1, '', '')
          if command[:3] == ['gh', 'api', '--silent']:
            return subprocess.CompletedProcess(command, 0, '', '')
          return subprocess.CompletedProcess(command, 0, '', '')

      context = core.ReleaseContext(
          root=Path(tmpdir),
          state=core.ReleaseState(Path(tmpdir) / 'state.json'),
          runner=ExistingTagRunner(),
          metadata=core.ReleaseMetadata.from_version('minor', '3.2.0'),
          fork_remote='origin',
          include_backend=True,
          include_sdk=True,
      )

      with mock.patch('builtins.input', return_value='r'):
        steps.step_create_sdk_release(context)

      self.assertIn(['gh', 'api', '--silent', 'repos/kubeflow/pipelines/git/ref/tags/sdk-3.2.0'], context.runner.commands)
      self.assertIn(['git', 'tag', '-f', 'sdk-3.2.0', 'release-3.2'], context.runner.commands)
      self.assertIn(['git', 'push', '--force', 'https://github.com/kubeflow/pipelines.git', 'sdk-3.2.0'], context.runner.commands)
      self.assertIn(['gh', 'release', 'create', 'sdk-3.2.0'], [command[:4] for command in context.runner.commands])
      self.assertIn('--verify-tag', context.runner.commands[-1])

  def test_create_sdk_release_pushes_missing_tag_before_creating_release(self):
    with TemporaryDirectory() as tmpdir:

      class MissingTagRunner:
        dry_run = False

        def __init__(self):
          self.commands = []

        def run(self, command, cwd=None, check=True):
          self.commands.append(command)
          if command[:4] == ['gh', 'release', 'view', 'sdk-3.2.0']:
            return subprocess.CompletedProcess(command, 1, '', '')
          if command[:3] == ['gh', 'api', '--silent']:
            return subprocess.CompletedProcess(command, 1, '', '')
          return subprocess.CompletedProcess(command, 0, '', '')

      context = core.ReleaseContext(
          root=Path(tmpdir),
          state=core.ReleaseState(Path(tmpdir) / 'state.json'),
          runner=MissingTagRunner(),
          metadata=core.ReleaseMetadata.from_version('minor', '3.2.0'),
          fork_remote='origin',
          include_backend=True,
          include_sdk=True,
      )

      steps.step_create_sdk_release(context)

      self.assertIn(['git', 'tag', '-f', 'sdk-3.2.0', 'release-3.2'], context.runner.commands)
      self.assertIn(['git', 'push', '--force', 'https://github.com/kubeflow/pipelines.git', 'sdk-3.2.0'], context.runner.commands)
      self.assertIn('--verify-tag', context.runner.commands[-1])


class CreateBackendReleaseStepTest(unittest.TestCase):

  def test_create_backend_release_generates_notes_without_prompting(self):
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

      with mock.patch('builtins.input') as input_mock, mock.patch.object(steps, '_generate_backend_release_notes', return_value=('3.1.0', '## Features\n\n* add pipelines (#1)')):
        steps.step_create_backend_release(context)

      input_mock.assert_not_called()
      release_command = context.runner.commands[-1]
      self.assertEqual(release_command[:4], ['gh', 'release', 'create', '3.2.0'])
      release_notes = release_command[release_command.index('--notes') + 1]
      self.assertIn('## Features\n\n* add pipelines (#1)', release_notes)
      self.assertIn('https://github.com/kubeflow/pipelines/compare/3.1.0...3.2.0', release_notes)


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

  def test_prepare_release_branch_skips_pull_without_source_upstream(self):
    with TemporaryDirectory() as tmpdir:
      state = core.ReleaseState(Path(tmpdir) / 'state.json')
      state.answers['release_source_branch'] = 'release-candidate'

      class NoUpstreamRunner(core.CommandRunner):

        def __init__(self):
          super().__init__(dry_run=True)

        def run(self, command, cwd=None, check=True):
          self.commands.append(command)
          if command == ['git', 'rev-parse', '--abbrev-ref', '--symbolic-full-name', '@{u}']:
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

      self.assertNotIn(['git', 'pull', '--ff-only'], context.runner.commands)
      self.assertIn(['git', 'checkout', '-b', 'release-3.2'], context.runner.commands)


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
          if command[:4] == ['git', 'ls-remote', '--exit-code', '--heads']:
            return subprocess.CompletedProcess(command, 1, '', '')
          if command[:4] == ['git', 'ls-remote', '--exit-code', '--tags']:
            return subprocess.CompletedProcess(command, 1, '', '')
          return subprocess.CompletedProcess(command, 0, '', '')

        def capture(self, command, cwd=None):
          self.commands.append(command)
          if 'view' in command and any('statusCheckRollup' in part for part in command):
            return '[{"name":"ci","status":"COMPLETED","conclusion":"SUCCESS"}]'
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
      self.assertEqual(head_value, 'testuser:1.3.0-update-version-tags')

  def test_step_merge_version_pr_pushes_missing_upstream_release_branch(self):
    with TemporaryDirectory() as tmpdir:
      class MissingUpstreamReleaseBranchRunner:

        dry_run = False

        def __init__(self):
          self.commands = []

        def run(self, command, cwd=None, check=True):
          self.commands.append(command)
          if command[:4] == ['git', 'ls-remote', '--exit-code', '--heads']:
            return subprocess.CompletedProcess(command, 1, '', '')
          if command[:4] == ['git', 'ls-remote', '--exit-code', '--tags']:
            return subprocess.CompletedProcess(command, 1, '', '')
          return subprocess.CompletedProcess(command, 0, '', '')

        def capture(self, command, cwd=None):
          self.commands.append(command)
          if 'view' in command and any('statusCheckRollup' in part for part in command):
            return '[{"name":"ci","status":"COMPLETED","conclusion":"SUCCESS"}]'
          if 'view' in command and any('mergeStateStatus' in part for part in command):
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
          ['git', 'push', 'https://github.com/kubeflow/pipelines.git', 'release-1.3'],
          runner.commands,
      )

  def test_step_merge_version_pr_skips_existing_upstream_release_branch_push(self):
    with TemporaryDirectory() as tmpdir:
      class ExistingUpstreamReleaseBranchRunner:

        dry_run = False

        def __init__(self):
          self.commands = []

        def run(self, command, cwd=None, check=True):
          self.commands.append(command)
          if command == ['git', 'ls-remote', '--exit-code', '--heads', 'https://github.com/kubeflow/pipelines.git', 'release-1.3']:
            return subprocess.CompletedProcess(command, 0, '', '')
          if command[:4] == ['git', 'ls-remote', '--exit-code', '--heads']:
            return subprocess.CompletedProcess(command, 1, '', '')
          if command[:4] == ['git', 'ls-remote', '--exit-code', '--tags']:
            return subprocess.CompletedProcess(command, 1, '', '')
          return subprocess.CompletedProcess(command, 0, '', '')

        def capture(self, command, cwd=None):
          self.commands.append(command)
          if 'view' in command and any('statusCheckRollup' in part for part in command):
            return '[{"name":"ci","status":"COMPLETED","conclusion":"SUCCESS"}]'
          if 'view' in command and any('mergeStateStatus' in part for part in command):
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
          ['git', 'push', 'https://github.com/kubeflow/pipelines.git', 'release-1.3'],
          runner.commands,
      )

  def test_step_merge_version_pr_uses_existing_remote_branch_when_requested(self):
    with TemporaryDirectory() as tmpdir:
      class ExistingRemoteBranchRunner:

        dry_run = False

        def __init__(self):
          self.commands = []

        def run(self, command, cwd=None, check=True):
          self.commands.append(command)
          if command[:4] == ['git', 'ls-remote', '--exit-code', '--heads']:
            return subprocess.CompletedProcess(command, 0, '', '')
          if command[:4] == ['git', 'ls-remote', '--exit-code', '--tags']:
            return subprocess.CompletedProcess(command, 1, '', '')
          return subprocess.CompletedProcess(command, 0, '', '')

        def capture(self, command, cwd=None):
          self.commands.append(command)
          if 'view' in command and any('statusCheckRollup' in part for part in command):
            return '[{"name":"ci","status":"COMPLETED","conclusion":"SUCCESS"}]'
          if 'view' in command and any('mergeStateStatus' in part for part in command):
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

      self.assertNotIn(
          ['git', 'push', '--set-upstream', 'git@github.com:testuser/pipelines.git', '1.3.0-update-version-tags'],
          runner.commands,
      )
      self.assertNotIn(
          ['git', 'push', '--delete', 'git@github.com:testuser/pipelines.git', '1.3.0-update-version-tags'],
          runner.commands,
      )

  def test_step_merge_version_pr_recreates_existing_remote_branch_when_requested(self):
    with TemporaryDirectory() as tmpdir:
      class ExistingRemoteBranchRunner:

        dry_run = False

        def __init__(self):
          self.commands = []

        def run(self, command, cwd=None, check=True):
          self.commands.append(command)
          if command[:4] == ['git', 'ls-remote', '--exit-code', '--heads']:
            return subprocess.CompletedProcess(command, 0, '', '')
          if command[:4] == ['git', 'ls-remote', '--exit-code', '--tags']:
            return subprocess.CompletedProcess(command, 1, '', '')
          return subprocess.CompletedProcess(command, 0, '', '')

        def capture(self, command, cwd=None):
          self.commands.append(command)
          if 'view' in command and any('statusCheckRollup' in part for part in command):
            return '[{"name":"ci","status":"COMPLETED","conclusion":"SUCCESS"}]'
          if 'view' in command and any('mergeStateStatus' in part for part in command):
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
          ['git', 'push', '--delete', 'git@github.com:testuser/pipelines.git', '1.3.0-update-version-tags'],
          runner.commands,
      )
      self.assertIn(
          ['git', 'push', '--set-upstream', 'git@github.com:testuser/pipelines.git', '1.3.0-update-version-tags'],
          runner.commands,
      )

  def test_step_merge_version_pr_uses_existing_remote_tag_when_requested(self):
    with TemporaryDirectory() as tmpdir:
      class ExistingRemoteTagRunner:

        dry_run = False

        def __init__(self):
          self.commands = []

        def run(self, command, cwd=None, check=True):
          self.commands.append(command)
          if command[:4] == ['git', 'ls-remote', '--exit-code', '--heads']:
            return subprocess.CompletedProcess(command, 1, '', '')
          if command[:4] == ['git', 'ls-remote', '--exit-code', '--tags']:
            return subprocess.CompletedProcess(command, 0, '', '')
          return subprocess.CompletedProcess(command, 0, '', '')

        def capture(self, command, cwd=None):
          self.commands.append(command)
          if 'view' in command and any('statusCheckRollup' in part for part in command):
            return '[{"name":"ci","status":"COMPLETED","conclusion":"SUCCESS"}]'
          if 'view' in command and any('mergeStateStatus' in part for part in command):
            return 'MERGED CLEAN'
          return 'https://github.com/kubeflow/pipelines/pull/1'

      runner = ExistingRemoteTagRunner()
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

      self.assertNotIn(['git', 'push', 'git@github.com:testuser/pipelines.git', '1.3.0'], runner.commands)
      self.assertNotIn(['git', 'push', '--delete', 'git@github.com:testuser/pipelines.git', '1.3.0'], runner.commands)

  def test_step_merge_version_pr_recreates_existing_remote_tag_when_requested(self):
    with TemporaryDirectory() as tmpdir:
      class ExistingRemoteTagRunner:

        dry_run = False

        def __init__(self):
          self.commands = []

        def run(self, command, cwd=None, check=True):
          self.commands.append(command)
          if command[:4] == ['git', 'ls-remote', '--exit-code', '--heads']:
            return subprocess.CompletedProcess(command, 1, '', '')
          if command[:4] == ['git', 'ls-remote', '--exit-code', '--tags']:
            return subprocess.CompletedProcess(command, 0, '', '')
          return subprocess.CompletedProcess(command, 0, '', '')

        def capture(self, command, cwd=None):
          self.commands.append(command)
          if 'view' in command and any('statusCheckRollup' in part for part in command):
            return '[{"name":"ci","status":"COMPLETED","conclusion":"SUCCESS"}]'
          if 'view' in command and any('mergeStateStatus' in part for part in command):
            return 'MERGED CLEAN'
          return 'https://github.com/kubeflow/pipelines/pull/1'

      runner = ExistingRemoteTagRunner()
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

      self.assertIn(['git', 'push', '--delete', 'git@github.com:testuser/pipelines.git', '1.3.0'], runner.commands)
      self.assertIn(['git', 'push', 'git@github.com:testuser/pipelines.git', '1.3.0'], runner.commands)


class StepSdkVersionFilesTest(unittest.TestCase):

  def test_generate_sdk_release_notes_from_sdk_conventional_commits(self):
    with TemporaryDirectory() as tmpdir:
      root = Path(tmpdir)

      def fake_check_output(command, cwd=None, text=True):
        self.assertEqual(cwd, root)
        if command[:3] == ['git', 'tag', '--list']:
          return 'sdk-1.2.3\nsdk-1.2.2\n'
        if command[:2] == ['git', 'log']:
          self.assertEqual(command[2], 'sdk-1.2.3..HEAD')
          return (
              'feat(sdk): add local execution support (#123)\n'
              'fix(backend): ignore non-SDK change (#124)\n'
              'fix(backend/sdk): repair compiler output (#125)\n'
              'chore(CI/SDK): update sdk test image (#126)\n'
              'feat(frontend): ignore frontend change (#127)\n'
              'feat(sdk)!: remove old runtime (#128)\n'
          )
        raise AssertionError(f'unexpected command: {command}')

      with mock.patch('subprocess.check_output', side_effect=fake_check_output):
        notes = steps._generate_sdk_release_notes(root)

      self.assertEqual(
          notes,
          '## Features\n\n'
          '* add local execution support (#123)\n\n'
          '## Breaking changes\n\n'
          '* remove old runtime (#128)\n\n'
          '## Deprecations\n\n'
          '## Bug fixes and other changes\n\n'
          '* repair compiler output (#125)\n'
          '* update sdk test image (#126)',
      )

  def test_generate_backend_release_notes_from_non_sdk_conventional_commits(self):
    with TemporaryDirectory() as tmpdir:
      root = Path(tmpdir)

      def fake_check_output(command, cwd=None, text=True):
        self.assertEqual(cwd, root)
        if command[:3] == ['git', 'tag', '--list']:
          return '1.2.3\n1.2.2\n'
        if command[:2] == ['git', 'log']:
          self.assertEqual(command[2], '1.2.3..HEAD')
          return (
              'feat(backend): add recurring run support (#123)\n'
              'fix(api): repair run filter parsing (#124)\n'
              'fix(sdk): ignore SDK-only change (#125)\n'
              'chore(frontend): update table copy (#126)\n'
              'feat(backend)!: remove old endpoint (#127)\n'
          )
        raise AssertionError(f'unexpected command: {command}')

      with mock.patch('subprocess.check_output', side_effect=fake_check_output):
        last_release, notes = steps._generate_backend_release_notes(root)

      self.assertEqual(last_release, '1.2.3')
      self.assertEqual(
          notes,
          '## Features\n\n'
          '* add recurring run support (#123)\n\n'
          '## Breaking changes\n\n'
          '* remove old endpoint (#127)\n\n'
          '## Deprecations\n\n'
          '## Bug fixes and other changes\n\n'
          '* repair run filter parsing (#124)\n'
          '* update table copy (#126)',
      )

  def test_update_sdk_version_files_replaces_version_without_trailing_garbage(self):
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

      steps._update_sdk_version_files(context)

      self.assertEqual(version_file.read_text(), "__version__ = '1.2.4'\n")

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
              'kfp-pipeline-spec>=1.2.3,<3\n'
              'kfp-server-api>=1.2.3,<3\n'
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
              "html_theme_options = {\n"
              "    'version_info': [\n"
              "        {\n"
              "            'version':\n"
              "                'https://kfp-kubernetes.readthedocs.io/en/kfp-kubernetes-1.2/',\n"
              "            'title':\n"
              "                '1.2',\n"
              "            'aliases': ['stable'],\n"
              "        },\n"
              "    ],\n"
              "}\n"
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
          if command[:4] == ['git', 'rev-parse', '--verify', 'refs/heads/1.3.0-update-version-tags']:
            return subprocess.CompletedProcess(command, 1, '', '')
          if command[:4] == ['git', 'rev-parse', '--verify', 'refs/tags/1.3.0']:
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

      steps.step_update_version_tags(context)

      self.assertEqual((root / 'sdk/python/kfp/version.py').read_text(), "__version__ = '1.3.0'\n")
      self.assertEqual((root / 'kubernetes_platform/python/kfp/kubernetes/__init__.py').read_text(), "__version__ = '1.3.0'\n")
      self.assertIn("VERSION = '1.3.0'", (root / 'api/v2alpha1/python/setup.py').read_text())
      self.assertIn('VERSION = "1.3.0"', (root / 'backend/api/v2beta1/python_http_client/setup.py').read_text())
      self.assertIn('__version__ = "1.3.0"', (root / 'backend/api/v2beta1/python_http_client/kfp_server_api/__init__.py').read_text())
      self.assertIn('kfp-pipeline-spec>=1.3.0,<3', (root / 'sdk/python/requirements.in').read_text())
      self.assertIn('kfp-server-api>=1.3.0,<3', (root / 'sdk/python/requirements.in').read_text())
      self.assertEqual((root / 'kubernetes_platform/python/requirements.in').read_text(), 'kfp>=1.3.0,<3\n')
      self.assertIn('"title": "1.3.0"', (root / 'docs/sdk/versions.json').read_text())
      self.assertIn('"aliases": []', (root / 'docs/sdk/versions.json').read_text())
      self.assertIn("'title':\n                '1.3'", (root / 'kubernetes_platform/python/docs/conf.py').read_text())
      self.assertIn("'aliases': []", (root / 'kubernetes_platform/python/docs/conf.py').read_text())
      self.assertIn('# 1.3.0', (root / 'sdk/RELEASE.md').read_text())
      self.assertIn('* New SDK feature.', (root / 'sdk/RELEASE.md').read_text())
      commands = [command for command, _, _ in context.runner.commands]
      self.assertLess(
          commands.index(['python', '-m', 'build', '.']),
          next(index for index, command in enumerate(commands) if command[:2] == ['bash', '-c']),
      )
      self.assertIn(['git', 'add', '--all'], commands)


class DryRunOutputTest(unittest.TestCase):

  def test_update_version_tags_cuts_update_branch_from_release_branch(self):
    class MissingBranchRunner(core.CommandRunner):

      def run(self, command, cwd=None, check=True):
        self.commands.append(command)
        if command[:4] == ['git', 'rev-parse', '--verify', 'refs/heads/3.2.0-update-version-tags']:
          return subprocess.CompletedProcess(command, 1, '', '')
        if command[:4] == ['git', 'rev-parse', '--verify', 'refs/tags/3.2.0']:
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

      steps.step_update_version_tags(context)

      self.assertEqual(context.runner.commands[0], ['git', 'checkout', 'release-3.2'])
      self.assertIn(['git', 'pull', '--ff-only'], context.runner.commands)
      self.assertIn(['git', 'checkout', '-b', '3.2.0-update-version-tags'], context.runner.commands)

  def test_update_version_tags_skips_pull_without_release_branch_upstream(self):
    class NoUpstreamRunner(core.CommandRunner):

      def run(self, command, cwd=None, check=True):
        self.commands.append(command)
        if command == ['git', 'rev-parse', '--abbrev-ref', '--symbolic-full-name', '@{u}']:
          return subprocess.CompletedProcess(command, 1, '', '')
        if command[:4] == ['git', 'rev-parse', '--verify', 'refs/heads/3.2.0-update-version-tags']:
          return subprocess.CompletedProcess(command, 1, '', '')
        if command[:4] == ['git', 'rev-parse', '--verify', 'refs/tags/3.2.0']:
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

      steps.step_update_version_tags(context)

      self.assertIn(['git', 'checkout', 'release-3.2'], context.runner.commands)
      self.assertNotIn(['git', 'pull', '--ff-only'], context.runner.commands)
      self.assertIn(['git', 'checkout', '-b', '3.2.0-update-version-tags'], context.runner.commands)

  def test_update_version_tags_recreates_existing_update_branch_when_requested(self):
    class ExistingBranchRunner(core.CommandRunner):

      def run(self, command, cwd=None, check=True):
        self.commands.append(command)
        if command[:4] == ['git', 'rev-parse', '--verify', 'refs/heads/3.2.0-update-version-tags']:
          return subprocess.CompletedProcess(command, 0, '', '')
        if command[:4] == ['git', 'rev-parse', '--verify', 'refs/tags/3.2.0']:
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
        steps.step_update_version_tags(context)

      self.assertIn(['git', 'branch', '-D', '3.2.0-update-version-tags'], context.runner.commands)
      self.assertIn(['git', 'checkout', '-b', '3.2.0-update-version-tags'], context.runner.commands)

  def test_update_version_tags_uses_existing_update_branch_when_requested(self):
    class ExistingBranchRunner(core.CommandRunner):

      def run(self, command, cwd=None, check=True):
        self.commands.append(command)
        if command[:4] == ['git', 'rev-parse', '--verify', 'refs/heads/3.2.0-update-version-tags']:
          return subprocess.CompletedProcess(command, 0, '', '')
        if command[:4] == ['git', 'rev-parse', '--verify', 'refs/tags/3.2.0']:
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
        steps.step_update_version_tags(context)

      self.assertIn(['git', 'checkout', '3.2.0-update-version-tags'], context.runner.commands)
      self.assertNotIn(['git', 'branch', '-D', '3.2.0-update-version-tags'], context.runner.commands)
      self.assertNotIn(['git', 'checkout', '-b', '3.2.0-update-version-tags'], context.runner.commands)

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

  def test_update_version_tags_does_not_edit_release_dockerfile_image(self):
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
          any('release/Dockerfile.release' in command for command in flattened_commands),
          flattened_commands,
      )
      self.assertFalse(
          any('test/release/Dockerfile.release' in command for command in flattened_commands),
          flattened_commands,
      )

  def test_update_version_tags_keeps_tool_images_on_master(self):
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
          any('PREBUILT_REMOTE_IMAGE=ghcr.io/kubeflow/kfp-api-generator:' in command for command in flattened_commands),
          flattened_commands,
      )
      self.assertFalse(
          any('RELEASE_IMAGE=ghcr.io/kubeflow/kfp-release:' in command for command in flattened_commands),
          flattened_commands,
      )
      self.assertFalse(
          any('ARG BASE_IMAGE=ghcr.io/kubeflow/kfp-api-generator:' in command for command in flattened_commands),
          flattened_commands,
      )

  def test_update_version_tags_recreates_existing_tag_when_requested(self):
    class ExistingTagRunner(core.CommandRunner):

      def run(self, command, cwd=None, check=True):
        self.commands.append(command)
        if command[:4] == ['git', 'rev-parse', '--verify', 'refs/heads/3.2.0-update-version-tags']:
          return subprocess.CompletedProcess(command, 1, '', '')
        if command[:4] == ['git', 'rev-parse', '--verify', 'refs/tags/3.2.0']:
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

      with mock.patch('builtins.input', return_value='r'):
        steps.step_update_version_tags(context)

      self.assertIn(['git', 'tag', '-d', '3.2.0'], context.runner.commands)
      self.assertIn(
          ['git', 'tag', '-a', '3.2.0', '-m', 'Kubeflow Pipelines 3.2.0 release'],
          context.runner.commands,
      )

  def test_update_version_tags_uses_existing_tag_when_requested(self):
    class ExistingTagRunner(core.CommandRunner):

      def run(self, command, cwd=None, check=True):
        self.commands.append(command)
        if command[:4] == ['git', 'rev-parse', '--verify', 'refs/heads/3.2.0-update-version-tags']:
          return subprocess.CompletedProcess(command, 1, '', '')
        if command[:4] == ['git', 'rev-parse', '--verify', 'refs/tags/3.2.0']:
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
        steps.step_update_version_tags(context)

      self.assertNotIn(['git', 'tag', '-d', '3.2.0'], context.runner.commands)
      self.assertNotIn(
          ['git', 'tag', '-a', '3.2.0', '-m', 'Kubeflow Pipelines 3.2.0 release'],
          context.runner.commands,
      )

  def test_update_version_tags_uses_master_release_image_for_minor_release(self):
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
          any('ghcr.io/kubeflow/kfp-release:master' in command for command in flattened_commands),
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

      with mock.patch.dict('os.environ', {'RTD_TOKEN': 'env-token'}), mock.patch('getpass.getpass') as getpass_mock, mock.patch.object(steps.rtd, 'ReadTheDocsClient') as client_cls, mock.patch.object(steps.rtd, 'update_release_docs') as update_docs:
        steps.step_confirm_rtd(context)

      getpass_mock.assert_not_called()
      client_cls.assert_called_once_with('env-token')
      update_docs.assert_called_once_with(client_cls.return_value, '3.2.0', 'release-3.2')

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

  def test_kfp_kubernetes_docs_branch_reuses_existing_remote_branch(self):
    class ExistingBranchRunner(core.CommandRunner):

      def run(self, command, cwd=None, check=True):
        self.commands.append(command)
        if command[:4] == ['git', 'ls-remote', '--exit-code', '--heads']:
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

      with mock.patch('builtins.input', return_value='reuse'):
        steps.step_create_kfp_kubernetes_docs_branch(context)

      self.assertEqual(context.runner.commands, [
          ['git', 'ls-remote', '--exit-code', '--heads', 'upstream', 'kfp-kubernetes-3.2'],
      ])

  def test_kfp_kubernetes_docs_branch_can_replace_existing_remote_branch(self):
    class ExistingBranchRunner(core.CommandRunner):

      def run(self, command, cwd=None, check=True):
        self.commands.append(command)
        if command[:4] == ['git', 'ls-remote', '--exit-code', '--heads']:
          return subprocess.CompletedProcess(command, 0, '', '')
        return subprocess.CompletedProcess(command, 0, '', '')

    with TemporaryDirectory() as tmpdir:
      root = Path(tmpdir)
      (root / 'kubernetes_platform/python/docs').mkdir(parents=True)
      (root / 'kubernetes_platform/python/docs/.readthedocs.yml').write_text('version: 2\n')
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

      with mock.patch('builtins.input', return_value='replace'):
        steps.step_create_kfp_kubernetes_docs_branch(context)

      self.assertIn(['git', 'push', 'upstream', ':kfp-kubernetes-3.2'], context.runner.commands)
      self.assertIn(['git', 'checkout', '-B', 'kfp-kubernetes-3.2'], context.runner.commands)
      self.assertIn(['git', 'push', '--set-upstream', 'upstream', 'kfp-kubernetes-3.2'], context.runner.commands)

  def test_kfp_kubernetes_docs_branch_can_force_push_existing_remote_branch(self):
    class ExistingBranchRunner(core.CommandRunner):

      def run(self, command, cwd=None, check=True):
        self.commands.append(command)
        if command[:4] == ['git', 'ls-remote', '--exit-code', '--heads']:
          return subprocess.CompletedProcess(command, 0, '', '')
        return subprocess.CompletedProcess(command, 0, '', '')

    with TemporaryDirectory() as tmpdir:
      root = Path(tmpdir)
      (root / 'kubernetes_platform/python/docs').mkdir(parents=True)
      (root / 'kubernetes_platform/python/docs/.readthedocs.yml').write_text('version: 2\n')
      context = core.ReleaseContext(
          root=root,
          state=core.ReleaseState(root / 'state.json'),
          runner=ExistingBranchRunner(dry_run=False),
          metadata=core.ReleaseMetadata.from_version('minor', '3.2.0'),
          fork_remote='git@github.com:testuser/pipelines.git',
          include_backend=True,
          include_sdk=True,
      )

      with mock.patch('builtins.input', return_value='force-push'):
        steps.step_create_kfp_kubernetes_docs_branch(context)

      self.assertIn(['git', 'checkout', '-B', 'kfp-kubernetes-3.2'], context.runner.commands)
      self.assertIn(['git', 'push', '--force-with-lease', '--set-upstream', 'upstream', 'kfp-kubernetes-3.2'], context.runner.commands)

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
