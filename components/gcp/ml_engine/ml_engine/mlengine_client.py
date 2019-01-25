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

import logging
import time

import googleapiclient.discovery as discovery

class MLEngineClient:
    def __init__(self):
        self._ml_client = discovery.build('ml', 'v1')

    def create_job(self, project_id, job):
        return self._ml_client.projects().jobs().create(
            parent = 'projects/{}'.format(project_id),
            body = job
        ).execute()

    def wait_for_operation_done(self, operation_name, wait_interval):
        while True:
            operation = self._ml_client.projects().operations().get(
                name = operation_name
            ).execute()
            done = operation.get('done', False)
            if done:
                return operation
            logging.info('Operation {} is not done. Wait for {}s.'.format(operation_name, wait_interval))
            time.sleep(wait_interval)

    def cancel_job(self, project_id, job_id):
        job_name = 'projects/{}/jobs/{}'.format(project_id, job_id)
        self._ml_client.projects().jobs().cancel(
            name = job_name,
            body = {
                'name': job_name
            },
        ).execute()

    def get_job(self, project_id, job_id):
        job_name = 'projects/{}/jobs/{}'.format(project_id, job_id)
        return self._ml_client.projects().jobs().get(
            name=job_name).execute()