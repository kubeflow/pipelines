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
from kfp import dsl
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


class TestOverrideCachingOptions(unittest.TestCase):

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
            temp_filepath = os.path.join(tempdir, 'pipeline.yaml')
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
            temp_filepath = os.path.join(tempdir, 'pipeline.yaml')
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


class TestValidatePipelineArguments(unittest.TestCase):

    def test_success1(self):

        @component
        def hello_world(text: str) -> str:
            return text

        @pipeline(name='hello-world')
        def my_pipeline(text: str = 'hi there'):
            hello_world(text=text)

        with tempfile.TemporaryDirectory() as tempdir:
            temp_filepath = os.path.join(tempdir, 'pipeline.yaml')
            Compiler().compile(
                pipeline_func=my_pipeline, package_path=temp_filepath)

            with open(temp_filepath, 'r') as f:
                pipeline_obj = yaml.safe_load(f)

        client._validate_pipeline_arguments(pipeline_obj, {})

    def test_success2(self):

        @component
        def hello_world(text: str) -> str:
            return text

        @pipeline(name='hello-world')
        def my_pipeline(text: str):
            hello_world(text=text)

        with tempfile.TemporaryDirectory() as tempdir:
            temp_filepath = os.path.join(tempdir, 'pipeline.yaml')
            Compiler().compile(
                pipeline_func=my_pipeline, package_path=temp_filepath)

            with open(temp_filepath, 'r') as f:
                pipeline_obj = yaml.safe_load(f)

        client._validate_pipeline_arguments(pipeline_obj, {'text': 'hi there'})

    def test_success3(self):

        from typing import Dict, List

        from kfp.dsl import component
        from kfp.dsl import Dataset
        from kfp.dsl import Output
        from kfp.dsl import OutputPath

        @component
        def preprocess(
            # An input parameter of type string.
            message: str,
            # An input parameter of type dict.
            input_dict_parameter: Dict[str, int],
            # An input parameter of type list.
            input_list_parameter: List[str],
            # Use Output[T] to get a metadata-rich handle to the output artifact
            # of type `Dataset`.
            output_dataset_one: Output[Dataset],
            # A locally accessible filepath for another output artifact of type
            # `Dataset`.
            output_dataset_two_path: OutputPath('Dataset'),
            # A locally accessible filepath for an output parameter of type string.
            output_parameter_path: OutputPath(str),
            # A locally accessible filepath for an output parameter of type bool.
            output_bool_parameter_path: OutputPath(bool),
            # A locally accessible filepath for an output parameter of type dict.
            output_dict_parameter_path: OutputPath(Dict[str, int]),
            # A locally accessible filepath for an output parameter of type list.
            output_list_parameter_path: OutputPath(List[str]),
        ):
            """Dummy preprocessing step."""

            # Use Dataset.path to access a local file path for writing.
            # One can also use Dataset.uri to access the actual URI file path.
            with open(output_dataset_one.path, 'w') as f:
                f.write(message)

            # OutputPath is used to just pass the local file path of the output artifact
            # to the function.
            with open(output_dataset_two_path, 'w') as f:
                f.write(message)

            with open(output_parameter_path, 'w') as f:
                f.write(message)

            with open(output_bool_parameter_path, 'w') as f:
                f.write(
                    str(True)
                )  # use either `str()` or `json.dumps()` for bool values.

            import json
            with open(output_dict_parameter_path, 'w') as f:
                f.write(json.dumps(input_dict_parameter))

            with open(output_list_parameter_path, 'w') as f:
                f.write(json.dumps(input_list_parameter))

        with tempfile.TemporaryDirectory() as tempdir:
            temp_filepath = os.path.join(tempdir, 'pipeline.yaml')
            Compiler().compile(
                pipeline_func=preprocess, package_path=temp_filepath)

            with open(temp_filepath, 'r') as f:
                pipeline_obj = yaml.safe_load(f)

        client._validate_pipeline_arguments(
            pipeline_obj, {
                'message': 'message',
                'input_dict_parameter': {
                    'key': 'value'
                },
                'input_list_parameter': [1, 2, 3]
            })

    def test_success4(self):

        @component
        def hello_world(text: str) -> str:
            return text

        @pipeline(name='hello-world')
        def my_pipeline():
            hello_world(text='text')

        with tempfile.TemporaryDirectory() as tempdir:
            temp_filepath = os.path.join(tempdir, 'pipeline.yaml')
            Compiler().compile(
                pipeline_func=my_pipeline, package_path=temp_filepath)

            with open(temp_filepath, 'r') as f:
                pipeline_obj = yaml.safe_load(f)

        client._validate_pipeline_arguments(pipeline_obj, {})

    def test_failure1(self):

        @component
        def hello_world(text: str) -> str:
            return text

        @pipeline(name='hello-world')
        def my_pipeline(text: str):
            hello_world(text=text)

        with tempfile.TemporaryDirectory() as tempdir:
            temp_filepath = os.path.join(tempdir, 'pipeline.yaml')
            Compiler().compile(
                pipeline_func=my_pipeline, package_path=temp_filepath)

            with open(temp_filepath, 'r') as f:
                pipeline_obj = yaml.safe_load(f)

        with self.assertRaisesRegex(
                ValueError,
                r"Cannot run pipeline. Pipeline submission is missing the following arguments: {'text'}"
        ):
            client._validate_pipeline_arguments(pipeline_obj, {})

    def test_failure2(self):

        @component
        def hello_world(text: str) -> str:
            return text

        @pipeline(name='hello-world')
        def my_pipeline(text: str, integer: int = 1):
            hello_world(text=text)

        with tempfile.TemporaryDirectory() as tempdir:
            temp_filepath = os.path.join(tempdir, 'pipeline.yaml')
            Compiler().compile(
                pipeline_func=my_pipeline, package_path=temp_filepath)

            with open(temp_filepath, 'r') as f:
                pipeline_obj = yaml.safe_load(f)

        with self.assertRaisesRegex(
                ValueError,
                r"Cannot run pipeline. Pipeline submission is missing the following arguments: {'text'}"
        ):
            client._validate_pipeline_arguments(pipeline_obj, {})

    def test_failure3(self):

        @component
        def hello_world(text: str) -> str:
            return text

        @pipeline(name='hello-world')
        def my_pipeline(text: str, integer: int):
            hello_world(text=text)

        with tempfile.TemporaryDirectory() as tempdir:
            temp_filepath = os.path.join(tempdir, 'pipeline.yaml')
            Compiler().compile(
                pipeline_func=my_pipeline, package_path=temp_filepath)

            with open(temp_filepath, 'r') as f:
                pipeline_obj = yaml.safe_load(f)

        with self.assertRaisesRegex(
                ValueError,
                r"Cannot run pipeline. Pipeline submission is missing the following arguments: {'text'}"
        ):
            client._validate_pipeline_arguments(pipeline_obj, {'integer': 10})

    def test_failure4(self):

        @component
        def hello_world(text: str) -> str:
            return text

        @pipeline(name='hello-world')
        def my_pipeline(text: str, integer: int,
                        artifact: dsl.Input[dsl.Artifact]):
            hello_world(text=text)

        with tempfile.TemporaryDirectory() as tempdir:
            temp_filepath = os.path.join(tempdir, 'pipeline.yaml')
            Compiler().compile(
                pipeline_func=my_pipeline, package_path=temp_filepath)

            with open(temp_filepath, 'r') as f:
                pipeline_obj = yaml.safe_load(f)

        with self.assertRaisesRegex(
                ValueError,
                r'Cannot run pipeline. Pipeline submission is missing the following arguments:'
        ):
            client._validate_pipeline_arguments(pipeline_obj, {'integer': 10})


if __name__ == '__main__':
    unittest.main()
