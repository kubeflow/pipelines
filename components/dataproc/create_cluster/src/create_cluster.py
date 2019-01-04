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

# Usage:
# python create_cluster.py  \
#   --project bradley-playground \
#   --zone us-central1-a \
#   --name ten4 \
#   --staging gs://bradley-playground


import argparse
import os

from common import _utils


def main(argv=None):
  parser = argparse.ArgumentParser(description='ML DataProc Setup')
  parser.add_argument('--project', type=str, help='Google Cloud project ID to use.')
  parser.add_argument('--region', type=str, help='Which zone for GCE VMs.')
  parser.add_argument('--name', type=str, help='The name of the cluster to create.')
  parser.add_argument('--staging', type=str, help='GCS path to use for staging.')
  args = parser.parse_args()

  code_path = os.path.dirname(os.path.realpath(__file__))
  init_file_source = os.path.join(code_path, 'initialization_actions.sh')
  dest_files = _utils.copy_resources_to_gcs([init_file_source], args.staging)

  try:
    api = _utils.get_client()
    print('Creating cluster...')
    create_response = _utils.create_cluster(api, args.project, args.region, args.name, dest_files[0])
    print('Cluster creation request submitted. Waiting for completion...')
    _utils.wait_for_operation(api, create_response['name'])
    with open('/output.txt', 'w') as f:
      f.write(args.name)
    print('Cluster created.')
  finally:
    _utils.remove_resources_from_gcs(dest_files)


if __name__== "__main__":
  main()
