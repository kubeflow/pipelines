# Copyright 2021 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import json
import logging
import os
import time
import random
from dataclasses import dataclass, asdict
from pprint import pprint
from typing import Dict, List, Callable, Optional
from google.protobuf.json_format import MessageToDict

import kfp
import kfp_server_api
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
            argo_workflow_name = argo_workflow.get('metadata').get('name')
            print(f'argo workflow name: {argo_workflow_name}')
            case.verify_func(
                run=run_detail.run,
                run_detail=run_detail,
                run_id=run_detail.run.id,
                mlmd_connection_config=mlmd_connection_config,
                argo_workflow_name=argo_workflow_name,
            )
        print('OK: all test cases passed!')

    _run_test(test_wrapper)


def _retry_with_backoff(fn: Callable, retries=5, backoff_in_seconds=1):
    i = 0
    while True:
        try:
            return fn()
        except Exception as e:
            if i >= retries:
                print(f"Failed after {retries} retries:")
                raise
            else:
                print(e)
                sleep = (backoff_in_seconds * 2**i + random.uniform(0, 1))
                print("  Retry after ", str(sleep) + "s")
                time.sleep(sleep)
                i += 1


def _run_test(callback):

    def main(
        output_directory: Optional[str] = None,  # example
        host: Optional[str] = None,
        external_host: Optional[str] = None,
        launcher_image: Optional[str] = None,
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
        if launcher_image is None:
            launcher_image = os.getenv('KFP_LAUNCHER_IMAGE')

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

            def _create_run():
                return client.create_run_from_pipeline_func(
                    pipeline_func,
                    mode=mode,
                    arguments={
                        **extra_arguments,
                        **arguments,
                    },
                    launcher_image=launcher_image,
                    experiment_name=experiment,
                )

            run_result = _retry_with_backoff(fn=_create_run)
            print("Run details page URL:")
            print(f"{external_host}/#/runs/details/{run_result.run_id}")
            run_detail = run_result.wait_for_run_completion(20 * MINUTE)
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

        # When running locally, port forward MLMD grpc service to localhost:8080 by:
        #
        # ```bash
        # NAMESPACE=kubeflow kubectl port-forward svc/metadata-grpc-service 8080:8080 -n $NAMESPACE
        # ```
        #
        # Then you can uncomment the following config instead.
        # mlmd_connection_config = metadata_store_pb2.MetadataStoreClientConfig(
        #     host='localhost',
        #     port=8080,
        # )
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
class KfpArtifact:
    name: str
    uri: str
    type: str
    metadata: dict

    @classmethod
    def new(
        cls,
        mlmd_artifact: metadata_store_pb2.Artifact,
        mlmd_artifact_type: metadata_store_pb2.ArtifactType,
        mlmd_event: metadata_store_pb2.Event,
    ):
        # event path is conceptually input/output name in a task
        # ref: https://github.com/google/ml-metadata/blob/78ea886c18979d79f3c224092245873474bfafa2/ml_metadata/proto/metadata_store.proto#L169-L180
        artifact_name = mlmd_event.path.steps[0].key
        # The original field is custom_properties, but MessageToDict converts it
        # to customProperties.
        metadata = MessageToDict(mlmd_artifact).get('customProperties', {})
        return cls(
            name=artifact_name,
            type=mlmd_artifact_type.name,
            uri=mlmd_artifact.uri,
            metadata=metadata
        )


@dataclass
class TaskInputs:
    parameters: dict
    artifacts: List[KfpArtifact]


@dataclass
class TaskOutputs:
    parameters: dict
    artifacts: List[KfpArtifact]


@dataclass
class KfpTask:
    '''A KFP runtime task'''
    name: str
    type: str
    inputs: TaskInputs
    outputs: TaskOutputs

    def get_dict(self):
        d = asdict(self)
        # remove uri, because they are not deterministic
        for artifact in d.get('inputs').get('artifacts'):
            artifact.pop('uri')
        for artifact in d.get('outputs').get('artifacts'):
            artifact.pop('uri')
        return d

    @classmethod
    def new(
        cls,
        context: metadata_store_pb2.Context,
        execution: metadata_store_pb2.Execution,
        execution_types_by_id,  # dict[int, metadata_store_pb2.ExecutionType]
        events_by_execution_id,  # dict[int, List[metadata_store_pb2.Event]]
        artifacts_by_id,  # dict[int, metadata_store_pb2.Artifact]
        artifact_types_by_id,  # dict[int, metadata_store_pb2.ArtifactType]
    ):
        execution_type = execution_types_by_id[execution.type_id]
        params = _parse_parameters(execution)
        events = events_by_execution_id.get(execution.id, [])
        input_artifacts = []
        output_artifacts = []
        if events:
            input_artifacts_info = [(e.artifact_id, e)
                                    for e in events
                                    if e.type == metadata_store_pb2.Event.INPUT]
            output_artifacts_info = [
                (e.artifact_id, e)
                for e in events
                if e.type == metadata_store_pb2.Event.OUTPUT
            ]

            def kfp_artifact(
                aid: int, e: metadata_store_pb2.Event
            ) -> KfpArtifact:
                mlmd_artifact = artifacts_by_id[aid]
                mlmd_type = artifact_types_by_id[mlmd_artifact.type_id]
                return KfpArtifact.new(
                    mlmd_artifact=mlmd_artifact,
                    mlmd_artifact_type=mlmd_type,
                    mlmd_event=e,
                )

            input_artifacts = [
                kfp_artifact(aid, e) for (aid, e) in input_artifacts_info
            ]
            input_artifacts.sort(key=lambda a: a.name)
            output_artifacts = [
                kfp_artifact(aid, e) for (aid, e) in output_artifacts_info
            ]
            output_artifacts.sort(key=lambda a: a.name)

        return cls(
            name=execution.custom_properties.get('task_name').string_value,
            type=execution_type.name,
            inputs=TaskInputs(
                parameters=params['inputs'], artifacts=input_artifacts
            ),
            outputs=TaskOutputs(
                parameters=params['outputs'], artifacts=output_artifacts
            ),
        )


class KfpMlmdClient:

    def __init__(
        self,
        mlmd_connection_config: Optional[
            metadata_store_pb2.MetadataStoreClientConfig] = None,
    ):
        if mlmd_connection_config is None:
            # default to value suitable for local testing
            mlmd_connection_config = metadata_store_pb2.MetadataStoreClientConfig(
                host='localhost',
                port=8080,
            )
        self.mlmd_store = metadata_store.MetadataStore(mlmd_connection_config)

    def get_tasks(self, argo_workflow_name: str):
        run_context = self.mlmd_store.get_context_by_type_and_name(
            type_name='kfp.PipelineRun',
            context_name=argo_workflow_name,
        )
        if not run_context:
            raise Exception(
                f'Cannot find kfp.PipelineRun context "{argo_workflow_name}"'
            )
        logging.info(
            f'run_context: name={run_context.name} id={run_context.id}'
        )
        executions = self.mlmd_store.get_executions_by_context(
            context_id=run_context.id
        )
        execution_types = self.mlmd_store.get_execution_types_by_id(
            list(set([e.type_id for e in executions]))
        )
        execution_types_by_id = {et.id: et for et in execution_types}
        events = self.mlmd_store.get_events_by_execution_ids([
            e.id for e in executions
        ])
        events_by_execution_id = {}
        for e in events:
            events_by_execution_id[
                e.execution_id
            ] = (events_by_execution_id.get(e.execution_id) or []) + [e]
        artifacts = self.mlmd_store.get_artifacts_by_context(
            context_id=run_context.id
        )
        artifacts_by_id = {a.id: a for a in artifacts}
        artifact_types = self.mlmd_store.get_artifact_types_by_id(
            list(set([a.type_id for a in artifacts]))
        )
        artifact_types_by_id = {at.id: at for at in artifact_types}
        _validate_executions_have_task_names(executions)
        tasks = [
            KfpTask.new(
                context=run_context,
                execution=e,
                execution_types_by_id=execution_types_by_id,
                events_by_execution_id=events_by_execution_id,
                artifacts_by_id=artifacts_by_id,
                artifact_types_by_id=artifact_types_by_id,
            ) for e in executions
        ]
        tasks_by_name = {t.name: t for t in tasks}
        return tasks_by_name


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
        if value.HasField('string_value'):
            raw_value = value.string_value
        if value.HasField('int_value'):
            raw_value = value.int_value
        if value.HasField('double_value'):
            raw_value = value.double_value
        if name.startswith('input:'):
            parameters['inputs'][name[len('input:'):]] = raw_value
        if name.startswith('output:'):
            parameters['outputs'][name[len('output:'):]] = raw_value
    return parameters
