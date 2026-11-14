"""Queue producers and consumers must share the exact candidate identity."""

import copy
import json
import subprocess
import tempfile
import unittest
from itertools import product
from pathlib import Path
from unittest import mock

import shared_ci_images as shared
from shared_ci_images_test import FakeAPI, identity

ROOT = Path(__file__).resolve().parents[3]


def queue_identity(run_id=10, branch='master', event='merge_group'):
    value = identity(run_id)
    value.update(
        event=event,
        pr=None,
        ref=f'refs/heads/gh-readonly-queue/{branch}/pr-23-abcdef',
    )
    value['merge_group'] = {
        'head_sha': value['source_sha'],
        'head_ref': value['ref'],
        'base_sha': 'c' * 40,
        'base_ref': f'refs/heads/{branch}',
    }
    return value


class SharedQueueImagesTest(unittest.TestCase):
    def test_queue_images_resolve_and_verify_both_architectures(self):
        for branch, architecture, producer_event, consumer_event in product(
            ['master', 'main'],
            ['amd64', 'arm64'],
            ['merge_group', 'workflow_dispatch'],
            ['merge_group', 'workflow_dispatch'],
        ):
            with (
                self.subTest(
                    branch=branch,
                    architecture=architecture,
                    producer=producer_event,
                    consumer=consumer_event,
                ),
                tempfile.TemporaryDirectory() as directory,
            ):
                directory = Path(directory)
                source = queue_identity(branch=branch, event=producer_event)
                consumer = queue_identity(
                    20, branch=branch, event=consumer_event
                )
                api = FakeAPI(source)
                api.current_run['head_branch'] = source['ref'].removeprefix(
                    'refs/heads/'
                )
                if architecture == 'amd64':
                    api.bundle('runtime-base-images', architecture, directory)
                    runtime = shared.resolve(
                        api,
                        consumer,
                        architecture,
                        'runtime',
                        directory / 'state.json',
                        timeout=1,
                        poll=1,
                    )
                    shared.verify(api, runtime, consumer, directory, 'flat')
                api.images(architecture, directory)
                state = shared.resolve(
                    api,
                    consumer,
                    architecture,
                    'images',
                    directory / 'state.json',
                    timeout=1,
                    poll=1,
                )
                shared.verify(api, state, consumer, directory, 'flat')
                self.assertEqual(state['identity'], source)

    def test_dispatched_and_native_queue_sources_share_exact_candidate(self):
        for branch, producer_event, consumer_event in product(
            ['master', 'main'],
            ['merge_group', 'workflow_dispatch'],
            ['merge_group', 'workflow_dispatch'],
        ):
            with self.subTest(
                branch=branch, producer=producer_event, consumer=consumer_event
            ):
                source = queue_identity(branch=branch, event=producer_event)
                consumer = queue_identity(
                    20, branch=branch, event=consumer_event
                )
                shared.same_source(source, consumer)
                api = FakeAPI(source)
                api.current_run['head_branch'] = source['ref'].removeprefix(
                    'refs/heads/'
                )
                self.assertEqual(
                    shared.candidate_runs([api.current_run], consumer),
                    [api.current_run],
                )
                shared.validate_run(api.current_run, source)
                api.current_run['head_branch'] = 'master'
                self.assertEqual(
                    shared.candidate_runs([api.current_run], consumer), []
                )
                with self.assertRaises(shared.InvalidEvidence):
                    shared.validate_run(api.current_run, source)

    def test_dispatched_queue_requires_complete_candidate_inputs(self):
        for branch in ['master', 'main']:
            source = queue_identity(branch=branch, event='workflow_dispatch')
            inputs = {
                'queue_head_sha': source['source_sha'],
                'queue_base_ref': branch,
                'queue_base_sha': source['merge_group']['base_sha'],
            }
            with tempfile.TemporaryDirectory() as directory:
                event = Path(directory) / 'event.json'
                env = {
                    'GITHUB_EVENT_PATH': str(event),
                    'GITHUB_EVENT_NAME': 'workflow_dispatch',
                    'GITHUB_REPOSITORY': source['repository'],
                    'GITHUB_REPOSITORY_ID': '100',
                    'GITHUB_REF': source['ref'],
                    'GITHUB_SHA': source['source_sha'],
                    'GITHUB_RUN_ID': '10',
                    'GITHUB_RUN_ATTEMPT': '1',
                }
                event.write_text(json.dumps({'inputs': inputs}))
                self.assertEqual(
                    shared.environment_identity(env, checkout=False), source
                )
                for key in inputs:
                    with self.subTest(branch=branch, missing=key):
                        bad = dict(inputs)
                        del bad[key]
                        event.write_text(json.dumps({'inputs': bad}))
                        with self.assertRaises(shared.InvalidEvidence):
                            shared.environment_identity(env, checkout=False)
                for bad in [
                    dict(inputs, queue_head_sha='d' * 40),
                    dict(inputs, queue_base_ref='release-2.18'),
                    dict(inputs, queue_base_sha='not-a-sha'),
                ]:
                    event.write_text(json.dumps({'inputs': bad}))
                    with self.assertRaises(shared.InvalidEvidence):
                        shared.environment_identity(env, checkout=False)
                event.write_text(json.dumps({'inputs': inputs}))
                env['GITHUB_REF'] = f'refs/heads/{branch}'
                with self.assertRaises(shared.InvalidEvidence):
                    shared.environment_identity(env, checkout=False)

    def test_queue_refs_cannot_fall_back_to_generic_manual_or_push_identity(
        self,
    ):
        for event in ['workflow_dispatch', 'push']:
            source = queue_identity(event=event)
            source.pop('merge_group')
            with (
                self.subTest(event=event),
                self.assertRaises(shared.InvalidEvidence),
            ):
                shared.validate_identity(source)
        source = queue_identity(event='workflow_dispatch')
        push = dict(source, event='push')
        with self.assertRaises(shared.InvalidEvidence):
            shared.same_source(push, source)

    def test_queue_provenance_rejects_other_candidate_or_base(self):
        source = queue_identity()
        for field, changed in [
            ('head_sha', 'b' * 40),
            ('base_sha', 'd' * 40),
            ('head_ref', 'refs/heads/gh-readonly-queue/master/pr-24-abcdef'),
            ('base_ref', 'refs/heads/release-2.18'),
        ]:
            with self.subTest(field=field):
                other = copy.deepcopy(source)
                other['merge_group'][field] = changed
                with self.assertRaises(shared.InvalidEvidence):
                    shared.same_source(source, other)
        other = dict(source, event='push')
        other.pop('merge_group')
        with self.assertRaises(shared.InvalidEvidence):
            shared.same_source(other, source)

    def test_queue_identity_rejects_cross_branch_and_release_refs(self):
        for branch in ['master', 'main']:
            for base in ['master', 'main', 'release-2.18']:
                if branch == base:
                    continue
                with self.subTest(branch=branch, base=base):
                    source = queue_identity(branch=branch)
                    source['merge_group']['base_ref'] = f'refs/heads/{base}'
                    with self.assertRaises(shared.InvalidEvidence):
                        shared.validate_identity(source)
        with self.assertRaises(shared.InvalidEvidence):
            shared.validate_identity(queue_identity(branch='release-2.18'))
        with self.assertRaises(shared.InvalidEvidence):
            shared.same_source(queue_identity(), queue_identity(branch='main'))

    def test_event_and_checkout_must_match_queue_identity(self):
        source = queue_identity()
        with tempfile.TemporaryDirectory() as directory:
            event = Path(directory) / 'event.json'
            event.write_text(json.dumps({'merge_group': source['merge_group']}))
            env = {
                'GITHUB_EVENT_PATH': str(event),
                'GITHUB_EVENT_NAME': 'merge_group',
                'GITHUB_REPOSITORY': source['repository'],
                'GITHUB_REPOSITORY_ID': '100',
                'GITHUB_REF': source['ref'],
                'GITHUB_SHA': source['source_sha'],
                'GITHUB_RUN_ID': '10',
                'GITHUB_RUN_ATTEMPT': '1',
            }
            with mock.patch.object(
                shared.subprocess,
                'check_output',
                return_value=source['source_sha'],
            ):
                self.assertEqual(shared.environment_identity(env), source)
            with mock.patch.object(
                shared.subprocess, 'check_output', return_value='d' * 40
            ):
                with self.assertRaises(shared.InvalidEvidence):
                    shared.environment_identity(env)
            env['GITHUB_SHA'] = 'd' * 40
            with self.assertRaises(shared.InvalidEvidence):
                shared.environment_identity(env, checkout=False)

    def test_queue_run_requires_candidate_branch(self):
        source = queue_identity()
        api = FakeAPI(source)
        for branch in [
            'master',
            'gh-readonly-queue/release-2.18/pr-23-abcdef',
            None,
        ]:
            with self.subTest(branch=branch):
                api.current_run['head_branch'] = branch
                with self.assertRaises(shared.InvalidEvidence):
                    shared.validate_run(api.current_run, source)

    def test_queue_architectures_do_not_use_single_pr_paths(self):
        subprocess.run(
            [
                'node',
                '-e',
                """
const {architectures} = require('./.github/resources/scripts/shared_ci_image_plan');
const assert = require('node:assert/strict');
for (const branch of ['master', 'main']) {
  assert.deepEqual(architectures({workflows: []}, 'merge_group',
    {merge_group: {base_ref: `refs/heads/${branch}`}}, []), {amd64: true, arm64: true});
}
assert.throws(() => architectures({workflows: []}, 'merge_group',
  {merge_group: {base_ref: 'refs/heads/release-2.18'}}, []));
""",
            ],
            cwd=ROOT,
            check=True,
        )


if __name__ == '__main__':
    unittest.main()
