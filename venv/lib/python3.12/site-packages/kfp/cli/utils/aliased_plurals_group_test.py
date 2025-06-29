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

import unittest

import click
from click import testing
from kfp.cli.utils import aliased_plurals_group


@click.group(cls=aliased_plurals_group.AliasedPluralsGroup)
def cli():
    pass


@cli.command()
def command():
    click.echo('Called command.')


class TestAliasedPluralsGroup(unittest.TestCase):

    @classmethod
    def setUpClass(cls):
        cls.runner = testing.CliRunner()

    def test_aliases_default_success(self):
        result = self.runner.invoke(cli, ['command'])
        self.assertEqual(result.exit_code, 0)
        self.assertEqual(result.output, 'Called command.\n')

    def test_aliases_plural_success(self):
        result = self.runner.invoke(cli, ['commands'])
        self.assertEqual(result.exit_code, 0)
        self.assertEqual(result.output, 'Called command.\n')

    def test_aliases_failure(self):
        result = self.runner.invoke(cli, ['commandss'])
        self.assertEqual(result.exit_code, 2)
        self.assertEqual("Error: Unrecognized command 'commandss'\n",
                         result.output)


if __name__ == '__main__':
    unittest.main()
