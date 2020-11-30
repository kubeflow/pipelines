# Copyright 2019 Google LLC
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
import os
import sys
import ml_metadata
from time import sleep
from ml_metadata.proto import metadata_store_pb2
from ml_metadata.metadata_store import metadata_store


def value_to_mlmd_value(value) -> metadata_store_pb2.Value:
    if value is None:
        return metadata_store_pb2.Value()
    if isinstance(value, int):
        return metadata_store_pb2.Value(int_value=value)
    if isinstance(value, float):
        return metadata_store_pb2.Value(double_value=value)
    return metadata_store_pb2.Value(string_value=str(value))


def connect_to_mlmd() -> metadata_store.MetadataStore:
    metadata_service_host = os.environ.get(
        'METADATA_GRPC_SERVICE_SERVICE_HOST', 'metadata-grpc-service')
    metadata_service_port = int(os.environ.get(
        'METADATA_GRPC_SERVICE_SERVICE_PORT', 8080))

    mlmd_connection_config = metadata_store_pb2.MetadataStoreClientConfig(
        host=metadata_service_host,
        port=metadata_service_port,
    )

    # Checking the connection to the Metadata store.
    for _ in range(100):
        try:
            mlmd_store = metadata_store.MetadataStore(mlmd_connection_config)
            # All get requests fail when the DB is empty, so we have to use a put request.
            # TODO: Replace with _ = mlmd_store.get_context_types() when https://github.com/google/ml-metadata/issues/28 is fixed
            _ = mlmd_store.put_execution_type(
                metadata_store_pb2.ExecutionType(
                    name="DummyExecutionType",
                )
            )
            return mlmd_store
        except Exception as e:
            print('Failed to access the Metadata store. Exception: "{}"'.format(str(e)), file=sys.stderr)
            sys.stderr.flush()
            sleep(1)

    raise RuntimeError('Could not connect to the Metadata store.')


def get_or_create_artifact_type(store, type_name, properties: dict = None) -> metadata_store_pb2.ArtifactType:
    try:
        artifact_type = store.get_artifact_type(type_name=type_name)
        return artifact_type
    except:
        artifact_type = metadata_store_pb2.ArtifactType(
            name=type_name,
            properties=properties,
        )
        artifact_type.id = store.put_artifact_type(artifact_type) # Returns ID
        return artifact_type


def get_or_create_execution_type(store, type_name, properties: dict = None) -> metadata_store_pb2.ExecutionType:
    try:
        execution_type = store.get_execution_type(type_name=type_name)
        return execution_type
    except:
        execution_type = metadata_store_pb2.ExecutionType(
            name=type_name,
            properties=properties,
        )
        execution_type.id = store.put_execution_type(execution_type) # Returns ID
        return execution_type


def get_or_create_context_type(store, type_name, properties: dict = None) -> metadata_store_pb2.ContextType:
    try:
        context_type = store.get_context_type(type_name=type_name)
        return context_type
    except:
        context_type = metadata_store_pb2.ContextType(
            name=type_name,
            properties=properties,
        )
        context_type.id = store.put_context_type(context_type) # Returns ID
        return context_type


def create_artifact_with_type(
    store,
    uri: str,
    type_name: str,
    properties: dict = None,
    type_properties: dict = None,
    custom_properties: dict = None,
) -> metadata_store_pb2.Artifact:
    artifact_type = get_or_create_artifact_type(
        store=store,
        type_name=type_name,
        properties=type_properties,
    )
    artifact = metadata_store_pb2.Artifact(
        uri=uri,
        type_id=artifact_type.id,
        properties=properties,
        custom_properties=custom_properties,
    )
    artifact.id = store.put_artifacts([artifact])[0]
    return artifact


def create_execution_with_type(
    store,
    type_name: str,
    properties: dict = None,
    type_properties: dict = None,
    custom_properties: dict = None,
) -> metadata_store_pb2.Execution:
    execution_type = get_or_create_execution_type(
        store=store,
        type_name=type_name,
        properties=type_properties,
    )
    execution = metadata_store_pb2.Execution(
        type_id=execution_type.id,
        properties=properties,
        custom_properties=custom_properties,
    )
    execution.id = store.put_executions([execution])[0]
    return execution


def create_context_with_type(
    store,
    context_name: str,
    type_name: str,
    properties: dict = None,
    type_properties: dict = None,
    custom_properties: dict = None,
) -> metadata_store_pb2.Context:
    # ! Context_name must be unique
    context_type = get_or_create_context_type(
        store=store,
        type_name=type_name,
        properties=type_properties,
    )
    context = metadata_store_pb2.Context(
        name=context_name,
        type_id=context_type.id,
        properties=properties,
        custom_properties=custom_properties,
    )
    context.id = store.put_contexts([context])[0]
    return context


