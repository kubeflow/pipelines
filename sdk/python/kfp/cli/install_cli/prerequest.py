
from ..common import utils, executer

def show_welcome_message():
  print("\n===== Welcome =====\n")
  print("\nWelcome to use Kubeflow Pipeline CLI Installer.\n")

def check_gcloud_auth_login():

  print("\n===== Check gcloud credentials =====\n")

  print("Executing 'gcloud auth print-access-token' to test whether already login")
  result = executer.execute("gcloud auth print-access-token")
  if result.has_error:
    utils.print_warning("Can't get access token, {0}".format(result.stderr))
  else:
    print("Already login, let's continue.")
    return

  result = executer.execute_subprocess("gcloud auth login")
  if result.returncode != 0:
    utils.print_error("Can't run 'gcloud auth login', can't continue the installation")
    exit(1)

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

  gcp_account = executer.execute('gcloud config get-value account').stdout.rstrip()
  print('Current GCP Account: {0}'.format(gcp_account))

