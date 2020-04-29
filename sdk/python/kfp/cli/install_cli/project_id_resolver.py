import click
from ..common import utils, executer


def resolve_project_id(input_project_id) -> str:
  print("\n===== Resolve GCP Project ID =====\n")

  if input_project_id == None:
    print("Didn't specify --project-id.")
    result = executer.execute('gcloud config get-value project')
    if result.has_error:
      project_in_context = ''
    else:
      project_in_context = result.stdout.rstrip()

    if project_in_context:
      return click.prompt('Input GCP Project ID', type=str, default=project_in_context)
    else:
      return click.prompt('Input GCP Project ID', type=str)
  else:
    print("GCP Project ID: {0}".format(input_project_id))
    return input_project_id
