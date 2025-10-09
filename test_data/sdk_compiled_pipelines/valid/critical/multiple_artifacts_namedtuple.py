import os
from typing import NamedTuple

from kfp import Client
from kfp import dsl


@dsl.component
def core_comp(ds1: dsl.Output[dsl.Dataset], ds2: dsl.Output[dsl.Dataset]):
    with open(ds1.path, 'w') as f:
        f.write('foo')
    with open(ds2.path, 'w') as f:
        f.write('bar')


@dsl.component
def crust_comp(
    ds1: dsl.Dataset,
    ds2: dsl.Dataset,
):
    with open(ds1.path, 'r') as f:
        print('ds1: ', f.read())
    with open(ds2.path, 'r') as f:
        print('ds2: ', f.read())


@dsl.pipeline
def core() -> NamedTuple(
    'outputs',
    ds1=dsl.Dataset,
    ds2=dsl.Dataset,
):  # type: ignore
    task = core_comp()
    task.set_caching_options(False)

    return task.outputs


@dsl.pipeline
def mantle() -> NamedTuple(
    'outputs',
    ds1=dsl.Dataset,
    ds2=dsl.Dataset,
):  # type: ignore
    dag_task = core()
    dag_task.set_caching_options(False)

    return dag_task.outputs


@dsl.pipeline(name=os.path.basename(__file__).removesuffix('.py') + '-pipeline')
def crust():
    dag_task = mantle()
    dag_task.set_caching_options(False)

    task = crust_comp(
        ds1=dag_task.outputs['ds1'],
        ds2=dag_task.outputs['ds2'],
    )
    task.set_caching_options(False)


if __name__ == '__main__':
    # Compiler().compile(pipeline_func=crust, package_path=f"{__file__.removesuffix('.py')}.yaml")
    client = Client()
    client.create_run_from_pipeline_func(crust)
