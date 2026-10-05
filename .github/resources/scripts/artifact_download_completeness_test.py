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

import yaml

REPOSITORY_ROOT = Path(__file__).resolve().parents[3]
ACTION_DIRECTORY = (
    REPOSITORY_ROOT / '.github/actions/download-artifact-with-retry')
ACTION_FILE = Path(
    os.environ.get('ARTIFACT_RETRY_ACTION', ACTION_DIRECTORY / 'action.yml'))
EXPRESSION = re.compile(r'\$\{\{(.*?)\}\}', re.DOTALL)


class CompositeAction:
    """Run real YAML conditions, environment and bash; mock only the download.

    This supports the expression subset used by this action, including
    the runner's implicit success condition and continue-on-error
    outcome semantics. No download retry or completeness decisions live
    in this harness.
    """

    def __init__(self, path, attempts, required_files=None):
        self.action = yaml.safe_load(ACTION_FILE.read_text())
        self.path = path
        self.attempts = attempts
        self.downloads = []
        self.steps = {}
        self.failed = False
        self.cancelled = False
        self.log = ''
        self.inputs = {
            key: str(value.get('default', ''))
            for key, value in self.action['inputs'].items()
        }
        self.inputs.update({'path': str(path), 'retry-delay-seconds': '0'})
        if required_files is not None:
            self.inputs['required-files'] = required_files

    def evaluate(self, expression):
        expression = expression.strip()
        match = EXPRESSION.fullmatch(expression)
        if match:
            expression = match.group(1).strip()

        def reference(match):
            parts = match.group().split('.')
            value = {
                'inputs': self.inputs,
                'steps': self.steps,
                'github': {
                    'action_path': str(ACTION_DIRECTORY)
                },
            }
            for part in parts:
                value = value.get(part, {}) if isinstance(value, dict) else ''
            return repr(value if not isinstance(value, dict) else '')

        expression = re.sub(r'\b(?:inputs|steps|github)(?:\.[\w-]+)+',
                            reference, expression)
        expression = expression.replace('&&', ' and ').replace('||', ' or ')
        expression = re.sub(r'!(?!=)', 'not ', expression)
        return eval(expression, {'__builtins__': {}}, {
            'always': lambda: True,
            'cancelled': lambda: self.cancelled,
            'success': lambda: not self.failed and not self.cancelled,
            'failure': lambda: self.failed,
            'true': True,
            'false': False,
        })

    def render(self, value):
        return EXPRESSION.sub(lambda match: str(self.evaluate(match.group())),
                              str(value))

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
                destination = Path(self.render(
                    step['with']['path'])).expanduser()
                if not destination.is_absolute():
                    destination = REPOSITORY_ROOT / destination
                destination.mkdir(parents=True, exist_ok=True)
                outcome = attempt(destination)
                if outcome == 'success':
                    outputs['download-path'] = os.path.abspath(destination)
            else:
                if step['shell'] != 'bash':
                    raise AssertionError(f'Unexpected shell: {step}')
                environment = os.environ.copy()
                environment['GITHUB_ACTION_PATH'] = str(ACTION_DIRECTORY)
                environment.update({
                    key: self.render(value)
                    for key, value in step.get('env', {}).items()
                })
                result = subprocess.run(
                    [
                        'bash', '--noprofile', '--norc', '-e', '-o', 'pipefail',
                        '-c',
                        self.render(step['run'])
                    ],
                    cwd=REPOSITORY_ROOT,
                    env=environment,
                    text=True,
                    capture_output=True,
                    check=False,
                )
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

    def attempt(path):
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
        self.path = Path(self.directory.name) / 'downloads'
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

    def test_successful_but_incomplete_download_retries(self):
        runner, result = self.run_action([
            download(files={'frontend/image.tar': 'partial'}),
            download(files=self.complete),
        ], self.required)
        self.assertEqual(runner.downloads, ['primary', 'retry'], runner.log)
        self.assertEqual(result, 'success', runner.log)

    def test_missing_file_after_both_attempts_fails(self):
        runner, result = self.run_action([download(), download()],
                                         self.required)
        self.assertEqual(result, 'failure', runner.log)
        self.assertEqual(runner.downloads, ['primary', 'retry'])
        self.assertIn('frontend/image.tar', runner.log)
        self.assertIn('runtime-base-images/images.tar', runner.log)

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

    def test_cleanup_failure_prevents_another_download(self):
        outside = Path(self.directory.name) / 'outside'
        outside.mkdir()
        target = outside / 'images.tar'
        target.write_text('keep')

        def incomplete_download(path):
            (path / 'runtime-base-images').symlink_to(
                outside, target_is_directory=True)
            return 'failure'

        runner, result = self.run_action([incomplete_download], self.required)
        self.assertEqual(result, 'failure', runner.log)
        self.assertEqual(runner.downloads, ['primary'])
        self.assertEqual(target.read_text(), 'keep')

    def test_invalid_initial_preparation_prevents_download(self):
        runner, result = self.run_action([], '../outside')
        self.assertEqual(result, 'failure', runner.log)
        self.assertEqual(runner.downloads, [])

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
        self.path = Path(self.directory.name)

    def run_helper(self, mode, required_files):
        return subprocess.run(
            ['bash', str(ACTION_DIRECTORY / 'required-files.sh'), mode],
            env={
                **os.environ, 'DOWNLOAD_PATH': str(self.path),
                'REQUIRED_FILES': required_files
            },
            text=True,
            capture_output=True,
            check=False,
        )

    def test_invalid_paths_fail_before_removing_any_file(self):
        sentinel = self.path / 'sentinel'
        for invalid in ('/absolute', '../outside', 'a/../outside', './sentinel',
                        'a//file', 'directory/'):
            for mode in ('prepare', 'verify'):
                with self.subTest(path=invalid, mode=mode):
                    sentinel.write_text('keep')
                    result = self.run_helper(mode, f'sentinel\n{invalid}')
                    self.assertNotEqual(result.returncode, 0)
                    self.assertEqual(sentinel.read_text(), 'keep')

    def test_symlink_is_rejected_without_touching_target(self):
        target = self.path / 'target'
        target.write_text('keep')
        (self.path / 'link').symlink_to(target)
        for mode in ('prepare', 'verify'):
            with self.subTest(mode=mode):
                result = self.run_helper(mode, 'link')
                self.assertNotEqual(result.returncode, 0)
                self.assertEqual(target.read_text(), 'keep')
                self.assertTrue((self.path / 'link').is_symlink())

    def test_nonempty_directory_is_not_a_valid_artifact_file(self):
        directory = self.path / 'directory'
        directory.mkdir()
        (directory / 'sentinel').write_text('keep')
        for mode in ('prepare', 'verify'):
            with self.subTest(mode=mode):
                result = self.run_helper(mode, 'directory')
                self.assertNotEqual(result.returncode, 0)
                self.assertTrue(directory.is_dir())

    def test_verify_rejects_empty_file(self):
        (self.path / 'empty.tar').touch()
        result = self.run_helper('verify', 'empty.tar')
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('empty.tar', result.stdout)

    def test_blank_lines_and_spaces_in_filename_are_supported(self):
        target = self.path / 'artifact with spaces.tar'
        target.write_text('image')
        result = self.run_helper('verify', '\nartifact with spaces.tar\n\n')
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_empty_required_files_fails_in_both_modes(self):
        for mode in ('prepare', 'verify'):
            for required_files in ('', '\n\n', ' \t\n'):
                with self.subTest(mode=mode, required_files=required_files):
                    result = self.run_helper(mode, required_files)
                    self.assertNotEqual(result.returncode, 0)
                    self.assertIn('Set required-files', result.stdout)


