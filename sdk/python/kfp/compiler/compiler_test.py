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

import collections
import json
import os
import re
import subprocess
import tempfile
from typing import Any, Dict, List, NamedTuple, Optional
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

        with self.assertRaisesRegex(
                TypeError,
                ' type "system.Model@0.0.1" cannot be paired with InputValuePlaceholder.'
        ):

            @dsl.pipeline(name='test-pipeline', pipeline_root='dummy_root')
            def my_pipeline():
                downstream_op(model=upstream_op().output)

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

        with self.assertRaisesRegex(
                TypeError,
                ' type "String" cannot be paired with InputPathPlaceholder.'):

            @dsl.pipeline(name='test-pipeline', pipeline_root='dummy_root')
            def my_pipeline(text: str):
                component_op(text=text)

    def test_compile_pipeline_with_missing_task_should_raise_error(self):

        with self.assertRaisesRegex(ValueError,
                                    'Task is missing from pipeline.'):

            @dsl.pipeline(name='test-pipeline', pipeline_root='dummy_root')
            def my_pipeline(text: str):
                pass

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

        with self.assertRaisesRegex(
                TypeError,
                ' type "Float" cannot be paired with InputUriPlaceholder.'):

            @dsl.pipeline(name='test-pipeline', pipeline_root='dummy_root')
            def my_pipeline(value: float):
                component_op(value=value)

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

        with self.assertRaisesRegex(
                TypeError,
                ' type "Integer" cannot be paired with OutputUriPlaceholder.'):

            @dsl.pipeline(name='test-pipeline', pipeline_root='dummy_root')
            def my_pipeline():
                component_op()

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
                pipeline_spec = yaml.safe_load(f)
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
        with self.assertRaisesRegex(
                type_utils.InconsistentTypeException,
                'Incompatible argument passed to the input "some_input" of '
                'component "compoent": Argument type "STRING" is incompatible '
                'with the input type "system.Artifact@0.0.1"'):

            @dsl.pipeline(name='test-pipeline', pipeline_root='gs://path')
            def my_pipeline(input1: str):
                component_op(some_input=input1)

    def test_passing_missing_type_annotation_on_pipeline_input_should_error(
            self):

        with self.assertRaisesRegex(
                TypeError, 'Missing type annotation for argument: input1'):

            @dsl.pipeline(name='test-pipeline', pipeline_root='gs://path')
            def my_pipeline(input1):
                pass

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

        with tempfile.TemporaryDirectory() as tmpdir:
            target_yaml_file = os.path.join(tmpdir, 'result.yaml')
            compiler.Compiler().compile(
                pipeline_func=my_pipeline, package_path=target_yaml_file)

            self.assertTrue(os.path.exists(target_yaml_file))

    def test_invalid_data_dependency_loop(self):

        @dsl.component
        def producer_op() -> str:
            return 'a'

        @dsl.component
        def dummy_op(msg: str = ''):
            pass

        with self.assertRaisesRegex(
                RuntimeError,
                'Task dummy-op cannot dependent on any task inside the group:'):

            @dsl.pipeline(name='test-pipeline')
            def my_pipeline(val: bool):
                with dsl.ParallelFor(['a, b']):
                    producer_task = producer_op()

                dummy_op(msg=producer_task.output)

    def test_valid_data_dependency_loop(self):

        @dsl.component
        def producer_op() -> str:
            return 'a'

        @dsl.component
        def dummy_op(msg: str = ''):
            pass

        @dsl.pipeline(name='test-pipeline')
        def my_pipeline(val: bool):
            with dsl.ParallelFor(['a, b']):
                producer_task = producer_op()
                dummy_op(msg=producer_task.output)

        with tempfile.TemporaryDirectory() as tmpdir:
            package_path = os.path.join(tmpdir, 'pipeline.yaml')
            compiler.Compiler().compile(
                pipeline_func=my_pipeline, package_path=package_path)

    def test_invalid_data_dependency_condition(self):

        @dsl.component
        def producer_op() -> str:
            return 'a'

        @dsl.component
        def dummy_op(msg: str = ''):
            pass

        with self.assertRaisesRegex(
                RuntimeError,
                'Task dummy-op cannot dependent on any task inside the group:'):

            @dsl.pipeline(name='test-pipeline')
            def my_pipeline(val: bool):
                with dsl.Condition(val == False):
                    producer_task = producer_op()

                dummy_op(msg=producer_task.output)

    def test_valid_data_dependency_condition(self):

        @dsl.component
        def producer_op() -> str:
            return 'a'

        @dsl.component
        def dummy_op(msg: str = ''):
            pass

        @dsl.pipeline(name='test-pipeline')
        def my_pipeline(val: bool):
            with dsl.Condition(val == False):
                producer_task = producer_op()
                dummy_op(msg=producer_task.output)

        with tempfile.TemporaryDirectory() as tmpdir:
            package_path = os.path.join(tmpdir, 'pipeline.yaml')
            compiler.Compiler().compile(
                pipeline_func=my_pipeline, package_path=package_path)

    def test_invalid_data_dependency_exit_handler(self):

        @dsl.component
        def producer_op() -> str:
            return 'a'

        @dsl.component
        def dummy_op(msg: str = ''):
            pass

        with self.assertRaisesRegex(
                RuntimeError,
                'Task dummy-op cannot dependent on any task inside the group:'):

            @dsl.pipeline(name='test-pipeline')
            def my_pipeline(val: bool):
                first_producer = producer_op()
                with dsl.ExitHandler(first_producer):
                    producer_task = producer_op()

                dummy_op(msg=producer_task.output)

    def test_valid_data_dependency_exit_handler(self):

        @dsl.component
        def producer_op() -> str:
            return 'a'

        @dsl.component
        def dummy_op(msg: str = ''):
            pass

        @dsl.pipeline(name='test-pipeline')
        def my_pipeline(val: bool):
            first_producer = producer_op()
            with dsl.ExitHandler(first_producer):
                producer_task = producer_op()
                dummy_op(msg=producer_task.output)

        with tempfile.TemporaryDirectory() as tmpdir:
            package_path = os.path.join(tmpdir, 'pipeline.yaml')
            compiler.Compiler().compile(
                pipeline_func=my_pipeline, package_path=package_path)

    def test_use_task_final_status_in_non_exit_op(self):

        @dsl.component
        def print_op(status: PipelineTaskFinalStatus):
            return status

        with self.assertRaisesRegex(
                ValueError,
                'PipelineTaskFinalStatus can only be used in an exit task.'):

            @dsl.pipeline(name='test-pipeline')
            def my_pipeline(text: bool):
                print_op()

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

        with self.assertRaisesRegex(
                ValueError,
                'PipelineTaskFinalStatus can only be used in an exit task.'):

            @dsl.pipeline(name='test-pipeline')
            def my_pipeline(text: bool):
                print_op()

    def test_compile_parallel_for_with_valid_parallelism(self):

        @dsl.component
        def producer_op(item: str) -> str:
            return item

        @dsl.pipeline(name='test-parallel-for-with-parallelism')
        def my_pipeline(text: bool):
            with dsl.ParallelFor(items=['a', 'b'], parallelism=2) as item:
                producer_task = producer_op(item=item)

        with tempfile.TemporaryDirectory() as tempdir:
            output_yaml = os.path.join(tempdir, 'result.yaml')
            compiler.Compiler().compile(
                pipeline_func=my_pipeline, package_path=output_yaml)
            with open(output_yaml, 'r') as f:
                pipeline_spec = yaml.safe_load(f)
        self.assertEqual(
            pipeline_spec['root']['dag']['tasks']['for-loop-2']
            ['iteratorPolicy']['parallelismLimit'], 2)

    def test_compile_parallel_for_with_invalid_parallelism(self):

        @dsl.component
        def producer_op(item: str) -> str:
            return item

        with self.assertRaisesRegex(ValueError,
                                    'ParallelFor parallelism must be >= 0.'):

            @dsl.pipeline(name='test-parallel-for-with-parallelism')
            def my_pipeline(text: bool):
                with dsl.ParallelFor(items=['a', 'b'], parallelism=-2) as item:
                    producer_task = producer_op(item=item)

    def test_compile_parallel_for_with_zero_parallelism(self):

        @dsl.component
        def producer_op(item: str) -> str:
            return item

        @dsl.pipeline(name='test-parallel-for-with-parallelism')
        def my_pipeline(text: bool):
            with dsl.ParallelFor(items=['a', 'b'], parallelism=0) as item:
                producer_task = producer_op(item=item)

            with dsl.ParallelFor(items=['a', 'b']) as item:
                producer_task = producer_op(item=item)

        with tempfile.TemporaryDirectory() as tempdir:
            output_yaml = os.path.join(tempdir, 'result.yaml')
            compiler.Compiler().compile(
                pipeline_func=my_pipeline, package_path=output_yaml)
            with open(output_yaml, 'r') as f:
                pipeline_spec = yaml.safe_load(f)
        for_loop_2 = pipeline_spec['root']['dag']['tasks']['for-loop-2']
        for_loop_4 = pipeline_spec['root']['dag']['tasks']['for-loop-4']
        with self.assertRaises(KeyError):
            for_loop_2['iteratorPolicy']
        with self.assertRaises(KeyError):
            for_loop_4['iteratorPolicy']

    def test_pipeline_in_pipeline(self):

        @dsl.component
        def print_op(msg: str):
            print(msg)

        @dsl.pipeline(name='graph-component')
        def graph_component(msg: str):
            print_op(msg=msg)

        @dsl.pipeline(name='test-pipeline')
        def my_pipeline():
            graph_component(msg='hello')

        with tempfile.TemporaryDirectory() as tmpdir:
            output_yaml = os.path.join(tmpdir, 'result.yaml')
            compiler.Compiler().compile(
                pipeline_func=my_pipeline, package_path=output_yaml)
            self.assertTrue(os.path.exists(output_yaml))

            with open(output_yaml, 'r') as f:
                pipeline_spec = yaml.safe_load(f)
                self.assertEqual(2, len(pipeline_spec['components']))
                self.assertTrue('comp-print-op' in pipeline_spec['components'])
                self.assertTrue(
                    'comp-graph-component' in pipeline_spec['components'])
                self.assertEqual(
                    1, len(pipeline_spec['deploymentSpec']['executors']))
                self.assertTrue('exec-print-op' in
                                pipeline_spec['deploymentSpec']['executors'])

    def test_pipeline_with_invalid_output(self):
        with self.assertRaisesRegex(ValueError,
                                    'Pipeline output not defined: msg1'):

            @dsl.component
            def print_op(msg: str) -> str:
                print(msg)

            @dsl.pipeline
            def my_pipeline() -> NamedTuple('Outputs', [
                ('msg', str),
            ]):
                task = print_op(msg='Hello')
                output = collections.namedtuple('Outputs', ['msg1'])
                return output(task.output)

    def test_pipeline_with_missing_output(self):
        with self.assertRaisesRegex(ValueError, 'Missing pipeline output: msg'):

            @dsl.component
            def print_op(msg: str) -> str:
                print(msg)

            @dsl.pipeline
            def my_pipeline() -> NamedTuple('Outputs', [
                ('msg', str),
            ]):
                task = print_op(msg='Hello')

        with self.assertRaisesRegex(ValueError,
                                    'Missing pipeline output: model'):

            @dsl.component
            def print_op(msg: str) -> str:
                print(msg)

            @dsl.pipeline
            def my_pipeline() -> NamedTuple('Outputs', [
                ('model', dsl.Model),
            ]):
                task = print_op(msg='Hello')


