from kfp import dsl
from kfp.client import Client
from kfp.compiler import Compiler

@dsl.component
def small_comp() -> str:
    return "privet"

@dsl.component
def large_comp(input: str):
    print("input :", input)


@dsl.pipeline
def small_matroushka_doll() -> str:
    task = small_comp()
    task.set_caching_options(False)
    return task.output

@dsl.pipeline
def medium_matroushka_doll() -> str:
    dag_task = small_matroushka_doll()
    dag_task.set_caching_options(False)
    return dag_task.output

@dsl.pipeline
def large_matroushka_doll():
    dag_task = medium_matroushka_doll()
    task = large_comp(input=dag_task.output)
    task.set_caching_options(False)
    dag_task.set_caching_options(False)


if __name__ == "__main__":
    client = Client()

    run = client.create_run_from_pipeline_func(
        pipeline_func=large_matroushka_doll,
        enable_caching=False,
    )
