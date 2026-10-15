#!/usr/bin/env python3
# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Execute the composite action with local download outcomes and files."""

import os
from pathlib import Path
import re
import subprocess
import tempfile
import unittest
from unittest import mock

from artifact_workflow_test_support import evaluate
from artifact_workflow_test_support import render
import yaml

REPOSITORY_ROOT = Path(__file__).resolve().parents[3]
ACTION_DIRECTORY = (
    REPOSITORY_ROOT / '.github/actions/download-artifact-with-retry')
ARTIFACT_FILES = REPOSITORY_ROOT / '.github/resources/scripts/artifact-files.sh'
ACTION_FILE = Path(
    os.environ.get('ARTIFACT_RETRY_ACTION', ACTION_DIRECTORY / 'action.yml'))


class CompositeAction:
    """Local regression simulator for this action, not a GitHub Actions runner.

    Execute the checked-in bash and YAML wiring with mocked downloads.
    Expression handling covers this action's string references, boolean
    operators and status functions only; it does not implement general
    Actions coercion, scheduling or cancellation. Hosted CI must verify
    runner behavior. Keeping retry decisions in the loaded YAML catches
    wiring regressions without duplicating the retry algorithm in the
    tests.
    """

    def __init__(self, path, attempts, required_files=None):
        self.action = yaml.safe_load(ACTION_FILE.read_text())
        self.path = path
        self.attempts = attempts
        self.downloads = []
        self.download_output_path = None
        self.steps = {}
        self.failed = False
        self.cancelled = False
        self.log = ''
        self.github = {
            'action_path': str(ACTION_DIRECTORY),
            'repository': 'owner/repository',
            'run_id': '123',
        }
        self.inputs = {
            key: render(value.get('default', ''), {'github': self.github})
            for key, value in self.action['inputs'].items()
        }
        self.inputs.update({'path': str(path), 'retry-delay-seconds': '0'})
        if required_files is not None:
            self.inputs['required-files'] = required_files

    def expression_context(self):
        return {
            'inputs': self.inputs,
            'steps': self.steps,
            'github': self.github,
        }

    def status_functions(self):
        return {
            'always': lambda: True,
            'cancelled': lambda: self.cancelled,
            'success': lambda: not self.failed and not self.cancelled,
            'failure': lambda: self.failed,
        }

    def evaluate(self, expression):
        return evaluate(expression, self.expression_context(),
                        self.status_functions())

    def render(self, value):
        return render(value, self.expression_context(), self.status_functions())

    def run(self):
        for index, step in enumerate(self.action['runs']['steps']):
            step_id = step.get('id', f'step-{index}')
            condition = str(step.get('if', 'success()'))
            status_check = re.search(r'\b(always|cancelled|success|failure)\(',
                                     condition)
            enabled = (status_check or not self.failed and not self.cancelled)
            if not enabled or not self.evaluate(condition):
                self.steps[step_id] = {'outcome': 'skipped'}
                continue
            outputs = {}
            if 'uses' in step:
                if not step['uses'].startswith('actions/download-artifact@'):
                    raise AssertionError(f'Unexpected external action: {step}')
                # Each callback models only the external action's extraction.
                attempt = self.attempts[len(self.downloads)]
                self.downloads.append(step_id)
                download_inputs = {
                    key: self.render(value)
                    for key, value in step['with'].items()
                }
                destination = Path(download_inputs['path']).expanduser()
                if not destination.is_absolute():
                    destination = REPOSITORY_ROOT / destination
                destination.mkdir(parents=True, exist_ok=True)
                outcome = attempt(destination, download_inputs)
                if outcome == 'success':
                    outputs['download-path'] = (
                        self.download_output_path or
                        os.path.abspath(destination))
            else:
                if step['shell'] != 'bash':
                    raise AssertionError(f'Unexpected shell: {step}')
                environment = os.environ.copy()
                # Only the action's declared env can supply its helper path.
                environment.pop('GITHUB_ACTION_PATH', None)
                environment.pop('ACTION_PATH', None)
                environment.pop('ARTIFACT_FILES', None)
                environment.update({
                    key: self.render(value)
                    for key, value in step.get('env', {}).items()
                })
                with tempfile.TemporaryDirectory() as directory:
                    output = Path(directory) / 'github-output'
                    output.touch()
                    environment['GITHUB_OUTPUT'] = str(output)
                    result = subprocess.run(
                        [
                            'bash', '--noprofile', '--norc', '-e', '-o',
                            'pipefail', '-c',
                            self.render(step['run'])
                        ],
                        cwd=REPOSITORY_ROOT,
                        env=environment,
                        text=True,
                        capture_output=True,
                        check=False,
                    )
                    # This action emits only single-line key=value outputs.
                    outputs.update(
                        line.split('=', 1)
                        for line in output.read_text().splitlines())
                self.log += result.stdout + result.stderr
                outcome = 'success' if result.returncode == 0 else 'failure'
            self.steps[step_id] = {'outcome': outcome, 'outputs': outputs}
            if outcome == 'cancelled':
                self.cancelled = True
            elif outcome == 'failure' and not step.get('continue-on-error'):
                self.failed = True
        return ('cancelled'
                if self.cancelled else 'failure' if self.failed else 'success')


