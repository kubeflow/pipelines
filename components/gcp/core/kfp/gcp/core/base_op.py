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

class BaseOp:
    def __init__(self):
        self.staging_states = {}
        self._staging_bucket = None
        self._stable_name = None

    def execute(self):
        original_sigterm_hanlder = signal.getsignal(signal.SIGTERM)
        signal.signal(signal.SIGTERM, self._exit_gracefully)
        try:
            output = self.on_executing()
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

    def enable_staging(self, staging_dir, stable_name):
        gs_prefix = 'gs://'
        if staging_dir.startswith(gs_prefix):
            staging_dir = staging_dir[len(gs_prefix):]
        splits = staging_dir.split('/', maxsplit=1)
        staging_bucket = splits[0]
        blob_path_prefix = ''
        if len(splits) == 2:
            blob_path_prefix = splits[1]
        blog_path = os.path.join(blob_path_prefix, 'staging/{}/{}'.format(
            self.__class__.__name__,
            stable_name
        ))
        self._storage = storage.Client()
        self._staging_bucket = staging_bucket
        self._staging_blob_name = blog_path       
        self._reload_states()

    def _exit_gracefully(self, signum, frame):
        logging.info('SIGTERM signal received.')
        if self._should_cancel():
            self.on_cancelling()
        else:
            self._stage_states()

    def _should_cancel(self):
        pod_name = os.environ.get('POD_NAME', None)
        if not pod_name:
            logging.warning("No POD_NAME env var. Exit without cancelling.")
            return False

        logging.info('Fetching latest pod metadata: {}.'.format(pod_name))
        config.load_incluster_config()
        v1=client.CoreV1Api()
        try:
            pod = v1.read_namespaced_pod(pod_name, 'kubeflow')
        except ApiException as e:
            logging.error("Exception when calling CoreV1Api->read_namespaced_pod: {}\n".format(e))
            return False

        if not pod.metadata or not pod.metadata.annotations:
            return False
        
        argo_execution_config_json = pod.metadata.annotations.get('workflows.argoproj.io/execution', None)
        if not argo_execution_config_json:
            return False
        
        try:
            argo_execution_config = json.loads(argo_execution_config_json)
        except e:
            logging.error("Error deserializing argo execution config: {}".format(e))
            return False
        
        deadline_json = argo_execution_config.get('deadline', None)
        if not deadline_json:
            return False
        
        try:
            deadline = datetime.strptime(deadline_json, '%Y-%m-%dT%H:%M:%SZ')
        except e:
            logging.error("Error converting deadline string to datetime: {}".format(e))
            return False
        
        return datetime.now() > deadline

    def _stage_states(self):
        if not self._staging_bucket or not self._staging_blob_name or not self.staging_states:
            return

        try:
            bucket = self._storage.get_bucket(self._staging_bucket)
        except google.cloud.exceptions.NotFound:
            logging.error('Unable to find staging bucket {}.'.format(self._staging_bucket))
            return
 
        blob = bucket.blob(self._staging_blob_name)

        try:
            states_json = json.dumps(self.staging_states)
        except TypeError as e:
            logging.error('Unable to dump staging states: {}'.format(self.staging_states))
            return

        try:
            blob.upload_from_string(json.dumps(self.staging_states))
        except google.cloud.exceptions.GoogleCloudError as e:
            logging.error('Failed to upload staging states: {}'.format(e))
            return

    def _reload_states(self):
        if not self._staging_bucket or not self._staging_blob_name:
            return

        try:
            bucket = self._storage.get_bucket(self._staging_bucket)
        except google.cloud.exceptions.NotFound:
            logging.error('Unable to find staging bucket {}.'.format(self._staging_bucket))
            return

        blob = bucket.blob(self._staging_blob_name)

        try:
            states_json = blob.download_as_string()
        except google.cloud.exceptions.NotFound:
            logging.warning('Unable to find staging blob {}.'.format(self._staging_blob_name))
            return

        try:
            self.staging_states = json.loads(states_json)
        except ValueError as e:
            logging.error('Unable to decode staging states: {}. Error: {}.'.format(states_json, e))
            return
