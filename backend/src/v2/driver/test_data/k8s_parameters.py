from typing import Optional, List

from kfp import dsl
from kfp import kubernetes
from kfp.dsl import Output, OutputPath


@dsl.component(packages_to_install=['kubernetes'])
def assert_values():
    import os
    from kubernetes import client, config

    cfg_key_1 = os.getenv("CFG_KEY_1", "didn't work")
    cfg_key_2 = os.getenv("CFG_KEY_2", "didn't work")
    cfg_key_3 = os.getenv("CFG_KEY_3", "didn't work")
    cfg_key_4 = os.getenv("CFG_KEY_4", "didn't work")

    assert cfg_key_1 == "value1"
    assert cfg_key_2 == "value2"
    assert cfg_key_3 == "value3"
    assert cfg_key_4 == "value4"

    with open('/tmp/config_map/cfgKey3', 'r') as f:
        assert f.read() == "value3"

    secret_key_1 = os.getenv("SECRET_KEY_1", "didn't work")
    secret_key_2 = os.getenv("SECRET_KEY_2", "didn't work")
    secret_key_3 = os.getenv("SECRET_KEY_3", "didn't work")
    secret_key_4 = os.getenv("SECRET_KEY_4", "didn't work")

    assert secret_key_1 == "value1"
    assert secret_key_2 == "value2"
    assert secret_key_3 == "value3"
    assert secret_key_4 == "value4"

    with open('/tmp/secret/secretKey3', 'r') as f:
        assert f.read() == "value3"

    # Get pod YAML
    config.load_incluster_config()
    v1 = client.CoreV1Api()
    pod_name = os.getenv('HOSTNAME')
    namespace = open('/var/run/secrets/kubernetes.io/serviceaccount/namespace').read()
    pod = v1.read_namespaced_pod(pod_name, namespace)

    # Get pod's pull secrets
    print("\nPod Pull Secrets:")
    pull_secrets = []
    if pod.spec.image_pull_secrets:
        for secret in pod.spec.image_pull_secrets:
            print(f"Secret name: {secret.name}")
            pull_secrets.append(secret.name)

    assert len(pull_secrets) == 6
    print(pull_secrets)
    assert pull_secrets == ['pull-secret-1', 'pull-secret-2', 'pull-secret-1', 'pull-secret-2', 'pull-secret-3', 'pull-secret-4']

    # Get pod's node selector
    print("\nPod Node Selector:")
    node_selector = pod.spec.node_selector
    print(node_selector)
    if node_selector:
        for key, value in node_selector.items():
            print(f"Node selector {key}: {value}")

    assert node_selector == {"kubernetes.io/arch": "amd64",}

    # Get pod's tolerations
    print("\nPod Tolerations:")
    tolerations = pod.spec.tolerations
    print(tolerations)

    # Get pod's node affinity
    print("\nPod Node Affinity:")
    node_affinity = pod.spec.affinity.node_affinity if pod.spec.affinity else None
    print(node_affinity)

    # Helper function to check node affinity match expression
    def has_match_expression(expressions, key, operator, values):
        for expr in expressions:
            if (expr.key == key and
                    expr.operator == operator and
                    expr.values == values):
                return True
        return False

    # Check node affinity rules
    assert node_affinity is not None
    required_terms = node_affinity.required_during_scheduling_ignored_during_execution.node_selector_terms
    assert len(required_terms) == 1
    match_expressions = required_terms[0].match_expressions
    assert len(match_expressions) == 1
    assert has_match_expression(match_expressions, "kubernetes.io/os", "In", ["linux"])

    # Helper function to check if a toleration exists
    def has_toleration(key, effect, operator, value=None, toleration_seconds=None):
        for t in tolerations:
            if (t.key == key and
                t.effect == effect and
                t.operator == operator and
                t.value == value and
                t.toleration_seconds == toleration_seconds):
                return True
        return False

    # Check each toleration individually
    assert has_toleration('some_foo_key1', 'NoSchedule', 'Equal', 'value1')
    assert has_toleration('some_foo_key2', 'NoExecute', 'Exists')
    assert has_toleration('some_foo_key3', 'NoSchedule', 'Equal', 'value1')
    assert has_toleration('some_foo_key4', 'NoSchedule', 'Equal', 'value2')
    assert has_toleration('some_foo_key5', 'NoExecute', 'Exists')
    assert has_toleration('some_foo_key6', 'NoSchedule', 'Equal', 'value3')

    # Get pod's PVCs and Empty Dir
    print("\nPod Volumes and PVCs:")
    volumes = pod.spec.volumes
    pvcs = []
    empty_dir = []
    if volumes:
        for volume in volumes:
            if volume.persistent_volume_claim:
                print(f"Volume name: {volume.name}")
                print(f"PVC name: {volume.persistent_volume_claim.claim_name}")
                pvcs.append(volume.persistent_volume_claim)
    assert len(pvcs) == 1
    assert pvcs[0].claim_name.endswith('pvc-1')

