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
