# Copyright 2021 The Kubeflow Authors
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
"""Tests for v2-compatible compiled pipelines."""

import os
import tempfile
from typing import Callable
import unittest
import yaml

from kfp import compiler, components, dsl
from kfp.components import InputPath, OutputPath


def preprocess(uri: str, some_int: int, output_parameter_one: OutputPath(int),
               output_dataset_one: OutputPath('Dataset')):
  '''Dummy Preprocess Step.'''
  with open(output_dataset_one, 'w') as f:
    f.write('Output dataset')
  with open(output_parameter_one, 'w') as f:
    f.write("{}".format(1234))


def train(dataset: InputPath('Dataset'),
          model: OutputPath('Model'),
          num_steps: int = 100):
  '''Dummy Training Step.'''

  with open(dataset, 'r') as input_file:
    input_string = input_file.read()
    with open(model, 'w') as output_file:
      for i in range(num_steps):
        output_file.write("Step {}\n{}\n=====\n".format(i, input_string))


preprocess_op = components.create_component_from_func(preprocess,
                                                      base_image='python:3.9')
train_op = components.create_component_from_func(train)


class TestV2CompatibleModeCompiler(unittest.TestCase):

  def _assert_compiled_pipeline_equals_golden(self,
                                              kfp_compiler: compiler.Compiler,
                                              pipeline_func: Callable,
                                              golden_yaml_filename: str):
    compiled_file = os.path.join(tempfile.mkdtemp(), 'workflow.yaml')
    kfp_compiler.compile(pipeline_func, package_path=compiled_file)

    test_data_dir = os.path.join(os.path.dirname(__file__), 'testdata')
    golden_file = os.path.join(test_data_dir, golden_yaml_filename)
    # Uncomment the following to update goldens.
    # TODO: place this behind some --update_goldens flag.
    # kfp_compiler.compile(pipeline_func, package_path=golden_file)

    with open(golden_file, 'r') as f:
      golden = yaml.safe_load(f)

    with open(compiled_file, 'r') as f:
      compiled = yaml.safe_load(f)

    for workflow in golden, compiled:
      del workflow['metadata']
      for template in workflow['spec']['templates']:
        template.pop('metadata', None)

        if 'initContainers' not in template:
          continue
        # Strip off the launcher image label before comparison
        for initContainer in template['initContainers']:
          initContainer['image'] = initContainer['image'].split(':')[0]

    self.maxDiff = None
    self.assertDictEqual(golden, compiled)

  def test_two_step_pipeline(self):

    @dsl.pipeline(pipeline_root='gs://output-directory/v2-artifacts',
                  name='my-test-pipeline')
    def v2_compatible_two_step_pipeline():
      preprocess_task = preprocess_op(uri='uri-to-import', some_int=12)
      train_task = train_op(
          num_steps=preprocess_task.outputs['output_parameter_one'],
          dataset=preprocess_task.outputs['output_dataset_one'])

    kfp_compiler = compiler.Compiler(
        mode=dsl.PipelineExecutionMode.V2_COMPATIBLE)
    self._assert_compiled_pipeline_equals_golden(
        kfp_compiler, v2_compatible_two_step_pipeline,
        'v2_compatible_two_step_pipeline.yaml')

  def test_custom_launcher(self):

    @dsl.pipeline(pipeline_root='gs://output-directory/v2-artifacts',
                  name='my-test-pipeline-with-custom-launcher')
    def v2_compatible_two_step_pipeline():
      preprocess_task = preprocess_op(uri='uri-to-import', some_int=12)
      train_task = train_op(
          num_steps=preprocess_task.outputs['output_parameter_one'],
          dataset=preprocess_task.outputs['output_dataset_one'])

    kfp_compiler = compiler.Compiler(
        mode=dsl.PipelineExecutionMode.V2_COMPATIBLE,
        launcher_image='my-custom-image')
    self._assert_compiled_pipeline_equals_golden(
        kfp_compiler, v2_compatible_two_step_pipeline,
        'v2_compatible_two_step_pipeline_with_custom_launcher.yaml')

  def test_constructing_container_op_directly_should_error(
      self):

    @dsl.pipeline(name='test-pipeline')
    def my_pipeline():
      dsl.ContainerOp(
          name='comp1',
          image='gcr.io/dummy',
          command=['python', 'main.py']
      )

    with self.assertRaisesRegex(
        RuntimeError,
        'Constructing ContainerOp instances directly is deprecated and not '
        'supported when compiling to v2 \(using v2 compiler or v1 compiler '
        'with V2_COMPATIBLE or V2_ENGINE mode\).'):
      compiler.Compiler(mode=dsl.PipelineExecutionMode.V2_COMPATIBLE).compile(
          pipeline_func=my_pipeline, package_path='result.json')

if __name__ == '__main__':
  unittest.main()
