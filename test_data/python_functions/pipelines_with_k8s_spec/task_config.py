# Copyright 2025 The Kubeflow Authors
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
from kfp import dsl
from kfp import compiler
from kfp import kubernetes


@dsl.component(
    task_config_passthroughs=[
        dsl.TaskConfigPassthrough(field=dsl.TaskConfigField.ENV, apply_to_task=True),
        dsl.TaskConfigField.RESOURCES,
        dsl.TaskConfigField.KUBERNETES_VOLUMES,
        dsl.TaskConfigField.KUBERNETES_NODE_SELECTOR,
        dsl.TaskConfigField.KUBERNETES_TOLERATIONS,
        dsl.TaskConfigField.KUBERNETES_AFFINITY,
    ],
)
def echo_task_config(workspace_path: str, task_config: dsl.TaskConfig):
    import dataclasses
    import os
    import pprint

    assert task_config is not None

    actual = dataclasses.asdict(task_config)
    pprint.pprint(actual)

    workspace_pvc_name = None
    for volume in actual['volumes']:
        if volume['name'] == 'kfp-workspace':
            workspace_pvc_name = volume['persistentVolumeClaim']['claimName']
            break
    assert workspace_pvc_name is not None

    expected = {
        'affinity': {
            'nodeAffinity': {
                'requiredDuringSchedulingIgnoredDuringExecution': {
                    'nodeSelectorTerms': [{
                        'matchExpressions': [{
                            'key': 'disktype',
                            'operator': 'In',
                            'values': ['ssd']
                        }]
                    }]
                }
            }
        },
        'env': [{
            'name': 'ENV1',
            'value': 'val1'
        }, {
            'name': 'ENV2',
            'value': 'val2'
        }],
        'node_selector': {
            'disktype': 'ssd'
        },
        'resources': {
            'limits': {
                'cpu': '100m',
                'memory': '100Mi',
                'nvidia.com/gpu': '1'
            },
            'requests': {
                'cpu': '100m',
                'memory': '100Mi'
            }
        },
        'tolerations': [{
            'effect': 'NoExecute',
            'key': 'example-key',
            'operator': 'Exists',
            'tolerationSeconds': 3600
        }],
        'volume_mounts': [{
            'mountPath': '/kfp-workspace',
            'name': 'kfp-workspace'
        }, {
            'mountPath': '/data',
            'name': 'kubernetes-task-config-pvc'
        }],
        'volumes': [{
            'name': 'kfp-workspace',
            'persistentVolumeClaim': {
                'claimName': workspace_pvc_name
            }
        }, {
            'name': 'kubernetes-task-config-pvc',
            'persistentVolumeClaim': {
                'claimName': 'kubernetes-task-config-pvc'
            }
        }]
    }

    assert actual == expected

    assert os.getenv('ENV1') == 'val1'
    assert os.getenv('ENV2') == 'val2'


@dsl.pipeline(
    name='task-config',
    description='A simple intro pipeline',
    pipeline_config=dsl.PipelineConfig(
        workspace=dsl.WorkspaceConfig(
            size='5Mi',
            kubernetes=dsl.KubernetesWorkspaceConfig(
                pvcSpecPatch={"storageClassName": "standard"}))))
def pipeline_task_config():
    """Pipeline that leverages dsl.TaskConfig."""
    pvc1 = kubernetes.CreatePVC(
        pvc_name='kubernetes-task-config-pvc',
        access_modes=['ReadWriteOnce'],
        size='5Mi',
        storage_class_name='standard',
    ).set_caching_options(False)
    echo_task_config_task = echo_task_config(
        workspace_path=dsl.WORKSPACE_PATH_PLACEHOLDER).set_caching_options(
            False).set_cpu_request('100m').set_memory_request(
                '100Mi').set_cpu_limit('100m').set_memory_limit(
                    '100Mi').set_accelerator_type(
                        'nvidia.com/gpu').set_accelerator_limit(1)

    kubernetes.mount_pvc(
        echo_task_config_task,
        pvc_name=pvc1.outputs['name'],
        mount_path='/data',
    )

    kubernetes.add_node_selector(
        echo_task_config_task,
        label_key='disktype',
        label_value='ssd',
    )

    kubernetes.add_toleration(
        echo_task_config_task,
        key='example-key',
        operator='Exists',
        effect='NoExecute',
        toleration_seconds=3600,
    )

    kubernetes.add_node_affinity(
        echo_task_config_task,
        match_expressions=[{
            'key': 'disktype',
            'operator': 'In',
            'values': ['ssd']
        }])

    echo_task_config_task.set_env_variable(name='ENV1', value='val1')
    echo_task_config_task.set_env_variable(name='ENV2', value='val2')

    delete_pvc1 = kubernetes.DeletePVC(pvc_name=pvc1.outputs['name']).after(
        echo_task_config_task).set_caching_options(False)


if __name__ == "__main__":
    # execute only if run as a script
    compiler.Compiler().compile(
        pipeline_func=pipeline_task_config,
        package_path=__file__.replace('.py', '.yaml'))
