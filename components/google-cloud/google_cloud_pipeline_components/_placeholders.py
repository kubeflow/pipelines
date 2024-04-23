# Copyright 2023 The Kubeflow Authors. All Rights Reserved.
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
"""Placeholders for use in component authoring."""

# prefer not using PIPELINE_TASK_ or PIPELINE_ prefix like KFP does for reduced
# verbosity
PROJECT_ID_PLACEHOLDER = "{{$.pipeline_google_cloud_project_id}}"
"""A placeholder used to obtain Google Cloud project id where the pipeline
executes. The placeholder value is set at pipeline runtime.
"""
LOCATION_PLACEHOLDER = "{{$.pipeline_google_cloud_location}}"
"""A placeholder used to obtain Google Cloud location where the pipeline
executes. The placeholder value is set at pipeline runtime.
"""
SERVICE_ACCOUNT_PLACEHOLDER = "{{$.pipeline_service_account}}"
"""A placeholder used to obtain service account that is defined in [PipelineJob](https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.pipelineJobs).
If PipelineJob doesn't have a service account set, this placeholder will be resolved to default service account.
The placeholder value is set at pipeline runtime.
"""
NETWORK_PLACEHOLDER = "{{$.pipeline_network}}"
"""A placeholder used to obtain network that is defined in [PipelineJob](https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.pipelineJobs).
If PipelineJob doesn't have a network set, this placeholder will be empty. The
placeholder value is set at pipeline runtime.
"""
PERSISTENT_RESOURCE_ID_PLACEHOLDER = "{{$.pipeline_persistent_resource_id}}"
"""A placeholder used to obtain persistent resource id that is defined in
PipelineJob [RuntimeConfig](https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.pipelineJobs#PipelineJob.RuntimeConfig).
If PipelineJob doesn't have a persistent resource id, this placeholder will be
empty. The placeholder value is set at pipeline runtime.
"""
ENCRYPTION_SPEC_KMS_KEY_NAME_PLACEHOLDER = "{{$.pipeline_encryption_key_name}}"
"""A placeholder used to obtain kmsKeyName that is defined in
PipelineJob's [EncryptionSpec](https://cloud.google.com/vertex-ai/docs/reference/rest/v1/EncryptionSpec).
If PipelineJob doesn't have a encryption key name, this placeholder will be
empty. The placeholder value is set at pipeline runtime.
"""


# omit placeholder type annotation to avoid dependency on KFP SDK internals
# placeholder is type kfp.dsl.placeholders.Placeholder
def json_escape(placeholder, level: int) -> str:
  if level not in {0, 1}:
    raise ValueError(f"Invalid level: {level}")
  # Placeholder implements __str__
  s = str(placeholder)

  return s.replace("}}", f".json_escape[{level}]}}}}")
