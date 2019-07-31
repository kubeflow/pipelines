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

from kfp.compiler._component_builder import _generate_dockerfile, _dependency_to_requirements, _func_to_entrypoint
from kfp.compiler._component_builder import CodeGenerator
from kfp.compiler._component_builder import VersionedDependency
from kfp.compiler._component_builder import DependencyHelper

import os
import unittest
import yaml
import tarfile
from pathlib import Path
import inspect
from collections import OrderedDict

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

class TestGenerator(unittest.TestCase):
  def test_generate_dockerfile(self):
    """ Test generate dockerfile """

    # prepare
    test_data_dir = os.path.join(os.path.dirname(__file__), 'testdata')
    target_dockerfile = os.path.join(test_data_dir, 'component.temp.dockerfile')
    golden_dockerfile_payload_one = '''\
FROM gcr.io/ngao-mlpipeline-testing/tensorflow:1.10.0
RUN apt-get update -y && apt-get install --no-install-recommends -y -q python3 python3-pip python3-setuptools
ADD main.py /ml/main.py
ENTRYPOINT ["python3", "-u", "/ml/main.py"]'''
    golden_dockerfile_payload_two = '''\
FROM gcr.io/ngao-mlpipeline-testing/tensorflow:1.10.0
RUN apt-get update -y && apt-get install --no-install-recommends -y -q python3 python3-pip python3-setuptools
ADD requirements.txt /ml/requirements.txt
RUN pip3 install -r /ml/requirements.txt
ADD main.py /ml/main.py
ENTRYPOINT ["python3", "-u", "/ml/main.py"]'''

    golden_dockerfile_payload_three = '''\
FROM gcr.io/ngao-mlpipeline-testing/tensorflow:1.10.0
RUN apt-get update -y && apt-get install --no-install-recommends -y -q python python-pip python-setuptools
ADD requirements.txt /ml/requirements.txt
RUN pip install -r /ml/requirements.txt
ADD main.py /ml/main.py
ENTRYPOINT ["python", "-u", "/ml/main.py"]'''
    # check
    _generate_dockerfile(filename=target_dockerfile, base_image='gcr.io/ngao-mlpipeline-testing/tensorflow:1.10.0',
                         entrypoint_filename='main.py', python_version='python3')
    with open(target_dockerfile, 'r') as f:
      target_dockerfile_payload = f.read()
    self.assertEqual(target_dockerfile_payload, golden_dockerfile_payload_one)
    _generate_dockerfile(filename=target_dockerfile, base_image='gcr.io/ngao-mlpipeline-testing/tensorflow:1.10.0',
                         entrypoint_filename='main.py', python_version='python3', requirement_filename='requirements.txt')
    with open(target_dockerfile, 'r') as f:
      target_dockerfile_payload = f.read()
    self.assertEqual(target_dockerfile_payload, golden_dockerfile_payload_two)
    _generate_dockerfile(filename=target_dockerfile, base_image='gcr.io/ngao-mlpipeline-testing/tensorflow:1.10.0',
                         entrypoint_filename='main.py', python_version='python2', requirement_filename='requirements.txt')
    with open(target_dockerfile, 'r') as f:
      target_dockerfile_payload = f.read()
    self.assertEqual(target_dockerfile_payload, golden_dockerfile_payload_three)

    self.assertRaises(ValueError, _generate_dockerfile, filename=target_dockerfile,
                      base_image='gcr.io/ngao-mlpipeline-testing/tensorflow:1.10.0', entrypoint_filename='main.py',
                      python_version='python4', requirement_filename='requirements.txt')

    # clean up
    os.remove(target_dockerfile)

  def test_generate_requirement(self):
    # prepare
    test_data_dir = os.path.join(os.path.dirname(__file__), 'testdata')
    temp_file = os.path.join(test_data_dir, 'test_requirements.tmp')

    dependencies = [
        VersionedDependency(name='tensorflow', min_version='0.10.0', max_version='0.11.0'),
        VersionedDependency(name='kubernetes', min_version='0.6.0'),
    ]
    _dependency_to_requirements(dependencies, filename=temp_file)
    golden_payload = '''\
tensorflow >= 0.10.0, <= 0.11.0
kubernetes >= 0.6.0
'''
    with open(temp_file, 'r') as f:
      target_payload = f.read()
    self.assertEqual(target_payload, golden_payload)
    os.remove(temp_file)

  def test_generate_entrypoint(self):
    """ Test entrypoint generation """

    # prepare
    test_data_dir = os.path.join(os.path.dirname(__file__), 'testdata')

    # check
    generated_codes = _func_to_entrypoint(component_func=sample_component_func)
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

    generated_codes = _func_to_entrypoint(component_func=sample_component_func_two)
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

    generated_codes = _func_to_entrypoint(component_func=sample_component_func_three)
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
    generated_codes = _func_to_entrypoint(component_func=sample_component_func_two, python_version='python2')
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