@dsl.component(packages_to_install=['kubernetes'])
def assert_values_two():
    from kubernetes import client, config
    import os

    # Get pod YAML
    config.load_incluster_config()
    v1 = client.CoreV1Api()
    pod_name = os.getenv('HOSTNAME')
    namespace = open('/var/run/secrets/kubernetes.io/serviceaccount/namespace').read()
    pod = v1.read_namespaced_pod(pod_name, namespace)

    print("\nPod Node Selector:")
    node_selector = pod.spec.node_selector
    print(node_selector)
    assert node_selector == {"kubernetes.io/os": "linux"}

    # Get pod's node affinity
    print("\nPod Node Affinity:")
    node_affinity = pod.spec.affinity.node_affinity if pod.spec.affinity else None
    print(node_affinity)

    # Helper function to check node affinity match expression
    def has_match_expression(expressions, key, operator, values):
        for expr in expressions:
            if (expr.key == key and
                    expr.operator == operator and
                    expr.values == values):
                return True
        return False

    # Check node affinity rules
    assert node_affinity is not None
    required_terms = node_affinity.required_during_scheduling_ignored_during_execution.node_selector_terms
    assert len(required_terms) == 1
    match_expressions = required_terms[0].match_expressions
    assert len(match_expressions) == 1
    assert has_match_expression(match_expressions, "kubernetes.io/os", "In", ["linux"])

@dsl.component(packages_to_install=['kubernetes'])
def assert_values_three():
    from kubernetes import client, config
    import os

    # Get pod YAML
    config.load_incluster_config()
    v1 = client.CoreV1Api()
    pod_name = os.getenv('HOSTNAME')
    namespace = open('/var/run/secrets/kubernetes.io/serviceaccount/namespace').read()
    pod = v1.read_namespaced_pod(pod_name, namespace)

    tolerations = pod.spec.tolerations
    print(tolerations)

    # Helper function to check if a toleration exists
    def has_toleration(key, effect, operator, value=None, toleration_seconds=None):
        for t in tolerations:
            if (t.key == key and
                    t.effect == effect and
                    t.operator == operator and
                    t.value == value and
                    t.toleration_seconds == toleration_seconds):
                return True
        return False

    # Check toleration
    assert has_toleration('some_foo_key4', 'NoSchedule', 'Equal', 'value2')
    assert has_toleration('some_foo_key5', 'NoExecute', 'Exists')

@dsl.component()
def cfg_name_generator(some_output: OutputPath(str)):
    configmap_name = "cfg-3"
    with open(some_output, 'w') as f:
        f.write(configmap_name)

@dsl.component()
def secret_name_generator(some_output: OutputPath(str)):
    secret_name = "secret-3"
    with open(some_output, 'w') as f:
        f.write(secret_name)

@dsl.component()
def get_access_mode(access_mode: OutputPath(List[str])):
    import json
    with open(access_mode, 'w') as f:
        f.write(json.dumps(["ReadWriteOnce"]))

@dsl.component()
def get_node_affinity(node_affinity: OutputPath(dict)):
    import json
    with open(node_affinity, 'w') as f:
        f.write(json.dumps(
            {
                "requiredDuringSchedulingIgnoredDuringExecution": {
                    "nodeSelectorTerms": [
                        {
                            "matchExpressions": [
                                {
                                    "key": "kubernetes.io/os",
                                    "operator": "In",
                                    "values": ["linux"]
                                }
                            ]
                        }
                    ]
                }
            }
        ))

