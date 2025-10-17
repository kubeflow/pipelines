from typing import Dict, Optional

from kfp import dsl, kubernetes


@dsl.component
def log_message(message: str) -> None:
    print(message)


@dsl.pipeline
def missing_kubernetes_optional_inputs_pipeline(
    optional_affinity: Optional[Dict[str, object]] = None,
    optional_selector: Optional[Dict[str, object]] = None,
    optional_secret_name: Optional[str] = None,
    optional_config_map_name: Optional[str] = None,
    optional_tolerations: Optional[Dict[str, object]] = None,
):
    affinity_task = log_message(message="missing node affinity optional input")
    affinity_task.set_caching_options(enable_caching=False)
    kubernetes.add_node_affinity_json(
        task=affinity_task,
        node_affinity_json=optional_affinity,
    )

    selector_task = log_message(message="missing node selector optional input")
    selector_task.set_caching_options(enable_caching=False)
    kubernetes.add_node_selector_json(
        task=selector_task,
        node_selector_json=optional_selector,
    )

    secret_task = log_message(message="missing secret and configmap optional inputs")
    secret_task.set_caching_options(enable_caching=False)
    kubernetes.use_secret_as_env(
        task=secret_task,
        secret_name=optional_secret_name,
        secret_key_to_env={"password": "PASSWORD"},
    )
    kubernetes.use_config_map_as_env(
        task=secret_task,
        config_map_name=optional_config_map_name,
        config_map_key_to_env={"setting": "SETTING"},
    )

    toleration_task = log_message(message="missing tolerations optional input")
    toleration_task.set_caching_options(enable_caching=False)
    kubernetes.add_toleration_json(
        task=toleration_task,
        toleration_json=optional_tolerations,
    )


if __name__ == "__main__":
    from kfp import compiler

    compiler.Compiler().compile(
        pipeline_func=missing_kubernetes_optional_inputs_pipeline,
        package_path="missing_kubernetes_optional_inputs.yaml",
    )
