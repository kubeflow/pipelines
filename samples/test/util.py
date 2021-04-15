import os
import logging
import json
from pprint import pprint
import kfp
import kfp_server_api
from typing import Dict, List, Callable, Optional
from dataclasses import dataclass
from ml_metadata import metadata_store
from ml_metadata.proto import metadata_store_pb2

MINUTE = 60


# Add **kwargs, so that when new arguments are added, this doesn't fail for
# unknown arguments.
def _default_verify_func(
    run_id: int, run: kfp_server_api.ApiRun,
    mlmd_connection_config: metadata_store_pb2.MetadataStoreClientConfig,
    **kwargs
):
    assert run.status == 'Succeeded'


def NEEDS_A_FIX(run_id, run, **kwargs):
    '''confirms a sample test case is failing and it needs to be fixed '''
    assert run.status == 'Failed'


@dataclass
class TestCase:
    '''Test case for running a KFP sample'''
    pipeline_func: Callable
    mode: kfp.dsl.PipelineExecutionMode = kfp.dsl.PipelineExecutionMode.V2_COMPATIBLE
    arguments: Optional[Dict[str, str]] = None
    verify_func: Callable[[
        int, kfp_server_api.ApiRun, kfp_server_api.
        ApiRunDetail, metadata_store_pb2.MetadataStoreClientConfig
    ], None] = _default_verify_func


def run_pipeline_func(test_cases: List[TestCase]):
    """Run a pipeline function and wait for its result.

    :param pipeline_func: pipeline function to run
    :type pipeline_func: function
    """

    def test_wrapper(
        run_pipeline: Callable[[Callable, kfp.dsl.PipelineExecutionMode, dict],
                               kfp_server_api.ApiRunDetail],
        mlmd_connection_config: metadata_store_pb2.MetadataStoreClientConfig,
    ):
        for case in test_cases:
            run_detail = run_pipeline(
                pipeline_func=case.pipeline_func,
                mode=case.mode,
                arguments=case.arguments or {}
            )
            pipeline_runtime: kfp_server_api.ApiPipelineRuntime = run_detail.pipeline_runtime
            argo_workflow = json.loads(pipeline_runtime.workflow_manifest)
            case.verify_func(
                run=run_detail.run,
                run_detail=run_detail,
                run_id=run_detail.run.id,
                mlmd_connection_config=mlmd_connection_config,
                argo_workflow_name=argo_workflow.get('metadata').get('name')
            )

    _run_test(test_wrapper)


