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

import logging
import tarfile
import tempfile
import os
import uuid

class ContainerBuilder(object):
  """
  ContainerBuilder helps build a container image
  """
  def __init__(self, gcs_staging, namespace):
    """
    Args:
      gcs_staging (str): GCS blob that can store temporary build files
    """
    if not gcs_staging.startswith('gs://'):
      raise ValueError('Error: {} should be a GCS path.'.format(gcs_staging))
    self._gcs_staging = gcs_staging
    self._namespace = namespace

  def _generate_kaniko_spec(self, context, docker_filename, target_image):
    """_generate_kaniko_yaml generates kaniko job yaml based on a template yaml """
    content = {
        'apiVersion': 'v1',
        'metadata': {
            'generateName': 'kaniko-',
            'namespace': self._namespace,
        },
        'kind': 'Pod',
        'spec': {
            'restartPolicy': 'Never',
            'containers': [{
                'name': 'kaniko',
                'args': ['--cache=true',
                         '--dockerfile=' + docker_filename,
                         '--context=' + context,
                         '--destination=' + target_image],
                'image': 'gcr.io/kaniko-project/executor@sha256:78d44ec4e9cb5545d7f85c1924695c89503ded86a59f92c7ae658afa3cff5400',
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
    return content

  def _wrap_dir_in_tarball(self, tarball_path, dir_name):
    """ _wrap_files_in_tarball creates a tarball for all the files in the directory"""
    old_wd = os.getcwd()
    os.chdir(dir_name)
    if not tarball_path.endswith('.tar.gz'):
      raise ValueError('the tarball path should end with .tar.gz')
    with tarfile.open(tarball_path, 'w:gz') as tarball:
      for f in os.listdir(dir_name):
        tarball.add(f)
    os.chdir(old_wd)


  def build(self, local_dir, docker_filename, target_image, timeout):
    """
    Args:
      local_dir (str): local directory that stores all the necessary build files
      docker_filename (str): the dockerfile name that is in the local_dir
      target_image (str): the target image tag to push the final image.
      timeout (int): time out in seconds
    """
    # Prepare build context
    with tempfile.TemporaryDirectory() as local_build_dir:
      from ._gcs_helper import GCSHelper
      logging.info('Generate build files.')
      local_tarball_path = os.path.join(local_build_dir, 'docker.tmp.tar.gz')
      self._wrap_dir_in_tarball(local_tarball_path, local_dir)
      # Upload to the context
      context = os.path.join(self._gcs_staging, str(uuid.uuid4()) + '.tar.gz')
      GCSHelper.upload_gcs_file(local_tarball_path, context)

      # Run kaniko job
      kaniko_spec = self._generate_kaniko_spec(context=context,
                                               docker_filename=docker_filename,
                                               target_image=target_image)
      logging.info('Start a kaniko job for build.')
      from ._k8s_helper import K8sHelper
      k8s_helper = K8sHelper()
      k8s_helper.run_job(kaniko_spec, timeout)
      logging.info('Kaniko job complete.')

      # Clean up
      GCSHelper.remove_gcs_blob(context)
