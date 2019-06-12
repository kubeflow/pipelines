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
    --mount-dict {"pvc-name":"/mnt"}
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
from distutils.util import strtobool
import json
import os
import logging
import requests
import subprocess
import six
import time
import yaml
from kubeflow.tf_operator import tf_job_client
from kubernetes import client as k8s_client
from kubernetes import config

def yamlOrJsonStr(str):
    if str == "" or str == None:
        return None
    try:
        return json.loads(str)
    except:
        return yaml.safe_load(str)

def _generate_train_yaml(src_filename, tfjob_ns, has_master, workers, pss, trainer_image, command, version, pvc_map):
  """_generate_train_yaml  generates train yaml files based on train.template.yaml"""
  with open(src_filename, 'r') as f:
    content = yaml.safe_load(f)

  content['metadata']['generateName'] = 'trainer-'
  content['metadata']['namespace'] = tfjob_ns
  content['apiVersion'] = 'kubeflow.org/' + version

  def insert_volume(spec):
    if not pvc_map:
        return
    for k, v in pvc_map.iteritems():
      spec['volumes'] = [{"name": k, "persistentVolumeClaim": {"claimName": k}}]
      spec['containers'][0]['volumeMounts'] = [{"mountPath": v, "name": k}]

  if pss:
    content['spec']['tfReplicaSpecs']['PS']['replicas'] = pss
    content['spec']['tfReplicaSpecs']['PS']['template']['spec']['containers'][0]['image'] = trainer_image
    content['spec']['tfReplicaSpecs']['PS']['template']['spec']['containers'][0]['command'] = command
    insert_volume(content['spec']['tfReplicaSpecs']['PS']['template']['spec'])
  else:
    content['spec']['tfReplicaSpecs'].pop('PS')

  if workers:
    content['spec']['tfReplicaSpecs']['Worker']['replicas'] = workers
    content['spec']['tfReplicaSpecs']['Worker']['template']['spec']['containers'][0]['image'] = trainer_image
    content['spec']['tfReplicaSpecs']['Worker']['template']['spec']['containers'][0]['command'] = command
    insert_volume(content['spec']['tfReplicaSpecs']['Worker']['template']['spec'])
  else:
    content['spec']['tfReplicaSpecs'].pop('Worker')

  if has_master:
    content['spec']['tfReplicaSpecs']['MASTER']['template']['spec']['containers'][0]['image'] = trainer_image
    content['spec']['tfReplicaSpecs']['MASTER']['template']['spec']['containers'][0]['command'] = command
    insert_volume(content['spec']['tfReplicaSpecs']['MASTER']['template']['spec'])
  else:
    content['spec']['tfReplicaSpecs'].pop('MASTER')

  return content

def main(argv=None):
  parser = argparse.ArgumentParser(description='Kubeflow TFJob launcher')
  parser.add_argument('--container-image', type=str,
                      help='''Container image to run using KubeFlow TFJob. The command line should be added after --.''')
  parser.add_argument('--has-master', type=strtobool, default=True)
  parser.add_argument('--workers', type=int, default=0)
  parser.add_argument('--pss', type=int, default=0)
  parser.add_argument('--tfjob-version', type=str,
                      default='v1beta2',
                      help='The version of the deployed tfjob.' +
                           'If not set, the default namespace is v1beta2.')
  parser.add_argument('--tfjob-ns', type=str,
                      default='kubeflow',
                      help='The namespace where the tfjob is submitted' +
                           'If not set, the default namespace is kubeflow.')
  parser.add_argument('--tfjob-timeout-minutes', type=int,
                      default=10,
                      help='Time in minutes to wait for the TFJob to complete')
  parser.add_argument('--output-dir', type=str)
  parser.add_argument('--mount-dict', type=yamlOrJsonStr, default={})
  parser.add_argument('--delete-after-done', type=strtobool,
                      default=True,
                      help='When tfjob done, delete the tfjob automatically if it is True.')
  parser.add_argument('--gke', type=strtobool,
                      default=False,
                      help='Whether kubeflow cluster installed in GKE cluster or not.')
  parser.add_argument('--cluster', type=str,
                      help='GKE cluster set up for kubeflow. If set, zone must be provided. ' +
                           'If not set, assuming this runs in a GKE container and current ' +
                           'cluster is used.')
  parser.add_argument('--zone', type=str, help='zone of the kubeflow cluster.')
  parser.add_argument('--ui-metadata-type', type=str, default='tensorboard')
  import sys
  all_args = sys.argv[1:]
  separator_idx = all_args.index('--')
  launcher_args = all_args[:separator_idx]
  remaining_args = all_args[separator_idx + 1:]
  
  args = parser.parse_args(launcher_args)

  logging.getLogger().setLevel(logging.INFO)
  args_dict = vars(args)

  if args.gke:
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
  has_master = args_dict.pop('has_master')
  pss = args_dict.pop('pss')
  tfjob_version = args_dict.pop('tfjob_version')
  tfjob_ns = args_dict.pop('tfjob_ns')
  tfjob_timeout_minutes = args_dict.pop('tfjob_timeout_minutes')
  trainer_image = args.container_image or os.environ['TRAINER_IMAGE_NAME']
  command=remaining_args
  logging.info('Generating training template.')
  template_file = os.path.join(os.path.dirname(os.path.realpath(__file__)), 'train.template.yaml')
  content_yaml = _generate_train_yaml(template_file, tfjob_ns, has_master, workers, pss, trainer_image, command, tfjob_version, args.mount_dict)

  logging.info('Start training.')
  # Set up handler for k8s clients
  config.load_incluster_config()
  api_client = k8s_client.ApiClient()
  create_response = tf_job_client.create_tf_job(api_client, content_yaml, version=tfjob_version)
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
      api_client, tfjob_ns, job_name, tfjob_version,
      timeout=datetime.timedelta(minutes=tfjob_timeout_minutes))
  if args.delete_after_done:
    tf_job_client.delete_tf_job(api_client, tfjob_ns, job_name, version=tfjob_version)
  with open('/output.txt', 'w') as f:
    f.write(json.dumps(wait_response))

if __name__== "__main__":
  main()
