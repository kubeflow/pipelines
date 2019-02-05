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
import json

from .. import common as gcp_common

_JOB_SUCCESSFUL_STATES = ['JOB_STATE_DONE', 'JOB_STATE_UPDATED', 'JOB_STATE_DRAINED']
_JOB_FAILED_STATES = ['JOB_STATE_STOPPED', 'JOB_STATE_FAILED', 'JOB_STATE_CANCELLED']
_JOB_TERMINATED_STATES = _JOB_SUCCESSFUL_STATES + _JOB_FAILED_STATES

def generate_job_name(job_name, context_id):
    """Generates a stable job name in the job context.

    If user provided ``job_name`` has value, the function will use it
    as a prefix and appends first 8 characters of ``context_id`` to 
    make the name unique across contexts. If the ``job_name`` is empty,
    it will use ``job-{context_id}`` as the job name.
    """
    if job_name:
        return '{}-{}'.format(
            gcp_common.normalize_name(job_name),
            context_id[:8])

    return 'job-{0}'.format(context_id)

def get_job_by_name(df_client, project_id, job_name, location=None):
    """Gets a job by its name.

    The function lists all jobs under a project or a region location.
    Compares their names with the ``job_name`` and return the job
    once it finds a match. If none of the jobs matches, it returns 
    ``None``.
    """
    page_token = None
    while True:
        response = df_client.list_aggregated_jobs(project_id, 
            page_size=50, page_token=page_token, location=location)
        for job in response.get('jobs', []):
            name = job.get('name', None)
            if job_name == name:
                return job
        page_token = response.get('nextPageToken', None)
        if not page_token:
            return None

def wait_for_job_done(df_client, project_id, job_id, location=None, wait_interval=30):
    while True:
        job = df_client.get_job(project_id, job_id, location=location)
        state = job.get('currentState', None)
        if is_job_done(state):
            return job
        elif is_job_terminated(state):
            # Terminated with error state
            raise RuntimeError('Job {} failed with error state: {}.'.format(
                job_id,
                state
            ))
        else:
            logging.info('Job {} is in pending state {}.'
                ' Waiting for {} seconds for next poll.'.format(
                    job_id,
                    state,
                    wait_interval
                ))
            time.sleep(wait_interval)


def is_job_terminated(job_state):
    return job_state in _JOB_TERMINATED_STATES

def is_job_done(job_state):
    return job_state in _JOB_SUCCESSFUL_STATES

def dump_metadata(output_metadata_path, project_id, job):
    location = job.get('location')
    job_id = job.get('id')
    metadata = {
        'outputs' : [{
            'type': 'link',
            'name': 'job details',
            'href': 'https://console.cloud.google.com/dataflow/'
                'jobsDetail/locations/{}/jobs/{}?project={}'.format(
                location, job_id, project_id)
        }]
    }
    logging.info('Dumping UI metadata: {}'.format(metadata))
    gcp_common.dump_file(output_metadata_path, json.dumps(metadata))

def dump_job(output_job_path, job):
    gcp_common.dump_file(output_job_path, json.dumps(job))
