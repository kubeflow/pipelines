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
from kubernetes import client as k8s_client
from kubernetes import config
import time
import logging
import re

from .. import dsl


class K8sHelper(object):
  """ Kubernetes Helper """

  def __init__(self):
    if not self._configure_k8s():
      raise Exception('K8sHelper __init__ failure')

  def _configure_k8s(self):
    try:
      config.load_kube_config()
      logging.info('Found local kubernetes config. Initialized with kube_config.')
    except:
      logging.info('Cannot Find local kubernetes config. Trying in-cluster config.')
      config.load_incluster_config()
      logging.info('Initialized with in-cluster config.')
    
    self._api_client = k8s_client.ApiClient()
    self._corev1 = k8s_client.CoreV1Api(self._api_client)
    return True

  def _create_k8s_job(self, yaml_spec):
    """ _create_k8s_job creates a kubernetes job based on the yaml spec """
    pod = k8s_client.V1Pod(metadata=k8s_client.V1ObjectMeta(generate_name=yaml_spec['metadata']['generateName']))
    container = k8s_client.V1Container(name = yaml_spec['spec']['containers'][0]['name'],
                                       image = yaml_spec['spec']['containers'][0]['image'],
                                       args = yaml_spec['spec']['containers'][0]['args'],
                                       volume_mounts = [k8s_client.V1VolumeMount(
                                           name=yaml_spec['spec']['containers'][0]['volumeMounts'][0]['name'],
                                           mount_path=yaml_spec['spec']['containers'][0]['volumeMounts'][0]['mountPath'],
                                       )],
                                       env = [k8s_client.V1EnvVar(
                                           name=yaml_spec['spec']['containers'][0]['env'][0]['name'],
                                           value=yaml_spec['spec']['containers'][0]['env'][0]['value'],
                                       )])
    pod.spec = k8s_client.V1PodSpec(restart_policy=yaml_spec['spec']['restartPolicy'],
                                    containers = [container],
                                    service_account_name=yaml_spec['spec']['serviceAccountName'],
                                    volumes=[k8s_client.V1Volume(
                                        name=yaml_spec['spec']['volumes'][0]['name'],
                                        secret=k8s_client.V1SecretVolumeSource(
                                           secret_name=yaml_spec['spec']['volumes'][0]['secret']['secretName'],
                                        )
                                    )])
    try:
      api_response = self._corev1.create_namespaced_pod(yaml_spec['metadata']['namespace'], pod)
      return api_response.metadata.name, True
    except k8s_client.rest.ApiException as e:
      logging.exception("Exception when calling CoreV1Api->create_namespaced_pod: {}\n".format(str(e)))
      return '', False

  def _wait_for_k8s_job(self, pod_name, yaml_spec, timeout):
    """ _wait_for_k8s_job waits for the job to complete """
    status = 'running'
    start_time = datetime.now()
    while status in ['pending', 'running']:
      # Pod pending values: https://github.com/kubernetes-client/python/blob/master/kubernetes/docs/V1PodStatus.md
      try:
        api_response = self._corev1.read_namespaced_pod(pod_name, yaml_spec['metadata']['namespace'])
        status = api_response.status.phase.lower()
        time.sleep(5)
        elapsed_time = (datetime.now() - start_time).seconds
        logging.info('{} seconds: waiting for job to complete'.format(elapsed_time))
        if elapsed_time > timeout:
          logging.info('Kubernetes job timeout')
          return False
      except k8s_client.rest.ApiException as e:
        logging.exception('Exception when calling CoreV1Api->read_namespaced_pod: {}\n'.format(str(e)))
        return False
    return status == 'succeeded'

  def _delete_k8s_job(self, pod_name, yaml_spec):
    """ _delete_k8s_job deletes a pod """
    try:
      api_response = self._corev1.delete_namespaced_pod(pod_name, yaml_spec['metadata']['namespace'], body=k8s_client.V1DeleteOptions())
    except k8s_client.rest.ApiException as e:
      logging.exception('Exception when calling CoreV1Api->delete_namespaced_pod: {}\n'.format(str(e)))

  def _read_pod_log(self, pod_name, yaml_spec):
    try:
      api_response = self._corev1.read_namespaced_pod_log(pod_name, yaml_spec['metadata']['namespace'])
    except k8s_client.rest.ApiException as e:
      logging.exception('Exception when calling CoreV1Api->read_namespaced_pod_log: {}\n'.format(str(e)))
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
      logging.info('Kubernetes job failed.')
      print(self._read_pod_log(pod_name, yaml_spec))
      return False
    self._delete_k8s_job(pod_name, yaml_spec)
    return succ

  @staticmethod
  def sanitize_k8s_name(name):
    """From _make_kubernetes_name
      sanitize_k8s_name cleans and converts the names in the workflow.
    """
    return re.sub('-+', '-', re.sub('[^-0-9a-z]+', '-', name.lower())).lstrip('-').rstrip('-')

  @staticmethod
  def convert_k8s_obj_to_json(k8s_obj):
    """
    Builds a JSON K8s object.

    If obj is None, return None.
    If obj is str, int, long, float, bool, return directly.
    If obj is datetime.datetime, datetime.date
        convert to string in iso8601 format.
    If obj is list, sanitize each element in the list.
    If obj is dict, return the dict.
    If obj is swagger model, return the properties dict.

    Args:
      obj: The data to serialize.
    Returns: The serialized form of data.
    """

    from six import text_type, integer_types, iteritems
    PRIMITIVE_TYPES = (float, bool, bytes, text_type) + integer_types
    from datetime import date, datetime
    if k8s_obj is None:
      return None
    elif isinstance(k8s_obj, PRIMITIVE_TYPES):
      return k8s_obj
    elif isinstance(k8s_obj, list):
      return [K8sHelper.convert_k8s_obj_to_json(sub_obj)
              for sub_obj in k8s_obj]
    elif isinstance(k8s_obj, tuple):
      return tuple(K8sHelper.convert_k8s_obj_to_json(sub_obj)
                   for sub_obj in k8s_obj)
    elif isinstance(k8s_obj, (datetime, date)):
      return k8s_obj.isoformat()
    elif isinstance(k8s_obj, dsl.PipelineParam): 
      if isinstance(k8s_obj.value, str):
        return k8s_obj.value
      return '{{inputs.parameters.%s}}' % k8s_obj.full_name
    
    if isinstance(k8s_obj, dict):
      obj_dict = k8s_obj
    else:
      # Convert model obj to dict except
      # attributes `swagger_types`, `attribute_map`
      # and attributes which value is not None.
      # Convert attribute name to json key in
      # model definition for request.
      obj_dict = {k8s_obj.attribute_map[attr]: getattr(k8s_obj, attr)
                  for attr, _ in iteritems(k8s_obj.swagger_types)
                  if getattr(k8s_obj, attr) is not None}

    return {key: K8sHelper.convert_k8s_obj_to_json(val)
            for key, val in iteritems(obj_dict)}