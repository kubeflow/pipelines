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

from kfp_component.core import KfpExecutionContext
from ._client import DataflowClient
from .. import common as gcp_common
from ._common_ops import (generate_job_name, get_job_by_name, 
    wait_for_job_done, dump_metadata, dump_job)

def launch_template(project_id, gcs_path, launch_parameters, 
    location=None, validate_only=None, wait_interval=30, 
    output_metadata_path='/tmp/mlpipeline-ui-metadata.json',
    output_job_path='/tmp/output.txt'):
    """Launchs a dataflow job from template.

    Args:
        project_id (str): Required. The ID of the Cloud Platform project 
            that the job belongs to.
        gcs_path (str): Required. A Cloud Storage path to the template 
            from which to create the job. Must be valid Cloud 
            Storage URL, beginning with 'gs://'.
        launch_parameters (dict): Parameters to provide to the template 
            being launched. Schema defined in 
            https://cloud.google.com/dataflow/docs/reference/rest/v1b3/LaunchTemplateParameters
        location (str): The regional endpoint to which to direct the 
            request.
        validate_only (boolean): If true, the request is validated but 
            not actually executed. Defaults to false.
        wait_interval (int): The wait seconds between polling.
        output_metadata_path (str): The output path of UI metadata file.
        output_job_path (str): The output path of completed job payload.
    
    Returns:
        The completed job.
    """
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
        job_name = generate_job_name(
            launch_parameters.get('jobName', None),
            ctx.context_id())
        job = get_job_by_name(df_client, project_id, job_name, 
            location)
        if not job:
            launch_parameters['jobName'] = job_name
            response = df_client.launch_template(project_id, gcs_path, 
                location, validate_only, launch_parameters)
            job = response.get('job')
        dump_metadata(output_metadata_path, project_id, job)
        job_id = job.get('id')
        job = wait_for_job_done(df_client, project_id, job_id, 
            location, wait_interval)
        dump_job(output_metadata_path, job)
        return job