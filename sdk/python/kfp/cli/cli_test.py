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
import yaml


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
        matches = re.match(r'^kfp \d+\.\d+\.\d+.*', result.output)
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


class TestKfpDslCompile(unittest.TestCase):

    def invoke(self, args):
        starting_args = ['dsl', 'compile']
        args = starting_args + args
        runner = testing.CliRunner()
        return runner.invoke(
            cli=cli.cli, args=args, catch_exceptions=False, obj={})

    def create_pipeline_file(self):
        pipeline_code = b"""
from kfp import dsl

@dsl.component
def my_component():
    pass

@dsl.pipeline(name="tiny-pipeline")
def my_pipeline():
    my_component_task = my_component()
"""
        temp_pipeline = tempfile.NamedTemporaryFile(suffix='.py', delete=False)
        temp_pipeline.write(pipeline_code)
        temp_pipeline.flush()
        return temp_pipeline

    def load_output_yaml(self, output_file):
        with open(output_file, 'r') as f:
            return yaml.safe_load(f)

    def test_compile_with_caching_flag_enabled(self):
        temp_pipeline = self.create_pipeline_file()
        output_file = 'test_output.yaml'

        result = self.invoke(
            ['--py', temp_pipeline.name, '--output', output_file])
        self.assertEqual(result.exit_code, 0)

        output_data = self.load_output_yaml(output_file)
        self.assertIn('root', output_data)
        self.assertIn('tasks', output_data['root']['dag'])
        for task in output_data['root']['dag']['tasks'].values():
            self.assertIn('cachingOptions', task)
            caching_options = task['cachingOptions']
            self.assertEqual(caching_options.get('enableCache'), True)

    def test_compile_with_caching_flag_disabled(self):
        temp_pipeline = self.create_pipeline_file()
        output_file = 'test_output.yaml'

        result = self.invoke([
            '--py', temp_pipeline.name, '--output', output_file,
            '--disable-execution-caching-by-default'
        ])
        self.assertEqual(result.exit_code, 0)

        output_data = self.load_output_yaml(output_file)
        self.assertIn('root', output_data)
        self.assertIn('tasks', output_data['root']['dag'])
        for task in output_data['root']['dag']['tasks'].values():
            self.assertIn('cachingOptions', task)
            caching_options = task['cachingOptions']
            self.assertEqual(caching_options, {})

    def test_compile_with_caching_disabled_env_var(self):
        temp_pipeline = self.create_pipeline_file()
        output_file = 'test_output.yaml'

        os.environ['KFP_DISABLE_EXECUTION_CACHING_BY_DEFAULT'] = 'true'
        result = self.invoke(
            ['--py', temp_pipeline.name, '--output', output_file])
        self.assertEqual(result.exit_code, 0)
        del os.environ['KFP_DISABLE_EXECUTION_CACHING_BY_DEFAULT']

        output_data = self.load_output_yaml(output_file)
        self.assertIn('root', output_data)
        self.assertIn('tasks', output_data['root']['dag'])
        for task in output_data['root']['dag']['tasks'].values():
            self.assertIn('cachingOptions', task)
            caching_options = task['cachingOptions']
            self.assertEqual(caching_options, {})

    def test_compile_with_kubernetes_manifest_format(self):
        with tempfile.NamedTemporaryFile(suffix='.py', delete=True) as temp_pipeline, \
             tempfile.NamedTemporaryFile(suffix='.yaml', delete=True) as output_file, \
             tempfile.NamedTemporaryFile(suffix='.yaml', delete=True) as output_file2:
            temp_pipeline.write(b"""
from kfp import dsl

@dsl.component
def my_component():
    pass

@dsl.pipeline(name="iris-pipeline")
def iris_pipeline():
    my_component()
""")
            temp_pipeline.flush()
            pipeline_name = 'iris-pipeline'
            pipeline_display_name = 'IrisPipeline'
            pipeline_version_name = 'iris-pipeline-v1'
            pipeline_version_display_name = 'IrisPipelineVersion'
            namespace = 'test-namespace'

            # Test with --kubernetes-manifest-format flag
            result = self.invoke([
                '--py', temp_pipeline.name, '--output', output_file.name,
                '--kubernetes-manifest-format', '--pipeline-display-name',
                pipeline_display_name, '--pipeline-version-name',
                pipeline_version_name, '--pipeline-version-display-name',
                pipeline_version_display_name, '--namespace', namespace,
                '--include-pipeline-manifest'
            ])
            self.assertEqual(result.exit_code, 0)

            with open(output_file.name, 'r') as f:
                docs = list(yaml.safe_load_all(f))
            kinds = [doc['kind'] for doc in docs]
            self.assertIn('Pipeline', kinds)
            self.assertIn('PipelineVersion', kinds)
            pipeline_doc = next(
                doc for doc in docs if doc['kind'] == 'Pipeline')
            pipeline_version_doc = next(
                doc for doc in docs if doc['kind'] == 'PipelineVersion')
            self.assertEqual(pipeline_doc['metadata']['name'], pipeline_name)
            self.assertEqual(pipeline_doc['metadata']['namespace'], namespace)
            self.assertEqual(pipeline_doc['spec']['displayName'],
                             pipeline_display_name)
            self.assertEqual(pipeline_version_doc['metadata']['name'],
                             pipeline_version_name)
            self.assertEqual(pipeline_version_doc['metadata']['namespace'],
                             namespace)
            self.assertEqual(pipeline_version_doc['spec']['displayName'],
                             pipeline_version_display_name)
            self.assertEqual(pipeline_version_doc['spec']['pipelineName'],
                             pipeline_name)

            # include_pipeline_manifest False
            result = self.invoke([
                '--py', temp_pipeline.name, '--output', output_file2.name,
                '--kubernetes-manifest-format', '--pipeline-display-name',
                pipeline_display_name, '--pipeline-version-name',
                pipeline_version_name, '--pipeline-version-display-name',
                pipeline_version_display_name, '--namespace', namespace
            ])
            self.assertEqual(result.exit_code, 0)

            with open(output_file2.name, 'r') as f:
                docs = list(yaml.safe_load_all(f))
            kinds = [doc['kind'] for doc in docs]
            self.assertNotIn('Pipeline', kinds)
            self.assertIn('PipelineVersion', kinds)
            self.assertEqual(len(kinds), 1)
            pipeline_version_doc = docs[0]
            self.assertEqual(pipeline_version_doc['metadata']['name'],
                             pipeline_version_name)
            self.assertEqual(pipeline_version_doc['metadata']['namespace'],
                             namespace)
            self.assertEqual(pipeline_version_doc['spec']['displayName'],
                             pipeline_version_display_name)
            self.assertEqual(pipeline_version_doc['spec']['pipelineName'],
                             pipeline_name)

    def test_compile_manifest_options_without_format_flag(self):
        with tempfile.NamedTemporaryFile(suffix='.py', delete=True) as temp_pipeline, \
             tempfile.NamedTemporaryFile(suffix='.yaml', delete=True) as output_file:
            temp_pipeline.write(b"""
from kfp import dsl

@dsl.component
def my_component():
    pass

@dsl.pipeline(name="iris-pipeline")
def iris_pipeline():
    my_component()
""")
            temp_pipeline.flush()
            pipeline_display_name = 'IrisPipeline'
            pipeline_version_name = 'iris-pipeline-v1'
            pipeline_version_display_name = 'IrisPipelineVersion'
            namespace = 'test-namespace'

            result = self.invoke([
                '--py', temp_pipeline.name, '--output', output_file.name,
                '--pipeline-display-name', pipeline_display_name,
                '--pipeline-version-name', pipeline_version_name,
                '--pipeline-version-display-name',
                pipeline_version_display_name, '--namespace', namespace,
                '--include-pipeline-manifest'
            ])
            self.assertEqual(result.exit_code, 0)
            self.assertIn(
                'Warning: Kubernetes manifest options were provided but --kubernetes-manifest-format was not set',
                result.output)
            # Should only output a regular pipeline spec, not Kubernetes manifests
            with open(output_file.name, 'r') as f:
                doc = yaml.safe_load(f)
            self.assertIn('pipelineInfo', doc)


if __name__ == '__main__':
    unittest.main()
