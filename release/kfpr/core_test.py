#!/usr/bin/env python3
"""Tests for release core helpers."""

import unittest
from unittest import mock
import contextlib
import io
from pathlib import Path
from tempfile import TemporaryDirectory

from kfpr import core
from kfpr.core import ReleaseMetadata, ReleaseState


class TestReleaseMetadata(unittest.TestCase):

  def test_from_version_major_release(self):
    metadata = ReleaseMetadata.from_version('major', '2.0.0')
    self.assertEqual(metadata.release_type, 'major')
    self.assertEqual(metadata.version, '2.0.0')
    self.assertEqual(metadata.major, 2)
    self.assertEqual(metadata.minor, 0)
    self.assertEqual(metadata.patch, 0)
    self.assertEqual(metadata.tag, '2.0.0')
    self.assertEqual(metadata.sdk_tag, 'sdk-2.0.0')
    self.assertEqual(metadata.release_branch, 'release-2.0')
    self.assertEqual(metadata.patch_branch, None)

  def test_from_version_minor_release(self):
    metadata = ReleaseMetadata.from_version('minor', '1.3.0')
    self.assertEqual(metadata.release_type, 'minor')
    self.assertEqual(metadata.version, '1.3.0')
    self.assertEqual(metadata.major, 1)
    self.assertEqual(metadata.minor, 3)
    self.assertEqual(metadata.patch, 0)
    self.assertEqual(metadata.tag, '1.3.0')
    self.assertEqual(metadata.sdk_tag, 'sdk-1.3.0')
    self.assertEqual(metadata.release_branch, 'release-1.3')
    self.assertEqual(metadata.patch_branch, None)

  def test_from_version_patch_release(self):
    metadata = ReleaseMetadata.from_version('patch', '1.3.2')
    self.assertEqual(metadata.release_type, 'patch')
    self.assertEqual(metadata.version, '1.3.2')
    self.assertEqual(metadata.major, 1)
    self.assertEqual(metadata.minor, 3)
    self.assertEqual(metadata.patch, 2)
    self.assertEqual(metadata.tag, '1.3.2')
    self.assertEqual(metadata.sdk_tag, 'sdk-1.3.2')
    self.assertEqual(metadata.release_branch, 'release-1.3')
    self.assertEqual(metadata.patch_branch, 'release-1.3.2')

  def test_from_version_invalid_release_type(self):
    with self.assertRaises(ValueError):
      ReleaseMetadata.from_version('invalid', '1.0.0')

  def test_from_version_invalid_semver_format(self):
    with self.assertRaises(ValueError):
      ReleaseMetadata.from_version('major', '1.0')

  def test_from_version_major_not_x_0_0(self):
    with self.assertRaises(ValueError):
      ReleaseMetadata.from_version('major', '1.1.0')

  def test_from_version_minor_not_x_y_0(self):
    with self.assertRaises(ValueError):
      ReleaseMetadata.from_version('minor', '1.3.1')


class TestReleaseState(unittest.TestCase):

  def test_load_nonexistent_path_returns_fresh_state(self):
    with TemporaryDirectory() as tmpdir:
      state_file = Path(tmpdir) / 'nonexistent' / 'state.json'
      loaded_state = ReleaseState.load(state_file)

      self.assertEqual(loaded_state.path, state_file)
      self.assertEqual(loaded_state.answers, {})
      self.assertEqual(loaded_state.completed_steps, [])

  def test_load_malformed_json_raises_value_error(self):
    with TemporaryDirectory() as tmpdir:
      state_file = Path(tmpdir) / 'state.json'
      state_file.write_text('{ invalid json }')

      with self.assertRaises(ValueError) as cm:
        ReleaseState.load(state_file)

      error_msg = str(cm.exception)
      self.assertIn('Invalid state file', error_msg)
      self.assertIn(str(state_file), error_msg)
      self.assertIn('malformed JSON', error_msg)
      self.assertIn('fix or delete', error_msg)

  def test_load_save_roundtrip(self):
    with TemporaryDirectory() as tmpdir:
      state_file = Path(tmpdir) / 'state.json'
      state = ReleaseState(state_file)
      state.answers['version'] = '1.0.0'
      state.save()

      loaded_state = ReleaseState.load(state_file)
      self.assertEqual(loaded_state.answers['version'], '1.0.0')

  def test_save_uses_tmp_file(self):
    with TemporaryDirectory() as tmpdir:
      state_file = Path(tmpdir) / 'state.json'
      state = ReleaseState(state_file)
      state.answers['version'] = '1.0.0'
      state.save()

      self.assertTrue(state_file.exists())
      tmp_file = state_file.with_suffix(state_file.suffix + '.tmp')
      self.assertFalse(tmp_file.exists())

  def test_mark_done_and_is_done(self):
    with TemporaryDirectory() as tmpdir:
      state_file = Path(tmpdir) / 'state.json'
      state = ReleaseState(state_file)

      self.assertFalse(state.is_done('step1'))
      state.mark_done('step1')
      self.assertTrue(state.is_done('step1'))
      state.save()

      loaded_state = ReleaseState.load(state_file)
      self.assertTrue(loaded_state.is_done('step1'))

  def test_multiple_steps(self):
    with TemporaryDirectory() as tmpdir:
      state_file = Path(tmpdir) / 'state.json'
      state = ReleaseState(state_file)

      state.mark_done('step1')
      state.mark_done('step2')
      state.save()

      loaded_state = ReleaseState.load(state_file)
      self.assertTrue(loaded_state.is_done('step1'))
      self.assertTrue(loaded_state.is_done('step2'))
      self.assertFalse(loaded_state.is_done('step3'))