class V2NamespaceAliasTest(unittest.TestCase):
    """Test that imports of both modules and objects are aliased (e.g. all
    import path variants work)."""

    # Note: The DeprecationWarning is only raised on the first import where
    # the kfp.v2 module is loaded. Due to the way we run tests in CI/CD, we cannot ensure that the kfp.v2 module will first be loaded in these tests,
    # so we do not test for the DeprecationWarning here.

    def test_import_namespace(self):
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
                yaml.safe_load(f)

    def test_import_modules(self):
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
                yaml.safe_load(f)

    def test_import_object(self):
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
                yaml.safe_load(f)


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

    def test_compile_container_component_simple(self):

        @dsl.container_component
        def hello_world_container() -> dsl.ContainerSpec:
            """Hello world component."""
            return dsl.ContainerSpec(
                image='python:3.7',
                command=['echo', 'hello world'],
                args=[],
            )

        with tempfile.TemporaryDirectory() as tempdir:
            output_json = os.path.join(tempdir, 'component.yaml')
            compiler.Compiler().compile(
                pipeline_func=hello_world_container,
                package_path=output_json,
                pipeline_name='hello-world-container')
            with open(output_json, 'r') as f:
                pipeline_spec = yaml.safe_load(f)
        self.assertEqual(
            pipeline_spec['deploymentSpec']['executors']
            ['exec-hello-world-container']['container']['command'],
            ['echo', 'hello world'])

    def test_compile_container_with_simple_io(self):

        @dsl.container_component
        def container_simple_io(text: str, output_path: dsl.OutputPath(str)):
            return dsl.ContainerSpec(
                image='python:3.7',
                command=['my_program', text],
                args=['--output_path', output_path])

        with tempfile.TemporaryDirectory() as tempdir:
            output_json = os.path.join(tempdir, 'component.yaml')
            compiler.Compiler().compile(
                pipeline_func=container_simple_io,
                package_path=output_json,
                pipeline_name='container-simple-io')
            with open(output_json, 'r') as f:
                pipeline_spec = yaml.safe_load(f)
        self.assertEqual(
            pipeline_spec['components']['comp-container-simple-io']
            ['inputDefinitions']['parameters']['text']['parameterType'],
            'STRING')
        self.assertEqual(
            pipeline_spec['components']['comp-container-simple-io']
            ['outputDefinitions']['parameters']['output_path']['parameterType'],
            'STRING')

    def test_compile_container_with_artifact_output(self):

        @dsl.container_component
        def container_with_artifact_output(
                num_epochs: int,  # also as an input
                model: dsl.Output[dsl.Model],
                model_config_path: dsl.OutputPath(str),
        ):
            return dsl.ContainerSpec(
                image='gcr.io/my-image',
                command=['sh', 'run.sh'],
                args=[
                    '--epochs',
                    num_epochs,
                    '--model_path',
                    model.uri,
                    '--model_config_path',
                    model_config_path,
                ])

        with tempfile.TemporaryDirectory() as tempdir:
            output_yaml = os.path.join(tempdir, 'component.yaml')
            compiler.Compiler().compile(
                pipeline_func=container_with_artifact_output,
                package_path=output_yaml,
                pipeline_name='container-with-artifact-output')
            with open(output_yaml, 'r') as f:
                pipeline_spec = yaml.safe_load(f)
        self.assertEqual(
            pipeline_spec['components']['comp-container-with-artifact-output']
            ['inputDefinitions']['parameters']['num_epochs']['parameterType'],
            'NUMBER_INTEGER')
        self.assertEqual(
            pipeline_spec['components']['comp-container-with-artifact-output']
            ['outputDefinitions']['artifacts']['model']['artifactType']
            ['schemaTitle'], 'system.Model')
        self.assertEqual(
            pipeline_spec['components']['comp-container-with-artifact-output']
            ['outputDefinitions']['parameters']['model_config_path']
            ['parameterType'], 'STRING')
        args_to_check = pipeline_spec['deploymentSpec']['executors'][
            'exec-container-with-artifact-output']['container']['args']
        self.assertEqual(args_to_check[3],
                         "{{$.outputs.artifacts['model'].uri}}")
        self.assertEqual(
            args_to_check[5],
            "{{$.outputs.parameters['model_config_path'].output_file}}")


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


