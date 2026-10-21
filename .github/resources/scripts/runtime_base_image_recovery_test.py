#!/usr/bin/env python3
# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Exercise runtime archive recovery without registry or GitHub access."""

import json
import os
from pathlib import Path
import subprocess
import tempfile
import textwrap
import unittest

import yaml

ROOT = Path(__file__).resolve().parents[3]
CONSUMER = ROOT / '.github/workflows/image-builds.yml'
PRODUCER = ROOT / '.github/workflows/runtime-base-images.yml'
BUILD_SCRIPT = ROOT / '.github/resources/scripts/build-runtime-base-images.sh'
SOURCE_SHA = 'current-source-sha'
MODELCAR_IMAGE = 'registry.domain.local/modelcar:test'


def runtime_job_text():
    return CONSUMER.read_text().split('  runtime-base-images:\n',
                                      1)[1].split('\n  image-build:', 1)[0]


def consumer_step(name):
    return runtime_job_text().split(f'      - name: {name}\n',
                                    1)[1].split('\n      - name:', 1)[0]


def producer_artifact(**overrides):
    artifact = {
        'id': 1,
        'created_at': '2026-09-24T01:01:30Z',
        'expired': False,
        'workflow_run': {
            'id': 10,
            'head_sha': 'old-master-sha',
            'head_branch': 'master',
            'head_repository_id': 1,
            'repository_id': 1,
        },
    }
    artifact.update(overrides)
    return artifact


class FakeCommandsTestCase(unittest.TestCase):

    def setUp(self):
        self.temporary_directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary_directory.cleanup)
        self.directory = Path(self.temporary_directory.name).resolve()
        self.bin_directory = self.directory / 'bin'
        self.bin_directory.mkdir()
        self.sleep_log = self.directory / 'sleep.log'
        self.environment = {
            **os.environ,
            'PATH': f'{self.bin_directory}:{os.environ["PATH"]}',
            'SLEEP_LOG': str(self.sleep_log),
        }
        self.write_command(
            'sleep', '#!/usr/bin/env bash\n'
            'printf \'%s\\n\' "$1" >> "$SLEEP_LOG"\n')

    def write_command(self, name, contents):
        path = self.bin_directory / name
        path.write_text(contents)
        path.chmod(0o755)

    def sleeps(self):
        return self.sleep_log.read_text().splitlines() if self.sleep_log.exists(
        ) else []


class ProducerLookupRecoveryTest(FakeCommandsTestCase):

    def run_lookup(self, artifacts, fail_api=False):
        payload = self.directory / 'artifacts.json'
        payload.write_text(json.dumps({'artifacts': artifacts}))
        github_output = self.directory / 'github-output'
        github_output.touch()
        gh_log = self.directory / 'gh.log'
        self.write_command(
            'gh', '#!/usr/bin/env bash\n'
            'printf \'%s\\n\' "$*" >> "$GH_LOG"\n'
            '[[ "$FAIL_API" == "true" ]] && exit 1\n'
            'cat "$ARTIFACT_PAYLOAD"\n')
        lookup = textwrap.dedent(
            consumer_step('Find runtime base image producer').split(
                '        run: |\n', 1)[1])
        result = subprocess.run(
            ['bash', '-e', '-o', 'pipefail', '-c', lookup],
            cwd=ROOT,
            env={
                **self.environment,
                'ARTIFACT_NAME':
                    'runtime-base-images-older-release-fingerprint',
                'ARTIFACT_PAYLOAD':
                    str(payload),
                'FAIL_API':
                    str(fail_api).lower(),
                'GH_LOG':
                    str(gh_log),
                'GITHUB_OUTPUT':
                    str(github_output),
                'GITHUB_REPOSITORY':
                    'kubeflow/pipelines',
                'SOURCE_SHA':
                    SOURCE_SHA,
            },
            capture_output=True,
            text=True,
            check=False,
            timeout=10,
        )
        return result, github_output.read_text(), gh_log.read_text().splitlines(
        )

    def test_trusted_hit_returns_producer_without_waiting(self):
        result, output, requests = self.run_lookup([producer_artifact()])

        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(output, 'run-id=10\n')
        self.assertEqual(len(requests), 1)
        self.assertIn('name=runtime-base-images-older-release-fingerprint',
                      requests[0])
        self.assertEqual(self.sleeps(), [])

    def test_missing_generation_falls_back_after_bounded_wait(self):
        self.assert_bounded_miss([])

    def test_expired_master_generation_falls_back(self):
        self.assert_bounded_miss([producer_artifact(expired=True)])

    def test_unrelated_fork_cannot_supply_recovery_archive(self):
        candidate = producer_artifact()
        candidate['workflow_run']['head_repository_id'] = 2
        self.assert_bounded_miss([candidate])

    def test_current_source_fork_producer_remains_eligible(self):
        candidate = producer_artifact()
        candidate['workflow_run'].update({
            'head_repository_id': 2,
            'head_sha': SOURCE_SHA,
            'head_branch': 'feature',
        })
        result, output, requests = self.run_lookup([candidate])

        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(output, 'run-id=10\n')
        self.assertEqual(len(requests), 1)

    def test_artifact_api_failure_does_not_block_local_generation(self):
        self.assert_bounded_miss([], fail_api=True)

    def assert_bounded_miss(self, artifacts, fail_api=False):
        result, output, requests = self.run_lookup(artifacts, fail_api)

        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(output, '')
        self.assertEqual(len(requests), 3)
        self.assertEqual(self.sleeps(), ['10', '10'])
        self.assertIn('::warning::', result.stdout)


