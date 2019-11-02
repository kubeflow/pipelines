#kubectl exec sleeper -- python3 -m pip install kubernetes 'ml-metadata==0.14' 'tensorflow>=1.15' --upgrade --quiet --user
#kubectl exec sleeper -- python3 -m pip freeze
#kubectl exec sleeper -it -- python3 -c "$(< metadata_writer.py)"

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

use_metadata_store = True
metadata_configmap_name = 'metadata-configmap'
metadata_configmap_namespace = 'default' # Not kubeflow
metadata_db_user = 'root'
metadata_db_password = ''

if use_metadata_store:
    #import site
    #import subprocess
    #import sys
    #subprocess.run([sys.executable, '-m', 'pip', 'install', 'ml-metadata==0.14', 'tensorflow>=1.15', '--upgrade', '--quiet', '--user'])
    #sys.path.append(site.getusersitepackages())

    import ml_metadata

    metadata_configmap_dict = k8s_api.read_namespaced_config_map(
        name=metadata_configmap_name,
        namespace=metadata_configmap_namespace,
    ).data

    from ml_metadata.proto import metadata_store_pb2
    from ml_metadata.metadata_store import metadata_store

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
    #print(mlmd_store.get_artifact_types())

import json
patch_retries = 100

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
            # TODO: Sleep for 100ms

def add_pod_metadata(
    namespace: str,
    pod_name: str,
    field: str,
    key: str,
    value: str,
    k8s_api: kubernetes.client.CoreV1Api = None,
):
    patch = {
        'metadata': {
            field: {key: value}
        }
    }
    patch_pod_metadata(
        namespace=namespace,
        pod_name=pod_name,
        patch=patch,
        k8s_api=k8s_api,
    )

ARGO_OUTPUTS_ANNOTATION_KEY = 'workflows.argoproj.io/outputs'
ARGO_TEMPLATE_ANNOTATION_KEY = 'workflows.argoproj.io/template'
COMPONENT_SPEC_ANNOTATION_KEY = 'pipelines.kubeflow.org/component_spec'
#PIPELINE_SPEC_ANNOTATION_KEY = 'pipelines.kubeflow.org/pipeline_spec'
EXECUTION_ID_ANNOTATION_KEY = 'pipelines.kubeflow.org/metadata_execution_id'
EXECUTION_ID_LABEL_KEY = 'pipelines.kubeflow.org/metadata_execution_id'
ARTIFACT_IDS_ANNOTATION_KEY = 'pipelines.kubeflow.org/metadata_artifact_ids'

ARGO_WORKFLOW_LABEL_KEY = 'workflows.argoproj.io/workflow'
KFP_METADATA_WRITTEN_LABEL_KEY = 'pipelines.kubeflow.org/metadata_written'


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
                    EXECUTION_ID_ANNOTATION_KEY: None,
                    ARTIFACT_IDS_ANNOTATION_KEY: None,
                },
                'labels': {
                    EXECUTION_ID_LABEL_KEY: None,
                    KFP_METADATA_WRITTEN_LABEL_KEY: None,
                }
            },
        )
#%%

cleanup_pods()


#%%

def add_execution(
) -> str:
    execution_id = str(len(executions) + 1)
    executions[execution_id] = obj.metadata.name
    return execution_id

def add_artifact(
    execution_id: str,
    output_name: str,
    uri: str,
    type: str = None,
):
    print(locals())
    artifact_id = str(len(artifacts) + 1)
    artifacts[artifact_id] = (execution_id, output_name, uri, type)
    return artifact_id


def output_name_to_argo(name: str) -> str:
    import re
    return re.sub('-+', '-', re.sub('[^-0-9a-z]+', '-', name.lower())).strip('-')

#%%

def get_or_add_execution_type_id_by_name(md_store, type_name): # -> ExecutionType ID
    try:
        execution_type = md_store.get_execution_type(type_name)
        return execution_type.id
    except:
        execution_type = metadata_store_pb2.ExecutionType(name=type_name)
        execution_type_id = md_store.put_execution_type(execution_type)
        return execution_type_id

def put_artifact(store, uri, type_name) -> int:
    return mlmd_store.put_artifacts([
        metadata_store_pb2.Artifact(
            uri=uri,
            type_id=store.put_artifact_type(
                metadata_store_pb2.ArtifactType(
                    name=type_name,
                ),
            ),
        ),
    ])[0]

#%%

executions = {} # id -> pod_name
execution_artifacts = {} # id = (execution_id, output_name) -> artifact_id
artifacts = {} # id -> Type, URL

# Fixed - TODO: Process existing pods first (but how to catch up the events? Maybe they can be replayed)
#               Check for Argo Pods without "metadata_artifacts_written: True" label

# list_pod_for_all_namespaces:
#   resource_version: Defaults to changes from the beginning of history.
#   label_selector: A selector to restrict the list of returned objects by their labels. Defaults to everything.
#   field_selector: A selector to restrict the list of returned objects by their fields. Defaults to everything.

