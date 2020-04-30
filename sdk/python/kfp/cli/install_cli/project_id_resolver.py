# Copyright 2020 Google LLC
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

import click
from ..common import utils, executer


def resolve_project_id(input_project_id) -> str:
  print("\n===== Resolve GCP Project ID =====\n")

  if input_project_id == None:
    print("Didn't specify --project-id.")
    result = executer.execute('gcloud config get-value project')
    if result.has_error:
      project_in_context = ''
    else:
      project_in_context = result.stdout.rstrip()

    if project_in_context:
      return click.prompt('Input GCP Project ID', type=str, default=project_in_context)
    else:
      return click.prompt('Input GCP Project ID', type=str)
  else:
    print("GCP Project ID: {0}".format(input_project_id))
    return input_project_id
