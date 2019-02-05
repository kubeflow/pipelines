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