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
import sys
import tempfile
import logging
from collections import OrderedDict
from pathlib import PurePath, Path
from ..components._components import _create_task_factory_from_component_spec

class GCSHelper(object):
  """ GCSHelper manages the connection with the GCS storage """

  @staticmethod
  def get_blob_from_gcs_uri(gcs_path):
    from google.cloud import storage
    pure_path = PurePath(gcs_path)
    gcs_bucket = pure_path.parts[1]
    gcs_blob = '/'.join(pure_path.parts[2:])
    client = storage.Client()
    bucket = client.get_bucket(gcs_bucket)
    blob = bucket.blob(gcs_blob)
    return blob

  @staticmethod
  def upload_gcs_file(local_path, gcs_path):
    blob = GCSHelper.get_blob_from_gcs_uri(gcs_path)
    blob.upload_from_filename(local_path)

  @staticmethod
  def remove_gcs_blob(gcs_path):
    blob = GCSHelper.get_blob_from_gcs_uri(gcs_path)
    blob.delete()

  @staticmethod
  def download_gcs_blob(local_path, gcs_path):
    blob = GCSHelper.get_blob_from_gcs_uri(gcs_path)
    blob.download_to_filename(local_path)

class VersionedDependency(object):
  """ DependencyVersion specifies the versions """
  def __init__(self, name, version=None, min_version=None, max_version=None):
    """ if version is specified, no need for min_version or max_version;
     if both are specified, version is adopted """
    self._name = name
    if version is not None:
      self._min_version = version
      self._max_version = version
    else:
      self._min_version = min_version
      self._max_version = max_version

  @property
  def name(self):
    return self._name

  @property
  def min_version(self):
    return self._min_version

  @min_version.setter
  def min_version(self, min_version):
    self._min_version = min_version

  def has_min_version(self):
    return self._min_version != None

  @property
  def max_version(self):
    return self._max_version

  @max_version.setter
  def max_version(self, max_version):
    self._max_version = max_version

  def has_max_version(self):
    return self._max_version != None

  def has_versions(self):
    return (self.has_min_version()) or (self.has_max_version())


class DependencyHelper(object):
  """ DependencyHelper manages software dependency information """
  def __init__(self):
    self._PYTHON_PACKAGE = 'PYTHON_PACKAGE'
    self._dependency = {self._PYTHON_PACKAGE:OrderedDict()}

  @property
  def python_packages(self):
    return self._dependency[self._PYTHON_PACKAGE]

  def add_python_package(self, dependency, override=True):
    """ add_single_python_package adds a dependency for the python package

    Args:
      name: package name
      version: it could be a specific version(1.10.0), or a range(>=1.0,<=2.0)
        if not specified, the default is resolved automatically by the pip system.
      override: whether to override the version if already existing in the dependency.
    """
    if dependency.name in self.python_packages and not override:
      return
    self.python_packages[dependency.name] = dependency

  def generate_pip_requirements(self, target_file):
    """ write the python packages to a requirement file
    the generated file follows the order of which the packages are added """
    with open(target_file, 'w') as f:
      for name, version in self.python_packages.items():
        version_str = ''
        if version.has_min_version():
          version_str += ' >= ' + version.min_version + ','
        if version.has_max_version():
          version_str += ' <= ' + version.max_version + ','
        f.write(name + version_str.rstrip(',') + '\n')

