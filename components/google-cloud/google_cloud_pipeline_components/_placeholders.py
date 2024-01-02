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

# prefer not using PIPELINE_TASK_ prefix like KFP does for reduced verbosity
PROJECT_ID_PLACEHOLDER = "{{$.pipeline_google_cloud_project_id}}"
LOCATION_PLACEHOLDER = "{{$.pipeline_google_cloud_location}}"


# omit placeholder type annotation to avoid dependency on KFP SDK internals
# placeholder is type kfp.dsl.placeholders.Placeholder
def json_escape(placeholder, level: int) -> str:
  if level not in {0, 1}:
    raise ValueError(f"Invalid level: {level}")
  # Placeholder implements __str__
  s = str(placeholder)

  return s.replace("}}", f".json_escape[{level}]}}}}")
