# Copyright 2020 Google LLC
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

import click

def resolve_kfp_input(instance_name, namespace):
  print("\n===== Resolve Instance Name, Namespace etc.  =====\n")

  if instance_name == None:
    print("Didn't specify --instance-name.")
    instance_name = click.prompt('Input Instance Name', type=str, default='kubeflowpipelines')
  else:
    print("Instance Name: {0}".format(instance_name))

  if namespace == None:
    print("Didn't specify --namespace.")
    namespace = click.prompt('Input namespace', type=str, default='kubeflow')
  else:
    print("Namespace: {0}".format(namespace))

  return instance_name, namespace
