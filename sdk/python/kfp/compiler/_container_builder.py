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

SERVICEACCOUNT_NAMESPACE = '/var/run/secrets/kubernetes.io/serviceaccount/namespace'
GCS_STAGING_BLOB_DEFAULT_PREFIX = 'kfp_container_build_staging'
GCR_DEFAULT_IMAGE_SUFFIX = 'kfp_container'

class ContainerBuilder(object):
  """
  ContainerBuilder helps build a container image
  """
  def __init__(self, gcs_staging=None, gcr_image_tag=None, namespace=None):
    """
    Args:
      gcs_staging (str): GCS bucket/blob that can store temporary build files,
          default is gs://PROJECT_ID/kfp_container_build_staging.
      gcr_image_tag (str): GCR image tag where the target image is pushed
      namespace (str): kubernetes namespace where the pod is launched,
          default is the same namespace as the notebook service account in cluster
              or 'kubeflow' if not in cluster
    """
    # Configure the GCS staging bucket
    if gcs_staging is None:
      try:
        gcs_bucket = self._get_project_id()
      except:
        raise ValueError('Please provide the gcr_staging.')
      self._gcs_staging = 'gs://' + gcs_bucket + '/' + GCS_STAGING_BLOB_DEFAULT_PREFIX
    else:
      from pathlib import PurePath
      path = PurePath(gcs_staging).parts
      if len(path) < 2 or not path[0].startswith('gs'):
        raise ValueError('Error: {} should be a GCS path.'.format(gcs_staging))
      gcs_bucket = path[1]
      self._gcs_staging = gcs_staging
    from ._gcs_helper import GCSHelper
    GCSHelper.create_gcs_bucket_if_not_exist(gcs_bucket)

    # Configure the GCR image tag
    # Kubeflow jupyter notebook is injected with a variable as an ID
    if gcr_image_tag is None:
      KF_NOTEBOOK_ENV = 'NB_PREFIX'
      nb_id = ''
      try:
        if KF_NOTEBOOK_ENV in os.environ.keys():
          nb_id = os.environ[KF_NOTEBOOK_ENV]
        else:
          nb_id = self._get_instance_id()
      except:
        raise ValueError('Please provide the gcr_image_tag.')
      nb_id = nb_id.replace('/', '-').strip('-')
      self._gcr_image_tag = os.path.join('gcr.io', self._get_project_id(), nb_id, GCR_DEFAULT_IMAGE_SUFFIX)
    else:
      if not gcr_image_tag.startswith('gcr.io'):
        raise ValueError('Error: {} should be a GCR path.'.format(gcr_image_tag))
      self._gcr_image_tag = gcr_image_tag

    # Configure the namespace
    self._namespace = namespace
    if namespace is None:
      if os.path.exists(SERVICEACCOUNT_NAMESPACE):
        with open(SERVICEACCOUNT_NAMESPACE, 'r') as f:
          self._namespace = f.read()
      else:
        self._namespace = 'kubeflow'

  def _get_project_id(self):
    import requests
    URL = "http://metadata.google.internal/computeMetadata/v1/project/project-id"
    headers = {
      'Metadata-Flavor': 'Google'
    }
    r = requests.get(url = URL, headers = headers)
    if not r.ok:
      raise RuntimeError('ContainerBuilder failed to retrieve the project id.')
    return r.text

  def _get_instance_id(self):
    import requests
    URL = "http://metadata.google.internal/computeMetadata/v1/instance/id"
    headers = {
      'Metadata-Flavor': 'Google'
    }
    r = requests.get(url = URL, headers = headers)
    if not r.ok:
      raise RuntimeError('ContainerBuilder failed to retrieve the instance id.')
    return r.text

  def _generate_kaniko_spec(self, context, docker_filename, target_image):
    """_generate_kaniko_yaml generates kaniko job yaml based on a template yaml """
    content = {
        'apiVersion': 'v1',
        'metadata': {
            'generateName': 'kaniko-',
            'namespace': self._namespace,
            'annotations': {
                'sidecar.istio.io/inject': 'false'
            },
        },
        'kind': 'Pod',
        'spec': {
            'restartPolicy': 'Never',
            'containers': [{
                'name': 'kaniko',
                'args': ['--cache=true',
                         '--dockerfile=' + docker_filename,
                         '--context=' + context,
                         '--destination=' + target_image,
                         '--digest-file=/dev/termination-log', # This is suggested by the Kaniko devs as a way to return the image digest from Kaniko Pod. See https://github.com/GoogleContainerTools/kaniko#--digest-file
                ],
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
    if not tarball_path.endswith('.tar.gz'):
      raise ValueError('the tarball path should end with .tar.gz')
    with tarfile.open(tarball_path, 'w:gz') as tarball:
      tarball.add(dir_name, arcname='')

  def build(self, local_dir, docker_filename, target_image=None, timeout=1000):
    """
    Args:
      local_dir (str): local directory that stores all the necessary build files
      docker_filename (str): the dockerfile name that is in the local_dir
      target_image (str): the target image tag to push the final image.
      timeout (int): time out in seconds. Default: 1000
    """
    target_image = self._gcr_image_tag if target_image is None else target_image
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
      result_pod_obj = k8s_helper.run_job(kaniko_spec, timeout)
      logging.info('Kaniko job complete.')

      # Clean up
      GCSHelper.remove_gcs_blob(context)

      # Returning image name with digest
      (image_repo, _, image_tag) = target_image.partition(':')
      # When Kaniko build completes successfully, the termination message is the hash digest of the newly built image. Otherwise it's empty. See https://github.com/GoogleContainerTools/kaniko#--digest-file https://kubernetes.io/docs/tasks/debug-application-cluster/determine-reason-pod-failure/#customizing-the-termination-message
      termination_message = [status.state.terminated.message for status in result_pod_obj.status.container_statuses if status.name == 'kaniko'][0] # Note: Using status.state instead of status.last_state since last_state entries can still be None
      image_digest = termination_message
      if not image_digest.startswith('sha256:'):
        raise RuntimeError("Kaniko returned invalid image digest: {}".format(image_digest))
      strict_image_name = image_repo + '@' + image_digest
      logging.info('Built and pushed image: {}.'.format(strict_image_name))
      return strict_image_name
