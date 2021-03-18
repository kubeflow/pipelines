# Copyright 2021 Google LLC
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

import googleapiclient.discovery as discovery
from googleapiclient import errors

class DataflowClient:
    def __init__(self):
        self._df = discovery.build('dataflow', 'v1b3', cache_discovery=False)

    def launch_template(self, project_id, gcs_path, location,
        validate_only, launch_parameters):
        return self._df.projects().locations().templates().launch(
            projectId = project_id,
            gcsPath = gcs_path,
            location = location,
            validateOnly = validate_only,
            body = launch_parameters
        ).execute()

    def get_job(self, project_id, job_id, location=None, view=None):
        return self._df.projects().locations().jobs().get(
            projectId = project_id,
            jobId = job_id,
            location = self._get_location(location),
            view = view
        ).execute()

    def cancel_job(self, project_id, job_id, location):
        return self._df.projects().locations().jobs().update(
            projectId = project_id,
            jobId = job_id,
            location = self._get_location(location),
            body = {
                'requestedState': 'JOB_STATE_CANCELLED'
            }
        ).execute()

    def list_aggregated_jobs(self, project_id, filter=None,
        view=None, page_size=None, page_token=None, location=None):
        return self._df.projects().jobs().aggregated(
            projectId = project_id,
            filter = filter,
            view = view,
            pageSize = page_size,
            pageToken = page_token,
            location = location).execute()

    def _get_location(self, location):
        if not location:
            location = 'us-central1'
        return location
