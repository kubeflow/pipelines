"""Queue producers and consumers must share the exact candidate identity."""

import copy
import json
import subprocess
import tempfile
import unittest
from pathlib import Path
from unittest import mock

import shared_ci_images as shared
from shared_ci_images_test import FakeAPI, identity

ROOT = Path(__file__).resolve().parents[3]


def queue_identity(run_id=10):
    value = identity(run_id)
    value.update(
        event='merge_group',
        pr=None,
        ref='refs/heads/gh-readonly-queue/master/pr-23-abcdef',
    )
    value['merge_group'] = {
        'head_sha': value['source_sha'],
        'head_ref': value['ref'],
        'base_sha': 'c' * 40,
        'base_ref': 'refs/heads/master',
    }
    return value


class SharedQueueImagesTest(unittest.TestCase):
    def test_queue_images_resolve_and_verify_both_architectures(self):
        for architecture in ['amd64', 'arm64']:
            with (
                self.subTest(architecture=architecture),
                tempfile.TemporaryDirectory() as directory,
            ):
                directory = Path(directory)
                source = queue_identity()
                consumer = queue_identity(20)
                api = FakeAPI(source)
                api.current_run['head_branch'] = source['ref'].removeprefix(
                    'refs/heads/'
                )
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
assert.deepEqual(architectures({workflows: []}, 'merge_group',
  {merge_group: {base_ref: 'refs/heads/master'}}, []), {amd64: true, arm64: true});
assert.throws(() => architectures({workflows: []}, 'merge_group',
  {merge_group: {base_ref: 'refs/heads/release-2.18'}}, []));
""",
            ],
            cwd=ROOT,
            check=True,
        )


if __name__ == '__main__':
    unittest.main()