def download(outcome='success', files=None):

    def attempt(path, inputs=None):
        for name, content in (files or {}).items():
            target = path / name
            target.parent.mkdir(parents=True, exist_ok=True)
            target.write_text(content)
        return outcome

    return attempt


class ArtifactDownloadCompletenessTest(unittest.TestCase):

    def setUp(self):
        self.directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.directory.cleanup)
        self.path = Path(self.directory.name).resolve() / 'downloads'
        self.required = 'frontend/image.tar\nruntime-base-images/images.tar'
        self.complete = {
            'frontend/image.tar': 'frontend image',
            'runtime-base-images/images.tar': 'runtime images',
        }

    def run_action(self, attempts, required_files=None):
        runner = CompositeAction(self.path, attempts, required_files)
        result = runner.run()
        return runner, result

    def test_complete_first_attempt_does_not_retry(self):
        runner, result = self.run_action([download(files=self.complete)],
                                         self.required)
        self.assertEqual(result, 'success', runner.log)
        self.assertEqual(runner.downloads, ['primary'])
        output = runner.action['outputs']['download-path']['value']
        self.assertEqual(runner.render(output), str(self.path))

    def test_windows_native_action_output_validates_complete_download(self):
        tools = Path(self.directory.name) / 'bin'
        tools.mkdir()
        converter = tools / 'cygpath'
        converter.write_text('#!/bin/sh\n'
                             '[ "$1" = "-u" ] && [ "$2" = "--" ] && '
                             '[ "$3" = "$CYGPATH_INPUT" ] || exit 9\n'
                             'printf "%s\\n" "$CYGPATH_TARGET"\n')
        converter.chmod(0o755)
        runner = CompositeAction(self.path, [download(files=self.complete)],
                                 self.required)
        runner.download_output_path = r'D:\a\pipelines\downloads'
        with mock.patch.dict(
                os.environ, {
                    'PATH': str(tools) + os.pathsep + os.environ['PATH'],
                    'CYGPATH_TARGET': str(self.path),
                    'CYGPATH_INPUT': runner.download_output_path,
                }):
            result = runner.run()
        self.assertEqual(result, 'success', runner.log)
        self.assertEqual(runner.downloads, ['primary'])
        self.assertEqual(runner.steps['verify-primary']['outputs']['complete'],
                         'true')

    def test_download_defaults_resolve_the_github_context(self):
        received = []

        def attempt(path, inputs):
            received.append(inputs)
            return download(files=self.complete)(path, inputs)

        runner, result = self.run_action([attempt], self.required)
        self.assertEqual(result, 'success', runner.log)
        self.assertEqual(received[0]['repository'], 'owner/repository')
        self.assertEqual(received[0]['run-id'], '123')

    def test_rendered_download_inputs_preserve_literal_characters(self):
        self.path = Path(
            self.directory.name).resolve() / 'downloads with spaces'
        filename = 'archive with ! && || != and \'single\' "double" quotes.tar'
        received = []

        def attempt(path, inputs):
            received.append(inputs)
            if len(received) == 1:
                return 'failure'
            return download(files={filename: 'image'})(path, inputs)

        runner = CompositeAction(self.path, [attempt, attempt], filename)
        runner.inputs.update({
            'pattern': '!*.dockerbuild',
            'name': 'artifact ! && || != \'single\' "double" quotes',
            'github-token': 'test-token!&&||!=',
            'repository': 'owner/repository',
            'run-id': '123',
            'merge-multiple': 'true',
        })
        self.assertEqual(runner.run(), 'success', runner.log)
        expected = {
            key: runner.inputs[key]
            for key in ('name', 'path', 'pattern', 'merge-multiple',
                        'github-token', 'repository', 'run-id')
        }
        self.assertEqual(received, [expected, expected])
        self.assertTrue((self.path / filename).is_file())

    def test_expression_literals_are_not_operator_rewritten(self):
        runner = CompositeAction(self.path, [], self.required)
        literal = "! && || != 'quoted' inputs.pattern"
        expression = "${{ '! && || != ''quoted'' inputs.pattern' }}"
        self.assertEqual(runner.render(expression), literal)
        runner.inputs['pattern'] = literal
        self.assertTrue(
            runner.evaluate(
                "inputs.pattern == '! && || != ''quoted'' inputs.pattern' "
                "&& !cancelled() && inputs.pattern != 'other'"))

    def test_successful_but_incomplete_download_retries(self):
        runner, result = self.run_action([
            download(files={'frontend/image.tar': 'partial'}),
            download(files=self.complete),
        ], self.required)
        self.assertEqual(runner.downloads, ['primary', 'retry'], runner.log)
        self.assertEqual(result, 'success', runner.log)
        self.assertEqual(runner.steps['verify-primary']['outcome'], 'success')
        self.assertEqual(runner.steps['verify-primary']['outputs']['complete'],
                         'false')
        self.assertNotIn('::warning::', runner.log)
        self.assertNotIn('::error::', runner.log)

    def test_missing_file_after_both_attempts_fails(self):
        runner, result = self.run_action([download(), download()],
                                         self.required)
        self.assertEqual(result, 'failure', runner.log)
        self.assertEqual(runner.downloads, ['primary', 'retry'])
        self.assertIn('frontend/image.tar', runner.log)
        self.assertIn('runtime-base-images/images.tar', runner.log)
        self.assertIn('after 2 attempt', runner.log)

    def test_transport_failure_retries(self):
        runner, result = self.run_action([
            download('failure', {'frontend/image.tar': 'partial'}),
            download(files=self.complete),
        ], self.required)
        self.assertEqual(result, 'success', runner.log)
        self.assertEqual(runner.downloads, ['primary', 'retry'])

    def test_failed_retry_with_complete_residue_cannot_pass(self):
        runner, result = self.run_action([
            download(files={'frontend/image.tar': 'partial'}),
            download('failure', self.complete),
        ], self.required)
        self.assertEqual(result, 'failure', runner.log)
        self.assertEqual(runner.downloads, ['primary', 'retry'])

    def test_retry_cannot_combine_partial_attempts(self):
        runner, result = self.run_action([
            download(files={'frontend/image.tar': 'old partial'}),
            download(files={'runtime-base-images/images.tar': 'new partial'}),
        ], self.required)
        self.assertEqual(result, 'failure', runner.log)
        self.assertEqual(runner.downloads, ['primary', 'retry'])

    def test_preexisting_declared_files_do_not_mask_missing_download(self):
        download(files=self.complete)(self.path)
        unrelated = self.path / 'unrelated.txt'
        unrelated.write_text('keep')
        runner, result = self.run_action([download(), download()],
                                         self.required)
        self.assertEqual(result, 'failure', runner.log)
        self.assertEqual(unrelated.read_text(), 'keep')

    def test_missing_required_files_fails_before_download(self):
        download(files={'existing.tar': 'keep'})(self.path)
        runner, result = self.run_action([])
        self.assertEqual(result, 'failure', runner.log)
        self.assertEqual(runner.downloads, [])
        self.assertIn('Set required-files', runner.log)
        self.assertIn('Artifact download did not start', runner.log)
        self.assertNotIn('after 2 attempt', runner.log)
        self.assertEqual((self.path / 'existing.tar').read_text(), 'keep')

    def test_empty_or_blank_required_files_fails_before_download(self):
        for required_files in ('', '\n\n', '  \t\n \t'):
            with self.subTest(required_files=required_files):
                runner, result = self.run_action([], required_files)
                self.assertEqual(result, 'failure', runner.log)
                self.assertEqual(runner.downloads, [])
                self.assertIn('Set required-files', runner.log)

    def test_two_transport_failures_do_not_pass(self):
        runner, result = self.run_action(
            [download('failure'), download('failure')], self.required)
        self.assertEqual(result, 'failure', runner.log)
        self.assertEqual(runner.downloads, ['primary', 'retry'])
        self.assertIn('after 2 attempt', runner.log)

    def test_cleanup_failure_prevents_another_download(self):
        outside = Path(self.directory.name) / 'outside'
        outside.mkdir()
        target = outside / 'images.tar'
        target.write_text('keep')

        def incomplete_download(path, inputs):
            (path / 'runtime-base-images').symlink_to(
                outside, target_is_directory=True)
            return 'failure'

        runner, result = self.run_action([incomplete_download], self.required)
        self.assertEqual(result, 'failure', runner.log)
        self.assertEqual(runner.downloads, ['primary'])
        self.assertIn('Artifact retry did not start', runner.log)
        self.assertNotIn('after 2 attempt', runner.log)
        self.assertEqual(target.read_text(), 'keep')

    def test_unsafe_download_is_fatal_without_wait_or_retry(self):
        outside = Path(self.directory.name) / 'outside.tar'
        outside.write_text('keep')
        for kind in ('symlink', 'directory'):
            with self.subTest(kind=kind):
                destination = self.path / kind

                def unsafe_download(path, inputs):
                    target = path / 'frontend/image.tar'
                    target.parent.mkdir(parents=True)
                    if kind == 'symlink':
                        target.symlink_to(outside)
                    else:
                        target.mkdir()
                    return 'success'

                runner = CompositeAction(destination, [unsafe_download],
                                         self.required)
                self.assertEqual(runner.run(), 'failure', runner.log)
                self.assertEqual(runner.downloads, ['primary'])
                self.assertEqual(runner.steps['verify-primary']['outcome'],
                                 'failure')
                self.assertNotIn('complete',
                                 runner.steps['verify-primary']['outputs'])
                self.assertNotIn('retrying in', runner.log)
                self.assertEqual(outside.read_text(), 'keep')

    def test_missing_verification_helper_is_fatal_without_retry(self):
        runner = CompositeAction(self.path, [download(files=self.complete)],
                                 self.required)
        verifier = next(step for step in runner.action['runs']['steps']
                        if step.get('id') == 'verify-primary')
        # Fault injection changes only helper availability, not retry logic.
        verifier['env']['ARTIFACT_FILES'] = str(self.path / 'missing-helper.sh')
        self.assertEqual(runner.run(), 'failure', runner.log)
        self.assertEqual(runner.downloads, ['primary'])
        self.assertEqual(runner.steps['verify-primary']['outcome'], 'failure')
        self.assertNotIn('complete', runner.steps['verify-primary']['outputs'])
        self.assertNotIn('retrying in', runner.log)

    def test_invalid_initial_preparation_prevents_download(self):
        runner, result = self.run_action([], '../outside')
        self.assertEqual(result, 'failure', runner.log)
        self.assertEqual(runner.downloads, [])
        self.assertIn('Artifact download did not start', runner.log)
        self.assertNotIn('after 2 attempt', runner.log)

    def test_tilde_destination_is_rejected_before_download(self):
        runner = CompositeAction(self.path, [], self.required)
        runner.inputs['path'] = '~/downloads'
        self.assertEqual(runner.run(), 'failure', runner.log)
        self.assertEqual(runner.downloads, [])

    def test_cancellation_does_not_retry_or_claim_success(self):
        runner, result = self.run_action([download('cancelled')], self.required)
        self.assertEqual(result, 'cancelled', runner.log)
        self.assertEqual(runner.downloads, ['primary'])
        self.assertEqual(runner.log, '')


