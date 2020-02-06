# Copyright 2019 kubeflow.org.
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

import datetime
import json
import logging
import multiprocessing
import time

from kubernetes import client as k8s_client
from kubernetes.client import rest

class K8sCR(object):
  def __init__(self, group, plural, version, client):
    self.group = group
    self.plural = plural
    self.version = version
    self.client = k8s_client.CustomObjectsApi(client)

  def wait_for_condition(self,
                         namespace,
                         name,
                         expected_conditions=[],
                         timeout=datetime.timedelta(days=365),
                         polling_interval=datetime.timedelta(seconds=30),
                         status_callback=None):
    """Waits until any of the specified conditions occur.
    Args:
      namespace: namespace for the CR.
      name: Name of the CR.
      expected_conditions: A list of conditions. Function waits until any of the
        supplied conditions is reached.
      timeout: How long to wait for the CR.
      polling_interval: How often to poll for the status of the CR.
      status_callback: (Optional): Callable. If supplied this callable is
        invoked after we poll the CR. Callable takes a single argument which
        is the CR.
    """
    end_time = datetime.datetime.now() + timeout
    while True:
      try:
        results = self.client.get_namespaced_custom_object(
          self.group, self.version, namespace, self.plural, name)
      except Exception as e:
        logging.error("There was a problem waiting for %s/%s %s in namespace %s; Exception: %s",
                       self.group, self.plural, name, namespace, e)
        raise

      if results:
        if status_callback:
          status_callback(results)
        expected, condition = self.is_expected_conditions(results, expected_conditions)
        if expected:
          logging.info("%s/%s %s in namespace %s has reached the expected condition: %s.",
                       self.group, self.plural, name, namespace, condition)
          return results
        else:
          if condition:
            logging.info("Current condition of %s/%s %s in namespace %s is %s.",
                  self.group, self.plural, name, namespace, condition)

      if datetime.datetime.now() + polling_interval > end_time:
        raise Exception(
          "Timeout waiting for {0}/{1} {2} in namespace {3} to enter one of the "
          "conditions {4}.".format(self.group, self.plural, name, namespace, expected_conditions))

      time.sleep(polling_interval.seconds)

  def is_expected_conditions(self, cr_object, expected_conditions):
    return False, ""

  def create(self, spec):
    """Create a CR.
    Args:
      spec: The spec for the CR.
    """
    try:
      # Create a Resource
      namespace = spec["metadata"].get("namespace", "default")
      logging.info("Creating %s/%s %s in namespace %s.",
        self.group, self.plural, spec["metadata"]["name"], namespace)
      api_response = self.client.create_namespaced_custom_object(
        self.group, self.version, namespace, self.plural, spec)
      logging.info("Created %s/%s %s in namespace %s.",
        self.group, self.plural, spec["metadata"]["name"], namespace)
      return api_response
    except rest.ApiException as e:
      self._log_and_raise_exception(e, "create")

  def delete(self, name, namespace):
    try:
      body = {
        # Set garbage collection so that CR won't be deleted until all
        # owned references are deleted.
        "propagationPolicy": "Foreground",
      }
      logging.info("Deleteing %s/%s %s in namespace %s.",
        self.group, self.plural, name, namespace)
      api_response = self.client.delete_namespaced_custom_object(
        self.group,
        self.version,
        namespace,
        self.plural,
        name,
        body)
      logging.info("Deleted %s/%s %s in namespace %s.",
        self.group, self.plural, name, namespace)
      return api_response
    except rest.ApiException as e:
      self._log_and_raise_exception(e, "delete")

  def _log_and_raise_exception(self, ex, action):
    message = ""
    if ex.message:
      message = ex.message
    if ex.body:
      try:
        body = json.loads(ex.body)
        message = body.get("message")
      except ValueError:
        logging.error("Exception when %s %s/%s: %s", action, self.group, self.plural, ex.body)
        raise

    logging.error("Exception when %s %s/%s: %s", action, self.group, self.plural, ex.body)
    raise ex