class DockerfileHelper(object):
  """ Dockerfile Helper generates a tarball with dockerfile, ready for docker build
      arc_dockerfile_name: dockerfile filename that is stored in the tarball """

  def __init__(self, arc_dockerfile_name):
    self._arc_dockerfile_name = arc_dockerfile_name
    self._ARC_REQUIREMENT_FILE = 'requirements.txt'

  def _generate_pip_requirement(self, dependency, requirement_filepath):
    dependency_helper = DependencyHelper()
    for version in dependency:
      dependency_helper.add_python_package(version)
    dependency_helper.generate_pip_requirements(requirement_filepath)

  def _generate_dockerfile_with_py(self, target_file, base_image, python_filepath, has_requirement_file, python_version):
    """ _generate_docker_file generates a simple dockerfile with the python path
    args:
      target_file (str): target file name for the dockerfile.
      base_image (str): the base image name.
      python_filepath (str): the path of the python file that is copied to the docker image.
      has_requirement_file (bool): whether it has a requirement file or not.
      python_version (str): choose python2 or python3
    """
    if python_version not in ['python2', 'python3']:
      raise ValueError('python_version has to be either python2 or python3')
    with open(target_file, 'w') as f:
      f.write('FROM ' + base_image + '\n')
      if python_version is 'python3':
        f.write('RUN apt-get update -y && apt-get install --no-install-recommends -y -q python3 python3-pip python3-setuptools\n')
      else:
        f.write('RUN apt-get update -y && apt-get install --no-install-recommends -y -q python python-pip python-setuptools\n')
      if has_requirement_file:
        f.write('ADD ' + self._ARC_REQUIREMENT_FILE + ' /ml/\n')
        if python_version is 'python3':
          f.write('RUN pip3 install -r /ml/' + self._ARC_REQUIREMENT_FILE + '\n')
        else:
          f.write('RUN pip install -r /ml/' + self._ARC_REQUIREMENT_FILE + '\n')
      f.write('ADD ' + python_filepath + " /ml/" + '\n')
      if python_version is 'python3':
        f.write('ENTRYPOINT ["python3", "/ml/' + python_filepath + '"]')
      else:
        f.write('ENTRYPOINT ["python", "/ml/' + python_filepath + '"]')

  def _wrap_files_in_tarball(self, tarball_path, files={}):
    """ _wrap_files_in_tarball creates a tarball for all the input files
    with the filename configured as the key of files """
    if not tarball_path.endswith('.tar.gz'):
      raise ValueError('the tarball path should end with .tar.gz')
    with tarfile.open(tarball_path, 'w:gz') as tarball:
      for key, value in files.items():
        tarball.add(value, arcname=key)

  def prepare_docker_tarball_with_py(self, arc_python_filename, python_filepath, base_image, local_tarball_path, python_version, dependency=None):
    """ prepare_docker_tarball is the API to generate dockerfile and prepare the tarball with python scripts
    args:
      python_version (str): choose python2 or python3
    """
    if python_version not in ['python2', 'python3']:
      raise ValueError('python_version has to be either python2 or python3')
    with tempfile.TemporaryDirectory() as local_build_dir:
      has_requirement_file = False
      local_requirement_path = os.path.join(local_build_dir, self._ARC_REQUIREMENT_FILE)
      if dependency is not None and len(dependency) != 0:
        self._generate_pip_requirement(dependency, local_requirement_path)
        has_requirement_file = True
      local_dockerfile_path = os.path.join(local_build_dir, self._arc_dockerfile_name)
      self._generate_dockerfile_with_py(local_dockerfile_path, base_image, arc_python_filename, has_requirement_file, python_version)
      file_lists =  {self._arc_dockerfile_name:local_dockerfile_path,
                     arc_python_filename:python_filepath}
      if has_requirement_file:
        file_lists[self._ARC_REQUIREMENT_FILE] = local_requirement_path
      self._wrap_files_in_tarball(local_tarball_path, file_lists)

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
    content = {
      'apiVersion': 'v1',
      'metadata': {
        'generateName': 'kaniko-',
        'namespace': 'kubeflow',
      },
      'kind': 'Pod',
      'spec': {
        'restartPolicy': 'Never',
        'containers': [{
          'name': 'kaniko',
          'args': ['--cache=true'],
          'image': 'gcr.io/kaniko-project/executor:v0.5.0',
          'env': [{
            'name': 'GOOGLE_APPLICATION_CREDENTIALS',
            'value': '/secret/gcp-credentials/user-gcp-sa.json'
          }],
          'volumeMounts': [{
            'mountPath': '/secret/gcp-credentials',
            'name': 'gcp-credentials',
          }],
        }],
        'volumes': [{
          'name': 'gcp-credentials',
          'secret': {
            'secretName': 'user-gcp-sa',
          },
        }],
        'serviceAccountName': 'default'}
    }

    content['metadata']['namespace'] = namespace
    args = content['spec']['containers'][0]['args']
    args.append('--dockerfile=' + arc_dockerfile_name)
    args.append('--context=' + gcs_path)
    args.append('--destination=' + target_image)
    return content

  #TODO: currently it supports single output, future support for multiple return values
  def _generate_entrypoint(self, component_func, python_version='python3'):
    '''
    args:
      python_version (str): choose python2 or python3, default is python3
    '''
    if python_version not in ['python2', 'python3']:
      raise ValueError('python_version has to be either python2 or python3')

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
    match = re.search(r'\n([ \t]+)[\w]+', component_src)
    indentation = match.group(1) if match else '\t'
    codegen = CodeGenerator(indentation=indentation)

    # Function signature
    new_func_name = 'wrapper_' + component_func.__name__
    codegen.begin()
    func_signature = 'def ' + new_func_name + '('
    for input_arg in input_args:
      func_signature += input_arg + ','
    func_signature += '_output_file):'
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
    codegen.writeline('import os')
    codegen.writeline('os.makedirs(os.path.dirname(_output_file))')
    codegen.writeline('with open(_output_file, "w") as data:')
    codegen.indent()
    codegen.writeline('data.write(str(output))')
    wrapper_code = codegen.end()

    # CLI codes
    codegen.begin()
    codegen.writeline('import argparse')
    codegen.writeline('parser = argparse.ArgumentParser(description="Parsing arguments")')
    for input_arg in input_args:
      codegen.writeline('parser.add_argument("' + input_arg + '", type=' + inputs[input_arg].__name__ + ')')
    codegen.writeline('parser.add_argument("_output_file", type=str)')
    codegen.writeline('args = vars(parser.parse_args())')
    codegen.writeline('')
    codegen.writeline('if __name__ == "__main__":')
    codegen.indent()
    codegen.writeline(new_func_name + '(**args)')

    # Remove the decorator from the component source
    src_lines = component_src.split('\n')
    start_line_num = 0
    for line in src_lines:
      if line.startswith('def '):
        break
      start_line_num += 1
    if python_version == 'python2':
      src_lines[start_line_num] = 'def ' + component_func.__name__ + '(' + ', '.join((inspect.getfullargspec(component_func).args)) + '):'
    dedecorated_component_src = '\n'.join(src_lines[start_line_num:])

    complete_component_code = dedecorated_component_src + '\n' + wrapper_code + '\n' + codegen.end()
    return complete_component_code

  def _build_image_from_tarball(self, local_tarball_path, namespace, timeout):
    GCSHelper.upload_gcs_file(local_tarball_path, self._gcs_path)
    kaniko_spec = self._generate_kaniko_spec(namespace=namespace,
                                             arc_dockerfile_name=self._arc_dockerfile_name,
                                             gcs_path=self._gcs_path,
                                             target_image=self._target_image)
    # Run kaniko job
    logging.info('Start a kaniko job for build.')
    from ._k8s_helper import K8sHelper
    k8s_helper = K8sHelper()
    k8s_helper.run_job(kaniko_spec, timeout)
    logging.info('Kaniko job complete.')

    # Clean up
    GCSHelper.remove_gcs_blob(self._gcs_path)

  def build_image_from_func(self, component_func, namespace, base_image, timeout, dependency, python_version='python3'):
    """ build_image builds an image for the given python function
    args:
      python_version (str): choose python2 or python3, default is python3
    """
    if python_version not in ['python2', 'python3']:
      raise ValueError('python_version has to be either python2 or python3')
    with tempfile.TemporaryDirectory() as local_build_dir:
      # Generate entrypoint and serialization python codes
      local_python_filepath = os.path.join(local_build_dir, self._arc_python_filepath)
      logging.info('Generate entrypoint and serialization codes.')
      complete_component_code = self._generate_entrypoint(component_func, python_version)
      with open(local_python_filepath, 'w') as f:
        f.write(complete_component_code)

      # Prepare build files
      logging.info('Generate build files.')
      local_tarball_path = os.path.join(local_build_dir, 'docker.tmp.tar.gz')
      docker_helper = DockerfileHelper(arc_dockerfile_name=self._arc_dockerfile_name)
      docker_helper.prepare_docker_tarball_with_py(python_filepath=local_python_filepath,
                                                   arc_python_filename=self._arc_python_filepath,
                                                   base_image=base_image,
                                                   local_tarball_path=local_tarball_path,
                                                   python_version=python_version,
                                                   dependency=dependency)
      self._build_image_from_tarball(local_tarball_path, namespace, timeout)

  def build_image_from_dockerfile(self, dockerfile_path, timeout, namespace):
    """ build_image_from_dockerfile builds an image based on the dockerfile """
    with tempfile.TemporaryDirectory() as local_build_dir:
      # Prepare build files
      logging.info('Generate build files.')
      local_tarball_path = os.path.join(local_build_dir, 'docker.tmp.tar.gz')
      docker_helper = DockerfileHelper(arc_dockerfile_name=self._arc_dockerfile_name)
      docker_helper.prepare_docker_tarball(dockerfile_path, local_tarball_path=local_tarball_path)
      self._build_image_from_tarball(local_tarball_path, namespace, timeout)