class RequiredFilesSafetyTest(unittest.TestCase):

    def setUp(self):
        self.directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.directory.cleanup)
        self.path = Path(self.directory.name).resolve()
        self.output = self.path / 'github-output'

    def run_helper(self, mode, required_files, download_path=None):
        self.output.write_text('')
        return subprocess.run(
            ['bash', str(ARTIFACT_FILES), mode],
            env={
                **os.environ,
                'DOWNLOAD_PATH':
                    str(self.path if download_path is None else download_path),
                'REQUIRED_FILES':
                    required_files,
                'GITHUB_OUTPUT':
                    str(self.output),
            },
            text=True,
            capture_output=True,
            check=False,
            cwd=self.path,
        )

    def test_destination_and_ancestor_symlinks_fail_before_filesystem_changes(
            self):
        outside = self.path / 'outside'
        outside.mkdir()
        sentinel = outside / 'archive.tar'
        sentinel.write_text('keep')
        link = self.path / 'destination'
        link.symlink_to(outside, target_is_directory=True)
        for destination in (str(link), str(link) + '/', str(link / 'new'),
                            'destination', 'destination/',
                            './destination/new/'):
            for mode in ('prepare', 'verify', 'check'):
                with self.subTest(destination=destination, mode=mode):
                    result = self.run_helper(mode, 'archive.tar', destination)
                    self.assertEqual(result.returncode, 2, result.stderr)
                    self.assertIn('destination contains a symlink',
                                  result.stdout)
                    self.assertEqual(sentinel.read_text(), 'keep')
                    self.assertFalse((outside / 'new').exists())

    def test_physical_absolute_and_relative_destinations_are_supported(self):
        for destination in ('new/nested/', './new/nested',
                            str(self.path / 'new/nested') + '/'):
            with self.subTest(destination=destination):
                result = self.run_helper('prepare', 'archive.tar', destination)
                self.assertEqual(result.returncode, 0,
                                 result.stdout + result.stderr)
                self.assertTrue((self.path / 'new/nested').is_dir())

    def test_native_windows_download_output_uses_actual_destination(self):
        tools = self.path / 'bin'
        tools.mkdir()
        converter = tools / 'cygpath'
        converter.write_text('#!/bin/sh\nprintf "%s\\n" "$CYGPATH_TARGET"\n')
        converter.chmod(0o755)
        target = self.path / 'download'
        target.mkdir()
        with mock.patch.dict(
                os.environ, {
                    'PATH': str(tools) + os.pathsep + os.environ['PATH'],
                    'CYGPATH_TARGET': str(target),
                }):
            for destination in (r'D:\a\_temp\download', 'D:/a/_temp/download'):
                with self.subTest(destination=destination):
                    archive = target / 'archive.tar'
                    archive.write_text('complete')
                    result = self.run_helper('verify', 'archive.tar',
                                             destination)
                    self.assertEqual(result.returncode, 0,
                                     result.stdout + result.stderr)
                    result = self.run_helper('prepare', 'archive.tar',
                                             destination)
                    self.assertEqual(result.returncode, 0,
                                     result.stdout + result.stderr)
                    self.assertFalse(archive.exists())

    def test_normalized_windows_destinations_keep_symlink_and_traversal_guards(
            self):
        tools = self.path / 'bin'
        tools.mkdir()
        converter = tools / 'cygpath'
        converter.write_text('#!/bin/sh\nprintf "%s\\n" "$CYGPATH_TARGET"\n')
        converter.chmod(0o755)
        outside = self.path / 'outside'
        outside.mkdir()
        sentinel = outside / 'archive.tar'
        sentinel.write_text('keep')
        link = self.path / 'link'
        link.symlink_to(outside, target_is_directory=True)
        for target in (str(link), str(self.path / 'missing/../outside')):
            with mock.patch.dict(
                    os.environ, {
                        'PATH': str(tools) + os.pathsep + os.environ['PATH'],
                        'CYGPATH_TARGET': target,
                    }):
                for mode in ('prepare', 'verify', 'check'):
                    result = self.run_helper(mode, 'archive.tar',
                                             r'D:\a\download')
                    self.assertEqual(result.returncode, 2,
                                     result.stdout + result.stderr)
                    self.assertEqual(sentinel.read_text(), 'keep')
                    self.assertFalse((self.path / 'missing').exists())

    def test_windows_parent_traversal_is_rejected_before_normalization(self):
        tools = self.path / 'bin'
        tools.mkdir()
        converter = tools / 'cygpath'
        converter.write_text(
            '#!/bin/sh\ntouch "$CONVERSION_MARKER"\nprintf "%s\\n" "$CYGPATH_TARGET"\n'
        )
        converter.chmod(0o755)
        outside = self.path / 'outside'
        outside.mkdir()
        sentinel = outside / 'archive.tar'
        sentinel.write_text('keep')
        marker = self.path / 'converted'
        with mock.patch.dict(
                os.environ, {
                    'PATH': str(tools) + os.pathsep + os.environ['PATH'],
                    'CYGPATH_TARGET': str(outside),
                    'CONVERSION_MARKER': str(marker),
                }):
            for destination in (r'D:\allowed\..\outside',
                                'D:/allowed/../outside'):
                for mode in ('prepare', 'verify', 'check'):
                    result = self.run_helper(mode, 'archive.tar', destination)
                    self.assertEqual(result.returncode, 2,
                                     result.stdout + result.stderr)
                    self.assertFalse(marker.exists())
                    self.assertEqual(sentinel.read_text(), 'keep')

    def test_windows_conversion_failure_preserves_existing_files(self):
        tools = self.path / 'bin'
        tools.mkdir()
        converter = tools / 'cygpath'
        converter.write_text('#!/bin/sh\nexit 7\n')
        converter.chmod(0o755)
        sentinel = self.path / 'archive.tar'
        sentinel.write_text('keep')
        with mock.patch.dict(os.environ, {
                'PATH': str(tools) + os.pathsep + os.environ['PATH'],
        }):
            result = self.run_helper('prepare', 'archive.tar', r'D:\a\download')
        self.assertEqual(result.returncode, 7, result.stdout + result.stderr)
        self.assertEqual(sentinel.read_text(), 'keep')

    def test_unset_destination_fails_before_touching_files(self):
        sentinel = self.path / 'archive.tar'
        sentinel.write_text('keep')
        environment = dict(os.environ, REQUIRED_FILES='archive.tar')
        environment.pop('DOWNLOAD_PATH', None)
        for mode in ('prepare', 'verify', 'check'):
            with self.subTest(mode=mode):
                result = subprocess.run(
                    ['bash', str(ARTIFACT_FILES), mode],
                    env=environment,
                    cwd=self.path,
                    capture_output=True,
                    text=True,
                    check=False)
                self.assertEqual(result.returncode, 2,
                                 result.stdout + result.stderr)
                self.assertIn('absolute or workspace-relative destination',
                              result.stdout)
                self.assertNotIn('unbound variable', result.stderr)
                self.assertEqual(sentinel.read_text(), 'keep')

    def test_empty_or_parent_traversal_destination_is_rejected(self):
        for destination in ('', '../outside', 'new/../outside'):
            with self.subTest(destination=destination):
                result = self.run_helper('prepare', 'archive.tar', destination)
                self.assertEqual(result.returncode, 2,
                                 result.stdout + result.stderr)
                self.assertFalse((self.path / 'new').exists())

    def test_invalid_paths_fail_before_removing_any_file(self):
        sentinel = self.path / 'sentinel'
        for invalid in ('/absolute', '../outside', 'a/../outside', './sentinel',
                        'a//file', 'directory/', ' \t../outside\r'):
            for mode in ('prepare', 'verify', 'check'):
                with self.subTest(path=invalid, mode=mode):
                    sentinel.write_text('keep')
                    result = self.run_helper(mode, f'sentinel\n{invalid}')
                    self.assertEqual(result.returncode, 2)
                    self.assertEqual(sentinel.read_text(), 'keep')
                    self.assertEqual(self.output.read_text(), '')

    def test_symlink_is_rejected_without_touching_target(self):
        target = self.path / 'target'
        target.write_text('keep')
        (self.path / 'link').symlink_to(target)
        for mode in ('prepare', 'verify', 'check'):
            with self.subTest(mode=mode):
                result = self.run_helper(mode, 'link')
                self.assertEqual(result.returncode, 2)
                self.assertEqual(target.read_text(), 'keep')
                self.assertTrue((self.path / 'link').is_symlink())

    def test_nonempty_directory_is_not_a_valid_artifact_file(self):
        directory = self.path / 'directory'
        directory.mkdir()
        (directory / 'sentinel').write_text('keep')
        for mode in ('prepare', 'verify', 'check'):
            with self.subTest(mode=mode):
                result = self.run_helper(mode, 'directory')
                self.assertEqual(result.returncode, 2)
                self.assertTrue(directory.is_dir())

    def test_verify_rejects_empty_file(self):
        (self.path / 'empty.tar').touch()
        result = self.run_helper('verify', 'empty.tar')
        self.assertEqual(result.returncode, 10)
        self.assertIn('empty.tar', result.stdout)

    def test_missing_file_has_distinct_incomplete_exit_code(self):
        result = self.run_helper('verify', 'missing.tar')
        self.assertEqual(result.returncode, 10)
        self.assertIn('missing.tar', result.stdout)

    def test_check_emits_only_recoverable_completeness_outputs(self):
        result = self.run_helper('check', 'image.tar')
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(self.output.read_text(), 'complete=false\n')
        (self.path / 'image.tar').write_text('image')
        result = self.run_helper('check', 'image.tar')
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(self.output.read_text(), 'complete=true\n')
        result = self.run_helper('check', '../unsafe.tar')
        self.assertEqual(result.returncode, 2)
        self.assertEqual(self.output.read_text(), '')

    def test_blank_lines_and_spaces_in_filename_are_supported(self):
        target = self.path / 'artifact with spaces.tar'
        target.write_text('image')
        result = self.run_helper('verify', '\nartifact with spaces.tar\n\n')
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_outer_whitespace_and_crlf_are_trimmed_but_internal_spaces_remain(
            self):
        target = self.path / 'artifact  with spaces.tar'
        required = ' \tartifact  with spaces.tar \t\r\n \t\r\n'
        target.write_text('image')
        result = self.run_helper('verify', required)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        result = self.run_helper('prepare', required)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertFalse(target.exists())

    def test_empty_required_files_fails_in_all_modes(self):
        for mode in ('prepare', 'verify', 'check'):
            for required_files in ('', '\n\n', ' \t\n'):
                with self.subTest(mode=mode, required_files=required_files):
                    result = self.run_helper(mode, required_files)
                    self.assertEqual(result.returncode, 2)
                    self.assertIn('Set required-files', result.stdout)


if __name__ == '__main__':
    unittest.main()
