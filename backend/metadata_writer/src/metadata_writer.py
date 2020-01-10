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

import hashlib
import os
import kubernetes

import ml_metadata
from ml_metadata.proto import metadata_store_pb2
from ml_metadata.metadata_store import metadata_store


namespace_to_watch = os.environ.get('NAMESPACE_TO_WATCH', 'default')


try:
    kubernetes.config.load_incluster_config()
except Exception as e:
    print(e)
    try:
        kubernetes.config.load_kube_config()
    except Exception as e:
        print(e)

k8s_api = kubernetes.client.CoreV1Api()
k8s_watch = kubernetes.watch.Watch()

#%%

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
    )
    artifact.id = store.put_artifacts([artifact])[0]
    return artifact


def create_execution_with_type(
    store,
    type_name: str,
    properties: dict = None,
    type_properties: dict = None,
) -> metadata_store_pb2.Execution:
    execution_type = get_or_create_execution_type(
        store=store,
        type_name=type_name,
        properties=type_properties,
    )
    execution = metadata_store_pb2.Execution(
        type_id=execution_type.id,
        properties=properties,
    )
    execution.id = store.put_executions([execution])[0]
    return execution


def create_context_with_type(
    store,
    context_name: str,
    type_name: str,
    properties: dict = None,
    type_properties: dict = None,
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
) -> metadata_store_pb2.Context:
    try:
        context = get_context_by_name(store, context_name)
        context_types = store.get_context_types_by_id([context.type_id])
        assert len(context_types) == 1
        assert context_types[0].name == type_name
        return context
    except:
        context = create_context_with_type(
            store=store,
            context_name=context_name,
            type_name=type_name,
            properties=properties,
            type_properties=type_properties,
        )
        return context


def create_new_execution_in_existing_context(
    store,
    execution_type_name: str,
    context_id: int,
    properties: dict = None,
    execution_type_properties: dict = None,
) -> metadata_store_pb2.Execution:
    execution = create_execution_with_type(
        store=store,
        properties=properties,
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
    # TODO: Remove when UX stops relying on thsese properties
    pipeline_name: str = None,
    run_id: str = None,
    instance_id: str = None,
) -> metadata_store_pb2.Execution:
    pipeline_name = pipeline_name or 'Context_' + str(context_id) + '_pipeline'
    run_id = run_id or 'Context_' + str(context_id) + '_run'
    instance_id = instance_id or execution_type_name
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
            #EXECUTION_PIPELINE_NAME_PROPERTY_NAME: metadata_store_pb2.Value(string_value=run_id), # Mistakenly used for grouping in the UX
            #EXECUTION_RUN_ID_PROPERTY_NAME: metadata_store_pb2.Value(string_value=run_id),
            EXECUTION_PIPELINE_NAME_PROPERTY_NAME: metadata_store_pb2.Value(string_value=pipeline_name), # Mistakenly used for grouping in the UX
            EXECUTION_RUN_ID_PROPERTY_NAME: metadata_store_pb2.Value(string_value=run_id),
            EXECUTION_COMPONENT_ID_PROPERTY_NAME: metadata_store_pb2.Value(string_value=instance_id), # should set to task ID, not component ID
        },
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
    artifact_name_path: metadata_store_pb2.Event.Path = None,
    milliseconds_since_epoch: int = None,
) -> metadata_store_pb2.Artifact:
    artifact = create_artifact_with_type(
        store=store,
        uri=uri,
        type_name=type_name,
        type_properties=artifact_type_properties,
        properties=properties,
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
    artifacts = store.get_artifacts_by_uri(artifact_uri)
    if len(artifacts) == 0:
        print('Warning: Not found upstream artifact with URI={}.'.format(artifact_uri))
        return None
    if len(artifacts) > 1:
        print('Warning: Found multiple artifacts with the same URI. {} Using the last one..'.format(artifacts))

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
) -> metadata_store_pb2.Artifact:
    properties = {
        ARTIFACT_IO_NAME_PROPERTY_NAME: metadata_store_pb2.Value(string_value=output_name),
    }
    if run_id:
        properties[ARTIFACT_PIPELINE_NAME_PROPERTY_NAME] = metadata_store_pb2.Value(string_value=str(run_id))
        properties[ARTIFACT_RUN_ID_PROPERTY_NAME] = metadata_store_pb2.Value(string_value=str(run_id))
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
        properties=properties,
        artifact_type_properties={
            ARTIFACT_IO_NAME_PROPERTY_NAME: metadata_store_pb2.STRING,
            ARTIFACT_PIPELINE_NAME_PROPERTY_NAME: metadata_store_pb2.STRING,
            ARTIFACT_RUN_ID_PROPERTY_NAME: metadata_store_pb2.STRING,
        },
        #milliseconds_since_epoch=int(datetime.now(timezone.utc).timestamp() * 1000), # Happens automatically
    )

