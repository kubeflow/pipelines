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
import os
import tempfile

from kfp_component.core import display
from .. import common as gcp_common
from ..storage import download_blob, parse_blob_path, is_gcs_path

_JOB_SUCCESSFUL_STATES = ['JOB_STATE_DONE', 'JOB_STATE_UPDATED', 'JOB_STATE_DRAINED']
_JOB_FAILED_STATES = ['JOB_STATE_STOPPED', 'JOB_STATE_FAILED', 'JOB_STATE_CANCELLED']
_JOB_TERMINATED_STATES = _JOB_SUCCESSFUL_STATES + _JOB_FAILED_STATES

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

def wait_and_dump_job(df_client, project_id, location, job, 
    wait_interval,
    job_id_output_path,
    job_object_output_path,
):
    display_job_link(project_id, job)
    job_id = job.get('id')
    job = wait_for_job_done(df_client, project_id, job_id, 
        location, wait_interval)
    gcp_common.dump_file(job_object_output_path, json.dumps(job))
    gcp_common.dump_file(job_id_output_path, job.get('id'))
    return job

def is_job_terminated(job_state):
    return job_state in _JOB_TERMINATED_STATES

def is_job_done(job_state):
    return job_state in _JOB_SUCCESSFUL_STATES

def display_job_link(project_id, job):
    location = job.get('location')
    job_id = job.get('id')
    display.display(display.Link(
        href = 'https://console.cloud.google.com/dataflow/'
            'jobsDetail/locations/{}/jobs/{}?project={}'.format(
            location, job_id, project_id),
        text = 'Job Details'
    ))

def stage_file(local_or_gcs_path):
    if not is_gcs_path(local_or_gcs_path):
        return local_or_gcs_path
    _, blob_path = parse_blob_path(local_or_gcs_path)
    file_name = os.path.basename(blob_path)
    local_file_path = os.path.join(tempfile.mkdtemp(), file_name)
    download_blob(local_or_gcs_path, local_file_path)
    return local_file_path

def get_staging_location(staging_dir, context_id):
    if not staging_dir:
        return None

    staging_location = os.path.join(staging_dir, context_id)
    logging.info('staging_location: {}'.format(staging_location))
    return staging_location

def read_job_id_and_location(storage_client, staging_location):
    if staging_location:
        job_blob = _get_job_blob(storage_client, staging_location)
        if job_blob.exists():
            job_data = job_blob.download_as_string().decode().split(',')
            # Returns (job_id, location)
            logging.info('Found existing job {}.'.format(job_data))
            return (job_data[0], job_data[1])

    return (None, None)

def upload_job_id_and_location(storage_client, staging_location, job_id, location):
    if not staging_location:
        return
    if not location:
        location = ''
    data = '{},{}'.format(job_id, location)
    job_blob = _get_job_blob(storage_client, staging_location)
    logging.info('Uploading {} to {}.'.format(data, job_blob))
    job_blob.upload_from_string(data)

def _get_job_blob(storage_client, staging_location):
    bucket_name, staging_blob_name = parse_blob_path(staging_location)
    job_blob_name = os.path.join(staging_blob_name, 'kfp/dataflow/launch_python/job.txt')
    bucket = storage_client.bucket(bucket_name)
    return bucket.blob(job_blob_name)