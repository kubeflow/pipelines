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
import shutil
import tempfile
import unittest

from typing import List

from kfp import components
from kfp.dsl import types
from kfp.v2 import compiler
from kfp.v2 import dsl
from kfp.v2.compiler import compiler_utils
from kfp.v2.dsl import PipelineTaskFinalStatus
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


class CompilerTest(unittest.TestCase):

    def test_compile_simple_pipeline(self):

        tmpdir = tempfile.mkdtemp()
        try:
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

            @dsl.pipeline(name='test-pipeline', pipeline_root='dummy_root')
            def simple_pipeline(pipeline_input: str = 'Hello KFP!'):
                producer = producer_op(input_param=pipeline_input)
                consumer = consumer_op(
                    input_model=producer.outputs['output_model'],
                    input_value=producer.outputs['output_value'])

            target_json_file = os.path.join(tmpdir, 'result.json')
            compiler.Compiler().compile(
                pipeline_func=simple_pipeline, package_path=target_json_file)

            self.assertTrue(os.path.exists(target_json_file))
        finally:
            shutil.rmtree(tmpdir)

    def test_compile_pipeline_with_dsl_graph_component_should_raise_error(self):

        with self.assertRaisesRegex(
                NotImplementedError,
                'dsl.graph_component is not yet supported in KFP v2 compiler.'):

            @dsl.component
            def flip_coin_op() -> str:
                import random
                result = 'heads' if random.randint(0, 1) == 0 else 'tails'
                return result

            @dsl.graph_component
            def flip_coin_graph_component():
                flip = flip_coin_op()
                with dsl.Condition(flip.output == 'heads'):
                    flip_coin_graph_component()

            @dsl.pipeline(name='test-pipeline', pipeline_root='dummy_root')
            def my_pipeline():
                flip_coin_graph_component()

            compiler.Compiler().compile(
                pipeline_func=my_pipeline, package_path='output.json')

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
                pipeline_func=my_pipeline, package_path='output.json')

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
                pipeline_func=my_pipeline, package_path='output.json')

    def test_compile_pipeline_with_missing_task_should_raise_error(self):

        @dsl.pipeline(name='test-pipeline', pipeline_root='dummy_root')
        def my_pipeline(text: str):
            pass

        with self.assertRaisesRegex(ValueError,
                                    'Task is missing from pipeline.'):
            compiler.Compiler().compile(
                pipeline_func=my_pipeline, package_path='output.json')

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
                pipeline_func=my_pipeline, package_path='output.json')

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
                pipeline_func=my_pipeline, package_path='output.json')

    def test_compile_pipeline_with_invalid_name_should_raise_error(self):

        def my_pipeline():
            VALID_PRODUCER_COMPONENT_SAMPLE(input_param='input')

        with self.assertRaisesRegex(
                ValueError,
                'Invalid pipeline name: .*\nPlease specify a pipeline name that matches'
        ):
            compiler.Compiler().compile(
                pipeline_func=my_pipeline, package_path='output.json')

    def test_set_pipeline_root_through_pipeline_decorator(self):

        tmpdir = tempfile.mkdtemp()
        try:

            @dsl.pipeline(name='test-pipeline', pipeline_root='gs://path')
            def my_pipeline():
                VALID_PRODUCER_COMPONENT_SAMPLE(input_param='input')

            target_json_file = os.path.join(tmpdir, 'result.json')
            compiler.Compiler().compile(
                pipeline_func=my_pipeline, package_path=target_json_file)

            self.assertTrue(os.path.exists(target_json_file))
            with open(target_json_file) as f:
                job_spec = json.load(f)
            self.assertEqual('gs://path',
                             job_spec['runtimeConfig']['gcsOutputDirectory'])
        finally:
            shutil.rmtree(tmpdir)

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
                TypeError,
                'Passing PipelineParam "input1" with type "String" \(as "Parameter"\) '
                'to component input "some_input" with type "None" \(as "Artifact"\) is '
                'incompatible. Please fix the type of the component input.'):
            compiler.Compiler().compile(
                pipeline_func=my_pipeline, package_path='output.json')

    def test_passing_missing_type_annotation_on_pipeline_input_should_error(
            self):

        @dsl.pipeline(name='test-pipeline', pipeline_root='gs://path')
        def my_pipeline(input1):
            pass

        with self.assertRaisesRegex(
                TypeError,
                'The pipeline argument \"input1\" is viewed as an artifact due '
                'to its type \"None\". And we currently do not support passing '
                'artifacts as pipeline inputs. Consider type annotating the '
                'argument with a primitive type, such as \"str\", \"int\", '
                '\"float\", \"bool\", \"dict\", and \"list\".'):
            compiler.Compiler().compile(
                pipeline_func=my_pipeline, package_path='output.json')

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
      - {name: input, type: MyDataset}
      implementation:
        container:
          image: dummy
          args:
          - {inputPath: input}
      """)

        @dsl.component
        def consumer_op2(input: dsl.Input[dsl.Dataset]):
            pass

        @dsl.pipeline(name='test-pipeline')
        def my_pipeline():
            consumer_op1(producer_op1().output)
            consumer_op1(producer_op2().output)
            consumer_op2(producer_op1().output)
            consumer_op2(producer_op2().output)

        try:
            tmpdir = tempfile.mkdtemp()
            target_json_file = os.path.join(tmpdir, 'result.json')
            compiler.Compiler().compile(
                pipeline_func=my_pipeline, package_path=target_json_file)

            self.assertTrue(os.path.exists(target_json_file))
        finally:
            shutil.rmtree(tmpdir)

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
      - {name: input, type: Artifact}
      implementation:
        container:
          image: dummy
          args:
          - {inputPath: input}
      """)

        @dsl.component
        def consumer_op2(input: dsl.Input[dsl.Artifact]):
            pass

        @dsl.pipeline(name='test-pipeline')
        def my_pipeline():
            consumer_op1(producer_op1().output)
            consumer_op1(producer_op2().output)
            consumer_op2(producer_op1().output)
            consumer_op2(producer_op2().output)

        try:
            tmpdir = tempfile.mkdtemp()
            target_json_file = os.path.join(tmpdir, 'result.json')
            compiler.Compiler().compile(
                pipeline_func=my_pipeline, package_path=target_json_file)

            self.assertTrue(os.path.exists(target_json_file))
        finally:
            shutil.rmtree(tmpdir)

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
        def consumer_op(input: dsl.Input[dsl.Dataset]):
            pass

        @dsl.pipeline(name='test-pipeline')
        def my_pipeline():
            consumer_op(producer_op1().output)
            consumer_op(producer_op2().output)

        with self.assertRaisesRegex(
                types.InconsistentTypeException,
                'Incompatible argument passed to the input "input" of component '
                '"Consumer op": Argument type "SomeArbitraryType" is incompatible with '
                'the input type "Dataset"'):
            compiler.Compiler().compile(
                pipeline_func=my_pipeline, package_path='result.json')

    def test_constructing_container_op_directly_should_error(self):

        @dsl.pipeline(name='test-pipeline')
        def my_pipeline():
            dsl.ContainerOp(
                name='comp1',
                image='gcr.io/dummy',
                command=['python', 'main.py'])

        with self.assertRaisesRegex(
                RuntimeError,
                'Constructing ContainerOp instances directly is deprecated and not '
                'supported when compiling to v2 \(using v2 compiler or v1 compiler '
                'with V2_COMPATIBLE or V2_ENGINE mode\).'):
            compiler.Compiler().compile(
                pipeline_func=my_pipeline, package_path='result.json')

    def test_use_task_final_status_in_non_exit_op(self):

        @dsl.component
        def print_op(status: PipelineTaskFinalStatus):
            return status

        @dsl.pipeline(name='test-pipeline')
        def my_pipeline():
            print_op()

        with self.assertRaisesRegex(
                ValueError,
                'PipelineTaskFinalStatus can only be used in an exit task.'):
            compiler.Compiler().compile(
                pipeline_func=my_pipeline, package_path='result.json')

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
        def my_pipeline():
            print_op()

        with self.assertRaisesRegex(
                ValueError,
                'PipelineTaskFinalStatus can only be used in an exit task.'):
            compiler.Compiler().compile(
                pipeline_func=my_pipeline, package_path='result.json')

    def test_set_retry(self):

        @dsl.component
        def add(a: float, b: float) -> float:
            return a + b

        @dsl.pipeline(name='test-pipeline')
        def my_pipeline(a: float = 1, b: float = 7):
            add_task = add(a, b)
            add_task.set_retry(
                num_retries=3,
                backoff_duration='30s',
                backoff_factor=1.0,
                backoff_max_duration='3h',
            )

        with tempfile.TemporaryDirectory() as tempdir:
            package_path = os.path.join(tempdir, 'pipeline.json')
            compiler.Compiler().compile(
                pipeline_func=my_pipeline, package_path=package_path)
            with open(package_path, 'r') as f:
                pipeline_spec_dict = yaml.safe_load(f)

        retry_policy = pipeline_spec_dict['pipelineSpec']['root']['dag'][
            'tasks']['add']['retryPolicy']
        self.assertEqual(retry_policy['maxRetryCount'], 3.0)
        self.assertEqual(retry_policy['backoffDuration'], '30s')
        self.assertEqual(retry_policy['backoffFactor'], 1.0)

        # tests backoff max duration cap of 1 hour
        self.assertEqual(retry_policy['backoffMaxDuration'], '3600s')

    def test_retry_policy_throws_warning(self):

        @dsl.component
        def add(a: float, b: float) -> float:
            return a + b

        @dsl.pipeline(name='test-pipeline')
        def my_pipeline(a: float = 1, b: float = 7):
            add_task = add(a, b)
            add_task.set_retry(num_retries=3, policy='Always')

        with tempfile.TemporaryDirectory() as tempdir, self.assertWarnsRegex(
                compiler_utils.NoOpWarning,
                "'retry_policy' is ignored when compiling to IR using the v2 compiler"
        ):
            package_path = os.path.join(tempdir, 'pipeline.json')
            compiler.Compiler().compile(
                pipeline_func=my_pipeline, package_path=package_path)

    def test_parallel_for_parallelism(self):

        @dsl.component
        def print_op(msg: str):
            print(msg)

        @dsl.pipeline(name='test-pipeline')
        def my_pipeline(loop_params: List[str] = ['hello', 'world']):
            with dsl.ParallelFor(loop_args=loop_params, parallelism=5) as item:
                print_op(item)

        with tempfile.TemporaryDirectory() as tempdir:
            package_path = os.path.join(tempdir, 'pipeline.json')
            compiler.Compiler().compile(
                pipeline_func=my_pipeline, package_path=package_path)
            with open(package_path, 'r') as f:
                pipeline_spec_dict = yaml.safe_load(f)

        iterator_policy = pipeline_spec_dict['pipelineSpec']['root']['dag'][
            'tasks']['for-loop-1']['iteratorPolicy']
        self.assertEqual(iterator_policy['parallelismLimit'], 5)

    def test_parallel_for_invalid_parallelism(self):

        @dsl.component
        def print_op(msg: str):
            print(msg)

        @dsl.pipeline(name='test-pipeline')
        def my_pipeline(loop_params: List[str] = ['hello', 'world']):
            with dsl.ParallelFor(loop_args=loop_params, parallelism=-5) as item:
                print_op(item)

        with tempfile.TemporaryDirectory() as tempdir:
            package_path = os.path.join(tempdir, 'pipeline.json')
            with self.assertRaisesRegex(
                    ValueError, 'ParallelFor parallelism must be >= 0.'):
                compiler.Compiler().compile(
                    pipeline_func=my_pipeline, package_path=package_path)

    def test_parallel_for_no_parallelism(self):

        @dsl.component
        def print_op(msg: str):
            print(msg)

        @dsl.pipeline(name='test-pipeline')
        def my_pipeline(loop_params: List[str] = ['hello', 'world']):
            with dsl.ParallelFor(loop_args=loop_params) as item:
                print_op(item)

            with dsl.ParallelFor(loop_args=loop_params, parallelism=0) as item:
                print_op(item)

        with tempfile.TemporaryDirectory() as tempdir:
            package_path = os.path.join(tempdir, 'pipeline.json')
            compiler.Compiler().compile(
                pipeline_func=my_pipeline, package_path=package_path)
            with open(package_path, 'r') as f:
                pipeline_spec_dict = yaml.safe_load(f)

        for_loop_1 = pipeline_spec_dict['pipelineSpec']['root']['dag']['tasks'][
            'for-loop-1']
        with self.assertRaises(KeyError):
            for_loop_1['iteratorPolicy']

        for_loop_2 = pipeline_spec_dict['pipelineSpec']['root']['dag']['tasks'][
            'for-loop-2']
        with self.assertRaises(KeyError):
            for_loop_2['iteratorPolicy']


if __name__ == '__main__':
    unittest.main()
