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

import shutil
import tempfile
import os

from ..common import utils, executer

def install(gcp_project_id, gcp_cluster_id, gcp_cluster_zone, gcs_default_bucket, instance_name, namespace,
    enable_managed_storage, cloud_sql_instance_name, cloud_sql_username, cloud_sql_password,
    install_version, keep_kustomize_directory):

  print("\n===== Installing Kubeflow Pipelines =====\n")
  print("gcp_project_id: {0}".format(gcp_project_id))
  print("gcp_cluster_id: {0}".format(gcp_cluster_id))
  print("gcp_cluster_zone: {0}".format(gcp_cluster_zone))
  print("gcs_default_bucket: {0}".format(gcs_default_bucket))
  print("instance_name: {0}".format(instance_name))
  print("namespace: {0}".format(namespace))
  print("enable_managed_storage: {0}".format(enable_managed_storage))
  print("cloud_sql_instance_name: {0}".format(cloud_sql_instance_name))
  print("cloud_sql_username: {0}".format(cloud_sql_username))
  print("install_version: {0}".format(install_version))
  print("\n")

  folder, cluster_scoped_folder = preparation_kustomize_folder()

  install_cluster_scoped(cluster_scoped_folder, namespace, install_version)
  wait_for_cluster_scoped_ready()

  # TODO: workload identity binding or ADC generation

  install_namespace_scoped(folder, gcp_project_id, gcs_default_bucket, instance_name, namespace,
      enable_managed_storage, cloud_sql_instance_name, cloud_sql_username, cloud_sql_password,
      install_version)
  wait_for_namespace_scoped_ready(instance_name, namespace)

  if keep_kustomize_directory:
    print('{0} is not deleted per --keep-kustomize-directory'.format(folder))
  else:
    cleanup_kustomize_folder(folder)

  utils.print_success('SUCCESS, you can find your installation here:')
  utils.print_success('http://console.cloud.google.com/ai-platform/pipelines')

def preparation_kustomize_folder():
  print("Preparing temp directory. It's only accessable by the current user.")
  # https://docs.python.org/3/library/tempfile.html
  # It's only accessiable by current user. It will be deleted after installation.
  folder = tempfile.mkdtemp()
  cluster_scoped_folder = os.path.join(folder, 'cluster-scoped')
  os.mkdir(cluster_scoped_folder)

  print("Created temp directory: {0}".format(folder))
  return folder, cluster_scoped_folder

def cleanup_kustomize_folder(folder):
  shutil.rmtree(folder)
  print("Deleted temp directory: {0}".format(folder))

def install_cluster_scoped(cluster_scoped_folder, namespace, install_version):
  print('Preparing cluster-scoped workspace')
  kustomize_file = os.path.join(cluster_scoped_folder, 'kustomization.yaml')
  with open(kustomize_file, 'w') as file:
    utils.write_line(file, 'apiVersion: kustomize.config.k8s.io/v1beta1')
    utils.write_line(file, 'kind: Kustomization')
    utils.write_line(file, 'bases:')
    utils.write_line(file, '  - github.com/kubeflow/pipelines/manifests/kustomize/cluster-scoped-resources{0}'.format(
        version_ref(install_version)))
    utils.write_line(file, '')
    utils.write_line(file, 'configMapGenerator:')
    utils.write_line(file, '  - name: pipeline-cluster-scoped-install-config')
    utils.write_line(file, '    env: params.env')
    utils.write_line(file, '    behavior: merge')

  param_file = os.path.join(cluster_scoped_folder, 'params.env')
  with open(param_file, 'w') as file:
    utils.write_line(file, 'namespace={0}'.format(namespace))

  cmd = 'kubectl apply -k {0}'.format(cluster_scoped_folder)
  print("Executing command to apply cluster scoped resources: {0}".format(cmd))
  result = executer.execute_subprocess(cmd)
  if result.returncode != 0:
    utils.print_error("Can't apply cluster scoped resources.")
    print("Please check your inputs and report issue: {0}".format(
        "https://github.com/kubeflow/pipelines/issues/new?template=BUG_REPORT.md"))
    exit(1)
  else:
    utils.print_success("Successfully applied cluster scoped resources.")

