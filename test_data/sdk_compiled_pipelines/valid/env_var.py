from kfp import dsl


@dsl.component(base_image="public.ecr.aws/docker/library/python:3.12")
def comp(env_var: str) -> str:
    import os

    value = os.getenv(env_var, "")

    if value == "":
        raise Exception("Env var is not set")

    return value


@dsl.pipeline
def test_env_exists(env_var: str) -> str:
    comp_task = comp(env_var=env_var)
    comp_task.set_caching_options(False)
    return comp_task.output