class RuntimeArchiveBuildTest(FakeCommandsTestCase):

    def setUp(self):
        super().setUp()
        self.docker_log = self.directory / 'docker.jsonl'
        self.output_directory = self.directory / 'archives with spaces'
        self.cache_directory = self.directory / 'runtime-base-images-cache'
        self.write_command(
            'docker', '''#!/usr/bin/env python3
import json
import os
from pathlib import Path
import sys

arguments = sys.argv[1:]
failed_cache = os.environ.get('EXPECTED_REMOVED_CACHE')
if failed_cache and Path(failed_cache).exists():
    sys.exit('Failed cache still occupies disk before rebuild')
with open(os.environ['DOCKER_LOG'], 'a') as log:
    log.write(json.dumps(arguments) + '\\n')
if arguments[0] == os.environ.get('FAIL_DOCKER_COMMAND'):
    sys.exit(1)
if arguments[0] == 'save':
    flag = '-o' if '-o' in arguments else '--output'
    Path(arguments[arguments.index(flag) + 1]).write_bytes(b'new archive')
''')

    def run_build(self, failed_command=''):
        result = subprocess.run(
            ['bash', str(BUILD_SCRIPT),
             str(self.output_directory)],
            cwd=ROOT,
            env={
                **self.environment,
                'DOCKER_LOG': str(self.docker_log),
                'FAIL_DOCKER_COMMAND': failed_command,
            },
            capture_output=True,
            text=True,
            check=False,
            timeout=10,
        )
        commands = [
            json.loads(line)
            for line in self.docker_log.read_text().splitlines()
        ]
        return result, commands

    def test_builds_both_archives_from_checked_out_inventory(self):
        result, commands = self.run_build()

        self.assertEqual(result.returncode, 0, result.stderr)
        for name in ('runtime-base-images.tar', 'modelcar.tar'):
            self.assertEqual((self.output_directory / name).read_bytes(),
                             b'new archive')
        inventory = (ROOT /
                     '.github/resources/runtime-base-images.txt').read_text()
        images = [
            line for line in inventory.splitlines()
            if line and not line.startswith('#')
        ]
        pulls = [command[1] for command in commands if command[0] == 'pull']
        self.assertEqual(pulls, images)
        builds = [command for command in commands if command[0] == 'build']
        self.assertEqual(len(builds), 1)
        self.assertIn(
            'test_data/sdk_compiled_pipelines/valid/critical/modelcar/Dockerfile',
            builds[0])
        self.assertIn(MODELCAR_IMAGE, builds[0])
        saves = [command for command in commands if command[0] == 'save']
        self.assertEqual(len(saves), 2)
        self.assertIn(MODELCAR_IMAGE, saves[1])
        self.assertEqual(self.sleeps(), [])

    def test_pull_failure_stops_before_saving_or_building_modelcar(self):
        result, commands = self.run_build('pull')

        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(len(commands), 5)
        self.assertTrue(all(command[0] == 'pull' for command in commands))
        self.assertEqual(len(self.sleeps()), 4)
        self.assertFalse(
            (self.output_directory / 'runtime-base-images.tar').exists())
        self.assertFalse((self.output_directory / 'modelcar.tar').exists())

    def test_modelcar_build_failure_is_bounded_and_not_saved(self):
        result, commands = self.run_build('build')

        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(sum(command[0] == 'build' for command in commands), 3)
        self.assertEqual(self.sleeps(), ['30', '30'])
        self.assertTrue(
            (self.output_directory / 'runtime-base-images.tar').exists())
        self.assertFalse((self.output_directory / 'modelcar.tar').exists())
        self.assertFalse(
            any(command[0] == 'save' and MODELCAR_IMAGE in command
                for command in commands))

    def run_recovery(self,
                     required_files=None,
                     failed_command='',
                     cache_path=None):
        job = yaml.safe_load(
            CONSUMER.read_text())['jobs']['runtime-base-images']
        step = next(step for step in job['steps']
                    if step.get('name') == 'Build runtime base image archives')
        manifest = (
            job['env']['RUNTIME_IMAGE_FILES']
            if required_files is None else required_files)
        summary = self.directory / 'summary'
        output = self.directory / 'recovery-output'
        output.touch()
        result = subprocess.run(
            ['bash', '-e', '-o', 'pipefail', '-c', step['run']],
            cwd=ROOT,
            env={
                **self.environment,
                'DOCKER_LOG':
                    str(self.docker_log),
                'FAIL_DOCKER_COMMAND':
                    failed_command,
                'DOWNLOAD_OUTCOME':
                    'failure',
                'ARTIFACTS_PATH':
                    job['env']['ARTIFACTS_PATH']
                    if cache_path is None else cache_path,
                'EXPECTED_REMOVED_CACHE':
                    str(self.cache_directory),
                'GITHUB_WORKSPACE':
                    str(self.directory),
                'REQUIRED_FILES':
                    manifest,
                'GITHUB_STEP_SUMMARY':
                    str(summary),
                'GITHUB_OUTPUT':
                    str(output),
                'RUNNER_TEMP':
                    str(self.directory),
            },
            capture_output=True,
            text=True,
            check=False,
            timeout=10,
        )
        outputs = dict(
            line.split('=', 1) for line in output.read_text().splitlines())
        return result, summary.read_text() if summary.exists() else '', outputs

    def test_recovery_reports_cache_failure_and_verifies_fresh_archives(self):
        self.cache_directory.mkdir()
        (self.cache_directory / 'fixture.tar').write_text('stale cache residue')
        result, summary, outputs = self.run_recovery()
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertIn('cache download: failure', summary)
        self.assertIn('Rebuilding and verifying', summary)
        rebuilt = Path(outputs['path'])
        self.assertNotEqual(rebuilt, self.cache_directory)
        self.assertFalse(self.cache_directory.exists())
        self.assertEqual(
            set(path.name for path in rebuilt.iterdir()),
            {'runtime-base-images.tar', 'modelcar.tar'})
        for path in rebuilt.iterdir():
            self.assertEqual(path.read_bytes(), b'new archive')

    def test_drifted_manifest_fails_after_rebuild_even_with_stale_cache_file(
            self):
        self.cache_directory.mkdir()
        (self.cache_directory / 'renamed-modelcar.tar').write_text('stale')
        result, _, outputs = self.run_recovery(
            'runtime-base-images.tar\nrenamed-modelcar.tar')
        self.assertNotEqual(result.returncode, 0)
        self.assertIn(
            'Missing or empty required artifact file: renamed-modelcar.tar',
            result.stdout)
        self.assertFalse(self.cache_directory.exists())
        self.assertEqual(outputs, {})

    def test_recovery_unlinks_cache_symlink_without_removing_target(self):
        outside = self.directory / 'outside'
        outside.mkdir()
        sentinel = outside / 'archive.tar'
        sentinel.write_text('keep')
        self.cache_directory.symlink_to(outside, target_is_directory=True)
        result, _, outputs = self.run_recovery()
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertTrue(outputs['path'])
        self.assertFalse(self.cache_directory.is_symlink())
        self.assertEqual(sentinel.read_text(), 'keep')

    def test_recovery_rejects_an_unexpected_cleanup_path(self):
        self.output_directory.mkdir()
        sentinel = self.output_directory / 'archive.tar'
        sentinel.write_text('keep')
        result, _, outputs = self.run_recovery(
            cache_path=str(self.output_directory))
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('Unexpected runtime image cache path', result.stdout)
        self.assertEqual(outputs, {})
        self.assertEqual(sentinel.read_text(), 'keep')
        self.assertFalse(self.docker_log.exists())

    def test_failed_rebuild_does_not_publish_a_directory(self):
        result, _, outputs = self.run_recovery(failed_command='build')
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(outputs, {})


