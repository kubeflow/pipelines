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

import os
import tempfile
import unittest

from absl.testing import parameterized
from kfp.client import client
from kfp.compiler import Compiler
from kfp.dsl import component
from kfp.dsl import pipeline
import yaml


class TestValidatePipelineName(parameterized.TestCase):

    @parameterized.parameters([
        'pipeline',
        'my-pipeline',
        'my-pipeline-1',
        '1pipeline',
        'pipeline1',
    ])
    def test_valid(self, name: str):
        client.validate_pipeline_resource_name(name)

    @parameterized.parameters([
        'my_pipeline',
        "person's-pipeline",
        'my pipeline',
        'pipeline.yaml',
    ])
    def test_invalid(self, name: str):
        with self.assertRaisesRegex(ValueError, r'Invalid pipeline name:'):
            client.validate_pipeline_resource_name(name)


class TestOverrideCachingOptions(parameterized.TestCase):

    def test_override_caching_from_pipeline(self):

        @component
        def hello_world(text: str) -> str:
            """Hello world component."""
            return text

        @pipeline(name='hello-world', description='A simple intro pipeline')
        def pipeline_hello_world(text: str = 'hi there'):
            """Hello world pipeline."""

            hello_world(text=text).set_caching_options(True)

        with tempfile.TemporaryDirectory() as tempdir:
            temp_filepath = os.path.join(tempdir, 'hello_world_pipeline.yaml')
            Compiler().compile(
                pipeline_func=pipeline_hello_world, package_path=temp_filepath)

            with open(temp_filepath, 'r') as f:
                pipeline_obj = yaml.safe_load(f)
                test_client = client.Client(namespace='dummy_namespace')
                test_client._override_caching_options(pipeline_obj, False)
                for _, task in pipeline_obj['root']['dag']['tasks'].items():
                    self.assertFalse(task['cachingOptions']['enableCache'])

    def test_override_caching_of_multiple_components(self):

        @component
        def hello_word(text: str) -> str:
            return text

        @component
        def to_lower(text: str) -> str:
            return text.lower()

        @pipeline(
            name='sample two-step pipeline',
            description='a minimal two-step pipeline')
        def pipeline_with_two_component(text: str = 'hi there'):

            component_1 = hello_word(text=text).set_caching_options(True)
            component_2 = to_lower(
                text=component_1.output).set_caching_options(True)

        with tempfile.TemporaryDirectory() as tempdir:
            temp_filepath = os.path.join(tempdir, 'hello_world_pipeline.yaml')
            Compiler().compile(
                pipeline_func=pipeline_with_two_component,
                package_path=temp_filepath)

            with open(temp_filepath, 'r') as f:
                pipeline_obj = yaml.safe_load(f)
                test_client = client.Client(namespace='dummy_namespace')
                test_client._override_caching_options(pipeline_obj, False)
                self.assertFalse(
                    pipeline_obj['root']['dag']['tasks']['hello-word']
                    ['cachingOptions']['enableCache'])
                self.assertFalse(pipeline_obj['root']['dag']['tasks']
                                 ['to-lower']['cachingOptions']['enableCache'])


if __name__ == '__main__':
    unittest.main()
