# Copyright 2020 Google LLC
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
import shutil
import tempfile
import unittest

from kfp.v2 import compiler
from kfp.v2 import components
from kfp.v2 import dsl


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

      @dsl.pipeline(name='two-step-pipeline')
      def simple_pipeline(pipeline_input='Hello KFP!'):
        producer = producer_op(input_param=pipeline_input)
        consumer = consumer_op(
            input_model=producer.outputs['output_model'],
            input_value=producer.outputs['output_value'])

      target_json_file = os.path.join(tmpdir, 'result.json')
      compiler.Compiler().compile(
          pipeline_func=simple_pipeline,
          pipeline_root='dummy_root',
          output_path=target_json_file)

      self.assertTrue(os.path.exists(target_json_file))
    finally:
      shutil.rmtree(tmpdir)

  def test_compile_pipeline_with_dsl_condition_should_raise_error(self):

    flip_coin_op = components.load_component_from_text("""
      name: flip coin
      inputs:
      - {name: name, type: String}
      outputs:
      - {name: result, type: String}
      implementation:
        container:
          image: gcr.io/my-project/my-image:tag
          args:
          - {inputValue: name}
          - {outputPath: result}
      """)

    print_op = components.load_component_from_text("""
      name: print
      inputs:
      - {name: name, type: String}
      - {name: msg, type: String}
      implementation:
        container:
          image: gcr.io/my-project/my-image:tag
          args:
          - {inputValue: name}
          - {inputValue: msg}
      """)

    @dsl.pipeline()
    def flipcoin():
      flip = flip_coin_op('flip')

      with dsl.Condition(flip.outputs['result'] == 'heads'):
        flip2 = flip_coin_op('flip-again')

        with dsl.Condition(flip2.outputs['result'] == 'tails'):
          print_op('print1', flip2.outputs['result'])

      with dsl.Condition(flip.outputs['result'] == 'tails'):
        print_op('print2', flip2.outputs['results'])

    with self.assertRaisesRegex(
        NotImplementedError,
        'dsl.Condition is not yet supported in KFP v2 compiler.'):
      compiler.Compiler().compile(
          pipeline_func=flipcoin,
          pipeline_root='dummy_root',
          output_path='output.json')

  def test_compile_pipeline_with_dsl_exithandler_should_raise_error(self):

    gcs_download_op = components.load_component_from_text("""
      name: GCS - Download
      inputs:
      - {name: url, type: String}
      outputs:
      - {name: result, type: String}
      implementation:
        container:
          image: gcr.io/my-project/my-image:tag
          args:
          - {inputValue: url}
          - {outputPath: result}
      """)

    echo_op = components.load_component_from_text("""
      name: echo
      inputs:
      - {name: msg, type: String}
      implementation:
        container:
          image: gcr.io/my-project/my-image:tag
          args:
          - {inputValue: msg}
      """)

    @dsl.pipeline()
    def download_and_print(url='gs://ml-pipeline/shakespeare/shakespeare1.txt'):
      """A sample pipeline showing exit handler."""

      exit_task = echo_op('exit!')

      with dsl.ExitHandler(exit_task):
        download_task = gcs_download_op(url)
        echo_task = echo_op(download_task.outputs['result'])

    with self.assertRaisesRegex(
        NotImplementedError,
        'dsl.ExitHandler is not yet supported in KFP v2 compiler.'):
      compiler.Compiler().compile(
          pipeline_func=download_and_print,
          pipeline_root='dummy_root',
          output_path='output.json')

  def test_compile_pipeline_with_dsl_parallelfor_should_raise_error(self):

    @components.create_component_from_func
    def print_op(s: str):
      print(s)

    @dsl.pipeline()
    def my_pipeline():
      loop_args = [{'A_a': 1, 'B_b': 2}, {'A_a': 10, 'B_b': 20}]
      with dsl.ParallelFor(loop_args, parallelism=10) as item:
        print_op(item)
        print_op(item.A_a)
        print_op(item.B_b)

    with self.assertRaisesRegex(
        NotImplementedError,
        'dsl.ParallelFor is not yet supported in KFP v2 compiler.'):
      compiler.Compiler().compile(
          pipeline_func=my_pipeline,
          pipeline_root='dummy_root',
          output_path='output.json')

  def test_compile_pipeline_with_dsl_graph_component_should_raise_error(self):

    with self.assertRaisesRegex(
        NotImplementedError,
        'dsl.graph_component is not yet supported in KFP v2 compiler.'):

      @dsl.graph_component
      def echo1_graph_component(text1):
        dsl.ContainerOp(
            name='echo1-task1',
            image='library/bash:4.4.23',
            command=['sh', '-c'],
            arguments=['echo "$0"', text1])

      @dsl.graph_component
      def echo2_graph_component(text2):
        dsl.ContainerOp(
            name='echo2-task1',
            image='library/bash:4.4.23',
            command=['sh', '-c'],
            arguments=['echo "$0"', text2])

      @dsl.pipeline()
      def opsgroups_pipeline(text1='message 1', text2='message 2'):
        step1_graph_component = echo1_graph_component(text1)
        step2_graph_component = echo2_graph_component(text2)
        step2_graph_component.after(step1_graph_component)

      compiler.Compiler().compile(
          pipeline_func=opsgroups_pipeline,
          pipeline_root='dummy_root',
          output_path='output.json')

  def test_compile_pipeline_with_misused_inputvalue_should_raise_error(self):

    component_op = components.load_component_from_text("""
        name: compoent with misused placeholder
        inputs:
        - {name: model, type: Model}
        implementation:
          container:
            image: dummy
            args:
            - {inputValue: model}
        """)

    def my_pipeline(model):
      component_op(model=model)

    with self.assertRaisesRegex(
        TypeError,
        ' type "Model" cannot be paired with InputValuePlaceholder.'):
      compiler.Compiler().compile(
          pipeline_func=my_pipeline,
          pipeline_root='dummy',
          output_path='output.json')

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

    def my_pipeline(text):
      component_op(text=text)

    with self.assertRaisesRegex(
        TypeError,
        ' type "String" cannot be paired with InputPathPlaceholder.'):
      compiler.Compiler().compile(
          pipeline_func=my_pipeline,
          pipeline_root='dummy',
          output_path='output.json')

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

    def my_pipeline(value):
      component_op(value=value)

    with self.assertRaisesRegex(
        TypeError,
        ' type "Float" cannot be paired with InputUriPlaceholder.'):
      compiler.Compiler().compile(
          pipeline_func=my_pipeline,
          pipeline_root='dummy',
          output_path='output.json')

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

    def my_pipeline():
      component_op()

    with self.assertRaisesRegex(
        TypeError,
        ' type "Integer" cannot be paired with OutputUriPlaceholder.'):
      compiler.Compiler().compile(
          pipeline_func=my_pipeline,
          pipeline_root='dummy',
          output_path='output.json')

  def test_compile_pipeline_with_outputpath_should_warn(self):

    with self.assertWarnsRegex(
        UserWarning, 'Local file paths are currently unsupported for I/O.'):
      component_op = components.load_component_from_text("""
          name: compoent use outputPath
          outputs:
          - {name: metrics, type: Metrics}
          implementation:
            container:
              image: dummy
              args:
              - {outputPath: metrics}
          """)

  def test_compile_pipeline_with_inputpath_should_warn(self):

    with self.assertWarnsRegex(
        UserWarning, 'Local file paths are currently unsupported for I/O.'):
      component_op = components.load_component_from_text("""
          name: compoent use inputPath
          inputs:
          - {name: data, type: Datasets}
          implementation:
            container:
              image: dummy
              args:
              - {inputPath: data}
          """)


if __name__ == '__main__':
  unittest.main()
