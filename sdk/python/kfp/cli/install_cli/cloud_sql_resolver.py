import click
from ..common import utils
from ..common import executer

def resolve_cloud_sql(enable_managed_storage,
                      cloud_sql_instance_name,
                      cloud_sql_username,
                      cloud_sql_password):
  print("\n===== Resolve CloudSQL =====\n")

  if enable_managed_storage == None:
    print("Didn't specify --enable-managed-storage.")
    enable_managed_storage = click.confirm(
        'Enable Managed Storage (CloudSQL/GCS for KFP system data)', default=False)
  elif enable_managed_storage == 'true':
    enable_managed_storage = True
  else:
    enable_managed_storage = False

  if not enable_managed_storage:
    utils.print_warning("Didn't enable managed storage (CloudSQL). Uninstall would erase KFP system data.")
    return (enable_managed_storage, None, None, None)

  if cloud_sql_instance_name == None:
    print("Didn't specify --cloud-sql-instance-name.")
    print("If you dont' have a CloudSQL instance yet, please create one by yourself.")
    print("https://console.cloud.google.com/sql/instances")

    cloud_sql_instance_name = ''
    while cloud_sql_instance_name.find(':') == -1:
      utils.print_warning("CloudSQL instance connection name is formatted as 'projectId:region:instanceName'")
      cloud_sql_instance_name = click.prompt('Input an existing CloudSQL instance connection name', type=str)

  if cloud_sql_username == None:
    print("Didn't specify --cloud-sql-username.")
    cloud_sql_username = click.prompt('CloudSQL username', type=str)

  if cloud_sql_password == None:
    print("Didn't specify --cloud-sql-password.")
    cloud_sql_password = click.prompt('CloudSQL password', type=str, hide_input=True, default="")

  return (enable_managed_storage, cloud_sql_instance_name, cloud_sql_username, cloud_sql_password)
