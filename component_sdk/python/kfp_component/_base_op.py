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
import os
import google
import json
from datetime import datetime
import logging
import sys
import re

from kubernetes import client, config
from kubernetes.client.rest import ApiException

KF_NAMESPACE = 'kubeflow'

class BaseOp:
    """Base class for operation running inside Kubeflow Pipelines.
    
    The base class is aware of the KFP environment and can cascade pipeline
    cancel or deadline event to the operation through ``on_cancelling`` 
    handler.
    """
    def __init__(self):
        config.load_incluster_config()
        self._v1_core = client.CoreV1Api()

    def execute(self):
        """Executes the operation."""
        original_sigterm_hanlder = signal.getsignal(signal.SIGTERM)
        signal.signal(signal.SIGTERM, self._exit_gracefully)
        try:
            return self.on_executing()
        except Exception as e:
            logging.error('Failed to execute the op: {}'.format(e))
            raise
        finally:
            signal.signal(signal.SIGTERM, original_sigterm_hanlder)

    def on_executing(self):
        """Triggers when execute method is called.
        Subclass should override this method.
        """
        pass

    def on_cancelling(self):
        """Triggers when the operation should be cancelled.
        Subclass should override this method.
        """
        pass

    def _exit_gracefully(self, signum, frame):
        logging.info('SIGTERM signal received.')
        if self._should_cancel():
            self.on_cancelling()

    def _should_cancel(self):
        """Checks argo's execution config deadline and decide whether the operation
        should be cancelled.

        Argo cancels workflow by setting deadline to 0 and sends SIGTERM
        signal to main container with 10s graceful period.
        """
        pod = self._load_pod()
        if not pod or not pod.metadata or not pod.metadata.annotations:
            return False
        
        argo_execution_config_json = pod.metadata.annotations.get('workflows.argoproj.io/execution', None)
        if not argo_execution_config_json:
            return False
        
        try:
            argo_execution_config = json.loads(argo_execution_config_json)
        except Exception as e:
            logging.error("Error deserializing argo execution config: {}".format(e))
            return False
        
        deadline_json = argo_execution_config.get('deadline', None)
        if not deadline_json:
            return False
        
        try:
            deadline = datetime.strptime(deadline_json, '%Y-%m-%dT%H:%M:%SZ')
        except Exception as e:
            logging.error("Error converting deadline string to datetime: {}".format(e))
            return False
        
        return datetime.now() > deadline
  
    def _load_pod(self):
        pod_name = os.environ.get('KFP_POD_NAME', None)
        if not pod_name:
            logging.warning("No KFP_POD_NAME env var. Exit without cancelling.")
            return None

        logging.info('Fetching latest pod metadata: {}.'.format(pod_name))
        try:
            return self._v1_core.read_namespaced_pod(pod_name, KF_NAMESPACE)
        except ApiException as e:
            logging.error("Exception when calling read pod {}: {}\n".format(pod_name, e))
            return None
