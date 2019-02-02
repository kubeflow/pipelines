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


import json
import logging
import re
import time

from googleapiclient import errors

from kfp_component.core import KfpExecutionContext
from ._client import MLEngineClient
from .. import common as gcp_common

def create_job(project_id, job, wait_interval=30):
    """Creates a MLEngine job.

    Args:
        project_id: the ID of the parent project of the job.
        job: the payload of the job. Must have ``jobId`` 
            and ``trainingInput`` or ``predictionInput`.
        wait_interval: optional wait interval between calls
            to get job status. Defaults to 30.

    """
    return CreateJobOp(project_id, job, 
        wait_interval).execute_and_wait()

class CreateJobOp:
    def __init__(self, project_id, job, wait_interval=30):
        self._ml = MLEngineClient()
        self._project_id = project_id
        self._job_id = None
        self._job = job
        self._wait_interval = wait_interval
    
    def execute_and_wait(self):
        with KfpExecutionContext(on_cancel=self._cancel) as ctx:
            self._set_job_id(ctx.context_id())
            self._dump_metadata()
            self._create_job()
            finished_job = self._wait_for_done()
            self._dump_job(finished_job)
            if finished_job['state'] != 'SUCCEEDED':
                raise RuntimeError('Job failed with state {}. Error: {}'.format(
                    finished_job['state'], finished_job.get('errorMessage', '')))
            return finished_job

    def _set_job_id(self, context_id):
        job_id = self._job.get('jobId', None)
        if not job_id:
            job_id = context_id
        job_id = gcp_common.normalize_name(job_id)
        self._job_id = job_id
        self._job['jobId'] = job_id

    def _cancel(self):
        try:
            logging.info('Cancelling job {}.'.format(self._job_id))
            self._ml.cancel_job(self._project_id, self._job_id)
            logging.info('Cancelled job {}.'.format(self._job_id))
        except errors.HttpError as e:
            # Best effort to cancel the job
            logging.error('Failed to cancel the job: {}'.format(e))
            pass

    def _create_job(self):
        try:
            self._ml.create_job(
                project_id = self._project_id,
                job = self._job
            )
        except errors.HttpError as e:
            if e.resp.status == 409:
                if not self._is_dup_job():
                    logging.error('Another job has been created with same name before: {}'.format(self._job_id))
                    raise
                logging.info('The job {} has been submitted before. Continue waiting.'.format(self._job_id))
            else:
                logging.error('Failed to create job.\nPayload: {}\nError: {}'.format(self._job, e))
                raise

    def _is_dup_job(self):
        existing_job = self._ml.get_job(self._project_id, self._job_id)
        return existing_job.get('trainingInput', None) == self._job.get('trainingInput', None) \
            and existing_job.get('predictionInput', None) == self._job.get('predictionInput', None)

    def _wait_for_done(self):
        while True:
            job = self._ml.get_job(self._project_id, self._job_id)
            if job.get('state', None) in ['SUCCEEDED', 'FAILED', 'CANCELLED']:
                return job
            # Move to config from flag
            logging.info('job status is {}, wait for {}s'.format(
                job.get('state', None), self._wait_for_done))
            time.sleep(self._wait_for_done)

    def _dump_metadata(self):
        metadata = {
            'outputs' : [{
                'type': 'sd-log',
                'resourceType': 'ml_job',
                'labels': {
                    'project_id': self._project_id,
                    'job_id': self._job_id
                }
            }, {
                'type': 'link',
                'name': 'job details',
                'href': 'https://console.cloud.google.com/mlengine/jobs/{}?project={}'.format(self._job_id, self._project_id)
            }]
        }
        if 'trainingInput' in self._job and 'jobDir' in self._job['trainingInput']:
            metadata['outputs'].append({
                'type': 'tensorboard',
                'source': self._job['trainingInput']['jobDir'],
            })
        logging.info('Dumping UI metadata: {}'.format(metadata))
        gcp_common.dump_file('/mlpipeline-ui-metadata.json', 
            json.dumps(metadata))

    def _dump_job(self, job):
        logging.info('Dumping job: {}'.format(job))
        gcp_common.dump_file('/output.txt', json.dumps(job))
        gcp_common.dump_file('/job_id.txt', job['jobId'])