class RuntimeArchiveRecoveryWiringTest(unittest.TestCase):

    def test_missing_or_failed_download_rebuilds_before_upload(self):
        job = runtime_job_text()
        download = consumer_step('Download runtime base images from producer')
        mirror = consumer_step('Configure Docker Hub mirror')
        build = consumer_step('Build runtime base image archives')
        upload = consumer_step('Upload artifact')
        recovery = "if: steps.download-runtime-base-images.outcome != 'success'"

        self.assertIn(
            "if: steps.runtime-base-images-producer.outputs.run-id != ''",
            download)
        self.assertIn('continue-on-error: true', download)
        self.assertIn('uses: ./.github/actions/download-artifact-with-retry',
                      download)
        self.assertIn(recovery, build)
        self.assertIn(recovery, mirror)
        self.assertIn('id: build-runtime-base-images', build)
        self.assertIn(
            'path: ${{ steps.build-runtime-base-images.outputs.path || env.ARTIFACTS_PATH }}',
            upload)
        self.assertLess(job.index(download), job.index(mirror))
        self.assertLess(job.index(mirror), job.index(build))
        self.assertLess(job.index(build), job.index(upload))
        self.assertNotIn('continue-on-error', build)
        self.assertNotIn('if:', upload)
        self.assertIn('name: ${{ env.ARTIFACT_NAME }}', upload)
        self.assertIn('ARTIFACT_NAME: "runtime-base-images"', job)
        self.assertIn('      actions: read\n      contents: read', job)
        self.assertNotIn(': write', job)
        self.assertNotIn('release-2.18', job)
        self.assertIn('required-files: ${{ env.RUNTIME_IMAGE_FILES }}',
                      download)
        self.assertIn('REQUIRED_FILES: ${{ env.RUNTIME_IMAGE_FILES }}', build)


if __name__ == '__main__':
    unittest.main()
