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
import subprocess
import re
import logging

from kfp_component.core import KfpExecutionContext
from ._client import DataflowClient
from .. import common as gcp_common
from ._common_ops import (generate_job_name, get_job_by_name, 
    wait_and_dump_job, stage_file)
from ._process import Process

def launch_python(python_file_path, project_id, requirements_file_path=None, 
    location=None, job_name_prefix=None, args=[], wait_interval=30):
    """Launch a self-executing beam python file.

    Args:
        python_file_path (str): The gcs or local path to the python file to run.
        project_id (str): The ID of the parent project.
        requirements_file_path (str): Optional, the gcs or local path to the pip 
            requirements file.
        location (str): The regional endpoint to which to direct the 
            request.
        job_name_prefix (str): Optional. The prefix of the genrated job
            name. If not provided, the method will generated a random name.
        args (list): The list of args to pass to the python file.
        wait_interval (int): The wait seconds between polling.
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
            job_name_prefix,
            ctx.context_id())
        # We will always generate unique name for the job. We expect
        # job with same name was created in previous tries from the same 
        # pipeline run.
        job = get_job_by_name(df_client, project_id, job_name, 
            location)
        if job:
            return wait_and_dump_job(df_client, project_id, location, job,
                wait_interval)

        _install_requirements(requirements_file_path)
        python_file_path = stage_file(python_file_path)
        cmd = _prepare_cmd(project_id, location, job_name, python_file_path, 
            args)
        sub_process = Process(cmd)
        for line in sub_process.read_lines():
            job_id = _extract_job_id(line)
            if job_id:
                logging.info('Found job id {}'.format(job_id))
                break
        sub_process.wait_and_check()
        if not job_id:
            logging.warning('No dataflow job was found when '
                'running the python file.')
            return None
        job = df_client.get_job(project_id, job_id, 
            location=location)
        return wait_and_dump_job(df_client, project_id, location, job,
            wait_interval)

def _prepare_cmd(project_id, location, job_name, python_file_path, args):
    dataflow_args = [
        '--runner', 'dataflow', 
        '--project', project_id,
        '--job-name', job_name]
    if location:
        dataflow_args += ['--location', location]
    return (['python2', '-u', python_file_path] + 
        dataflow_args + args)

def _extract_job_id(line):
    job_id_pattern = re.compile(
        br'.*console.cloud.google.com/dataflow.*/jobs/([a-z|0-9|A-Z|\-|\_]+).*')
    matched_job_id = job_id_pattern.search(line or '')
    if matched_job_id:
        return matched_job_id.group(1).decode()
    return None

def _install_requirements(requirements_file_path):
    if not requirements_file_path:
        return
    requirements_file_path = stage_file(requirements_file_path)
    subprocess.check_call(['pip2', 'install', '-r', requirements_file_path])