import functools
@functools.lru_cache(maxsize=128)
def get_context_by_name(
    store,
    context_name: str,
) -> metadata_store_pb2.Context:
    matching_contexts = [context for context in store.get_contexts() if context.name == context_name]
    assert len(matching_contexts) <= 1
    if len(matching_contexts) == 0:
        raise ValueError('Context with name "{}" was not found'.format(context_name))
    return matching_contexts[0]


def get_or_create_context_with_type(
    store,
    context_name: str,
    type_name: str,
    properties: dict = None,
    type_properties: dict = None,
    custom_properties: dict = None,
) -> metadata_store_pb2.Context:
    try:
        context = get_context_by_name(store, context_name)
    except:
        context = create_context_with_type(
            store=store,
            context_name=context_name,
            type_name=type_name,
            properties=properties,
            type_properties=type_properties,
            custom_properties=custom_properties,
        )
        return context

    # Verifying that the context has the expected type name
    context_types = store.get_context_types_by_id([context.type_id])
    assert len(context_types) == 1
    if context_types[0].name != type_name:
        raise RuntimeError('Context "{}" was found, but it has type "{}" instead of "{}"'.format(context_name, context_types[0].name, type_name))
    return context


def create_new_execution_in_existing_context(
    store,
    execution_type_name: str,
    context_id: int,
    properties: dict = None,
    execution_type_properties: dict = None,
    custom_properties: dict = None,
) -> metadata_store_pb2.Execution:
    execution = create_execution_with_type(
        store=store,
        properties=properties,
        custom_properties=custom_properties,
        type_name=execution_type_name,
        type_properties=execution_type_properties,
    )
    association = metadata_store_pb2.Association(
        execution_id=execution.id,
        context_id=context_id,
    )

    store.put_attributions_and_associations([], [association])
    return execution


RUN_CONTEXT_TYPE_NAME = "KfpRun"
KFP_EXECUTION_TYPE_NAME_PREFIX = 'components.'

ARTIFACT_IO_NAME_PROPERTY_NAME = "name"
EXECUTION_COMPONENT_ID_PROPERTY_NAME = "component_id"# ~= Task ID

#TODO: Get rid of these when https://github.com/tensorflow/tfx/issues/905 and https://github.com/kubeflow/pipelines/issues/2562 are fixed
ARTIFACT_PIPELINE_NAME_PROPERTY_NAME = "pipeline_name"
EXECUTION_PIPELINE_NAME_PROPERTY_NAME = "pipeline_name"
CONTEXT_PIPELINE_NAME_PROPERTY_NAME = "pipeline_name"
ARTIFACT_RUN_ID_PROPERTY_NAME = "run_id"
EXECUTION_RUN_ID_PROPERTY_NAME = "run_id"
CONTEXT_RUN_ID_PROPERTY_NAME = "run_id"

KFP_POD_NAME_EXECUTION_PROPERTY_NAME = 'kfp_pod_name'

ARTIFACT_ARGO_ARTIFACT_PROPERTY_NAME = 'argo_artifact'


def get_or_create_run_context(
    store,
    run_id: str,
) -> metadata_store_pb2.Context:
    context = get_or_create_context_with_type(
        store=store,
        context_name=run_id,
        type_name=RUN_CONTEXT_TYPE_NAME,
        type_properties={
            CONTEXT_PIPELINE_NAME_PROPERTY_NAME: metadata_store_pb2.STRING,
            CONTEXT_RUN_ID_PROPERTY_NAME: metadata_store_pb2.STRING,
        },
        properties={
            CONTEXT_PIPELINE_NAME_PROPERTY_NAME: metadata_store_pb2.Value(string_value=run_id),
            CONTEXT_RUN_ID_PROPERTY_NAME: metadata_store_pb2.Value(string_value=run_id),
        },
    )
    return context


