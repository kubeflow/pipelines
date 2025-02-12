import os

from kfp import Client
from kfp import dsl


@dsl.component
def core_comp() -> str:
    return 'foo'


@dsl.component
def crust_comp(input: str):
    print('input :', input)


@dsl.pipeline
def core() -> str:
    task = core_comp()

    return task.output


@dsl.pipeline
def mantle() -> str:
    dag_task = core()

    return dag_task.output


@dsl.pipeline(name=os.path.basename(__file__).removesuffix('.py') + '-pipeline')
def crust():
    dag_task = mantle()
    task = crust_comp(input=dag_task.output)


if __name__ == '__main__':
    # Compiler().compile(pipeline_func=crust, package_path=f"{__file__.removesuffix('.py')}.yaml")
    client = Client()
    client.create_run_from_pipeline_func(crust)
