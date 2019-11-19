#kubectl exec sleeper -- python3 -m pip install kubernetes 'ml-metadata==0.14' 'tensorflow>=1.15' --upgrade --quiet --user
#kubectl exec sleeper -- python3 -m pip freeze
#kubectl exec sleeper -it -- python3 -c "$(< metadata_writer.py)"
#kubectl cp metadata-writer-sleeper:/tmp ./
#kubectl exec metadata-writer-sleeper -it -- python3 -c "$(< metadata_writer.py)"

import kubernetes

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

metadata_configmap_name = 'metadata-configmap'
metadata_configmap_namespace = 'default' # Not kubeflow
metadata_db_user = 'root'
metadata_db_password = ''

#use_metadata_store = True
#if use_metadata_store:
#import site
#import subprocess
#import sys
#subprocess.run([sys.executable, '-m', 'pip', 'install', 'ml-metadata==0.14', 'tensorflow>=1.15', '--upgrade', '--quiet', '--user'])
#sys.path.append(site.getusersitepackages())

import ml_metadata

from ml_metadata.proto import metadata_store_pb2
from ml_metadata.metadata_store import metadata_store

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
) -> metadata_store_pb2.Execution:
    return create_new_execution_in_existing_context(
        store=store,
        execution_type_name=KFP_EXECUTION_TYPE_NAME_PREFIX + execution_type_name,
        context_id=context_id,
        execution_type_properties={
            EXECUTION_PIPELINE_NAME_PROPERTY_NAME: metadata_store_pb2.STRING,
            EXECUTION_RUN_ID_PROPERTY_NAME: metadata_store_pb2.STRING,
            EXECUTION_COMPONENT_ID_PROPERTY_NAME: metadata_store_pb2.STRING,
        },
        properties={
            #EXECUTION_PIPELINE_NAME_PROPERTY_NAME: metadata_store_pb2.Value(string_value=run_id), # Mistakenly used for grouping in the UX
            #EXECUTION_RUN_ID_PROPERTY_NAME: metadata_store_pb2.Value(string_value=run_id),
            EXECUTION_PIPELINE_NAME_PROPERTY_NAME: metadata_store_pb2.Value(string_value='Context_' + str(context_id) + '_pipeline'), # Mistakenly used for grouping in the UX
            EXECUTION_RUN_ID_PROPERTY_NAME: metadata_store_pb2.Value(string_value='Context_' + str(context_id) + '_run'),
            EXECUTION_COMPONENT_ID_PROPERTY_NAME: metadata_store_pb2.Value(string_value=execution_type_name), # should set to task ID, not component ID
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


def create_new_input_artifact(
    store,
    execution_id: int,
    context_id: int,
    uri: str,
    type_name: str,
    input_name: str,
    run_id: str = None,
) -> metadata_store_pb2.Artifact:
    properties = {
        ARTIFACT_IO_NAME_PROPERTY_NAME: metadata_store_pb2.Value(string_value=input_name),
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
        event_type=metadata_store_pb2.Event.INPUT,
        artifact_name_path=metadata_store_pb2.Event.Path(
            steps=[
                metadata_store_pb2.Event.Path.Step(
                    key=input_name,
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

metadata_configmap_dict = k8s_api.read_namespaced_config_map(
    name=metadata_configmap_name,
    namespace=metadata_configmap_namespace,
).data

mlmd_connection_config = metadata_store_pb2.ConnectionConfig(
    mysql=metadata_store_pb2.MySQLDatabaseConfig(
        host=metadata_configmap_dict['mysql_host'],
        port=int(metadata_configmap_dict['mysql_port']),
        database=metadata_configmap_dict['mysql_database'],
        user=metadata_db_user,
        password=metadata_db_password,
    ),
)
mlmd_store = metadata_store.MetadataStore(mlmd_connection_config)


#%%

ARGO_OUTPUTS_ANNOTATION_KEY = 'workflows.argoproj.io/outputs'
ARGO_TEMPLATE_ANNOTATION_KEY = 'workflows.argoproj.io/template'
KFP_COMPONENT_SPEC_ANNOTATION_KEY = 'pipelines.kubeflow.org/component_spec'
#KFP_PIPELINE_SPEC_ANNOTATION_KEY = 'pipelines.kubeflow.org/pipeline_spec'
#METADATA_EXECUTION_ID_ANNOTATION_KEY = 'pipelines.kubeflow.org/metadata_execution_id'
METADATA_EXECUTION_ID_LABEL_KEY = 'pipelines.kubeflow.org/metadata_execution_id'
METADATA_CONTEXT_ID_LABEL_KEY = 'pipelines.kubeflow.org/metadata_context_id'
METADATA_ARTIFACT_IDS_ANNOTATION_KEY = 'pipelines.kubeflow.org/metadata_artifact_ids'
METADATA_INPUT_ARTIFACT_IDS_ANNOTATION_KEY = 'pipelines.kubeflow.org/metadata_input_artifact_ids'
METADATA_OUTPUT_ARTIFACT_IDS_ANNOTATION_KEY = 'pipelines.kubeflow.org/metadata_output_artifact_ids'

ARGO_WORKFLOW_LABEL_KEY = 'workflows.argoproj.io/workflow'
METADATA_WRITTEN_LABEL_KEY = 'pipelines.kubeflow.org/metadata_written'


#pods_namespace = 'kubeflow' # Kubeflow deployment # FIX!!!
#pods_namespace = 'default' # Marketplace deployment # FIX!!!

#%%

def cleanup_pods():
    for pod in k8s_api.list_pod_for_all_namespaces(label_selector=ARGO_WORKFLOW_LABEL_KEY).items:
        patch_pod_metadata(
            namespace=pod.metadata.namespace,
            pod_name=pod.metadata.name,
            patch={
                'annotations': {
                    #METADATA_EXECUTION_ID_ANNOTATION_KEY: None,
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


#assert False


#%%

def output_name_to_argo(name: str) -> str:
    import re
    return re.sub('-+', '-', re.sub('[^-0-9a-z]+', '-', name.lower())).strip('-')


def argo_artifact_to_uri(artifact: dict) -> str:
    return 'https:/artifacts/get?source=minio&bucket={bucket}&key={key}'.format(
        bucket=artifact['s3']['bucket'],
        key=artifact['s3']['key'],
    )


#%%
"""
for event in k8s_watch.stream(
    k8s_api.list_pod_for_all_namespaces,
    #label_selector=ARGO_WORKFLOW_LABEL_KEY + ',' + '!' + METADATA_WRITTEN_LABEL_KEY,
    label_selector=ARGO_WORKFLOW_LABEL_KEY,
    #timeout_seconds=3,
):
    obj = event['object']
    print(event['type'], obj.metadata.name, obj.metadata.resource_version)
    try:
        import yaml
        with open('/tmp/pod_' + obj.metadata.name + '_' + obj.metadata.resource_version, 'w') as f:
            f.write(yaml.dump(obj.to_dict()))
    except:
        pass



exit(0)
"""

#%%

# Caches (not expected to be persistent)
pod_name_to_execution_id = {} # Updates happen fast. I've seen new ID being assigned to an execution 3 times.
workflow_name_to_context_id = {}
pods_with_written_metadata = set()

#while True:
for event in k8s_watch.stream(
    k8s_api.list_pod_for_all_namespaces,
    #label_selector=ARGO_WORKFLOW_LABEL_KEY + ',' + '!' + METADATA_WRITTEN_LABEL_KEY,
    label_selector=ARGO_WORKFLOW_LABEL_KEY,
    #timeout_seconds=3,
):
    try:
        obj = event['object']
        print(event['type'], obj.metadata.name, obj.metadata.resource_version)
        if event['type'] == 'ERROR':
            print(event)
        try:
            import yaml
            with open('/tmp/pod_' + obj.metadata.name + '_' + obj.metadata.resource_version, 'w') as f:
                f.write(yaml.dump(obj.to_dict()))
        except:
            pass
        assert obj.kind == 'Pod'

        if METADATA_WRITTEN_LABEL_KEY in obj.metadata.labels:
            continue

        argo_workflow_name = obj.metadata.labels[ARGO_WORKFLOW_LABEL_KEY] # Should exist due to initial filtering
        argo_template = json.loads(obj.metadata.annotations[ARGO_TEMPLATE_ANNOTATION_KEY])
        argo_template_name = argo_template['name']

        component_name = argo_template_name
        argo_output_name_to_type = {}
        if KFP_COMPONENT_SPEC_ANNOTATION_KEY in obj.metadata.annotations:
            component_spec = json.loads(obj.metadata.annotations[KFP_COMPONENT_SPEC_ANNOTATION_KEY])
            component_name = component_spec.get('name', None)
            output_name_to_type = {output['name']: output.get('type', None) for output in component_spec.get('outputs', [])}
            argo_output_name_to_type = {output_name_to_argo(k): v for k, v in output_name_to_type.items() if v}

        if obj.metadata.name in pod_name_to_execution_id:
            execution_id = pod_name_to_execution_id[obj.metadata.name]
            context_id = workflow_name_to_context_id[argo_workflow_name]

        elif METADATA_EXECUTION_ID_LABEL_KEY in obj.metadata.labels:
        #if METADATA_EXECUTION_ID_LABEL_KEY in obj.metadata.labels:
            execution_id = int(obj.metadata.labels[METADATA_EXECUTION_ID_LABEL_KEY])
            context_id = int(obj.metadata.labels[METADATA_CONTEXT_ID_LABEL_KEY])
            print('Found execution id: {}, context id: {} for pod {}.'.format(execution_id, context_id, obj.metadata.name))
        else:
            #if event['type'] != 'ADDED':
            #    continue # Only add execution ID once

            run_context = get_or_create_run_context(
                store=mlmd_store,
                run_id=argo_workflow_name, # We can switch to internal run IDs once backend starts adding them
            )

            # Adding new execution to the database
            execution = create_new_execution_in_existing_run_context(
                store=mlmd_store,
                context_id=run_context.id,
                execution_type_name=component_name,
                #?? execution_name = obj.metadata.name,
            )
            execution_id = execution.id
            context_id = run_context.id

            obj.metadata.labels[METADATA_EXECUTION_ID_LABEL_KEY] = execution_id
            obj.metadata.labels[METADATA_CONTEXT_ID_LABEL_KEY] = context_id

            metadata_to_add = {
                'labels': {
                    METADATA_EXECUTION_ID_LABEL_KEY: str(execution_id),
                    METADATA_CONTEXT_ID_LABEL_KEY: str(context_id),
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

            #TODO: Write input artifacts. Unfortunately, DSL compiler loses this information

        if obj.status.phase == 'Succeeded' and obj.metadata.name not in pods_with_written_metadata: # phase = One of Pending,Running,Succeeded,Failed,Unknown
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
                    artifact_type_name = argo_output_name_to_type.get(name, 'NoType') # Cannot be None or ''

                    print('Artifact: ' + str(dict(
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
                        run_id='Context_' + str(context_id) + '_run',
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
        #print(event)
        print(e)

""" 
avolkov@avolkov:~/_projects/pipelines_worktree2/sdk/metadata_writer$ kubectl exec sleeper -it -- python3 -c "$(< metadata_writer.py)"
ADDED tfx-pipeline-lmbw6-1947672582
New execution id: 1 for pod tfx-pipeline-lmbw6-1947672582.
{'execution_id': '1', 'output_name': 'data', 'uri': 'https:/artifacts/get?source=minio&bucket=mlpipeline&key=artifacts/tfx-pipeline-lmbw6/tfx-pipeline-lmbw6-1947672582/download-from-gcs-data.tgz', 'type': None}
ADDED tfx-pipeline-lmbw6-2473736504
New execution id: 2 for pod tfx-pipeline-lmbw6-2473736504.
{'execution_id': '2', 'output_name': 'output', 'uri': 'https:/artifacts/get?source=minio&bucket=mlpipeline&key=artifacts/tfx-pipeline-lmbw6/tfx-pipeline-lmbw6-2473736504/schemagen-output.tgz', 'type': 'Schema'}
ADDED tfx-pipeline-lmbw6-2908704420
New execution id: 3 for pod tfx-pipeline-lmbw6-2908704420.
{'execution_id': '3', 'output_name': 'output', 'uri': 'https:/artifacts/get?source=minio&bucket=mlpipeline&key=artifacts/tfx-pipeline-lmbw6/tfx-pipeline-lmbw6-2908704420/statisticsgen-output.tgz', 'type': 'ExampleStatistics'}
ADDED tfx-pipeline-lmbw6-2172955551
New execution id: 4 for pod tfx-pipeline-lmbw6-2172955551.
{'execution_id': '4', 'output_name': 'example-artifacts', 'uri': 'https:/artifacts/get?source=minio&bucket=mlpipeline&key=artifacts/tfx-pipeline-lmbw6/tfx-pipeline-lmbw6-2172955551/csvexamplegen-example-artifacts.tgz', 'type': 'ExamplesPath'}


avolkov@avolkov:~/_projects/pipelines_worktree2/sdk/metadata_writer$ kubectl exec sleeper -it -- python3 -c "$(< metadata_writer.py)"
ADDED tfx-pipeline-lmbw6-2908704420
New execution id: 1 for pod tfx-pipeline-lmbw6-2908704420.
{'execution_id': '1', 'output_name': 'output', 'uri': 'https:/artifacts/get?source=minio&bucket=mlpipeline&key=artifacts/tfx-pipeline-lmbw6/tfx-pipeline-lmbw6-2908704420/statisticsgen-output.tgz', 'type': 'ExampleStatistics'}
{'execution_id': '1', 'component_name': 'Statisticsgen', 'output_artifacts': [{'output_name': 'output', 'uri': 'https:/artifacts/get?source=minio&bucket=mlpipeline&key=artifacts/tfx-pipeline-lmbw6/tfx-pipeline-lmbw6-2908704420/statisticsgen-output.tgz', 'type': 'ExampleStatistics'}]}
ADDED tfx-pipeline-lmbw6-2172955551
New execution id: 2 for pod tfx-pipeline-lmbw6-2172955551.
{'execution_id': '2', 'output_name': 'example-artifacts', 'uri': 'https:/artifacts/get?source=minio&bucket=mlpipeline&key=artifacts/tfx-pipeline-lmbw6/tfx-pipeline-lmbw6-2172955551/csvexamplegen-example-artifacts.tgz', 'type': 'ExamplesPath'}
{'execution_id': '2', 'component_name': 'CsvExampleGen', 'output_artifacts': [{'output_name': 'example-artifacts', 'uri': 'https:/artifacts/get?source=minio&bucket=mlpipeline&key=artifacts/tfx-pipeline-lmbw6/tfx-pipeline-lmbw6-2172955551/csvexamplegen-example-artifacts.tgz', 'type': 'ExamplesPath'}]}
ADDED tfx-pipeline-lmbw6-1947672582
New execution id: 3 for pod tfx-pipeline-lmbw6-1947672582.
{'execution_id': '3', 'output_name': 'data', 'uri': 'https:/artifacts/get?source=minio&bucket=mlpipeline&key=artifacts/tfx-pipeline-lmbw6/tfx-pipeline-lmbw6-1947672582/download-from-gcs-data.tgz', 'type': None}
{'execution_id': '3', 'component_name': 'Download from GCS', 'output_artifacts': [{'output_name': 'data', 'uri': 'https:/artifacts/get?source=minio&bucket=mlpipeline&key=artifacts/tfx-pipeline-lmbw6/tfx-pipeline-lmbw6-1947672582/download-from-gcs-data.tgz', 'type': None}]}
ADDED tfx-pipeline-lmbw6-2473736504
New execution id: 4 for pod tfx-pipeline-lmbw6-2473736504.
{'execution_id': '4', 'output_name': 'output', 'uri': 'https:/artifacts/get?source=minio&bucket=mlpipeline&key=artifacts/tfx-pipeline-lmbw6/tfx-pipeline-lmbw6-2473736504/schemagen-output.tgz', 'type': 'Schema'}
{'execution_id': '4', 'component_name': 'Schemagen', 'output_artifacts': [{'output_name': 'output', 'uri': 'https:/artifacts/get?source=minio&bucket=mlpipeline&key=artifacts/tfx-pipeline-lmbw6/tfx-pipeline-lmbw6-2473736504/schemagen-output.tgz', 'type': 'Schema'}]}


# Many duplicate events and also a DELETED event:

ADDED tfx-pipeline-lngmn-813654318
New execution id: 42 for pod tfx-pipeline-lngmn-813654318.
Execution: {'context_name': 'tfx-pipeline-lngmn', 'execution_id': 42, 'execution_name': 'tfx-pipeline-lngmn-813654318', 'component_name': 'Schemagen'}
MODIFIED tfx-pipeline-lngmn-813654318
New execution id: 43 for pod tfx-pipeline-lngmn-813654318.
Execution: {'context_name': 'tfx-pipeline-lngmn', 'execution_id': 43, 'execution_name': 'tfx-pipeline-lngmn-813654318', 'component_name': 'Schemagen'}
MODIFIED tfx-pipeline-lngmn-813654318
New execution id: 44 for pod tfx-pipeline-lngmn-813654318.
Execution: {'context_name': 'tfx-pipeline-lngmn', 'execution_id': 44, 'execution_name': 'tfx-pipeline-lngmn-813654318', 'component_name': 'Schemagen'}
MODIFIED tfx-pipeline-lngmn-813654318
New execution id: 45 for pod tfx-pipeline-lngmn-813654318.
Execution: {'context_name': 'tfx-pipeline-lngmn', 'execution_id': 45, 'execution_name': 'tfx-pipeline-lngmn-813654318', 'component_name': 'Schemagen'}
MODIFIED tfx-pipeline-lngmn-813654318
New execution id: 46 for pod tfx-pipeline-lngmn-813654318.
Execution: {'context_name': 'tfx-pipeline-lngmn', 'execution_id': 46, 'execution_name': 'tfx-pipeline-lngmn-813654318', 'component_name': 'Schemagen'}
MODIFIED tfx-pipeline-lngmn-813654318
New execution id: 47 for pod tfx-pipeline-lngmn-813654318.
Execution: {'context_name': 'tfx-pipeline-lngmn', 'execution_id': 47, 'execution_name': 'tfx-pipeline-lngmn-813654318', 'component_name': 'Schemagen'}
MODIFIED tfx-pipeline-lngmn-813654318
New execution id: 48 for pod tfx-pipeline-lngmn-813654318.
Execution: {'context_name': 'tfx-pipeline-lngmn', 'execution_id': 48, 'execution_name': 'tfx-pipeline-lngmn-813654318', 'component_name': 'Schemagen'}
MODIFIED tfx-pipeline-lngmn-813654318
New execution id: 49 for pod tfx-pipeline-lngmn-813654318.
Execution: {'context_name': 'tfx-pipeline-lngmn', 'execution_id': 49, 'execution_name': 'tfx-pipeline-lngmn-813654318', 'component_name': 'Schemagen'}
Artifact: {'output_name': 'output', 'uri': 'https:/artifacts/get?source=minio&bucket=mlpipeline&key=artifacts/tfx-pipeline-lngmn/tfx-pipeline-lngmn-813654318/schemagen-output.tgz', 'type': 'Schema'}
DELETED tfx-pipeline-lngmn-813654318
New execution id: 50 for pod tfx-pipeline-lngmn-813654318.
Execution: {'context_name': 'tfx-pipeline-lngmn', 'execution_id': 50, 'execution_name': 'tfx-pipeline-lngmn-813654318', 'component_name': 'Schemagen'}
Artifact: {'output_name': 'output', 'uri': 'https:/artifacts/get?source=minio&bucket=mlpipeline&key=artifacts/tfx-pipeline-lngmn/tfx-pipeline-lngmn-813654318/schemagen-output.tgz', 'type': 'Schema'}


# Double executions + multiple modifications + double artifacts + DELETED:

ADDED tfx-pipeline-nnwhb-4013877105
New execution id: 99 for pod tfx-pipeline-nnwhb-4013877105.
Execution: {'context_name': 'tfx-pipeline-nnwhb', 'execution_id': 99, 'execution_name': 'tfx-pipeline-nnwhb-4013877105', 'component_name': 'Schemagen'}
MODIFIED tfx-pipeline-nnwhb-4013877105
New execution id: 100 for pod tfx-pipeline-nnwhb-4013877105.
Execution: {'context_name': 'tfx-pipeline-nnwhb', 'execution_id': 100, 'execution_name': 'tfx-pipeline-nnwhb-4013877105', 'component_name': 'Schemagen'}
MODIFIED tfx-pipeline-nnwhb-4013877105
Found execution id: 99 for pod tfx-pipeline-nnwhb-4013877105.
MODIFIED tfx-pipeline-nnwhb-4013877105
Found execution id: 99 for pod tfx-pipeline-nnwhb-4013877105.
MODIFIED tfx-pipeline-nnwhb-4013877105
Found execution id: 100 for pod tfx-pipeline-nnwhb-4013877105.
MODIFIED tfx-pipeline-nnwhb-4013877105
Found execution id: 100 for pod tfx-pipeline-nnwhb-4013877105.
MODIFIED tfx-pipeline-nnwhb-4013877105
Found execution id: 100 for pod tfx-pipeline-nnwhb-4013877105.
MODIFIED tfx-pipeline-nnwhb-4013877105
Found execution id: 100 for pod tfx-pipeline-nnwhb-4013877105.
MODIFIED tfx-pipeline-nnwhb-4013877105
Found execution id: 100 for pod tfx-pipeline-nnwhb-4013877105.
Artifact: {'output_name': 'output', 'uri': 'https:/artifacts/get?source=minio&bucket=mlpipeline&key=artifacts/tfx-pipeline-nnwhb/tfx-pipeline-nnwhb-4013877105/schemagen-output.tgz', 'type': 'Schema'}
DELETED tfx-pipeline-nnwhb-4013877105
Found execution id: 100 for pod tfx-pipeline-nnwhb-4013877105.
Artifact: {'output_name': 'output', 'uri': 'https:/artifacts/get?source=minio&bucket=mlpipeline&key=artifacts/tfx-pipeline-nnwhb/tfx-pipeline-nnwhb-4013877105/schemagen-output.tgz', 'type': 'Schema'}



ADDED pipeline1-gzmbd-1417507587 5854984
New execution id: 124 for pod pipeline1-gzmbd-1417507587.
Execution: {'context_name': 'pipeline1-gzmbd', 'execution_id': 124, 'execution_name': 'pipeline1-gzmbd-1417507587', 'component_name': 'Write numbers'}
MODIFIED pipeline1-gzmbd-1417507587 5854986
New execution id: 125 for pod pipeline1-gzmbd-1417507587.
Execution: {'context_name': 'pipeline1-gzmbd', 'execution_id': 125, 'execution_name': 'pipeline1-gzmbd-1417507587', 'component_name': 'Write numbers'}
MODIFIED pipeline1-gzmbd-1417507587 5854988
New execution id: 126 for pod pipeline1-gzmbd-1417507587.
Execution: {'context_name': 'pipeline1-gzmbd', 'execution_id': 126, 'execution_name': 'pipeline1-gzmbd-1417507587', 'component_name': 'Write numbers'}
MODIFIED pipeline1-gzmbd-1417507587 5854989
Found execution id: 124 for pod pipeline1-gzmbd-1417507587.
MODIFIED pipeline1-gzmbd-1417507587 5854990
Found execution id: 125 for pod pipeline1-gzmbd-1417507587.
MODIFIED pipeline1-gzmbd-1417507587 5854991
Found execution id: 126 for pod pipeline1-gzmbd-1417507587.
MODIFIED pipeline1-gzmbd-1417507587 5854999
Found execution id: 126 for pod pipeline1-gzmbd-1417507587.
MODIFIED pipeline1-gzmbd-1417507587 5855003
Found execution id: 126 for pod pipeline1-gzmbd-1417507587.
MODIFIED pipeline1-gzmbd-1417507587 5855004
Found execution id: 126 for pod pipeline1-gzmbd-1417507587.
Artifact: {'output_name': 'numbers', 'uri': 'https:/artifacts/get?source=minio&bucket=mlpipeline&key=artifacts/pipeline1-gzmbd/pipeline1-gzmbd-1417507587/write-numbers-numbers.tgz', 'type': 'String'}
MODIFIED pipeline1-gzmbd-1417507587 5855007
MODIFIED pipeline1-gzmbd-1417507587 5855014


...
ADDED tfx-pipeline-lmbw6-1947672582 5488020
ADDED pipeline1-4cd5g-2150790726 5858993
ADDED tfx-pipeline-lmbw6-2473736504 5488022
ERROR None None
{'type': 'ERROR', 'object': {'api_version': 'v1',
 'kind': 'Status',
 'metadata': {'annotations': None,
              'cluster_name': None,
              'creation_timestamp': None,
              'deletion_grace_period_seconds': None,
              'deletion_timestamp': None,
              'finalizers': None,
              'generate_name': None,
              'generation': None,
              'initializers': None,
              'labels': None,
              'managed_fields': None,
              'name': None,
              'namespace': None,
              'owner_references': None,
              'resource_version': None,
              'self_link': None,
              'uid': None},
 'spec': None,
 'status': {'conditions': None,
            'container_statuses': None,
            'host_ip': None,
            'init_container_statuses': None,
            'message': None,
            'nominated_node_name': None,
            'phase': None,
            'pod_ip': None,
            'qos_class': None,
            'reason': None,
            'start_time': None}}, 'raw_object': {'kind': 'Status', 'apiVersion': 'v1', 'metadata': {}, 'status': 'Failure', 'message': 'too old resource version: 5488022 (5858592)', 'reason': 'Gone', 'code': 410}}
"""


"""
#kubectl create -n kubeflow -f - <<EOF
kubectl create -f - <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: sleeper
spec:
  containers:
  - name: main
    image: python:3.7
    command:
    - sleep
    - '100000'
EOF
"""

"""
#kubectl create -n kubeflow -f - <<EOF
kubectl create -f - <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: metadata-writer-sleeper
spec:
  containers:
  - name: main
    image: tensorflow/tensorflow:1.15.0-py3
    command:
    - sh
    - -e
    - -c
    - |
      python3 -m pip install kubernetes 'ml-metadata==0.14' --upgrade --quiet --user
      sleep 100000
EOF
"""

"""
kubectl create -f - <<EOF
apiVersion: apps/v1beta2
kind: Deployment
metadata:
  name: metadata-writer-sleeper-deployment
  selector:
    matchLabels:
      app: metadata-writer
spec:
  template:
    metadata:
      labels:
        app: metadata-writer
    containers:
    - name: main
        image: tensorflow/tensorflow:1.15.0-py3
        command:
        - sh
        - -e
        - -c
        - |
        python3 -m pip install kubernetes 'ml-metadata==0.14' --upgrade --quiet --user
        sleep 100000
EOF
"""