import json
from typing import *

from kfp import dsl
from kfp.dsl import *


@dsl.platform_component
def DataflowFlexTemplateJobOp(
    container_spec_gcs_path: str,
    gcp_resources: dsl.OutputPath(str),
    location: str = 'us-central1',
    job_name: str = '',
    parameters: Dict[str, str] = {},
    launch_options: Dict[str, str] = {},
    num_workers: int = 0,
    max_workers: int = 0,
    service_account_email: str = '',
    temp_location: str = '',
    machine_type: str = '',
    additional_experiments: List[str] = [],
    network: str = '',
    subnetwork: str = '',
    additional_user_labels: Dict[str, str] = {},
    kms_key_name: str = '',
    ip_configuration: str = '',
    worker_region: str = '',
    worker_zone: str = '',
    enable_streaming_engine: bool = False,
    flexrs_goal: str = '',
    staging_location: str = '',
    sdk_container_image: str = '',
    disk_size_gb: int = 0,
    autoscaling_algorithm: str = '',
    dump_heap_on_oom: bool = False,
    save_heap_dumps_to_gcs_path: str = '',
    launcher_machine_type: str = '',
    enable_launcher_vm_serial_port_logging: bool = False,
    update: bool = False,
    transform_name_mappings: Dict[str, str] = {},
    validate_only: bool = False,
    project: str = PROJECT_ID_PLACEHOLDER,
):
    return dsl.PlatformComponent(
        platform='google_cloud',
        config={
            'task_type': 'DataflowFlexTemplateJobOp',
            # https://cloud.google.com/dataflow/docs/reference/rest#rest-resource:-v1b3.projects.locations.flextemplates
            'project': project,
            'location': location,
            'outputs': {
                'gcp_resources': gcp_resources
            },
            'body': {
                'launch_parameter': {
                    'job_name': job_name,
                    'container_spec_gcs_path': container_spec_gcs_path,
                    'parameters': parameters,
                    'launch_options': launch_options,
                    'environment': {
                        'num_workers':
                            num_workers,
                        'max_workers':
                            max_workers,
                        'service_account_email':
                            service_account_email,
                        'temp_location':
                            temp_location,
                        'machine_type':
                            machine_type,
                        'additional_experiments':
                            additional_experiments,
                        'network':
                            network,
                        'subnetwork':
                            subnetwork,
                        'additional_user_labels':
                            additional_user_labels,
                        'kms_key_name':
                            kms_key_name,
                        'ip_configuration':
                            ip_configuration,
                        'worker_region':
                            worker_region,
                        'worker_zone':
                            worker_zone,
                        'enable_streaming_engine':
                            enable_streaming_engine,
                        'flexrs_goal':
                            flexrs_goal,
                        'staging_location':
                            staging_location,
                        'sdk_container_image':
                            sdk_container_image,
                        'disk_size_gb':
                            disk_size_gb,
                        'autoscaling_algorithm':
                            autoscaling_algorithm,
                        'dump_heap_on_oom':
                            dump_heap_on_oom,
                        'save_heap_dumps_to_gcs_path':
                            save_heap_dumps_to_gcs_path,
                        'launcher_machine_type':
                            launcher_machine_type,
                        'enable_launcher_vm_serial_port_logging':
                            enable_launcher_vm_serial_port_logging
                    },
                    'update': update,
                    'transform_name_mappings': transform_name_mappings
                },
                'validate_only': validate_only
            }
        },
    )


@dsl.container_component
def ModelGetOp(
    model: dsl.Output[VertexModel],
    model_name: str,
    project: str = PROJECT_ID_PLACEHOLDER,
    location: str = 'us-central1',
):
    return dsl.PlatformComponent(
        platform='google_cloud',
        config={
            'task_type': 'ModelGetOp',
            'project': project,
            'location': location,
            'body': {
                'name': {
                    f'projects/{project}/locations/{location}/models/{model_name}'
                }
            },
            'outputs': {
                'model': model
            },
        })


# 1: return full model
# cons:
# - asymmetrtical interface: curated set of inputs, but full blob output
# - breaking change for return

# 2: return select fields
# cons:
# - expressiveness limitations? need to express name/URI/metadata declaratively

# 3: return full
# cons: curated set of inputs, but full blob output


@dsl.container_component
def ModelGetOp(
    model: dsl.Output[VertexModel],
    model_name: str,
    project: str = PROJECT_ID_PLACEHOLDER,
    location: str = 'us-central1',
):
    # use $response to represent the response variable to which the CEL is applied
    name = model.name
    uri = f'https://{location}-aiplatform.googleapis.com/v1/ + $response.name'
    metadata = {'resourceName': '$response.name'}
    return dsl.PlatformComponent(
        platform='google_cloud',
        config={
            'task_type':
                'http',
            'method':
                'GET',
            'endpoint':
                f'https://{location}-aiplatform.googleapis.com/v1/projects/{project}/locations/{location}/models/{model_name}',
            'outputs': {
                'parameters': {
                    'example_param': {
                        'destination': model,
                        'cel': '$response.name',
                    },
                    'artifacts': {
                        'model': [{
                            'name': name,
                            'uri': uri,
                            'metadata': metadata,
                        }]
                    }
                }
            },
        })


