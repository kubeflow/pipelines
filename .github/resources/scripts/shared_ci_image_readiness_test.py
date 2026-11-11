"""One source-bound image readiness result fans out safely to workflow
lanes."""

import copy
import json
import tempfile
import unittest
from pathlib import Path
from unittest import mock

import shared_ci_images as shared
import yaml
from shared_ci_images_test import FakeAPI, identity

ROOT = Path(__file__).resolve().parents[3]
READY_WORKFLOW = './.github/workflows/shared-ci-images-ready.yml'
READY_OUTPUT = '${{ needs.image-artifacts-ready.outputs.state }}'
CONSUMERS = {
    './.github/actions/deploy': 'image_ready_state',
    './.github/actions/shared-ci-images': 'ready-state',
}


class ReadinessStateTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.directory = Path(self.temp.name)
        self.consumer = identity(20)
        self.api = FakeAPI(identity())
        self.api.images('amd64', self.directory)
        self.api.runtime(self.directory)
        self.path = self.directory / 'state.json'

    def prepare(self):
        return shared.resolve(
            self.api,
            self.consumer,
            'amd64',
            'all',
            self.path,
            timeout=10,
            poll=1,
        )

    def test_eight_lanes_share_one_readiness_metadata_pass(self):
        with (
            mock.patch.object(self.api, 'runs', wraps=self.api.runs) as runs,
            mock.patch.object(self.api, 'run', wraps=self.api.run) as run,
            mock.patch.object(
                self.api, 'artifacts', wraps=self.api.artifacts
            ) as artifacts,
        ):
            for lane in range(8):
                path = self.directory / f'old-lane-{lane}.json'
                for kind in ('runtime', 'images'):
                    shared.resolve(self.api, self.consumer, 'amd64', kind, path)
            baseline = runs.call_count + run.call_count + artifacts.call_count
            self.assertEqual(baseline, 48)
            for request in (runs, run, artifacts):
                request.reset_mock()
            state = self.prepare()
            for lane in range(8):
                for kind in ('runtime', 'images'):
                    imported = shared.import_ready_state(
                        json.dumps(state), self.consumer, 'amd64', kind
                    )
                    shared.emit_outputs(
                        self.directory / f'out-{lane}', imported, kind
                    )
            self.assertEqual(runs.call_count, 1)
            self.assertEqual(run.call_count, 1)
            self.assertEqual(artifacts.call_count, 1)
            # Each archive is still checked against fresh metadata after download.
            shared.verify(
                self.api,
                imported,
                self.consumer,
                self.directory,
                'flat',
                'images',
            )
            self.assertEqual(artifacts.call_count, 2)
            self.assertEqual(run.call_count, 2)

    def test_combined_readiness_waits_for_runtime_as_well_as_branch_images(
        self,
    ):
        runtime_id = next(
            k
            for k, a in self.api.items.items()
            if a['name'].startswith('runtime-base-images')
        )
        runtime = self.api.items.pop(runtime_id)
        now = [0]

        def sleep(seconds):
            now[0] += seconds
            self.api.items[runtime_id] = runtime

        state = shared.resolve(
            self.api,
            self.consumer,
            'amd64',
            'all',
            self.path,
            timeout=10,
            poll=1,
            clock=lambda: now[0],
            sleep=sleep,
        )
        self.assertEqual(now[0], 1)
        self.assertEqual(set(state['sets']), {'runtime', 'images'})
        self.assertEqual(len(state['sets']['images']), len(shared.IMAGES))
        self.assertEqual(
            set(state['sets']['runtime'][0]['files']),
            {'runtime-base-images.tar', 'modelcar.tar'},
        )

    def test_cli_passes_readiness_output_to_a_retried_consumer(self):
        output = self.directory / 'ready-output'
        api = self.api

        class FakeGitHub:
            def __new__(cls, repository):
                return api

        with (
            mock.patch.object(
                shared, 'environment_identity', return_value=self.consumer
            ),
            mock.patch.object(shared, 'GitHub', FakeGitHub),
            mock.patch(
                'sys.argv',
                [
                    'shared_ci_images.py',
                    'prepare',
                    '--architecture',
                    'amd64',
                    '--state',
                    str(self.path),
                    '--output',
                    str(output),
                ],
            ),
        ):
            self.assertEqual(shared.main(), 0)
        name, value = output.read_text().strip().split('=', 1)
        self.assertEqual(name, 'state')
        retry = dict(self.consumer, run_attempt=2)
        imported_path = self.directory / 'retried-state'
        imported_output = self.directory / 'download-output'
        with (
            mock.patch.object(
                shared, 'environment_identity', return_value=retry
            ),
            mock.patch.object(shared, 'GitHub', FakeGitHub),
            mock.patch.object(
                api, 'runs', side_effect=AssertionError('rediscovery')
            ),
            mock.patch.object(
                api, 'run', side_effect=AssertionError('readiness polling')
            ),
            mock.patch.object(
                api,
                'artifacts',
                side_effect=AssertionError('readiness polling'),
            ),
            mock.patch.dict('os.environ', {'SHARED_IMAGE_READY_STATE': value}),
            mock.patch(
                'sys.argv',
                [
                    'shared_ci_images.py',
                    'import-ready',
                    '--architecture',
                    'amd64',
                    '--kind',
                    'runtime',
                    '--state',
                    str(imported_path),
                    '--output',
                    str(imported_output),
                ],
            ),
        ):
            self.assertEqual(shared.main(), 0)
        imported = shared.load_state(imported_path, retry, 'amd64')
        self.assertEqual(imported['identity'], identity())
        self.assertIn('artifact-ids=', imported_output.read_text())
        self.assertIn('modelcar.tar', imported_output.read_text())

    def test_readiness_failure_never_publishes_a_state_output(self):
        self.api.current_run.update(status='completed', conclusion='failure')
        output = self.directory / 'outputs'
        api = self.api

        class FakeGitHub:
            def __new__(cls, repository):
                return api

        with (
            mock.patch.object(
                shared, 'environment_identity', return_value=self.consumer
            ),
            mock.patch.object(shared, 'GitHub', FakeGitHub),
            mock.patch(
                'sys.argv',
                [
                    'shared_ci_images.py',
                    'prepare',
                    '--architecture',
                    'amd64',
                    '--state',
                    str(self.path),
                    '--output',
                    str(output),
                ],
            ),
        ):
            self.assertEqual(shared.main(), 1)
        self.assertFalse(output.exists())

    def test_same_consumer_run_retry_reuses_readiness_without_changing_producer(
        self,
    ):
        state = self.prepare()
        retry = dict(self.consumer, run_attempt=2)
        imported = shared.import_ready_state(
            json.dumps(state), retry, 'amd64', 'images'
        )
        self.assertEqual(imported['consumer'], retry)
        self.assertEqual(imported['identity'], identity())
        shared.verify(
            self.api, imported, retry, self.directory, 'flat', 'images'
        )
        self.api.current_run['run_attempt'] = 2
        with self.assertRaisesRegex(
            shared.InvalidEvidence, 'run_attempt mismatch'
        ):
            shared.verify(
                self.api, imported, retry, self.directory, 'flat', 'images'
            )

    def test_foreign_source_run_architecture_and_future_attempt_are_rejected(
        self,
    ):
        state = self.prepare()
        variants = [
            (dict(self.consumer, run_id=21), 'amd64'),
            (dict(self.consumer, source_sha='d' * 40), 'amd64'),
            (self.consumer, 'arm64'),
        ]
        for consumer, architecture in variants:
            with (
                self.subTest(consumer=consumer, architecture=architecture),
                self.assertRaises(shared.InvalidEvidence),
            ):
                shared.import_ready_state(
                    json.dumps(state), consumer, architecture, 'images'
                )
        state['consumer']['run_attempt'] = 2
        with self.assertRaises(shared.InvalidEvidence):
            shared.import_ready_state(
                json.dumps(state),
                dict(self.consumer, run_attempt=1),
                'amd64',
                'images',
            )

    def test_incomplete_or_mixed_readiness_is_rejected_before_download(self):
        state = self.prepare()
        variants = []
        missing = copy.deepcopy(state)
        missing['sets']['images'].pop()
        variants.append(missing)
        duplicate = copy.deepcopy(state)
        duplicate['sets']['images'][0] = duplicate['sets']['images'][1]
        variants.append(duplicate)
        mixed = copy.deepcopy(state)
        mixed['sets']['runtime'][0]['identity']['run_attempt'] = 2
        variants.append(mixed)
        unknown = copy.deepcopy(state)
        unknown['sets']['other'] = []
        variants.append(unknown)
        for value in variants:
            with (
                self.subTest(value=value),
                self.assertRaises(shared.InvalidEvidence),
            ):
                shared.import_ready_state(
                    json.dumps(value), self.consumer, 'amd64', 'images'
                )

    def test_imported_state_still_rejects_deleted_artifacts_and_failed_producer(
        self,
    ):
        state = self.prepare()
        imported = shared.import_ready_state(
            json.dumps(state), self.consumer, 'amd64', 'runtime'
        )
        runtime_id = imported['sets']['runtime'][0]['artifact_id']
        item = self.api.items.pop(runtime_id)
        with self.assertRaises(shared.InvalidEvidence):
            shared.verify(
                self.api,
                imported,
                self.consumer,
                self.directory,
                'flat',
                'runtime',
            )
        self.api.items[runtime_id] = item
        self.api.current_run.update(status='completed', conclusion='failure')
        with self.assertRaises(shared.InvalidEvidence):
            shared.verify(
                self.api,
                imported,
                self.consumer,
                self.directory,
                'flat',
                'runtime',
            )

    def test_arm_readiness_does_not_require_amd64_runtime(self):
        directory = self.directory / 'arm'
        directory.mkdir()
        api = FakeAPI(identity())
        api.images('arm64', directory)
        state = shared.resolve(api, self.consumer, 'arm64', 'images', self.path)
        imported = shared.import_ready_state(
            json.dumps(state), self.consumer, 'arm64', 'images'
        )
        self.assertEqual(set(imported['sets']), {'images'})
        shared.verify(api, imported, self.consumer, directory, 'flat')


