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
    wait_for_job_done, dump_metadata, dump_job)

def launch_python(python_file_path, project_id, location=None, 
    job_name_prefix=None, args=[], wait_interval=30,
    output_metadata_path='/tmp/mlpipeline-ui-metadata.json',
    output_job_path='/tmp/output.txt'):
    df_client = DataflowClient()
    job_id = None
    sub_process = None
    def cancel():
        if sub_process:
            sub_process.terminate()
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
        job = get_job_by_name(df_client, project_id, job_name, 
            location)
        if job:
            dump_metadata(output_metadata_path, project_id, 
                job)
            job_id = job.get('id')
            job = wait_for_job_done(df_client, project_id, 
                job_id, location, wait_interval)
        if not job:
            dataflow_args = [
                '--runner', 'dataflow', 
                '--project', project_id,
                '--job-name', job_name]
            if location:
                dataflow_args += ['--location', location]
            cmd = (['python2', '-u', python_file_path] + 
                dataflow_args + args)
            for line in _run_cmd(cmd):
                if job_id:
                    continue
                job_id = _extract_job_id(line)
                if job_id:
                    
            p = subprocess.Popen(cmd,
                            stdout=subprocess.PIPE,
                            stderr=subprocess.STDOUT,
                            close_fds=True,
                            shell=False)
            for line in iter(p.stdout.readline, b''):
                if not job_id:
                    job_id = _extract_job_id(line)
                logging.info('Subprocess: {}'.format(line))
            p.stdout.close()
            return_code = p.wait()
            logging.info('Subprocess exit with code {}.'.format(
                return_code))

def _extract_job_id(line):
    job_id_pattern = re.compile(
        br'.*console.cloud.google.com/dataflow.*/jobs/([a-z|0-9|A-Z|\-|\_]+).*')
    matched_job_id = job_id_pattern.search(line or '')
    if matched_job_id:
        return matched_job_id.group(1).decode()
    return None

def _run_cmd(cmd):
    p = subprocess.Popen(cmd,
                    stdout=subprocess.PIPE,
                    stderr=subprocess.STDOUT,
                    close_fds=True,
                    shell=False)
    for line in iter(p.stdout.readline, b''):
        # if not job_id:
        #     job_id = _extract_job_id(line)
        logging.info('Subprocess: {}'.format(line))
        yield line
    p.stdout.close()
    return_code = p.wait()
    logging.info('Subprocess exit with code {}.'.format(
        return_code))