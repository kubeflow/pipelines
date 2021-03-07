import json
from typing import Callable

from kfp import dsl
from kfp.compiler import _default_transformers
from kfp.pipeline_spec import pipeline_spec_pb2
from kfp.v2 import compiler

from google.protobuf import json_format
from kubernetes import client as k8s_client

_LAUNCHER_CONTAINER = dsl.UserContainer(name="kfp-launcher",
                                        image="gcr.io/ml-pipeline/kfp-launcher",
                                        command="/bin/mount_launcher.sh",
                                        mirror_volume_mounts=True)


# def get_pipeline_spec(
#     pipeline_func: Callable) -> pipeline_spec_pb2.PipelineSpec:
#   """Computes PipelineSpec from the specified pipeline function.

#     Args:
#       pipeline_func: A function defining a KFP pipeline.

#     Returns:
#       A PipelineSpec protocol buffer.
#     """
#   output_directory = getattr(pipeline_func, 'output_directory', None)
#   pipeline_name = getattr(pipeline_func, 'name', None)
#   pipeline_job = compiler.Compiler()._create_pipeline(pipeline_func,
#                                                       output_directory,
#                                                       pipeline_name)
#   json_spec = json_format.MessageToJson(pipeline_job.pipeline_spec)
#   result = pipeline_spec_pb2.PipelineSpec()
#   json_format.Parse(json_spec, result)
#   return result


def update_op(op: dsl.ContainerOp, pipeline_name: dsl.PipelineParam,
              pipeline_root: dsl.PipelineParam) -> None:
  """Updates the passed in Op for running in v2-compatible mode.

    Args:
      op: The Op to update.
      pipeline_spec: The PipelineSpec for the pipeline under which `op`
        runs.
      pipeline_root: The root output directory for pipeline artifacts.
    """
  # Inject the launcher binary and overwrite the entrypoint.
  op.add_init_container(_LAUNCHER_CONTAINER)
  op.add_volume(k8s_client.V1Volume(name='kfp-launcher'))
  op.add_volume_mount(
      k8s_client.V1VolumeMount(name='kfp-launcher', mount_path='/kfp-launcher'))

  op.command = [
      "/kfp-launcher/launch",
      "--mlmd_server_address",
      "$(METADATA_GRPC_SERVICE_HOST)",
      "--mlmd_server_port",
      "$(METADATA_GRPC_SERVICE_PORT)",
      "--runtime_info_json",
      "$(KFP_V2_RUNTIME_INFO)",
      "--container_image",
      "$(KFP_V2_IMAGE)",
      "--task_name",
      op.name,
      "--pipeline_name",
      pipeline_name,
      "--pipeline_run_id",
      "$(WORKFLOW_ID)",
      "--pipeline_task_id",
      "$(KFP_POD_NAME)",
      "--pipeline_root",
      pipeline_root,
  ]

  # Mount necessary environment variables.
  op.apply(_default_transformers.add_kfp_pod_env)
  op.container.add_env_variable(
      k8s_client.V1EnvVar(name="KFP_V2_IMAGE", value=op.container.image))

  config_map_ref = k8s_client.V1ConfigMapEnvSource(
      name='metadata-grpc-configmap', optional=True)
  op.container.add_env_from(
      k8s_client.V1EnvFromSource(config_map_ref=config_map_ref))

  op.arguments = list(op.container_spec.command) + list(op.container_spec.args)

  runtime_info = {
      "inputParameters": {},
      "inputArtifacts": {},
      "outputParameters": {},
      "outputArtifacts": {},
  }

  component_spec = op.component_spec
  for parameter, spec in component_spec.input_definitions.parameters.items():
    parameter_info = {
        "parameterType":
            pipeline_spec_pb2.PrimitiveType.PrimitiveTypeEnum.Name(spec.type),
        "parameterValue":
            op._parameter_arguments[parameter],
    }
    runtime_info["inputParameters"][parameter] = parameter_info

  for artifact_name, spec in component_spec.input_definitions.artifacts.items():
    artifact_info = {"fileInputPath": op.input_artifact_paths[artifact_name]}
    runtime_info["inputArtifacts"][artifact_name] = artifact_info

  for parameter, spec in component_spec.output_definitions.parameters.items():
    parameter_info = {
        "parameterType":
            pipeline_spec_pb2.PrimitiveType.PrimitiveTypeEnum.Name(spec.type),
        "fileOutputPath":
            op.file_outputs[parameter],
    }
    runtime_info["outputParameters"][parameter] = parameter_info

  for artifact_name, spec in component_spec.output_definitions.artifacts.items(
  ):
    # TODO: Assert instance_schema.
    artifact_info = {
        # Type used to register output artifacts.
        "artifactSchema": spec.artifact_type.instance_schema,
        # File used to write out the registered artifact ID.
        "fileOutputPath": op.file_outputs[artifact_name],
    }
    runtime_info["outputArtifacts"][artifact_name] = artifact_info

  op.container.add_env_variable(
      k8s_client.V1EnvVar(name="KFP_V2_RUNTIME_INFO",
                          value=json.dumps(runtime_info)))

  op.pod_annotations['pipelines.kubeflow.org/v2_component'] = "true"