class ReadinessWorkflowTest(unittest.TestCase):
    def test_every_consumer_is_guarded_by_one_successful_readiness_job(self):
        callers = 0
        consumers = 0
        for path in (ROOT / '.github/workflows').glob('*.y*ml'):
            workflow = yaml.safe_load(path.read_text())
            jobs = workflow['jobs']
            consuming = {
                name: [
                    s
                    for s in job.get('steps', [])
                    if s.get('uses') in CONSUMERS
                ]
                for name, job in jobs.items()
            }
            consuming = {
                name: steps for name, steps in consuming.items() if steps
            }
            if not consuming:
                continue
            callers += 1
            ready_jobs = [
                name
                for name, job in jobs.items()
                if job.get('uses') == READY_WORKFLOW
            ]
            self.assertEqual(ready_jobs, ['image-artifacts-ready'], path.name)
            ready = jobs['image-artifacts-ready']
            self.assertNotIn('strategy', ready)
            self.assertNotIn('continue-on-error', ready)
            for name, steps in consuming.items():
                consumers += 1
                with self.subTest(workflow=path.name, job=name):
                    job = jobs[name]
                    self.assertEqual(job['needs'], 'image-artifacts-ready')
                    self.assertNotIn('always()', job.get('if', ''))
                    self.assertNotIn('!cancelled()', job.get('if', ''))
                    for step in steps:
                        self.assertEqual(
                            step['with'][CONSUMERS[step['uses']]], READY_OUTPUT
                        )
            if path.name == 'upgrade-test.yml':
                self.assertEqual(ready['if'], '${{ false }}')
                self.assertTrue(
                    all(
                        jobs[name]['if'] == '${{ false }}' for name in consuming
                    )
                )
            else:
                self.assertNotIn('if', ready)
            events = workflow.get('on', workflow.get(True))
            self.assertIn(
                '.github/workflows/shared-ci-images-ready.yml',
                events['pull_request']['paths'],
            )
        self.assertEqual(callers, 8)
        self.assertEqual(consumers, 14)

    def test_readiness_propagates_failure_and_consumers_keep_safe_default(self):
        workflow = yaml.safe_load(
            (ROOT / '.github/workflows/shared-ci-images-ready.yml').read_text()
        )
        self.assertEqual(set(workflow['jobs']), {'ready'})
        ready = workflow['jobs']['ready']
        resolve = next(
            step for step in ready['steps'] if step.get('id') == 'resolve'
        )
        self.assertNotIn('continue-on-error', ready)
        self.assertNotIn('continue-on-error', resolve)
        self.assertIn('shared_ci_images.py prepare', resolve['run'])
        self.assertEqual(
            ready['outputs']['state'], '${{ steps.resolve.outputs.state }}'
        )
        deploy = yaml.safe_load(
            (ROOT / '.github/actions/deploy/action.yml').read_text()
        )
        self.assertEqual(deploy['inputs']['image_ready_state']['default'], '')
        for step in deploy['runs']['steps']:
            if step.get('uses') == './.github/actions/shared-ci-images':
                self.assertEqual(
                    step['with']['ready-state'],
                    '${{ inputs.image_ready_state }}',
                )
        action = yaml.safe_load(
            (ROOT / '.github/actions/shared-ci-images/action.yml').read_text()
        )
        self.assertEqual(action['inputs']['ready-state']['default'], '')
        resolver = next(
            step
            for step in action['runs']['steps']
            if step.get('id') == 'resolve'
        )
        self.assertIn('command=resolve', resolver['run'])
        self.assertIn('command=import-ready', resolver['run'])
        self.assertIn('[[ -n "$SHARED_IMAGE_READY_STATE" ]]', resolver['run'])
        self.assertEqual(
            resolver['env']['SHARED_IMAGE_READY_STATE'],
            '${{ inputs.ready-state }}',
        )
        verification = action['runs']['steps'][-1]
        self.assertIn('shared_ci_images.py verify', verification['run'])
        self.assertNotIn('if', verification)
        self.assertNotIn('continue-on-error', verification)


if __name__ == '__main__':
    unittest.main()
