from typing import Dict, Optional

from kfp import dsl, kubernetes


@dsl.component
def comp() -> None:
    print("secrets and configmaps test")


@dsl.pipeline
def secrets_configmaps_demo(
    secret_volume_name: Optional[str] = None,
    secret_env_name: Optional[str] = None,
    configmap_volume_name: Optional[str] = None,
    configmap_env_name: Optional[str] = None,
    image_pull_secret_name: Optional[str] = None,
):
    task = comp()
    task.set_caching_options(enable_caching=False)

    kubernetes.use_secret_as_volume(
        task=task,
        secret_name=secret_volume_name,
        mount_path="/mnt/secret",
    )

    kubernetes.use_secret_as_env(
        task=task,
        secret_name=secret_env_name,
        secret_key_to_env={"password": "PASSWORD"},
    )

    kubernetes.use_config_map_as_volume(
        task=task,
        config_map_name=configmap_volume_name,
        mount_path="/mnt/config",
    )

    kubernetes.use_config_map_as_env(
        task=task,
        config_map_name=configmap_env_name,
        config_map_key_to_env={"setting": "SETTING"},
    )

    kubernetes.set_image_pull_secrets(
        task=task,
        secret_names=[name for name in [image_pull_secret_name] if name is not None],
    )


if __name__ == "__main__":
    from kfp import compiler

    compiler.Compiler().compile(
        pipeline_func=secrets_configmaps_demo,
        package_path="secrets_configmaps_pipeline.yaml",
    )
