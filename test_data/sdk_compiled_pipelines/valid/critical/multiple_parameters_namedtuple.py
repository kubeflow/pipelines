import os
from typing import NamedTuple

from kfp import Client
from kfp import dsl


@dsl.component
def core_comp() -> NamedTuple('outputs', val1=str, val2=str):  # type: ignore
    outputs = NamedTuple('outputs', val1=str, val2=str)
    return outputs('foo', 'bar')


@dsl.component
def crust_comp(val1: str, val2: str):
    print('val1: ', val1)
    print('val2: ', val2)


@dsl.pipeline
def core() -> NamedTuple('outputs', val1=str, val2=str):  # type: ignore
    task = core_comp()
    task.set_caching_options(False)

    return task.outputs


@dsl.pipeline
def mantle() -> NamedTuple('outputs', val1=str, val2=str):  # type: ignore
    dag_task = core()
    dag_task.set_caching_options(False)

    return dag_task.outputs


@dsl.pipeline(name=os.path.basename(__file__).removesuffix('.py') + '-pipeline')
def crust():
    dag_task = mantle()
    dag_task.set_caching_options(False)

    task = crust_comp(
        val1=dag_task.outputs['val1'],
        val2=dag_task.outputs['val2'],
    )
    task.set_caching_options(False)


if __name__ == '__main__':
    # Compiler().compile(pipeline_func=crust, package_path=f"{__file__.removesuffix('.py')}.yaml")
    client = Client()
    client.create_run_from_pipeline_func(crust)