def _configure_logger(logger):
  """ _configure_logger configures the logger such that the info level logs
  go to the stdout and the error(or above) level logs go to the stderr.
  It is important for the Jupyter notebook log rendering """
  if hasattr(_configure_logger, 'configured'):
    # Skip the logger configuration the second time this function
    # is called to avoid multiple streamhandlers bound to the logger.
    return
  setattr(_configure_logger, 'configured', 'true')
  logger.setLevel(logging.INFO)
  info_handler = logging.StreamHandler(stream=sys.stdout)
  info_handler.addFilter(lambda record: record.levelno <= logging.INFO)
  info_handler.setFormatter(logging.Formatter('%(asctime)s:%(levelname)s:%(message)s', datefmt='%Y-%m-%d %H:%M:%S'))
  error_handler = logging.StreamHandler(sys.stderr)
  error_handler.addFilter(lambda record: record.levelno > logging.INFO)
  error_handler.setFormatter(logging.Formatter('%(asctime)s:%(levelname)s:%(message)s', datefmt='%Y-%m-%d %H:%M:%S'))
  logger.addHandler(info_handler)
  logger.addHandler(error_handler)

def _generate_pythonop(component_func, target_image, target_component_file=None):
  """ Generate operator for the pipeline authors
  The returned value is in fact a function, which should generates a container_op instance. """

  from ..components._python_op import _python_function_name_to_component_name
  from ..components._structures import InputSpec, InputValuePlaceholder, OutputPathPlaceholder, OutputSpec, ContainerImplementation, ContainerSpec, ComponentSpec


  #Component name and description are derived from the function's name and docstribng, but can be overridden by @python_component function decorator
  #The decorator can set the _component_human_name and _component_description attributes. getattr is needed to prevent error when these attributes do not exist.
  component_name = getattr(component_func, '_component_human_name', None) or _python_function_name_to_component_name(component_func.__name__)
  component_description = getattr(component_func, '_component_description', None) or (component_func.__doc__.strip() if component_func.__doc__ else None)

  #TODO: Humanize the input/output names
  input_names = inspect.getfullargspec(component_func)[0]

  output_name = 'output'
  component_spec = ComponentSpec(
      name=component_name,
      description=component_description,
      inputs=[InputSpec(name=input_name, type='str') for input_name in input_names], #TODO: Chnage type to actual type
      outputs=[OutputSpec(name=output_name)],
      implementation=ContainerImplementation(
          container=ContainerSpec(
              image=target_image,
              #command=['python3', program_file], #TODO: Include the command line
              args=[InputValuePlaceholder(input_name) for input_name in input_names] + [OutputPathPlaceholder(output_name)],
          )
      )
  )
  
  target_component_file = target_component_file or getattr(component_func, '_component_target_component_file', None)
  if target_component_file:
    from ..components._yaml_utils import dump_yaml
    component_text = dump_yaml(component_spec.to_struct())
    Path(target_component_file).write_text(component_text)

  return _create_task_factory_from_component_spec(component_spec)

