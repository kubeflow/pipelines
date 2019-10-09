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

from kfp.containers._component_builder import _generate_dockerfile, _dependency_to_requirements, VersionedDependency, DependencyHelper

import os
import unittest
import yaml
import tarfile
from pathlib import Path
import inspect
from collections import OrderedDict
from typing import NamedTuple

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
'''
    golden_dockerfile_payload_two = '''\
FROM gcr.io/ngao-mlpipeline-testing/tensorflow:1.10.0
RUN apt-get update -y && apt-get install --no-install-recommends -y -q python3 python3-pip python3-setuptools
ADD requirements.txt /ml/requirements.txt
RUN python3 -m pip install -r /ml/requirements.txt
ADD main.py /ml/main.py
'''

    golden_dockerfile_payload_three = '''\
FROM gcr.io/ngao-mlpipeline-testing/tensorflow:1.10.0
RUN apt-get update -y && apt-get install --no-install-recommends -y -q python python-pip python-setuptools
ADD requirements.txt /ml/requirements.txt
RUN python -m pip install -r /ml/requirements.txt
ADD main.py /ml/main.py
'''
    # check
    _generate_dockerfile(filename=target_dockerfile, base_image='gcr.io/ngao-mlpipeline-testing/tensorflow:1.10.0',
                         python_version='python3', add_files={'main.py': '/ml/main.py'})
    with open(target_dockerfile, 'r') as f:
      target_dockerfile_payload = f.read()
    self.assertEqual(target_dockerfile_payload, golden_dockerfile_payload_one)
    _generate_dockerfile(filename=target_dockerfile, base_image='gcr.io/ngao-mlpipeline-testing/tensorflow:1.10.0',
                         python_version='python3', requirement_filename='requirements.txt', add_files={'main.py': '/ml/main.py'})
    with open(target_dockerfile, 'r') as f:
      target_dockerfile_payload = f.read()
    self.assertEqual(target_dockerfile_payload, golden_dockerfile_payload_two)
    _generate_dockerfile(filename=target_dockerfile, base_image='gcr.io/ngao-mlpipeline-testing/tensorflow:1.10.0',
                         python_version='python2', requirement_filename='requirements.txt', add_files={'main.py': '/ml/main.py'})
    with open(target_dockerfile, 'r') as f:
      target_dockerfile_payload = f.read()
    self.assertEqual(target_dockerfile_payload, golden_dockerfile_payload_three)

    self.assertRaises(ValueError, _generate_dockerfile, filename=target_dockerfile,
                      base_image='gcr.io/ngao-mlpipeline-testing/tensorflow:1.10.0',
                      python_version='python4', requirement_filename='requirements.txt', add_files={'main.py': '/ml/main.py'})

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