@dsl.component()
def generate_requests_resources(cpu_request_out: OutputPath(str), memory_request_out: OutputPath(str)):
    with open(cpu_request_out, 'w') as f:
        f.write('100m')
    with open(memory_request_out, 'w') as f:
        f.write('500Mi')

@dsl.component()
def get_node_affinity(node_affinity: OutputPath(dict)):
    import json
    with open(node_affinity, 'w') as f:
        f.write(json.dumps(
            {
                "requiredDuringSchedulingIgnoredDuringExecution": {
                    "nodeSelectorTerms": [
                        {
                            "matchExpressions": [
                                {
                                    "key": "kubernetes.io/os",
                                    "operator": "In",
                                    "values": ["linux"]
                                }
                            ]
                        }
                    ]
                }
            }
        ))

# TODO (HumairAK): Empty Dir and Field Path TaskOutputParameters
# not supported yet
# @dsl.component()
# def get_empty_dir_volume_name(volume_name: OutputPath(str)):
#     with open(volume_name, 'w') as f:
#         f.write("bar-dir")
#
# @dsl.component()
# def get_field_path_env_var(env_var: OutputPath(str)):
#     with open(env_var, 'w') as f:
#         f.write("SERVICE_ACCOUNT_NAME")

node_selector_default = {"kubernetes.io/os": "linux"}

toleration_list_default = [
    {
        "key": "some_foo_key4",
        "operator": "Equal",
        "value": "value2",
        "effect": "NoSchedule"
    },
    {
        "key": "some_foo_key5",
        "operator": "Exists",
        "effect": "NoExecute"
    }
]

toleration_dict_default = {
    "key": "some_foo_key6",
    "operator": "Equal",
    "value": "value3",
    "effect": "NoSchedule"
}

@dsl.pipeline
def secondary_pipeline(train_tolerations: list):
    task = assert_values_three()
    kubernetes.add_toleration_json(task, train_tolerations)

default_node_affinity = {
                "requiredDuringSchedulingIgnoredDuringExecution": {
                    "nodeSelectorTerms": [
                        {
                            "matchExpressions": [
                                {
                                    "key": "kubernetes.io/os",
                                    "operator": "In",
                                    "values": ["linux"]
                                }
                            ]
                        }
                    ]
                }
            }