class CommandRunnerTest(unittest.TestCase):

  def test_dry_run_records_command_without_running(self):
    runner = core.CommandRunner(dry_run=True)
    runner.run(['git', 'status'])
    self.assertEqual(runner.commands, [['git', 'status']])


class GithubCommandTest(unittest.TestCase):

  def test_image_workflow_command(self):
    metadata = core.ReleaseMetadata.from_version('patch', '3.2.1')
    command = core.image_workflow_command(metadata)
    self.assertEqual(command[:4], ['gh', 'workflow', 'run', 'image-builds-release.yml'])
    self.assertIn('src_branch=release-3.2', command)
    self.assertIn('target_tag=3.2.1', command)

  def test_sdk_workflow_command(self):
    metadata = core.ReleaseMetadata.from_version('minor', '3.2.0')
    command = core.sdk_workflow_command(metadata)
    self.assertEqual(command[:4], ['gh', 'workflow', 'run', 'publish-packages.yml'])
    self.assertIn('tag=sdk-3.2.0', command)
    self.assertIn('packages=all', command)

  def test_wait_for_pr_merge_dry_run_returns_immediately(self):
    runner = core.CommandRunner(dry_run=True)
    core.wait_for_pr_merge(runner, 'https://github.com/kubeflow/pipelines/pull/1')
    self.assertEqual(len(runner.commands), 0)

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
      core.watch_pr_ci(runner, 'https://github.com/kubeflow/pipelines/pull/1')

    self.assertIn('unit-tests failed', str(cm.exception))

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

  def test_watch_pr_ci_fails_fast_on_failed_status_context(self):
    class FailingStatusRunner:

      dry_run = False

      def capture(self, command, cwd=None):
        return '[{"context":"license/cla","state":"FAILURE"}]'

    with mock.patch('time.sleep', side_effect=AssertionError('slept instead of failing')):
      with self.assertRaises(RuntimeError) as cm:
        core.watch_pr_ci(FailingStatusRunner(), 'https://github.com/kubeflow/pipelines/pull/1')

    self.assertIn('license/cla failed', str(cm.exception))

  def test_watch_pr_ci_returns_when_status_contexts_succeed(self):
    class PassingStatusRunner:

      dry_run = False

      def capture(self, command, cwd=None):
        return '[{"context":"license/cla","state":"SUCCESS"}]'

    with mock.patch('time.sleep', side_effect=AssertionError('slept instead of returning')):
      core.watch_pr_ci(PassingStatusRunner(), 'https://github.com/kubeflow/pipelines/pull/1')

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
        return '' if self.captures == 1 else '12345\thttps://github.com/kubeflow/pipelines/actions/runs/12345'

      def run(self, command, cwd=None, check=True):
        self.commands.append(command)

    runner = DelayedRunRunner()
    with mock.patch('time.sleep'):
      core.watch_latest_workflow_run(runner, 'test-workflow.yml', 'main')

    self.assertEqual(runner.commands[-1], ['gh', 'run', 'watch', '12345'])
    self.assertNotIn(['gh', 'run', 'watch', ''], runner.commands)

  def test_watch_latest_workflow_run_prints_run_url(self):
    class RunUrlRunner:

      dry_run = False

      def __init__(self):
        self.commands = []

      def capture(self, command, cwd=None):
        self.commands.append(command)
        return '12345\thttps://github.com/kubeflow/pipelines/actions/runs/12345'

      def run(self, command, cwd=None, check=True):
        self.commands.append(command)

    runner = RunUrlRunner()
    with mock.patch('builtins.print') as print_mock:
      core.watch_latest_workflow_run(runner, 'test-workflow.yml', 'main')

    output = '\n'.join(str(call.args[0]) for call in print_mock.call_args_list)
    self.assertIn('Workflow run: https://github.com/kubeflow/pipelines/actions/runs/12345', output)
    self.assertEqual(runner.commands[-1], ['gh', 'run', 'watch', '12345'])