#for event in k8s_watch.stream(k8s_api.list_pod_for_all_namespaces):
for event in k8s_watch.stream(
    k8s_api.list_pod_for_all_namespaces,
    label_selector=ARGO_WORKFLOW_LABEL_KEY + ',' + '!' + KFP_METADATA_WRITTEN_LABEL_KEY,
    timeout_seconds=3,
):
    try:
        obj = event['object']
        print(event['type'], obj.metadata.name)
        assert obj.kind == 'Pod'

        if EXECUTION_ID_LABEL_KEY in obj.metadata.labels:
            execution_id = obj.metadata.labels[EXECUTION_ID_LABEL_KEY]
            print('Found execution id: {} for pod {}.'.format(execution_id, obj.metadata.name))
        else:
            execution_id = add_execution()
            executions[execution_id] = obj.metadata.name
            # !!! Add annotation:
            obj.metadata.annotations[EXECUTION_ID_ANNOTATION_KEY] = execution_id
            add_pod_metadata(
                namespace=obj.metadata.namespace,
                pod_name=obj.metadata.name,
                #field='annotations',
                #key=EXECUTION_ID_ANNOTATION_KEY,
                field='labels',
                key=EXECUTION_ID_LABEL_KEY,
                value=execution_id,
            )
            print('New execution id: {} for pod {}.'.format(execution_id, obj.metadata.name))

        
        #if event['type'] == 'ADDED':
        #    pass

        #elif event['object'] == 'MODIFIED':
        if obj.status.phase not in ['Pending', 'Running']:
            # phase = One of Pending,Running,Succeeded,Failed,Unknown
            # Skip if pod does not have execution id? We should probably have local cache

            #execution_id2 = obj.metadata.annotations[EXECUTION_ID_ANNOTATION_KEY]
            #execution_id2 = obj.metadata.labels[EXECUTION_ID_LABEL_KEY]
            #print('Found execution id: {} for pod {}.'.format(execution_id2, obj.metadata.name))

            #assert obj.metadata.name in executions.values() # The pod name should have been added before
            if obj.status.phase == 'Succeeded':
                if ARGO_OUTPUTS_ANNOTATION_KEY in obj.metadata.annotations: # Should be present
                    argo_outputs = json.loads(obj.metadata.annotations[ARGO_OUTPUTS_ANNOTATION_KEY])
                    argo_template = json.loads(obj.metadata.annotations[ARGO_TEMPLATE_ANNOTATION_KEY])
                    argo_template_name = argo_template['name']
                    argo_output_artifacts = {}

                    for artifact in argo_outputs.get('artifacts', []):
                        art_name = artifact['name']
                        output_prefix = argo_template_name + '-'
                        if art_name.startswith(output_prefix):
                            art_name = art_name[len(output_prefix):]
                        #art_key = artifact['s3']['key']
                        argo_output_artifacts[art_name] = artifact

                    component_name = None
                    argo_output_name_to_type = {}
                    if COMPONENT_SPEC_ANNOTATION_KEY in obj.metadata.annotations:
                        component_spec = json.loads(obj.metadata.annotations[COMPONENT_SPEC_ANNOTATION_KEY])
                        component_name = component_spec.get('name', None)
                        output_name_to_type = {output['name']: output.get('type', None) for output in component_spec.get('outputs', [])}
                        argo_output_name_to_type = {output_name_to_argo(k): v for k, v in output_name_to_type.items() if v}
                        #print(argo_output_name_to_type)
                    
                    output_artifacts = []
                    for name, art in argo_output_artifacts.items():
                        artifact_uri = 'https:/artifacts/get?source=minio&bucket={bucket}&key={key}'.format(
                            bucket=art['s3']['bucket'],
                            key=art['s3']['key'],
                        )
                        add_artifact(
                            execution_id=execution_id,
                            output_name=name,
                            uri=artifact_uri,
                            type=argo_output_name_to_type.get(name, None),
                        )
                        output_artifacts.append(
                            dict(
                                output_name=name,
                                uri=artifact_uri,
                                type=argo_output_name_to_type.get(name, None),
                            )
                        )

                    execution_and_artifacts = dict(
                        execution_id=execution_id,
                        component_name=component_name,
                        output_artifacts=output_artifacts,
                    )
                    print(execution_and_artifacts)


    except Exception as e:
        #print(event)
        print(e)

""" 
$ kubectl exec sleeper -it -- python3 -c "$(< metadata_writer.py)"
ADDED tfx-pipeline-lmbw6-2908704420
New execution id: 1 for pod tfx-pipeline-lmbw6-2908704420.
add_artifact() got an unexpected keyword argument 'name'
ADDED tfx-pipeline-lmbw6-2172955551
New execution id: 2 for pod tfx-pipeline-lmbw6-2172955551.
add_artifact() got an unexpected keyword argument 'name'
ADDED tfx-pipeline-lmbw6-1947672582
New execution id: 3 for pod tfx-pipeline-lmbw6-1947672582.
add_artifact() got an unexpected keyword argument 'name'
ADDED tfx-pipeline-lmbw6-2473736504
New execution id: 4 for pod tfx-pipeline-lmbw6-2473736504.
add_artifact() got an unexpected keyword argument 'name'
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
{'execution_id': '4', 'component_name': 'Schemagen', 'output_artifacts': [{'output_name': 'output', 'uri': 'https:/artifacts/get?source=minio&bucket=mlpipeline&key=artifacts/tfx-pipeline-lmbw6/tfx-pipeline-lmbw6-2473736504/schemagen-output.tgz', 'type': 'Schema'}]} """