@dsl.pipeline
def primary_pipeline(
        configmap_parm: str = 'cfg-2',
        secret_param: str = 'secret-2',
        pull_secret_1: str = 'pull-secret-1',
        pull_secret_2: str = 'pull-secret-2',
        pull_secret_3: str = 'pull-secret-3',
        node_selector_input: dict = {"kubernetes.io/os": "linux"},
        tolerations_list_input: list = toleration_list_default,
        tolerations_dict_input: dict = toleration_dict_default,
        pvc_name_suffix_input: str = '-pvc-1',
        empty_dir_mnt_path: str = '/empty_dir/path',
        field_path: str = 'spec.serviceAccountName',
        default_node_affinity_input: dict = default_node_affinity,
        cpu_limit: str = '200m',
        memory_limit: str = '500Mi',
        container_image: str = 'python:3.9',
):

    cfg_name_generator_task = cfg_name_generator()
    secret_name_generator_task = secret_name_generator()

    task = assert_values().set_caching_options(enable_caching=False)

    # configmap verification
    kubernetes.use_config_map_as_env(
        task,
        config_map_name='cfg-1',
        config_map_key_to_env={
            'cfgKey1': 'CFG_KEY_1',
            'cfgKey2': 'CFG_KEY_2'
        })
    kubernetes.use_config_map_as_env(
        task,
        config_map_name=configmap_parm,
        config_map_key_to_env={
            'cfgKey3': 'CFG_KEY_3',
        })
    kubernetes.use_config_map_as_env(
        task,
        config_map_name=cfg_name_generator_task.output,
        config_map_key_to_env={
            'cfgKey4': 'CFG_KEY_4',
        })

    kubernetes.use_config_map_as_volume(
        task,
        config_map_name=configmap_parm,
        mount_path='/tmp/config_map')

    # secret verification
    kubernetes.use_secret_as_env(
        task,
        secret_name='secret-1',
        secret_key_to_env={
            'secretKey1': 'SECRET_KEY_1',
            'secretKey2': 'SECRET_KEY_2'
        })
    kubernetes.use_secret_as_env(
        task,
        secret_name=secret_param,
        secret_key_to_env={
            'secretKey3': 'SECRET_KEY_3',
        })
    kubernetes.use_secret_as_env(
        task,
        secret_name=secret_name_generator_task.output,
        secret_key_to_env={
            'secretKey4': 'SECRET_KEY_4',
        })

    kubernetes.use_secret_as_volume(
        task,
        secret_name=secret_param,
        mount_path='/tmp/secret')

    # pull secrets
    kubernetes.set_image_pull_secrets(
        task,
        secret_names=[pull_secret_1, pull_secret_2])
    kubernetes.set_image_pull_secrets(
        task,
        secret_names=["pull-secret-1", "pull-secret-2"])
    kubernetes.set_image_pull_secrets(
        task,
        secret_names=([pull_secret_3, "pull-secret-4"])
    )

    # node selector
    kubernetes.add_node_selector_json(
        task,
        node_selector_json={
            "kubernetes.io/arch": "amd64",
        },
    )

    # You can't append node selectors, to verify ComponentInput option
    # in another task
    task_2 = assert_values_two().set_caching_options(enable_caching=False)
    kubernetes.add_node_selector_json(
        task_2,
        node_selector_json=node_selector_input,
    )

    # tolerations
    kubernetes.add_toleration_json(task, [
        {
            "key": "some_foo_key1",
            "operator": "Equal",
            "value": "value1",
            "effect": "NoSchedule"
        },
        {
            "key": "some_foo_key2",
            "operator": "Exists",
            "effect": "NoExecute"
        }
    ])
    kubernetes.add_toleration_json(task, {
        "key": "some_foo_key3",
        "operator": "Equal",
        "value": "value1",
        "effect": "NoSchedule"
    })
    kubernetes.add_toleration_json(task, tolerations_dict_input)
    kubernetes.add_toleration_json(task, tolerations_list_input)

    # cpu/memory/container image
    generate_requests_resources_task = generate_requests_resources()
    task.set_cpu_request(generate_requests_resources_task.outputs["cpu_request_out"])
    task.set_memory_request(generate_requests_resources_task.outputs["memory_request_out"])

    task.set_cpu_limit(cpu_limit)
    task.set_memory_limit(memory_limit)
    task.set_container_image(container_image)

    # Test nested toleration
    secondary_pipeline(train_tolerations=tolerations_list_input)

    # PVCs
    access_mode_task  =  get_access_mode()
    output_pvc_task = kubernetes.CreatePVC(
        pvc_name_suffix=pvc_name_suffix_input, # Component Input Parameter
        access_modes=access_mode_task.output,  # Task Output Parameter
        size="5Mi",                            # Runtime Constant
    )
    kubernetes.mount_pvc(
        task,
        pvc_name=output_pvc_task.output, # Task Output Parameter
        mount_path='/pvc/path')
    output_pvc_delete_task = kubernetes.DeletePVC(pvc_name=output_pvc_task.output)
    output_pvc_delete_task.after(task)

    # node affinity
    get_node_affinity_task = get_node_affinity()
    kubernetes.add_node_affinity_json(
        task,
        node_affinity_json=get_node_affinity_task.output  # Task Output Parameter
    )

    kubernetes.add_node_affinity_json(
        task_2,
        default_node_affinity_input  # Component Input Parameter
    )


    # TODO(HumairAK) Empty dir doesn't support parameterization
    # empty dir
    # empty_dir_volume_name_task = get_empty_dir_volume_name()
    # kubernetes.empty_dir_mount(
    #     task,
    #     empty_dir_volume_name_task.output,
    #     empty_dir_mnt_path,
    #     None,
    #     '1Mi',
    # )

    # TODO(HumairAK) Field Path doesn't support parameterization
    # field Path
    # get_field_path_env_var_task = get_field_path_env_var()
    # kubernetes.use_field_path_as_env(
    #     task,
    #     field_path=field_path,                       # Component Input Parameter
    #     env_name=get_field_path_env_var_task.output, # Task Output Parameter
    # )


if __name__ == '__main__':
    from kfp import compiler
    compiler.Compiler().compile(
        pipeline_func=primary_pipeline,
        package_path=__file__.replace('.py', '.yaml'))