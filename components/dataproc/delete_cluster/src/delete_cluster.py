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
# python delete_cluster.py  \
#   --project bradley-playground \
#   --region us-central1 \
#   --name ten4


import argparse

from common import _utils


def main(argv=None):
  parser = argparse.ArgumentParser(description='ML DataProc Deletion')
  parser.add_argument('--project', type=str, help='Google Cloud project ID to use.')
  parser.add_argument('--region', type=str, help='Which zone to run the analyzer.')
  parser.add_argument('--name', type=str, help='The name of the cluster to create.')
  args = parser.parse_args()

  api = _utils.get_client()
  print('Tearing down cluster...')
  delete_response = _utils.delete_cluster(api, args.project, args.region, args.name)
  print('Cluster deletion request submitted. Waiting for completion...')
  _utils.wait_for_operation(api, delete_response['name'])
  print('Cluster deleted.')


if __name__== "__main__":
  main()
