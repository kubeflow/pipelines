#!/usr/bin/env python3
"""Tests for terminal formatting and prompts."""

import contextlib
import io
import unittest
from unittest import mock

from kfpr import core


class OutputStyleTest(unittest.TestCase):
    def test_underline_links_underlines_urls(self):
        self.assertEqual(
            core.underline_links('Open https://github.com/kubeflow/pipelines'),
            'Open \033[4mhttps://github.com/kubeflow/pipelines\033[0m',
        )


class PromptValidationTest(unittest.TestCase):
    def test_prompt_choice_bolds_prompt(self):
        with mock.patch('builtins.input', return_value='y') as input_mock:
            answer = core.prompt_choice(
                'Continue from this checkpoint?', ['y', 'n'], default='n'
            )

        self.assertEqual(answer, 'y')
        self.assertEqual(
            input_mock.call_args.args[0],
            '\033[1mContinue from this checkpoint? (y/n) [n]: \033[0m',
        )

    def test_prompt_numbered_choice_accepts_number(self):
        with (
            contextlib.redirect_stdout(io.StringIO()) as output,
            mock.patch(
                'builtins.input',
                return_value='2',
            ),
        ):
            answer = core.prompt_numbered_choice(
                'Release type', ['major', 'minor', 'patch']
            )

        self.assertEqual(answer, 'minor')
        self.assertIn('1) major', output.getvalue())
        self.assertIn('2) minor', output.getvalue())
        self.assertIn('3) patch', output.getvalue())

    def test_prompt_numbered_choice_reasks_for_invalid_number(self):
        with (
            contextlib.redirect_stdout(io.StringIO()) as output,
            mock.patch(
                'builtins.input',
                side_effect=['4', '3'],
            ),
        ):
            answer = core.prompt_numbered_choice(
                'Release type', ['major', 'minor', 'patch']
            )

        self.assertEqual(answer, 'patch')
        self.assertIn('Choose a number from 1 to 3.', output.getvalue())


if __name__ == '__main__':
    unittest.main()
