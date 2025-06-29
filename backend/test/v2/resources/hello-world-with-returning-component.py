from kfp import dsl


@dsl.component(base_image="public.ecr.aws/docker/library/python:3.12")
def comp(message: str) -> str:
    print(message)
    return message


@dsl.component(base_image="public.ecr.aws/docker/library/python:3.12")
def process_message(msg: str) -> str:
    # For demonstration, just append a string
    return f"Processed: {msg}"


@dsl.pipeline
def my_pipeline(message: str) -> str:
    first = comp(message=message)
    second = process_message(msg=first.output)
    return second.output
