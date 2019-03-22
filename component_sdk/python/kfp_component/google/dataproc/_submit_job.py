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
import time

from ._client import DataprocClient
from kfp_component.core import KfpExecutionContext, display
from .. import common as gcp_common

def submit_job(project_id, region, cluster_name, job, wait_interval=30):
    """Submits a Cloud Dataproc job.
    
    Args:
        project_id (str): Required. The ID of the Google Cloud Platform project 
            that the cluster belongs to.
        region (str): Required. The Cloud Dataproc region in which to handle the 
            request.
        cluster_name (str): Required. The cluster to run the job.
        job (dict): Optional. The full payload of a [Dataproc job](
            https://cloud.google.com/dataproc/docs/reference/rest/v1/projects.regions.jobs).
        wait_interval (int): The wait seconds between polling the operation. 
            Defaults to 30s.

    Returns:
        The created job payload.

    Output Files:
        $KFP_OUTPUT_PATH/dataproc/job_id.txt: The ID of the created job.
    """
    if 'reference' not in job:
        job['reference'] = {}
    job['reference']['projectId'] = project_id
    if 'placement' not in job:
        job['placement'] = {}
    job['placement']['clusterName'] = cluster_name
    client = DataprocClient()
    job_id = None
    with KfpExecutionContext(
        on_cancel=lambda: client.cancel_job(
            project_id, region, job_id)) as ctx:
        submitted_job = client.submit_job(project_id, region, job, 
            request_id=ctx.context_id())
        job_id = submitted_job['reference']['jobId']
        _dump_metadata(submitted_job, region)
        submitted_job = _wait_for_job_done(client, project_id, region, 
            job_id, wait_interval)
        return _dump_job(submitted_job)

def _wait_for_job_done(client, project_id, region, job_id, wait_interval):
    while True:
        job = client.get_job(project_id, region, job_id)
        state = job['status']['state']
        if state == 'DONE':
            return job
        if state == 'ERROR':
            raise RuntimeError(job['status']['details'])
        time.sleep(wait_interval)

def _dump_metadata(job, region):
    display.display(display.Link(
        'https://console.cloud.google.com/dataproc/jobs/{}?project={}&region={}'.format(
            job.get('reference').get('jobId'), 
            job.get('reference').get('projectId'), 
            region),
        'Job Details'
    ))

def _dump_job(job):
    gcp_common.dump_file('/tmp/kfp/output/dataproc/job.json', 
        json.dumps(job))
    gcp_common.dump_file('/tmp/kfp/output/dataproc/job_id.txt',
        job.get('reference').get('jobId'))
    return job