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

from kfp.compiler._component_builder import GCSHelper
from kfp.compiler._component_builder import DockerfileHelper
from kfp.compiler._component_builder import CodeGenerator
from kfp.compiler._component_builder import ImageBuilder
from kfp.compiler._component_builder import VersionedDependency
from kfp.compiler._component_builder import DependencyHelper

import os
import unittest
import yaml
import tarfile
from pathlib import Path
import inspect
from collections import OrderedDict

GCS_BASE = 'gs://kfp-testing/'

class TestVersionedDependency(unittest.TestCase):

  def test_version(self):
    """ test version overrides min_version and max_version """
    version = VersionedDependency(name='tensorflow', version='0.3.0', min_version='0.1.0', max_version='0.4.0')
    self.assertTrue(version.min_version == '0.3.0')
    self.assertTrue(version.max_version == '0.3.0')
    self.assertTrue(version.has_versions())
    self.assertTrue(version.name == 'tensorflow')

  def test_minmax_version(self):
    """ test if min_version and max_version are configured when version is not given """
    version = VersionedDependency(name='tensorflow', min_version='0.1.0', max_version='0.4.0')
    self.assertTrue(version.min_version == '0.1.0')
    self.assertTrue(version.max_version == '0.4.0')
    self.assertTrue(version.has_versions())

  def test_min_or_max_version(self):
    """ test if min_version and max_version are configured when version is not given """
    version = VersionedDependency(name='tensorflow', min_version='0.1.0')
    self.assertTrue(version.min_version == '0.1.0')
    self.assertTrue(version.has_versions())
    version = VersionedDependency(name='tensorflow', max_version='0.3.0')
    self.assertTrue(version.max_version == '0.3.0')
    self.assertTrue(version.has_versions())

  def test_no_version(self):
    """ test the no version scenario """
    version = VersionedDependency(name='tensorflow')
    self.assertFalse(version.has_min_version())
    self.assertFalse(version.has_max_version())
    self.assertFalse(version.has_versions())

class TestDependencyHelper(unittest.TestCase):

  def test_generate_requirement(self):
    """ Test generating requirement file """

    # prepare
    test_data_dir = os.path.join(os.path.dirname(__file__), 'testdata')
    temp_file = os.path.join(test_data_dir, 'test_requirements.tmp')

    dependency_helper = DependencyHelper()
    dependency_helper.add_python_package(dependency=VersionedDependency(name='tensorflow', min_version='0.10.0', max_version='0.11.0'))
    dependency_helper.add_python_package(dependency=VersionedDependency(name='kubernetes', min_version='0.6.0'))
    dependency_helper.add_python_package(dependency=VersionedDependency(name='pytorch', max_version='0.3.0'))
    dependency_helper.generate_pip_requirements(temp_file)

    golden_requirement_payload = '''\
tensorflow >= 0.10.0, <= 0.11.0
kubernetes >= 0.6.0
pytorch <= 0.3.0
'''
    with open(temp_file, 'r') as f:
      target_requirement_payload = f.read()
    self.assertEqual(target_requirement_payload, golden_requirement_payload)
    os.remove(temp_file)

  def test_add_python_package(self):
    """ Test add_python_package """

    # prepare
    test_data_dir = os.path.join(os.path.dirname(__file__), 'testdata')
    temp_file = os.path.join(test_data_dir, 'test_requirements.tmp')

    dependency_helper = DependencyHelper()
    dependency_helper.add_python_package(dependency=VersionedDependency(name='tensorflow', min_version='0.10.0', max_version='0.11.0'))
    dependency_helper.add_python_package(dependency=VersionedDependency(name='kubernetes', min_version='0.6.0'))
    dependency_helper.add_python_package(dependency=VersionedDependency(name='tensorflow', min_version='0.12.0'), override=True)
    dependency_helper.add_python_package(dependency=VersionedDependency(name='kubernetes', min_version='0.8.0'), override=False)
    dependency_helper.add_python_package(dependency=VersionedDependency(name='pytorch', version='0.3.0'))
    dependency_helper.generate_pip_requirements(temp_file)
    golden_requirement_payload = '''\
tensorflow >= 0.12.0
kubernetes >= 0.6.0
pytorch >= 0.3.0, <= 0.3.0
'''
    with open(temp_file, 'r') as f:
      target_requirement_payload = f.read()
    self.assertEqual(target_requirement_payload, golden_requirement_payload)
    os.remove(temp_file)


