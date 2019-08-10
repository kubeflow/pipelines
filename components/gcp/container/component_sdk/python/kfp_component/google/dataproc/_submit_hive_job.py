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

from ._submit_job import submit_job

def submit_hive_job(project_id, region, cluster_name, 
    queries=[], query_file_uri=None, script_variables={}, hive_job={}, 
    job={}, wait_interval=30):
    """Submits a Cloud Dataproc job for running Apache Hive queries on YARN.
    
    Args:
        project_id (str): Required. The ID of the Google Cloud Platform project 
            that the cluster belongs to.
        region (str): Required. The Cloud Dataproc region in which to handle the 
            request.
        cluster_name (str): Required. The cluster to run the job.
        queries (list): Required. The queries to execute. You do not need to 
            terminate a query with a semicolon. Multiple queries can be specified 
            in one string by separating each with a semicolon. 
        query_file_uri (str): The HCFS URI of the script that contains Hive queries.
        script_variables (dict): Optional. Mapping of query variable names to 
            values (equivalent to the Hive command: SET name="value";).
        hive_job (dict): Optional. The full payload of a [Hive job](
            https://cloud.google.com/dataproc/docs/reference/rest/v1/HiveJob)
        job (dict): Optional. The full payload of a [Dataproc job](
            https://cloud.google.com/dataproc/docs/reference/rest/v1/projects.regions.jobs).
        wait_interval (int): The wait seconds between polling the operation. 
            Defaults to 30s.

    Returns:
        The created job payload.

    Output Files:
        $KFP_OUTPUT_PATH/dataproc/job_id.txt: The ID of the created job.
    """
    if not hive_job:
        hive_job = {}
    if not job:
        job = {}
    if queries:
        hive_job['queryList'] = { 'queries': queries }
    if query_file_uri:
        hive_job['queryFileUri'] = query_file_uri
    if script_variables:
        hive_job['scriptVariables'] = script_variables
    job['hiveJob'] = hive_job
    return submit_job(project_id, region, cluster_name, job, wait_interval)
