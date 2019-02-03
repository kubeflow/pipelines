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

class LaunchTemplateOp(BaseOp):

    def __init__(self, project_id, gcs_path, launch_parameters, location=None, validate_only=None, staging_dir=None, wait_interval=30):
        super().__init__()
        self._df = utils.create_df_client()
        self._project_id = project_id
        self._gcs_path = gcs_path
        launch_parameters['jobName'] = name_utils.normalize(launch_parameters['jobName'])
        self._launch_parameters = launch_parameters
        self._location = location
        self._validate_only = validate_only
        self._wait_interval = wait_interval
        if staging_dir:
            self.enable_staging(staging_dir, '{}/{}'.format(project_id, launch_parameters['jobName']))
        self._job_id = self.staging_states.get('job_id', None)

    def on_executing(self):
        if self._job_id:
            return self._wait_for_done(self._job_id)

        self._dump_metadata()
        request = self._df.projects().templates().launch(
            projectId = self._project_id,
            gcsPath = self._gcs_path,
            location = self._location,
            validateOnly = self._validate_only,
            body = self._launch_parameters
        )
        response = request.execute()
        return self._wait_for_done(response.get('job').get('id'))

    def on_cancelling(self):
        if self._job_id:
            self._df.projects().jobs().update(
                projectId = self._project_id,
                jobId = self._job_id,
                location = self._location,
                body = {
                    'requestedState': 'JOB_STATE_CANCELLED'
                }
            ).execute()

    def _wait_for_done(self, job_id):
        self.staging_states['job_id'] = job_id
        while True:
            job = self._get_job(job_id, full = False)
            current_state = job.get('currentState', None)
            if current_state in ['JOB_STATE_DONE']:
                return self._get_job(job_id, full = True)

            if current_state in [
                'JOB_STATE_STOPPED',
                'JOB_STATE_FAILED',
                'JOB_STATE_CANCELLED',
                'JOB_STATE_DRAINED']:
                raise RuntimeError(
                    'Job {} failed with status {}. Check logs for details.'.format(
                        job_id, current_state))

            logging.info('job status is {}, wait for {}s'.format(
                current_state, self._wait_interval))
            time.sleep(self._wait_interval)
        
    def _get_job(self, job_id, full):
        job_view = 'JOB_VIEW_SUMMARY'
        if full:
            job_view = 'JOB_VIEW_ALL'
        return self._df.projects().jobs().get(
            projectId = self._project_id,
            jobId = job_id,
            location = self._location,
            view = job_view
        ).execute()

    def _dump_metadata(self):
        #TODO: implement
        pass