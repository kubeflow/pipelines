# Copyright 2018 Google Inc. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except
# in compliance with the License. You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software distributed under the License
# is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express
# or implied. See the License for the specific language governing permissions and limitations under
# the License.


# Usage:
# python train.py  \
#   --job-dir=gs://ml-pipeline-playground/flower/trainer \
#   --transformed-data-dir=gs://ml-pipeline-playground/flower/transformed \
#   --schema=gs://ml-pipeline-playground/flower/schema.json \
#   --target=label \
#   --hidden-layer-size=100,50 \
#   --steps=2000 \
#   --workers=3 \
#   --pss=1 \

# TODO: Add unit/integration tests

import argparse
import datetime
import json
import os
import logging
import requests
import subprocess
import six
from tensorflow.python.lib.io import file_io
import time
import yaml
from py import tf_job_client
from kubernetes import client as k8s_client
from kubernetes import config


def _generate_train_yaml(src_filename, workers, pss, args_list):
  """_generate_train_yaml  generates train yaml files based on train.template.yaml"""
  with open(src_filename, 'r') as f:
    content = yaml.load(f)

  content['metadata']['generateName'] = 'trainer-'

  if workers and pss:
    content['spec']['tfReplicaSpecs']['PS']['replicas'] = pss
    content['spec']['tfReplicaSpecs']['PS']['template']['spec']['containers'][0]['command'].extend(args_list)
    content['spec']['tfReplicaSpecs']['Worker']['replicas'] = workers
    content['spec']['tfReplicaSpecs']['Worker']['template']['spec']['containers'][0]['command'].extend(args_list)
    content['spec']['tfReplicaSpecs']['MASTER']['template']['spec']['containers'][0]['command'].extend(args_list)
  else:
    # If no workers and pss set, default is 1.
    master_spec = content['spec']['tfReplicaSpecs']['MASTER']
    worker_spec = content['spec']['tfReplicaSpecs']['Worker']
    ps_spec = content['spec']['tfReplicaSpecs']['PS']
    master_spec['template']['spec']['containers'][0]['command'].extend(args_list)
    worker_spec['template']['spec']['containers'][0]['command'].extend(args_list)
    ps_spec['template']['spec']['containers'][0]['command'].extend(args_list)

  return content

def main(argv=None):
  parser = argparse.ArgumentParser(description='ML Trainer')
  parser.add_argument('--job-dir', type=str)
  parser.add_argument('--transformed-data-dir',
                      type=str,
                      required=True,
                      help='GCS path containing tf-transformed training and eval data.')
  parser.add_argument('--schema',
                      type=str,
                      required=True,
                      help='GCS json schema file path.')
  parser.add_argument('--target',
                      type=str,
                      required=True,
                      help='The name of the column to predict in training data.')
  parser.add_argument('--learning-rate',
                      type=float,
                      default=0.1,
                      help='Learning rate for training.')
  parser.add_argument('--optimizer',
                      choices=['Adam', 'SGD', 'Adagrad'],
                      default='Adagrad',
                      help='Optimizer for training. If not provided, '
                           'tf.estimator default will be used.')
  parser.add_argument('--hidden-layer-size',
                      type=str,
                      default='100',
                      help='comma separated hidden layer sizes. For example "200,100,50".')
  parser.add_argument('--steps',
                      type=int,
                      help='Maximum number of training steps to perform. If unspecified, will '
                           'honor epochs.')
  parser.add_argument('--epochs',
                      type=int,
                      help='Maximum number of training data epochs on which to train. If '
                           'both "steps" and "epochs" are specified, the training '
                           'job will run for "steps" or "epochs", whichever occurs first.')
  parser.add_argument('--preprocessing-module',
                      type=str,
                      required=False,
                      help=('GCS path to a python file defining '
                            '"preprocess" and "get_feature_columns" functions.'))
  parser.add_argument('--workers', type=int, default=0)
  parser.add_argument('--pss', type=int, default=0)
  parser.add_argument('--cluster', type=str,
                      help='GKE cluster set up for kubeflow. If set, zone must be provided. ' +
                           'If not set, assuming this runs in a GKE container and current ' +
                           'cluster is used.')
  parser.add_argument('--zone', type=str, help='zone of the kubeflow cluster.')
  parser.add_argument('--kfversion', type=str,
                      default='v1alpha2',
                      help='The version of the deployed kubeflow. ' +
                           'If not set, the default version is v1alpha2')
  args = parser.parse_args()

  logging.getLogger().setLevel(logging.INFO)
  args_dict = vars(args)
  if args.cluster and args.zone:
    cluster = args_dict.pop('cluster')
    zone = args_dict.pop('zone')
  else:
    # Get culster name and zone from metadata
    metadata_server = "http://metadata/computeMetadata/v1/instance/"
    metadata_flavor = {'Metadata-Flavor' : 'Google'}
    cluster = requests.get(metadata_server + "attributes/cluster-name",
                           headers = metadata_flavor).text
    zone = requests.get(metadata_server + "zone",
                        headers = metadata_flavor).text.split('/')[-1]

  logging.info('Getting credentials for GKE cluster %s.' % cluster)
  subprocess.call(['gcloud', 'container', 'clusters', 'get-credentials', cluster,
                   '--zone', zone])

  workers = args_dict.pop('workers')
  pss = args_dict.pop('pss')
  kf_version = args_dict.pop('kfversion')
  args_list = ['--%s=%s' % (k.replace('_', '-'),v)
               for k,v in six.iteritems(args_dict) if v is not None]
  logging.info('Generating training template.')
  template_file = os.path.join(os.path.dirname(os.path.realpath(__file__)), 'train.template.yaml')
  content_yaml = _generate_train_yaml(template_file, workers, pss, args_list)
  
  logging.info('Start training.')
  # Set up handler for k8s clients
  config.load_incluster_config()
  api_client = k8s_client.ApiClient()
  create_response = tf_job_client.create_tf_job(api_client, content_yaml, version=kf_version)
  job_name = create_response['metadata']['name']

  # Create metadata.json file for visualization.
  metadata = {
    'outputs' : [{
      'type': 'tensorboard',
      'source': args.job_dir,
    }]
  }
  with file_io.FileIO(os.path.join(args.job_dir, 'metadata.json'), 'w') as f:
    json.dump(metadata, f)

  wait_response = tf_job_client.wait_for_job(api_client, 'default', job_name, kf_version)
  succ = True
  #TODO: update this failure checking after tf-operator has the condition checking function.
  if 'Worker' in wait_response['status']['tfReplicaStatuses']:
    if 'Failed' in wait_response['status']['tfReplicaStatuses']['Worker']:
      logging.error('Training failed since workers failed.')
      succ = False
  if 'PS' in wait_response['status']['tfReplicaStatuses']:
    if 'Failed' in wait_response['status']['tfReplicaStatuses']['PS']:
      logging.error('Training failed since PSs failed.')
      succ = False
  if 'MASTER' in wait_response['status']['tfReplicaStatuses']:
    if 'Failed' in wait_response['status']['tfReplicaStatuses']['MASTER']:
      logging.error('Training failed since MASTER failed.')
      succ = False
  if succ:
    logging.info('Training success.')

  tf_job_client.delete_tf_job(api_client, 'default', job_name, version=kf_version)
  with open('/output.txt', 'w') as f:
    f.write(args.job_dir)

if __name__== "__main__":
  main()
