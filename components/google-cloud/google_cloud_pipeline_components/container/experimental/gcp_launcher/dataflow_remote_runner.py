# Copyright 2021 The Kubeflow Authors. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""GCP launcher for Dataflow jobs."""

def create_python_job(
    job_type,
    project,
    location,
    payload,
    gcp_resources,
):
  """Creates a Dataflow job a from a dict of dataflow python parameters.


  Args:
    job_type: Required.This Enum will specify which dataflow job type to be
      launched.
    project: Required. The project of which the resource will be launched.
    location: Required. The region of which the resource will be launched.
    payload: Required. The serialized json of parameters for python job. Note
      this can contain the Pipeline Placeholders. The payload is a json
      serialized dictionary which includes the following parameters
      python_file_path - Required. The path to the Cloud Storage bucket or local
      directory containing the Python file to be run. staging_dir - Required.
      The path to the Cloud Storage directory where the staging files are
      stored. A random subdirectory will be created under the staging directory
      to keep the job information.This is done so that you can resume the job in
      case of failure. The command line arguments, staging_location and
      temp_location, of the Beam code are passed through staging_dir.
      requirements_file_path - Optional. The path to the Cloud Storage bucket or
      local directory containing the pip requirements file. args - Optional.The
      list of arguments to pass to the Python file.
    gcp_resources: A placeholder output for returning job_id.
  Returns: And instance of GCPResouces proto with the dataflow Job ID which is
    stored in gcp_resources path.
  """
  raise NotImplementedError("This feature has not been implmented yet")


def create_flex_template_job(
    job_type,
    project,
    location,
    payload,
    gcp_resources,
):
  """Creates a Dataflow job a using a Flex template launch request.

  Args:
    job_type: Required.This Enum will specify which dataflow job type to be
      launched.
    project: Required. The project of which the resource will be launched.
    location: Required. The region of which the resource will be launched.
    payload: Required. The serialized json of flex template request body. Note
      this can contain the Pipeline Placeholders. for more details on flex
      template launch request see,
      https://cloud.google.com/dataflow/docs/reference/rest/v1b3/projects.locations.flexTemplates/launch
    gcp_resources: A placeholder output for returning job_id.
  Returns: And instance of GCPResouces proto with the dataflow Job ID which is
    stored in gcp_resources path.
  """
  raise NotImplementedError("This feature has not been implmented yet")


def create_classic_template_job(
    job_type,
    project,
    location,
    payload,
    gcp_resources,
):
  """Creates a Dataflow job a using a classic template launch request.

  Args:
    job_type: Required.This Enum will specify which dataflow job type to be
      launched.
    project: Required. The project of which the resource will be launched.
    location: Required. The region of which the resource will be launched.
    payload: Required. The serialized json of parameters for classic template
      job. Note this can contain the Pipeline Placeholders. The payload is a
      json serialized dictionary which includes the following parameters
      gcs_path - Required. The path to the Cloud Storage for the tempalte.
      launch_parameters - Required. The classic template request body, see below
      for more details.
      https://cloud.google.com/dataflow/docs/reference/rest/v1b3/projects.templates/launch
    gcp_resources: A placeholder output for returning job_id.
  Returns: And instance of GCPResouces proto with the dataflow Job ID which is
    stored in gcp_resources path.
  """
  raise NotImplementedError("This feature has not been implmented yet")
