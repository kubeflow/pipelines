import shutil
import tempfile
import os

from ..common import utils, executer

def install(gcp_project_id, gcp_cluster_id, gcp_cluster_zone, gcs_default_bucket, instance_name, namespace):

  print("\n===== Installing Kubeflow Pipelines =====\n")
  print("gcp_project_id: {0}".format(gcp_project_id))
  print("gcp_cluster_id: {0}".format(gcp_cluster_id))
  print("gcp_cluster_zone: {0}".format(gcp_cluster_zone))
  print("gcs_default_bucket: {0}".format(gcs_default_bucket))
  print("instance_name: {0}".format(instance_name))
  print("namespace: {0}".format(namespace))
  print("\n")

  folder, cluster_scoped_folder = preparation_kustomize_folder()
  install_cluster_scoped(cluster_scoped_folder)
  wait_for_cluster_scoped_ready()
  # TODO: workload identity binding or ADC generation
  install_namespace_scoped(folder, gcp_project_id, gcs_default_bucket, instance_name, namespace)
  wait_for_namespace_scoped_ready(instance_name, namespace)

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

def install_cluster_scoped(cluster_scoped_folder):
  kustomize_file = os.path.join(cluster_scoped_folder, 'kustomization.yaml')
  # TODO: change it to use template
  with open(kustomize_file, 'w') as file:
    utils.write_line(file, 'apiVersion: kustomize.config.k8s.io/v1beta1')
    utils.write_line(file, 'kind: Kustomization')
    utils.write_line(file, 'bases:')
    utils.write_line(file, '  - github.com/kubeflow/pipelines/manifests/kustomize/cluster-scoped-resources?ref=0.5.0')

  # TODO: create Namespace in this step, another PR to refactory manifest

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
    # returncode = 1 also can mean timeout but it actually ready.
    print('Failed or timeout with code: {0}. Retrying'.format(result.returncode))
    result = executer.execute_subprocess(cmd)
    if result.returncode != 0:
      utils.print_error("Wait for cluster scoped resources be ready but failed.")
      exit(1)
  utils.print_success("Cluster scoped resources ready.")

def handle_workload_identity():
  print("!!! Not Implemented !!!")
  exit(1)

def install_namespace_scoped(folder, gcp_project_id, gcs_default_bucket, instance_name, namespace):
  kustomize_file = os.path.join(folder, 'kustomization.yaml')
  # TODO: change it to use template
  with open(kustomize_file, 'w') as file:
    utils.write_line(file, 'apiVersion: kustomize.config.k8s.io/v1beta1')
    utils.write_line(file, 'kind: Kustomization')
    utils.write_line(file, 'bases:')
    utils.write_line(file, '  - github.com/kubeflow/pipelines/manifests/kustomize/env/dev?ref=0.5.0') # TODO change to env/gcp
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
  # TODO: change it to use template
  with open(param_file, 'w') as file:
    utils.write_line(file, 'appName={0}'.format(instance_name))
    # utils.write_line(file, 'bucketName={0}'.format(gcs_default_bucket))
    # utils.write_line(file, 'gcsProjectId={0}'.format(gcp_project_id))
    # utils.write_line(file, 'gcsCloudSqlInstanceName={0}'.format(gcp_cloud_sql))

  param_db_secret_file = os.path.join(folder, 'params-db-secret.env')
  # TODO: change it to use template
  with open(param_db_secret_file, 'w') as file:
    # utils.write_line(file, 'username={0}'.format(db_username))
    # utils.write_line(file, 'password={0}'.format(db_password))
    pass

  cmd = 'kubectl apply -k {0}'.format(folder)
  print("Executing command to apply namespace scoped resources: {0}".format(cmd))
  result = executer.execute_subprocess(cmd)
  if result.returncode != 0:
    utils.print_error("Can't apply namespace scoped resources.")
    print("Please check your inputs and report issue: {0}".format(
        "https://github.com/kubeflow/pipelines/issues/new?template=BUG_REPORT.md"))
    exit(1)
  else:
    utils.print_success("Successfully applied namespace scoped resources.")

def wait_for_namespace_scoped_ready(instance_name, namespace):
  cmd = 'kubectl wait applications/{0} -n {1} --for condition=Ready --timeout=1800s'.format(
      instance_name, namespace)
  print("Executing command to wait for namespace scoped resources ready (it may takes several mintues): {0}".format(cmd))
  result = executer.execute_subprocess(cmd)
  if result.returncode != 0:
    # returncode = 1 also can mean timeout but it actually ready.
    print('Failed or timeout with code: {0}. Retrying'.format(result.returncode))
    result = executer.execute_subprocess(cmd)
    if result.returncode != 0:
      utils.print_error("Wait for namespace scoped resources be ready but failed.")
      exit(1)
  utils.print_success("Namespace scoped resources ready.")

