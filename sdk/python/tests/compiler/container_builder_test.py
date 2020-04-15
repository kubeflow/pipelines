# Copyright 2019 Google LLC
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

import os
import tarfile
import unittest
import yaml
import tempfile
import mock
from kfp.containers._component_builder import ContainerBuilder

GCS_BASE = 'gs://kfp-testing/'
DEFAULT_IMAGE_NAME = 'gcr.io/kfp-testing/image'

@mock.patch('kfp.containers._gcs_helper.GCSHelper')
class TestContainerBuild(unittest.TestCase):

  def test_wrap_dir_in_tarball(self, mock_gcshelper):
    """ Test wrap files in a tarball """

    # prepare
    temp_tarball = os.path.join(os.path.dirname(__file__), 'test_data.tmp.tar.gz')
    with tempfile.TemporaryDirectory() as test_data_dir:
      temp_file_one = os.path.join(test_data_dir, 'test_data_one.tmp')
      temp_file_two = os.path.join(test_data_dir, 'test_data_two.tmp')
      with open(temp_file_one, 'w') as f:
        f.write('temporary file one content')
      with open(temp_file_two, 'w') as f:
        f.write('temporary file two content')

      # check
      builder = ContainerBuilder(gcs_staging=GCS_BASE, default_image_name=DEFAULT_IMAGE_NAME, namespace='')
      builder._wrap_dir_in_tarball(temp_tarball, test_data_dir)
    self.assertTrue(os.path.exists(temp_tarball))
    with tarfile.open(temp_tarball) as temp_tarball_handle:
      temp_files = temp_tarball_handle.getmembers()
      for temp_file in temp_files:
        self.assertTrue(temp_file.name in ['test_data_one.tmp', 'test_data_two.tmp', ''])

    # clean up
    os.remove(temp_tarball)

  def test_generate_kaniko_yaml(self, mock_gcshelper):
    """ Test generating the kaniko job yaml """

    # prepare
    test_data_dir = os.path.join(os.path.dirname(__file__), 'testdata')

    # check
    builder = ContainerBuilder(gcs_staging=GCS_BASE,
                               default_image_name=DEFAULT_IMAGE_NAME,
                               namespace='default')
    generated_yaml = builder._generate_kaniko_spec(docker_filename='dockerfile',
                                                   context='gs://mlpipeline/kaniko_build.tar.gz',
                                                   target_image='gcr.io/mlpipeline/kaniko_image:latest')
    with open(os.path.join(test_data_dir, 'kaniko.basic.yaml'), 'r') as f:
      golden = yaml.safe_load(f)

    self.assertEqual(golden, generated_yaml)

  def test_generate_kaniko_yaml_kubeflow(self, mock_gcshelper):
    """ Test generating the kaniko job yaml for Kubeflow deployment """

    # prepare
    test_data_dir = os.path.join(os.path.dirname(__file__), 'testdata')

    # check
    builder = ContainerBuilder(gcs_staging=GCS_BASE,
                               default_image_name=DEFAULT_IMAGE_NAME,
                               namespace='user',
                               service_account='default-editor',)
    generated_yaml = builder._generate_kaniko_spec(docker_filename='dockerfile',
                                                   context='gs://mlpipeline/kaniko_build.tar.gz',
                                                   target_image='gcr.io/mlpipeline/kaniko_image:latest',)
    with open(os.path.join(test_data_dir, 'kaniko.kubeflow.yaml'), 'r') as f:
      golden = yaml.safe_load(f)

    self.assertEqual(golden, generated_yaml)
