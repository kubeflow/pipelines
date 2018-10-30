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

from datetime import datetime
import time
import logging

class K8sHelper(object):
  """ Kubernetes Helper """

  def __init__(self):
    if not self._configure_k8s():
      raise Exception('K8sHelper __init__ failure')

  def _configure_k8s(self):
    try:
      from kubernetes import client as k8s_client
      from kubernetes import config
    except ImportError:
      logging.getLogger(__name__).error('Kubernetes client is not installed')
      return False
    config.load_incluster_config()
    self._api_client = k8s_client.ApiClient()
    self._corev1 = k8s_client.CoreV1Api(self._api_client)
    return True

  def _create_k8s_job(self, yaml_spec):
    """ _create_k8s_job creates a kubernetes job based on the yaml spec """
    try:
      from kubernetes import client as k8s_client
      from kubernetes import config
    except ImportError:
      logging.getLogger(__name__).error('Kubernetes client is not installed')
      return '', False
    pod = k8s_client.V1Pod(metadata=k8s_client.V1ObjectMeta(generate_name=yaml_spec['metadata']['generateName']))
    container = k8s_client.V1Container(name = yaml_spec['spec']['containers'][0]['name'],
                                       image= yaml_spec['spec']['containers'][0]['image'],
                                       args = yaml_spec['spec']['containers'][0]['args'])
    pod.spec = k8s_client.V1PodSpec(restart_policy=yaml_spec['spec']['restartPolicy'],
                                    containers = [container],
                                    service_account_name=yaml_spec['spec']['serviceAccountName'])
    try:
      api_response = self._corev1.create_namespaced_pod(yaml_spec['metadata']['namespace'], pod)
      return api_response.metadata.name, True
    except k8s_client.rest.ApiException as e:
      logging.getLogger(__name__).exception("Exception when calling CoreV1Api->create_namespaced_pod: {}\n".format(str(e)))
      return '', False

  def _wait_for_k8s_job(self, pod_name, yaml_spec, timeout):
    """ _wait_for_k8s_job waits for the job to complete """
    try:
      from kubernetes import client as k8s_client
      from kubernetes import config
    except ImportError:
      logging.getLogger(__name__).error('Kubernetes client is not installed')
      return False
    status = 'running'
    start_time = datetime.now()
    while status in ['pending', 'running']:
      # Pod pending values: https://github.com/kubernetes-client/python/blob/master/kubernetes/docs/V1PodStatus.md
      try:
        api_response = self._corev1.read_namespaced_pod(pod_name, yaml_spec['metadata']['namespace'])
        status = api_response.status.phase.lower()
        time.sleep(5)
        elapsed_time = (datetime.now() - start_time).seconds
        logging.getLogger(__name__).info('{} seconds: waiting for job to complete'.format(elapsed_time))
        if elapsed_time > timeout:
          logging.getLogger(__name__).info('Kubernetes job timeout')
          return False
      except k8s_client.rest.ApiException as e:
        logging.getLogger(__name__).exception('Exception when calling CoreV1Api->read_namespaced_pod: {}\n'.format(str(e)))
        return False
    return status == 'succeeded'

  def _delete_k8s_job(self, pod_name, yaml_spec):
    """ _delete_k8s_job deletes a pod """
    try:
      from kubernetes import client as k8s_client
      from kubernetes import config
    except ImportError:
      logging.getLogger(__name__).error('Kubernetes client is not installed')
      return False
    try:
      api_response = self._corev1.delete_namespaced_pod(pod_name, yaml_spec['metadata']['namespace'], k8s_client.V1DeleteOptions())
    except k8s_client.rest.ApiException as e:
      logging.getLogger(__name__).exception('Exception when calling CoreV1Api->delete_namespaced_pod: {}\n'.format(str(e)))

  def _read_pod_log(self, pod_name, yaml_spec):
    try:
      from kubernetes import client as k8s_client
      from kubernetes import config
    except ImportError:
      logging.getLogger(__name__).error('Kubernetes client is not installed')
      return False
    try:
      api_response = self._corev1.read_namespaced_pod_log(pod_name, yaml_spec['metadata']['namespace'])
    except k8s_client.rest.ApiException as e:
      logging.getLogger(__name__).exception('Exception when calling CoreV1Api->read_namespaced_pod_log: {}\n'.format(str(e)))
      return False
    return api_response

  def run_job(self, yaml_spec, timeout=600):
    """ run_job runs a kubernetes job and clean up afterwards """
    pod_name, succ = self._create_k8s_job(yaml_spec)
    if not succ:
      return False
    # timeout in seconds
    succ = self._wait_for_k8s_job(pod_name, yaml_spec, timeout)
    if not succ:
      logging.getLogger(__name__).info('Kubernetes job failed.')
    #TODO: investigate the read log error
    # print(self._read_pod_log(pod_name, yaml_spec))
    self._delete_k8s_job(pod_name, yaml_spec)
    return succ