_PROJECT_ROOT = os.path.abspath(os.path.join(__file__, *([os.path.pardir] * 5)))
_TEST_DATA_DIR = os.path.join(_PROJECT_ROOT, 'sdk', 'python', 'test_data')
PIPELINES_TEST_DATA_DIR = os.path.join(_TEST_DATA_DIR, 'pipelines')
UNSUPPORTED_COMPONENTS_TEST_DATA_DIR = os.path.join(_TEST_DATA_DIR,
                                                    'components', 'unsupported')


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

    # TODO: the sample does not throw as expected.
    # def test_compile_unsupported_components_with_output_named_tuple(self):
    #     with self.assertRaisesRegex(TODO):
    #         self._test_compile(
    #             'output_named_tuple',
    #             directory=UNSUPPORTED_COMPONENTS_TEST_DATA_DIR,
    #             fn='output_named_tuple')

    # TODO: the sample does not throw as expected.
    # def test_compile_unsupported_components_with_task_status(self):
    #     with self.assertRaisesRegex(TODO):
    #         self._test_compile(
    #             'task_status',
    #             directory=UNSUPPORTED_COMPONENTS_TEST_DATA_DIR,
    #             fn='task_status')


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


class TestSetRetryCompilation(unittest.TestCase):

    def test_set_retry(self):

        @dsl.component
        def hello_world(text: str) -> str:
            """Hello world component."""
            return text

        @dsl.pipeline(name='hello-world', description='A simple intro pipeline')
        def pipeline_hello_world(text: str = 'hi there'):
            """Hello world pipeline."""

            hello_world(text=text).set_retry(
                num_retries=3,
                backoff_duration='30s',
                backoff_factor=1.0,
                backoff_max_duration='3h',
            )

        with tempfile.TemporaryDirectory() as tempdir:
            package_path = os.path.join(tempdir, 'pipeline.yaml')
            compiler.Compiler().compile(
                pipeline_func=pipeline_hello_world, package_path=package_path)
            pipeline_spec = pipeline_spec_from_file(package_path)

        retry_policy = pipeline_spec.root.dag.tasks['hello-world'].retry_policy
        self.assertEqual(retry_policy.max_retry_count, 3)
        self.assertEqual(retry_policy.backoff_duration.seconds, 30)
        self.assertEqual(retry_policy.backoff_factor, 1.0)
        self.assertEqual(retry_policy.backoff_max_duration.seconds, 3600)