class DeployRequiredFilesTest(unittest.TestCase):

    def test_deploy_emits_inventory_and_optional_modelcar_archive(self):
        deploy = yaml.safe_load(
            (REPOSITORY_ROOT / '.github/actions/deploy/action.yml').read_text())
        steps = deploy['runs']['steps']
        producer = next(
            step for step in steps if step.get('id') == 'image-artifacts')
        consumer = next(
            step for step in steps if step.get('uses') ==
            './.github/actions/download-artifact-with-retry')
        self.assertEqual(consumer['with']['required-files'],
                         '${{ steps.image-artifacts.outputs.required-files }}')
        self.assertEqual(producer['env']['LOAD_MODELCAR_FIXTURE'],
                         '${{ inputs.load_modelcar_fixture }}')

        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            scripts = root / '.github/resources/scripts'
            scripts.mkdir(parents=True)
            inventory = REPOSITORY_ROOT / '.github/resources/scripts/ci-image-artifacts.sh'
            (scripts / inventory.name).write_text(inventory.read_text())
            waiter = scripts / 'wait-for-image-artifacts.sh'
            waiter.write_text('#!/usr/bin/env bash\nexit 0\n')
            waiter.chmod(0o755)
            output = root / 'output'
            for modelcar in ('false', 'true'):
                with self.subTest(modelcar=modelcar):
                    output.write_text('')
                    result = subprocess.run(
                        ['bash', '-e', '-o', 'pipefail', '-c', producer['run']],
                        cwd=root,
                        env={
                            **os.environ, 'GITHUB_OUTPUT': str(output),
                            'LOAD_MODELCAR_FIXTURE': modelcar
                        },
                        text=True,
                        capture_output=True,
                        check=False,
                    )
                    self.assertEqual(result.returncode, 0, result.stderr)
                    lines = output.read_text().splitlines()
                    self.assertEqual(lines[0], 'required-files<<EOF')
                    self.assertEqual(lines[-1], 'EOF')
                    archives = set(lines[1:-1])
                    expected = {
                        f'{name}/{name}.tar'
                        for name in ('apiserver', 'scheduledworkflow',
                                     'persistenceagent', 'frontend',
                                     'viewer-crd-controller', 'driver',
                                     'launcher', 'runtime-base-images')
                    }
                    if modelcar == 'true':
                        expected.add('runtime-base-images/modelcar.tar')
                    self.assertEqual(archives, expected)


if __name__ == '__main__':
    unittest.main()