# End metadata helpers


#%%
import json
patch_retries = 20
sleep_time = 0.1

from time import sleep

def patch_pod_metadata(
    namespace: str,
    pod_name: str,
    patch: dict,
    k8s_api: kubernetes.client.CoreV1Api = None,
):
    k8s_api = k8s_api or kubernetes.client.CoreV1Api()
    patch = {
        'metadata': patch
    }
    for retry in range(patch_retries):
        try:
            pod = k8s_api.patch_namespaced_pod(
                name=pod_name,
                namespace=namespace,
                body=patch,
            )
            return pod
        except Exception as e:
            print(e)
            sleep(sleep_time)


def add_pod_metadata(
    namespace: str,
    pod_name: str,
    field: str,
    key: str,
    value: str,
    k8s_api: kubernetes.client.CoreV1Api = None,
):
    metadata_patch = {
        str(field): {str(key): str(value)}
    }
    patch_pod_metadata(
        namespace=namespace,
        pod_name=pod_name,
        patch=metadata_patch,
        k8s_api=k8s_api,
    )


#%%

#Connecting to MetadataDB

import os

metadata_service_host = os.environ.get('METADATA_SERVICE_SERVICE_HOST', 'metadata-service')
metadata_service_port = int(os.environ.get('METADATA_SERVICE_SERVICE_PORT', 8080))


mlmd_connection_config = metadata_store_pb2.MetadataStoreClientConfig(
    host=metadata_service_host,
    port=metadata_service_port,
)
mlmd_store = metadata_store.MetadataStore(mlmd_connection_config)
print("Connected to the metadata store")

#%%

ARGO_OUTPUTS_ANNOTATION_KEY = 'workflows.argoproj.io/outputs'
ARGO_TEMPLATE_ANNOTATION_KEY = 'workflows.argoproj.io/template'
KFP_COMPONENT_SPEC_ANNOTATION_KEY = 'pipelines.kubeflow.org/component_spec'
METADATA_EXECUTION_ID_LABEL_KEY = 'pipelines.kubeflow.org/metadata_execution_id'
METADATA_CONTEXT_ID_LABEL_KEY = 'pipelines.kubeflow.org/metadata_context_id'
METADATA_ARTIFACT_IDS_ANNOTATION_KEY = 'pipelines.kubeflow.org/metadata_artifact_ids'
METADATA_INPUT_ARTIFACT_IDS_ANNOTATION_KEY = 'pipelines.kubeflow.org/metadata_input_artifact_ids'
METADATA_OUTPUT_ARTIFACT_IDS_ANNOTATION_KEY = 'pipelines.kubeflow.org/metadata_output_artifact_ids'

ARGO_WORKFLOW_LABEL_KEY = 'workflows.argoproj.io/workflow'
ARGO_COMPLETED_LABEL_KEY = 'workflows.argoproj.io/completed'
METADATA_WRITTEN_LABEL_KEY = 'pipelines.kubeflow.org/metadata_written'


#%%

def cleanup_pods():
    for pod in k8s_api.list_namespaced_pod(namespace=namespace_to_watch, label_selector=ARGO_WORKFLOW_LABEL_KEY).items:
        patch_pod_metadata(
            namespace=pod.metadata.namespace,
            pod_name=pod.metadata.name,
            patch={
                'annotations': {
                    METADATA_ARTIFACT_IDS_ANNOTATION_KEY: None,
                    METADATA_INPUT_ARTIFACT_IDS_ANNOTATION_KEY: None,
                    METADATA_OUTPUT_ARTIFACT_IDS_ANNOTATION_KEY: None,
                },
                'labels': {
                    METADATA_EXECUTION_ID_LABEL_KEY: None,
                    METADATA_WRITTEN_LABEL_KEY: None,
                    METADATA_CONTEXT_ID_LABEL_KEY: None,
                }
            },
        )
#%%

#cleanup_pods()


#%%

def output_name_to_argo(name: str) -> str:
    import re
    return re.sub('-+', '-', re.sub('[^-0-9a-z]+', '-', name.lower())).strip('-')


def argo_artifact_to_uri(artifact: dict) -> str:
    if 's3' in artifact:
        s3_artifact = artifact['s3']
        return 'https:/artifacts/get?source=minio&bucket={bucket}&key={key}'.format(
            bucket=s3_artifact.get('bucket', ''),
            key=s3_artifact.get('key', ''),
        )
    elif 'raw' in artifact:
        return None
    else:
        return None


