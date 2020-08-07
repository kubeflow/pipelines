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

from googleapiclient import errors

from kfp_component.core import display
from ._client import MLEngineClient
from .. import common as gcp_common

def wait_existing_version(ml_client, version_name, wait_interval):
    while True:
        existing_version = ml_client.get_version(version_name)
        if not existing_version:
            return None
        state = existing_version.get('state', None)
        if not state in ['CREATING', 'DELETING', 'UPDATING']:
            return existing_version
        logging.info('Version is in {} state. Wait for {}s'.format(
            state, wait_interval
        ))
        time.sleep(wait_interval)

def wait_for_operation_done(ml_client, operation_name, action, wait_interval):
    """Waits for an operation to be done.

    Args:
        operation_name: the name of the operation.
        action: the action name of the operation.
        wait_interval: the wait interview between pulling job
            status.

    Returns:
        The completed operation.

    Raises:
        RuntimeError if the operation has error.
    """
    operation = None
    while True:
        operation = ml_client.get_operation(operation_name)
        done = operation.get('done', False)
        if done:
            break
        logging.info('Operation {} is not done. Wait for {}s.'.format(operation_name, wait_interval))
        time.sleep(wait_interval)
    error = operation.get('error', None)
    if error:
        raise RuntimeError('Failed to complete {} operation {}: {} {}'.format(
            action,
            operation_name,
            error.get('code', 'Unknown code'),
            error.get('message', 'Unknown message'),
        ))
    return operation

def wait_for_job_done(ml_client, project_id, job_id, wait_interval, show_tensorboard=True,
    job_object_output_path='/tmp/kfp/output/ml_engine/job.json',
    job_id_output_path='/tmp/kfp/output/ml_engine/job_id.txt',
    job_dir_output_path='/tmp/kfp/output/ml_engine/job_dir.txt',
):
    """Waits for a CMLE job done.

    Args:
        ml_client: CMLE google api client
        project_id: the ID of the project which has the job
        job_id: the ID of the job to wait
        wait_interval: the interval in seconds to wait between polls.
        show_tensorboard: True to dump Tensorboard metadata.

    Returns:
        The completed job.

    Raises:
        RuntimeError if the job finishes with failed or cancelled state.
    """
    metadata_dumped = False
    while True:
        job = ml_client.get_job(project_id, job_id)
        print(job)
        if not metadata_dumped:
            _dump_job_metadata(project_id, job_id, job, show_tensorboard=show_tensorboard)
            metadata_dumped = True
        if job.get('state', None) in ['SUCCEEDED', 'FAILED', 'CANCELLED']:
            break
        # Move to config from flag
        logging.info('job status is {}, wait for {}s'.format(
            job.get('state', None), wait_interval))
        time.sleep(wait_interval)

    _dump_job(
        job=job,
        job_object_output_path=job_object_output_path,
        job_id_output_path=job_id_output_path,
        job_dir_output_path=job_dir_output_path,
    )

    if job['state'] != 'SUCCEEDED':
        raise RuntimeError('Job failed with state {}. Error: {}'.format(
            job['state'], job.get('errorMessage', '')))
    return job

def _dump_job_metadata(project_id, job_id, job, show_tensorboard=True):
    display.display(display.Link(
        'https://console.cloud.google.com/mlengine/jobs/{}?project={}'.format(
            job_id, project_id),
        'Job Details'
    ))
    display.display(display.Link(
        'https://console.cloud.google.com/logs/viewer?project={}&resource=ml_job/job_id/{}&interval=NO_LIMIT'.format(
            project_id, job_id),
        'Logs'
    ))
    if show_tensorboard and 'trainingInput' in job and 'jobDir' in job['trainingInput']:
        display.display(display.Tensorboard(
            job['trainingInput']['jobDir']))

def _dump_job(
    job,
    job_object_output_path,
    job_id_output_path,
    job_dir_output_path,
):
    logging.info('Dumping job: {}'.format(job))
    gcp_common.dump_file(job_object_output_path, json.dumps(job))
    gcp_common.dump_file(job_id_output_path, job['jobId'])
    job_dir = ''
    if 'trainingInput' in job and 'jobDir' in job['trainingInput']:
        job_dir = job['trainingInput']['jobDir']
    gcp_common.dump_file(job_dir_output_path, job_dir)

def cancel_job(ml_client, project_id, job_id):
    """Cancels a CMLE job.

    Args:
        ml_client: CMLE google api client
        project_id: the ID of the project which has the job
        job_id: the ID of the job to cancel
    """
    try:
        logging.info('Cancelling job {}.'.format(job_id))
        ml_client.cancel_job(project_id, job_id)
        logging.info('Cancelled job {}.'.format(job_id))
    except errors.HttpError as e:
        # Best effort to cancel the job
        logging.error('Failed to cancel the job: {}'.format(e))
        pass