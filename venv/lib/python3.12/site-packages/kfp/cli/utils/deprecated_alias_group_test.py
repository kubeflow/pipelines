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
from kfp.cli.utils import deprecated_alias_group


@click.group(
    cls=deprecated_alias_group.deprecated_alias_group_factory(
        {'deprecated': 'new'}))
def cli():
    pass


@cli.command()
def new():
    click.echo('Called new command.')


class TestAliasedPluralsGroup(unittest.TestCase):

    @classmethod
    def setUpClass(cls):
        cls.runner = testing.CliRunner()

    def test_new_call(self):
        result = self.runner.invoke(cli, ['new'])
        self.assertEqual(result.exit_code, 0)
        self.assertEqual(result.output, 'Called new command.\n')

    def test_deprecated_call(self):
        result = self.runner.invoke(cli, ['deprecated'])
        self.assertEqual(result.exit_code, 0)
        self.assertTrue('Called new command.\n' in result.output)
        self.assertTrue(
            "Warning: 'deprecated' is deprecated, use 'new' instead." in
            result.output)


if __name__ == '__main__':
    unittest.main()
