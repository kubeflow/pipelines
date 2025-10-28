from typing import Any, Dict, List, Optional

from kfp import dsl, kubernetes


@dsl.component
def log_message(message: str) -> None:
    print(message)


@dsl.pipeline
def missing_kubernetes_optional_inputs_pipeline(
    optional_affinity: Optional[Dict[str, Any]] = None,
    enable_affinity: bool = False,
    optional_selector: Optional[Dict[str, Any]] = None,
    enable_selector: bool = False,
    optional_secret_name: Optional[str] = None,
    enable_secret_env: bool = False,
    optional_config_map_name: Optional[str] = None,
    enable_config_map_env: bool = False,
    optional_tolerations: Optional[List[Dict[str, Any]]] = None,
    enable_tolerations: bool = False,
):
    base_task = log_message(message="baseline task")
    base_task.set_caching_options(enable_caching=False)

    with dsl.If(enable_affinity == True):
        affinity_task = log_message(message="node affinity applied")
        affinity_task.set_caching_options(enable_caching=False)
        kubernetes.add_node_affinity_json(
            task=affinity_task,
            node_affinity_json=optional_affinity,
        )

    with dsl.If(enable_selector == True):
        selector_task = log_message(message="node selector applied")
        selector_task.set_caching_options(enable_caching=False)
        kubernetes.add_node_selector_json(
            task=selector_task,
            node_selector_json=optional_selector,
        )

    with dsl.If(enable_secret_env == True):
        secret_task = log_message(message="secret env applied")
        secret_task.set_caching_options(enable_caching=False)
        kubernetes.use_secret_as_env(
            task=secret_task,
            secret_name=optional_secret_name,
            secret_key_to_env={"password": "PASSWORD"},
        )

    with dsl.If(enable_config_map_env == True):
        config_map_task = log_message(message="configmap env applied")
        config_map_task.set_caching_options(enable_caching=False)
        kubernetes.use_config_map_as_env(
            task=config_map_task,
            config_map_name=optional_config_map_name,
            config_map_key_to_env={"setting": "SETTING"},
        )

    with dsl.If(enable_tolerations == True):
        toleration_task = log_message(message="tolerations applied")
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
