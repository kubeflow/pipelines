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

def _generate_train_yaml(src_filename, dst_filename, workers, pss, job_name, args_list):
  """_generate_train_yaml  generates train yaml files based on train.template.yaml"""
  with open(src_filename, 'r') as f:
    content = yaml.load(f)

  if not job_name:
    # This is for unit test validation.
    job_name = 'trainer-' + datetime.datetime.now().strftime('%y%m%d-%H%M%S')
  content['metadata']['name'] = job_name
  if workers and pss:
    content['spec']['tfReplicaSpecs']['PS']['replicas'] = pss
    content['spec']['tfReplicaSpecs']['PS']['template']['spec']['containers'][0]['command'].extend(args_list)
    content['spec']['tfReplicaSpecs']['Worker']['replicas'] = workers
    content['spec']['tfReplicaSpecs']['Worker']['template']['spec']['containers'][0]['command'].extend(args_list)
    content['spec']['tfReplicaSpecs']['MASTER']['template']['spec']['containers'][0]['command'].extend(args_list)
  else:
    # No workers and pss set. Remove the sections because setting replicas=0 doesn't work.
    master_spec = content['spec']['tfReplicaSpecs']['MASTER']
    content['spec']['tfReplicaSpecs'].pop('PS')
    content['spec']['tfReplicaSpecs'].pop('Worker')
    # Set worker parameters. worker is the only item in replicaSpecs in this case.
    master_spec['template']['spec']['containers'][0]['command'].extend(args_list)

  with open(dst_filename, 'w') as f:
    yaml.dump(content, f, default_flow_style=False)

  return job_name


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
  parser.add_argument('--workers', type=int, default=0)
  parser.add_argument('--pss', type=int, default=0)
  parser.add_argument('--cluster', type=str,
                      help='GKE cluster set up for kubeflow. If set, zone must be provided. ' +
                           'If not set, assuming this runs in a GKE container and current ' +
                           'cluster is used.')
  parser.add_argument('--zone', type=str, help='zone of the kubeflow cluster.')
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
  args_list = ['--%s=%s' % (k.replace('_', '-'),v)
               for k,v in six.iteritems(args_dict) if v is not None]
  logging.info('Generating training template.')
  template_file = os.path.join(os.path.dirname(os.path.realpath(__file__)), 'train.template.yaml')

  job_name = _generate_train_yaml(template_file, 'train.yaml', workers, pss, '', args_list)
  
  logging.info('Start training.')
  subprocess.call(['kubectl', 'create', '-f', 'train.yaml', '--namespace', 'kubeflow'])

  # Create metadata.json file for visualization.
  metadata = {
    'outputs' : [{
      'type': 'tensorboard',
      'source': args.job_dir,
    }]
  }
  with file_io.FileIO(os.path.join(args.job_dir, 'metadata.json'), 'w') as f:
    json.dump(metadata, f)

  # TODO: Replace polling with kubeflow API calls.
  while True:
    time.sleep(2)
    check_job_commands = ['kubectl', 'describe', 'tfjob', job_name, '--namespace', 'kubeflow']
    kubectl_proc = subprocess.Popen(check_job_commands, stdout=subprocess.PIPE)
    grep_proc = subprocess.Popen(['grep', 'Succeeded'], stdin=kubectl_proc.stdout,
                                 stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    kubectl_proc.stdout.close() 
    stdout, stderr = grep_proc.communicate()
    result = stdout.rstrip().split(':')
    if len(result) == 2:
      logging.info('Training done.')
      with open('/output.txt', 'w') as f:
        f.write(args.job_dir)
      break

    check_job_commands = ['kubectl', 'describe', 'tfjob', job_name, '--namespace', 'kubeflow']
    kubectl_proc = subprocess.Popen(check_job_commands, stdout=subprocess.PIPE)
    grep_proc = subprocess.Popen(['grep', 'Active'], stdin=kubectl_proc.stdout,
                                 stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    kubectl_proc.stdout.close()
    stdout, stderr = grep_proc.communicate()
    result = stdout.rstrip().split(':')
    if len(result) == 2:
      continue

    # TODO: Switching to K8s API to handle errors.
    logging.error('Training failed.')
    logging.info(subprocess.check_output(check_job_commands))

  with open('/output.txt', 'w') as f:
    f.write(args.job_dir)

if __name__== "__main__":
  main()
