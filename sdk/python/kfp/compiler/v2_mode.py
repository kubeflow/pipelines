import json
from os import pipe
from typing import Callable

from kfp import dsl
from kfp.pipeline_spec import pipeline_spec_pb2
from kfp.v2 import compiler

from google.protobuf import json_format
from kubernetes import client as k8s_client

_LAUNCHER_CONTAINER = dsl.UserContainer(
    name="kfp-launcher",
    image="gcr.io/ajay-aiplatform/kfp-launcher",
    command="/bin/mount_launcher.sh",
    mirror_volume_mounts=True)


def get_pipeline_spec(
        pipeline_func: Callable) -> pipeline_spec_pb2.PipelineSpec:
    c = compiler.Compiler()
    output_directory = getattr(pipeline_func, 'output_directory', None)
    pipeline_name = getattr(pipeline_func, 'name', None)
    pipeline_job = compiler.Compiler()._create_pipeline(pipeline_func,
                                                        output_directory,
                                                        pipeline_name)
    json_spec = json_format.MessageToJson(pipeline_job.pipeline_spec)
    result = pipeline_spec_pb2.PipelineSpec()
    json_format.Parse(json_spec, result)
    # print(result)
    return result
    # return pipeline_job_pb.pipeline_spec
    # pipeline_job = json_format.MessageToDict(pipeline_job_pb.pipeline_spec)


def update_op(op: dsl.ContainerOp,
              pipeline_spec: pipeline_spec_pb2.PipelineSpec) -> None:
    op.add_init_container(_LAUNCHER_CONTAINER)
    op.add_volume(k8s_client.V1Volume(name='kfp-launcher'))
    op.add_volume_mount(
        k8s_client.V1VolumeMount(name='kfp-launcher',
                                 mount_path='/kfp-launcher'))

    old_cmd = op.command
    op.command = [
        "/kfp-launcher/launch",
        "--mlmd_server_address",
        "$(METADATA_GRPC_SERVICE_HOST)",
        "--mlmd_server_port",
        "$(METADATA_GRPC_SERVICE_PORT)",
        "--task_spec_json",
        "$(KFP_V2_TASK_SPEC)",
        "--component_spec_json",
        "$(KFP_V2_COMPONENT_SPEC)",
        "--runtime_info_json",
        "$(KFP_V2_RUNTIME_INFO)",
        "--task_name",
        op.name,
        "--pipeline_name",
        pipeline_spec.pipeline_info.name,
        "--pipeline_run_id",
        "$(WORKFLOW_ID)",
        "--pipeline_task_id",
        "$(KFP_POD_NAME)",
        "--pipeline_root",
        # dsl.PipelineParam(name='pipeline_root', value="gs://my-bucket"),
        "gs://ml-pipeline-artifacts",
    ]
    # op.arguments = old_cmd + op.arguments

    config_map_ref = k8s_client.V1ConfigMapEnvSource(
        name='metadata-grpc-configmap', optional=True)
    op.container.add_env_from(
        k8s_client.V1EnvFromSource(config_map_ref=config_map_ref))

    # print("OP {}".format(op))
    # print("Name {}: INPUTS: {}".format(op.name, op.inputs))
    # print("Name {}: OUTPUTS: {}".format(op.name, op.outputs))

    # api_proj = k8s_client.V1DownwardAPIProjection(
    #   items=[
    #     k8s_client.V1DownwardAPIVolumeFile(
    #       path="task_spec",
    #       field_ref=k8s_client.V1ObjectFieldSelector(field_path="metadata.annotations"),
    #     ),
    #   ],
    # )

    # projection = k8s_client.V1VolumeProjection(downward_api=api_proj)
    # vol = k8s_client.V1Volume(name="anno", downward_api=api_proj)
    # op.add_volume(vol)
    # op.container.add_volume_mount(k8s_client.V1VolumeMount("/kfp-v2", name="anno"))

    # def _to_json(a):
    #     from google.protobuf.struct_pb2 import Struct
    #     from google.protobuf import json_format
    #     s = Struct()
    #     json_format.ParseDict(a, s)
    #     return json_format.MessageToJson(s)

    # op = dsl.ContainerOp()
    to_json = lambda x: json_format.MessageToJson(x)
    task_name = 'task-{}'.format(op.name)
    task_spec = pipeline_spec.root.dag.tasks[task_name]
    component_name = pipeline_spec.root.dag.tasks[task_name].component_ref.name
    component_spec = pipeline_spec.components[component_name]

    deployment_spec = pipeline_spec_pb2.PipelineDeploymentConfig()
    json_spec = json_format.MessageToJson(pipeline_spec.deployment_spec)
    json_format.Parse(json_spec, deployment_spec)
    executor = deployment_spec.executors[component_spec.executor_label]
    assert executor.HasField('container')
    # print(executor.container)

    # executor_spec = pipeline_spec.

    op.arguments = list(executor.container.command) + list(
        executor.container.args)

    op.container.add_env_variable(
        k8s_client.V1EnvVar(name="KFP_V2_TASK_SPEC", value=to_json(task_spec)))
    op.container.add_env_variable(
        k8s_client.V1EnvVar(name="KFP_V2_COMPONENT_SPEC",
                            value=to_json(component_spec)))

    # input = pipeline_spec_pb2.ExecutorInput()
    runtime_info = {
        "inputParameters": {},
        "inputArtifacts": {},
        "outputParameters": {},
        "outputArtifacts": {},
    }
    for parameter, spec in component_spec.input_definitions.parameters.items():
        runtime_info["inputParameters"][parameter] = op._parameter_arguments[
            parameter]
        # input.inputs.parameters[
        #     parameter].string_value = op._parameter_arguments[parameter]

    for artifact_name, spec in component_spec.input_definitions.artifacts.items(
    ):
        print("INPUTS: ", op.input_artifact_paths)
        artifact_info = {
            "fileInputPath": op.input_artifact_paths[artifact_name]
        }
        runtime_info["inputArtifacts"][artifact_name] = artifact_info
        # runtime_info["inputArtifacts"][artifact_name] = op.artifact_arguments[
        # artifact_name]

    for parameter, spec in component_spec.output_definitions.parameters.items():
        parameter_info = {
            "parameterType":
                pipeline_spec_pb2.PrimitiveType.PrimitiveTypeEnum.Name(spec.type
                                                                      ),
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

    # print(op.file_outputs)
    # print("JSON INPUT: {}".format(json_format.MessageToJson(input)))
    print("JSON INPUT: {}".format(json.dumps(runtime_info)))

    op.container.add_env_variable(
        k8s_client.V1EnvVar(name="KFP_V2_RUNTIME_INFO",
                            value=json.dumps(runtime_info)))

    # for input in component_spec['inputDefinitions']['parameters']:
    #     print
    # op.apply(add_pod_env)
    # print(pipeline_job.pipeline_spec)

    # if pipeline_job is not None:
    #   op.pod_annotations['pipelines.kubeflow.org/v2_task_spec'] = pipeline_job
