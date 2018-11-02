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


from googleapiclient import discovery
from oauth2client.client import GoogleCredentials
import base64, sys, json
import tensorflow as tf
import argparse
import os
import subprocess
import logging

logging.getLogger().setLevel(logging.INFO)

def parse_arguments():
  """Parse command line arguments."""
  parser = argparse.ArgumentParser()
  parser.add_argument('--project_id',
                      type = str,
                      required = True,
                      default = '',
                      help = 'Pass in your project id.')
  parser.add_argument('--output',
                      type = str,
                      help = 'Path to GCS location to store output.')
  parser.add_argument('--train_csv',
                      type = str,
                      default = 'gs://cloud-ml-data/img/flower_photos/train_set.csv',
                      help = 'Path to training csv file.')
  parser.add_argument('--validation_csv',
                      type = str,
                      default = 'gs://cloud-ml-data/img/flower_photos/eval_set.csv',
                      help = 'Path to validation csv file.')
  parser.add_argument('--labels',
                      type = str,
                      default = 'gs://flowers_resnet/labels.txt',
                      help = 'Path to labels.txt.')
  args = parser.parse_args()
  return args


if __name__== "__main__":
  args = parse_arguments()

  output_dir = args.output + '/tpu/preprocessed'

  with open("/output.txt", "w") as output_file:
    output_file.write(output_dir)

  logging.info('Removing old data from ' + output_dir)
  subprocess.call('gsutil -m rm -rf ' + output_dir, shell=True)
  logging.info('Start preprocessing data ....')
  logging.info('Copying labels to ./resnet/')
  subprocess.check_call('gsutil cp ' + args.labels + ' ./resnet/', shell=True)
  subprocess.check_call('python -m trainer.preprocess \
   --train_csv ' + args.train_csv + ' \
   --validation_csv ' + args.validation_csv + ' \
   --labels_file ./resnet/labels.txt \
   --project_id ' + args.project_id + ' \
   --output_dir ' + output_dir, shell=True)