def is_tfx_pod(pod) -> bool:
    main_containers = [container for container in pod.spec.containers if container.name == 'main']
    if len(main_containers) != 1:
        return False
    main_container = main_containers[0]
    return main_container.command and main_container.command[-1].endswith('tfx/orchestration/kubeflow/container_entrypoint.py')

#%%

# Caches (not expected to be persistent)
# These caches are only used to prevent race conditions. Race conditions happen because the writer can see multiple versions of K8s object before the applied labels show up.
# They are expected to be lost when restarting the service.
# The operation of the Metadata Writer remains correct even if it's getting restarted frequently. (Kubernetes only sends the latest version of resource for new watchers.)
# Technically, we could remove the objects from cache as soon as we see that our labels have been applied successfully.
pod_name_to_execution_id = {}
workflow_name_to_context_id = {}
pods_with_written_metadata = set()

while True:
    print("Start watching Kubernetes Pods created by Argo")
    for event in k8s_watch.stream(
        k8s_api.list_namespaced_pod,
        namespace=namespace_to_watch,
        #label_selector=ARGO_WORKFLOW_LABEL_KEY + ',' + '!' + METADATA_WRITTEN_LABEL_KEY,
        label_selector=ARGO_WORKFLOW_LABEL_KEY,
    ):
        try:
            obj = event['object']
            print('Kubernetes Pod event: ', event['type'], obj.metadata.name, obj.metadata.resource_version)
            if event['type'] == 'ERROR':
                print(event)

            # Logging pod changes for debugging
            try:
                import yaml
                with open('/tmp/pod_' + obj.metadata.name + '_' + obj.metadata.resource_version, 'w') as f:
                    f.write(yaml.dump(obj.to_dict()))
            except:
                pass
            assert obj.kind == 'Pod'

            if METADATA_WRITTEN_LABEL_KEY in obj.metadata.labels:
                continue

            # Skip TFX pods - they have their own metadata writers
            if is_tfx_pod(obj):
                continue

            argo_workflow_name = obj.metadata.labels[ARGO_WORKFLOW_LABEL_KEY] # Should exist due to initial filtering
            argo_template = json.loads(obj.metadata.annotations[ARGO_TEMPLATE_ANNOTATION_KEY])
            argo_template_name = argo_template['name']

            component_name = argo_template_name
            component_version = component_name
            argo_output_name_to_type = {}
            if KFP_COMPONENT_SPEC_ANNOTATION_KEY in obj.metadata.annotations:
                component_spec_text = obj.metadata.annotations[KFP_COMPONENT_SPEC_ANNOTATION_KEY]
                component_spec = json.loads(component_spec_text)
                component_spec_digest = hashlib.sha256(component_spec_text.encode()).hexdigest()
                component_name = component_spec.get('name', component_name)
                component_version = component_name + '@sha256=' + component_spec_digest
                output_name_to_type = {output['name']: output.get('type', None) for output in component_spec.get('outputs', [])}
                argo_output_name_to_type = {output_name_to_argo(k): v for k, v in output_name_to_type.items() if v}

            if obj.metadata.name in pod_name_to_execution_id:
                execution_id = pod_name_to_execution_id[obj.metadata.name]
                context_id = workflow_name_to_context_id[argo_workflow_name]
            elif METADATA_EXECUTION_ID_LABEL_KEY in obj.metadata.labels:
                execution_id = int(obj.metadata.labels[METADATA_EXECUTION_ID_LABEL_KEY])
                context_id = int(obj.metadata.labels[METADATA_CONTEXT_ID_LABEL_KEY])
                print('Found execution id: {}, context id: {} for pod {}.'.format(execution_id, context_id, obj.metadata.name))
            else:
                run_context = get_or_create_run_context(
                    store=mlmd_store,
                    run_id=argo_workflow_name, # We can switch to internal run IDs once backend starts adding them
                )

                # Adding new execution to the database
                execution = create_new_execution_in_existing_run_context(
                    store=mlmd_store,
                    context_id=run_context.id,
                    execution_type_name=KFP_EXECUTION_TYPE_NAME_PREFIX + component_version,
                    pipeline_name=argo_workflow_name,
                    run_id=argo_workflow_name,
                    instance_id=component_name,
                )

                argo_input_artifacts = argo_template.get('inputs', {}).get('artifacts', [])
                input_artifact_ids = []
                for argo_artifact in argo_input_artifacts:
                    artifact_uri = argo_artifact_to_uri(argo_artifact)
                    if not artifact_uri:
                        continue

                    input_name = argo_artifact.get('path', '') # Every artifact should have a path in Argo
                    input_artifact_path_prefix = '/tmp/inputs/'
                    input_artifact_path_postfix = '/data'
                    if input_name.startswith(input_artifact_path_prefix):
                        input_name = input_name[len(input_artifact_path_prefix):]
                    if input_name.endswith(input_artifact_path_postfix):
                        input_name = input_name[0: -len(input_artifact_path_postfix)]

                    artifact = link_execution_to_input_artifact(
                        store=mlmd_store,
                        execution_id=execution.id,
                        uri=artifact_uri,
                        input_name=input_name,
                    )
                    if artifact is None:
                        # TODO: Maybe there is a better way to handle missing upstream artifacts
                        continue

                    input_artifact_ids.append(dict(
                        id=artifact.id,
                        name=input_name,
                        uri=artifact.uri,
                    ))
                    print('Found Input Artifact: ' + str(dict(
                        input_name=input_name,
                        id=artifact.id,
                        uri=artifact.uri,
                    )))

                execution_id = execution.id
                context_id = run_context.id

                obj.metadata.labels[METADATA_EXECUTION_ID_LABEL_KEY] = execution_id
                obj.metadata.labels[METADATA_CONTEXT_ID_LABEL_KEY] = context_id

                metadata_to_add = {
                    'labels': {
                        METADATA_EXECUTION_ID_LABEL_KEY: str(execution_id),
                        METADATA_CONTEXT_ID_LABEL_KEY: str(context_id),
                    },
                    'annotations': {
                        METADATA_INPUT_ARTIFACT_IDS_ANNOTATION_KEY: json.dumps(input_artifact_ids),
                    },
                }

                patch_pod_metadata(
                    namespace=obj.metadata.namespace,
                    pod_name=obj.metadata.name,
                    patch=metadata_to_add,
                )
                pod_name_to_execution_id[obj.metadata.name] = execution_id
                workflow_name_to_context_id[argo_workflow_name] = context_id

                print('New execution id: {}, context id: {} for pod {}.'.format(execution_id, context_id, obj.metadata.name))

                print('Execution: ' + str(dict(
                    context_id=context_id,
                    context_name=argo_workflow_name,
                    execution_id=execution_id,
                    execution_name=obj.metadata.name,
                    component_name=component_name,
                )))

                # TODO: Log input parameters as execution options.
                # Unfortunately, DSL compiler loses the information about inputs and their arguments.

            if (
                obj.metadata.name not in pods_with_written_metadata
                and (
                    obj.metadata.labels.get(ARGO_COMPLETED_LABEL_KEY, 'false') == 'true'
                    or ARGO_OUTPUTS_ANNOTATION_KEY in obj.metadata.annotations
                )
            ):
                artifact_ids = []

                if ARGO_OUTPUTS_ANNOTATION_KEY in obj.metadata.annotations: # Should be present
                    argo_outputs = json.loads(obj.metadata.annotations[ARGO_OUTPUTS_ANNOTATION_KEY])
                    argo_output_artifacts = {}

                    for artifact in argo_outputs.get('artifacts', []):
                        art_name = artifact['name']
                        output_prefix = argo_template_name + '-'
                        if art_name.startswith(output_prefix):
                            art_name = art_name[len(output_prefix):]
                        argo_output_artifacts[art_name] = artifact
                    
                    output_artifacts = []
                    for name, art in argo_output_artifacts.items():
                        artifact_uri = argo_artifact_to_uri(art)
                        if not artifact_uri:
                            continue
                        artifact_type_name = argo_output_name_to_type.get(name, 'NoType') # Cannot be None or ''

                        print('Adding Output Artifact: ' + str(dict(
                            output_name=name,
                            uri=artifact_uri,
                            type=artifact_type_name,
                        )))

                        artifact = create_new_output_artifact(
                            store=mlmd_store,
                            execution_id=execution_id,
                            context_id=context_id,
                            uri=artifact_uri,
                            type_name=artifact_type_name,
                            output_name=name,
                            #run_id='Context_' + str(context_id) + '_run',
                            run_id=argo_workflow_name,
                        )

                        artifact_ids.append(dict(
                            id=artifact.id,
                            name=name,
                            uri=artifact_uri,
                            type=artifact_type_name,
                        ))

                metadata_to_add = {
                    'labels': {
                        METADATA_WRITTEN_LABEL_KEY: 'true',
                    },
                    'annotations': {
                        METADATA_OUTPUT_ARTIFACT_IDS_ANNOTATION_KEY: json.dumps(artifact_ids),
                    },
                }
                
                patch_pod_metadata(
                    namespace=obj.metadata.namespace,
                    pod_name=obj.metadata.name,
                    patch=metadata_to_add,
                )

                pods_with_written_metadata.add(obj.metadata.name)

        except Exception as e:
            import traceback
            print(traceback.format_exc())