def build_python_component(component_func, target_image, base_image=None, dependency=[], staging_gcs_path=None, build_image=True, timeout=600, namespace='kubeflow', target_component_file=None, python_version='python3'):
  """ build_component automatically builds a container image for the component_func
  based on the base_image and pushes to the target_image.

  Args:
    component_func (python function): The python function to build components upon
    base_image (str): Docker image to use as a base image
    target_image (str): Full URI to push the target image
    staging_gcs_path (str): GCS blob that can store temporary build files
    target_image (str): target image path
    build_image (bool): whether to build the image or not. Default is True.
    timeout (int): the timeout for the image build(in secs), default is 600 seconds
    namespace (str): the namespace within which to run the kubernetes kaniko job, default is "kubeflow"
    dependency (list): a list of VersionedDependency, which includes the package name and versions, default is empty
    python_version (str): choose python2 or python3, default is python3
  Raises:
    ValueError: The function is not decorated with python_component decorator or the python_version is neither python2 nor python3
  """

  _configure_logger(logging.getLogger())

  if component_func is None:
    raise ValueError('component_func must not be None')
  if target_image is None:
    raise ValueError('target_image must not be None')

  if python_version not in ['python2', 'python3']:
    raise ValueError('python_version has to be either python2 or python3')

  if build_image:
    if staging_gcs_path is None:
      raise ValueError('staging_gcs_path must not be None')

    if base_image is None:
      base_image = getattr(component_func, '_component_base_image', None)
    if base_image is None:
      raise ValueError('base_image must not be None')

    logging.info('Build an image that is based on ' +
                                   base_image +
                                   ' and push the image to ' +
                                   target_image)
    builder = ImageBuilder(gcs_base=staging_gcs_path, target_image=target_image)
    builder.build_image_from_func(component_func, namespace=namespace,
                                  base_image=base_image, timeout=timeout,
                                  python_version=python_version, dependency=dependency)
    logging.info('Build component complete.')
  return _generate_pythonop(component_func, target_image, target_component_file)

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
  _configure_logger(logging.getLogger())
  builder = ImageBuilder(gcs_base=staging_gcs_path, target_image=target_image)
  builder.build_image_from_dockerfile(dockerfile_path=dockerfile_path, timeout=timeout, namespace=namespace)
  logging.info('Build image complete.')