def wait_for_cluster_scoped_ready():
  cmd = 'kubectl wait crd/applications.app.k8s.io --for condition=established --timeout=60s'
  print("Executing command to wait for cluster scoped resources ready: {0}".format(cmd))
  result = executer.execute_subprocess(cmd)
  if result.returncode != 0:
    print('Failed or timeout with code: {0}. Retry once'.format(result.returncode))
    result = executer.execute_subprocess(cmd)
    if result.returncode != 0:
      utils.print_error("Wait for cluster scoped resources be ready but failed.")
      exit(1)
  utils.print_success("Cluster scoped resources ready.")

def install_namespace_scoped(folder, gcp_project_id, gcs_default_bucket, instance_name, namespace,
    enable_managed_storage, cloud_sql_instance_name, cloud_sql_username, cloud_sql_password,
    install_version):
  print('Preparing namespace-scoped workspace')
  kustomize_file = os.path.join(folder, 'kustomization.yaml')
  with open(kustomize_file, 'w') as file:
    utils.write_line(file, 'apiVersion: kustomize.config.k8s.io/v1beta1')
    utils.write_line(file, 'kind: Kustomization')
    utils.write_line(file, 'bases:')
    utils.write_line(file, '  - github.com/kubeflow/pipelines/manifests/kustomize/env/{0}{1}'.format(
        'gcp' if enable_managed_storage else 'dev', version_ref(install_version)))
    utils.write_line(file, '')
    utils.write_line(file, 'commonLabels:')
    utils.write_line(file, '  application-crd-id: kubeflow-pipelines')
    utils.write_line(file, '')
    utils.write_line(file, 'configMapGenerator:')
    utils.write_line(file, '  - name: pipeline-install-config')
    utils.write_line(file, '    env: params.env')
    utils.write_line(file, '    behavior: merge')
    utils.write_line(file, '')
    utils.write_line(file, 'secretGenerator:')
    utils.write_line(file, '  - name: mysql-secret')
    utils.write_line(file, '    env: params-db-secret.env')
    utils.write_line(file, '    behavior: merge')
    utils.write_line(file, '')
    utils.write_line(file, 'namespace: {0}'.format(namespace))
    utils.write_line(file, '')

  param_file = os.path.join(folder, 'params.env')
  with open(param_file, 'w') as file:
    utils.write_line(file, 'appName={0}'.format(instance_name))
    if enable_managed_storage:
      utils.write_line(file, 'bucketName={0}'.format(gcs_default_bucket))
      utils.write_line(file, 'gcsProjectId={0}'.format(gcp_project_id))
      utils.write_line(file, 'gcsCloudSqlInstanceName={0}'.format(cloud_sql_instance_name))

  param_db_secret_file = os.path.join(folder, 'params-db-secret.env')
  with open(param_db_secret_file, 'w') as file:
    if enable_managed_storage:
      utils.write_line(file, 'username={0}'.format(cloud_sql_username))
      utils.write_line(file, 'password={0}'.format(cloud_sql_password))

  cmd = 'kubectl apply -k {0}'.format(folder)
  print("Executing command to apply namespace scoped resources: {0}".format(cmd))
  result = executer.execute_subprocess(cmd)
  if result.returncode != 0:
    utils.print_error("Can't apply namespace scoped resources.")
    installation_fatal(folder)
  else:
    utils.print_success("Successfully applied namespace scoped resources.")

def wait_for_namespace_scoped_ready(instance_name, namespace):
  cmd = 'kubectl wait applications/{0} -n {1} --for condition=Ready --timeout=1800s'.format(
      instance_name, namespace)
  print("Executing command to wait for namespace scoped resources ready (it may takes several mintues): {0}".format(cmd))
  result = executer.execute_subprocess(cmd)
  if result.returncode != 0:
    print('Failed or timeout with code: {0}. Retry once.'.format(result.returncode))
    result = executer.execute_subprocess(cmd)
    if result.returncode != 0:
      utils.print_error("Wait for namespace scoped resources be ready but failed.")
      installation_fatal(folder)
  utils.print_success("Namespace scoped resources ready.")

def installation_fatal(folder):
  print("Please check your inputs and report issue.")
  print("https://github.com/kubeflow/pipelines/issues/new?template=BUG_REPORT.md")
  print("Temp directory {0} is not deleted for your debugging purpose".format(folder))
  exit(1)

def version_ref(install_version):
  # ?ref=x.x.x
  return '' if install_version == 'latest' else '?ref={0}'.format(install_version)

def handle_workload_identity():
  print("!!! Not Implemented !!!")
  exit(1)
