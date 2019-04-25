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

import argparse
##### Input/Output Instruction ######
# input: project, model, version

# Parsing the input arguments
def parse_arguments():
  """Parse command line arguments."""

  parser = argparse.ArgumentParser()
  parser.add_argument('--project',
                      type=str,
                      required=True,
                      help='The project name')
  parser.add_argument('--model',
                      type=str,
                      required=True,
                      help='The model name')
  parser.add_argument('--version',
                      type=str,
                      required=True,
                      help="model version")
  args = parser.parse_args()
  return args

# Google API Client utilities
#TODO: generalize the function such that it handles multiple version deletions.
def delete_model(project_name, model_name, version):
  from googleapiclient import discovery
  import time
  api = discovery.build('ml', 'v1')
  api.projects().models().versions().delete(name='projects/' + project_name + '/models/' + model_name + '/versions/' + version).execute()
  version_deleted = False
  while not version_deleted:
    try:
      api.projects().models().delete(name='projects/' + project_name + '/models/' + model_name).execute()
      version_deleted = True
    except:
      print('waiting for the versions to be deleted before the model is deleted')
      time.sleep(5)
  print('model ' + model_name + ' deleted.')

def main():
  args = parse_arguments()
  project_name = args.project
  model_name = args.model
  version = args.version
  delete_model(project_name=project_name, model_name=model_name, version=version)

if __name__ == "__main__":
  main()