from google.protobuf import json_format


class TestMultipleExitHandlerCompilation(unittest.TestCase):

    def test_basic(self):

        @dsl.component
        def print_op(message: str):
            print(message)

        @dsl.pipeline(name='pipeline-with-multiple-exit-handlers')
        def my_pipeline():
            first_exit_task = print_op(message='First exit task.')

            with dsl.ExitHandler(first_exit_task):
                print_op(message='Inside first exit handler.')

            second_exit_task = print_op(message='Second exit task.')
            with dsl.ExitHandler(second_exit_task):
                print_op(message='Inside second exit handler.')

        with tempfile.TemporaryDirectory() as tempdir:
            package_path = os.path.join(tempdir, 'pipeline.yaml')
            compiler.Compiler().compile(
                pipeline_func=my_pipeline, package_path=package_path)
            pipeline_spec = pipeline_spec_from_file(package_path)
        # check that the exit handler dags exist
        self.assertEqual(
            pipeline_spec.components['comp-exit-handler-1'].dag
            .tasks['print-op-2'].inputs.parameters['message'].runtime_value
            .constant.string_value, 'Inside first exit handler.')
        self.assertEqual(
            pipeline_spec.components['comp-exit-handler-2'].dag
            .tasks['print-op-4'].inputs.parameters['message'].runtime_value
            .constant.string_value, 'Inside second exit handler.')
        # check that the exit handler dags are in the root dag
        self.assertIn('exit-handler-1', pipeline_spec.root.dag.tasks)
        self.assertIn('exit-handler-2', pipeline_spec.root.dag.tasks)
        # check that the exit tasks are in the root dag
        self.assertIn('print-op', pipeline_spec.root.dag.tasks)
        self.assertEqual(
            pipeline_spec.root.dag.tasks['print-op'].inputs
            .parameters['message'].runtime_value.constant.string_value,
            'First exit task.')
        self.assertIn('print-op-3', pipeline_spec.root.dag.tasks)
        self.assertEqual(
            pipeline_spec.root.dag.tasks['print-op-3'].inputs
            .parameters['message'].runtime_value.constant.string_value,
            'Second exit task.')

    def test_nested_unsupported(self):

        @dsl.component
        def print_op(message: str):
            print(message)

        with self.assertRaisesRegex(
                ValueError,
                r'ExitHandler can only be used within the outermost scope of a pipeline function definition\.'
        ):

            @dsl.pipeline(name='pipeline-with-multiple-exit-handlers')
            def my_pipeline():
                first_exit_task = print_op(message='First exit task.')

                with dsl.ExitHandler(first_exit_task):
                    print_op(message='Inside first exit handler.')

                    second_exit_task = print_op(message='Second exit task.')
                    with dsl.ExitHandler(second_exit_task):
                        print_op(message='Inside second exit handler.')


