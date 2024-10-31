import os

from kfp import Client
from kfp import dsl
from kfp.compiler import Compiler


@dsl.component
def core_comp() -> int:
    return 1


@dsl.component
def crust_comp(x: int, y: int):
    print('sum :', x + y)


@dsl.pipeline
def core() -> int:
    task = core_comp()
    task.set_caching_options(False)

    return task.output


@dsl.pipeline
def mantle() -> int:
    dag_task = core()
    dag_task.set_caching_options(False)

    return dag_task.output


@dsl.pipeline(name=os.path.basename(__file__).removesuffix('.py') + '-pipeline')
def crust():
    dag_task = mantle()
    dag_task.set_caching_options(False)

    task = crust_comp(x=2, y=dag_task.output)
    task.set_caching_options(False)


if __name__ == '__main__':
    Compiler().compile(
        pipeline_func=crust,
        package_path=f"{__file__.removesuffix('.py')}.yaml")
    client = Client()
    client.create_run_from_pipeline_func(crust)