class TestDockerfileHelper(unittest.TestCase):

  def test_wrap_files_in_tarball(self):
    """ Test wrap files in a tarball """

    # prepare
    test_data_dir = os.path.join(os.path.dirname(__file__), 'testdata')
    temp_file_one = os.path.join(test_data_dir, 'test_data_one.tmp')
    temp_file_two = os.path.join(test_data_dir, 'test_data_two.tmp')
    temp_tarball = os.path.join(test_data_dir, 'test_data.tmp.tar.gz')
    with open(temp_file_one, 'w') as f:
      f.write('temporary file one content')
    with open(temp_file_two, 'w') as f:
      f.write('temporary file two content')

    # check
    docker_helper = DockerfileHelper(arc_dockerfile_name='')
    docker_helper._wrap_files_in_tarball(temp_tarball, {'dockerfile':temp_file_one, 'main.py':temp_file_two})
    self.assertTrue(os.path.exists(temp_tarball))
    with tarfile.open(temp_tarball) as temp_tarball_handle:
      temp_files = temp_tarball_handle.getmembers()
      self.assertTrue(len(temp_files) == 2)
      for temp_file in temp_files:
        self.assertTrue(temp_file.name in ['dockerfile', 'main.py'])

    # clean up
    os.remove(temp_file_one)
    os.remove(temp_file_two)
    os.remove(temp_tarball)

  def test_generate_dockerfile(self):
    """ Test generate dockerfile """

    # prepare
    test_data_dir = os.path.join(os.path.dirname(__file__), 'testdata')
    target_dockerfile = os.path.join(test_data_dir, 'component.temp.dockerfile')
    golden_dockerfile_payload_one = '''\
FROM gcr.io/ngao-mlpipeline-testing/tensorflow:1.10.0
RUN apt-get update -y && apt-get install --no-install-recommends -y -q python3 python3-pip python3-setuptools
ADD main.py /ml/
ENTRYPOINT ["python3", "/ml/main.py"]'''
    golden_dockerfile_payload_two = '''\
FROM gcr.io/ngao-mlpipeline-testing/tensorflow:1.10.0
RUN apt-get update -y && apt-get install --no-install-recommends -y -q python3 python3-pip python3-setuptools
ADD requirements.txt /ml/
RUN pip3 install -r /ml/requirements.txt
ADD main.py /ml/
ENTRYPOINT ["python3", "/ml/main.py"]'''

    golden_dockerfile_payload_three = '''\
FROM gcr.io/ngao-mlpipeline-testing/tensorflow:1.10.0
RUN apt-get update -y && apt-get install --no-install-recommends -y -q python python-pip python-setuptools
ADD requirements.txt /ml/
RUN pip install -r /ml/requirements.txt
ADD main.py /ml/
ENTRYPOINT ["python", "/ml/main.py"]'''
    # check
    docker_helper = DockerfileHelper(arc_dockerfile_name=target_dockerfile)
    docker_helper._generate_dockerfile_with_py(target_file=target_dockerfile, base_image='gcr.io/ngao-mlpipeline-testing/tensorflow:1.10.0',
                                               python_filepath='main.py', has_requirement_file=False, python_version='python3')
    with open(target_dockerfile, 'r') as f:
      target_dockerfile_payload = f.read()
    self.assertEqual(target_dockerfile_payload, golden_dockerfile_payload_one)
    docker_helper._generate_dockerfile_with_py(target_file=target_dockerfile, base_image='gcr.io/ngao-mlpipeline-testing/tensorflow:1.10.0',
                                               python_filepath='main.py', has_requirement_file=True, python_version='python3')
    with open(target_dockerfile, 'r') as f:
      target_dockerfile_payload = f.read()
    self.assertEqual(target_dockerfile_payload, golden_dockerfile_payload_two)
    docker_helper._generate_dockerfile_with_py(target_file=target_dockerfile, base_image='gcr.io/ngao-mlpipeline-testing/tensorflow:1.10.0',
                                               python_filepath='main.py', has_requirement_file=True, python_version='python2')
    with open(target_dockerfile, 'r') as f:
      target_dockerfile_payload = f.read()
    self.assertEqual(target_dockerfile_payload, golden_dockerfile_payload_three)

    self.assertRaises(ValueError, docker_helper._generate_dockerfile_with_py, target_file=target_dockerfile,
                      base_image='gcr.io/ngao-mlpipeline-testing/tensorflow:1.10.0', python_filepath='main.py',
                      has_requirement_file=True, python_version='python4')

    # clean up
    os.remove(target_dockerfile)

  def test_prepare_docker_with_py(self):
    """ Test the whole prepare docker from python function """

    # prepare
    test_data_dir = os.path.join(os.path.dirname(__file__), 'testdata')
    python_filepath = os.path.join(test_data_dir, 'basic.py')
    local_tarball_path = os.path.join(test_data_dir, 'test_docker.tar.gz')

    # check
    docker_helper = DockerfileHelper(arc_dockerfile_name='dockerfile')
    docker_helper.prepare_docker_tarball_with_py(arc_python_filename='main.py', python_filepath=python_filepath,
                                                 base_image='gcr.io/ngao-mlpipeline-testing/tensorflow:1.8.0',
                                                 local_tarball_path=local_tarball_path, python_version='python3')
    with tarfile.open(local_tarball_path) as temp_tarball_handle:
      temp_files = temp_tarball_handle.getmembers()
      self.assertTrue(len(temp_files) == 2)
      for temp_file in temp_files:
        self.assertTrue(temp_file.name in ['dockerfile', 'main.py'])

    # clean up
    os.remove(local_tarball_path)

  def test_prepare_docker_with_py_and_dependency(self):
    """ Test the whole prepare docker from python function and dependencies """

    # prepare
    test_data_dir = os.path.join(os.path.dirname(__file__), 'testdata')
    python_filepath = os.path.join(test_data_dir, 'basic.py')
    local_tarball_path = os.path.join(test_data_dir, 'test_docker.tar.gz')

    # check
    docker_helper = DockerfileHelper(arc_dockerfile_name='dockerfile')
    dependencies = {
      VersionedDependency(name='tensorflow', min_version='0.10.0', max_version='0.11.0'),
      VersionedDependency(name='kubernetes', min_version='0.6.0'),
    }
    docker_helper.prepare_docker_tarball_with_py(arc_python_filename='main.py', python_filepath=python_filepath,
                                                 base_image='gcr.io/ngao-mlpipeline-testing/tensorflow:1.8.0',
                                                 local_tarball_path=local_tarball_path, python_version='python3',
                                                 dependency=dependencies)
    with tarfile.open(local_tarball_path) as temp_tarball_handle:
      temp_files = temp_tarball_handle.getmembers()
      self.assertTrue(len(temp_files) == 3)
      for temp_file in temp_files:
        self.assertTrue(temp_file.name in ['dockerfile', 'main.py', 'requirements.txt'])

      # clean up
    os.remove(local_tarball_path)

  def test_prepare_docker_tarball(self):
    """ Test the whole prepare docker tarball """

    # prepare
    test_data_dir = os.path.join(os.path.dirname(__file__), 'testdata')
    dockerfile_path = os.path.join(test_data_dir, 'component.target.dockerfile')
    Path(dockerfile_path).touch()
    local_tarball_path = os.path.join(test_data_dir, 'test_docker.tar.gz')

    # check
    docker_helper = DockerfileHelper(arc_dockerfile_name='dockerfile')
    docker_helper.prepare_docker_tarball(dockerfile_path=dockerfile_path, local_tarball_path=local_tarball_path)
    with tarfile.open(local_tarball_path) as temp_tarball_handle:
      temp_files = temp_tarball_handle.getmembers()
      self.assertTrue(len(temp_files) == 1)
      for temp_file in temp_files:
        self.assertTrue(temp_file.name in ['dockerfile'])

    # clean up
    os.remove(local_tarball_path)
    os.remove(dockerfile_path)

