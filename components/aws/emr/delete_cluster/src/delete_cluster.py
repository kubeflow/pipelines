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
import logging

from common import _utils

def main(argv=None):
  parser = argparse.ArgumentParser(description='Shutdown EMR cluster')
  parser.add_argument('--region', type=str, help='The region where the cluster launches.')
  parser.add_argument('--jobflow_id', type=str, help='Job flows to be shutdown.')
  parser.add_argument('--job_id', type=str, help='Job id before cluster termination.')
  args = parser.parse_args()

  logging.getLogger().setLevel(logging.INFO)
  client = _utils.get_client(args.region)
  logging.info('Tearing down cluster...')
  _utils.delete_cluster(client, args.jobflow_id)
  logging.info('Cluster deletion request submitted. Cluster will be shut down in the background')

if __name__== "__main__":
  main()
