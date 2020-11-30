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
import hashlib
import os
import sys
import re
import kubernetes
import yaml
from time import sleep

from metadata_helpers import *


namespace_to_watch = os.environ.get('NAMESPACE_TO_WATCH', 'default')


kubernetes.config.load_incluster_config()
k8s_api = kubernetes.client.CoreV1Api()
k8s_watch = kubernetes.watch.Watch()


patch_retries = 20
sleep_time = 0.1


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


#Connecting to MetadataDB
mlmd_store = connect_to_mlmd()
print("Connected to the metadata store")


ARGO_OUTPUTS_ANNOTATION_KEY = 'workflows.argoproj.io/outputs'
ARGO_TEMPLATE_ANNOTATION_KEY = 'workflows.argoproj.io/template'
KFP_COMPONENT_SPEC_ANNOTATION_KEY = 'pipelines.kubeflow.org/component_spec'
KFP_PARAMETER_ARGUMENTS_ANNOTATION_KEY = 'pipelines.kubeflow.org/arguments.parameters'
METADATA_EXECUTION_ID_LABEL_KEY = 'pipelines.kubeflow.org/metadata_execution_id'
METADATA_CONTEXT_ID_LABEL_KEY = 'pipelines.kubeflow.org/metadata_context_id'
METADATA_ARTIFACT_IDS_ANNOTATION_KEY = 'pipelines.kubeflow.org/metadata_artifact_ids'
METADATA_INPUT_ARTIFACT_IDS_ANNOTATION_KEY = 'pipelines.kubeflow.org/metadata_input_artifact_ids'
METADATA_OUTPUT_ARTIFACT_IDS_ANNOTATION_KEY = 'pipelines.kubeflow.org/metadata_output_artifact_ids'

ARGO_WORKFLOW_LABEL_KEY = 'workflows.argoproj.io/workflow'
ARGO_COMPLETED_LABEL_KEY = 'workflows.argoproj.io/completed'
METADATA_WRITTEN_LABEL_KEY = 'pipelines.kubeflow.org/metadata_written'


def output_name_to_argo(name: str) -> str:
    import re
    # This sanitization code should be kept in sync with the code in the DSL compiler.
    # See https://github.com/kubeflow/pipelines/blob/39975e3cde7ba4dcea2bca835b92d0fe40b1ae3c/sdk/python/kfp/compiler/_k8s_helper.py#L33
    return re.sub('-+', '-', re.sub('[^-_0-9A-Za-z]+', '-', name)).strip('-')

def is_s3_endpoint(endpoint: str) -> bool:
    return re.search('^.*s3.*amazonaws.com.*$', endpoint)

def get_object_store_provider(endpoint: str) -> bool:
    if is_s3_endpoint(endpoint):
        return 's3'
    else:
        return 'minio'

def argo_artifact_to_uri(artifact: dict) -> str:
    # s3 here means s3 compatible object storage. not AWS S3.
    if 's3' in artifact:
        s3_artifact = artifact['s3']
        return '{provider}://{bucket}/{key}'.format(
            provider=get_object_store_provider(s3_artifact['endpoint']),
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
        label_selector=ARGO_WORKFLOW_LABEL_KEY,
        timeout_seconds=1800,  # Sometimes watch gets stuck
        _request_timeout=2000,  # Sometimes HTTP GET gets stuck
    ):
        try:
            obj = event['object']
            print('Kubernetes Pod event: ', event['type'], obj.metadata.name, obj.metadata.resource_version)
            if event['type'] == 'ERROR':
                print(event)

            pod_name = obj.metadata.name

            # Logging pod changes for debugging
            with open('/tmp/pod_' + obj.metadata.name + '_' + obj.metadata.resource_version, 'w') as f:
                f.write(yaml.dump(obj.to_dict()))

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

                # Saving input paramater arguments
                execution_custom_properties = {}
                if KFP_PARAMETER_ARGUMENTS_ANNOTATION_KEY in obj.metadata.annotations:
                    parameter_arguments_json = obj.metadata.annotations[KFP_PARAMETER_ARGUMENTS_ANNOTATION_KEY]
                    try:
                        parameter_arguments = json.loads(parameter_arguments_json)
                        for paramater_name, parameter_value in parameter_arguments.items():
                            execution_custom_properties['input:' + paramater_name] = parameter_value
                    except Exception:
                        pass

                # Adding new execution to the database
                execution = create_new_execution_in_existing_run_context(
                    store=mlmd_store,
                    context_id=run_context.id,
                    execution_type_name=KFP_EXECUTION_TYPE_NAME_PREFIX + component_version,
                    pod_name=pod_name,
                    pipeline_name=argo_workflow_name,
                    run_id=argo_workflow_name,
                    instance_id=component_name,
                    custom_properties=execution_custom_properties,
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
                            argo_artifact=art,
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
