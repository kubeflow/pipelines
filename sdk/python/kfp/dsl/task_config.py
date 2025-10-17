"""Definition for TaskConfig."""

import dataclasses
from typing import Any, Dict, List, Optional


# TaskConfig needs to stay aligned with the TaskConfig in backend/src/v2/driver/driver.go.
@dataclasses.dataclass
class TaskConfig:
    """Configurations for a task.

    Annotate a component parameter with this type when you want the task's
    runtime configuration to be forwarded to an external workload and optionally applied to the
    task's own pod. This is useful when the task launches another Kubernetes resource (for example,
    a Kubeflow Trainer job).

    This is an empty value by default and is populated if the component is annotated with
    task_config_passthroughs.

    All fields are optional and map 1:1 to fragments of the Kubernetes Pod
    spec. These fields have values as Python dictionaries/lists that conform to the
    Kubernetes JSON schema.

    Example:
      ::

        @dsl.component(
            packages_to_install=["kubernetes"],
            task_config_passthroughs=[
                dsl.TaskConfigField.RESOURCES,
                dsl.TaskConfigField.KUBERNETES_TOLERATIONS,
                dsl.TaskConfigField.KUBERNETES_NODE_SELECTOR,
                dsl.TaskConfigField.KUBERNETES_AFFINITY,
                dsl.TaskConfigPassthrough(field=dsl.TaskConfigField.ENV, apply_to_task=True),
                dsl.TaskConfigPassthrough(field=dsl.TaskConfigField.KUBERNETES_VOLUMES, apply_to_task=True),
            ],
        )
        def train(num_nodes: int, workspace_path: str, output_model: dsl.Output[dsl.Model], task_config: dsl.TaskConfig):
            import os
            import shutil
            from kubernetes import client as k8s_client, config
            config.load_incluster_config()

            with open(
                    "/var/run/secrets/kubernetes.io/serviceaccount/namespace", "r"
                ) as ns_file:
                    namespace = ns_file.readline()

            train_job_script = "with open('/kfp-workspace/model', 'w') as f: f.write('hello')"

            dataset_path = os.path.join(workspace_path, "dataset")
            with open(dataset_path, "w") as f:
                f.write("Prepare dataset here...")

            train_job = {
                    "apiVersion": "trainer.kubeflow.org/v1alpha1",
                    "kind": "TrainJob",
                    "metadata": {"name": f"kfp-train-job", "namespace": namespace},
                    "spec": {
                        "runtimeRef": {"name": "torch-distributed"},
                        "trainer": {
                            "numNodes": num_nodes,
                            "resourcesPerNode": task_config.resources,
                            "env": task_config.env,
                            "command": ["python", "-c", train_job_script],
                        },
                        "podSpecOverrides": [
                            {
                                "targetJobs": [{"name": "node"}],
                                "volumes": task_config.volumes,
                                "containers": [
                                    {
                                        "name": "node",
                                        "volumeMounts": task_config.volume_mounts,
                                    }
                                ],
                                "nodeSelector": task_config.node_selector,
                                "tolerations": task_config.tolerations,
                            }
                        ],
                    },
                }
            print(train_job)
            api_client = k8s_client.ApiClient()
            custom_objects_api = k8s_client.CustomObjectsApi(api_client)
            response = custom_objects_api.create_namespaced_custom_object(
                group="trainer.kubeflow.org",
                version="v1alpha1",
                namespace=namespace,
                plural="trainjobs",
                body=train_job,
            )
            job_name = response["metadata"]["name"]
            print(f"TrainJob {job_name} created successfully")

            print("Polling train job code goes here...")

            print("Copying output model")
            shutil.copy(os.path.join(workspace_path, "model"), output_model.path)

        @dsl.pipeline
        def example_task_config():
            train_task = train(num_nodes=1, workspace_path=dsl.WORKSPACE_PATH_PLACEHOLDER)
            train_task.set_cpu_request("1")
            train_task.set_memory_request("20Gi")
            train_task.set_cpu_limit("2")
            train_task.set_memory_limit("50Gi")
            train_task.set_accelerator_type("nvidia.com/gpu")
            train_task.set_accelerator_limit("1")
    """

    affinity: Optional[Dict[str, Any]] = None
    tolerations: Optional[List[Dict[str, Any]]] = None
    node_selector: Optional[Dict[str, str]] = None
    env: Optional[List[Dict[str, Any]]] = None
    volumes: Optional[List[Dict[str, Any]]] = None
    volume_mounts: Optional[List[Dict[str, Any]]] = None
    resources: Optional[Dict[str, Any]] = None