def _run_test(callback):

    def main(
        output_directory: Optional[str] = None,  # example
        host: Optional[str] = None,
        external_host: Optional[str] = None,
        launcher_image: Optional['URI'] = None,
        experiment: str = 'v2_sample_test_samples',
        metadata_service_host: Optional[str] = None,
        metadata_service_port: int = 8080,
    ):
        """Test file CLI entrypoint used by Fire.

        :param host: Hostname pipelines can access, defaults to 'http://ml-pipeline:8888'.
        :type host: str, optional
        :param external_host: External hostname users can access from their browsers.
        :type external_host: str, optional
        :param output_directory: pipeline output directory that holds intermediate
        artifacts, example gs://your-bucket/path/to/workdir.
        :type output_directory: str, optional
        :param launcher_image: override launcher image, only used in V2_COMPATIBLE mode
        :type launcher_image: URI, optional
        :param experiment: experiment the run is added to, defaults to 'v2_sample_test_samples'
        :type experiment: str, optional
        :param metadata_service_host: host for metadata grpc service, defaults to METADATA_GRPC_SERVICE_HOST or 'metadata-grpc-service'
        :type metadata_service_host: str, optional
        :param metadata_service_port: port for metadata grpc service, defaults to 8080
        :type metadata_service_port: int, optional
        """

        # Default to env values, so people can set up their env and run these
        # tests without specifying any commands.
        if host is None:
            host = os.getenv('KFP_HOST', 'http://ml-pipeline:8888')
        if external_host is None:
            external_host = host
        if output_directory is None:
            output_directory = os.getenv('KFP_OUTPUT_DIRECTORY')
        if metadata_service_host is None:
            metadata_service_host = os.getenv(
                'METADATA_GRPC_SERVICE_HOST', 'metadata-grpc-service'
            )

        client = kfp.Client(host=host)

        def run_pipeline(
            pipeline_func: Callable,
            mode: kfp.dsl.PipelineExecutionMode = kfp.dsl.PipelineExecutionMode.
            V2_COMPATIBLE,
            arguments: dict = {},
        ) -> kfp_server_api.ApiRunDetail:
            extra_arguments = {}
            if mode != kfp.dsl.PipelineExecutionMode.V1_LEGACY:
                extra_arguments = {
                    kfp.dsl.ROOT_PARAMETER_NAME: output_directory
                }
            run_result = client.create_run_from_pipeline_func(
                pipeline_func,
                mode=mode,
                arguments={
                    **extra_arguments,
                    **arguments,
                },
                launcher_image=launcher_image,
                experiment_name=experiment,
            )
            print("Run details page URL:")
            print(f"{external_host}/#/runs/details/{run_result.run_id}")
            run_detail = run_result.wait_for_run_completion(10 * MINUTE)
            # Hide detailed information for pretty printing
            workflow_spec = run_detail.run.pipeline_spec.workflow_manifest
            workflow_manifest = run_detail.pipeline_runtime.workflow_manifest
            run_detail.run.pipeline_spec.workflow_manifest = None
            run_detail.pipeline_runtime.workflow_manifest = None
            pprint(run_detail)
            # Restore workflow manifest, because test cases may use it
            run_detail.run.pipeline_spec.workflow_manifest = workflow_spec
            run_detail.pipeline_runtime.workflow_manifest = workflow_manifest
            return run_detail

        mlmd_connection_config = metadata_store_pb2.MetadataStoreClientConfig(
            host=metadata_service_host,
            port=metadata_service_port,
        )
        callback(
            run_pipeline=run_pipeline,
            mlmd_connection_config=mlmd_connection_config
        )

    import fire
    fire.Fire(main)


@dataclass
class TaskInputs:
    parameters: dict


@dataclass
class TaskOutputs:
    parameters: dict


@dataclass
class KfpTask:
    '''A KFP runtime task'''
    context: metadata_store_pb2.Context
    execution: metadata_store_pb2.Execution
    inputs: TaskInputs
    outputs: TaskOutputs


class KfpMlmdClient:

    def __init__(
        self,
        mlmd_connection_config: metadata_store_pb2.MetadataStoreClientConfig
    ):
        self.mlmd_store = metadata_store.MetadataStore(mlmd_connection_config)

    def get_tasks(self, argo_workflow_name: str):
        run_context = self.mlmd_store.get_context_by_type_and_name(
            type_name='kfp.PipelineRun',
            context_name=argo_workflow_name,
        )
        logging.info(
            f'run_context: name={run_context.name} id={run_context.id}'
        )
        execution_list = self.mlmd_store.get_executions_by_context(
            context_id=run_context.id
        )
        _validate_executions_have_task_names(execution_list)
        tasks = {
            e.custom_properties.get('task_name').string_value:
            _kfp_task(context=run_context, execution=e) for e in execution_list
        }

        return tasks


def _kfp_task(
    context: metadata_store_pb2.Execution,
    execution: metadata_store_pb2.Execution,
):
    params = _parse_parameters(execution)
    return KfpTask(
        context=context,
        execution=execution,
        inputs=TaskInputs(parameters=params['inputs']),
        outputs=TaskOutputs(parameters=params['outputs']),
    )


def _validate_executions_have_task_names(execution_list):
    executions_without_task_name = [
        e for e in execution_list
        if not e.custom_properties.get('task_name').string_value
    ]
    if executions_without_task_name:
        raise Exception(
            f'some executions are missing task_name custom property. executions:\n{executions_without_task_name}'
        )


def _parse_parameters(execution: metadata_store_pb2.Execution) -> dict:
    custom_properties = execution.custom_properties
    parameters = {'inputs': {}, 'outputs': {}}
    for item in custom_properties.items():
        (name, value) = item
        raw_value = None
        if value.string_value:
            raw_value = value.string_value
        if value.int_value:
            raw_value = value.int_value
        if name.startswith('input:'):
            parameters['inputs'][name[len('input:'):]] = raw_value
        if name.startswith('output:'):
            parameters['outputs'][name[len('output:'):]] = raw_value
    return parameters
