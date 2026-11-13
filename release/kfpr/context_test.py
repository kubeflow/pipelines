#!/usr/bin/env python3
"""Tests for release context assembly."""

import contextlib
import io
import unittest
from pathlib import Path
from tempfile import TemporaryDirectory
from unittest import mock

from kfpr import core
from kfpr.core import ReleaseState


class CollectContextTest(unittest.TestCase):
    def test_collect_context_reasks_for_invalid_version_and_fork_remote(self):
        with TemporaryDirectory() as tmpdir:
            state = ReleaseState(Path(tmpdir) / 'state.json')
            args = type('Args', (), {'dry_run': True})()

            with (
                contextlib.redirect_stdout(io.StringIO()),
                mock.patch(
                    'builtins.input',
                    side_effect=[
                        '2',
                        '3.2',
                        '3.2.0',
                        '@droctothorpe',
                        'droctothorpe',
                        '3.1.1',
                    ],
                ),
            ):
                context = core.collect_context(args, state)

            self.assertEqual(context.metadata.tag, '3.2.0')
            self.assertEqual(
                context.fork_remote,
                'https://github.com/droctothorpe/pipelines.git',
            )

    def test_collect_context_prompts_for_major_minor_source_branch(self):
        with TemporaryDirectory() as tmpdir:
            state = ReleaseState(Path(tmpdir) / 'state.json')
            args = type(
                'Args',
                (),
                {'dry_run': True, 'prompt_release_source_branch': True},
            )()

            with (
                contextlib.redirect_stdout(io.StringIO()),
                mock.patch(
                    'builtins.input',
                    side_effect=[
                        '2',
                        '3.2.0',
                        'droctothorpe',
                        'release-candidate',
                        '3.1.1',
                    ],
                ),
            ):
                core.collect_context(args, state)

            self.assertEqual(
                state.answers['release_source_branch'], 'release-candidate'
            )
            self.assertEqual(state.answers['previous_release'], '3.1.1')

    def test_collect_context_uses_release_source_branch_arg_without_prompt(
        self,
    ):
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

            with (
                contextlib.redirect_stdout(io.StringIO()),
                mock.patch(
                    'builtins.input',
                    side_effect=['2', '3.2.0', 'droctothorpe', '3.1.1'],
                ),
            ):
                core.collect_context(args, state)

            self.assertEqual(
                state.answers['release_source_branch'], 'release-candidate'
            )

    def test_collect_context_defers_blank_source_to_repository_default(self):
        with TemporaryDirectory() as tmpdir:
            state = ReleaseState(Path(tmpdir) / 'state.json')
            args = type(
                'Args',
                (),
                {'dry_run': True, 'prompt_release_source_branch': True},
            )()

            with (
                contextlib.redirect_stdout(io.StringIO()),
                mock.patch(
                    'builtins.input',
                    side_effect=['2', '3.2.0', 'droctothorpe', '', '3.1.1'],
                ),
            ):
                core.collect_context(args, state)

            self.assertNotIn('release_source_branch', state.answers)


if __name__ == '__main__':
    unittest.main()
