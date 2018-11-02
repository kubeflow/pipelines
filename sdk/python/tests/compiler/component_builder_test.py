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

import os
import unittest
import yaml
import tarfile
from pathlib import Path
import inspect

GCS_BASE = 'gs://ngao-mlpipeline-testing/'

class TestGCSHelper(unittest.TestCase):

  def test_upload_gcs_path(self):
    """ test uploading gcs file """
    # prepare
    test_data_dir = os.path.join(os.path.dirname(__file__), 'testdata')
    temp_file = os.path.join(test_data_dir, 'test_data.tmp')
    temp_downloaded_file = os.path.join(test_data_dir, 'test_data.tmp.downloaded')
    Path(temp_file).touch()
    gcs_path = os.path.join(GCS_BASE, 'test_data.tmp')

    # check
    try:
      GCSHelper.upload_gcs_file(temp_file, gcs_path)
      GCSHelper.download_gcs_blob(temp_downloaded_file, gcs_path)
      GCSHelper.remove_gcs_blob(gcs_path)
    except:
      self.fail('GCS helper failure')

    # clean up
    os.remove(temp_file)
    os.remove(temp_downloaded_file)

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
    docker_helper = DockerfileHelper(arc_dockerfile_name='', gcs_path='')
    self.assertTrue(docker_helper._wrap_files_in_tarball(temp_tarball, {'dockerfile':temp_file_one, 'main.py':temp_file_two}))
    self.assertTrue(os.path.exists(temp_tarball))
    temp_tarball_handler = tarfile.open(temp_tarball)
    temp_files = temp_tarball_handler.getmembers()
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
    golden_dockerfile_payload = '''\
FROM gcr.io/ngao-mlpipeline-testing/tensorflow:1.10.0
RUN apt-get update -y && apt-get install --no-install-recommends -y -q python3 python3-pip python3-setuptools
RUN pip3 install fire
ADD main.py /ml/
ENTRYPOINT ["python3", "/ml/main.py"]'''

    # check
    docker_helper = DockerfileHelper(arc_dockerfile_name=target_dockerfile, gcs_path='')
    docker_helper._generate_dockerfile_with_py(target_file=target_dockerfile, base_image='gcr.io/ngao-mlpipeline-testing/tensorflow:1.10.0', python_filepath='main.py')
    with open(target_dockerfile, 'r') as f:
      target_dockerfile_payload = f.read()
    self.assertEqual(target_dockerfile_payload, golden_dockerfile_payload)

    # clean up
    os.remove(target_dockerfile)

  def test_prepare_docker_with_py(self):
    """ Test the whole prepare docker from python function """

    # prepare
    test_data_dir = os.path.join(os.path.dirname(__file__), 'testdata')
    python_filepath = os.path.join(test_data_dir, 'basic.py')
    downloaded_tarball = os.path.join(test_data_dir, 'test_docker.tar.gz')
    gcs_tar_path = os.path.join(GCS_BASE, 'test_docker.tar.gz')

    # check
    docker_helper = DockerfileHelper(arc_dockerfile_name='dockerfile', gcs_path=gcs_tar_path)
    docker_helper.prepare_docker_tarball_with_py(arc_python_filename='main.py', python_filepath=python_filepath, base_image='gcr.io/ngao-mlpipeline-testing/tensorflow:1.8.0')
    GCSHelper.download_gcs_blob(downloaded_tarball, gcs_tar_path)
    temp_tarball_handler = tarfile.open(downloaded_tarball)
    temp_files = temp_tarball_handler.getmembers()
    self.assertTrue(len(temp_files) == 2)
    for temp_file in temp_files:
      self.assertTrue(temp_file.name in ['dockerfile', 'main.py'])

    # clean up
    os.remove(downloaded_tarball)
    GCSHelper.remove_gcs_blob(gcs_tar_path)

  def test_prepare_docker_tarball(self):
    """ Test the whole prepare docker tarball """

    # prepare
    test_data_dir = os.path.join(os.path.dirname(__file__), 'testdata')
    dockerfile_path = os.path.join(test_data_dir, 'component.target.dockerfile')
    Path(dockerfile_path).touch()
    downloaded_tarball = os.path.join(test_data_dir, 'test_docker.tar.gz')
    gcs_tar_path = os.path.join(GCS_BASE, 'test_docker.tar.gz')

    # check
    docker_helper = DockerfileHelper(arc_dockerfile_name='dockerfile', gcs_path=gcs_tar_path)
    docker_helper.prepare_docker_tarball(dockerfile_path=dockerfile_path)
    GCSHelper.download_gcs_blob(downloaded_tarball, gcs_tar_path)
    temp_tarball_handler = tarfile.open(downloaded_tarball)
    temp_files = temp_tarball_handler.getmembers()
    self.assertTrue(len(temp_files) == 1)
    for temp_file in temp_files:
      self.assertTrue(temp_file.name in ['dockerfile'])

    # clean up
    os.remove(downloaded_tarball)
    os.remove(dockerfile_path)
    GCSHelper.remove_gcs_blob(gcs_tar_path)

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

def wrapper_sample_component_func(a,b):
  output = sample_component_func(str(a),int(b))
  with open("/output.txt", "w") as f:
    f.write(str(output))

import fire
if __name__ == "__main__":
  fire.Fire(wrapper_sample_component_func)
'''
    self.assertEqual(golden, generated_codes)

    generated_codes = builder._generate_entrypoint(component_func=sample_component_func_two)
    golden = '''\
def sample_component_func_two(a: str, b: int) -> float:
  result = 3.45
  if a == 'succ':
    result = float(b + 5)
  return result

def wrapper_sample_component_func_two(a,b):
  output = sample_component_func_two(str(a),int(b))
  with open("/output.txt", "w") as f:
    f.write(str(output))

import fire
if __name__ == "__main__":
  fire.Fire(wrapper_sample_component_func_two)
'''
    self.assertEqual(golden, generated_codes)

    generated_codes = builder._generate_entrypoint(component_func=sample_component_func_three)
    golden = '''\
def sample_component_func_three() -> float:
  return 1.0

def wrapper_sample_component_func_three():
  output = sample_component_func_three()
  with open("/output.txt", "w") as f:
    f.write(str(output))

import fire
if __name__ == "__main__":
  fire.Fire(wrapper_sample_component_func_three)
'''
    self.assertEqual(golden, generated_codes)