@dsl.platform_component
def DataflowFlexTemplateJobOp(
    container_spec_gcs_path: str,
    # unused
    gcp_resources: dsl.OutputPath(str),
    location: str = 'us-central1',
    job_name: str = '',
    parameters: Dict[str, str] = {},
    launch_options: Dict[str, str] = {},
    num_workers: int = 0,
    max_workers: int = 0,
    service_account_email: str = '',
    temp_location: str = '',
    machine_type: str = '',
    additional_experiments: List[str] = [],
    network: str = '',
    subnetwork: str = '',
    additional_user_labels: Dict[str, str] = {},
    kms_key_name: str = '',
    ip_configuration: str = '',
    worker_region: str = '',
    worker_zone: str = '',
    enable_streaming_engine: bool = False,
    flexrs_goal: str = '',
    staging_location: str = '',
    sdk_container_image: str = '',
    disk_size_gb: int = 0,
    autoscaling_algorithm: str = '',
    dump_heap_on_oom: bool = False,
    save_heap_dumps_to_gcs_path: str = '',
    launcher_machine_type: str = '',
    enable_launcher_vm_serial_port_logging: bool = False,
    update: bool = False,
    transform_name_mappings: Dict[str, str] = {},
    validate_only: bool = False,
    project: str = PROJECT_ID_PLACEHOLDER,
):
    return dsl.PlatformComponent(
        platform='vertex',
        config={
            'task_type':
                'http',
            # https://cloud.google.com/dataflow/docs/reference/rest#rest-resource:-v1b3.projects.locations.flextemplates
            'url':
                f'https://dataflow.googleapis.com/v1b3/projects/{project}/locations/{location}',
            'body': {
                'launch_parameter': {
                    'job_name': job_name,
                    'container_spec_gcs_path': container_spec_gcs_path,
                    'parameters': parameters,
                    'launch_options': launch_options,
                    'environment': {
                        'num_workers':
                            num_workers,
                        'max_workers':
                            max_workers,
                        'service_account_email':
                            service_account_email,
                        'temp_location':
                            temp_location,
                        'machine_type':
                            machine_type,
                        'additional_experiments':
                            additional_experiments,
                        'network':
                            network,
                        'subnetwork':
                            subnetwork,
                        'additional_user_labels':
                            additional_user_labels,
                        'kms_key_name':
                            kms_key_name,
                        'ip_configuration':
                            ip_configuration,
                        'worker_region':
                            worker_region,
                        'worker_zone':
                            worker_zone,
                        'enable_streaming_engine':
                            enable_streaming_engine,
                        'flexrs_goal':
                            flexrs_goal,
                        'staging_location':
                            staging_location,
                        'sdk_container_image':
                            sdk_container_image,
                        'disk_size_gb':
                            disk_size_gb,
                        'autoscaling_algorithm':
                            autoscaling_algorithm,
                        'dump_heap_on_oom':
                            dump_heap_on_oom,
                        'save_heap_dumps_to_gcs_path':
                            save_heap_dumps_to_gcs_path,
                        'launcher_machine_type':
                            launcher_machine_type,
                        'enable_launcher_vm_serial_port_logging':
                            enable_launcher_vm_serial_port_logging
                    },
                    'update': update,
                    'transform_name_mappings': transform_name_mappings
                },
                'validate_only': validate_only
            },
            'outputs': {
                'gcp_resources': {
                    # backend recursively resolves CEL from response and writes to gcp_resources
                    # http://google3/third_party/py/google_cloud_pipeline_components/google_cloud_pipeline_components/proto/gcp_resources.proto;l=7-25;rcl=421120500
                    # but this is still incomplete... how would a component author instruct the backend to write the error if it occurs?
                    # too much expressiveness required client-side
                    'resources': [{
                        '"https://dataflow.googleapis.com/v1b3/projects/" + $response.job.projectId + "/locations/" + $response.job.stepsLocation + "/jobs/" + $response.job.id'
                    }]
                }
            },
        },
    )


# -- challenges --
# backend doesn't really know how to construct the resource url for gcp_resources in a general fashion --> outputs are generally weird... how should gcp_resources be used?
# no obvious way to instruct the backend to parse the body to create outputs


@kfp.platforms.platform_component
def TuningOp(
    model_template: str,
    finetuning_steps: int,
    inputs_length: int,
    targets_length: int,
    accelerator_count: int = 8,
    replica_count: int = 1,
    gcp_resources: dsl.OutputPath(str),
    saved_model: dsl.Output[dsl.Artifact],
    project: str,
    location: str = 'us-central1',
    accelerator_type: str = 'TPU_V2',
    machine_type: str = 'cloud-tpu',
):
    return kfp.platforms.PlatformComponent(
        platform='google_cloud',
        config={
            'project': project,
            'location': location,
            'tuning_op': {
                # in practice this will not be a flat struct
                'model_template': model_template,
                'finetuning_steps': finetuning_steps,
                'inputs_length': inputs_length,
                'targets_length': targets_length,
                'accelerator_count': accelerator_count,
                'replica_count': replica_count,
                'accelerator_type': accelerator_type,
                'machine_type': machine_type,
            },
            'outputs': {
                'gcp_resources': gcp_resources,
                'saved_model': saved_model,
                'saved_model': saved_model,
            },
            # include version, since is no longer provided by the GCPC image tag
            'version': gcpc.__version__,
        })
