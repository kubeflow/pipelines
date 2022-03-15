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
import sys
import tempfile
import unittest

from absl.testing import parameterized
from kfp import compiler
from kfp import components
from kfp import dsl
from kfp.components.types import type_utils
from kfp.dsl import PipelineTaskFinalStatus

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


class CompilerTest(parameterized.TestCase):

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

            @dsl.pipeline(name='test-pipeline')
            def simple_pipeline(pipeline_input: str = 'Hello KFP!'):
                producer = producer_op(input_param=pipeline_input)
                consumer = consumer_op(
                    input_model=producer.outputs['output_model'],
                    input_value=producer.outputs['output_value'])

            target_json_file = os.path.join(tmpdir, 'result.json')
            compiler.Compiler().compile(
                pipeline_func=simple_pipeline, package_path=target_json_file)

            self.assertTrue(os.path.exists(target_json_file))
            with open(target_json_file, 'r') as f:
                print(f.read())
        finally:
            shutil.rmtree(tmpdir)

    def test_compile_pipeline_with_bool(self):

        tmpdir = tempfile.mkdtemp()
        try:
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

            target_json_file = os.path.join(tmpdir, 'result.json')
            compiler.Compiler().compile(
                pipeline_func=simple_pipeline, package_path=target_json_file)

            self.assertTrue(os.path.exists(target_json_file))
            with open(target_json_file, 'r') as f:
                print(f.read())
        finally:
            shutil.rmtree(tmpdir)

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
                pipeline_spec = json.load(f)
            self.assertEqual('gs://path', pipeline_spec['defaultPipelineRoot'])
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
                type_utils.InconsistentTypeException,
                'Incompatible argument passed to the input "some_input" of '
                'component "compoent": Argument type "STRING" is incompatible '
                'with the input type "Artifact"'):
            compiler.Compiler().compile(
                pipeline_func=my_pipeline, package_path='output.json')

    def test_passing_missing_type_annotation_on_pipeline_input_should_error(
            self):

        @dsl.pipeline(name='test-pipeline', pipeline_root='gs://path')
        def my_pipeline(input1):
            pass

        with self.assertRaisesRegex(
                TypeError, 'Missing type annotation for argument: input1'):
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
        def consumer_op(input1: dsl.Input[dsl.Dataset]):
            pass

        @dsl.pipeline(name='test-pipeline')
        def my_pipeline():
            consumer_op(input1=producer_op1().output)
            consumer_op(input1=producer_op2().output)

        with self.assertRaisesRegex(
                type_utils.InconsistentTypeException,
                'Incompatible argument passed to the input "input1" of component'
                ' "Consumer op": Argument type "SomeArbitraryType" is'
                ' incompatible with the input type "Dataset"'):
            compiler.Compiler().compile(
                pipeline_func=my_pipeline, package_path='result.json')

    @parameterized.parameters(
        {
            'pipeline_name': 'my-pipeline',
            'is_valid': True,
        },
        {
            'pipeline_name': 'p' * 128,
            'is_valid': True,
        },
        {
            'pipeline_name': 'p' * 129,
            'is_valid': False,
        },
        {
            'pipeline_name': 'my_pipeline',
            'is_valid': False,
        },
        {
            'pipeline_name': '-my-pipeline',
            'is_valid': False,
        },
        {
            'pipeline_name': 'My pipeline',
            'is_valid': False,
        },
    )
    def test_validate_pipeline_name(self, pipeline_name, is_valid):

        if is_valid:
            compiler.Compiler()._validate_pipeline_name(pipeline_name)
        else:
            with self.assertRaisesRegex(ValueError, 'Invalid pipeline name: '):
                compiler.Compiler()._validate_pipeline_name('my_pipeline')

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
                pipeline_func=my_pipeline, package_path='result.json')

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
                pipeline_func=my_pipeline, package_path='result.json')

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
        def my_pipeline(text: bool):
            print_op()

        with self.assertRaisesRegex(
                ValueError,
                'PipelineTaskFinalStatus can only be used in an exit task.'):
            compiler.Compiler().compile(
                pipeline_func=my_pipeline, package_path='result.json')


# pylint: disable=import-outside-toplevel,unused-import,import-error,redefined-outer-name,reimported
class V2NamespaceAliasTest(unittest.TestCase):
    """Test that imports of both modules and objects are aliased
    (e.g. all import path variants work).
    """

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
            temp_filepath = os.path.join(tempdir, 'hello_world_pipeline.json')
            v2.compiler.Compiler().compile(
                pipeline_func=pipeline_hello_world, package_path=temp_filepath)

            with open(temp_filepath, "r") as f:
                json.load(f)

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
            temp_filepath = os.path.join(tempdir, 'hello_world_pipeline.json')
            compiler.Compiler().compile(
                pipeline_func=pipeline_hello_world, package_path=temp_filepath)

            with open(temp_filepath, "r") as f:
                json.load(f)

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
            temp_filepath = os.path.join(tempdir, 'hello_world_pipeline.json')
            Compiler().compile(
                pipeline_func=pipeline_hello_world, package_path=temp_filepath)

            with open(temp_filepath, "r") as f:
                json.load(f)


if __name__ == '__main__':
    unittest.main()