# hello function is used by the TestCodeGenerator to verify the auto generated python function
def hello():
  print("hello")

class TestCodeGenerator(unittest.TestCase):
  def test_codegen(self):
    """ Test code generator a function"""
    codegen = CodeGenerator(indentation='  ')
    codegen.begin()
    codegen.writeline('def hello():')
    codegen.indent()
    codegen.writeline('print("hello")')
    generated_codes = codegen.end()
    self.assertEqual(generated_codes, inspect.getsource(hello))

def sample_component_func(a: str, b: int) -> float:
  result = 3.45
  if a == "succ":
    result = float(b + 5)
  return result

def basic_decorator(name):
  def wrapper(func):
    return func
  return wrapper

@basic_decorator(name='component_sample')
def sample_component_func_two(a: str, b: int) -> float:
  result = 3.45
  if a == 'succ':
    result = float(b + 5)
  return result

def sample_component_func_three() -> float:
  return 1.0

class TestImageBuild(unittest.TestCase):

  def test_generate_kaniko_yaml(self):
    """ Test generating the kaniko job yaml """

    # prepare
    test_data_dir = os.path.join(os.path.dirname(__file__), 'testdata')

    # check
    builder = ImageBuilder(gcs_base=GCS_BASE, target_image='')
    generated_yaml = builder._generate_kaniko_spec(namespace='default', arc_dockerfile_name='dockerfile',
                                                   gcs_path='gs://mlpipeline/kaniko_build.tar.gz', target_image='gcr.io/mlpipeline/kaniko_image:latest')
    with open(os.path.join(test_data_dir, 'kaniko.basic.yaml'), 'r') as f:
      golden = yaml.load(f)

    self.assertEqual(golden, generated_yaml)

  def test_generate_entrypoint(self):
    """ Test entrypoint generation """

    # prepare
    test_data_dir = os.path.join(os.path.dirname(__file__), 'testdata')

    # check
    builder = ImageBuilder(gcs_base=GCS_BASE, target_image='')
    generated_codes = builder._generate_entrypoint(component_func=sample_component_func)
    golden = '''\
def sample_component_func(a: str, b: int) -> float:
  result = 3.45
  if a == "succ":
    result = float(b + 5)
  return result

def wrapper_sample_component_func(a,b,_output_file):
  output = sample_component_func(str(a),int(b))
  import os
  os.makedirs(os.path.dirname(_output_file))
  with open(_output_file, "w") as data:
    data.write(str(output))

import argparse
parser = argparse.ArgumentParser(description="Parsing arguments")
parser.add_argument("a", type=str)
parser.add_argument("b", type=int)
parser.add_argument("_output_file", type=str)
args = vars(parser.parse_args())

if __name__ == "__main__":
  wrapper_sample_component_func(**args)
'''
    self.assertEqual(golden, generated_codes)

    generated_codes = builder._generate_entrypoint(component_func=sample_component_func_two)
    golden = '''\
def sample_component_func_two(a: str, b: int) -> float:
  result = 3.45
  if a == 'succ':
    result = float(b + 5)
  return result

def wrapper_sample_component_func_two(a,b,_output_file):
  output = sample_component_func_two(str(a),int(b))
  import os
  os.makedirs(os.path.dirname(_output_file))
  with open(_output_file, "w") as data:
    data.write(str(output))

import argparse
parser = argparse.ArgumentParser(description="Parsing arguments")
parser.add_argument("a", type=str)
parser.add_argument("b", type=int)
parser.add_argument("_output_file", type=str)
args = vars(parser.parse_args())

if __name__ == "__main__":
  wrapper_sample_component_func_two(**args)
'''
    self.assertEqual(golden, generated_codes)

    generated_codes = builder._generate_entrypoint(component_func=sample_component_func_three)
    golden = '''\
def sample_component_func_three() -> float:
  return 1.0

def wrapper_sample_component_func_three(_output_file):
  output = sample_component_func_three()
  import os
  os.makedirs(os.path.dirname(_output_file))
  with open(_output_file, "w") as data:
    data.write(str(output))

import argparse
parser = argparse.ArgumentParser(description="Parsing arguments")
parser.add_argument("_output_file", type=str)
args = vars(parser.parse_args())

if __name__ == "__main__":
  wrapper_sample_component_func_three(**args)
'''
    self.assertEqual(golden, generated_codes)

  def test_generate_entrypoint_python2(self):
    """ Test entrypoint generation for python2"""

    # prepare
    test_data_dir = os.path.join(os.path.dirname(__file__), 'testdata')

    # check
    builder = ImageBuilder(gcs_base=GCS_BASE, target_image='')
    generated_codes = builder._generate_entrypoint(component_func=sample_component_func_two, python_version='python2')
    golden = '''\
def sample_component_func_two(a, b):
  result = 3.45
  if a == 'succ':
    result = float(b + 5)
  return result

def wrapper_sample_component_func_two(a,b,_output_file):
  output = sample_component_func_two(str(a),int(b))
  import os
  os.makedirs(os.path.dirname(_output_file))
  with open(_output_file, "w") as data:
    data.write(str(output))

import argparse
parser = argparse.ArgumentParser(description="Parsing arguments")
parser.add_argument("a", type=str)
parser.add_argument("b", type=int)
parser.add_argument("_output_file", type=str)
args = vars(parser.parse_args())

if __name__ == "__main__":
  wrapper_sample_component_func_two(**args)
'''
    self.assertEqual(golden, generated_codes)