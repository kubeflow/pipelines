from kfp import dsl


@dsl.component(base_image="public.ecr.aws/docker/library/python:3.12")
def comp(message: str) -> str:
    print(message)
    return message


@dsl.pipeline
def my_pipeline(message: str) -> str:
    return comp(message=message).output
