# Copyright 2018 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


import mlp
import mlpc
import os
import shutil
import subprocess
import sys
import tempfile
import unittest
import yaml


class TestCompiler(unittest.TestCase):

  def test_operator_to_template(self):
    """Test converting operator to template"""

    with mlp.Pipeline('somename') as p:
      msg1 = mlp.PipelineParam('msg1')
      msg2 = mlp.PipelineParam('msg2', value='hello')
      op = mlp.ContainerOp(name='echo', image='image', command=['sh', '-c'],
                           arguments=['echo %s %s | tee /tmp/message.txt' % (msg1, msg2)],
                           argument_inputs=[msg2, msg1],
                           file_outputs={'merged': '/tmp/message.txt'})
    golden_output = {
      'container': {
        'args': [
          'echo {{inputs.parameters.msg1}} {{inputs.parameters.msg2}} | tee /tmp/message.txt'
        ],
        'command': ['sh', '-c'],
        'image': 'image'},
        'inputs': {'parameters':
          [
            {'name': 'msg1'},
            {'name': 'msg2', 'value': 'hello'}
          ]},
        'name': 'echo',
        'outputs': {
          'parameters': [
            {'name': 'echo-merged',
             'valueFrom': {'path': '/tmp/message.txt'}
            }]
       }}
    self.maxDiff = None
    self.assertEqual(golden_output, mlpc.Compiler()._op_to_template(op))

  def test_basic_workflow(self):
    """Test compiling a basic workflow."""

    test_data_dir = os.path.join(os.path.dirname(__file__), 'testdata')
    sys.path.append(test_data_dir)
    import basic
    tmpdir = tempfile.mkdtemp()
    package_path = os.path.join(tmpdir, 'workflow.yaml')
    try:
      mlpc.Compiler().compile(basic.save_most_frequent_word, package_path)
      with open(os.path.join(test_data_dir, 'basic.yaml'), 'r') as f:
        golden = yaml.load(f)
      with open(package_path, 'r') as f:
        compiled = yaml.load(f)

      self.maxDiff = None
      # Comment next line for generating golden yaml.
      self.assertEqual(golden, compiled)
    finally:
      # Replace next line with commented line for gathering golden yaml.
      shutil.rmtree(tmpdir)
      # print(tmpdir)

  def test_composing_workflow(self):
    """Test compiling a simple workflow, and a bigger one composed from the simple one."""

    test_data_dir = os.path.join(os.path.dirname(__file__), 'testdata')
    sys.path.append(test_data_dir)
    import compose
    tmpdir = tempfile.mkdtemp()
    try:
      # First make sure the simple pipeline can be compiled.
      simple_package_path = os.path.join(tmpdir, 'simple.yaml')
      mlpc.Compiler().compile(compose.save_most_frequent_word, simple_package_path)

      # Then make sure the composed pipeline can be compiled and also compare with golden.
      compose_package_path = os.path.join(tmpdir, 'compose.yaml')
      mlpc.Compiler().compile(compose.download_save_most_frequent_word, compose_package_path)
      with open(os.path.join(test_data_dir, 'compose.yaml'), 'r') as f:
        golden = yaml.load(f)
      with open(compose_package_path, 'r') as f:
        compiled = yaml.load(f)

      self.maxDiff = None
      # Comment next line for generating golden yaml.
      self.assertEqual(golden, compiled)
    finally:
      # Replace next line with commented line for gathering golden yaml.
      shutil.rmtree(tmpdir)
      # print(tmpdir)

  def test_invalid_pipelines(self):
    """Test invalid pipelines."""

    @mlp.pipeline(
      name='name',
      description='description'
    )
    def invalid_param_annotation(message: str, outputpath: mlp.PipelineParam):
      pass

    with self.assertRaises(ValueError):
      mlpc.Compiler()._compile(invalid_param_annotation)

    @mlp.pipeline(
      name='name',
      description='description'
    )
    def missing_param_annotation(message: mlp.PipelineParam, outputpath):
      pass

    with self.assertRaises(ValueError):
      mlpc.Compiler()._compile(missing_param_annotation)

    def missing_decoration(message: mlp.PipelineParam):
      pass

    with self.assertRaises(ValueError):
      mlpc.Compiler()._compile(missing_decoration)

  def test_package_compile(self):
    """Test compiling python packages."""

    test_data_dir = os.path.join(os.path.dirname(__file__), 'testdata')
    test_package_dir = os.path.join(test_data_dir, 'testpackage')
    tmpdir = tempfile.mkdtemp()
    cwd = os.getcwd()
    try:
      os.chdir(test_package_dir)
      subprocess.check_call(['python3', 'setup.py', 'sdist', '--format=gztar', '-d', tmpdir])
      package_path = os.path.join(tmpdir, 'testsample-0.1.tar.gz')
      target_yaml = os.path.join(tmpdir, 'compose.yaml')
      subprocess.check_call([
          'dsl-compile', '--package', package_path, '--namespace', 'mypipeline',
          '--output', target_yaml, '--function', 'download_save_most_frequent_word'])
      with open(os.path.join(test_data_dir, 'compose.yaml'), 'r') as f:
        golden = yaml.load(f)
      with open(target_yaml, 'r') as f:
        compiled = yaml.load(f)

      self.maxDiff = None
      self.assertEqual(golden, compiled)
    finally:
      shutil.rmtree(tmpdir)
      os.chdir(cwd)


  def test_py_compile(self):
    """Test compiling python files."""

    test_data_dir = os.path.join(os.path.dirname(__file__), 'testdata')
    py_file = os.path.join(test_data_dir, 'basic.py')
    tmpdir = tempfile.mkdtemp()
    try:
      target_yaml = os.path.join(tmpdir, 'basic.yaml')
      subprocess.check_call([
          'dsl-compile', '--py', py_file, '--output', target_yaml])
      with open(os.path.join(test_data_dir, 'basic.yaml'), 'r') as f:
        golden = yaml.load(f)
      with open(target_yaml, 'r') as f:
        compiled = yaml.load(f)

      self.maxDiff = None
      self.assertEqual(golden, compiled)
    finally:
      shutil.rmtree(tmpdir)
