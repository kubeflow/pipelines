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

from ._create_job import create_job

def create_training_job(project_id, python_module=None, package_uris=None, 
    job_id_prefix=None, args=None, job_dir=None, training_inputs=None, 
    wait_interval=30):
    """Creates a MLEngine training job.

    Args:
        project_id (str): the ID of the parent project of the job.
        python_module (str): The Python module name to run after 
            installing the packages.
        package_uris (list): The Google Cloud Storage location of the packages 
            with the training program and any additional dependencies. 
            The maximum number of package URIs is 100.
        job_id_prefix (str): the prefix of the generated job id.
        args (list): Command line arguments to pass to the program.
        job_dir (str): A Google Cloud Storage path in which to store training 
            outputs and other data needed for training. This path is passed 
            to your TensorFlow program as the '--job-dir' command-line 
            argument. The benefit of specifying this field is that Cloud ML 
            validates the path for use in training.
        training_inputs (dict): Input parameters to create a training job.
        wait_interval (int): optional wait interval between calls
            to get job status. Defaults to 30.
    """
    if not training_inputs:
        training_inputs = {}
    if python_module:
        training_inputs['pythonModule'] = python_module
    if package_uris:
        training_inputs['packageUris'] = package_uris
    if args:
        training_inputs['args'] = args
    if job_dir:
        training_inputs['jobDir'] = job_dir
    job = {
        'trainingInputs': training_inputs
    }
    return create_job(project_id, job, job_id_prefix, wait_interval)