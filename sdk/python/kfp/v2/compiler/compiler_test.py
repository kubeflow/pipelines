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

from kfp import components
from kfp.v2 import compiler
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
      def simple_pipeline(pipeline_input='Hello KFP!',):
        producer = producer_op(input_param=pipeline_input)
        consumer = consumer_op(
            input_model=producer.outputs['output_model'],
            input_value=producer.outputs['output_value'])

      target_json_file = os.path.join(tmpdir, 'result.json')
      compiler.Compiler().compile(simple_pipeline, target_json_file)

      self.assertTrue(os.path.exists(target_json_file))
    finally:
      shutil.rmtree(tmpdir)

  def test_compile_pipeline_with_dsl_condition_should_raise_error(self):

    class FlipCoinOp(dsl.ContainerOp):

      def __init__(self, name):
        super(FlipCoinOp, self).__init__(
            name=name,
            image='python:alpine3.6',
            command=['sh', '-c'],
            arguments=[
                'python -c "import random; result = \'heads\' if random.randint(0,1) == 0 '
                'else \'tails\'; print(result)" | tee /tmp/output'
            ],
            file_outputs={'output': '/tmp/output'})

    class PrintOp(dsl.ContainerOp):

      def __init__(self, name, msg):
        super(PrintOp, self).__init__(
            name=name, image='alpine:3.6', command=['echo', msg])

    @dsl.pipeline()
    def flipcoin():
      flip = FlipCoinOp('flip')

      with dsl.Condition(flip.output == 'heads'):
        flip2 = FlipCoinOp('flip-again')

        with dsl.Condition(flip2.output == 'tails'):
          PrintOp('print1', flip2.output)

      with dsl.Condition(flip.output == 'tails'):
        PrintOp('print2', flip2.output)

    with self.assertRaises(NotImplementedError) as cm:
      compiler.Compiler().compile(flipcoin, 'output.json')

    self.assertEqual('dsl.Condition is not yet supported in KFP v2 compiler.',
                     str(cm.exception))

  def test_compile_pipeline_with_dsl_exithandler_should_raise_error(self):

    def gcs_download_op(url):
      return dsl.ContainerOp(
          name='GCS - Download',
          image='google/cloud-sdk:279.0.0',
          command=['sh', '-c'],
          arguments=['gsutil cat $0 | tee $1', url, '/tmp/results.txt'],
          file_outputs={
              'data': '/tmp/results.txt',
          })

    def echo_op(text):
      return dsl.ContainerOp(
          name='echo',
          image='library/bash:4.4.23',
          command=['sh', '-c'],
          arguments=['echo "$0"', text],
      )

    @dsl.pipeline()
    def download_and_print(url='gs://ml-pipeline/shakespeare/shakespeare1.txt'):
      """A sample pipeline showing exit handler."""

      exit_task = echo_op('exit!')

      with dsl.ExitHandler(exit_task):
        download_task = gcs_download_op(url)
        echo_task = echo_op(download_task.output)

    with self.assertRaises(NotImplementedError) as cm:
      compiler.Compiler().compile(download_and_print, 'output.json')

    self.assertEqual('dsl.ExitHandler is not yet supported in KFP v2 compiler.',
                     str(cm.exception))

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

    with self.assertRaises(NotImplementedError) as cm:
      compiler.Compiler().compile(my_pipeline, 'output.json')

    self.assertEqual('dsl.ParallelFor is not yet supported in KFP v2 compiler.',
                     str(cm.exception))

  def test_compile_pipeline_with_dsl_graph_component_should_raise_error(self):

    with self.assertRaises(NotImplementedError) as cm:

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

      compiler.Compiler().compile(opsgroups_pipeline, 'output.json')

    self.assertEqual(
        'dsl.graph_component is not yet supported in KFP v2 compiler.',
        str(cm.exception))


if __name__ == '__main__':
  unittest.main()
