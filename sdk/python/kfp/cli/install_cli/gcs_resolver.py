import click
from ..common import utils
from ..common import executer

def resolve_gcs_default_bucket(gcp_project_id, gcs_default_bucket) -> str:
  print("\n===== Resolve GCS Default Bucket =====\n")

  if gcs_default_bucket == None:
    print("Didn't specify --gcs-default-bucket.")
    gcs_default_bucket = click.prompt(
        'Input GCS Default Bucket', type=str,
        default='{0}-kubeflowpipelines-default'.format(gcp_project_id))

  GS_PREFIX = "gs://"
  if gcs_default_bucket.startswith(GS_PREFIX):
    gcs_default_bucket = gcs_default_bucket[len(GS_PREFIX):]

  # check whether bucket already exist
  cmd = "gsutil ls -p {0} gs://{1}".format(gcp_project_id, gcs_default_bucket)
  print("Executing command to check whether bucket exists: {0}".format(cmd))
  cmd_result = executer.execute_subprocess(cmd)
  if cmd_result.returncode:
    result = click.confirm("Seem can't find the bucket, do you want to create the bucket?", default=True)
    if result:
      create_bucket(gcp_project_id, gcs_default_bucket)
    else:
      print("Please prepare a correct bucket or allow create one.")
      exit(1)

  print("GCS Default Bucket: {0}".format(gcs_default_bucket))

def create_bucket(gcp_project_id, gcs_default_bucket):
  cmd = "gsutil mb -p {0} gs://{1}".format(gcp_project_id, gcs_default_bucket)
  cmd_result = executer.execute_subprocess(cmd)
  if cmd_result.returncode:
    utils.print_error("{0}".format(cmd_result.stderr))
    exit(1)
