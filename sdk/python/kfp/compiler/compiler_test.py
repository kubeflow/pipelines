# Copyright 2020-2022 The Kubeflow Authors
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

import json
import os
import re
import subprocess
import tempfile
from typing import Any, Dict, List, Optional
import unittest

from absl.testing import parameterized
from click import testing
from google.protobuf import json_format
from kfp import components
from kfp import dsl
from kfp.cli import cli
from kfp.compiler import compiler
from kfp.components.types import type_utils
from kfp.dsl import PipelineTaskFinalStatus
from kfp.pipeline_spec import pipeline_spec_pb2
import yaml

VALID_PRODUCER_COMPONENT_SAMPLE = components.load_component_from_text("""
    name: producer
    inputs:
    - {name: input_param, type: String}
    outputs:
    - {name: output_model, type: Model}
    - {name: output_value, type: Integer}
    implementation:
      container:
        image: gcr.io/my-project/my-image:tag
        args:
        - {inputValue: input_param}
        - {outputPath: output_model}
        - {outputPath: output_value}
    """)


class TestCompilePipeline(parameterized.TestCase):

    def test_compile_simple_pipeline(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            producer_op = components.load_component_from_text("""
      name: producer
      inputs:
      - {name: input_param, type: String}
      outputs:
      - {name: output_model, type: Model}
      - {name: output_value, type: Integer}
      implementation:
        container:
          image: gcr.io/my-project/my-image:tag
          args:
          - {inputValue: input_param}
          - {outputPath: output_model}
          - {outputPath: output_value}
      """)

            consumer_op = components.load_component_from_text("""
      name: consumer
      inputs:
      - {name: input_model, type: Model}
      - {name: input_value, type: Integer}
      implementation:
        container:
          image: gcr.io/my-project/my-image:tag
          args:
          - {inputPath: input_model}
          - {inputValue: input_value}
      """)

            @dsl.pipeline(name='test-pipeline')
            def simple_pipeline(pipeline_input: str = 'Hello KFP!'):
                producer = producer_op(input_param=pipeline_input)
                consumer = consumer_op(
                    input_model=producer.outputs['output_model'],
                    input_value=producer.outputs['output_value'])

            target_file = os.path.join(tmpdir, 'result.yaml')

            compiler.Compiler().compile(
                pipeline_func=simple_pipeline, package_path=target_file)

            self.assertTrue(os.path.exists(target_file))
            with open(target_file, 'r') as f:
                f.read()

    def test_compile_pipeline_with_bool(self):

        with tempfile.TemporaryDirectory() as tmpdir:
            predict_op = components.load_component_from_text("""
      name: predict
      inputs:
      - {name: generate_explanation, type: Boolean, default: False}
      implementation:
        container:
          image: gcr.io/my-project/my-image:tag
          args:
          - {inputValue: generate_explanation}
      """)

            @dsl.pipeline(name='test-boolean-pipeline')
            def simple_pipeline():
                predict_op(generate_explanation=True)

            target_json_file = os.path.join(tmpdir, 'result.yaml')
            compiler.Compiler().compile(
                pipeline_func=simple_pipeline, package_path=target_json_file)

            self.assertTrue(os.path.exists(target_json_file))
            with open(target_json_file, 'r') as f:
                f.read()

    def test_compile_pipeline_with_dsl_graph_component_should_raise_error(self):

        with self.assertRaisesRegex(
                AttributeError,
                "module 'kfp.dsl' has no attribute 'graph_component'"):

            @dsl.graph_component
            def flip_coin_graph_component():
                flip = flip_coin_op()
                with dsl.Condition(flip.output == 'heads'):
                    flip_coin_graph_component()

    def test_compile_pipeline_with_misused_inputvalue_should_raise_error(self):

        upstream_op = components.load_component_from_text("""
        name: upstream compoent
        outputs:
        - {name: model, type: Model}
        implementation:
          container:
            image: dummy
            args:
            - {outputPath: model}
        """)
        downstream_op = components.load_component_from_text("""
        name: compoent with misused placeholder
        inputs:
        - {name: model, type: Model}
        implementation:
          container:
            image: dummy
            args:
            - {inputValue: model}
        """)

        @dsl.pipeline(name='test-pipeline', pipeline_root='dummy_root')
        def my_pipeline():
            downstream_op(model=upstream_op().output)

        with self.assertRaisesRegex(
                TypeError,
                ' type "Model" cannot be paired with InputValuePlaceholder.'):
            compiler.Compiler().compile(
                pipeline_func=my_pipeline, package_path='output.yaml')

    def test_compile_pipeline_with_misused_inputpath_should_raise_error(self):

        component_op = components.load_component_from_text("""
        name: compoent with misused placeholder
        inputs:
        - {name: text, type: String}
        implementation:
          container:
            image: dummy
            args:
            - {inputPath: text}
        """)

        @dsl.pipeline(name='test-pipeline', pipeline_root='dummy_root')
        def my_pipeline(text: str):
            component_op(text=text)

        with self.assertRaisesRegex(
                TypeError,
                ' type "String" cannot be paired with InputPathPlaceholder.'):
            compiler.Compiler().compile(
                pipeline_func=my_pipeline, package_path='output.yaml')

    def test_compile_pipeline_with_missing_task_should_raise_error(self):

        @dsl.pipeline(name='test-pipeline', pipeline_root='dummy_root')
        def my_pipeline(text: str):
            pass

        with self.assertRaisesRegex(ValueError,
                                    'Task is missing from pipeline.'):
            compiler.Compiler().compile(
                pipeline_func=my_pipeline, package_path='output.yaml')

    def test_compile_pipeline_with_misused_inputuri_should_raise_error(self):

        component_op = components.load_component_from_text("""
        name: compoent with misused placeholder
        inputs:
        - {name: value, type: Float}
        implementation:
          container:
            image: dummy
            args:
            - {inputUri: value}
        """)

        @dsl.pipeline(name='test-pipeline', pipeline_root='dummy_root')
        def my_pipeline(value: float):
            component_op(value=value)

        with self.assertRaisesRegex(
                TypeError,
                ' type "Float" cannot be paired with InputUriPlaceholder.'):
            compiler.Compiler().compile(
                pipeline_func=my_pipeline, package_path='output.yaml')

    def test_compile_pipeline_with_misused_outputuri_should_raise_error(self):

        component_op = components.load_component_from_text("""
        name: compoent with misused placeholder
        outputs:
        - {name: value, type: Integer}
        implementation:
          container:
            image: dummy
            args:
            - {outputUri: value}
        """)

        @dsl.pipeline(name='test-pipeline', pipeline_root='dummy_root')
        def my_pipeline():
            component_op()

        with self.assertRaisesRegex(
                TypeError,
                ' type "Integer" cannot be paired with OutputUriPlaceholder.'):
            compiler.Compiler().compile(
                pipeline_func=my_pipeline, package_path='output.yaml')

    def test_compile_pipeline_with_invalid_name_should_raise_error(self):

        @dsl.pipeline(name='')
        def my_pipeline():
            VALID_PRODUCER_COMPONENT_SAMPLE(input_param='input')

        with tempfile.TemporaryDirectory() as tmpdir:
            output_path = os.path.join(tmpdir, 'output.yaml')

            compiler.Compiler().compile(
                pipeline_func=my_pipeline, package_path=output_path)

    def test_set_pipeline_root_through_pipeline_decorator(self):

        with tempfile.TemporaryDirectory() as tmpdir:

            @dsl.pipeline(name='test-pipeline', pipeline_root='gs://path')
            def my_pipeline():
                VALID_PRODUCER_COMPONENT_SAMPLE(input_param='input')

            target_json_file = os.path.join(tmpdir, 'result.yaml')
            compiler.Compiler().compile(
                pipeline_func=my_pipeline, package_path=target_json_file)

            self.assertTrue(os.path.exists(target_json_file))
            with open(target_json_file) as f:
                pipeline_spec = yaml.load(f)
            self.assertEqual('gs://path', pipeline_spec['defaultPipelineRoot'])

    def test_passing_string_parameter_to_artifact_should_error(self):

        component_op = components.load_component_from_text("""
      name: compoent
      inputs:
      - {name: some_input, type: , description: an uptyped input}
      implementation:
        container:
          image: dummy
          args:
          - {inputPath: some_input}
      """)

        @dsl.pipeline(name='test-pipeline', pipeline_root='gs://path')
        def my_pipeline(input1: str):
            component_op(some_input=input1)

        with self.assertRaisesRegex(
                type_utils.InconsistentTypeException,
                'Incompatible argument passed to the input "some_input" of '
                'component "compoent": Argument type "STRING" is incompatible '
                'with the input type "Artifact"'):
            compiler.Compiler().compile(
                pipeline_func=my_pipeline, package_path='output.yaml')

    def test_passing_missing_type_annotation_on_pipeline_input_should_error(
            self):

        @dsl.pipeline(name='test-pipeline', pipeline_root='gs://path')
        def my_pipeline(input1):
            pass

        with self.assertRaisesRegex(
                TypeError, 'Missing type annotation for argument: input1'):
            compiler.Compiler().compile(
                pipeline_func=my_pipeline, package_path='output.yaml')

    def test_passing_generic_artifact_to_input_expecting_concrete_artifact(
            self):

        producer_op1 = components.load_component_from_text("""
      name: producer compoent
      outputs:
      - {name: output, type: Artifact}
      implementation:
        container:
          image: dummy
          args:
          - {outputPath: output}
      """)

        @dsl.component
        def producer_op2(output: dsl.Output[dsl.Artifact]):
            pass

        consumer_op1 = components.load_component_from_text("""
      name: consumer compoent
      inputs:
      - {name: input1, type: MyDataset}
      implementation:
        container:
          image: dummy
          args:
          - {inputPath: input1}
      """)

        @dsl.component
        def consumer_op2(input1: dsl.Input[dsl.Dataset]):
            pass

        @dsl.pipeline(name='test-pipeline')
        def my_pipeline():
            consumer_op1(input1=producer_op1().output)
            consumer_op1(input1=producer_op2().output)
            consumer_op2(input1=producer_op1().output)
            consumer_op2(input1=producer_op2().output)

        with tempfile.TemporaryDirectory() as tmpdir:
            target_yaml_file = os.path.join(tmpdir, 'result.yaml')
            compiler.Compiler().compile(
                pipeline_func=my_pipeline, package_path=target_yaml_file)

            self.assertTrue(os.path.exists(target_yaml_file))

    def test_passing_concrete_artifact_to_input_expecting_generic_artifact(
            self):

        producer_op1 = components.load_component_from_text("""
      name: producer compoent
      outputs:
      - {name: output, type: Dataset}
      implementation:
        container:
          image: dummy
          args:
          - {outputPath: output}
      """)

        @dsl.component
        def producer_op2(output: dsl.Output[dsl.Model]):
            pass

        consumer_op1 = components.load_component_from_text("""
      name: consumer compoent
      inputs:
      - {name: input1, type: Artifact}
      implementation:
        container:
          image: dummy
          args:
          - {inputPath: input1}
      """)

        @dsl.component
        def consumer_op2(input1: dsl.Input[dsl.Artifact]):
            pass

        @dsl.pipeline(name='test-pipeline')
        def my_pipeline():
            consumer_op1(input1=producer_op1().output)
            consumer_op1(input1=producer_op2().output)
            consumer_op2(input1=producer_op1().output)
            consumer_op2(input1=producer_op2().output)

        with tempfile.TemporaryDirectory() as tmpdir:
            target_yaml_file = os.path.join(tmpdir, 'result.yaml')
            compiler.Compiler().compile(
                pipeline_func=my_pipeline, package_path=target_yaml_file)

            self.assertTrue(os.path.exists(target_yaml_file))

    def test_passing_arbitrary_artifact_to_input_expecting_concrete_artifact(
            self):

        producer_op1 = components.load_component_from_text("""
      name: producer compoent
      outputs:
      - {name: output, type: SomeArbitraryType}
      implementation:
        container:
          image: dummy
          args:
          - {outputPath: output}
      """)

        @dsl.component
        def consumer_op(input1: dsl.Input[dsl.Dataset]):
            pass

        @dsl.pipeline(name='test-pipeline')
        def my_pipeline():
            consumer_op(input1=producer_op1().output)
            consumer_op(input1=producer_op2().output)

        with self.assertRaisesRegex(
                type_utils.InconsistentTypeException,
                'Incompatible argument passed to the input "input1" of component'
                ' "consumer-op": Argument type "SomeArbitraryType" is'
                ' incompatible with the input type "Dataset"'):
            compiler.Compiler().compile(
                pipeline_func=my_pipeline, package_path='result.yaml')

    def test_invalid_after_dependency(self):

        @dsl.component
        def producer_op() -> str:
            return 'a'

        @dsl.component
        def dummy_op(msg: str = ''):
            pass

        @dsl.pipeline(name='test-pipeline')
        def my_pipeline(text: str):
            with dsl.Condition(text == 'a'):
                producer_task = producer_op()

            dummy_op().after(producer_task)

        with self.assertRaisesRegex(
                RuntimeError,
                'Task dummy-op cannot dependent on any task inside the group:'):
            compiler.Compiler().compile(
                pipeline_func=my_pipeline, package_path='result.yaml')

    def test_invalid_data_dependency(self):

        @dsl.component
        def producer_op() -> str:
            return 'a'

        @dsl.component
        def dummy_op(msg: str = ''):
            pass

        @dsl.pipeline(name='test-pipeline')
        def my_pipeline(text: bool):
            with dsl.ParallelFor(['a, b']):
                producer_task = producer_op()

            dummy_op(msg=producer_task.output)

        with self.assertRaisesRegex(
                RuntimeError,
                'Task dummy-op cannot dependent on any task inside the group:'):
            compiler.Compiler().compile(
                pipeline_func=my_pipeline, package_path='result.yaml')

    def test_use_task_final_status_in_non_exit_op(self):

        @dsl.component
        def print_op(status: PipelineTaskFinalStatus):
            return status

        @dsl.pipeline(name='test-pipeline')
        def my_pipeline(text: bool):
            print_op()

        with self.assertRaisesRegex(
                ValueError,
                'PipelineTaskFinalStatus can only be used in an exit task.'):
            compiler.Compiler().compile(
                pipeline_func=my_pipeline, package_path='result.yaml')

    def test_use_task_final_status_in_non_exit_op_yaml(self):

        print_op = components.load_component_from_text("""
name: Print Op
inputs:
- {name: message, type: PipelineTaskFinalStatus}
implementation:
  container:
    image: python:3.7
    command:
    - echo
    - {inputValue: message}
""")

        @dsl.pipeline(name='test-pipeline')
        def my_pipeline(text: bool):
            print_op()

        with self.assertRaisesRegex(
                ValueError,
                'PipelineTaskFinalStatus can only be used in an exit task.'):
            compiler.Compiler().compile(
                pipeline_func=my_pipeline, package_path='result.yaml')


# pylint: disable=import-outside-toplevel,unused-import,import-error,redefined-outer-name,reimported
class V2NamespaceAliasTest(unittest.TestCase):
    """Test that imports of both modules and objects are aliased (e.g. all
    import path variants work)."""

    # Note: The DeprecationWarning is only raised on the first import where
    # the kfp.v2 module is loaded. Due to the way we run tests in CI/CD, we cannot ensure that the kfp.v2 module will first be loaded in these tests,
    # so we do not test for the DeprecationWarning here.

    def test_import_namespace(self):  # pylint: disable=no-self-use
        from kfp import v2

        @v2.dsl.component
        def hello_world(text: str) -> str:
            """Hello world component."""
            return text

        @v2.dsl.pipeline(
            name='hello-world', description='A simple intro pipeline')
        def pipeline_hello_world(text: str = 'hi there'):
            """Hello world pipeline."""

            hello_world(text=text)

        with tempfile.TemporaryDirectory() as tempdir:
            # you can e.g. create a file here:
            temp_filepath = os.path.join(tempdir, 'hello_world_pipeline.yaml')
            v2.compiler.Compiler().compile(
                pipeline_func=pipeline_hello_world, package_path=temp_filepath)

            with open(temp_filepath, 'r') as f:
                yaml.load(f)

    def test_import_modules(self):  # pylint: disable=no-self-use
        from kfp.v2 import compiler
        from kfp.v2 import dsl

        @dsl.component
        def hello_world(text: str) -> str:
            """Hello world component."""
            return text

        @dsl.pipeline(name='hello-world', description='A simple intro pipeline')
        def pipeline_hello_world(text: str = 'hi there'):
            """Hello world pipeline."""

            hello_world(text=text)

        with tempfile.TemporaryDirectory() as tempdir:
            # you can e.g. create a file here:
            temp_filepath = os.path.join(tempdir, 'hello_world_pipeline.yaml')
            compiler.Compiler().compile(
                pipeline_func=pipeline_hello_world, package_path=temp_filepath)

            with open(temp_filepath, 'r') as f:
                yaml.load(f)

    def test_import_object(self):  # pylint: disable=no-self-use
        from kfp.v2.compiler import Compiler
        from kfp.v2.dsl import component
        from kfp.v2.dsl import pipeline

        @component
        def hello_world(text: str) -> str:
            """Hello world component."""
            return text

        @pipeline(name='hello-world', description='A simple intro pipeline')
        def pipeline_hello_world(text: str = 'hi there'):
            """Hello world pipeline."""

            hello_world(text=text)

        with tempfile.TemporaryDirectory() as tempdir:
            # you can e.g. create a file here:
            temp_filepath = os.path.join(tempdir, 'hello_world_pipeline.yaml')
            Compiler().compile(
                pipeline_func=pipeline_hello_world, package_path=temp_filepath)

            with open(temp_filepath, 'r') as f:
                yaml.load(f)


class TestWriteToFileTypes(parameterized.TestCase):
    pipeline_name = 'test-pipeline'

    def make_pipeline_spec(self):

        @dsl.component
        def dummy_op():
            pass

        @dsl.pipeline(name=self.pipeline_name)
        def my_pipeline():
            task = dummy_op()

        return my_pipeline

    @parameterized.parameters(
        {'extension': '.yaml'},
        {'extension': '.yml'},
    )
    def test_can_write_to_yaml(self, extension):

        with tempfile.TemporaryDirectory() as tmpdir:
            pipeline_spec = self.make_pipeline_spec()

            target_file = os.path.join(tmpdir, f'result{extension}')
            compiler.Compiler().compile(
                pipeline_func=pipeline_spec, package_path=target_file)

            self.assertTrue(os.path.exists(target_file))
            with open(target_file) as f:
                pipeline_spec = yaml.safe_load(f)

            self.assertEqual(self.pipeline_name,
                             pipeline_spec['pipelineInfo']['name'])

    def test_can_write_to_json(self):

        with tempfile.TemporaryDirectory() as tmpdir:
            pipeline_spec = self.make_pipeline_spec()

            target_file = os.path.join(tmpdir, 'result.json')
            with self.assertWarnsRegex(DeprecationWarning,
                                       r'Compiling to JSON is deprecated'):
                compiler.Compiler().compile(
                    pipeline_func=pipeline_spec, package_path=target_file)
            with open(target_file) as f:
                pipeline_spec = json.load(f)

            self.assertEqual(self.pipeline_name,
                             pipeline_spec['pipelineInfo']['name'])

    def test_cannot_write_to_bad_extension(self):

        with tempfile.TemporaryDirectory() as tmpdir:

            pipeline_spec = self.make_pipeline_spec()

            target_file = os.path.join(tmpdir, 'result.bad_extension')
            with self.assertRaisesRegex(ValueError,
                                        r'.* should end with "\.yaml".*'):
                compiler.Compiler().compile(
                    pipeline_func=pipeline_spec, package_path=target_file)

    def test_compile_pipeline_with_default_value(self):

        with tempfile.TemporaryDirectory() as tmpdir:
            producer_op = components.load_component_from_text("""
      name: producer
      inputs:
      - {name: location, type: String, default: 'us-central1'}
      - {name: name, type: Integer, default: 1}
      - {name: nodefault, type: String}
      implementation:
        container:
          image: gcr.io/my-project/my-image:tag
          args:
          - {inputValue: location}
      """)

            @dsl.pipeline(name='test-pipeline')
            def simple_pipeline():
                producer = producer_op(location='1', nodefault='string')

            target_json_file = os.path.join(tmpdir, 'result.json')
            compiler.Compiler().compile(
                pipeline_func=simple_pipeline, package_path=target_json_file)

            self.assertTrue(os.path.exists(target_json_file))
            with open(target_json_file, 'r') as f:
                f.read()
                pass

    def test_compile_fails_with_bad_pipeline_func(self):
        with self.assertRaisesRegex(ValueError,
                                    r'Unsupported pipeline_func type'):
            compiler.Compiler().compile(
                pipeline_func=None, package_path='/tmp/pipeline.yaml')


class TestCompileComponent(parameterized.TestCase):

    @parameterized.parameters(['.json', '.yaml', '.yml'])
    def test_compile_component_simple(self, extension: str):

        @dsl.component
        def hello_world(text: str) -> str:
            """Hello world component."""
            return text

        with tempfile.TemporaryDirectory() as tempdir:
            output_json = os.path.join(tempdir, f'component{extension}')
            compiler.Compiler().compile(
                pipeline_func=hello_world, package_path=output_json)
            with open(output_json, 'r') as f:
                pipeline_spec = yaml.safe_load(f)

        self.assertEqual(pipeline_spec['pipelineInfo']['name'], 'hello-world')

    def test_compile_component_two_inputs(self):

        @dsl.component
        def hello_world(text: str, integer: int) -> str:
            """Hello world component."""
            print(integer)
            return text

        with tempfile.TemporaryDirectory() as tempdir:
            output_json = os.path.join(tempdir, 'component.yaml')
            compiler.Compiler().compile(
                pipeline_func=hello_world, package_path=output_json)
            with open(output_json, 'r') as f:
                pipeline_spec = yaml.safe_load(f)

        self.assertEqual(
            pipeline_spec['root']['inputDefinitions']['parameters']['integer']
            ['parameterType'], 'NUMBER_INTEGER')

    def test_compile_component_with_default(self):

        @dsl.component
        def hello_world(text: str = 'default_string') -> str:
            """Hello world component."""
            return text

        with tempfile.TemporaryDirectory() as tempdir:
            output_json = os.path.join(tempdir, 'component.yaml')
            compiler.Compiler().compile(
                pipeline_func=hello_world, package_path=output_json)
            with open(output_json, 'r') as f:
                pipeline_spec = yaml.safe_load(f)

        self.assertEqual(pipeline_spec['pipelineInfo']['name'], 'hello-world')
        self.assertEqual(
            pipeline_spec['root']['inputDefinitions']['parameters']['text']
            ['defaultValue'], 'default_string')

    def test_compile_component_with_pipeline_name(self):

        @dsl.component
        def hello_world(text: str = 'default_string') -> str:
            """Hello world component."""
            return text

        with tempfile.TemporaryDirectory() as tempdir:
            output_json = os.path.join(tempdir, 'component.yaml')
            compiler.Compiler().compile(
                pipeline_func=hello_world,
                package_path=output_json,
                pipeline_name='custom-name')
            with open(output_json, 'r') as f:
                pipeline_spec = yaml.safe_load(f)

        self.assertEqual(pipeline_spec['pipelineInfo']['name'], 'custom-name')

    def test_compile_component_with_pipeline_parameters_override(self):

        @dsl.component
        def hello_world(text: str) -> str:
            """Hello world component."""
            return text

        with tempfile.TemporaryDirectory() as tempdir:
            output_json = os.path.join(tempdir, 'component.yaml')
            compiler.Compiler().compile(
                pipeline_func=hello_world,
                package_path=output_json,
                pipeline_parameters={'text': 'override_string'})
            with open(output_json, 'r') as f:
                pipeline_spec = yaml.safe_load(f)

        self.assertEqual(
            pipeline_spec['root']['inputDefinitions']['parameters']['text']
            ['defaultValue'], 'override_string')


class TestCompileBadInput(unittest.TestCase):

    def test_compile_non_pipeline_func(self):
        with self.assertRaisesRegex(ValueError,
                                    'Unsupported pipeline_func type.'):
            compiler.Compiler().compile(
                pipeline_func=lambda x: x, package_path='output.json')

    def test_compile_int(self):
        with self.assertRaisesRegex(ValueError,
                                    'Unsupported pipeline_func type.'):
            compiler.Compiler().compile(
                pipeline_func=1, package_path='output.json')


def pipeline_spec_from_file(filepath: str) -> str:
    with open(filepath, 'r') as f:
        dictionary = yaml.safe_load(f)
    return json_format.ParseDict(dictionary, pipeline_spec_pb2.PipelineSpec())


class TestWriteIrToFile(unittest.TestCase):

    @classmethod
    def setUpClass(cls) -> None:
        pipeline_spec = pipeline_spec_pb2.PipelineSpec()
        pipeline_spec.pipeline_info.name = 'pipeline-name'
        cls.pipeline_spec = pipeline_spec

    def test_yaml(self):
        with tempfile.TemporaryDirectory() as tempdir:
            temp_filepath = os.path.join(tempdir, 'output.yaml')
            compiler.write_pipeline_spec_to_file(self.pipeline_spec,
                                                 temp_filepath)
            actual = pipeline_spec_from_file(temp_filepath)
        self.assertEqual(actual, self.pipeline_spec)

    def test_yml(self):
        with tempfile.TemporaryDirectory() as tempdir:
            temp_filepath = os.path.join(tempdir, 'output.yml')
            compiler.write_pipeline_spec_to_file(self.pipeline_spec,
                                                 temp_filepath)
            actual = pipeline_spec_from_file(temp_filepath)
        self.assertEqual(actual, self.pipeline_spec)

    def test_json(self):
        with tempfile.TemporaryDirectory() as tempdir, self.assertWarnsRegex(
                DeprecationWarning, r'Compiling to JSON is deprecated'):
            temp_filepath = os.path.join(tempdir, 'output.json')
            compiler.write_pipeline_spec_to_file(self.pipeline_spec,
                                                 temp_filepath)
            actual = pipeline_spec_from_file(temp_filepath)
        self.assertEqual(actual, self.pipeline_spec)

    def test_incorrect_extension(self):
        with tempfile.TemporaryDirectory() as tempdir, self.assertRaisesRegex(
                ValueError, r'should end with "\.yaml"\.'):
            temp_filepath = os.path.join(tempdir, 'output.txt')
            compiler.write_pipeline_spec_to_file(self.pipeline_spec,
                                                 temp_filepath)


PIPELINES_TEST_DATA_DIR = os.path.join(
    os.path.dirname(__file__), 'test_data', 'pipelines')
UNSUPPORTED_COMPONENTS_TEST_DATA_DIR = os.path.join(
    os.path.dirname(__file__), 'test_data', 'components', 'unsupported')

UNSUPPORTED_COMPONENT_TEST_CASES = [
    'artifact_consumer',
    'output_named_tuple',
    'task_status',
    'train',
]


class TestReadWriteEquality(parameterized.TestCase):

    @classmethod
    def setUpClass(cls):
        cls.runner = testing.CliRunner()

    def _test_compile(self,
                      file_base_name: str,
                      directory: str,
                      fn: Optional[str] = None,
                      additional_arguments: Optional[List[str]] = None) -> None:
        py_file = os.path.join(directory, f'{file_base_name}.py')

        golden_compiled_file = os.path.join(directory, f'{file_base_name}.yaml')

        if additional_arguments is None:
            additional_arguments = []

        function_parts = [] if fn is None else ['--function', fn]

        with tempfile.TemporaryDirectory() as tmpdir:
            generated_compiled_file = os.path.join(tmpdir,
                                                   f'{file_base_name}.yaml')

            result = self.runner.invoke(
                cli=cli.cli,
                args=[
                    'dsl', 'compile', '--py', py_file, '--output',
                    generated_compiled_file
                ] + function_parts + additional_arguments,
                catch_exceptions=False)

            if result.exit_code != 0:
                print(result.output)
            self.assertEqual(result.exit_code, 0)

            compiled = load_compiled_file(generated_compiled_file)

        golden = load_compiled_file(golden_compiled_file)
        self.assertEqual(golden, compiled)

    def test_two_step_pipeline(self):
        self._test_compile(
            'two_step_pipeline',
            directory=PIPELINES_TEST_DATA_DIR,
            additional_arguments=[
                '--pipeline-parameters', '{"text":"Hello KFP!"}'
            ])

    def test_two_step_pipeline_failure_parameter_parse(self):
        with self.assertRaisesRegex(json.decoder.JSONDecodeError,
                                    r'Unterminated string starting at:'):
            self._test_compile(
                'two_step_pipeline',
                directory=PIPELINES_TEST_DATA_DIR,
                additional_arguments=[
                    '--pipeline-parameters', '{"text":"Hello KFP!}'
                ])

    def test_compile_components_not_found(self):
        with self.assertRaisesRegex(
                ValueError,
                r'Pipeline function or component "step1" not found in module two_step_pipeline\.py\.'
        ):
            self._test_compile(
                'two_step_pipeline',
                directory=PIPELINES_TEST_DATA_DIR,
                fn='step1')

    def test_deprecation_warning(self):
        res = subprocess.run(['dsl-compile', '--help'], capture_output=True)
        self.assertIn('Deprecated. Please use `kfp dsl compile` instead.)',
                      res.stdout.decode('utf-8'))

    @parameterized.parameters(UNSUPPORTED_COMPONENT_TEST_CASES)
    def test_compile_unsupported_components(self, component_name: str):
        with self.assertRaisesRegex(TypeError, r'is not a valid input'):
            self._test_compile(
                component_name,
                directory=UNSUPPORTED_COMPONENTS_TEST_DATA_DIR,
                fn=component_name)


def ignore_kfp_version_helper(spec: Dict[str, Any]) -> Dict[str, Any]:
    """Ignores kfp sdk versioning in command.

    Takes in a YAML input and ignores the kfp sdk versioning in command
    for comparison between compiled file and goldens.
    """
    pipeline_spec = spec.get('pipelineSpec', spec)

    if 'executors' in pipeline_spec['deploymentSpec']:
        for executor in pipeline_spec['deploymentSpec']['executors']:
            pipeline_spec['deploymentSpec']['executors'][
                executor] = yaml.safe_load(
                    re.sub(
                        r"'kfp==(\d+).(\d+).(\d+)(-[a-z]+.\d+)?'", 'kfp',
                        yaml.dump(
                            pipeline_spec['deploymentSpec']['executors']
                            [executor],
                            sort_keys=True)))
    return spec


def load_compiled_file(filename: str) -> Dict[str, Any]:
    with open(filename, 'r') as f:
        contents = yaml.safe_load(f)
        pipeline_spec = contents[
            'pipelineSpec'] if 'pipelineSpec' in contents else contents
        # ignore the sdkVersion
        del pipeline_spec['sdkVersion']
        return ignore_kfp_version_helper(contents)


if __name__ == '__main__':
    unittest.main()
