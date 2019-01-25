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

import gcp_common
from kfp_component import BaseOp
from ml_engine.mlengine_client import MLEngineClient

class CreateJobOp(BaseOp):
    
    def __init__(self, project_id, job):
        super().__init__()
        self._ml = MLEngineClient()
        self._project_id = project_id
        self._job_id = gcp_common.normalize_name(job['jobId'])
        job['jobId'] = self._job_id
        self._job = job
    
    def on_executing(self):
        self._dump_metadata()

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
        
        finished_job = self._wait_for_done()
        self._dump_job(finished_job)
        if finished_job['state'] != 'SUCCEEDED':
            raise RuntimeError('Job failed with state {}. Error: {}'.format(
                finished_job['state'], finished_job.get('errorMessage', '')))
        return finished_job

    def on_cancelling(self):
        try:
            logging.info('Cancelling job {}.'.format(job_name))
            self._ml.cancel_job(self._project_id, self._job_id)
            logging.info('Cancelled job {}.'.format(job_name))
        except errors.HttpError as e:
            # Best effort to cancel the job
            logging.error('Failed to cancel the job: {}'.format(e))
            pass

    def _is_dup_job(self):
        existing_job = self._get_job()
        return existing_job.get('trainingInput', None) == self._job.get('trainingInput', None) \
            and existing_job.get('predictionInput', None) == self._job.get('predictionInput', None)

    def _wait_for_done(self):
        while True:
            job = self._ml.get_job(self._project_id, self._job_id)
            if job.get('state', None) in ['SUCCEEDED', 'FAILED', 'CANCELLED']:
                return job
            # Move to config from flag
            logging.info('job status is {}, wait for {}s'.format(job.get('state', None), 30))
            time.sleep(30)
        return job

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
        with open('/tmp/mlpipeline-ui-metadata.json', 'w') as f:
            json.dump(metadata, f)

    def _dump_job(self, job):
        logging.info('Dumping job: {}'.format(job))
        with open('/tmp/job.json', 'w') as f:
            json.dump(job, f)
