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
  parser.add_argument('--bucket',
                      type = str,
                      default = 'flowers_resnet',
                      help = 'Path to GCS bucket.')
  parser.add_argument('--region',
                      type = str,
                      default = 'us-central1',
                      help = 'Region to use.')
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
  parser.add_argument('--version',
                      type = str,
                      default = 'resnet',
                      help = 'Architecture used for this model')
  args = parser.parse_args()
  return args


if __name__== "__main__":
  args = parse_arguments()
  os.environ["PROJECT"] = args.project_id
  os.environ["BUCKET"] = args.bucket
  os.environ["REGION"] = args.region

  output_dir = 'gs://' + args.bucket + '/tpu/' + args.version + '/data'

  with open("./output.txt", "w") as output_file:
    output_file.write(output_dir)

  logging.info('Removing old data from ' + output_dir)
  subprocess.call('gsutil -m rm -rf ' + output_dir, shell=True)
  logging.info('Start preprocessing data ....')
  logging.info('Copying labels to ./resnet/')
  subprocess.call('gsutil cp ' + args.labels + ' ./resnet/', shell=True)
  subprocess.call('python -m trainer.preprocess \
   --train_csv ' + args.train_csv + ' \
   --validation_csv ' + args.validation_csv + ' \
   --labels_file ./resnet/labels.txt \
   --project_id ' + args.project_id + ' \
   --output_dir ' + output_dir, shell=True)