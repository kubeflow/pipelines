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

def submit_pyspark_job(project_id, region, cluster_name, 
    main_python_file_uri=None, args=[], pyspark_job={}, job={}, 
    wait_interval=30):
    if main_python_file_uri:
        pyspark_job['mainPythonFileUri'] = main_python_file_uri
    if args:
        pyspark_job['args'] = args
    job['pysparkJob'] = pyspark_job
    return submit_job(project_id, region, cluster_name, job, wait_interval)