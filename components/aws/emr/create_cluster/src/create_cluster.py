# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import argparse
import os
import logging
from pathlib2 import Path

from common import _utils

try:
  unicode
except NameError:
  unicode = str


def main(argv=None):
  parser = argparse.ArgumentParser(description='Create EMR Cluster')
  parser.add_argument('--region', type=str, help='EMR Cluster region.')
  parser.add_argument('--name', type=str, help='The name of the cluster to create.')
  parser.add_argument('--release_label', type=str, default="emr-5.23.0" ,help='The Amazon EMR release label, which determines the version of open-source application packages installed on the cluster.')
  parser.add_argument('--log_s3_uri', type=str, help='The path to the Amazon S3 location where logs for this cluster are stored.')
  parser.add_argument('--instance_type', type=str, default="m4.xlarge", help='The EC2 instance type of master, the core and task nodes.')
  parser.add_argument('--instance_count', type=int, default=3, help='The number of EC2 instances in the cluster.')

  parser.add_argument('--output_location_file', type=str, help='File path where the program will write the Amazon S3 URI of the transform job results.')
  args = parser.parse_args()

  logging.getLogger().setLevel(logging.INFO)
  client = _utils.get_client(args.region)
  logging.info('Creating cluster...')
  create_response = _utils.create_cluster(client, args.name, args.log_s3_uri, args.release_label, args.instance_type, args.instance_count)
  logging.info('Cluster creation request submitted. Waiting for completion...')
  _utils.wait_for_cluster(client, create_response['JobFlowId'])

  Path('/output.txt').write_text(unicode(create_response['JobFlowId']))
  logging.info('Cluster created.')

if __name__== "__main__":
  main()
