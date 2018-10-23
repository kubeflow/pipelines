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

"""
Usage:
python launch_tf_job.py
    --workers=3
    --pss=1
    --container-image=gcr.io/${PROJECT_ID}/ml-pipeline-kubeflow-tf-trainer:${TAG_NAME}
    --output-dir gs://ml-pipeline-playground/flower/trainer
    --ui-metadata-type tensorboard
    --
    python -m trainer.task
    --job-dir=gs://ml-pipeline-playground/flower/trainer
    --transformed-data-dir=gs://ml-pipeline-playground/flower/transformed
    --schema=gs://ml-pipeline-playground/flower/schema.json
    --target=label
    --hidden-layer-size=100,50
    --steps=2000
"""
# TODO: Add unit/integration tests

import argparse
import datetime
import json
import os
import logging
import requests
import subprocess
import six
import time
import yaml
from py import tf_job_client
from kubernetes import client as k8s_client
from kubernetes import config


def _generate_train_yaml(src_filename, tfjob_ns, workers, pss, trainer_image, command):
  """_generate_train_yaml  generates train yaml files based on train.template.yaml"""
  with open(src_filename, 'r') as f:
    content = yaml.load(f)

  content['metadata']['generateName'] = 'trainer-'
  content['metadata']['namespace'] = tfjob_ns

  if workers and pss:
    content['spec']['tfReplicaSpecs']['PS']['replicas'] = pss
    content['spec']['tfReplicaSpecs']['PS']['template']['spec']['containers'][0]['image'] = trainer_image
    content['spec']['tfReplicaSpecs']['PS']['template']['spec']['containers'][0]['command'] = command
    content['spec']['tfReplicaSpecs']['Worker']['replicas'] = workers
    content['spec']['tfReplicaSpecs']['Worker']['template']['spec']['containers'][0]['image'] = trainer_image
    content['spec']['tfReplicaSpecs']['Worker']['template']['spec']['containers'][0]['command'] = command
    content['spec']['tfReplicaSpecs']['MASTER']['template']['spec']['containers'][0]['image'] = trainer_image
    content['spec']['tfReplicaSpecs']['MASTER']['template']['spec']['containers'][0]['command'] = command
  else:
    # If no workers and pss set, default is 1.
    master_spec = content['spec']['tfReplicaSpecs']['MASTER']
    worker_spec = content['spec']['tfReplicaSpecs']['Worker']
    ps_spec = content['spec']['tfReplicaSpecs']['PS']
    master_spec['template']['spec']['containers'][0]['image'] = trainer_image
    master_spec['template']['spec']['containers'][0]['command'] = command
    worker_spec['template']['spec']['containers'][0]['image'] = trainer_image
    worker_spec['template']['spec']['containers'][0]['command'] = command
    ps_spec['template']['spec']['containers'][0]['image'] = trainer_image
    ps_spec['template']['spec']['containers'][0]['command'] = command

  return content

def main(argv=None):
  parser = argparse.ArgumentParser(description='Kubeflow TFJob launcher')
  parser.add_argument('--container-image', type=str,
                      help='''Container image to run using KubeFlow TFJob. The command line should be added after --.''')
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
  parser.add_argument('--tfjob-ns', type=str,
                      default='default',
                      help='The namespace where the tfjob is submitted' +
                           'If not set, the default namespace is default')
  parser.add_argument('--tfjob-timeout-minutes', type=int,
                      default=10,
                      help='Time in minutes to wait for the TFJob to complete')
  parser.add_argument('--output-dir', type=str)
  parser.add_argument('--ui-metadata-type', type=str, default='tensorboard')
  import sys
  all_args = sys.argv[1:]
  separator_idx = all_args.index('--')
  launcher_args = all_args[:separator_idx]
  remaining_args = all_args[separator_idx + 1:]
  
  args = parser.parse_args(launcher_args)

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
  tfjob_ns = args_dict.pop('tfjob_ns')
  tfjob_timeout_minutes = args_dict.pop('tfjob_timeout_minutes')
  trainer_image = args.container_image or os.environ['TRAINER_IMAGE_NAME']
  command=remaining_args
  logging.info('Generating training template.')
  template_file = os.path.join(os.path.dirname(os.path.realpath(__file__)), 'train.template.yaml')
  content_yaml = _generate_train_yaml(template_file, tfjob_ns, workers, pss, trainer_image, command)

  logging.info('Start training.')
  # Set up handler for k8s clients
  config.load_incluster_config()
  api_client = k8s_client.ApiClient()
  create_response = tf_job_client.create_tf_job(api_client, content_yaml, version=kf_version)
  job_name = create_response['metadata']['name']

  if args.output_dir:
    # Create metadata.json file for visualization.
    metadata = {
      'outputs' : [{
        'type': args.ui_metadata_type,
        'source': args.output_dir,
      }]
    }
    with open('/mlpipeline-ui-metadata.json', 'w') as f:
      json.dump(metadata, f)

  wait_response = tf_job_client.wait_for_job(
      api_client, tfjob_ns, job_name, kf_version,
      timeout=datetime.timedelta(minutes=tfjob_timeout_minutes))
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

  #TODO: remove this after kubeflow fixes the wait_for_job issue
  # because the wait_for_job returns when the worker finishes but the master might not be complete yet.
  if 'MASTER' in wait_response['status']['tfReplicaStatuses'] and 'active' in wait_response['status']['tfReplicaStatuses']['MASTER']:
    master_active = True
    while master_active:
      # Wait for master to finish
      time.sleep(2)
      wait_response = tf_job_client.wait_for_job(api_client, tfjob_ns, job_name, kf_version,
                                             timeout=datetime.timedelta(minutes=tfjob_timeout_minutes))
      if 'active' not in wait_response['status']['tfReplicaStatuses']['MASTER']:
        master_active = False

  if succ:
    logging.info('Training success.')

  tf_job_client.delete_tf_job(api_client, tfjob_ns, job_name, version=kf_version)
  with open('/output.txt', 'w') as f:
    f.write(args.output_dir)

if __name__== "__main__":
  main()
