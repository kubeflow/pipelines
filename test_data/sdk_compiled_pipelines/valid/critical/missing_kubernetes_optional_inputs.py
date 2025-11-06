from typing import Any, Dict, List, Optional

from kfp import dsl, kubernetes


@dsl.component
def log_message(message: str) -> None:
    print(message)


@dsl.pipeline
def missing_kubernetes_optional_inputs_pipeline(
    optional_affinity: Optional[Dict[str, Any]] = None,
    optional_selector: Optional[Dict[str, Any]] = None,
    optional_secret_name: Optional[str] = None,
    optional_config_map_name: Optional[str] = None,
    optional_image_pull_secret_name: Optional[str] = None,
    optional_tolerations: Optional[List[Dict[str, Any]]] = None,
):
    task = log_message(message="baseline task")
    task.set_caching_options(enable_caching=False)

    kubernetes.add_node_affinity_json(
        task=task,
        node_affinity_json=optional_affinity,
    )

    kubernetes.add_node_selector_json(
        task=task,
        node_selector_json=optional_selector,
    )

    kubernetes.use_secret_as_env(
        task=task,
        secret_name=optional_secret_name,
        secret_key_to_env={"password": "PASSWORD"},
        optional=True,
    )
    kubernetes.use_secret_as_volume(
        task=task,
        secret_name=optional_secret_name,
        mount_path="/mnt/secret",
        optional=True,
    )

    kubernetes.use_config_map_as_env(
        task=task,
        config_map_name=optional_config_map_name,
        config_map_key_to_env={"setting": "SETTING"},
        optional=True,
    )
    kubernetes.use_config_map_as_volume(
        task=task,
        config_map_name=optional_config_map_name,
        mount_path="/mnt/config",
        optional=True,
    )

    kubernetes.set_image_pull_secrets(
        task=task,
        secret_names=[optional_image_pull_secret_name],
    )

    kubernetes.add_toleration_json(
        task=task,
        toleration_json=optional_tolerations,
    )


if __name__ == "__main__":
    from kfp import compiler

    compiler.Compiler().compile(
        pipeline_func=missing_kubernetes_optional_inputs_pipeline,
        package_path="missing_kubernetes_optional_inputs.yaml",
    )
