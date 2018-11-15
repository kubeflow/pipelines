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

import tarfile
import uuid
import os
import inspect
import re
import tempfile
import logging
from google.cloud import storage
from pathlib import PurePath, Path
from .. import dsl
from ..components._components import _create_task_factory_from_component_dict
from ._k8s_helper import K8sHelper

class GCSHelper(object):
  """ GCSHelper manages the connection with the GCS storage """

  @staticmethod
  def upload_gcs_file(local_path, gcs_path):
    pure_path = PurePath(gcs_path)
    gcs_bucket = pure_path.parts[1]
    gcs_blob = '/'.join(pure_path.parts[2:])
    client = storage.Client()
    bucket = client.get_bucket(gcs_bucket)
    blob = bucket.blob(gcs_blob)
    blob.upload_from_filename(local_path)

  @staticmethod
  def remove_gcs_blob(gcs_path):
    pure_path = PurePath(gcs_path)
    gcs_bucket = pure_path.parts[1]
    gcs_blob = '/'.join(pure_path.parts[2:])
    client = storage.Client()
    bucket = client.get_bucket(gcs_bucket)
    blob = bucket.blob(gcs_blob)
    blob.delete()

  @staticmethod
  def download_gcs_blob(local_path, gcs_path):
    pure_path = PurePath(gcs_path)
    gcs_bucket = pure_path.parts[1]
    gcs_blob = '/'.join(pure_path.parts[2:])
    client = storage.Client()
    bucket = client.get_bucket(gcs_bucket)
    blob = bucket.blob(gcs_blob)
    blob.download_to_filename(local_path)


class DockerfileHelper(object):
  """ Dockerfile Helper generates a tarball with dockerfile, ready for docker build
      arc_dockerfile_name: dockerfile filename that is stored in the tarball """

  def __init__(self, arc_dockerfile_name):
    self._arc_dockerfile_name = arc_dockerfile_name

  def _generate_dockerfile_with_py(self, target_file, base_image, python_filepath):
    """ _generate_docker_file generates a simple dockerfile with the python path """
    with open(target_file, 'w') as f:
      f.write('FROM ' + base_image + '\n')
      f.write('RUN apt-get update -y && apt-get install --no-install-recommends -y -q python3 python3-pip python3-setuptools\n')
      f.write('RUN pip3 install fire\n')
      f.write('ADD ' + python_filepath + " /ml/" + '\n')
      f.write('ENTRYPOINT ["python3", "/ml/' + python_filepath + '"]')

  def _wrap_files_in_tarball(self, tarball_path, files={}):
    """ _wrap_files_in_tarball creates a tarball for all the input files
    with the filename configured as the key of files """
    if not tarball_path.endswith('.tar.gz'):
      raise ValueError('the tarball path should end with .tar.gz')
    with tarfile.open(tarball_path, 'w:gz') as tarball:
      for key, value in files.items():
        tarball.add(value, arcname=key)

  def prepare_docker_tarball_with_py(self, arc_python_filename, python_filepath, base_image, local_tarball_path):
    """ prepare_docker_tarball is the API to generate dockerfile and prepare the tarball with python scripts """
    with tempfile.TemporaryDirectory() as local_build_dir:
      local_dockerfile_path = os.path.join(local_build_dir, self._arc_dockerfile_name)
      self._generate_dockerfile_with_py(local_dockerfile_path, base_image, arc_python_filename)
      self._wrap_files_in_tarball(local_tarball_path, {self._arc_dockerfile_name:local_dockerfile_path,
                                                       arc_python_filename:python_filepath})

  def prepare_docker_tarball(self, dockerfile_path, local_tarball_path):
    """ prepare_docker_tarball is the API to prepare a tarball with the dockerfile """
    self._wrap_files_in_tarball(local_tarball_path, {self._arc_dockerfile_name:dockerfile_path})

class CodeGenerator(object):
  """ CodeGenerator helps to generate python codes with identation """
  def __init__(self, indentation='\t'):
    self._indentation = indentation
    self._code = []
    self._level = 0

  def begin(self):
    self._code = []
    self._level = 0

  def indent(self):
    self._level += 1

  def dedent(self):
    if self._level == 0:
      raise Exception('CodeGenerator dedent error')
    self._level -= 1

  def writeline(self, line):
    self._code.append(self._indentation * self._level + line)

  def end(self):
    line_sep = '\n'
    return line_sep.join(self._code) + line_sep

