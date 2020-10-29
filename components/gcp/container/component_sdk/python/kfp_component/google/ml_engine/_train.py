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

from fire import decorators
from ._create_job import create_job


@decorators.SetParseFns(python_version=str, runtime_version=str)
def train(project_id,
          job_id_output_path,
          job_dir_output_path,
          python_module=None,
          package_uris=None,
          region=None,
          args=None,
          job_dir=None,
          python_version=None,
          runtime_version=None,
          master_image_uri=None,
          worker_image_uri=None,
          training_input=None,
          job_id_prefix=None,
          job_id=None,
          wait_interval=30):
    """Creates a MLEngine training job.

    Args:
        project_id (str): Required. The ID of the parent project of the job.
        job_id_output_path (str): Required. Path for the ID of the created job.
        job_dir_output_path (str): Required. Path for the directory of the job.
        python_module (str): Required. The Python module name to run after
            installing the packages.
        package_uris (list): Required. The Google Cloud Storage location of
            the packages with the training program and any additional
            dependencies. The maximum number of package URIs is 100.
        region (str): Required. The Google Compute Engine region to run the
            training job in
        args (list): Command line arguments to pass to the program.
        job_dir (str): A Google Cloud Storage path in which to store training
            outputs and other data needed for training. This path is passed to
            your TensorFlow program as the '--job-dir' command-line argument.
            The benefit of specifying this field is that Cloud ML validates the
            path for use in training.
        python_version (str): Optional. The version of Python used in
            training. If not set, the default version is '2.7'. Python '3.5' is
            available when runtimeVersion is set to '1.4' and above. Python
            '2.7' works with all supported runtime versions.
        runtime_version (str): Optional. The Cloud ML Engine runtime version
            to use for training. If not set, Cloud ML Engine uses the default
            stable version, 1.0.
        master_image_uri (str): The Docker image to run on the master replica.
            This image must be in Container Registry.
        worker_image_uri (str): The Docker image to run on the worker replica.
            This image must be in Container Registry.
        training_input (dict): Input parameters to create a training job.
        job_id_prefix (str): the prefix of the generated job id.
        job_id (str): the created job_id, takes precedence over generated job
            id if set.
        wait_interval (int): optional wait interval between calls to get job
            status. Defaults to 30.
    """
    if not training_input:
        training_input = {}
    if python_module:
        training_input['pythonModule'] = python_module
    if package_uris:
        training_input['packageUris'] = package_uris
    if region:
        training_input['region'] = region
    if args:
        training_input['args'] = args
    if job_dir:
        training_input['jobDir'] = job_dir
    if python_version:
        training_input['pythonVersion'] = python_version
    if runtime_version:
        training_input['runtimeVersion'] = runtime_version
    if master_image_uri:
        if 'masterConfig' not in training_input:
            training_input['masterConfig'] = {}
        training_input['masterConfig']['imageUri'] = master_image_uri
    if worker_image_uri:
        if 'workerConfig' not in training_input:
            training_input['workerConfig'] = {}
        training_input['workerConfig']['imageUri'] = worker_image_uri
    job = {'trainingInput': training_input}
    return create_job(
        project_id=project_id,
        job=job,
        job_id_prefix=job_id_prefix,
        job_id=job_id,
        wait_interval=wait_interval,
        job_id_output_path=job_id_output_path,
        job_dir_output_path=job_dir_output_path)
