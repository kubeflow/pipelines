# Copyright 2022 The Kubeflow Authors
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

import functools
import itertools
import os
import re
import subprocess
import tempfile
from typing import List
import unittest
from unittest import mock

from absl.testing import parameterized
import click
from click import testing
from kfp.cli import cli
from kfp.cli import compile_


class TestCliNounAliases(unittest.TestCase):

    def setUp(self):
        runner = testing.CliRunner()
        self.invoke = functools.partial(
            runner.invoke, cli=cli.cli, catch_exceptions=False, obj={})

    def test_aliases_singular(self):
        result = self.invoke(args=['component'])
        self.assertEqual(result.exit_code, 0)

    def test_aliases_plural(self):
        result = self.invoke(args=['components'])
        self.assertEqual(result.exit_code, 0)

    def test_aliases_fails(self):
        result = self.invoke(args=['componentss'])
        self.assertEqual(result.exit_code, 2)
        self.assertEqual("Error: Unrecognized command 'componentss'\n",
                         result.output)


class TestAliases(unittest.TestCase):

    @classmethod
    def setUpClass(cls):
        runner = testing.CliRunner()
        cls.invoke = functools.partial(
            runner.invoke, cli=cli.cli, catch_exceptions=True, obj={})

    def test_aliases_singular(self):
        result = self.invoke(args=['component'])
        self.assertEqual(result.exit_code, 0)

    def test_aliases_plural(self):
        result = self.invoke(args=['components'])
        self.assertEqual(result.exit_code, 0)

    def test_aliases_fails(self):
        result = self.invoke(args=['componentss'])
        self.assertEqual(result.exit_code, 2)
        self.assertEqual("Error: Unrecognized command 'componentss'\n",
                         result.output)


class TestCliAutocomplete(parameterized.TestCase):

    def setUp(self):
        runner = testing.CliRunner()
        self.invoke = functools.partial(
            runner.invoke, cli=cli.cli, catch_exceptions=False, obj={})

    @parameterized.parameters(['bash', 'zsh', 'fish'])
    def test_show_autocomplete(self, shell):
        result = self.invoke(args=['--show-completion', shell])
        expected = cli._create_completion(shell)
        self.assertIn(expected, result.output)
        self.assertEqual(result.exit_code, 0)

    @parameterized.parameters(['bash', 'zsh', 'fish'])
    def test_install_autocomplete_with_empty_file(self, shell):
        with tempfile.TemporaryDirectory() as tempdir:
            with mock.patch('os.path.expanduser', return_value=tempdir):
                temp_path = os.path.join(tempdir, *cli.SHELL_FILES[shell])
                os.makedirs(os.path.dirname(temp_path), exist_ok=True)

                result = self.invoke(args=['--install-completion', shell])
                expected = cli._create_completion(shell)

                with open(temp_path) as f:
                    last_line = f.readlines()[-1]
                self.assertEqual(expected + '\n', last_line)
                self.assertEqual(result.exit_code, 0)

    @parameterized.parameters(
        list(itertools.product(['bash', 'zsh', 'fish'], [True, False])))
    def test_install_autocomplete_with_unempty_file(self, shell,
                                                    has_trailing_newline):
        with tempfile.TemporaryDirectory() as tempdir:
            with mock.patch('os.path.expanduser', return_value=tempdir):
                temp_path = os.path.join(tempdir, *cli.SHELL_FILES[shell])
                os.makedirs(os.path.dirname(temp_path), exist_ok=True)

                existing_file_contents = [
                    'something\n',
                    'something else' + ('\n' if has_trailing_newline else ''),
                ]
                with open(temp_path, 'w') as f:
                    f.writelines(existing_file_contents)

                result = self.invoke(args=['--install-completion', shell])
                expected = cli._create_completion(shell)

                with open(temp_path) as f:
                    last_line = f.readlines()[-1]
                self.assertEqual(expected + '\n', last_line)
                self.assertEqual(result.exit_code, 0)


class TestCliVersion(unittest.TestCase):

    def setUp(self):
        runner = testing.CliRunner()
        self.invoke = functools.partial(
            runner.invoke, cli=cli.cli, catch_exceptions=False, obj={})

    def test_version(self):
        result = self.invoke(args=['--version'])
        self.assertEqual(result.exit_code, 0)
        matches = re.match(r'^kfp \d\.\d\.\d.*', result.output)
        self.assertTrue(matches)


class TestDslCompile(parameterized.TestCase):

    def invoke(self, args: List[str]) -> testing.Result:
        starting_args = ['dsl', 'compile']
        args = starting_args + args
        runner = testing.CliRunner()
        return runner.invoke(
            cli=cli.cli, args=args, catch_exceptions=False, obj={})

    def invoke_deprecated(self, args: List[str]) -> testing.Result:
        runner = testing.CliRunner()
        return runner.invoke(
            cli=compile_.compile_, args=args, catch_exceptions=False, obj={})

    def test_deprecated_command_is_found(self):
        result = self.invoke_deprecated(['--help'])
        self.assertEqual(result.exit_code, 0)

    def test_deprecation_warning(self):
        res = subprocess.run(['dsl-compile', '--help'], capture_output=True)
        self.assertIn('Deprecated. Please use `kfp dsl compile` instead.)',
                      res.stdout.decode('utf-8'))


info_dict = cli.cli.to_info_dict(ctx=click.Context(cli.cli))
commands_dict = {
    command: list(body.get('commands', {}).keys())
    for command, body in info_dict['commands'].items()
}
noun_verb_list = [
    (noun, verb) for noun, verbs in commands_dict.items() for verb in verbs
]


class TestSmokeTestAllCommandsWithHelp(parameterized.TestCase):

    @classmethod
    def setUpClass(cls):
        cls.runner = testing.CliRunner()

        cls.vals = [('run', 'list')]

    @parameterized.parameters(*noun_verb_list)
    def test(self, noun: str, verb: str):
        with mock.patch('kfp.cli.cli.client.Client'):
            result = self.runner.invoke(
                args=[noun, verb, '--help'],
                cli=cli.cli,
                catch_exceptions=False,
                obj={})
            self.assertTrue(result.output.startswith('Usage: '))
            self.assertEqual(result.exit_code, 0)


if __name__ == '__main__':
    unittest.main()
