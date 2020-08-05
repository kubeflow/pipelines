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

def submit_spark_job(project_id, region, cluster_name, job_id_output_path,
    main_jar_file_uri=None, main_class=None, args=[], spark_job={}, job={}, 
    wait_interval=30):
    """Submits a Cloud Dataproc job for running Apache Spark applications on YARN.
    
    Args:
        project_id (str): Required. The ID of the Google Cloud Platform project 
            that the cluster belongs to.
        region (str): Required. The Cloud Dataproc region in which to handle the 
            request.
        cluster_name (str): Required. The cluster to run the job.
        main_jar_file_uri (str): The HCFS URI of the jar file that contains the main class.
        main_class (str): The name of the driver's main class. The jar file that 
            contains the class must be in the default CLASSPATH or specified in 
            jarFileUris. 
        args (list): Optional. The arguments to pass to the driver. Do not include 
            arguments, such as --conf, that can be set as job properties, since a 
            collision may occur that causes an incorrect job submission.
        spark_job (dict): Optional. The full payload of a [SparkJob](
            https://cloud.google.com/dataproc/docs/reference/rest/v1/SparkJob).
        job (dict): Optional. The full payload of a [Dataproc job](
            https://cloud.google.com/dataproc/docs/reference/rest/v1/projects.regions.jobs).
        wait_interval (int): The wait seconds between polling the operation. 
            Defaults to 30s.
        job_id_output_path (str): Path for the ID of the created job

    Returns:
        The created job payload.
    """
    if not spark_job:
        spark_job = {}
    if not job:
        job = {}
    if main_jar_file_uri:
        spark_job['mainJarFileUri'] = main_jar_file_uri
    if main_class:
        spark_job['mainClass'] = main_class
    if args:
        spark_job['args'] = args
    job['sparkJob'] = spark_job
    return submit_job(project_id, region, cluster_name, job, wait_interval, job_id_output_path=job_id_output_path)