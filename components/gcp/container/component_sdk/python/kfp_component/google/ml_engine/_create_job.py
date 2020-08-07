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

from ._common_ops import wait_for_job_done, cancel_job

from kfp_component.core import KfpExecutionContext
from ._client import MLEngineClient
from .. import common as gcp_common

def create_job(
    project_id,
    job,
    job_id_prefix=None,
    job_id=None,
    wait_interval=30,
    job_object_output_path='/tmp/kfp/output/ml_engine/job.json',
    job_id_output_path='/tmp/kfp/output/ml_engine/job_id.txt',
    job_dir_output_path='/tmp/kfp/output/ml_engine/job_dir.txt',
):
    """Creates a MLEngine job.

    Args:
        project_id: the ID of the parent project of the job.
        job: the payload of the job. Must have ``jobId`` 
            and ``trainingInput`` or ``predictionInput`.
        job_id_prefix: the prefix of the generated job id.
        job_id: the created job_id, takes precedence over generated job
            id if set.
        wait_interval: optional wait interval between calls
            to get job status. Defaults to 30.
        job_object_output_path: Path for the json payload of the create job.
        job_id_output_path: Path for the ID of the created job.
        job_dir_output_path: Path for the `jobDir` of the training job.
    """
    return CreateJobOp(
        project_id=project_id,
        job=job,
        job_id_prefix=job_id_prefix,
        job_id=job_id,
        wait_interval=wait_interval,
        job_object_output_path=job_object_output_path,
        job_id_output_path=job_id_output_path,
        job_dir_output_path=job_dir_output_path,
    ).execute_and_wait()

class CreateJobOp:
    def __init__(self,project_id, job, job_id_prefix=None, job_id=None,
        wait_interval=30,
        job_object_output_path=None,
        job_id_output_path=None,
        job_dir_output_path=None,
    ):
        self._ml = MLEngineClient()
        self._project_id = project_id
        self._job_id_prefix = job_id_prefix
        self._job_id = job_id
        self._job = job
        self._wait_interval = wait_interval
        self._job_object_output_path = job_object_output_path
        self._job_id_output_path = job_id_output_path
        self._job_dir_output_path = job_dir_output_path
    
    def execute_and_wait(self):
        with KfpExecutionContext(on_cancel=lambda: cancel_job(self._ml, self._project_id, self._job_id)) as ctx:
            self._set_job_id(ctx.context_id())
            self._create_job()
            return wait_for_job_done(self._ml, self._project_id, self._job_id, self._wait_interval,
                job_object_output_path=self._job_object_output_path,
                job_id_output_path=self._job_id_output_path,
                job_dir_output_path=self._job_dir_output_path,
            )

    def _set_job_id(self, context_id):
        if self._job_id:
            job_id = self._job_id
        elif self._job_id_prefix:
            job_id = self._job_id_prefix + context_id[:16]
        else:
            job_id = 'job_' + context_id
        job_id = gcp_common.normalize_name(job_id)
        self._job_id = job_id
        self._job['jobId'] = job_id

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