class ImageBuilder(object):
  """ Component Builder. """
  def __init__(self, gcs_base, target_image):
    self._arc_dockerfile_name = 'dockerfile'
    self._arc_python_filepath = 'main.py'
    self._tarball_name = str(uuid.uuid4()) + '.tar.gz'
    self._gcs_base = gcs_base
    if not self._check_gcs_path(self._gcs_base):
      raise Exception('ImageBuild __init__ failure.')
    self._gcs_path = os.path.join(self._gcs_base, self._tarball_name)
    self._target_image = target_image

  def _check_gcs_path(self, gcs_path):
    """ _check_gcs_path check both the path validity and write permissions """
    logging.info('Checking path: {}...'.format(gcs_path))
    if not gcs_path.startswith('gs://'):
      logging.error('Error: {} should be a GCS path.'.format(gcs_path))
      return False
    return True

  def _generate_kaniko_spec(self, namespace, arc_dockerfile_name, gcs_path, target_image):
    """_generate_kaniko_yaml generates kaniko job yaml based on a template yaml """
    content = {'apiVersion': 'v1',
               'metadata': {
                 'generateName': 'kaniko-',
                 'namespace': 'default'},
               'kind': 'Pod',
               'spec': {
                 'restartPolicy': 'Never',
                 'containers': [
                   {'name': 'kaniko',
                    'args': ['--cache=true'],
                    'image': 'gcr.io/kaniko-project/executor:v0.5.0'}],
                 'serviceAccountName': 'default'}}

    content['metadata']['namespace'] = namespace
    args = content['spec']['containers'][0]['args']
    args.append('--dockerfile=' + arc_dockerfile_name)
    args.append('--context=' + gcs_path)
    args.append('--destination=' + target_image)
    return content

  #TODO: currently it supports single output, future support for multiple return values
  def _generate_entrypoint(self, component_func):
    fullargspec = inspect.getfullargspec(component_func)
    annotations = fullargspec[6]
    input_args = fullargspec[0]
    inputs = {}
    for key, value in annotations.items():
      if key != 'return':
        inputs[key] = value
    if len(input_args) != len(inputs):
      raise Exception('Some input arguments do not contain annotations.')
    if 'return' in  annotations and annotations['return'] not in [int, float, str, bool]:
      raise Exception('Output type not supported and supported types are [int, float, str, bool]')
    # inputs is a dictionary with key of argument name and value of type class
    # output is a type class, e.g. str and int.

    # Follow the same indentation with the component source codes.
    component_src = inspect.getsource(component_func)
    match = re.search('\n([ \t]+)[\w]+', component_src)
    indentation = match.group(1) if match else '\t'
    codegen = CodeGenerator(indentation=indentation)

    # Function signature
    new_func_name = 'wrapper_' + component_func.__name__
    codegen.begin()
    func_signature = 'def ' + new_func_name + '('
    for input_arg in input_args:
      func_signature += input_arg + ','
    if len(input_args) > 0:
      func_signature = func_signature[:-1]
    func_signature += '):'
    codegen.writeline(func_signature)

    # Call user function
    codegen.indent()
    call_component_func = 'output = ' + component_func.__name__ + '('
    for input_arg in input_args:
      call_component_func += inputs[input_arg].__name__ + '(' + input_arg + '),'
    call_component_func = call_component_func.rstrip(',')
    call_component_func += ')'
    codegen.writeline(call_component_func)

    # Serialize output
    codegen.writeline('with open("/output.txt", "w") as f:')
    codegen.indent()
    codegen.writeline('f.write(str(output))')
    wrapper_code = codegen.end()

    # CLI codes
    codegen.begin()
    codegen.writeline('import fire')
    codegen.writeline('if __name__ == "__main__":')
    codegen.indent()
    codegen.writeline('fire.Fire(' + new_func_name + ')')

    # Remove the decorator from the component source
    src_lines = component_src.split('\n')
    start_line_num = 0
    for line in src_lines:
      if line.startswith('def '):
        break
      start_line_num += 1
    dedecorated_component_src = '\n'.join(src_lines[start_line_num:])

    complete_component_code = dedecorated_component_src + '\n' + wrapper_code + '\n' + codegen.end()
    return complete_component_code

  def build_image_from_func(self, component_func, namespace, base_image, timeout):
    """ build_image builds an image for the given python function"""

    # Generate entrypoint and serialization python codes
    with tempfile.TemporaryDirectory() as local_build_dir:
      local_python_filepath = os.path.join(local_build_dir, self._arc_python_filepath)
      logging.info('Generate entrypoint and serialization codes.')
      complete_component_code = self._generate_entrypoint(component_func)
      with open(local_python_filepath, 'w') as f:
        f.write(complete_component_code)

      # Prepare build files
      logging.info('Generate build files.')
      docker_helper = DockerfileHelper(arc_dockerfile_name=self._arc_dockerfile_name)
      local_tarball_file = os.path.join(local_build_dir, 'docker.tmp.tar.gz')
      docker_helper.prepare_docker_tarball_with_py(python_filepath=local_python_filepath,
                                                   arc_python_filename=self._arc_python_filepath,
                                                   base_image=base_image, local_tarball_path=local_tarball_file)
      GCSHelper.upload_gcs_file(local_tarball_file, self._gcs_path)

      kaniko_spec = self._generate_kaniko_spec(namespace=namespace,
                                               arc_dockerfile_name=self._arc_dockerfile_name,
                                               gcs_path=self._gcs_path,
                                               target_image=self._target_image)
      # Run kaniko job
      logging.info('Start a kaniko job for build.')
      k8s_helper = K8sHelper()
      k8s_helper.run_job(kaniko_spec, timeout)
      logging.info('Kaniko job complete.')

      # Clean up
      GCSHelper.remove_gcs_blob(self._gcs_path)

  def build_image_from_dockerfile(self, dockerfile_path, timeout, namespace):
    """ build_image_from_dockerfile builds an image directly """
    with tempfile.TemporaryDirectory() as local_build_dir:
      logging.info('Generate build files.')
      docker_helper = DockerfileHelper(arc_dockerfile_name=self._arc_dockerfile_name)
      local_tarball_file = os.path.join(local_build_dir, 'docker.tmp.tar.gz')
      docker_helper.prepare_docker_tarball(dockerfile_path, local_tarball_path=local_tarball_file)
      GCSHelper.upload_gcs_file(local_tarball_file, self._gcs_path)

      kaniko_spec = self._generate_kaniko_spec(namespace=namespace, arc_dockerfile_name=self._arc_dockerfile_name,
                                             gcs_path=self._gcs_path, target_image=self._target_image)
      logging.info('Start a kaniko job for build.')
      k8s_helper = K8sHelper()
      k8s_helper.run_job(kaniko_spec, timeout)
      logging.info('Kaniko job complete.')

      # Clean up
      GCSHelper.remove_gcs_blob(self._gcs_path)

