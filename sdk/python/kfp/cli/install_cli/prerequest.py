
from ..common import utils, executer

def show_welcome_message():
  print("""
===== Welcome =====

Welcome to use Kubeflow Pipeline Installer.
If you want to install to GCP, please make sure 'gcloud auth login' to get authenticated before launch this installer
""")

def request_tool(name):
  """Check whether `name` is on PATH and marked as executable."""

  from shutil import which

  has_tool = which(name) is not None
  if has_tool:
    print('Prerequisite Check: {0} INSTALLED'.format(name))
  else:
    utils.print_error('Prerequisite Check: {0} NOT-INSTALLED. Please install it.'.format(name))

  return has_tool

def check_tools():

  print("\n===== Check required tools =====\n")

  has_tool = request_tool('kubectl')
  has_tool = request_tool('gcloud') and has_tool
  has_tool = request_tool('gsutil') and has_tool

  if not has_tool:
    exit(1)

def check_gcp_account():

  print("\n===== Check GCP account =====\n")

  gcp_account = executer.execute_stdout('gcloud config get-value account'.split()).rstrip()
  print('Current GCP Account: {0}'.format(gcp_account))

