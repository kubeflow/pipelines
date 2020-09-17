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

from google.cloud import storage
from kfp_component.core import KfpExecutionContext
from ._client import DataflowClient
from ._common_ops import (wait_and_dump_job, get_staging_location, 
    read_job_id_and_location, upload_job_id_and_location)

def launch_template(project_id, gcs_path, launch_parameters, 
    location=None, validate_only=None, staging_dir=None, 
    wait_interval=30,
    job_id_output_path='/tmp/kfp/output/dataflow/job_id.txt',
    job_object_output_path='/tmp/kfp/output/dataflow/job.json',
):
    """Launchs a dataflow job from template.

    Args:
        project_id (str): Required. The ID of the Cloud Platform project 
            that the job belongs to.
        gcs_path (str): Required. A Cloud Storage path to the template 
            from which to create the job. Must be valid Cloud 
            Storage URL, beginning with 'gs://'.
        launch_parameters (dict): Parameters to provide to the template 
            being launched. Schema defined in 
            https://cloud.google.com/dataflow/docs/reference/rest/v1b3/LaunchTemplateParameters.
            `jobName` will be replaced by generated name.
        location (str): The regional endpoint to which to direct the 
            request.
        validate_only (boolean): If true, the request is validated but 
            not actually executed. Defaults to false.
        staging_dir (str): Optional. The GCS directory for keeping staging files. 
            A random subdirectory will be created under the directory to keep job info
            for resuming the job in case of failure.
        wait_interval (int): The wait seconds between polling.
    
    Returns:
        The completed job.
    """
    storage_client = storage.Client()
    df_client = DataflowClient()
    job_id = None
    def cancel():
        if job_id:
            df_client.cancel_job(
                project_id,
                job_id,
                location
            )
    with KfpExecutionContext(on_cancel=cancel) as ctx:
        staging_location = get_staging_location(staging_dir, ctx.context_id())
        job_id, _ = read_job_id_and_location(storage_client, staging_location)
        # Continue waiting for the job if it's has been uploaded to staging location.
        if job_id:
            job = df_client.get_job(project_id, job_id, location)
            return wait_and_dump_job(df_client, project_id, location, job,
                wait_interval,
                job_id_output_path=job_id_output_path,
                job_object_output_path=job_object_output_path,
            )

        if not launch_parameters:
            launch_parameters = {}
        launch_parameters['jobName'] = 'job-' + ctx.context_id()
        response = df_client.launch_template(project_id, gcs_path, 
            location, validate_only, launch_parameters)
        job = response.get('job', None)
        if not job:
            # Validate only mode
            return job
        job_id = job.get('id')
        upload_job_id_and_location(storage_client, staging_location, job_id, location)
        return wait_and_dump_job(df_client, project_id, location, job,
            wait_interval,
            job_id_output_path=job_id_output_path,
            job_object_output_path=job_object_output_path,
        )