def _generate_pythonop(component_func, target_image):
  """ Generate operator for the pipeline authors
  component_meta is a dict of name, description, base_image, target_image, input_list
  The returned value is in fact a function, which should generates a container_op instance. """
  component_meta = dsl.PythonComponent.get_python_component(component_func)
  input_names = inspect.getfullargspec(component_func)[0]

  component_artifact = {}
  component_artifact['name'] = component_meta['name']
  component_artifact['description'] = component_meta['description']
  component_artifact['outputs'] = [{'name': 'output'}]
  component_artifact['inputs'] = []
  component_artifact['implementation'] = {
    'dockerContainer': {
      'image': target_image,
      'arguments': [],
      'fileOutputs': {
        'output': '/output.txt'
      }
    }
  }
  for input in input_names:
    component_artifact['inputs'].append({
      'name': input,
      'type': 'str'
    })
    component_artifact['implementation']['dockerContainer']['arguments'].append({'value': input})
  return _create_task_factory_from_component_dict(component_artifact)

def build_python_component(component_func, staging_gcs_path, target_image, build_image=True, timeout=600, namespace='kubeflow'):
  """ build_component automatically builds a container image for the component_func
  based on the base_image and pushes to the target_image.

  Args:
    component_func (python function): The python function to build components upon
    staging_gcs_path (str): GCS blob that can store temporary build files
    timeout (int): the timeout for the image build(in secs), default is 600 seconds
    namespace (str): the namespace within which to run the kubernetes kaniko job, default is "kubeflow"
    build_image (bool): whether to build the image or not. Default is True.

  Raises:
    ValueError: The function is not decorated with python_component decorator
  """
  logging.basicConfig()
  logging.getLogger().setLevel('INFO')
  component_meta = dsl.PythonComponent.get_python_component(component_func)
  component_meta['inputs'] = inspect.getfullargspec(component_func)[0]

  if component_meta is None:
    raise ValueError('The function "%s" does not exist. '
                     'Did you forget @dsl.python_component decoration?' % component_func)
  logging.info('Build an image that is based on ' +
                                   component_meta['base_image'] +
                                   ' and push the image to ' +
                                   target_image)
  if build_image:
    builder = ImageBuilder(gcs_base=staging_gcs_path, target_image=target_image)
    builder.build_image_from_func(component_func, namespace=namespace,
                                  base_image=component_meta['base_image'], timeout=timeout)
    logging.info('Build component complete.')
  return _generate_pythonop(component_func, target_image)

def build_docker_image(staging_gcs_path, target_image, dockerfile_path, timeout=600, namespace='kubeflow'):
  """ build_docker_image automatically builds a container image based on the specification in the dockerfile and
  pushes to the target_image.

  Args:
    staging_gcs_path (str): GCS blob that can store temporary build files
    target_image (str): gcr path to push the final image
    dockerfile_path (str): local path to the dockerfile
    timeout (int): the timeout for the image build(in secs), default is 600 seconds
    namespace (str): the namespace within which to run the kubernetes kaniko job, default is "kubeflow"
  """
  logging.basicConfig()
  logging.getLogger().setLevel('INFO')
  builder = ImageBuilder(gcs_base=staging_gcs_path, target_image=target_image)
  builder.build_image_from_dockerfile(dockerfile_path=dockerfile_path, timeout=timeout, namespace=namespace)
  logging.info('Build image complete.')
