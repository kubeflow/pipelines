


def resolve_project_id(input_project_id) -> str:
  print("\n===== Resolve GCP Project ID =====\n")

  if input_project_id == None:
    print("Didn't specify --project-id.")
    return resolve_project_id_core()
  else:
    print("GCP Project ID: {0}".format(input_project_id))
    return input_project_id

def resolve_project_id_core() -> str:
  return input('Input GCP Project ID: ')
