"""Latest aliases follow published backend versions, not release type."""

import json
import subprocess
import unittest
from pathlib import Path
from tempfile import TemporaryDirectory
from unittest import mock

from kfpr import core, steps


class LatestReleaseTest(unittest.TestCase):
    def setUp(self):
        directory = TemporaryDirectory()
        self.addCleanup(directory.cleanup)
        root = Path(directory.name)
        self.runner = mock.Mock(spec=core.CommandRunner, dry_run=False)
        self.runner.run.return_value = subprocess.CompletedProcess([], 1)
        self.context = core.ReleaseContext(
            root,
            core.ReleaseState(root / 'state.json'),
            self.runner,
            core.ReleaseMetadata.from_version('patch', '2.17.3'),
            'origin',
            True,
            True,
        )
        self.context.state.answers['release_sha'] = 'a' * 40
        self.releases('2.17.2')
        for target in (
            'qualification.require_qualification',
            '_ensure_release_tag',
            'watch_latest_workflow_run',
        ):
            patcher = mock.patch('kfpr.steps.' + target)
            patcher.start()
            self.addCleanup(patcher.stop)

    def releases(self, *tags):
        self.runner.capture.return_value = json.dumps(
            [
                [
                    {'tag_name': tag, 'draft': False, 'prerelease': False}
                    for tag in tags
                ]
            ]
        )

    def test_newest_patch_promotes_images_and_backend_release(self):
        steps.step_publish_images(self.context)
        self.assertIn('set_latest=true', self.runner.run.call_args.args[0])
        with (
            mock.patch.object(steps, 'prompt_required', return_value='2.17.2'),
            mock.patch.object(
                steps, '_generate_backend_release_notes', return_value='Fixes'
            ),
        ):
            steps.step_create_backend_release(self.context)
        self.assertIn('--latest=true', self.runner.run.call_args.args[0])

    def test_newer_stable_line_prevents_image_and_backend_downgrade(self):
        self.releases('2.17.2', '2.18.0')
        steps.step_publish_images(self.context)
        self.assertIn('set_latest=false', self.runner.run.call_args.args[0])
        with (
            mock.patch.object(steps, 'prompt_required', return_value='2.17.2'),
            mock.patch.object(
                steps, '_generate_backend_release_notes', return_value='Fixes'
            ),
        ):
            steps.step_create_backend_release(self.context)
        self.assertIn('--latest=false', self.runner.run.call_args.args[0])

    def test_numeric_semver_all_pages_and_ignored_releases(self):
        cases = [
            ('2.9.9', True),
            ('2.17.3', True),
            ('2.17.10', False),
            ('v2.18.0', False),
            ('2.17.3+build.1', True),
        ]
        for tag, expected in cases:
            with self.subTest(tag=tag):
                self.releases(tag)
                self.assertEqual(
                    steps._should_set_backend_latest(self.context), expected
                )
        self.runner.capture.return_value = json.dumps(
            [
                [{'tag_name': '2.17.2', 'draft': False, 'prerelease': False}],
                [
                    {'tag_name': '3.0.0', 'draft': True, 'prerelease': False},
                    {
                        'tag_name': '3.0.0-rc.1',
                        'draft': False,
                        'prerelease': True,
                    },
                    {
                        'tag_name': 'sdk-3.0.0',
                        'draft': False,
                        'prerelease': False,
                    },
                ],
            ]
        )
        self.assertTrue(steps._should_set_backend_latest(self.context))
        self.assertIn('--paginate', self.runner.capture.call_args.args[0])
        self.assertIn('--slurp', self.runner.capture.call_args.args[0])
        self.runner.capture.return_value = '[]'
        self.assertTrue(steps._should_set_backend_latest(self.context))

    def test_malformed_or_unavailable_discovery_prevents_mutations(self):
        for payload in (
            'invalid',
            '{}',
            '[{}]',
            '[[{}]]',
            '[[{"tag_name":"unknown","draft":false,"prerelease":false}]]',
        ):
            with self.subTest(payload=payload):
                self.runner.capture.return_value = payload
                with mock.patch.object(steps, '_ensure_release_tag') as tag:
                    with self.assertRaises((ValueError, RuntimeError)):
                        steps.step_publish_images(self.context)
                    tag.assert_not_called()
                self.runner.run.assert_not_called()
        self.runner.capture.side_effect = RuntimeError('GitHub unavailable')
        with self.assertRaisesRegex(RuntimeError, 'unavailable'):
            steps.step_publish_images(self.context)
        self.runner.run.assert_not_called()

    def test_sdk_never_takes_shared_backend_latest(self):
        self.context.metadata = core.ReleaseMetadata.from_version(
            'minor', '2.18.0'
        )
        with mock.patch.object(
            steps, '_sdk_release_is_consolidated', return_value=False
        ):
            steps.step_create_sdk_release(self.context)
        self.assertIn('--latest=false', self.runner.run.call_args.args[0])
        self.runner.capture.assert_not_called()

    def test_dry_run_does_not_query_github(self):
        self.runner.dry_run = True
        self.assertFalse(steps._should_set_backend_latest(self.context))
        self.runner.capture.assert_not_called()