def create_new_execution_in_existing_run_context(
    store,
    execution_type_name: str,
    context_id: int,
    pod_name: str,
    # TODO: Remove when UX stops relying on thsese properties
    pipeline_name: str = None,
    run_id: str = None,
    instance_id: str = None,
    custom_properties = None,
) -> metadata_store_pb2.Execution:
    pipeline_name = pipeline_name or 'Context_' + str(context_id) + '_pipeline'
    run_id = run_id or 'Context_' + str(context_id) + '_run'
    instance_id = instance_id or execution_type_name
    mlmd_custom_properties = {}
    for property_name, property_value in (custom_properties or {}).items():
        mlmd_custom_properties[property_name] = value_to_mlmd_value(property_value)
    mlmd_custom_properties[KFP_POD_NAME_EXECUTION_PROPERTY_NAME] = metadata_store_pb2.Value(string_value=pod_name)
    return create_new_execution_in_existing_context(
        store=store,
        execution_type_name=execution_type_name,
        context_id=context_id,
        execution_type_properties={
            EXECUTION_PIPELINE_NAME_PROPERTY_NAME: metadata_store_pb2.STRING,
            EXECUTION_RUN_ID_PROPERTY_NAME: metadata_store_pb2.STRING,
            EXECUTION_COMPONENT_ID_PROPERTY_NAME: metadata_store_pb2.STRING,
        },
        # TODO: Remove when UX stops relying on thsese properties
        properties={
            EXECUTION_PIPELINE_NAME_PROPERTY_NAME: metadata_store_pb2.Value(string_value=pipeline_name), # Mistakenly used for grouping in the UX
            EXECUTION_RUN_ID_PROPERTY_NAME: metadata_store_pb2.Value(string_value=run_id),
            EXECUTION_COMPONENT_ID_PROPERTY_NAME: metadata_store_pb2.Value(string_value=instance_id), # should set to task ID, not component ID
        },
        custom_properties=mlmd_custom_properties,
    )


def create_new_artifact_event_and_attribution(
    store,
    execution_id: int,
    context_id: int,
    uri: str,
    type_name: str,
    event_type: metadata_store_pb2.Event.Type,
    properties: dict = None,
    artifact_type_properties: dict = None,
    custom_properties: dict = None,
    artifact_name_path: metadata_store_pb2.Event.Path = None,
    milliseconds_since_epoch: int = None,
) -> metadata_store_pb2.Artifact:
    artifact = create_artifact_with_type(
        store=store,
        uri=uri,
        type_name=type_name,
        type_properties=artifact_type_properties,
        properties=properties,
        custom_properties=custom_properties,
    )
    event = metadata_store_pb2.Event(
        execution_id=execution_id,
        artifact_id=artifact.id,
        type=event_type,
        path=artifact_name_path,
        milliseconds_since_epoch=milliseconds_since_epoch,
    )
    store.put_events([event])

    attribution = metadata_store_pb2.Attribution(
        context_id=context_id,
        artifact_id=artifact.id,
    )
    store.put_attributions_and_associations([attribution], [])

    return artifact


def link_execution_to_input_artifact(
    store,
    execution_id: int,
    uri: str,
    input_name: str,
) -> metadata_store_pb2.Artifact:
    artifacts = store.get_artifacts_by_uri(uri)
    if len(artifacts) == 0:
        print('Error: Not found upstream artifact with URI={}.'.format(uri), file=sys.stderr)
        return None
    if len(artifacts) > 1:
        print('Error: Found multiple artifacts with the same URI. {} Using the last one..'.format(artifacts), file=sys.stderr)

    artifact = artifacts[-1]

    event = metadata_store_pb2.Event(
        execution_id=execution_id,
        artifact_id=artifact.id,
        type=metadata_store_pb2.Event.INPUT,
        path=metadata_store_pb2.Event.Path(
            steps=[
                metadata_store_pb2.Event.Path.Step(
                    key=input_name,
                ),
            ]
        ),
    )
    store.put_events([event])
    return artifact


def create_new_output_artifact(
    store,
    execution_id: int,
    context_id: int,
    uri: str,
    type_name: str,
    output_name: str,
    run_id: str = None,
    argo_artifact: dict = None,
) -> metadata_store_pb2.Artifact:
    custom_properties = {
        ARTIFACT_IO_NAME_PROPERTY_NAME: metadata_store_pb2.Value(string_value=output_name),
    }
    if run_id:
        custom_properties[ARTIFACT_PIPELINE_NAME_PROPERTY_NAME] = metadata_store_pb2.Value(string_value=str(run_id))
        custom_properties[ARTIFACT_RUN_ID_PROPERTY_NAME] = metadata_store_pb2.Value(string_value=str(run_id))
    if argo_artifact:
        custom_properties[ARTIFACT_ARGO_ARTIFACT_PROPERTY_NAME] = metadata_store_pb2.Value(string_value=json.dumps(argo_artifact, sort_keys=True))
    return create_new_artifact_event_and_attribution(
        store=store,
        execution_id=execution_id,
        context_id=context_id,
        uri=uri,
        type_name=type_name,
        event_type=metadata_store_pb2.Event.OUTPUT,
        artifact_name_path=metadata_store_pb2.Event.Path(
            steps=[
                metadata_store_pb2.Event.Path.Step(
                    key=output_name,
                    #index=0,
                ),
            ]
        ),
        custom_properties=custom_properties,
        #milliseconds_since_epoch=int(datetime.now(timezone.utc).timestamp() * 1000), # Happens automatically
    )
