from ..common import utils


def resolve_project_id(input_project_id) -> str:
  print("\n===== Resolve GCP Project ID =====\n")

  if input_project_id == None:
    print("Didn't specify --project-id.")
    return utils.input_must_have('Input GCP Project ID: ')
  else:
    print("GCP Project ID: {0}".format(input_project_id))
    return input_project_id