class InlineCommandTest(unittest.TestCase):

  def test_kfp_requirements_command(self):
    command = core.kfp_requirements_command()
    self.assertEqual(command[:2], ['bash', '-c'])
    self.assertIn('pip-compile', command[2])
    self.assertIn('$GIT_ROOT/api/v2alpha1/python/dist', command[2])
    self.assertIn('$GIT_ROOT/backend/api/v2beta1/python_http_client/dist', command[2])
    self.assertIn('> requirements.txt', command[2])

  def test_kubernetes_requirements_command(self):
    command = core.kubernetes_requirements_command()
    self.assertEqual(command[:2], ['bash', '-c'])
    self.assertIn('pip-compile', command[2])
    self.assertIn('$GIT_ROOT/sdk/python/dist', command[2])
    self.assertIn('$GIT_ROOT/api/v2alpha1/python/dist', command[2])
    self.assertIn('$GIT_ROOT/backend/api/v2beta1/python_http_client/dist', command[2])
    self.assertIn('> requirements.txt', command[2])

  def test_release_version_bump_command_runs_container_directly(self):
    command = core.release_version_bump_command(Path('/repo'), 'release-3.2')
    self.assertEqual(command[:3], ['docker', 'run', '--rm'])
    self.assertIn('ghcr.io/kubeflow/kfp-release:release-3.2', command)
    self.assertIn('/bin/bash', command)
    self.assertIn('-c', command)
    self.assertNotIn('-lc', command)
    script = command[-1]
    self.assertIn('git-cliff -c cliff.toml --unreleased --tag "$TAG_NAME" --prepend CHANGELOG.md', script)
    self.assertIn('"$REPO_ROOT/manifests/gcp_marketplace/hack/release.sh" "$TAG_NAME"', script)
    self.assertIn('"$REPO_ROOT/backend/api/build_kfp_server_api_python_package.sh"', script)
    self.assertIn('REQUIRED_NODE_VERSION=', script)
    self.assertIn('which node >/dev/null', script)
    self.assertIn('yq -V | grep 3.', script)
    self.assertIn('python3 -c "import setuptools"', script)
    self.assertIn('go env GOPATH', script)
    self.assertNotIn('check-release-needed-tools.sh', script)
    self.assertNotIn('make release-in-place', command)

  def test_release_version_bump_command_can_use_master_image(self):
    command = core.release_version_bump_command(Path('/repo'), 'release-3.2', 'master')

    self.assertIn('ghcr.io/kubeflow/kfp-release:master', command)
    self.assertNotIn('ghcr.io/kubeflow/kfp-release:release-3.2', command)

  def test_kfp_kubernetes_docs_build_command_runs_container_directly(self):
    command = core.kfp_kubernetes_docs_build_command(Path('/repo'), 'release-3.2')
    self.assertEqual(command[:3], ['docker', 'run', '--rm'])
    self.assertIn('ghcr.io/kubeflow/kfp-api-generator:release-3.2', command)
    script = command[-1]
    self.assertIn('wget -qO api/v2alpha1/google/rpc/status.proto', script)
    self.assertIn('python3 generate_proto.py', script)
    self.assertIn('python3 setup.py sdist', script)
    self.assertNotIn('make', command)


