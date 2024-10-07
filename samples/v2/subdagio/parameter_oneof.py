import os

from kfp import Client
from kfp import dsl

@dsl.component
def flip_coin() -> str:
    import random
    return 'heads' if random.randint(0, 1) == 0 else 'tails'

@dsl.component
def core_comp(input: str) -> str:
    print('input :', input)
    return input

@dsl.component
def core_output_comp(input: str, output_key: dsl.OutputPath(str)):
    print('input :', input)
    with open(output_key, 'w') as f:
        f.write(input)

@dsl.component
def crust_comp(input: str):
    print('input :', input)

@dsl.pipeline
def core() -> str:
    flip_coin_task = flip_coin().set_caching_options(False)
    with dsl.If(flip_coin_task.output == 'heads'):
        t1 = core_comp(input='Got heads!').set_caching_options(False)
    with dsl.Else():
        t2 = core_output_comp(input='Got tails!').set_caching_options(False)
    return dsl.OneOf(t1.output, t2.outputs['output_key'])

@dsl.pipeline
def mantle() -> str:
    dag_task = core()
    dag_task.set_caching_options(False)

    return dag_task.output

@dsl.pipeline(name=os.path.basename(__file__).removesuffix('.py') + '-pipeline')
def crust():
    dag_task = mantle()
    dag_task.set_caching_options(False)

    task = crust_comp(input=dag_task.output)
    task.set_caching_options(False)


if __name__ == '__main__':
    # Compiler().compile(pipeline_func=crust, package_path=f"{__file__.removesuffix('.py')}.yaml")
    client = Client()
    client.create_run_from_pipeline_func(crust)
