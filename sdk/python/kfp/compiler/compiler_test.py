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
import textwrap
from typing import Any, Dict, List, NamedTuple, Optional
import unittest

from absl.testing import parameterized
from click import testing
from google.protobuf import json_format
import kfp
from kfp import components
from kfp import dsl
from kfp.cli import cli
from kfp.compiler import compiler
from kfp.compiler import compiler_utils
from kfp.components import graph_component
from kfp.components import pipeline_task
from kfp.components import yaml_component
from kfp.components.types import type_utils
from kfp.dsl import Artifact
from kfp.dsl import ContainerSpec
from kfp.dsl import Input
from kfp.dsl import Model
from kfp.dsl import Output
from kfp.dsl import OutputPath
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

    def test_can_use_dsl_attribute_on_kfp(self):

        @kfp.dsl.component
        def identity(string: str) -> str:
            return string

        @kfp.dsl.pipeline
        def my_pipeline(string: str = 'string'):
            op1 = identity(string=string)

        with tempfile.TemporaryDirectory() as tmpdir:
            compiler.Compiler().compile(
                pipeline_func=my_pipeline,
                package_path=os.path.join(tmpdir, 'pipeline.yaml'))

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

        @dsl.pipeline(name='test-pipeline', pipeline_root='gs://path')
        def my_pipeline():
            VALID_PRODUCER_COMPONENT_SAMPLE(input_param='input')

        self.assertEqual(my_pipeline.pipeline_spec.default_pipeline_root,
                         'gs://path')

    def test_set_display_name_through_pipeline_decorator(self):

        @dsl.pipeline(display_name='my display name')
        def my_pipeline():
            VALID_PRODUCER_COMPONENT_SAMPLE(input_param='input')

        self.assertEqual(my_pipeline.pipeline_spec.pipeline_info.display_name,
                         'my display name')

    def test_set_name_and_display_name_through_pipeline_decorator(self):

        @dsl.pipeline(
            name='my-pipeline-name',
            display_name='my display name',
        )
        def my_pipeline():
            VALID_PRODUCER_COMPONENT_SAMPLE(input_param='input')

        self.assertEqual(my_pipeline.pipeline_spec.pipeline_info.name,
                         'my-pipeline-name')
        self.assertEqual(my_pipeline.pipeline_spec.pipeline_info.display_name,
                         'my display name')

    def test_set_description_through_pipeline_decorator(self):

        @dsl.pipeline(description='Prefer me.')
        def my_pipeline():
            """Don't prefer me"""
            VALID_PRODUCER_COMPONENT_SAMPLE(input_param='input')

        self.assertEqual(my_pipeline.pipeline_spec.pipeline_info.description,
                         'Prefer me.')

    def test_set_description_through_pipeline_docstring_short(self):

        @dsl.pipeline
        def my_pipeline():
            """Docstring-specified description."""
            VALID_PRODUCER_COMPONENT_SAMPLE(input_param='input')

        self.assertEqual(my_pipeline.pipeline_spec.pipeline_info.description,
                         'Docstring-specified description.')

    def test_set_description_through_pipeline_docstring_long(self):

        @dsl.pipeline
        def my_pipeline():
            """Docstring-specified description.

            More information about this pipeline."""
            VALID_PRODUCER_COMPONENT_SAMPLE(input_param='input')

        self.assertEqual(
            my_pipeline.pipeline_spec.pipeline_info.description,
            'Docstring-specified description.\nMore information about this pipeline.'
        )

    def test_passing_string_parameter_to_artifact_should_error(self):

        component_op = components.load_component_from_text("""
      name: component
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
                "Incompatible argument passed to the input 'some_input' of "
                "component 'component': Argument type 'STRING' is incompatible "
                "with the input type 'system.Artifact@0.0.1'"):

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
                compiler_utils.InvalidTopologyException,
                r'Illegal task dependency across DSL context managers\. A downstream task cannot depend on an upstream task within a dsl\.ParallelFor context unless the downstream is within that context too or the outputs are begin fanned-in to a list using dsl\.Collected\. Found task dummy-op which depends on upstream task producer-op within an uncommon dsl\.ParallelFor context\.'
        ):

            @dsl.pipeline(name='test-pipeline')
            def my_pipeline(val: bool):
                with dsl.ParallelFor(['a, b']):
                    producer_task = producer_op()

                dummy_op(msg=producer_task.output)

    def test_invalid_data_dependency_condition(self):

        @dsl.component
        def producer_op() -> str:
            return 'a'

        @dsl.component
        def dummy_op(msg: str = ''):
            pass

        with self.assertRaisesRegex(
                compiler_utils.InvalidTopologyException,
                r'Illegal task dependency across DSL context managers\. A downstream task cannot depend on an upstream task within a dsl\.Condition context unless the downstream is within that context too\. Found task dummy-op which depends on upstream task producer-op within an uncommon dsl\.Condition context\.'
        ):

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
                compiler_utils.InvalidTopologyException,
                r'Illegal task dependency across DSL context managers\. A downstream task cannot depend on an upstream task within a dsl\.ExitHandler context unless the downstream is within that context too\. Found task dummy-op which depends on upstream task producer-op-2 within an uncommon dsl\.ExitHandler context\.'
        ):

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

    def test_task_final_status_parameter_type_is_used(self):
        # previously compiled to STRUCT type, so checking that this is updated

        @dsl.component
        def identity(string: str) -> str:
            return string

        @dsl.component
        def exit_comp(status: dsl.PipelineTaskFinalStatus):
            print(status)

        @dsl.pipeline
        def my_pipeline():
            exit_task = exit_comp()
            with dsl.ExitHandler(exit_task=exit_task):
                identity(string='hi')

        self.assertEqual(
            my_pipeline.pipeline_spec.components['comp-exit-comp']
            .input_definitions.parameters['status'].parameter_type,
            pipeline_spec_pb2.ParameterType.TASK_FINAL_STATUS)

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
        with self.assertRaisesRegex(
                ValueError, r'Pipeline or component output not defined: msg1'):

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


class TestBooleanInputCompiledCorrectly(unittest.TestCase):
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

    def test_constant_passed_to_component(self):

        @dsl.component
        def comp(boolean1: bool, boolean2: bool) -> bool:
            return boolean1

        @dsl.pipeline
        def my_pipeline():
            comp(boolean1=True, boolean2=False)

        with tempfile.TemporaryDirectory() as tmpdir:
            pipeline_spec_path = os.path.join(tmpdir, 'output.yaml')
            compiler.Compiler().compile(my_pipeline, pipeline_spec_path)
            pipeline_spec = pipeline_spec_from_file(pipeline_spec_path)
        self.assertTrue(
            pipeline_spec.root.dag.tasks['comp'].inputs.parameters['boolean1']
            .runtime_value.constant.bool_value)
        self.assertFalse(
            pipeline_spec.root.dag.tasks['comp'].inputs.parameters['boolean2']
            .runtime_value.constant.bool_value)


# helper component defintions for the ValidLegalTopologies tests
@dsl.component
def print_op(message: str):
    print(message)


@dsl.component
def return_1() -> int:
    return 1


@dsl.component
def args_generator_op() -> List[Dict[str, str]]:
    return [{'A_a': '1', 'B_b': '2'}, {'A_a': '10', 'B_b': '20'}]


class TestValidLegalTopologies(unittest.TestCase):

    def test_inside_of_root_group_permitted(self):

        @dsl.pipeline()
        def my_pipeline():
            return_1_task = return_1()

            one = print_op(message='1')
            two = print_op(message='2')
            three = print_op(message=str(return_1_task.output))

        with tempfile.TemporaryDirectory() as tempdir:
            package_path = os.path.join(tempdir, 'pipeline.yaml')
            compiler.Compiler().compile(
                pipeline_func=my_pipeline, package_path=package_path)

    def test_upstream_inside_deeper_condition_blocked(self):

        with self.assertRaisesRegex(
                compiler_utils.InvalidTopologyException,
                r'Illegal task dependency across DSL context managers. A downstream task cannot depend on an upstream task within a dsl.Condition context unless the downstream is within that context too\. Found task print-op-3 which depends on upstream task print-op-2 within an uncommon dsl\.Condition context\.'
        ):

            @dsl.pipeline()
            def my_pipeline():
                return_1_task = return_1()

                one = print_op(message='1')
                with dsl.Condition(return_1_task.output == 1):
                    two = print_op(message='2')

                three = print_op(message='3').after(two)

            with tempfile.TemporaryDirectory() as tempdir:
                package_path = os.path.join(tempdir, 'pipeline.yaml')
                compiler.Compiler().compile(
                    pipeline_func=my_pipeline, package_path=package_path)

    def test_upstream_in_the_same_condition_permitted(self):

        @dsl.pipeline()
        def my_pipeline():
            return_1_task = return_1()

            with dsl.Condition(return_1_task.output == 1):
                one = return_1()
                two = print_op(message='2')
                three = print_op(message=str(one.output))

        with tempfile.TemporaryDirectory() as tempdir:
            package_path = os.path.join(tempdir, 'pipeline.yaml')
            compiler.Compiler().compile(
                pipeline_func=my_pipeline, package_path=package_path)

    def test_downstream_inside_deeper_condition_permitted(self):

        @dsl.pipeline()
        def my_pipeline():
            return_1_task = return_1()

            one = print_op(message='1')
            with dsl.Condition(return_1_task.output == 1):
                two = print_op(message='2')
                three = print_op(message='3').after(one)

        with tempfile.TemporaryDirectory() as tempdir:
            package_path = os.path.join(tempdir, 'pipeline.yaml')
            compiler.Compiler().compile(
                pipeline_func=my_pipeline, package_path=package_path)

    def test_downstream_and_upstream_in_different_condition_on_same_level_blocked(
            self):

        with self.assertRaisesRegex(
                compiler_utils.InvalidTopologyException,
                r'Illegal task dependency across DSL context managers. A downstream task cannot depend on an upstream task within a dsl\.Condition context unless the downstream is within that context too\. Found task print-op-3 which depends on upstream task print-op-2 within an uncommon dsl\.Condition context\.'
        ):

            @dsl.pipeline()
            def my_pipeline():
                return_1_task = return_1()

                one = print_op(message='1')
                with dsl.Condition(return_1_task.output == 1):
                    two = print_op(message='2')

                with dsl.Condition(return_1_task.output == 1):
                    three = print_op(message='3').after(two)

            with tempfile.TemporaryDirectory() as tempdir:
                package_path = os.path.join(tempdir, 'pipeline.yaml')
                compiler.Compiler().compile(
                    pipeline_func=my_pipeline, package_path=package_path)

    def test_downstream_inside_deeper_nested_condition_permitted(self):

        @dsl.pipeline()
        def my_pipeline():
            return_1_task = return_1()
            return_1_task2 = return_1()

            with dsl.Condition(return_1_task.output == 1):
                one = return_1()
                with dsl.Condition(return_1_task2.output == 1):
                    two = print_op(message='2')
                    three = print_op(message=str(one.output))

        with tempfile.TemporaryDirectory() as tempdir:
            package_path = os.path.join(tempdir, 'pipeline.yaml')
            compiler.Compiler().compile(
                pipeline_func=my_pipeline, package_path=package_path)

    def test_upstream_inside_deeper_nested_condition_blocked(self):

        with self.assertRaisesRegex(
                compiler_utils.InvalidTopologyException,
                r'Illegal task dependency across DSL context managers\. A downstream task cannot depend on an upstream task within a dsl\.Condition context unless the downstream is within that context too\. Found task print-op-3 which depends on upstream task print-op-2 within an uncommon dsl\.Condition context\.'
        ):

            @dsl.pipeline()
            def my_pipeline():
                return_1_task = return_1()

                with dsl.Condition(return_1_task.output == 1):
                    one = print_op(message='1')
                    with dsl.Condition(return_1_task.output == 1):
                        two = print_op(message='2')
                    three = print_op(message='3').after(two)

            with tempfile.TemporaryDirectory() as tempdir:
                package_path = os.path.join(tempdir, 'pipeline.yaml')
                compiler.Compiler().compile(
                    pipeline_func=my_pipeline, package_path=package_path)

    def test_upstream_in_same_for_loop_with_downstream_permitted(self):

        @dsl.pipeline()
        def my_pipeline():
            args_generator = args_generator_op()

            with dsl.ParallelFor(args_generator.output):
                one = print_op(message='1')
                two = print_op(message='3').after(one)

        with tempfile.TemporaryDirectory() as tempdir:
            package_path = os.path.join(tempdir, 'pipeline.yaml')
            compiler.Compiler().compile(
                pipeline_func=my_pipeline, package_path=package_path)

    def test_downstream_not_in_same_for_loop_with_upstream_blocked(self):

        with self.assertRaisesRegex(
                compiler_utils.InvalidTopologyException,
                r'Illegal task dependency across DSL context managers\. A downstream task cannot depend on an upstream task within a dsl\.ParallelFor context unless the downstream is within that context too or the outputs are begin fanned-in to a list using dsl\.Collected\. Found task print-op-2 which depends on upstream task print-op within an uncommon dsl\.ParallelFor context\.'
        ):

            @dsl.pipeline()
            def my_pipeline():
                args_generator = args_generator_op()

                with dsl.ParallelFor(args_generator.output):
                    one = print_op(message='1')
                two = print_op(message='3').after(one)

            with tempfile.TemporaryDirectory() as tempdir:
                package_path = os.path.join(tempdir, 'pipeline.yaml')
                compiler.Compiler().compile(
                    pipeline_func=my_pipeline, package_path=package_path)

    def test_downstream_not_in_same_for_loop_with_upstream_seperate_blocked(
            self):

        with self.assertRaisesRegex(
                compiler_utils.InvalidTopologyException,
                r'Illegal task dependency across DSL context managers\. A downstream task cannot depend on an upstream task within a dsl\.ParallelFor context unless the downstream is within that context too or the outputs are begin fanned-in to a list using dsl\.Collected\. Found task print-op-2 which depends on upstream task print-op within an uncommon dsl\.ParallelFor context\.'
        ):

            @dsl.pipeline()
            def my_pipeline():
                args_generator = args_generator_op()

                with dsl.ParallelFor(args_generator.output):
                    one = print_op(message='1')

                with dsl.ParallelFor(args_generator.output):
                    two = print_op(message='3').after(one)

            with tempfile.TemporaryDirectory() as tempdir:
                package_path = os.path.join(tempdir, 'pipeline.yaml')
                compiler.Compiler().compile(
                    pipeline_func=my_pipeline, package_path=package_path)

    def test_downstream_not_in_same_for_loop_with_upstream_nested_blocked(self):

        with self.assertRaisesRegex(
                compiler_utils.InvalidTopologyException,
                r'Downstream tasks in a nested ParallelFor group cannot depend on an upstream task in a shallower ParallelFor group.'
        ):

            @dsl.pipeline()
            def my_pipeline():
                args_generator = args_generator_op()

                with dsl.ParallelFor(args_generator.output):
                    one = print_op(message='1')

                    with dsl.ParallelFor(args_generator.output):
                        two = print_op(message='3').after(one)

            with tempfile.TemporaryDirectory() as tempdir:
                package_path = os.path.join(tempdir, 'pipeline.yaml')
                compiler.Compiler().compile(
                    pipeline_func=my_pipeline, package_path=package_path)

    def test_downstream_in_condition_nested_in_a_for_loop(self):

        @dsl.pipeline()
        def my_pipeline():
            return_1_task = return_1()

            with dsl.ParallelFor([1, 2, 3]):
                one = print_op(message='1')
                with dsl.Condition(return_1_task.output == 1):
                    two = print_op(message='2').after(one)

        with tempfile.TemporaryDirectory() as tempdir:
            package_path = os.path.join(tempdir, 'pipeline.yaml')
            compiler.Compiler().compile(
                pipeline_func=my_pipeline, package_path=package_path)

    def test_downstream_in_a_for_loop_nested_in_a_condition(self):

        @dsl.pipeline()
        def my_pipeline():
            return_1_task = return_1()

            with dsl.Condition(return_1_task.output == 1):
                one = print_op(message='1')
                with dsl.ParallelFor([1, 2, 3]):
                    two = print_op(message='2').after(one)

        with tempfile.TemporaryDirectory() as tempdir:
            package_path = os.path.join(tempdir, 'pipeline.yaml')
            compiler.Compiler().compile(
                pipeline_func=my_pipeline, package_path=package_path)

    def test_downstream_in_a_nested_for_loop_not_related_to_upstream(self):

        @dsl.pipeline()
        def my_pipeline():
            return_1_task = return_1()

            with dsl.ParallelFor([1, 2, 3]):
                one = print_op(message='1')
                with dsl.ParallelFor([1, 2, 3]):
                    two = print_op(message='2').after(return_1_task)

        with tempfile.TemporaryDirectory() as tempdir:
            package_path = os.path.join(tempdir, 'pipeline.yaml')
            compiler.Compiler().compile(
                pipeline_func=my_pipeline, package_path=package_path)


class TestCannotUseAfterCrossDAG(unittest.TestCase):

    def test_inner_task_prevented(self):
        with self.assertRaisesRegex(
                compiler_utils.InvalidTopologyException,
                r'Illegal task dependency across DSL context managers\. A downstream task cannot depend on an upstream task within a dsl\.ExitHandler context unless the downstream is within that context too\. Found task print-op-4 which depends on upstream task print-op-2 within an uncommon dsl\.ExitHandler context\.'
        ):

            @dsl.component
            def print_op(message: str):
                print(message)

            @dsl.pipeline(name='pipeline-with-multiple-exit-handlers')
            def my_pipeline():
                first_exit_task = print_op(message='First exit task.')

                with dsl.ExitHandler(first_exit_task):
                    first_print_op = print_op(
                        message='Inside first exit handler.')

                second_exit_task = print_op(message='Second exit task.')
                with dsl.ExitHandler(second_exit_task):
                    print_op(message='Inside second exit handler.').after(
                        first_print_op)

            with tempfile.TemporaryDirectory() as tempdir:
                package_path = os.path.join(tempdir, 'pipeline.yaml')
                compiler.Compiler().compile(
                    pipeline_func=my_pipeline, package_path=package_path)

    def test_exit_handler_task_prevented(self):
        with self.assertRaisesRegex(
                compiler_utils.InvalidTopologyException,
                r'Illegal task dependency across DSL context managers\. A downstream task cannot depend on an upstream task within a dsl\.ExitHandler context unless the downstream is within that context too\. Found task print-op-4 which depends on upstream task print-op-2 within an uncommon dsl\.ExitHandler context\.'
        ):

            @dsl.component
            def print_op(message: str):
                print(message)

            @dsl.pipeline(name='pipeline-with-multiple-exit-handlers')
            def my_pipeline():
                first_exit_task = print_op(message='First exit task.')

                with dsl.ExitHandler(first_exit_task):
                    first_print_op = print_op(
                        message='Inside first exit handler.')

                second_exit_task = print_op(message='Second exit task.')
                with dsl.ExitHandler(second_exit_task):
                    x = print_op(message='Inside second exit handler.')
                    x.after(first_print_op)

            with tempfile.TemporaryDirectory() as tempdir:
                package_path = os.path.join(tempdir, 'pipeline.yaml')
                compiler.Compiler().compile(
                    pipeline_func=my_pipeline, package_path=package_path)

    def test_within_same_exit_handler_permitted(self):

        @dsl.component
        def print_op(message: str):
            print(message)

        @dsl.pipeline(name='pipeline-with-multiple-exit-handlers')
        def my_pipeline():
            first_exit_task = print_op(message='First exit task.')

            with dsl.ExitHandler(first_exit_task):
                first_print_op = print_op(
                    message='First task inside first exit handler.')
                second_print_op = print_op(
                    message='Second task inside first exit handler.').after(
                        first_print_op)

            second_exit_task = print_op(message='Second exit task.')
            with dsl.ExitHandler(second_exit_task):
                print_op(message='Inside second exit handler.')

        with tempfile.TemporaryDirectory() as tempdir:
            package_path = os.path.join(tempdir, 'pipeline.yaml')
            compiler.Compiler().compile(
                pipeline_func=my_pipeline, package_path=package_path)

    def test_outside_of_condition_blocked(self):
        with self.assertRaisesRegex(
                compiler_utils.InvalidTopologyException,
                r'Illegal task dependency across DSL context managers\. A downstream task cannot depend on an upstream task within a dsl\.Condition context unless the downstream is within that context too\. Found task print-op-3 which depends on upstream task print-op within an uncommon dsl\.Condition context\.'
        ):

            @dsl.component
            def print_op(message: str):
                print(message)

            @dsl.component
            def return_1() -> int:
                return 1

            @dsl.pipeline(name='pipeline-with-multiple-exit-handlers')
            def my_pipeline():
                return_1_task = return_1()

                with dsl.Condition(return_1_task.output == 1):
                    one = print_op(message='1')
                    two = print_op(message='2')
                three = print_op(message='3').after(one)

            with tempfile.TemporaryDirectory() as tempdir:
                package_path = os.path.join(tempdir, 'pipeline.yaml')
                compiler.Compiler().compile(
                    pipeline_func=my_pipeline, package_path=package_path)

    def test_inside_of_condition_permitted(self):

        @dsl.component
        def print_op(message: str):
            print(message)

        @dsl.component
        def return_1() -> int:
            return 1

        @dsl.pipeline(name='pipeline-with-multiple-exit-handlers')
        def my_pipeline():
            return_1_task = return_1()

            with dsl.Condition(return_1_task.output == '1'):
                one = print_op(message='1')
                two = print_op(message='2').after(one)
            three = print_op(message='3')

        with tempfile.TemporaryDirectory() as tempdir:
            package_path = os.path.join(tempdir, 'pipeline.yaml')
            compiler.Compiler().compile(
                pipeline_func=my_pipeline, package_path=package_path)


@dsl.component
def identity(string: str, model: bool) -> str:
    return string


class TestYamlComments(unittest.TestCase):

    def test_comments_include_inputs_and_outputs_and_pipeline_name(self):

        @dsl.pipeline()
        def my_pipeline(sample_input1: bool = True,
                        sample_input2: str = 'string') -> str:
            op1 = identity(string=sample_input2, model=sample_input1)
            result = op1.output
            return result

        with tempfile.TemporaryDirectory() as tmpdir:
            pipeline_spec_path = os.path.join(tmpdir, 'output.yaml')
            compiler.Compiler().compile(
                pipeline_func=my_pipeline, package_path=pipeline_spec_path)
            with open(pipeline_spec_path, 'r+') as f:
                yaml_content = f.read()

        inputs_string = textwrap.dedent("""\
                # Inputs:
                #    sample_input1: bool [Default: True]
                #    sample_input2: str [Default: 'string']
                """)

        outputs_string = textwrap.dedent("""\
                # Outputs:
                #    Output: str
                """)

        name_string = '# Name: my-pipeline'

        self.assertIn(name_string, yaml_content)

        self.assertIn(inputs_string, yaml_content)

        self.assertIn(outputs_string, yaml_content)

    def test_no_description(self):

        @dsl.pipeline()
        def pipeline_with_no_description(sample_input1: bool = True,
                                         sample_input2: str = 'string') -> str:
            op1 = identity(string=sample_input2, model=sample_input1)
            result = op1.output
            return result

        with tempfile.TemporaryDirectory() as tmpdir:
            pipeline_spec_path = os.path.join(tmpdir, 'output.yaml')
            compiler.Compiler().compile(
                pipeline_func=pipeline_with_no_description,
                package_path=pipeline_spec_path)
            with open(pipeline_spec_path, 'r+') as f:
                yaml_content = f.read()

            # load and recompile to ensure idempotent description
            loaded_pipeline = components.load_component_from_file(
                pipeline_spec_path)

            compiler.Compiler().compile(
                pipeline_func=loaded_pipeline, package_path=pipeline_spec_path)

            with open(pipeline_spec_path, 'r+') as f:
                reloaded_yaml_content = f.read()

        comment_description = '# Description:'
        self.assertNotIn(comment_description, yaml_content)
        self.assertNotIn(comment_description, reloaded_yaml_content)
        proto_description = ''
        self.assertEqual(
            pipeline_with_no_description.pipeline_spec.pipeline_info
            .description, proto_description)
        self.assertEqual(
            loaded_pipeline.pipeline_spec.pipeline_info.description,
            proto_description)

    def test_description_from_docstring(self):

        @dsl.pipeline()
        def pipeline_with_description(sample_input1: bool = True,
                                      sample_input2: str = 'string') -> str:
            """This is a description of this pipeline."""
            op1 = identity(string=sample_input2, model=sample_input1)
            result = op1.output
            return result

        with tempfile.TemporaryDirectory() as tmpdir:
            pipeline_spec_path = os.path.join(tmpdir, 'output.yaml')
            compiler.Compiler().compile(
                pipeline_func=pipeline_with_description,
                package_path=pipeline_spec_path)
            with open(pipeline_spec_path, 'r+') as f:
                yaml_content = f.read()

            # load and recompile to ensure idempotent description
            loaded_pipeline = components.load_component_from_file(
                pipeline_spec_path)

            compiler.Compiler().compile(
                pipeline_func=loaded_pipeline, package_path=pipeline_spec_path)

            with open(pipeline_spec_path, 'r+') as f:
                reloaded_yaml_content = f.read()

        comment_description = '# Description: This is a description of this pipeline.'
        self.assertIn(comment_description, yaml_content)
        self.assertIn(comment_description, reloaded_yaml_content)
        proto_description = 'This is a description of this pipeline.'
        self.assertEqual(
            pipeline_with_description.pipeline_spec.pipeline_info.description,
            proto_description)
        self.assertEqual(
            loaded_pipeline.pipeline_spec.pipeline_info.description,
            proto_description)

    def test_description_from_decorator(self):

        @dsl.pipeline(description='Prefer this description.')
        def pipeline_with_description(sample_input1: bool = True,
                                      sample_input2: str = 'string') -> str:
            """Don't prefer this description."""
            op1 = identity(string=sample_input2, model=sample_input1)
            result = op1.output
            return result

        with tempfile.TemporaryDirectory() as tmpdir:
            pipeline_spec_path = os.path.join(tmpdir, 'output.yaml')
            compiler.Compiler().compile(
                pipeline_func=pipeline_with_description,
                package_path=pipeline_spec_path)
            with open(pipeline_spec_path, 'r+') as f:
                yaml_content = f.read()

            # load and recompile to ensure idempotent description
            loaded_pipeline = components.load_component_from_file(
                pipeline_spec_path)

            compiler.Compiler().compile(
                pipeline_func=loaded_pipeline, package_path=pipeline_spec_path)

            with open(pipeline_spec_path, 'r+') as f:
                reloaded_yaml_content = f.read()

        comment_description = '# Description: Prefer this description.'
        self.assertIn(comment_description, yaml_content)
        self.assertIn(loaded_pipeline.pipeline_spec.pipeline_info.description,
                      reloaded_yaml_content)
        proto_description = 'Prefer this description.'
        self.assertEqual(
            pipeline_with_description.pipeline_spec.pipeline_info.description,
            proto_description)
        self.assertEqual(
            loaded_pipeline.pipeline_spec.pipeline_info.description,
            proto_description)

    def test_comments_on_pipeline_with_no_inputs_or_outputs(self):

        @dsl.pipeline()
        def pipeline_with_no_inputs() -> str:
            op1 = identity(string='string', model=True)
            result = op1.output
            return result

        with tempfile.TemporaryDirectory() as tmpdir:
            pipeline_spec_path = os.path.join(tmpdir, 'output.yaml')
            compiler.Compiler().compile(
                pipeline_func=pipeline_with_no_inputs,
                package_path=pipeline_spec_path)

            with open(pipeline_spec_path, 'r+') as f:
                yaml_content = f.read()

        inputs_string = '# Inputs:'

        self.assertNotIn(inputs_string, yaml_content)

        @dsl.pipeline()
        def pipeline_with_no_outputs(sample_input1: bool = True,
                                     sample_input2: str = 'string'):
            identity(string=sample_input2, model=sample_input1)

        with tempfile.TemporaryDirectory() as tmpdir:
            pipeline_spec_path = os.path.join(tmpdir, 'output.yaml')
            compiler.Compiler().compile(
                pipeline_func=pipeline_with_no_outputs,
                package_path=pipeline_spec_path)

            with open(pipeline_spec_path, 'r+') as f:
                yaml_content = f.read()

        outputs_string = '# Outputs:'

        self.assertNotIn(outputs_string, yaml_content)

    def test_comments_follow_pattern(self):

        @dsl.pipeline()
        def my_pipeline(sample_input1: bool = True,
                        sample_input2: str = 'string') -> str:
            """This is a definition of this pipeline."""
            op1 = identity(string=sample_input2, model=sample_input1)
            result = op1.output
            return result

        with tempfile.TemporaryDirectory() as tmpdir:
            pipeline_spec_path = os.path.join(tmpdir, 'output.yaml')
            compiler.Compiler().compile(
                pipeline_func=my_pipeline, package_path=pipeline_spec_path)

            with open(pipeline_spec_path, 'r+') as f:
                yaml_content = f.read()

        pattern_sample = textwrap.dedent("""\
                # PIPELINE DEFINITION
                # Name: my-pipeline
                # Description: This is a definition of this pipeline.
                # Inputs:
                #    sample_input1: bool [Default: True]
                #    sample_input2: str [Default: 'string']
                # Outputs:
                #    Output: str
                """)

        self.assertIn(pattern_sample, yaml_content)

    def test_verbose_comment_characteristics(self):

        @dsl.component
        def output_model(metrics: Output[Model]):
            """Dummy component that outputs metrics with a random accuracy."""
            import random
            result = random.randint(0, 100)
            metrics.log_metric('accuracy', result)

        @dsl.pipeline(name='Test pipeline')
        def my_pipeline(sample_input1: bool,
                        sample_input2: str,
                        sample_input3: Input[Model],
                        sample_input4: float = 3.14,
                        sample_input5: list = [1],
                        sample_input6: dict = {'one': 1},
                        sample_input7: int = 5) -> Model:
            """This is a definition of this pipeline."""

            task = output_model()
            return task.output

        with tempfile.TemporaryDirectory() as tmpdir:
            pipeline_spec_path = os.path.join(tmpdir, 'output.yaml')
            compiler.Compiler().compile(
                pipeline_func=my_pipeline, package_path=pipeline_spec_path)

            with open(pipeline_spec_path, 'r+') as f:
                yaml_content = f.read()

        predicted_comment = textwrap.dedent("""\
                # PIPELINE DEFINITION
                # Name: test-pipeline
                # Description: This is a definition of this pipeline.
                # Inputs:
                #    sample_input1: bool
                #    sample_input2: str
                #    sample_input3: system.Model
                #    sample_input4: float [Default: 3.14]
                #    sample_input5: list [Default: [1.0]]
                #    sample_input6: dict [Default: {'one': 1.0}]
                #    sample_input7: int [Default: 5.0]
                # Outputs:
                #    Output: system.Model
                """)

        self.assertIn(predicted_comment, yaml_content)

    def test_comments_on_compiled_components(self):

        @dsl.component
        def my_component(string: str, model: bool) -> str:
            """component description."""
            return string

        with tempfile.TemporaryDirectory() as tmpdir:
            pipeline_spec_path = os.path.join(tmpdir, 'output.yaml')
            compiler.Compiler().compile(
                pipeline_func=my_component, package_path=pipeline_spec_path)

            with open(pipeline_spec_path, 'r+') as f:
                yaml_content = f.read()

        predicted_comment = textwrap.dedent("""\
                # PIPELINE DEFINITION
                # Name: my-component
                # Description: component description.
                # Inputs:
                #    model: bool
                #    string: str
                """)

        self.assertIn(predicted_comment, yaml_content)

        @dsl.container_component
        def my_container_component(text: str, output_path: OutputPath(str)):
            """component description."""
            return ContainerSpec(
                image='python:3.7',
                command=['my_program', text],
                args=['--output_path', output_path])

        with tempfile.TemporaryDirectory() as tmpdir:
            pipeline_spec_path = os.path.join(tmpdir, 'output.yaml')
            compiler.Compiler().compile(
                pipeline_func=my_container_component,
                package_path=pipeline_spec_path)

            with open(pipeline_spec_path, 'r+') as f:
                yaml_content = f.read()

        predicted_comment = textwrap.dedent("""\
                # PIPELINE DEFINITION
                # Name: my-container-component
                # Description: component description.
                # Inputs:
                #    text: str
                """)

        self.assertIn(predicted_comment, yaml_content)

    def test_comments_idempotency(self):

        @dsl.pipeline()
        def my_pipeline(sample_input1: bool = True,
                        sample_input2: str = 'string') -> str:
            """My description."""
            op1 = identity(string=sample_input2, model=sample_input1)
            result = op1.output
            return result

        with tempfile.TemporaryDirectory() as tmpdir:
            pipeline_spec_path = os.path.join(tmpdir, 'output.yaml')
            compiler.Compiler().compile(
                pipeline_func=my_pipeline, package_path=pipeline_spec_path)
            with open(pipeline_spec_path, 'r+') as f:
                yaml_content = f.read()
            comp = components.load_component_from_file(pipeline_spec_path)
            compiler.Compiler().compile(
                pipeline_func=comp, package_path=pipeline_spec_path)
            with open(pipeline_spec_path, 'r+') as f:
                reloaded_yaml_content = f.read()

        predicted_comment = textwrap.dedent("""\
                # PIPELINE DEFINITION
                # Name: my-pipeline
                # Description: My description.
                # Inputs:
                #    sample_input1: bool [Default: True]
                #    sample_input2: str [Default: 'string']
                # Outputs:
                #    Output: str
                """)

        # test initial comments
        self.assertIn(predicted_comment, yaml_content)

        # test reloaded comments
        self.assertIn(predicted_comment, reloaded_yaml_content)

    def test_comment_with_multiline_docstring(self):

        @dsl.pipeline()
        def pipeline_with_multiline_definition(
                sample_input1: bool = True,
                sample_input2: str = 'string') -> str:
            """docstring short description.
            docstring long description. docstring long description.
            """
            op1 = identity(string=sample_input2, model=sample_input1)
            result = op1.output
            return result

        with tempfile.TemporaryDirectory() as tmpdir:
            pipeline_spec_path = os.path.join(tmpdir, 'output.yaml')
            compiler.Compiler().compile(
                pipeline_func=pipeline_with_multiline_definition,
                package_path=pipeline_spec_path)

            with open(pipeline_spec_path, 'r+') as f:
                yaml_content = f.read()

        description_string = textwrap.dedent("""\
            # Description: docstring short description.
            #              docstring long description. docstring long description.
            """)

        self.assertIn(description_string, yaml_content)

        @dsl.pipeline()
        def pipeline_with_multiline_definition(
                sample_input1: bool = True,
                sample_input2: str = 'string') -> str:
            """
            docstring long description.
            docstring long description.
            docstring long description.
            """
            op1 = identity(string=sample_input2, model=sample_input1)
            result = op1.output
            return result

        with tempfile.TemporaryDirectory() as tmpdir:
            pipeline_spec_path = os.path.join(tmpdir, 'output.yaml')
            compiler.Compiler().compile(
                pipeline_func=pipeline_with_multiline_definition,
                package_path=pipeline_spec_path)

            with open(pipeline_spec_path, 'r+') as f:
                yaml_content = f.read()

        description_string = textwrap.dedent("""\
            # Description: docstring long description.
            #              docstring long description.
            #              docstring long description.
            """)

        self.assertIn(description_string, yaml_content)

    def test_idempotency_on_comment_with_multiline_docstring(self):

        @dsl.pipeline()
        def my_pipeline(sample_input1: bool = True,
                        sample_input2: str = 'string') -> str:
            """docstring short description.
            docstring long description.
            docstring long description.
            """
            op1 = identity(string=sample_input2, model=sample_input1)
            result = op1.output
            return result

        with tempfile.TemporaryDirectory() as tmpdir:
            pipeline_spec_path = os.path.join(tmpdir, 'output.yaml')
            compiler.Compiler().compile(
                pipeline_func=my_pipeline, package_path=pipeline_spec_path)
            with open(pipeline_spec_path, 'r+') as f:
                yaml_content = f.read()
            comp = components.load_component_from_file(pipeline_spec_path)
            compiler.Compiler().compile(
                pipeline_func=comp, package_path=pipeline_spec_path)
            with open(pipeline_spec_path, 'r+') as f:
                reloaded_yaml_content = f.read()

        predicted_comment = textwrap.dedent("""\
                # PIPELINE DEFINITION
                # Name: my-pipeline
                # Description: docstring short description.
                #              docstring long description.
                #              docstring long description.
                # Inputs:
                #    sample_input1: bool [Default: True]
                #    sample_input2: str [Default: 'string']
                # Outputs:
                #    Output: str
                """)

        # test initial comments
        self.assertIn(predicted_comment, yaml_content)

        # test reloaded comments
        self.assertIn(predicted_comment, reloaded_yaml_content)


class TestCompileThenLoadThenUseWithOptionalInputs(unittest.TestCase):

    def test__component__param__None_default(self):

        @dsl.component
        def comp(x: Optional[int] = None):
            print(x)

        @dsl.pipeline
        def my_pipeline():
            comp()

        # test can use without args before compile
        self.assertTrue(my_pipeline.pipeline_spec.components['comp-comp']
                        .input_definitions.parameters['x'].is_optional)

        with tempfile.TemporaryDirectory() as tmpdir:
            path = os.path.join(tmpdir, 'comp.yaml')
            compiler.Compiler().compile(comp, path)
            loaded_comp = components.load_component_from_file(path)

        @dsl.pipeline
        def my_pipeline():
            loaded_comp()

        # test can use without args after compile and load
        self.assertTrue(my_pipeline.pipeline_spec.components['comp-comp']
                        .input_definitions.parameters['x'].is_optional)

    def test__component__param__non_None_default(self):

        @dsl.component
        def comp(x: int = 1):
            print(x)

        @dsl.pipeline
        def my_pipeline():
            comp()

        self.assertIn('comp-comp', my_pipeline.pipeline_spec.components.keys())

        with tempfile.TemporaryDirectory() as tmpdir:
            path = os.path.join(tmpdir, 'comp.yaml')
            compiler.Compiler().compile(comp, path)
            loaded_comp = components.load_component_from_file(path)

        @dsl.pipeline
        def my_pipeline():
            loaded_comp()

        self.assertTrue(my_pipeline.pipeline_spec.components['comp-comp']
                        .input_definitions.parameters['x'].is_optional)

    def test__pipeline__param__None_default(self):

        @dsl.component
        def comp(x: Optional[int] = None):
            print(x)

        @dsl.pipeline
        def inner_pipeline(x: Optional[int] = None):
            comp(x=x)

        @dsl.pipeline
        def my_pipeline():
            inner_pipeline()

        self.assertTrue(my_pipeline.pipeline_spec.components['comp-comp']
                        .input_definitions.parameters['x'].is_optional)
        self.assertTrue(
            my_pipeline.pipeline_spec.components['comp-inner-pipeline']
            .input_definitions.parameters['x'].is_optional)

        with tempfile.TemporaryDirectory() as tmpdir:
            path = os.path.join(tmpdir, 'comp.yaml')
            compiler.Compiler().compile(inner_pipeline, path)
            loaded_comp = components.load_component_from_file(path)

        @dsl.pipeline
        def my_pipeline():
            loaded_comp()

        self.assertTrue(my_pipeline.pipeline_spec.components['comp-comp']
                        .input_definitions.parameters['x'].is_optional)
        self.assertTrue(
            my_pipeline.pipeline_spec.components['comp-inner-pipeline']
            .input_definitions.parameters['x'].is_optional)

    def test__pipeline__param__non_None_default(self):

        @dsl.component
        def comp(x: Optional[int] = None):
            print(x)

        @dsl.pipeline
        def inner_pipeline(x: int = 1):
            comp(x=x)

        @dsl.pipeline
        def my_pipeline():
            inner_pipeline()

        self.assertTrue(my_pipeline.pipeline_spec.components['comp-comp']
                        .input_definitions.parameters['x'].is_optional)
        self.assertTrue(
            my_pipeline.pipeline_spec.components['comp-inner-pipeline']
            .input_definitions.parameters['x'].is_optional)

        with tempfile.TemporaryDirectory() as tmpdir:
            path = os.path.join(tmpdir, 'comp.yaml')
            compiler.Compiler().compile(inner_pipeline, path)
            loaded_comp = components.load_component_from_file(path)

        @dsl.pipeline
        def my_pipeline():
            loaded_comp()

        self.assertTrue(my_pipeline.pipeline_spec.components['comp-comp']
                        .input_definitions.parameters['x'].is_optional)
        self.assertTrue(
            my_pipeline.pipeline_spec.components['comp-inner-pipeline']
            .input_definitions.parameters['x'].is_optional)

    def test__component__artifact(self):

        @dsl.component
        def comp(x: Optional[Input[Artifact]] = None):
            print(x)

        @dsl.pipeline
        def my_pipeline():
            comp()

        self.assertTrue(my_pipeline.pipeline_spec.components['comp-comp']
                        .input_definitions.artifacts['x'].is_optional)

        with tempfile.TemporaryDirectory() as tmpdir:
            path = os.path.join(tmpdir, 'comp.yaml')
            compiler.Compiler().compile(comp, path)
            loaded_comp = components.load_component_from_file(path)

        @dsl.pipeline
        def my_pipeline():
            loaded_comp()

        self.assertTrue(my_pipeline.pipeline_spec.components['comp-comp']
                        .input_definitions.artifacts['x'].is_optional)

    def test__pipeline__artifact(self):

        @dsl.component
        def comp(x: Optional[Input[Artifact]] = None):
            print(x)

        @dsl.pipeline
        def inner_pipeline(x: Optional[Input[Artifact]] = None):
            comp(x=x)

        @dsl.pipeline
        def my_pipeline():
            inner_pipeline()

        self.assertTrue(my_pipeline.pipeline_spec.components['comp-comp']
                        .input_definitions.artifacts['x'].is_optional)

        with tempfile.TemporaryDirectory() as tmpdir:
            path = os.path.join(tmpdir, 'comp.yaml')
            compiler.Compiler().compile(comp, path)
            loaded_comp = components.load_component_from_file(path)

        @dsl.pipeline
        def my_pipeline():
            loaded_comp()

        self.assertTrue(my_pipeline.pipeline_spec.components['comp-comp']
                        .input_definitions.artifacts['x'].is_optional)


class TestCompileOptionalArtifacts(unittest.TestCase):

    def test_python_comp(self):

        @dsl.component
        def comp(x: Optional[Input[Artifact]] = None):
            print(x)

        artifact_spec_from_root = comp.pipeline_spec.root.input_definitions.artifacts[
            'x']
        self.assertTrue(artifact_spec_from_root.is_optional)

        artifact_spec_from_comp = comp.pipeline_spec.components[
            'comp-comp'].input_definitions.artifacts['x']
        self.assertTrue(artifact_spec_from_comp.is_optional)

    def test_python_comp_with_model(self):

        @dsl.component
        def comp(x: Optional[Input[Model]] = None):
            print(x)

        artifact_spec_from_root = comp.pipeline_spec.root.input_definitions.artifacts[
            'x']
        self.assertTrue(artifact_spec_from_root.is_optional)

        artifact_spec_from_comp = comp.pipeline_spec.components[
            'comp-comp'].input_definitions.artifacts['x']
        self.assertTrue(artifact_spec_from_comp.is_optional)

    def test_python_comp_without_optional_type_modifier(self):

        @dsl.component
        def comp(x: Input[Model] = None):
            print(x)

        artifact_spec_from_root = comp.pipeline_spec.root.input_definitions.artifacts[
            'x']
        self.assertTrue(artifact_spec_from_root.is_optional)

        artifact_spec_from_comp = comp.pipeline_spec.components[
            'comp-comp'].input_definitions.artifacts['x']
        self.assertTrue(artifact_spec_from_comp.is_optional)

    def test_container_comp(self):

        @dsl.container_component
        def comp(x: Optional[Input[Artifact]] = None):
            return dsl.ContainerSpec(
                image='alpine',
                command=[
                    dsl.IfPresentPlaceholder(
                        input_name='x',
                        then=['echo', x.uri],
                        else_=['echo', 'No artifact provided!'])
                ])

        artifact_spec_from_root = comp.pipeline_spec.root.input_definitions.artifacts[
            'x']
        self.assertTrue(artifact_spec_from_root.is_optional)

        artifact_spec_from_comp = comp.pipeline_spec.components[
            'comp-comp'].input_definitions.artifacts['x']
        self.assertTrue(artifact_spec_from_comp.is_optional)

    def test_pipeline(self):

        @dsl.component
        def comp():
            print('hello')

        @dsl.pipeline
        def my_pipeline(x: Optional[Input[Artifact]] = None):
            comp()

        artifact_spec_from_root = my_pipeline.pipeline_spec.root.input_definitions.artifacts[
            'x']
        self.assertTrue(artifact_spec_from_root.is_optional)

    def test_pipeline_without_optional_type_modifier(self):

        @dsl.component
        def comp():
            print('hello')

        @dsl.pipeline
        def my_pipeline(x: Input[Artifact] = None):
            comp()

        artifact_spec_from_root = my_pipeline.pipeline_spec.root.input_definitions.artifacts[
            'x']
        self.assertTrue(artifact_spec_from_root.is_optional)

    def test_pipeline_and_inner_component_together(self):

        @dsl.component
        def comp(x: Optional[Input[Model]] = None):
            print(x)

        @dsl.pipeline
        def my_pipeline(x: Optional[Input[Artifact]] = None):
            comp()

        artifact_spec_from_root = my_pipeline.pipeline_spec.root.input_definitions.artifacts[
            'x']
        self.assertTrue(artifact_spec_from_root.is_optional)

        artifact_spec_from_comp = my_pipeline.pipeline_spec.components[
            'comp-comp'].input_definitions.artifacts['x']
        self.assertTrue(artifact_spec_from_comp.is_optional)

    def test_invalid_default_comp(self):
        with self.assertRaisesRegex(
                ValueError,
                'Optional Input artifacts may only have default value None'):

            @dsl.component
            def comp(x: Optional[Input[Model]] = 1):
                print(x)

        with self.assertRaisesRegex(
                ValueError,
                'Optional Input artifacts may only have default value None'):

            @dsl.component
            def comp(x: Optional[Input[Model]] = Model(
                name='', uri='', metadata={})):
                print(x)

    def test_invalid_default_pipeline(self):

        @dsl.component
        def comp():
            print('hello')

        with self.assertRaisesRegex(
                ValueError,
                'Optional Input artifacts may only have default value None'):

            @dsl.pipeline
            def my_pipeline(x: Input[Artifact] = 1):
                comp()

        with self.assertRaisesRegex(
                ValueError,
                'Optional Input artifacts may only have default value None'):

            @dsl.pipeline
            def my_pipeline(x: Input[Artifact] = Artifact(
                name='', uri='', metadata={})):
                comp()


class TestCrossTasksGroupFanInCollection(unittest.TestCase):

    def test_missing_collected_with_correct_annotation(self):
        from typing import List

        from kfp import dsl

        @dsl.component
        def double(num: int) -> int:
            return 2 * num

        @dsl.component
        def add(nums: List[int]) -> int:
            return sum(nums)

        with self.assertRaisesRegex(
                type_utils.InconsistentTypeException,
                "Argument type 'NUMBER_INTEGER' is incompatible with the input type 'LIST'"
        ):

            @dsl.pipeline
            def math_pipeline() -> int:
                with dsl.ParallelFor([1, 2, 3]) as v:
                    t = double(num=v)

                return add(nums=t.output).output

    def test_missing_collected_with_incorrect_annotation(self):

        @dsl.component
        def double(num: int) -> int:
            return 2 * num

        @dsl.component
        def add(nums: int) -> int:
            return nums

        # the annotation is incorrect, but the user didn't use dsl.Collected
        with self.assertRaisesRegex(
                compiler_utils.InvalidTopologyException,
                r'Illegal task dependency across DSL context managers\. A downstream task cannot depend on an upstream task within a dsl\.ParallelFor context unless the downstream is within that context too or the outputs are begin fanned-in to a list using dsl\.Collected\. Found task add which depends on upstream task double within an uncommon dsl\.ParallelFor context\.'
        ):

            @dsl.pipeline
            def math_pipeline() -> int:
                with dsl.ParallelFor([1, 2, 3]) as v:
                    t = double(num=v)

                return add(nums=t.output).output

    def test_producer_condition_legal1(self):
        from kfp import dsl

        @dsl.component
        def double(num: int) -> int:
            return 2 * num

        @dsl.component
        def add(nums: List[int]) -> int:
            return sum(nums)

        @dsl.pipeline
        def math_pipeline(text: str) -> int:
            with dsl.Condition(text == 'text'):
                with dsl.ParallelFor([1, 2, 3]) as v:
                    t = double(num=v)

                return add(nums=dsl.Collected(t.output)).output

    def test_producer_condition_legal2(self):

        @dsl.component
        def double(num: int) -> int:
            return 2 * num

        @dsl.component
        def add(nums: List[int]) -> int:
            return sum(nums)

        @dsl.pipeline
        def my_pipeline(a: str):
            with dsl.ParallelFor([1, 2, 3]) as v:
                with dsl.Condition(v == 1):
                    t = double(num=v)

            with dsl.Condition(a == 'a'):
                x = add(nums=dsl.Collected(t.output))

    def test_producer_condition_illegal1(self):

        @dsl.component
        def double(num: int) -> int:
            return 2 * num

        @dsl.component
        def add(nums: List[int]) -> int:
            return sum(nums)

        with self.assertRaisesRegex(
                compiler_utils.InvalidTopologyException,
                r'Illegal task dependency across DSL context managers\. When using dsl\.Collected to fan-in outputs from a task within a dsl\.ParallelFor context, the dsl\.ParallelFor context manager cannot be nested within a dsl.Condition context manager unless the consumer task is too\. Task add consumes from double within a dsl\.Condition context\.'
        ):

            @dsl.pipeline
            def my_pipeline(a: str = '', b: str = ''):
                with dsl.Condition(a == 'a'):
                    with dsl.ParallelFor([1, 2, 3]) as v:
                        t = double(num=v)

                with dsl.Condition(b == 'b'):
                    x = add(nums=dsl.Collected(t.output))

    def test_producer_condition_illegal2(self):

        @dsl.component
        def double(num: int) -> int:
            return 2 * num

        @dsl.component
        def add(nums: List[int]) -> int:
            return sum(nums)

        with self.assertRaisesRegex(
                compiler_utils.InvalidTopologyException,
                r'Illegal task dependency across DSL context managers\. When using dsl\.Collected to fan-in outputs from a task within a dsl\.ParallelFor context, the dsl\.ParallelFor context manager cannot be nested within a dsl\.Condition context manager unless the consumer task is too\. Task add consumes from double within a dsl\.Condition context\.'
        ):

            @dsl.pipeline
            def my_pipeline(a: str = ''):
                with dsl.Condition(a == 'a'):
                    with dsl.ParallelFor([1, 2, 3]) as v:
                        t = double(num=v)
                add(nums=dsl.Collected(t.output))

    def test_producer_exit_handler_illegal1(self):

        @dsl.component
        def double(num: int) -> int:
            return 2 * num

        @dsl.component
        def exit_comp():
            print('Running exit task!')

        @dsl.component
        def add(nums: List[int]) -> int:
            return sum(nums)

        with self.assertRaisesRegex(
                compiler_utils.InvalidTopologyException,
                r'Illegal task dependency across DSL context managers\. When using dsl\.Collected to fan-in outputs from a task within a dsl\.ParallelFor context, the dsl\.ParallelFor context manager cannot be nested within a dsl\.ExitHandler context manager unless the consumer task is too\. Task add consumes from double within a dsl\.ExitHandler context\.'
        ):

            @dsl.pipeline
            def my_pipeline():
                with dsl.ExitHandler(exit_comp()):
                    with dsl.ParallelFor([1, 2, 3]) as v:
                        t = double(num=v)
                add(nums=dsl.Collected(t.output))

    def test_parallelfor_nested_legal_params1(self):

        @dsl.component
        def add_two_ints(num1: int, num2: int) -> int:
            return num1 + num2

        @dsl.component
        def add(nums: List[List[int]]) -> int:
            import itertools
            return sum(itertools.chain(*nums))

        @dsl.pipeline
        def my_pipeline():
            with dsl.ParallelFor([1, 2, 3]) as v1:
                with dsl.ParallelFor([1, 2, 3]) as v2:
                    t = add_two_ints(num1=v1, num2=v2)

            x = add(nums=dsl.Collected(t.output))

    def test_parallelfor_nested_legal_params2(self):

        @dsl.component
        def add_two_ints(num1: int, num2: int) -> int:
            return num1 + num2

        @dsl.component
        def add(nums: List[List[int]]) -> int:
            import itertools
            return sum(itertools.chain(*nums))

        @dsl.pipeline
        def my_pipeline():
            with dsl.ParallelFor([1, 2, 3]) as v1:
                with dsl.ParallelFor([1, 2, 3]) as v2:
                    t = add_two_ints(num1=v1, num2=v2)

                x = add(nums=dsl.Collected(t.output))

    def test_producer_and_consumer_in_same_context(self):

        @dsl.component
        def double(num: int) -> int:
            return 2 * num

        @dsl.component
        def add(nums: List[int]) -> int:
            return sum(nums)

        with self.assertRaisesRegex(
                compiler_utils.InvalidTopologyException,
                r'dsl\.Collected can only be used to fan-in outputs produced by a task within a dsl\.ParallelFor context to a task outside of the dsl\.ParallelFor context\. Producer task double is either not in a dsl\.ParallelFor context or is only in a dsl\.ParallelFor that also contains consumer task add\.'
        ):

            @dsl.pipeline
            def math_pipeline():
                with dsl.ParallelFor([1, 2, 3]) as x:
                    t = double(num=x)
                    add(nums=dsl.Collected(t.output))

    def test_no_parallelfor_context(self):

        @dsl.component
        def double(num: int) -> int:
            return 2 * num

        @dsl.component
        def add(nums: List[int]) -> int:
            return sum(nums)

        with self.assertRaisesRegex(
                compiler_utils.InvalidTopologyException,
                r'dsl\.Collected can only be used to fan-in outputs produced by a task within a dsl\.ParallelFor context to a task outside of the dsl\.ParallelFor context\. Producer task double is either not in a dsl\.ParallelFor context or is only in a dsl\.ParallelFor that also contains consumer task add\.'
        ):

            @dsl.pipeline
            def math_pipeline():
                t = double(num=1)
                add(nums=dsl.Collected(t.output))


class TestValidIgnoreUpstreamTaskSyntax(unittest.TestCase):

    def test_basic_permitted(self):

        @dsl.component
        def fail_op(message: str) -> str:
            import sys
            print(message)
            sys.exit(1)
            return message

        @dsl.component
        def print_op(message: str = 'default'):
            print(message)

        @dsl.pipeline()
        def my_pipeline(sample_input1: str = 'message'):
            task = fail_op(message=sample_input1)
            clean_up_task = print_op(
                message=task.output).ignore_upstream_failure()

        self.assertEqual(
            my_pipeline.pipeline_spec.root.dag.tasks['print-op'].trigger_policy
            .strategy, 2)
        self.assertEqual(
            my_pipeline.pipeline_spec.root.dag.tasks['fail-op'].trigger_policy
            .strategy, 0)

    def test_component_with_no_input_permitted(self):

        @dsl.component
        def fail_op(message: str) -> str:
            import sys
            print(message)
            sys.exit(1)
            return message

        @dsl.component
        def print_op():
            print('default')

        @dsl.pipeline()
        def my_pipeline(sample_input1: str = 'message'):
            task = fail_op(message=sample_input1)
            clean_up_task = print_op().ignore_upstream_failure()

        self.assertEqual(
            my_pipeline.pipeline_spec.root.dag.tasks['print-op'].trigger_policy
            .strategy, 2)
        self.assertEqual(
            my_pipeline.pipeline_spec.root.dag.tasks['fail-op'].trigger_policy
            .strategy, 0)

    def test_clean_up_on_wrapped_pipeline_permitted(self):

        @dsl.component
        def fail_op(message: str = 'message') -> str:
            import sys
            print(message)
            sys.exit(1)
            return message

        @dsl.component
        def print_op(message: str = 'message'):
            print(message)

        @dsl.pipeline
        def wrapped_pipeline(message: str = 'message') -> str:
            task = fail_op(message=message)
            return task.output

        @dsl.pipeline
        def my_pipeline(sample_input1: str = 'message'):
            task = wrapped_pipeline(message=sample_input1)
            clean_up_task = print_op(
                message=task.output).ignore_upstream_failure()

        self.assertEqual(
            my_pipeline.pipeline_spec.root.dag.tasks['print-op'].trigger_policy
            .strategy, 2)
        self.assertEqual(
            my_pipeline.pipeline_spec.root.dag.tasks['wrapped-pipeline']
            .trigger_policy.strategy, 0)

    def test_ignore_upstream_on_pipeline_task(self):

        @dsl.component
        def fail_op(message: str = 'message') -> str:
            import sys
            print(message)
            sys.exit(1)
            return message

        @dsl.component
        def identity(message: str = 'message') -> str:
            return message

        @dsl.pipeline
        def wrapped_pipeline(message: str = 'message') -> str:
            task = identity(message=message)
            return task.output

        @dsl.pipeline
        def my_pipeline(sample_input1: str = 'message'):
            task = fail_op(message=sample_input1)
            clean_up_task = wrapped_pipeline(
                message=task.output).ignore_upstream_failure()

        self.assertEqual(
            my_pipeline.pipeline_spec.root.dag.tasks['wrapped-pipeline']
            .trigger_policy.strategy, 2)
        self.assertEqual(
            my_pipeline.pipeline_spec.root.dag.tasks['fail-op'].trigger_policy
            .strategy, 0)

    def test_clean_up_task_with_no_default_value_for_upstream_input_blocked(
            self):

        @dsl.component
        def fail_op(message: str) -> str:
            import sys
            print(message)
            sys.exit(1)
            return message

        @dsl.component
        def print_op(message: str):
            print(message)

        with self.assertRaisesRegex(
                ValueError, r'Tasks can only use .ignore_upstream_failure()'):

            @dsl.pipeline()
            def my_pipeline(sample_input1: str = 'message'):
                task = fail_op(message=sample_input1)
                clean_up_task = print_op(
                    message=task.output).ignore_upstream_failure()

    def test_clean_up_task_with_no_default_value_for_pipeline_input_permitted(
            self):

        @dsl.component
        def fail_op(message: str) -> str:
            import sys
            print(message)
            sys.exit(1)
            return message

        @dsl.component
        def print_op(message: str):
            print(message)

        @dsl.pipeline()
        def my_pipeline(sample_input1: str = 'message'):
            task = fail_op(message=sample_input1)
            clean_up_task = print_op(
                message=sample_input1).ignore_upstream_failure()

        self.assertEqual(
            my_pipeline.pipeline_spec.root.dag.tasks['print-op'].trigger_policy
            .strategy, 2)
        self.assertEqual(
            my_pipeline.pipeline_spec.root.dag.tasks['fail-op'].trigger_policy
            .strategy, 0)

    def test_clean_up_task_with_no_default_value_for_constant_permitted(self):

        @dsl.component
        def fail_op(message: str) -> str:
            import sys
            print(message)
            sys.exit(1)
            return message

        @dsl.component
        def print_op(message: str):
            print(message)

        @dsl.pipeline()
        def my_pipeline(sample_input1: str = 'message'):
            task = fail_op(message=sample_input1)
            clean_up_task = print_op(message='sample').ignore_upstream_failure()

        self.assertEqual(
            my_pipeline.pipeline_spec.root.dag.tasks['print-op'].trigger_policy
            .strategy, 2)
        self.assertEqual(
            my_pipeline.pipeline_spec.root.dag.tasks['fail-op'].trigger_policy
            .strategy, 0)


class TestOutputDefinitionsPresentWhenCompilingComponents(unittest.TestCase):

    def test_basic_permitted(self):

        @dsl.component
        def comp(message: str) -> str:
            return message

        self.assertIn('Output',
                      comp.pipeline_spec.root.output_definitions.parameters)


class TestListOfArtifactsInterfaceCompileAndLoad(unittest.TestCase):

    def test_python_component(self):

        @dsl.component
        def python_component(input_list: Input[List[Artifact]]):
            pass

        self.assertEqual(
            python_component.pipeline_spec.root.input_definitions
            .artifacts['input_list'].is_artifact_list, True)
        self.assertEqual(
            python_component.pipeline_spec.components['comp-python-component']
            .input_definitions.artifacts['input_list'].is_artifact_list, True)

        with tempfile.TemporaryDirectory() as tempdir:
            output_yaml = os.path.join(tempdir, 'pipeline.yaml')
            compiler.Compiler().compile(
                pipeline_func=python_component, package_path=output_yaml)
            loaded_component = components.load_component_from_file(output_yaml)

        self.assertEqual(
            loaded_component.pipeline_spec.root.input_definitions
            .artifacts['input_list'].is_artifact_list, True)
        self.assertEqual(
            loaded_component.pipeline_spec.components['comp-python-component']
            .input_definitions.artifacts['input_list'].is_artifact_list, True)

    def test_container_component(self):

        @dsl.container_component
        def container_component(input_list: Input[List[Artifact]]):
            return dsl.ContainerSpec(
                image='alpine', command=['echo'], args=['hello world'])

        self.assertEqual(
            container_component.pipeline_spec.root.input_definitions
            .artifacts['input_list'].is_artifact_list, True)
        self.assertEqual(
            container_component.pipeline_spec
            .components['comp-container-component'].input_definitions
            .artifacts['input_list'].is_artifact_list, True)

        with tempfile.TemporaryDirectory() as tempdir:
            output_yaml = os.path.join(tempdir, 'pipeline.yaml')
            compiler.Compiler().compile(
                pipeline_func=container_component, package_path=output_yaml)
            loaded_component = components.load_component_from_file(output_yaml)

        self.assertEqual(
            loaded_component.pipeline_spec.root.input_definitions
            .artifacts['input_list'].is_artifact_list, True)
        self.assertEqual(
            loaded_component.pipeline_spec
            .components['comp-container-component'].input_definitions
            .artifacts['input_list'].is_artifact_list, True)

    def test_pipeline(self):

        @dsl.component
        def python_component(input_list: Input[List[Artifact]]):
            pass

        @dsl.pipeline
        def pipeline_component(input_list: Input[List[Artifact]]):
            python_component(input_list=input_list)

        self.assertEqual(
            pipeline_component.pipeline_spec.root.input_definitions
            .artifacts['input_list'].is_artifact_list, True)
        self.assertEqual(
            pipeline_component.pipeline_spec.components['comp-python-component']
            .input_definitions.artifacts['input_list'].is_artifact_list, True)

        with tempfile.TemporaryDirectory() as tempdir:
            output_yaml = os.path.join(tempdir, 'pipeline.yaml')
            compiler.Compiler().compile(
                pipeline_func=pipeline_component, package_path=output_yaml)
            loaded_component = components.load_component_from_file(output_yaml)

        self.assertEqual(
            loaded_component.pipeline_spec.root.input_definitions
            .artifacts['input_list'].is_artifact_list, True)
        self.assertEqual(
            loaded_component.pipeline_spec.components['comp-python-component']
            .input_definitions.artifacts['input_list'].is_artifact_list, True)


def foo_platform_set_bar_feature(task: pipeline_task.PipelineTask,
                                 val: str) -> pipeline_task.PipelineTask:
    platform_key = 'platform_foo'
    feature_key = 'bar'

    platform_struct = task.platform_config.get(platform_key, {})
    platform_struct[feature_key] = val
    task.platform_config[platform_key] = platform_struct
    return task


def foo_platform_append_bop_feature(task: pipeline_task.PipelineTask,
                                    val: str) -> pipeline_task.PipelineTask:
    platform_key = 'platform_foo'
    feature_key = 'bop'

    platform_struct = task.platform_config.get(platform_key, {})
    feature_list = platform_struct.get(feature_key, [])
    feature_list.append(val)
    platform_struct[feature_key] = feature_list
    task.platform_config[platform_key] = platform_struct
    return task


def baz_platform_set_bat_feature(task: pipeline_task.PipelineTask,
                                 val: str) -> pipeline_task.PipelineTask:
    platform_key = 'platform_baz'
    feature_key = 'bat'

    platform_struct = task.platform_config.get(platform_key, {})
    platform_struct[feature_key] = val
    task.platform_config[platform_key] = platform_struct
    return task


@dsl.component
def comp():
    pass


def compile_and_reload(
        pipeline: graph_component.GraphComponent
) -> yaml_component.YamlComponent:
    with tempfile.TemporaryDirectory() as tempdir:
        output_yaml = os.path.join(tempdir, 'pipeline.yaml')
        compiler.Compiler().compile(
            pipeline_func=pipeline, package_path=output_yaml)
        return components.load_component_from_file(output_yaml)


class TestResourceConfig(unittest.TestCase):

    def test_cpu_memory_optional(self):

        @dsl.component
        def predict_op() -> str:
            return 'a'

        @dsl.pipeline
        def simple_pipeline():
            predict_op()
            predict_op().set_cpu_limit('5')
            predict_op().set_memory_limit('50G')
            predict_op().set_cpu_request('2').set_cpu_limit(
                '5').set_memory_request('4G').set_memory_limit('50G')

        dict_format = json_format.MessageToDict(simple_pipeline.pipeline_spec)

        self.assertNotIn(
            'resources', dict_format['deploymentSpec']['executors']
            ['exec-predict-op']['container'])

        self.assertEqual(
            5, dict_format['deploymentSpec']['executors']['exec-predict-op-2']
            ['container']['resources']['cpuLimit'])
        self.assertNotIn(
            'memoryLimit', dict_format['deploymentSpec']['executors']
            ['exec-predict-op-2']['container']['resources'])

        self.assertEqual(
            50, dict_format['deploymentSpec']['executors']['exec-predict-op-3']
            ['container']['resources']['memoryLimit'])
        self.assertNotIn(
            'cpuLimit', dict_format['deploymentSpec']['executors']
            ['exec-predict-op-3']['container']['resources'])

        self.assertEqual(
            2, dict_format['deploymentSpec']['executors']['exec-predict-op-4']
            ['container']['resources']['cpuRequest'])
        self.assertEqual(
            5, dict_format['deploymentSpec']['executors']['exec-predict-op-4']
            ['container']['resources']['cpuLimit'])
        self.assertEqual(
            4, dict_format['deploymentSpec']['executors']['exec-predict-op-4']
            ['container']['resources']['memoryRequest'])
        self.assertEqual(
            50, dict_format['deploymentSpec']['executors']['exec-predict-op-4']
            ['container']['resources']['memoryLimit'])


class TestPlatformConfig(unittest.TestCase):

    def test_no_platform_config(self):

        @dsl.pipeline
        def my_pipeline():
            task = comp()

        expected = pipeline_spec_pb2.PlatformSpec()
        self.assertEqual(my_pipeline.platform_spec, expected)

        with tempfile.TemporaryDirectory() as tempdir:
            output_yaml = os.path.join(tempdir, 'pipeline.yaml')
            compiler.Compiler().compile(
                pipeline_func=my_pipeline, package_path=output_yaml)
            loaded_comp = components.load_component_from_file(output_yaml)

            with open(output_yaml) as f:
                raw_docs = list(yaml.safe_load_all(f))

        self.assertEqual(loaded_comp.platform_spec, expected)
        # also check that it doesn't write an empty second document
        self.assertEqual(len(raw_docs), 1)

    def test_one_task_one_platform(self):

        @dsl.pipeline
        def my_pipeline():
            task = comp()
            foo_platform_set_bar_feature(task, 12)

        expected = pipeline_spec_pb2.PlatformSpec()
        json_format.ParseDict(
            {
                'platforms': {
                    'platform_foo': {
                        'deployment_spec': {
                            'executors': {
                                'exec-comp': {
                                    'bar': 12
                                }
                            }
                        }
                    }
                }
            }, expected)
        self.assertEqual(my_pipeline.platform_spec, expected)

        loaded_pipeline = compile_and_reload(my_pipeline)

        self.assertEqual(loaded_pipeline.platform_spec, expected)

        # test that it can be compiled _again_ after reloading (tests YamlComponent internals)
        compile_and_reload(loaded_pipeline)

    def test_many_tasks_multiple_platforms(self):

        @dsl.pipeline
        def my_pipeline():
            task = comp()
            foo_platform_set_bar_feature(task, 12)
            foo_platform_append_bop_feature(task, 'element')
            baz_platform_set_bat_feature(task, 'hello')

        expected = pipeline_spec_pb2.PlatformSpec()
        json_format.ParseDict(
            {
                'platforms': {
                    'platform_foo': {
                        'deployment_spec': {
                            'executors': {
                                'exec-comp': {
                                    'bar': 12,
                                    'bop': ['element']
                                }
                            }
                        }
                    },
                    'platform_baz': {
                        'deployment_spec': {
                            'executors': {
                                'exec-comp': {
                                    'bat': 'hello'
                                }
                            }
                        }
                    }
                }
            }, expected)
        self.assertEqual(my_pipeline.platform_spec, expected)

        loaded_pipeline = compile_and_reload(my_pipeline)

        self.assertEqual(loaded_pipeline.platform_spec, expected)

        # test that it can be compiled _again_ after reloading (tests YamlComponent internals)
        compile_and_reload(loaded_pipeline)

    def test_many_tasks_many_platforms(self):

        @dsl.pipeline
        def my_pipeline():
            task1 = comp()
            foo_platform_set_bar_feature(task1, 12)
            foo_platform_append_bop_feature(task1, 'a')
            baz_platform_set_bat_feature(task1, 'hello')
            task2 = comp()
            foo_platform_set_bar_feature(task2, 20)
            foo_platform_append_bop_feature(task2, 'b')
            foo_platform_append_bop_feature(task2, 'c')

        expected = pipeline_spec_pb2.PlatformSpec()
        json_format.ParseDict(
            {
                'platforms': {
                    'platform_foo': {
                        'deployment_spec': {
                            'executors': {
                                'exec-comp': {
                                    'bar': 12,
                                    'bop': ['a']
                                },
                                'exec-comp-2': {
                                    'bar': 20,
                                    'bop': ['b', 'c']
                                }
                            }
                        }
                    },
                    'platform_baz': {
                        'deployment_spec': {
                            'executors': {
                                'exec-comp': {
                                    'bat': 'hello'
                                }
                            }
                        }
                    }
                }
            }, expected)

        self.assertEqual(my_pipeline.platform_spec, expected)

        loaded_pipeline = compile_and_reload(my_pipeline)

        self.assertEqual(loaded_pipeline.platform_spec, expected)

        # test that it can be compiled _again_ after reloading (tests YamlComponent internals)
        compile_and_reload(loaded_pipeline)

    def test_multiple_pipelines_with_inner_tasks(self):

        @dsl.pipeline
        def pipeline1():
            task1 = comp()
            foo_platform_set_bar_feature(task1, 12)
            foo_platform_append_bop_feature(task1, 'a')
            baz_platform_set_bat_feature(task1, 'hello')

        pipeline1_platform_spec = pipeline_spec_pb2.PlatformSpec()
        pipeline1_platform_spec.CopyFrom(pipeline1.platform_spec)

        @dsl.pipeline
        def pipeline2():
            task2 = comp()
            foo_platform_set_bar_feature(task2, 20)
            foo_platform_append_bop_feature(task2, 'b')
            foo_platform_append_bop_feature(task2, 'c')

        pipeline2_platform_spec = pipeline_spec_pb2.PlatformSpec()
        pipeline2_platform_spec.CopyFrom(pipeline2.platform_spec)

        @dsl.pipeline
        def my_pipeline():
            pipeline1()
            pipeline2()
            task3 = comp()
            foo_platform_set_bar_feature(task3, 20)
            foo_platform_append_bop_feature(task3, 'd')
            foo_platform_append_bop_feature(task3, 'e')

        expected = pipeline_spec_pb2.PlatformSpec()
        json_format.ParseDict(
            {
                'platforms': {
                    'platform_foo': {
                        'deployment_spec': {
                            'executors': {
                                'exec-comp': {
                                    'bar': 12,
                                    'bop': ['a']
                                },
                                'exec-comp-2': {
                                    'bar': 20,
                                    'bop': ['b', 'c']
                                },
                                'exec-comp-3': {
                                    'bar': 20,
                                    'bop': ['d', 'e']
                                }
                            }
                        }
                    },
                    'platform_baz': {
                        'deployment_spec': {
                            'executors': {
                                'exec-comp': {
                                    'bat': 'hello'
                                }
                            }
                        }
                    }
                }
            }, expected)

        self.assertEqual(my_pipeline.platform_spec, expected)

        loaded_pipeline = compile_and_reload(my_pipeline)
        self.assertEqual(loaded_pipeline.platform_spec, expected)

        # pipeline1 and pipeline2 platform specs should be unchanged
        self.assertEqual(pipeline1_platform_spec, pipeline1.platform_spec)
        self.assertEqual(pipeline2_platform_spec, pipeline2.platform_spec)

        # test that it can be compiled _again_ after reloading (tests YamlComponent internals)
        compile_and_reload(loaded_pipeline)

    def test_task_with_exit_handler(self):

        @dsl.pipeline
        def my_pipeline():
            exit_task = comp()
            foo_platform_set_bar_feature(exit_task, 12)
            with dsl.ExitHandler(exit_task):
                worker_task = comp()
                foo_platform_append_bop_feature(worker_task, 'a')
                baz_platform_set_bat_feature(worker_task, 'hello')

        expected = pipeline_spec_pb2.PlatformSpec()
        json_format.ParseDict(
            {
                'platforms': {
                    'platform_foo': {
                        'deployment_spec': {
                            'executors': {
                                'exec-comp': {
                                    'bar': 12,
                                },
                                'exec-comp-2': {
                                    'bop': ['a']
                                }
                            }
                        }
                    },
                    'platform_baz': {
                        'deployment_spec': {
                            'executors': {
                                'exec-comp-2': {
                                    'bat': 'hello'
                                }
                            }
                        }
                    },
                }
            }, expected)
        self.assertEqual(my_pipeline.platform_spec, expected)

        loaded_pipeline = compile_and_reload(my_pipeline)

        self.assertEqual(loaded_pipeline.platform_spec, expected)

        # test that it can be compiled _again_ after reloading (tests YamlComponent internals)
        compile_and_reload(loaded_pipeline)

    def test_pipeline_as_exit_handler(self):

        @dsl.pipeline
        def pipeline1():
            task1 = comp()
            foo_platform_set_bar_feature(task1, 12)
            foo_platform_append_bop_feature(task1, 'element')
            baz_platform_set_bat_feature(task1, 'hello')

        pipeline1_platform_spec = pipeline_spec_pb2.PlatformSpec()
        pipeline1_platform_spec.CopyFrom(pipeline1.platform_spec)

        @dsl.pipeline
        def pipeline2():
            task2 = comp()
            foo_platform_set_bar_feature(task2, 20)
            foo_platform_append_bop_feature(task2, 'element1')
            foo_platform_append_bop_feature(task2, 'element2')

        pipeline2_platform_spec = pipeline_spec_pb2.PlatformSpec()
        pipeline2_platform_spec.CopyFrom(pipeline2.platform_spec)

        @dsl.pipeline
        def my_pipeline():
            exit_task = pipeline1()
            with dsl.ExitHandler(exit_task=exit_task):
                pipeline2()
                task3 = comp()
                foo_platform_set_bar_feature(task3, 20)
                foo_platform_append_bop_feature(task3, 'element3')
                foo_platform_append_bop_feature(task3, 'element4')

        expected = pipeline_spec_pb2.PlatformSpec()
        json_format.ParseDict(
            {
                'platforms': {
                    'platform_foo': {
                        'deployment_spec': {
                            'executors': {
                                'exec-comp': {
                                    'bar': 20,
                                    'bop': ['element1', 'element2']
                                },
                                'exec-comp-2': {
                                    'bar': 20,
                                    'bop': ['element3', 'element4']
                                },
                                'exec-comp-3': {
                                    'bar': 12,
                                    'bop': ['element']
                                },
                            }
                        }
                    },
                    'platform_baz': {
                        'deployment_spec': {
                            'executors': {
                                'exec-comp-3': {
                                    'bat': 'hello'
                                }
                            }
                        }
                    }
                }
            }, expected)
        self.assertEqual(my_pipeline.platform_spec, expected)

        loaded_pipeline = compile_and_reload(my_pipeline)
        self.assertEqual(loaded_pipeline.platform_spec, expected)

        # pipeline1 and pipeline2 platform specs should be unchanged
        self.assertEqual(pipeline1_platform_spec, pipeline1.platform_spec)
        self.assertEqual(pipeline2_platform_spec, pipeline2.platform_spec)

        # test that it can be compiled _again_ after reloading (tests YamlComponent internals)
        compile_and_reload(loaded_pipeline)

    def test_cannot_compile_to_json(self):

        @dsl.pipeline
        def my_pipeline():
            task = comp()
            foo_platform_set_bar_feature(task, 12)

        with self.assertRaisesRegex(
                ValueError,
                r"Platform-specific features are only supported when serializing to YAML\. Argument for 'package_path' has file extension '\.json'\."
        ):
            with tempfile.TemporaryDirectory() as tempdir:
                output_yaml = os.path.join(tempdir, 'pipeline.json')
                compiler.Compiler().compile(
                    pipeline_func=my_pipeline, package_path=output_yaml)

    def test_cannot_set_platform_specific_config_on_in_memory_pipeline_task(
            self):

        @dsl.pipeline
        def inner():
            task = comp()

        with self.assertRaisesRegex(
                ValueError,
                r'Platform\-specific features can only be set on primitive components\. Found platform\-specific feature set on a pipeline\.'
        ):

            @dsl.pipeline
            def outer():
                task = inner()
                foo_platform_set_bar_feature(task, 12)


class ExtractInputOutputDescription(unittest.TestCase):

    def test_no_descriptions(self):
        from kfp import dsl

        @dsl.component
        def comp(
            string: str,
            in_artifact: Input[Artifact],
            out_artifact: Output[Artifact],
        ) -> str:
            return string

        Outputs = NamedTuple(
            'Outputs',
            out_str=str,
            out_artifact=Artifact,
        )

        @dsl.pipeline
        def my_pipeline(
            string: str,
            in_artifact: Input[Artifact],
        ) -> Outputs:
            t = comp(
                string=string,
                in_artifact=in_artifact,
            )
            return Outputs(
                out_str=t.outputs['Output'],
                out_artifact=t.outputs['out_artifact'])

        pipeline_spec = my_pipeline.pipeline_spec

        # test pipeline
        # check key with assertIn first to prevent false negatives with errored key and easier debugging
        self.assertIn('string', pipeline_spec.root.input_definitions.parameters)
        self.assertEqual(
            pipeline_spec.root.input_definitions.parameters['string']
            .description, '')
        self.assertIn('in_artifact',
                      pipeline_spec.root.input_definitions.artifacts)
        self.assertEqual(
            pipeline_spec.root.input_definitions.artifacts['in_artifact']
            .description, '')
        self.assertIn('out_str',
                      pipeline_spec.root.output_definitions.parameters)
        self.assertEqual(
            pipeline_spec.root.output_definitions.parameters['out_str']
            .description, '')
        self.assertIn('out_artifact',
                      pipeline_spec.root.output_definitions.artifacts)
        self.assertEqual(
            pipeline_spec.root.output_definitions.artifacts['out_artifact']
            .description, '')

        # test component
        # check key with assertIn first to prevent false negatives with errored key and easier debugging
        self.assertIn(
            'string',
            pipeline_spec.components['comp-comp'].input_definitions.parameters)
        self.assertEqual(
            pipeline_spec.components['comp-comp'].input_definitions
            .parameters['string'].description, '')
        self.assertIn(
            'in_artifact',
            pipeline_spec.components['comp-comp'].input_definitions.artifacts)
        self.assertEqual(
            pipeline_spec.components['comp-comp'].input_definitions
            .artifacts['in_artifact'].description, '')
        self.assertIn(
            'Output',
            pipeline_spec.components['comp-comp'].output_definitions.parameters)
        self.assertEqual(
            pipeline_spec.components['comp-comp'].output_definitions
            .parameters['Output'].description, '')
        self.assertIn(
            'out_artifact',
            pipeline_spec.components['comp-comp'].output_definitions.artifacts)
        self.assertEqual(
            pipeline_spec.components['comp-comp'].output_definitions
            .artifacts['out_artifact'].description, '')

    def test_google_style(self):

        @dsl.component
        def comp(
            string: str,
            in_artifact: Input[Artifact],
            out_artifact: Output[Artifact],
        ) -> str:
            """Component description.

            Args:
                string: Component input string.
                in_artifact: Component input artifact.

            Returns:
                Output: Component output string.
                out_artifact: Component output artifact.
            """
            return string

        Outputs = NamedTuple(
            'Outputs',
            out_str=str,
            out_artifact=Artifact,
        )

        @dsl.pipeline
        def my_pipeline(
            string: str,
            in_artifact: Input[Artifact],
        ) -> Outputs:
            """Pipeline description.

            Args:
                string: Pipeline input string.
                in_artifact: Pipeline input artifact.

            Returns:
                out_str: Pipeline output string.
                out_artifact: Pipeline output artifact.
            """
            t = comp(
                string=string,
                in_artifact=in_artifact,
            )
            return Outputs(
                out_str=t.outputs['Output'],
                out_artifact=t.outputs['out_artifact'])

        pipeline_spec = my_pipeline.pipeline_spec

        # test pipeline
        # check key with assertIn first to prevent false negatives with errored key and easier debugging
        self.assertIn('string', pipeline_spec.root.input_definitions.parameters)
        self.assertEqual(
            pipeline_spec.root.input_definitions.parameters['string']
            .description, 'Pipeline input string.')
        self.assertIn('in_artifact',
                      pipeline_spec.root.input_definitions.artifacts)
        self.assertEqual(
            pipeline_spec.root.input_definitions.artifacts['in_artifact']
            .description, 'Pipeline input artifact.')
        self.assertIn('out_str',
                      pipeline_spec.root.output_definitions.parameters)
        self.assertEqual(
            pipeline_spec.root.output_definitions.parameters['out_str']
            .description, 'Pipeline output string.')
        self.assertIn('out_artifact',
                      pipeline_spec.root.output_definitions.artifacts)
        self.assertEqual(
            pipeline_spec.root.output_definitions.artifacts['out_artifact']
            .description, 'Pipeline output artifact.')

        # test component
        # check key with assertIn first to prevent false negatives with errored key and easier debugging
        self.assertIn(
            'string',
            pipeline_spec.components['comp-comp'].input_definitions.parameters)
        self.assertEqual(
            pipeline_spec.components['comp-comp'].input_definitions
            .parameters['string'].description, 'Component input string.')
        self.assertIn(
            'in_artifact',
            pipeline_spec.components['comp-comp'].input_definitions.artifacts)
        self.assertEqual(
            pipeline_spec.components['comp-comp'].input_definitions
            .artifacts['in_artifact'].description, 'Component input artifact.')
        self.assertIn(
            'Output',
            pipeline_spec.components['comp-comp'].output_definitions.parameters)
        self.assertEqual(
            pipeline_spec.components['comp-comp'].output_definitions
            .parameters['Output'].description, 'Component output string.')
        self.assertIn(
            'out_artifact',
            pipeline_spec.components['comp-comp'].output_definitions.artifacts)
        self.assertEqual(
            pipeline_spec.components['comp-comp'].output_definitions
            .artifacts['out_artifact'].description,
            'Component output artifact.')

    def test_inner_return_keywords_does_not_mess_up_extraction(self):
        # we do string replacement for Return and Returns, so need to ensure having those words elsewhere plays well with extraction
        @dsl.component
        def comp(
            string: str,
            in_artifact: Input[Artifact],
            out_artifact: Output[Artifact],
        ) -> str:
            """Return Component Returns description.

            Args:
                string: Component Return input string.
                in_artifact: Component Returns input artifact.

            Returns:
                Output: Component output string.
                out_artifact: Component output artifact.
            """
            return string

        Outputs = NamedTuple(
            'Outputs',
            out_str=str,
            out_artifact=Artifact,
        )

        @dsl.pipeline
        def my_pipeline(
            string: str,
            in_artifact: Input[Artifact],
        ) -> Outputs:
            """Pipeline description. Returns

            Args:
                string: Return Pipeline input string. Returns
                in_artifact: Pipeline input Return artifact.

            Returns:
                out_str: Pipeline output string.
                out_artifact: Pipeline output artifact.
            """
            t = comp(
                string=string,
                in_artifact=in_artifact,
            )
            return Outputs(
                out_str=t.outputs['Output'],
                out_artifact=t.outputs['out_artifact'])

        pipeline_spec = my_pipeline.pipeline_spec

        # test pipeline
        # check key with assertIn first to prevent false negatives with errored key and easier debugging
        self.assertIn('string', pipeline_spec.root.input_definitions.parameters)
        self.assertEqual(
            pipeline_spec.root.input_definitions.parameters['string']
            .description, 'Return Pipeline input string. Returns')
        self.assertIn('in_artifact',
                      pipeline_spec.root.input_definitions.artifacts)
        self.assertEqual(
            pipeline_spec.root.input_definitions.artifacts['in_artifact']
            .description, 'Pipeline input Return artifact.')
        self.assertIn('out_str',
                      pipeline_spec.root.output_definitions.parameters)
        self.assertEqual(
            pipeline_spec.root.output_definitions.parameters['out_str']
            .description, 'Pipeline output string.')
        self.assertIn('out_artifact',
                      pipeline_spec.root.output_definitions.artifacts)
        self.assertEqual(
            pipeline_spec.root.output_definitions.artifacts['out_artifact']
            .description, 'Pipeline output artifact.')

        # test component
        # check key with assertIn first to prevent false negatives with errored key and easier debugging
        self.assertIn(
            'string',
            pipeline_spec.components['comp-comp'].input_definitions.parameters)
        self.assertEqual(
            pipeline_spec.components['comp-comp'].input_definitions
            .parameters['string'].description, 'Component Return input string.')
        self.assertIn(
            'in_artifact',
            pipeline_spec.components['comp-comp'].input_definitions.artifacts)
        self.assertEqual(
            pipeline_spec.components['comp-comp'].input_definitions
            .artifacts['in_artifact'].description,
            'Component Returns input artifact.')
        self.assertIn(
            'Output',
            pipeline_spec.components['comp-comp'].output_definitions.parameters)
        self.assertEqual(
            pipeline_spec.components['comp-comp'].output_definitions
            .parameters['Output'].description, 'Component output string.')
        self.assertIn(
            'out_artifact',
            pipeline_spec.components['comp-comp'].output_definitions.artifacts)
        self.assertEqual(
            pipeline_spec.components['comp-comp'].output_definitions
            .artifacts['out_artifact'].description,
            'Component output artifact.')


if __name__ == '__main__':
    unittest.main()