class ParseGithubOwnerTest(unittest.TestCase):

  def test_normalize_fork_remote_expands_bare_username(self):
    self.assertEqual(
        core.normalize_fork_remote('droctothorpe'),
        'https://github.com/droctothorpe/pipelines.git',
    )

  def test_normalize_fork_remote_keeps_ssh_remote(self):
    self.assertEqual(
        core.normalize_fork_remote('git@github.com:user123/pipelines.git'),
        'git@github.com:user123/pipelines.git',
    )

  def test_normalize_fork_remote_keeps_https_remote(self):
    self.assertEqual(
        core.normalize_fork_remote('https://github.com/user123/pipelines.git'),
        'https://github.com/user123/pipelines.git',
    )

  def test_normalize_fork_remote_rejects_at_username(self):
    with self.assertRaises(ValueError):
      core.normalize_fork_remote('@droctothorpe')

  def test_parse_ssh_format(self):
    self.assertEqual(
        core.parse_github_owner('git@github.com:user123/pipelines.git'),
        'user123',
    )

  def test_parse_https_format(self):
    self.assertEqual(
        core.parse_github_owner('https://github.com/user456/pipelines.git'),
        'user456',
    )

  def test_parse_invalid_format_raises_value_error(self):
    with self.assertRaises(ValueError) as cm:
      core.parse_github_owner('invalid-remote-url')

    error_msg = str(cm.exception)
    self.assertIn('Cannot parse GitHub owner', error_msg)
    self.assertIn('invalid-remote-url', error_msg)
    self.assertIn('Expected format', error_msg)


class PromptValidationTest(unittest.TestCase):

  def test_prompt_numbered_choice_accepts_number(self):
    with contextlib.redirect_stdout(io.StringIO()) as output, mock.patch(
        'builtins.input',
        return_value='2',
    ):
      answer = core.prompt_numbered_choice('Release type', ['major', 'minor', 'patch'])

    self.assertEqual(answer, 'minor')
    self.assertIn('1) major', output.getvalue())
    self.assertIn('2) minor', output.getvalue())
    self.assertIn('3) patch', output.getvalue())

  def test_prompt_numbered_choice_reasks_for_invalid_number(self):
    with contextlib.redirect_stdout(io.StringIO()) as output, mock.patch(
        'builtins.input',
        side_effect=['4', '3'],
    ):
      answer = core.prompt_numbered_choice('Release type', ['major', 'minor', 'patch'])

    self.assertEqual(answer, 'patch')
    self.assertIn('Choose a number from 1 to 3.', output.getvalue())

  def test_collect_context_reasks_for_invalid_version_and_fork_remote(self):
    with TemporaryDirectory() as tmpdir:
      state = ReleaseState(Path(tmpdir) / 'state.json')
      args = type('Args', (), {'dry_run': True})()

      with contextlib.redirect_stdout(io.StringIO()), mock.patch(
          'builtins.input',
          side_effect=[
              '2',
              '3.2',
              '3.2.0',
              '@droctothorpe',
              'droctothorpe',
          ],
      ):
        context = core.collect_context(args, state)

      self.assertEqual(context.metadata.tag, '3.2.0')
      self.assertEqual(context.fork_remote, 'https://github.com/droctothorpe/pipelines.git')

  def test_collect_context_prompts_for_major_minor_source_branch(self):
    with TemporaryDirectory() as tmpdir:
      state = ReleaseState(Path(tmpdir) / 'state.json')
      args = type('Args', (), {'dry_run': True, 'prompt_release_source_branch': True})()

      with contextlib.redirect_stdout(io.StringIO()), mock.patch(
          'builtins.input',
          side_effect=['2', '3.2.0', 'droctothorpe', 'release-candidate'],
      ):
        core.collect_context(args, state)

      self.assertEqual(state.answers['release_source_branch'], 'release-candidate')

  def test_collect_context_uses_release_source_branch_arg_without_prompt(self):
    with TemporaryDirectory() as tmpdir:
      state = ReleaseState(Path(tmpdir) / 'state.json')
      args = type(
          'Args',
          (),
          {
              'dry_run': True,
              'prompt_release_source_branch': True,
              'release_source_branch': 'release-candidate',
          },
      )()

      with contextlib.redirect_stdout(io.StringIO()), mock.patch(
          'builtins.input',
          side_effect=['2', '3.2.0', 'droctothorpe'],
      ):
        core.collect_context(args, state)

      self.assertEqual(state.answers['release_source_branch'], 'release-candidate')

  def test_collect_context_defaults_major_minor_source_branch_to_master(self):
    with TemporaryDirectory() as tmpdir:
      state = ReleaseState(Path(tmpdir) / 'state.json')
      args = type('Args', (), {'dry_run': True, 'prompt_release_source_branch': True})()

      with contextlib.redirect_stdout(io.StringIO()), mock.patch(
          'builtins.input',
          side_effect=['2', '3.2.0', 'droctothorpe', ''],
      ):
        core.collect_context(args, state)

      self.assertEqual(state.answers['release_source_branch'], 'master')


if __name__ == '__main__':
  unittest.main()