class TestBoolInputParameterWithDefaultSerializesCorrectly(unittest.TestCase):
    # test with default = True, may have false test successes due to protocol buffer boolean default of False
    def test_python_component(self):

        @dsl.component
        def comp(boolean: bool = True) -> bool:
            return boolean

        # test inner component interface
        self.assertEqual(
            comp.pipeline_spec.components['comp-comp'].input_definitions
            .parameters['boolean'].default_value.bool_value, True)

        # test outer pipeline "wrapper" interface
        self.assertEqual(
            comp.pipeline_spec.root.input_definitions.parameters['boolean']
            .default_value.bool_value, True)

    def test_python_component_with_overrides(self):

        @dsl.component
        def comp(boolean: bool = False) -> bool:
            return boolean

        with tempfile.TemporaryDirectory() as tmpdir:
            pipeline_spec_path = os.path.join(tmpdir, 'output.yaml')
            compiler.Compiler().compile(
                comp, pipeline_spec_path, pipeline_parameters={'boolean': True})
            pipeline_spec = pipeline_spec_from_file(pipeline_spec_path)

        # test outer pipeline "wrapper" interface
        self.assertEqual(
            pipeline_spec.root.input_definitions.parameters['boolean']
            .default_value.bool_value, True)

    def test_container_component(self):

        @dsl.container_component
        def comp(boolean: bool = True):
            return dsl.ContainerSpec(image='alpine', command=['echo', boolean])

        # test inner component interface
        self.assertEqual(
            comp.pipeline_spec.components['comp-comp'].input_definitions
            .parameters['boolean'].default_value.bool_value, True)

        # test pipeline "wrapper" interface
        self.assertEqual(
            comp.pipeline_spec.root.input_definitions.parameters['boolean']
            .default_value.bool_value, True)

    def test_container_component_with_overrides(self):

        @dsl.container_component
        def comp(boolean: bool = True):
            return dsl.ContainerSpec(image='alpine', command=['echo', boolean])

        with tempfile.TemporaryDirectory() as tmpdir:
            pipeline_spec_path = os.path.join(tmpdir, 'output.yaml')
            compiler.Compiler().compile(
                comp, pipeline_spec_path, pipeline_parameters={'boolean': True})
            pipeline_spec = pipeline_spec_from_file(pipeline_spec_path)

        # test outer pipeline "wrapper" interface
        self.assertEqual(
            pipeline_spec.root.input_definitions.parameters['boolean']
            .default_value.bool_value, True)

    def test_pipeline_no_input(self):

        @dsl.component
        def comp(boolean: bool = True) -> bool:
            return boolean

        @dsl.pipeline
        def pipeline_no_input():
            comp()

        # test inner component interface
        self.assertEqual(
            pipeline_no_input.pipeline_spec.components['comp-comp']
            .input_definitions.parameters['boolean'].default_value.bool_value,
            True)

    def test_pipeline_with_input(self):

        @dsl.component
        def comp(boolean: bool = True) -> bool:
            return boolean

        @dsl.pipeline
        def pipeline_with_input(boolean: bool = True):
            comp(boolean=boolean)

        # test inner component interface
        self.assertEqual(
            pipeline_with_input.pipeline_spec.components['comp-comp']
            .input_definitions.parameters['boolean'].default_value.bool_value,
            True)

        # test pipeline interface
        self.assertEqual(
            pipeline_with_input.pipeline_spec.root.input_definitions
            .parameters['boolean'].default_value.bool_value, True)

    def test_pipeline_with_with_overrides(self):

        @dsl.component
        def comp(boolean: bool = True) -> bool:
            return boolean

        @dsl.pipeline
        def pipeline_with_input(boolean: bool = False):
            comp(boolean=boolean)

        with tempfile.TemporaryDirectory() as tmpdir:
            pipeline_spec_path = os.path.join(tmpdir, 'output.yaml')
            compiler.Compiler().compile(
                pipeline_with_input,
                pipeline_spec_path,
                pipeline_parameters={'boolean': True})
            pipeline_spec = pipeline_spec_from_file(pipeline_spec_path)

        # test inner component interface
        self.assertEqual(
            pipeline_spec.components['comp-comp'].input_definitions
            .parameters['boolean'].default_value.bool_value, True)

        # test pipeline interface
        self.assertEqual(
            pipeline_spec.root.input_definitions.parameters['boolean']
            .default_value.bool_value, True)


if __name__ == '__main__':
    unittest.main()
