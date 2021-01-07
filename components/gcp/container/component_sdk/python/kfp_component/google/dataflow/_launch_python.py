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
import subprocess
import re
import logging
import os

from google.cloud import storage
from kfp_component.core import KfpExecutionContext
from ._client import DataflowClient
from ._common_ops import (wait_and_dump_job, stage_file, get_staging_location,
    read_job_id_and_location, upload_job_id_and_location)
from ._process import Process
from ..storage import parse_blob_path

def launch_python(python_file_path, project_id, region, staging_dir=None, requirements_file_path=None,
    args=[], wait_interval=30,
    job_id_output_path='/tmp/kfp/output/dataflow/job_id.txt',
    job_object_output_path='/tmp/kfp/output/dataflow/job.json',
):
    """Launch a self-executing beam python file.

    Args:
        python_file_path (str): The gcs or local path to the python file to run.
        project_id (str): The ID of the GCP project to run the Dataflow job.
        region (str): The GCP region to run the Dataflow job.
        staging_dir (str): Optional. The GCS directory for keeping staging files.
            A random subdirectory will be created under the directory to keep job info
            for resuming the job in case of failure and it will be passed as
            `staging_location` and `temp_location` command line args of the beam code.
        requirements_file_path (str): Optional, the gcs or local path to the pip
            requirements file.
        args (list): The list of args to pass to the python file.
        wait_interval (int): The wait seconds between polling.
    Returns:
        The completed job.
    """
    storage_client = storage.Client()
    df_client = DataflowClient()
    job_id = None
    location = None
    def cancel():
        if job_id:
            df_client.cancel_job(
                project_id,
                job_id,
                location
            )
    with KfpExecutionContext(on_cancel=cancel) as ctx:
        staging_location = get_staging_location(staging_dir, ctx.context_id())
        job_id, location = read_job_id_and_location(storage_client, staging_location)
        # Continue waiting for the job if it's has been uploaded to staging location.
        if job_id:
            job = df_client.get_job(project_id, job_id, location)
            return wait_and_dump_job(df_client, project_id, location, job,
                wait_interval,
                job_id_output_path=job_id_output_path,
                job_object_output_path=job_object_output_path,
            )

        _install_requirements(requirements_file_path)
        python_file_path = stage_file(python_file_path)
        cmd = _prepare_cmd(project_id, region, python_file_path, args, staging_location)
        sub_process = Process(cmd)
        for line in sub_process.read_lines():
            job_id, location = _extract_job_id_and_location(line)
            if job_id:
                logging.info('Found job id {} and location {}.'.format(job_id, location))
                upload_job_id_and_location(storage_client, staging_location, job_id, location)
                break
        sub_process.wait_and_check()
        if not job_id:
            logging.warning('No dataflow job was found when '
                'running the python file.')
            return None
        job = df_client.get_job(project_id, job_id,
            location=location)
        return wait_and_dump_job(df_client, project_id, location, job,
            wait_interval,
            job_id_output_path=job_id_output_path,
            job_object_output_path=job_object_output_path,
        )

def _prepare_cmd(project_id, region, python_file_path, args, staging_location):
    dataflow_args = [
        '--runner', 'DataflowRunner',
        '--project', project_id,
        '--region', region]
    if staging_location:
        dataflow_args += ['--staging_location', staging_location, '--temp_location', staging_location]
    return (['python', '-u', python_file_path] +
        dataflow_args + args)

def _extract_job_id_and_location(line):
    """Returns (job_id, location) from matched log.
    """
    job_id_pattern = re.compile(
        br'.*console.cloud.google.com/dataflow/jobs/(?P<location>[a-z|0-9|A-Z|\-|\_]+)/(?P<job_id>[a-z|0-9|A-Z|\-|\_]+).*')
    matched_job_id = job_id_pattern.search(line or '')
    if matched_job_id:
        return (matched_job_id.group('job_id').decode(), matched_job_id.group('location').decode())
    return (None, None)

def _install_requirements(requirements_file_path):
    if not requirements_file_path:
        return
    requirements_file_path = stage_file(requirements_file_path)
    subprocess.check_call(['pip', 'install', '-r', requirements_file_path])
