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

import json
import os
import shutil
import tempfile
import unittest

from kfp.v2 import components
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

    @dsl.pipeline(name='pipeline-with-exit-handler')
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

      @dsl.pipeline(name='pipeline-with-graph-component')
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
        TypeError, ' type "Float" cannot be paired with InputUriPlaceholder.'):
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

  def test_compile_pipeline_with_invalid_name_should_raise_error(self):

    def my_pipeline():
      pass

    with self.assertRaisesRegex(
        ValueError,
        'Invalid pipeline name: .*\nPlease specify a pipeline name that matches'
    ):
      compiler.Compiler().compile(
          pipeline_func=my_pipeline,
          pipeline_root='dummy',
          output_path='output.json')

  def test_compile_pipeline_with_importer_on_inputpath_should_raise_error(self):

    # YAML componet authoring
    component_op = components.load_component_from_text("""
        name: compoent with misused placeholder
        inputs:
        - {name: model, type: Model}
        implementation:
          container:
            image: dummy
            args:
            - {inputPath: model}
        """)

    @dsl.pipeline(name='my-component')
    def my_pipeline(model):
      component_op(model=model)

    with self.assertRaisesRegex(
        TypeError,
        'Input "model" with type "Model" is not connected to any upstream '
        'output. However it is used with InputPathPlaceholder.'):
      compiler.Compiler().compile(
          pipeline_func=my_pipeline,
          pipeline_root='dummy',
          output_path='output.json')

    # Python function based component authoring
    def my_component(datasets: components.InputPath('Datasets')):
      pass

    component_op = components.create_component_from_func(my_component)

    @dsl.pipeline(name='my-component')
    def my_pipeline(datasets):
      component_op(datasets=datasets)

    with self.assertRaisesRegex(
        TypeError,
        'Input "datasets" with type "Datasets" is not connected to any upstream '
        'output. However it is used with InputPathPlaceholder.'):
      compiler.Compiler().compile(
          pipeline_func=my_pipeline,
          pipeline_root='dummy',
          output_path='output.json')

  def test_set_pipeline_root_through_pipeline_decorator(self):

    tmpdir = tempfile.mkdtemp()
    try:

      @dsl.pipeline(name='my-pipeline', pipeline_root='gs://path')
      def my_pipeline():
        pass

      target_json_file = os.path.join(tmpdir, 'result.json')
      compiler.Compiler().compile(
          pipeline_func=my_pipeline, output_path=target_json_file)

      self.assertTrue(os.path.exists(target_json_file))
      with open(target_json_file) as f:
        job_spec = json.load(f)
      self.assertEqual('gs://path',
                       job_spec['runtimeConfig']['gcsOutputDirectory'])
    finally:
      shutil.rmtree(tmpdir)

  def test_set_pipeline_root_through_compile_method(self):

    tmpdir = tempfile.mkdtemp()
    try:

      @dsl.pipeline(name='my-pipeline', pipeline_root='gs://path')
      def my_pipeline():
        pass

      target_json_file = os.path.join(tmpdir, 'result.json')
      compiler.Compiler().compile(
          pipeline_func=my_pipeline,
          pipeline_root='gs://path-override',
          output_path=target_json_file)

      self.assertTrue(os.path.exists(target_json_file))
      with open(target_json_file) as f:
        job_spec = json.load(f)
      self.assertEqual('gs://path-override',
                       job_spec['runtimeConfig']['gcsOutputDirectory'])
    finally:
      shutil.rmtree(tmpdir)

  def test_missing_pipeline_root_is_allowed_but_warned(self):

    tmpdir = tempfile.mkdtemp()
    try:

      @dsl.pipeline(name='my-pipeline')
      def my_pipeline():
        pass

      target_json_file = os.path.join(tmpdir, 'result.json')
      with self.assertWarnsRegex(UserWarning, 'pipeline_root is None or empty'):
        compiler.Compiler().compile(
            pipeline_func=my_pipeline, output_path=target_json_file)

      self.assertTrue(os.path.exists(target_json_file))
      with open(target_json_file) as f:
        job_spec = json.load(f)
      self.assertTrue('gcsOutputDirectory' not in job_spec['runtimeConfig'])
    finally:
      shutil.rmtree(tmpdir)


if __name__ == '__main__':
  unittest.main()
