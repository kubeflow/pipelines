import signal
import os
from kubernetes import client, config
from kubernetes.client.rest import ApiException
from google.cloud import storage
import google
import json
from datetime import datetime
import logging
import sys
import re

class BaseOp:
    def __init__(self, name):
        self.name = name
        self.staging_states = {}
        self._storage = storage.Client()
        self._staging_bucket = None
        self._staging_path = None
        self._argo_workflow_name = None
        self._argo_node_name = None
        self._load_k8s_client()
        self._load_argo_metadata()
        self._load_staging_location()
        self._load_staging_states()

    def execute(self):
        original_sigterm_hanlder = signal.getsignal(signal.SIGTERM)
        signal.signal(signal.SIGTERM, self._exit_gracefully)
        try:
            self.on_executing()
        except:
            e = sys.exc_info()[0]
            logging.error('Failed to execute the op: {}'.format(e))
            self._stage_states()
            raise
        finally:
            signal.signal(signal.SIGTERM, original_sigterm_hanlder)

    def on_executing(self):
        pass

    def on_cancelling(self):
        pass

    def _exit_gracefully(self, signum, frame):
        logging.info('SIGTERM signal received.')
        if self._should_cancel():
            self.on_cancelling()
        else:
            self._stage_states()

    def _should_cancel(self):
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

    def _stage_states(self):
        if not self._staging_bucket or not self._staging_path or not self.staging_states:
            return

        try:
            bucket = self._storage.get_bucket(self._staging_bucket)
        except google.cloud.exceptions.NotFound:
            logging.error('Unable to find staging bucket {}.'.format(self._staging_bucket))
            return
 
        blob = bucket.blob(self._staging_path)

        try:
            states_json = json.dumps(self.staging_states)
        except TypeError as e:
            logging.error('Unable to dump staging states: {}'.format(self.staging_states))
            return

        try:
            blob.upload_from_string(states_json)
        except google.cloud.exceptions.GoogleCloudError as e:
            logging.error('Failed to upload staging states: {}'.format(e))
            return

    def _load_k8s_client(self):
        config.load_incluster_config()
        self._v1_core = client.CoreV1Api()

    def _load_argo_metadata(self):
        # Load argo metadata at start of an OP, as pod might be deleted in case of preemption.
        pod = self._load_pod()
        if not pod or not pod.metadata or not pod.metadata.labels or not pod.metadata.annotations:
            return

        self._argo_workflow_name = pod.metadata.labels.get('workflows.argoproj.io/workflow', None)
        argo_node_name = pod.metadata.annotations.get('workflows.argoproj.io/node-name', None)
        if argo_node_name:
            self._argo_node_name = re.sub(r'\s+\(\d\)', '', argo_node_name)

    def _load_staging_location(self):
        tmp_location = os.environ.get('KFP_TMP_LOCATION', None)
        if not tmp_location:
            return

        if not self._argo_workflow_name or not self._argo_node_name:
            return

        gs_prefix = 'gs://'
        if tmp_location.startswith(gs_prefix):
            tmp_location = tmp_location[len(gs_prefix):]
        splits = tmp_location.split('/', 1)
        self._staging_bucket = splits[0]
        blob_path_prefix = ''
        if len(splits) == 2:
            blob_path_prefix = splits[1]
        self._staging_path = os.path.join(blob_path_prefix, 'tmp/{}/{}/{}'.format(
            self._argo_workflow_name,
            self._argo_node_name,
            self.name
        ))

    def _load_staging_states(self):
        if not self._staging_bucket or not self._staging_path:
            return

        try:
            bucket = self._storage.get_bucket(self._staging_bucket)
        except google.cloud.exceptions.NotFound:
            logging.error('Unable to find staging bucket {}.'.format(self._staging_bucket))
            return

        blob = bucket.blob(self._staging_path)

        try:
            states_json = blob.download_as_string()
        except google.cloud.exceptions.NotFound:
            logging.info('Unable to find staging blob {}.'.format(self._staging_path))
            return

        try:
            self.staging_states = json.loads(states_json)
        except ValueError as e:
            logging.error('Unable to decode staging states: {}. Error: {}.'.format(states_json, e))
            return
  
    def _load_pod(self):
        pod_name = os.environ.get('POD_NAME', None)
        if not pod_name:
            logging.warning("No POD_NAME env var. Exit without cancelling.")
            return None

        logging.info('Fetching latest pod metadata: {}.'.format(pod_name))
        try:
            return self._v1_core.read_namespaced_pod(pod_name, 'kubeflow')
        except ApiException as e:
            logging.error("Exception when calling CoreV1Api->read_namespaced_pod: {}\n".format(e))
            return None
