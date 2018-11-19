# Copyright 2018 Google LLC
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


import kfp.compiler as compiler
import kfp.dsl as dsl
import os
import shutil
import subprocess
import sys
import tarfile
import tempfile
import unittest
import yaml

class TestCompiler(unittest.TestCase):

  def test_operator_to_template(self):
    """Test converting operator to template"""

    from kubernetes import client as k8s_client

    with dsl.Pipeline('somename') as p:
      msg1 = dsl.PipelineParam('msg1')
      msg2 = dsl.PipelineParam('msg2', value='value2')
      op = dsl.ContainerOp(name='echo', image='image', command=['sh', '-c'],
                           arguments=['echo %s %s | tee /tmp/message.txt' % (msg1, msg2)],
                           file_outputs={'merged': '/tmp/message.txt'})
      op.add_volume_mount(k8s_client.V1VolumeMount(
          mount_path='/secret/gcp-credentials',
          name='gcp-credentials'))
      op.add_env_variable(k8s_client.V1EnvVar(
          name='GOOGLE_APPLICATION_CREDENTIALS',
          value='/secret/gcp-credentials/user-gcp-sa.json'))
    golden_output = {
      'container': {
        'image': 'image',
        'args': [
          'echo {{inputs.parameters.msg1}} {{inputs.parameters.msg2}} | tee /tmp/message.txt'
        ],
        'command': ['sh', '-c'],
        'env': [
          {
            'name': 'GOOGLE_APPLICATION_CREDENTIALS',
            'value': '/secret/gcp-credentials/user-gcp-sa.json'
          }
        ],
        'volumeMounts':[
          {
            'mountPath': '/secret/gcp-credentials',
            'name': 'gcp-credentials',
          }
        ]
      },
      'inputs': {'parameters':
        [
          {'name': 'msg1'},
          {'name': 'msg2', 'value': 'value2'},
        ]},
      'name': 'echo',
      'outputs': {
        'parameters': [
          {'name': 'echo-merged',
           'valueFrom': {'path': '/tmp/message.txt'}
          }],
        'artifacts': [{
          'name': 'mlpipeline-ui-metadata',
          'path': '/mlpipeline-ui-metadata.json',
          's3': {
            'accessKeySecret': {
              'key': 'accesskey',
              'name': 'mlpipeline-minio-artifact',
            },
            'bucket': 'mlpipeline',
            'endpoint': 'minio-service.kubeflow:9000',
            'insecure': True,
            'key': 'runs/{{workflow.uid}}/{{pod.name}}/mlpipeline-ui-metadata.tgz',
            'secretKeySecret': {
              'key': 'secretkey',
              'name': 'mlpipeline-minio-artifact',
            }
          }
        },{
          'name': 'mlpipeline-metrics',
          'path': '/mlpipeline-metrics.json',
          's3': {
            'accessKeySecret': {
              'key': 'accesskey',
              'name': 'mlpipeline-minio-artifact',
            },
            'bucket': 'mlpipeline',
            'endpoint': 'minio-service.kubeflow:9000',
            'insecure': True,
            'key': 'runs/{{workflow.uid}}/{{pod.name}}/mlpipeline-metrics.tgz',
            'secretKeySecret': {
              'key': 'secretkey',
              'name': 'mlpipeline-minio-artifact',
            }
          }
        }]
      }
    }

    self.maxDiff = None
    self.assertEqual(golden_output, compiler.Compiler()._op_to_template(op))

  def _get_yaml_from_tar(self, tar_file):
    with tarfile.open(tar_file, 'r:gz') as tar:
      return yaml.load(tar.extractfile(tar.getmembers()[0]))

  def test_basic_workflow(self):
    """Test compiling a basic workflow."""

    test_data_dir = os.path.join(os.path.dirname(__file__), 'testdata')
    sys.path.append(test_data_dir)
    import basic
    tmpdir = tempfile.mkdtemp()
    package_path = os.path.join(tmpdir, 'workflow.tar.gz')
    try:
      compiler.Compiler().compile(basic.save_most_frequent_word, package_path)
      with open(os.path.join(test_data_dir, 'basic.yaml'), 'r') as f:
        golden = yaml.load(f)
      compiled = self._get_yaml_from_tar(package_path)

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
      simple_package_path = os.path.join(tmpdir, 'simple.tar.gz')
      compiler.Compiler().compile(compose.save_most_frequent_word, simple_package_path)

      # Then make sure the composed pipeline can be compiled and also compare with golden.
      compose_package_path = os.path.join(tmpdir, 'compose.tar.gz')
      compiler.Compiler().compile(compose.download_save_most_frequent_word, compose_package_path)
      with open(os.path.join(test_data_dir, 'compose.yaml'), 'r') as f:
        golden = yaml.load(f)
      compiled = self._get_yaml_from_tar(compose_package_path)

      self.maxDiff = None
      # Comment next line for generating golden yaml.
      self.assertEqual(golden, compiled)
    finally:
      # Replace next line with commented line for gathering golden yaml.
      shutil.rmtree(tmpdir)
      # print(tmpdir)

  def test_invalid_pipelines(self):
    """Test invalid pipelines."""

    @dsl.pipeline(
      name='name',
      description='description'
    )
    def invalid_param_defaults(message, outputpath='something'):
      pass

    with self.assertRaises(ValueError):
      compiler.Compiler()._compile(invalid_param_defaults)

    def missing_decoration(message: dsl.PipelineParam):
      pass

    with self.assertRaises(ValueError):
      compiler.Compiler()._compile(missing_decoration)

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
      target_tar = os.path.join(tmpdir, 'compose.tar.gz')
      subprocess.check_call([
          'dsl-compile', '--package', package_path, '--namespace', 'mypipeline',
          '--output', target_tar, '--function', 'download_save_most_frequent_word'])
      with open(os.path.join(test_data_dir, 'compose.yaml'), 'r') as f:
        golden = yaml.load(f)
      compiled = self._get_yaml_from_tar(target_tar)

      self.maxDiff = None
      self.assertEqual(golden, compiled)
    finally:
      shutil.rmtree(tmpdir)
      os.chdir(cwd)

  def _test_py_compile(self, file_base_name):
    test_data_dir = os.path.join(os.path.dirname(__file__), 'testdata')
    py_file = os.path.join(test_data_dir, file_base_name + '.py')
    tmpdir = tempfile.mkdtemp()
    try:
      target_tar = os.path.join(tmpdir, file_base_name + '.tar.gz')
      subprocess.check_call([
          'dsl-compile', '--py', py_file, '--output', target_tar])
      with open(os.path.join(test_data_dir, file_base_name + '.yaml'), 'r') as f:
        golden = yaml.load(f)
      compiled = self._get_yaml_from_tar(target_tar)

      self.maxDiff = None
      self.assertEqual(golden, compiled)
    finally:
      shutil.rmtree(tmpdir)
    
  def test_py_compile_basic(self):
    """Test basic sequential pipeline."""
    self._test_py_compile('basic')

  def test_py_compile_condition(self):
    """Test a pipeline with conditions."""
    self._test_py_compile('coin')

  def test_py_compile_immediate_value(self):
    """Test a pipeline with immediate value parameter."""
    self._test_py_compile('immediate_value')

  def test_py_compile_default_value(self):
    """Test a pipeline with a parameter with default value."""
    self._test_py_compile('default_value')

  def test_py_volume(self):
    """Test a pipeline with a volume and volume mount."""
    self._test_py_compile('volume')
