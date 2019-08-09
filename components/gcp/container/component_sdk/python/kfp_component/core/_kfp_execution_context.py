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
import signal
import logging
import json
from datetime import datetime
import os
import hashlib
import uuid
import re

from kubernetes import client, config
from kubernetes.client.rest import ApiException

DEFAULT_NAMESPACE = 'kubeflow'
KFP_POD_ENV_NAME = 'KFP_POD_NAME'
KFP_NAMESPACE_ENV_NAME = 'KFP_NAMESPACE'
ARGO_EXECUTION_CONTROL_ANNOTATION = 'workflows.argoproj.io/execution'
ARGO_NODE_NAME_ANNOTATION = 'workflows.argoproj.io/node-name'

class KfpExecutionContext:
    """Execution context for running inside Kubeflow Pipelines.

    The base class is aware of the KFP environment and can cascade 
    pipeline cancel or deadline event to the operation through 
    ``on_cancel`` handler.

    Args:
        on_cancel: optional, function to handle KFP cancel event.
    """
    def __init__(self, on_cancel=None):
        self._load_kfp_environment()
        self._context_id = self._generate_context_id()
        logging.info('Start KFP context with ID: {}.'.format(
            self._context_id))
        self._on_cancel = on_cancel
        self._original_sigterm_hanlder = None

    def __enter__(self):
        self._original_sigterm_hanlder = signal.getsignal(signal.SIGTERM)
        signal.signal(signal.SIGTERM, self._exit_gracefully)
        return self

    def __exit__(self, type, value, traceback):
        signal.signal(signal.SIGTERM, self._original_sigterm_hanlder)

    def context_id(self):
        """Returns a stable context ID across retries. The ID is in 
        32 bytes hex format.
        """
        return self._context_id
    
    def under_kfp_environment(self):
        """Returns true if the execution is under KFP environment.
        """
        return self._pod_name and self._k8s_client and self._argo_node_name

    def _generate_context_id(self):
        if self.under_kfp_environment():
            stable_name_name = re.sub(r'\(\d+\)$', '', self._argo_node_name)
            return hashlib.md5(bytes(stable_name_name.encode())).hexdigest()
        else:
            return uuid.uuid1().hex

    def _load_kfp_environment(self):
        self._pod_name = os.environ.get(KFP_POD_ENV_NAME, None)
        self._namespace = os.environ.get(KFP_NAMESPACE_ENV_NAME, DEFAULT_NAMESPACE)
        if not self._pod_name:
            self._k8s_client = None
        else:
            try:
                config.load_incluster_config()
                self._k8s_client = client.CoreV1Api()
            except Exception as e:
                logging.warning('Failed to load kubernetes client:'
                    ' {}.'.format(e))
                self._k8s_client = None
        if self._pod_name and self._k8s_client:
            self._argo_node_name = self._get_argo_node_name()

        if not self.under_kfp_environment():
            logging.warning('Running without KFP context.')

    def _get_argo_node_name(self):
        pod = self._get_pod()
        if not pod or not pod.metadata or not pod.metadata.annotations:
            return None

        return pod.metadata.annotations.get(
            ARGO_NODE_NAME_ANNOTATION, None)

    def _exit_gracefully(self, signum, frame):
        logging.info('SIGTERM signal received.')
        if (self._on_cancel and 
                self.under_kfp_environment() and 
                self._should_cancel()):
            logging.info('Cancelling...')
            self._on_cancel()
        
        logging.info('Exit')

    def _should_cancel(self):
        """Checks argo's execution config deadline and decide whether the operation
        should be cancelled.

        Argo cancels workflow by setting deadline to 0 and sends SIGTERM
        signal to main container with 10s graceful period.
        """
        pod = self._get_pod()
        if not pod or not pod.metadata or not pod.metadata.annotations:
            logging.info('No pod metadata or annotations.')
            return False

        argo_execution_config_json = pod.metadata.annotations.get(
            ARGO_EXECUTION_CONTROL_ANNOTATION, None)
        if not argo_execution_config_json:
            logging.info('No argo execution config data.')
            return False
        
        try:
            argo_execution_config = json.loads(argo_execution_config_json)
        except Exception as e:
            logging.error("Error deserializing argo execution config: {}".format(e))
            return False
        
        deadline_json = argo_execution_config.get('deadline', None)
        if not deadline_json:
            logging.info('No argo execution deadline config.')
            return False
        
        try:
            deadline = datetime.strptime(deadline_json, '%Y-%m-%dT%H:%M:%SZ')
        except Exception as e:
            logging.error("Error converting deadline string to datetime: {}".format(e))
            return False
        
        return datetime.now() > deadline

    def _get_pod(self):
        logging.info('Fetching latest pod metadata: {}.'.format(
            self._pod_name))
        try:
            return self._k8s_client.read_namespaced_pod(
                self._pod_name, self._namespace)
        except Exception as e:
            logging.error('Failed to get pod: {}'.format(e))
            return None
