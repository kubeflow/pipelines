"""Utility functions used to create custom Kubeflow components."""

from typing import Any

from google_cloud_pipeline_components import _image


def build_custom_job_payload(
    *,
    display_name: str,
    image_uri: str,
    args: list[str],
    machine_type: str = 'n1-standard-4',
    service_account: str = '',
    network: str = '',
    reserved_ip_ranges: list[str] = [],
    enable_web_access: bool = False,
    encryption_spec_key_name: str = '',
    accelerator_type: str = 'ACCELERATOR_TYPE_UNSPECIFIED',
    accelerator_count: int = 0,
) -> dict[str, Any]:
  """Generates payload for a CustomJob in a Sec4 horizontal compliant way.

  Args:
    display_name: CustomJob display name. Can contain up to 128 UTF-8
      characters.
    machine_type: The type of the machine. See the list of machine types
      supported for custom training:
      https://cloud.google.com/vertex-ai/docs/training/configure-compute#machine-types
    accelerator_type: The type of accelerator(s) that may be attached to the
      machine as per acceleratorCount.
      https://cloud.google.com/vertex-ai/docs/reference/rest/v1/MachineSpec#AcceleratorType
    accelerator_count: The number of accelerators to attach to the machine.
    image_uri: Docker image URI to use for the CustomJob.
    args: Arguments to pass to the Docker image.
    service_account: Sets the default service account for workload run-as
      account. The service account running the pipeline
      (https://cloud.google.com/vertex-ai/docs/pipelines/configure-project#service-account)
      submitting jobs must have act-as permission on this run-as account. If
      unspecified, the Vertex AI Custom Code Service
      Agent(https://cloud.google.com/vertex-ai/docs/general/access-control#service-agents)
      for the CustomJob's project.
    network: The full name of the Compute Engine network to which the job should
      be peered. For example, projects/12345/global/networks/myVPC. Format is of
      the form projects/{project}/global/networks/{network}. Where {project} is
      a project number, as in 12345, and {network} is a network name. Private
      services access must already be configured for the network. If left
      unspecified, the job is not peered with any network.
    reserved_ip_ranges: A list of names for the reserved ip ranges under the VPC
      network that can be used for this job. If set, we will deploy the job
      within the provided ip ranges. Otherwise, the job will be deployed to any
      ip ranges under the provided VPC network.
    enable_web_access: Whether you want Vertex AI to enable [interactive shell
      access](https://cloud.google.com/vertex-ai/docs/training/monitor-debug-interactive-shell)
      to training containers. If set to `true`, you can access interactive
      shells at the URIs given by [CustomJob.web_access_uris][].
    encryption_spec_key_name: Customer-managed encryption key options for the
      CustomJob. If this is set, then all resources created by the CustomJob
      will be encrypted with the provided encryption key.

  Returns:
    CustomJob payload dictionary.
  """
  payload = {
      'display_name': str(display_name),
      'job_spec': {
          'worker_pool_specs': [{
              'replica_count': '1',
              'machine_spec': {
                  'machine_type': str(machine_type),
                  'accelerator_type': str(accelerator_type),
                  'accelerator_count': int(accelerator_count),
              },
              'container_spec': {
                  'image_uri': image_uri,
                  'args': args,
              },
          }],
          'service_account': str(service_account),
          'network': str(network),
          'reserved_ip_ranges': reserved_ip_ranges,
          'enable_web_access': bool(enable_web_access),
      },
      'encryption_spec': {'kms_key_name': str(encryption_spec_key_name)},
  }
  return payload
